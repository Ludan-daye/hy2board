# Trojan TCP Fallback (Surge) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Trojan inbound (TCP 8443) on each VLESS node, emitted as the `<node>-T` node in the Surge subscription, so Surge users get a TCP fallback when UDP is blocked.

**Architecture:** Reuse the VLESS control plane unchanged — same Xray, same on-node agent (gains a trojan-inbound sync), same node-secret client endpoint (gains a per-user `password`), same Xray-stats metering (auto-counts Trojan bytes). Panel emits Trojan only in Surge. Scope note: Shadowrocket Trojan is deferred — Shadowrocket users use the existing `vless://` URI.

**Tech Stack:** Go (Gin/GORM/SQLite) panel; Xray Trojan inbound + self-signed cert on nodes; bash/jq agent. Build/test in `golang:1.26-alpine` (`CGO_ENABLED=1`).

**Test runner:** `docker run --rm -v /opt/hy2board:/app -v hy2board_gocache:/go -w /app -e CGO_ENABLED=1 golang:1.26-alpine sh -c "apk add --no-cache gcc musl-dev >/dev/null 2>&1; go test ./<pkg>/ -run <name> -v"`

**Sync note:** edit in `/root/ludandaye/ladder/hy2board` (commit/push) AND scp each changed file to `/opt/hy2board` before building.

---

## File structure

| File | Responsibility | New? |
|---|---|---|
| `internal/util/trojan.go` (+`_test.go`) | `TrojanPassword(username)` deterministic | create |
| `internal/model/node.go` | add `TrojanEnabled/TrojanPort/TrojanSNI` | modify |
| `internal/handler/node_vless.go` (+test) | add `password` to client list | modify |
| `internal/service/trojan_subscription.go` (+test) | `NodeHasTrojan`, `TrojanSurgeLine` | create |
| `internal/service/subscription.go` | emit Trojan in Surge | modify |
| `deploy/vless-agent.sh` | sync trojan inbound too | modify |
| `deploy/trojan-add.sh` | add trojan inbound+cert+ufw to a node | create |

---

## Task 1: TrojanPassword util

**Files:** Create `internal/util/trojan.go`, `internal/util/trojan_test.go`

- [ ] **Step 1: Failing test**

```go
// internal/util/trojan_test.go
package util

import "testing"

func TestTrojanPasswordDeterministicAndDistinct(t *testing.T) {
	a := TrojanPassword("ludandaye")
	if a != TrojanPassword("ludandaye") {
		t.Fatal("not deterministic")
	}
	if a == TrojanPassword("zyk") {
		t.Fatal("different users must differ")
	}
	if len(a) != 64 { // sha256 hex
		t.Fatalf("want 64 hex chars, got %d", len(a))
	}
}
```

- [ ] **Step 2: Run, expect FAIL** (`undefined: TrojanPassword`).

- [ ] **Step 3: Implement**

```go
// internal/util/trojan.go
package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// TrojanPassword is a deterministic per-user Trojan password derived from the
// username (distinct from hy2_password). Stable across nodes and the panel.
func TrojanPassword(username string) string {
	sum := sha256.Sum256([]byte("hy2board-trojan:" + username))
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** — `git add internal/util/trojan*.go && git commit -m "feat(trojan): deterministic per-user password"`

---

## Task 2: Node model Trojan fields

**Files:** Modify `internal/model/node.go`

- [ ] **Step 1:** After the VLESS fields in `type Node struct`, add:

```go
	// Trojan TCP fallback (Surge clients)
	TrojanEnabled bool   `gorm:"default:false" json:"trojan_enabled"`
	TrojanPort    int    `gorm:"default:0" json:"trojan_port"`
	TrojanSNI     string `gorm:"default:''" json:"trojan_sni"`
```

GORM AutoMigrate adds these nullable columns on boot (additive, safe).

- [ ] **Step 2:** Build check: test-runner `go build -o /tmp/hy2board .` → `BUILD_OK`.
- [ ] **Step 3: Commit** — `git commit -am "feat(trojan): node model trojan columns"`

---

## Task 3: Client endpoint returns Trojan password

**Files:** Modify `internal/handler/node_vless.go`, `internal/handler/node_vless_test.go`

- [ ] **Step 1: Extend the test** — in `TestVlessClientsReturnsOnlyActiveUsers`, change the final assertion to also require a password:

```go
	if len(got) != 1 || got[0]["email"] != "alice" || got[0]["uuid"] == "" || got[0]["password"] == "" {
		t.Fatalf("expected only active alice with uuid+password, got %#v", got)
	}
```

- [ ] **Step 2: Run, expect FAIL** (password empty).

- [ ] **Step 3: Implement** — in `VlessClients`, change the append to include the trojan password:

```go
		out = append(out, gin.H{
			"uuid":     util.VlessUUID(u.Username),
			"password": util.TrojanPassword(u.Username),
			"email":    u.Username,
		})
```

- [ ] **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** — `git commit -am "feat(trojan): client endpoint returns per-user trojan password"`

---

## Task 4: Trojan Surge subscription helper

**Files:** Create `internal/service/trojan_subscription.go`, `internal/service/trojan_subscription_test.go`

- [ ] **Step 1: Failing test**

```go
// internal/service/trojan_subscription_test.go
package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestTrojanSurgeLine(t *testing.T) {
	u := model.User{Username: "alice"}
	n := model.Node{Name: "HK1-plain", Host: "38.47.108.14", TrojanEnabled: true, TrojanPort: 8443, TrojanSNI: "www.apple.com"}
	if !NodeHasTrojan(n) {
		t.Fatal("NodeHasTrojan should be true")
	}
	line := TrojanSurgeLine(u, n)
	// Surge-valid `trojan` type, never `vless`; name suffixed -T; tls fields present.
	if !strings.HasPrefix(line, "HK1-plain-T = trojan,") ||
		!strings.Contains(line, "38.47.108.14, 8443") ||
		!strings.Contains(line, "sni=www.apple.com") ||
		!strings.Contains(line, "skip-cert-verify=true") ||
		!strings.Contains(line, "password=") {
		t.Fatalf("bad surge trojan line: %s", line)
	}
}
```

- [ ] **Step 2: Run, expect FAIL** (undefined `NodeHasTrojan`/`TrojanSurgeLine`).

- [ ] **Step 3: Implement**

```go
// internal/service/trojan_subscription.go
package service

import (
	"fmt"

	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
)

// NodeHasTrojan reports whether a node should emit a Trojan line.
func NodeHasTrojan(n model.Node) bool {
	return n.TrojanEnabled && n.TrojanPort > 0
}

// TrojanSurgeLine renders a Surge `trojan` proxy line. Name reuses VlessName
// (<node>-T) so the "TCP fallback" node is consistent across formats.
func TrojanSurgeLine(u model.User, n model.Node) string {
	return fmt.Sprintf("%s = trojan, %s, %d, password=%s, sni=%s, skip-cert-verify=true",
		VlessName(n), n.Host, n.TrojanPort, util.TrojanPassword(u.Username), n.TrojanSNI)
}
```

- [ ] **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** — `git add internal/service/trojan_subscription*.go && git commit -m "feat(trojan): surge proxy line helper"`

---

## Task 5: Emit Trojan in the Surge generator

**Files:** Modify `internal/service/subscription.go`

In `GenerateSurgeWithCustomRules` the loop already declares `vlessNames` and does
`names = append(names, vlessNames...)` after the loop (currently a no-op). Reuse it
for the Trojan `-T` names.

- [ ] **Step 1:** In the Surge node loop, replace the existing VLESS-skip comment block:

```go
		lines = append(lines, proxy)
		names[i] = n.Name
		// VLESS not emitted for Surge: Surge has no `vless` proxy type (it would
		// error "Unknown proxy type: vless" and break the whole config). Surge users
		// get HY2 only; the VLESS fallback is delivered via the Clash/URI formats.

		if hasChain {
```

with:

```go
		lines = append(lines, proxy)
		names[i] = n.Name
		if NodeHasTrojan(n) {
			lines = append(lines, TrojanSurgeLine(user, n))
			vlessNames = append(vlessNames, VlessName(n))
		}

		if hasChain {
```

- [ ] **Step 2: Build** — `go build -o /tmp/hy2board .` → `BUILD_OK`.

- [ ] **Step 3: Byte-identical guard test (manual):** with no trojan-enabled node, Surge output is unchanged because `NodeHasTrojan` is false → `vlessNames` stays empty → the append is a no-op. Verified at deploy (Task 6 Step 4).

- [ ] **Step 4: Commit** — `git commit -am "feat(trojan): emit <node>-T trojan node in Surge"`

---

## Task 6: Deploy panel

- [ ] **Step 1:** scp changed Go files to `/opt/hy2board` (util/trojan.go, model/node.go, handler/node_vless.go, service/trojan_subscription.go, subscription.go, and the two new test files).
- [ ] **Step 2:** Full suite — test-runner `go test ./internal/...` → all `ok`.
- [ ] **Step 3:** Tag rollback + build + hot swap:

```bash
cd /opt/hy2board
docker tag "$(docker inspect hy2board-hy2board-1 --format '{{.Image}}')" hy2board-hy2board:rollback-pretrojan
docker compose build && docker compose up -d && sleep 8
```

- [ ] **Step 4: Verify** — auth `ok:true`; trojan columns migrated (`PRAGMA table_info(nodes)`); the client endpoint now returns `password`; **Surge subscription byte-identical to before** (no trojan node enabled yet — capture md5 before/after).
- [ ] **Step 5: Commit + push** all panel changes.

---

## Task 7: On-node agent — sync Trojan inbound

**Files:** Modify `deploy/vless-agent.sh`

- [ ] **Step 1:** Replace the `sync_users` jq/diff so it sets BOTH inbounds' clients (the trojan selector is a no-op on nodes without a trojan inbound):

```bash
sync_users() {
  : "${PANEL_CLIENTS_URL:?}" "${NODE_SECRET:?}"
  local clients new
  clients="$(curl -fsS --max-time 10 -H "Authorization: ${NODE_SECRET}" "$PANEL_CLIENTS_URL")" || {
    echo "[vless-agent] panel unreachable; keeping current users"; return 0; }
  echo "$clients" | jq -e 'type=="array"' >/dev/null 2>&1 || {
    echo "[vless-agent] bad client list; skipping"; return 0; }

  new="$(jq --argjson c "$clients" '
    (.inbounds[] | select(.tag=="vless-reality") | .settings.clients) =
      ($c | map({id:.uuid, email:.email, flow:"xtls-rprx-vision"})) |
    (.inbounds[] | select(.tag=="trojan") | .settings.clients) =
      ($c | map({password:.password, email:.email}))
  ' "$XRAY_CONFIG")"

  if diff -q <(jq -S . "$XRAY_CONFIG") <(echo "$new" | jq -S .) >/dev/null 2>&1; then
    return 0
  fi
  echo "$new" > "$XRAY_CONFIG"
  systemctl restart xray
  echo "[vless-agent] user set changed -> xray restarted ($(echo "$clients" | jq length) users)"
}
```

- [ ] **Step 2: Syntax** — `bash -n deploy/vless-agent.sh` → OK.
- [ ] **Step 3: Commit + push** — `git commit -am "feat(trojan): agent syncs trojan inbound clients"`

---

## Task 8: Node setup — `deploy/trojan-add.sh`

**Files:** Create `deploy/trojan-add.sh`

Adds a Trojan inbound + self-signed cert to an EXISTING Xray node (preserving the
VLESS-Reality inbound and keys), installs the updated agent, opens ufw, restarts,
and prints the panel registration values.

- [ ] **Step 1: Write the script**

```bash
#!/usr/bin/env bash
# Add a Trojan inbound (TCP) to an already-VLESS node. Idempotent.
# Run on the node as root, in the dir containing the updated vless-agent.sh:
#   TROJAN_SNI=www.apple.com bash trojan-add.sh
set -euo pipefail
TROJAN_SNI="${TROJAN_SNI:-www.apple.com}"
TROJAN_PORT="${TROJAN_PORT:-8443}"
CFG="/usr/local/etc/xray/config.json"
CRT="/usr/local/etc/xray/trojan.crt"; KEY="/usr/local/etc/xray/trojan.key"
SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
command -v jq >/dev/null || { apt-get update -y && apt-get install -y jq; }

# 1) self-signed cert (CN = SNI), readable by the 'nobody' xray user
if [ ! -f "$CRT" ]; then
  openssl ecparam -genkey -name prime256v1 -out "$KEY"
  openssl req -new -x509 -days 36500 -key "$KEY" -out "$CRT" -subj "/CN=${TROJAN_SNI}"
fi
chmod 644 "$KEY" "$CRT"

# 2) add/replace the trojan inbound, preserving everything else
tmp="$(jq --arg crt "$CRT" --arg key "$KEY" --argjson port "$TROJAN_PORT" '
  .inbounds = ((.inbounds | map(select(.tag != "trojan"))) + [{
    tag:"trojan", listen:"0.0.0.0", port:$port, protocol:"trojan",
    settings:{clients:[]},
    streamSettings:{network:"tcp", security:"tls",
      tlsSettings:{certificates:[{certificateFile:$crt, keyFile:$key}]}}
  }])' "$CFG")"
echo "$tmp" > "$CFG"

# 3) updated agent (syncs trojan too) + ufw + restart + sync
install -m 0755 "${SELF_DIR}/vless-agent.sh" /usr/local/bin/vless-agent.sh
if command -v ufw >/dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "${TROJAN_PORT}/tcp" >/dev/null 2>&1 || true; ufw reload >/dev/null 2>&1 || true
fi
systemctl restart xray
systemctl start vless-agent-sync.service || true
sleep 2
echo "trojan inbound clients: $(jq '.inbounds[]|select(.tag=="trojan")|.settings.clients|length' "$CFG")"
echo "REGISTER on panel for this node: trojan_enabled=1 trojan_port=${TROJAN_PORT} trojan_sni=${TROJAN_SNI}"
```

- [ ] **Step 2: Syntax** — `bash -n deploy/trojan-add.sh` → OK.
- [ ] **Step 3: Commit + push.**

---

## Task 9: Roll out to HK1, SG1, JP2

For each node (creds: HK1 `6Zx83TP2YrV54b02`, SG1/`8011`, JP2 `d5fYhwrq04F5TML0`):

- [ ] **Step 1:** scp the updated `deploy/vless-agent.sh` + `deploy/trojan-add.sh` to the node (or `curl` from GitHub raw), then `TROJAN_SNI=www.apple.com bash trojan-add.sh`.
- [ ] **Step 2:** Confirm `ss -lntp | grep :8443` (xray) and `journalctl -u vless-agent-sync -n 5` shows the trojan clients synced.
- [ ] **Step 3:** Register in the panel (per node):

```sql
UPDATE nodes SET trojan_enabled=1, trojan_port=8443, trojan_sni='www.apple.com'
WHERE name='HK1-plain';   -- and SG1-plain, JP2-plain
```

- [ ] **Step 4:** From the panel, verify `panel->node TCP 8443` REACHABLE (open node cloud firewall/ufw if not).
- [ ] **Step 5:** Verify the Surge subscription now contains `HK1-plain-T = trojan, ...` (and SG1/JP2), and Clash/URI still show VLESS, and **the Surge config has zero `vless` lines**.

---

## Task 10: Validate from Surge (user)

- [ ] On the UDP-blocked home WiFi, refresh the Surge subscription, pick `HK1-plain-T` (now Trojan), confirm it connects where HY2 fails. Confirm `traffic_used` increments (Xray stats already meter Trojan).

---

## Self-review

- **Spec coverage:** Trojan inbound+cert (T8), agent sync (T7), endpoint password (T3), TrojanPassword (T1), node model (T2), Surge emit (T5), helper (T4), metering (reused — Xray stats already aggregate all inbounds, no task needed), enforcement (reused — endpoint filter + agent removes from both inbounds), rollout+register (T9), validation (T10). Shadowrocket Trojan intentionally descoped (URI path covers it) — documented in header.
- **Placeholders:** none — every code step is concrete.
- **Type consistency:** `TrojanPassword(string) string` (T1) used in T3/T4; `NodeHasTrojan`/`TrojanSurgeLine` (T4) used in T5; `VlessName` reused for the `-T` name; node fields `TrojanEnabled/TrojanPort/TrojanSNI` consistent T2/T4/T9; endpoint key `password` consistent T3/T7.
