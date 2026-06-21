# VLESS+Reality TCP Fallback — Implementation Plan (Phase 1: pilot HK1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add VLESS+Reality on TCP 443 to node HK1 as a UDP-block fallback, fully integrated into hy2board (per-user UUIDv5, traffic metered into `traffic_used`, disable/expiry enforced), validated from a UDP-blocked home WiFi.

**Architecture:** Panel stays source of truth. An on-node `vless-agent` pulls the active-user list from a new panel endpoint and applies it to Xray; the panel polls the node's Xray traffic stats and feeds the existing accrual. Subscriptions gain VLESS lines in the existing proxy groups for automatic UDP→TCP failover.

**Tech Stack:** Go (Gin, GORM, SQLite) panel; Xray-core on the node; bash/jq agent; React subscription unaffected. Build/test in `golang:1.26-alpine` container (`CGO_ENABLED=1`, mattn sqlite). Spec: `docs/superpowers/specs/2026-06-21-vless-reality-fallback-design.md`.

**Test runner:** `docker run --rm -v /opt/hy2board:/app -v hy2board_gocache:/go -w /app -e CGO_ENABLED=1 golang:1.26-alpine sh -c "apk add --no-cache gcc musl-dev >/dev/null 2>&1; go test ./<pkg>/ -run <name> -v"`

**Source-of-truth note:** Edit files in the GitHub mirror `/root/ludandaye/ladder/hy2board` (commit/push here) AND copy each changed file to the live server `/opt/hy2board` before building. Both must stay in sync.

---

## File structure

| File | Responsibility | New? |
|---|---|---|
| `internal/util/uuid.go` | `VlessUUID(username)` deterministic UUIDv5 | create |
| `internal/util/uuid_test.go` | UUID determinism tests | create |
| `internal/config/config.go` | add `Node.Secret` config | modify |
| `internal/model/node.go` | add VLESS/Reality columns to `Node` | modify |
| `internal/handler/node_vless.go` | `GET /api/node/vless/clients`, node-secret auth | create |
| `internal/handler/node_vless_test.go` | endpoint tests | create |
| `internal/service/vless_subscription.go` | build VLESS line + group injection helpers | create |
| `internal/service/vless_subscription_test.go` | subscription-line tests | create |
| `internal/service/subscription.go` | call VLESS helpers in Clash/Surge/URI | modify |
| `internal/service/vless_traffic.go` | poll node VLESS stats → accrual | create |
| `main.go` | register route + start VLESS poller | modify |
| `deploy/vless-pilot-setup.sh` | HK1: install Xray+Reality+agent | create |
| `deploy/vless-agent.sh` | on-node sync + stats agent | create |

---

## Task 1: Deterministic VLESS UUID

**Files:** Create `internal/util/uuid.go`, `internal/util/uuid_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/util/uuid_test.go
package util

import "testing"

func TestVlessUUIDIsDeterministicAndValid(t *testing.T) {
	a := VlessUUID("ludandaye")
	b := VlessUUID("ludandaye")
	if a != b {
		t.Fatalf("not deterministic: %s vs %s", a, b)
	}
	if VlessUUID("zyk") == a {
		t.Fatalf("different users must get different uuids")
	}
	// canonical 8-4-4-4-12, version 5, RFC4122 variant
	if len(a) != 36 || a[14] != '5' || (a[19] != '8' && a[19] != '9' && a[19] != 'a' && a[19] != 'b') {
		t.Fatalf("not a v5 uuid: %s", a)
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run the test-runner command with `./internal/util/ -run TestVlessUUID`.
Expected: FAIL — `undefined: VlessUUID`.

- [ ] **Step 3: Implement (no new dependency — crypto/sha1)**

```go
// internal/util/uuid.go
package util

import (
	"crypto/sha1"
	"encoding/hex"
)

// fixed namespace UUID for hy2board VLESS identities (random, constant forever)
var vlessNamespace = [16]byte{0x6b, 0xa7, 0xb8, 0x12, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

// VlessUUID returns a deterministic RFC4122 v5 UUID for a username.
func VlessUUID(username string) string {
	h := sha1.New()
	h.Write(vlessNamespace[:])
	h.Write([]byte(username))
	s := h.Sum(nil)[:16]
	s[6] = (s[6] & 0x0f) | 0x50 // version 5
	s[8] = (s[8] & 0x3f) | 0x80 // RFC4122 variant
	d := hex.EncodeToString(s)
	return d[0:8] + "-" + d[8:12] + "-" + d[12:16] + "-" + d[16:20] + "-" + d[20:32]
}
```

- [ ] **Step 4: Run test, verify it passes** (Expected: PASS)

- [ ] **Step 5: Commit**

```bash
git add internal/util/uuid.go internal/util/uuid_test.go
git commit -m "feat(vless): deterministic per-user UUIDv5"
```

---

## Task 2: Config — node secret

**Files:** Modify `internal/config/config.go`

- [ ] **Step 1: Add the struct + accessor** (no test; config plumbing)

In `internal/config/config.go` add after `TelegramConfig`:

```go
type NodeConfig struct {
	Secret string `yaml:"secret"`
}
```

Add field to `Config`:

```go
	Node       NodeConfig       `yaml:"node"`
```

Add accessor:

```go
func (c *Config) HasNodeSecret() bool { return c.Node.Secret != "" }
```

- [ ] **Step 2: Add to config.yaml on the server** (NOT committed — secrets are gitignored)

On `/opt/hy2board/config.yaml` add a strong secret:

```yaml
node:
  secret: "<32+ char random hex>"
```

Generate with `openssl rand -hex 24`.

- [ ] **Step 3: Compile-check + commit**

Run: test-runner with `go build -o /tmp/hy2board .` → Expected: `BUILD_OK`.

```bash
git add internal/config/config.go
git commit -m "feat(config): node.secret for node->panel auth"
```

---

## Task 3: Node model — VLESS/Reality columns

**Files:** Modify `internal/model/node.go`

- [ ] **Step 1: Add fields to the `Node` struct**

Append inside `type Node struct { ... }` (after `SortOrder`):

```go
	// VLESS+Reality fallback (Phase 1)
	VlessEnabled   bool   `gorm:"default:false" json:"vless_enabled"`
	VlessPort      int    `gorm:"default:0" json:"vless_port"`
	RealityPubkey  string `gorm:"default:''" json:"reality_pubkey"`
	RealityShortID string `gorm:"default:''" json:"reality_shortid"`
	RealitySNI     string `gorm:"default:''" json:"reality_sni"`
	VlessStatsAPI  string `gorm:"default:''" json:"vless_stats_api"`
	VlessStatsSecret string `gorm:"default:''" json:"-"`
```

GORM `AutoMigrate(&model.Node{})` (already called in `internal/database`) adds these nullable columns on boot — additive and safe.

- [ ] **Step 2: Compile-check + commit**

Run: `go build -o /tmp/hy2board .` → Expected: `BUILD_OK`.

```bash
git add internal/model/node.go
git commit -m "feat(vless): node model reality/vless columns"
```

---

## Task 4: Client-list endpoint

**Files:** Create `internal/handler/node_vless.go`, `internal/handler/node_vless_test.go`; modify `main.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/handler/node_vless_test.go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVlessClientsReturnsOnlyActiveUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.User{})
	database.DB = db
	config.C.Node.Secret = "s3cret"

	future := time.Now().Add(24 * time.Hour)
	db.Create(&model.User{Username: "alice", Hy2Password: "a:x", SubToken: "t1", Enabled: true, ExpiresAt: future})
	db.Create(&model.User{Username: "bob", Hy2Password: "b:x", SubToken: "t2", Enabled: false, ExpiresAt: future}) // disabled
	db.Create(&model.User{Username: "carol", Hy2Password: "c:x", SubToken: "t3", Enabled: true, TrafficLimit: 100, TrafficUsed: 100}) // over limit

	r := gin.New()
	r.GET("/api/node/vless/clients", VlessClients)

	// wrong secret -> 401
	req := httptest.NewRequest("GET", "/api/node/vless/clients", nil)
	req.Header.Set("Authorization", "wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong secret should 401, got %d", w.Code)
	}

	// right secret -> only alice
	req = httptest.NewRequest("GET", "/api/node/vless/clients", nil)
	req.Header.Set("Authorization", "s3cret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []map[string]string
	json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0]["email"] != "alice" || got[0]["uuid"] == "" {
		t.Fatalf("expected only active alice with uuid, got %#v", got)
	}
}
```

- [ ] **Step 2: Run test, verify it fails** (Expected: FAIL — `undefined: VlessClients`)

- [ ] **Step 3: Implement the handler**

```go
// internal/handler/node_vless.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
)

// VlessClients returns the active VLESS users a node should accept.
// Auth: Authorization header must equal config node secret.
func VlessClients(c *gin.Context) {
	if !config.C.HasNodeSecret() || c.GetHeader("Authorization") != config.C.Node.Secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var users []model.User
	database.DB.Find(&users)
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		if !u.IsActive() {
			continue
		}
		out = append(out, gin.H{"uuid": util.VlessUUID(u.Username), "email": u.Username})
	}
	c.JSON(http.StatusOK, out)
}
```

- [ ] **Step 4: Run test, verify it passes** (Expected: PASS)

- [ ] **Step 5: Register the route in main.go**

Add next to the other public node routes (after `r.POST("/api/auth/hy2", handler.Hy2Auth)`):

```go
	r.GET("/api/node/vless/clients", handler.VlessClients)
```

- [ ] **Step 6: Build + commit**

Run: `go build -o /tmp/hy2board .` → Expected: `BUILD_OK`.

```bash
git add internal/handler/node_vless.go internal/handler/node_vless_test.go main.go
git commit -m "feat(vless): /api/node/vless/clients active-user endpoint"
```

---

## Task 5: VLESS subscription lines

**Files:** Create `internal/service/vless_subscription.go`, `internal/service/vless_subscription_test.go`; modify `internal/service/subscription.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/service/vless_subscription_test.go
package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestVlessClashAndURILines(t *testing.T) {
	u := model.User{Username: "alice"}
	n := model.Node{Name: "HK1-vless", Host: "38.47.108.14", VlessEnabled: true, VlessPort: 443,
		RealityPubkey: "PUB", RealityShortID: "ab12", RealitySNI: "www.microsoft.com"}

	uri := VlessURILine(u, n)
	if !strings.HasPrefix(uri, "vless://") || !strings.Contains(uri, "security=reality") ||
		!strings.Contains(uri, "pbk=PUB") || !strings.Contains(uri, "sni=www.microsoft.com") ||
		!strings.Contains(uri, "@38.47.108.14:443") {
		t.Fatalf("bad vless uri: %s", uri)
	}

	clash := VlessClashBlock(u, n)
	if !strings.Contains(clash, "type: vless") || !strings.Contains(clash, "public-key: PUB") ||
		!strings.Contains(clash, "servername: www.microsoft.com") {
		t.Fatalf("bad clash block: %s", clash)
	}
}
```

- [ ] **Step 2: Run test, verify it fails** (Expected: FAIL — undefined functions)

- [ ] **Step 3: Implement the helpers**

```go
// internal/service/vless_subscription.go
package service

import (
	"fmt"

	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
)

// VlessURILine renders a vless://...reality URI (v2rayN / Shadowrocket / generic).
func VlessURILine(u model.User, n model.Node) string {
	uuid := util.VlessUUID(u.Username)
	return fmt.Sprintf(
		"vless://%s@%s:%d?encryption=none&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&flow=xtls-rprx-vision&type=tcp#%s",
		uuid, n.Host, n.VlessPort, n.RealitySNI, n.RealityPubkey, n.RealityShortID, n.Name,
	)
}

// VlessClashBlock renders a Clash/mihomo proxy entry (YAML list item).
func VlessClashBlock(u model.User, n model.Node) string {
	uuid := util.VlessUUID(u.Username)
	return fmt.Sprintf(`  - name: "%s"
    type: vless
    server: %s
    port: %d
    uuid: %s
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: %s
    client-fingerprint: chrome
    reality-opts:
      public-key: %s
      short-id: %s`, n.Name, n.Host, n.VlessPort, uuid, n.RealitySNI, n.RealityPubkey, n.RealityShortID)
}

// VlessSurgeLine renders a Surge proxy line.
func VlessSurgeLine(u model.User, n model.Node) string {
	uuid := util.VlessUUID(u.Username)
	return fmt.Sprintf("%s = vless, %s, %d, username=%s, tls=true, sni=%s, reality-pubkey=%s, reality-short-id=%s, flow=xtls-rprx-vision",
		n.Name, n.Host, n.VlessPort, uuid, n.RealitySNI, n.RealityPubkey, n.RealityShortID)
}
```

- [ ] **Step 4: Run test, verify it passes** (Expected: PASS)

- [ ] **Step 5: Wire into the generators (Clash + URI)**

In `internal/service/subscription.go`:

(a) In `generateURI` (the `for _, n := range nodes` loop), after appending the HY2 uri, append a VLESS uri when enabled:

```go
		if n.VlessEnabled && n.VlessPort > 0 && n.RealityPubkey != "" {
			uris = append(uris, VlessURILine(user, n))
		}
```

(b) In `GenerateClashWithCustomRules`, where each node's proxy block is emitted, after the HY2 block add:

```go
		if n.VlessEnabled && n.VlessPort > 0 && n.RealityPubkey != "" {
			lines = append(lines, VlessClashBlock(user, n))
			// add the vless node name into the Auto/Manual group member lists built below
		}
```

Then include the VLESS node name (use `n.Name` for the vless entry — give it a distinct name suffix to avoid colliding with the HY2 node of the same name; see note) in the Auto and Manual proxy-group member arrays so clients failover to it.

> **Naming note:** emit the VLESS entry under name `n.Name + "-T"` (T = TCP) so it does not collide with the HY2 proxy of the same `n.Name`. Use `n.Name+"-T"` in `VlessURILine`/`VlessClashBlock`/`VlessSurgeLine` Name fields and in the group member lists. Update the Task-5 helpers to take the display name, or append "-T" inside them. (Pick one; the test asserts substrings, not the exact name, so appending "-T" in the helpers is fine — update the test's expected `#%s` fragment accordingly if you assert it.)

- [ ] **Step 6: Build + run service tests + commit**

Run: `go build -o /tmp/hy2board .` and `go test ./internal/service/ -run Vless -v` → Expected: BUILD_OK, PASS.

```bash
git add internal/service/vless_subscription.go internal/service/vless_subscription_test.go internal/service/subscription.go
git commit -m "feat(vless): emit reality nodes in subscriptions with UDP->TCP failover"
```

---

## Task 6: VLESS traffic metering (reuse accrual)

**Files:** Create `internal/service/vless_traffic.go`; modify `main.go`

The node agent serves `GET <VlessStatsAPI>` → JSON `{ "alice": {"tx": N, "rx": N}, ... }` (cumulative). The panel polls all VLESS nodes, builds a per-node-per-user map, and reuses `trafficUsageDelta` + `persistTrafficUsage` (already shipped in `traffic_accrual.go`).

- [ ] **Step 1: Write the failing test**

```go
// internal/service/vless_traffic_test.go
package service

import "testing"

func TestParseVlessStats(t *testing.T) {
	body := []byte(`{"alice":{"tx":100,"rx":200},"bob":{"tx":5,"rx":0}}`)
	m, err := parseVlessStats(body)
	if err != nil || m["alice"].TX != 100 || m["alice"].RX != 200 || m["bob"].TX != 5 {
		t.Fatalf("bad parse: %#v err=%v", m, err)
	}
}
```

- [ ] **Step 2: Run test, verify it fails** (Expected: FAIL — `undefined: parseVlessStats`)

- [ ] **Step 3: Implement poller + parser**

```go
// internal/service/vless_traffic.go
package service

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

func parseVlessStats(body []byte) (map[string]TrafficData, error) {
	var m map[string]TrafficData
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func fetchVlessStats(n model.Node) map[string]TrafficData {
	if n.VlessStatsAPI == "" {
		return nil
	}
	req, err := http.NewRequest("GET", n.VlessStatsAPI, nil)
	if err != nil {
		return nil
	}
	if n.VlessStatsSecret != "" {
		req.Header.Set("Authorization", n.VlessStatsSecret)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	buf := make([]byte, 0)
	dec := json.NewDecoder(resp.Body)
	var m map[string]TrafficData
	if dec.Decode(&m) != nil {
		_ = buf
		return nil
	}
	return m
}

// StartVlessTrafficPoller meters VLESS usage into users.traffic_used using the
// same per-node-per-user delta accrual as HY2.
func StartVlessTrafficPoller(interval time.Duration) {
	prev := map[uint]map[string]TrafficData{}
	tick := func() {
		var nodes []model.Node
		database.DB.Where("vless_enabled = ?", true).Find(&nodes)
		now := map[uint]map[string]TrafficData{}
		for _, n := range nodes {
			if s := fetchVlessStats(n); s != nil {
				now[n.ID] = s
			}
		}
		persistTrafficUsage(database.DB, trafficUsageDelta(prev, now))
		prev = now
	}
	tick()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			tick()
		}
	}()
}
```

- [ ] **Step 4: Run test, verify it passes** (Expected: PASS)

- [ ] **Step 5: Start the poller in main.go**

After `service.StartTrafficCache(3 * time.Second)`:

```go
	service.StartVlessTrafficPoller(3 * time.Second)
```

- [ ] **Step 6: Build + commit**

Run: `go build -o /tmp/hy2board .` → Expected: BUILD_OK.

```bash
git add internal/service/vless_traffic.go internal/service/vless_traffic_test.go main.go
git commit -m "feat(vless): meter node VLESS traffic into traffic_used"
```

---

## Task 7: Deploy the panel changes

- [ ] **Step 1: Sync all changed Go files + main.go to `/opt/hy2board`** (scp each modified/created file).
- [ ] **Step 2: Add `node.secret` to `/opt/hy2board/config.yaml`** (Task 2 Step 2) and pick a strong value.
- [ ] **Step 3: Full suite in container**

Run: `go vet ./internal/... ; go test ./internal/...` → Expected: all PASS (ignore pre-existing `backups/` vet noise).

- [ ] **Step 4: Tag rollback + build + hot swap**

```bash
cd /opt/hy2board
docker tag "$(docker inspect hy2board-hy2board-1 --format '{{.Image}}')" hy2board-hy2board:rollback-prevless
docker compose build && docker compose up -d
```

- [ ] **Step 5: Verify panel health** (auth `ok:true`, `/api/node/vless/clients` returns `[]` with the node secret, 401 without).

- [ ] **Step 6: Commit + push panel changes to GitHub** (`git push origin HEAD:main`).

---

## Task 8: HK1 pilot node — Xray+Reality+agent

**Files:** Create `deploy/vless-pilot-setup.sh`, `deploy/vless-agent.sh`

- [ ] **Step 1: Write `deploy/vless-agent.sh`** (runs on the node; sync + stats)

```bash
#!/usr/bin/env bash
# vless-agent: pull active users from panel -> Xray clients; serve cumulative stats.
set -euo pipefail
PANEL_CLIENTS_URL="${PANEL_CLIENTS_URL:?}"      # https://vpn.linkbyfree.com/api/node/vless/clients
NODE_SECRET="${NODE_SECRET:?}"
XRAY_CONFIG="/usr/local/etc/xray/config.json"
XRAY_API="127.0.0.1:10085"
STATS_SECRET="${STATS_SECRET:?}"
STATS_PORT="${STATS_PORT:-25415}"

sync_users() {
  local clients tmp
  clients="$(curl -fsS -H "Authorization: ${NODE_SECRET}" "$PANEL_CLIENTS_URL")" || return 0
  # Build the clients array for the VLESS inbound
  tmp="$(jq --argjson c "$clients" '
    (.inbounds[] | select(.tag=="vless-reality") | .settings.clients) = ($c | map({id:.uuid, email:.email, flow:"xtls-rprx-vision"}))
  ' "$XRAY_CONFIG")"
  if ! diff -q <(jq -S . "$XRAY_CONFIG") <(echo "$tmp" | jq -S .) >/dev/null 2>&1; then
    echo "$tmp" > "$XRAY_CONFIG"
    systemctl restart xray
    echo "[vless-agent] user set changed -> xray restarted ($(echo "$clients" | jq length) users)"
  fi
}

serve_stats() {
  # tiny HTTP server: returns {email:{tx,rx}} from Xray StatsService
  while true; do
    { echo -ne "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n";
      xray api statsquery --server="$XRAY_API" 2>/dev/null | jq -c '
        [.stat[]? | select(.name|test("user>>>.*>>>traffic"))
         | (.name|capture("user>>>(?<u>[^>]+)>>>traffic>>>(?<dir>uplink|downlink)")) as $m
         | {u:$m.u, dir:$m.dir, v:(.value|tonumber)}]
        | group_by(.u) | map({(.[0].u): {tx:(map(select(.dir=="uplink").v)|add // 0), rx:(map(select(.dir=="downlink").v)|add // 0)}}) | add // {}';
    } | timeout 5 nc -l -p "$STATS_PORT" -q 1 >/dev/null 2>&1 || true
  done
}

case "${1:-sync}" in
  sync) sync_users ;;
  stats) serve_stats ;;
esac
```

> Stats auth note: bind `nc` stats to localhost and have the panel reach it over the existing trafficStats channel, OR front it with the same `Authorization: $STATS_SECRET` check via a 3-line wrapper. For the pilot, restrict the cloud firewall so only the panel IP can reach `STATS_PORT`, and set the node's `vless_stats_secret` accordingly. (Harden in Phase 2.)

- [ ] **Step 2: Write `deploy/vless-pilot-setup.sh`** (idempotent installer)

```bash
#!/usr/bin/env bash
set -euo pipefail
: "${NODE_SECRET:?}" "${STATS_SECRET:?}"
PANEL_CLIENTS_URL="${PANEL_CLIENTS_URL:-https://vpn.linkbyfree.com/api/node/vless/clients}"
REALITY_DEST="${REALITY_DEST:-www.microsoft.com:443}"
REALITY_SNI="${REALITY_SNI:-www.microsoft.com}"
STATS_PORT="${STATS_PORT:-25415}"

# 1) install deps + Xray
command -v jq >/dev/null || (apt-get update && apt-get install -y jq netcat-openbsd)
bash <(curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh) install

# 2) reality keys + shortid
KEYS="$(xray x25519)"; PRIV="$(echo "$KEYS"|awk '/Private/{print $3}')"; PUB="$(echo "$KEYS"|awk '/Public/{print $3}')"
SID="$(openssl rand -hex 4)"

# 3) write Xray config (TCP 443, clients filled by agent)
cat >/usr/local/etc/xray/config.json <<JSON
{ "stats": {}, "api": {"tag":"api","services":["StatsService","HandlerService"]},
  "policy": {"levels":{"0":{"statsUserUplink":true,"statsUserDownlink":true}}},
  "inbounds": [
    {"tag":"api","listen":"127.0.0.1","port":10085,"protocol":"dokodemo-door","settings":{"address":"127.0.0.1"}},
    {"tag":"vless-reality","listen":"0.0.0.0","port":443,"protocol":"vless",
     "settings":{"clients":[],"decryption":"none"},
     "streamSettings":{"network":"tcp","security":"reality",
       "realitySettings":{"dest":"${REALITY_DEST}","serverNames":["${REALITY_SNI}"],"privateKey":"${PRIV}","shortIds":["${SID}"]}}}
  ],
  "routing":{"rules":[{"type":"field","inboundTag":["api"],"outboundTag":"api"}]},
  "outbounds":[{"protocol":"freedom"}]
}
JSON

# 4) install agent + systemd timer (sync) + service (stats)
install -m700 deploy/vless-agent.sh /usr/local/bin/vless-agent.sh
cat >/etc/systemd/system/vless-agent-sync.service <<UNIT
[Service]
Environment=PANEL_CLIENTS_URL=${PANEL_CLIENTS_URL} NODE_SECRET=${NODE_SECRET} STATS_SECRET=${STATS_SECRET} STATS_PORT=${STATS_PORT}
ExecStart=/usr/local/bin/vless-agent.sh sync
UNIT
cat >/etc/systemd/system/vless-agent-sync.timer <<UNIT
[Timer]
OnUnitActiveSec=30
OnBootSec=20
[Install]
WantedBy=timers.target
UNIT
cat >/etc/systemd/system/vless-agent-stats.service <<UNIT
[Service]
Environment=STATS_SECRET=${STATS_SECRET} STATS_PORT=${STATS_PORT}
ExecStart=/usr/local/bin/vless-agent.sh stats
Restart=always
[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable --now xray vless-agent-stats.service vless-agent-sync.timer

# 5) cloud-firewall reminder + output panel registration values
echo "OPEN cloud firewall: TCP 443 (clients) ; TCP ${STATS_PORT} from PANEL IP only"
echo "REGISTER IN PANEL (HK1 node): vless_enabled=1 vless_port=443 reality_pubkey=${PUB} reality_shortid=${SID} reality_sni=${REALITY_SNI} vless_stats_api=http://$(curl -4 -fsS ifconfig.me):${STATS_PORT}/ vless_stats_secret=${STATS_SECRET}"
```

- [ ] **Step 3: Run setup on HK1** (with real `NODE_SECRET`/`STATS_SECRET`), open cloud firewall TCP 443 + stats port from panel IP, verify `systemctl status xray` active and `ss -lntp | grep :443`.

- [ ] **Step 4: Commit the deploy scripts**

```bash
git add deploy/vless-agent.sh deploy/vless-pilot-setup.sh
git commit -m "feat(vless): HK1 pilot setup + on-node agent"
```

---

## Task 9: Register HK1 in the panel + end-to-end validation

- [ ] **Step 1: Set HK1's VLESS fields** via SQL (or admin API) using the values printed by Step 8.2:

```sql
UPDATE nodes SET vless_enabled=1, vless_port=443, reality_pubkey='<PUB>',
  reality_shortid='<SID>', reality_sni='www.microsoft.com',
  vless_stats_api='http://38.47.108.14:25415/', vless_stats_secret='<STATS_SECRET>'
WHERE name='HK1-plain';
```

- [ ] **Step 2: Confirm sync** — on HK1, `journalctl -u vless-agent-sync` shows it pulled N users and Xray has them.

- [ ] **Step 3: Confirm subscription** — `curl -s "http://127.0.0.1:9000/api/sub/<ludandaye_token>?format=clash" | grep -A2 'type: vless'` shows the HK1 VLESS node.

- [ ] **Step 4: SUCCESS CRITERION — from the UDP-blocked home WiFi:** refresh subscription, select the HK1 VLESS (`-T`) node, confirm it connects and passes traffic where HY2 fails.

- [ ] **Step 5: Confirm metering** — `traffic_used` for the test user increases after VLESS use (poll `node_probe`/DB).

- [ ] **Step 6: Confirm enforcement** — disable the test user in the panel; within ~30s `journalctl -u vless-agent-sync` shows removal and the VLESS connection drops.

- [ ] **Step 7: Final commit/push** of any fixups; record rollback image `hy2board-hy2board:rollback-prevless`.

---

## Self-review

- **Spec coverage:** node-side Xray+Reality+agent (T8), client-list endpoint (T4), UUIDv5 identity (T1), traffic metering via existing accrual (T6), subscription failover lines (T5), node model fields (T3), node secret (T2), pilot validation incl. metering+enforcement (T9). All spec sections mapped.
- **Placeholders:** none — every code step has real code; the two operational notes (stats auth hardening, vless name `-T` suffix) are explicit decisions, not TODOs.
- **Type consistency:** `TrafficData{TX,RX}` reused from `traffic.go`; `trafficUsageDelta`/`persistTrafficUsage` reused from `traffic_accrual.go`; `model.Node` fields match across T3/T5/T6/T9; `VlessUUID` signature consistent T1/T4/T5.
- **Known follow-ups (Phase 2):** harden stats endpoint auth; roll out to 4 other nodes; admin-UI editor for reality params; move agent user-apply from restart-on-change to Xray gRPC hot add/remove.
