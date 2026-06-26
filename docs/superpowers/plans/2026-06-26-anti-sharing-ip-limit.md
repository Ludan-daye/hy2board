# Anti-sharing per-user IP limit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Limit how many distinct source IPs a user may use concurrently — hard-block the over-limit connection at `Hy2Auth`, kick already-connected excess via HY2 `/kick`, and alert the admin + notify the user over Telegram.

**Architecture:** IPs are only observable at the auth callback, so an in-memory per-user active-IP store (`ipsession.go`) is fed by `Hy2Auth` and consulted for the hard block. A background step in the existing `StartTrafficCache` refresh loop reads HY2 `/online` per-user maps + the store to trim/kick over-limit users and alert. `MaxIPs` (0 = unlimited) is added to `User` + `Plan`, flowing exactly like `TrafficLimit`.

**Tech Stack:** Go (Gin + GORM + SQLite, CGO mattn sqlite), `go-telegram-bot-api/v5`, React 19 + Vite + axios. Backend tests are plain `go test` (no DB needed — all new logic is pure or `httptest`).

---

## Test runner (used in every backend task)

All new tests are DB-free (pure logic / `httptest`). Run them with Go in a throwaway container (the dev box has Docker; if not, run the same `go test` on the server after rsync):

```bash
GOTEST() { docker run --rm -v /root/ludandaye/ladder/hy2board:/app -v hy2gocache:/go -w /app \
  -e CGO_ENABLED=1 golang:1.26-alpine sh -c "apk add --no-cache gcc musl-dev >/dev/null 2>&1 && go test $*"; }
```
Expected pass output ends with `ok  github.com/ludandaye/hy2board/internal/...`.

---

## File structure

| File | Responsibility |
|---|---|
| `internal/service/ipsession.go` (new) | in-memory per-user active-IP store: Touch/Trim/Distinct/RecentlyBlocked/ShouldAlert |
| `internal/service/ipsession_test.go` (new) | store unit tests |
| `internal/service/traffic.go` (modify) | add `GetNodeOnlineMap` (per-user counts) + `KickUser` (HY2 `/kick`) |
| `internal/service/traffic_test.go` (new) | httptest for the two new funcs |
| `internal/service/sharing.go` (new) | `planSharingActions` (pure selector) + `enforceSharing` glue + TG notify + message formatters |
| `internal/service/sharing_test.go` (new) | selector + message formatter tests |
| `internal/service/cache.go` (modify) | `NodeSnapshot.OnlineUsers`; call `GetNodeOnlineMap`; call `enforceSharing` each refresh |
| `internal/handler/hy2auth.go` (modify) | hard block: parse IP → `TouchIP`; `ipFromAddr` helper |
| `internal/handler/hy2auth_test.go` (new) | `ipFromAddr` unit test |
| `internal/model/user.go`, `plan.go` (modify) | `MaxIPs` field |
| `internal/handler/user.go`, `plan.go` (modify) | `MaxIPs` in request structs + plan→user copy |
| `web/src/components/UserEditModal.tsx`, `PlanEditModal.tsx` (modify) | `max_ips` editor field |

---

## Task 1: IP session store (`ipsession.go`)

**Files:** Create `internal/service/ipsession.go`, `internal/service/ipsession_test.go`

- [ ] **Step 1: Write the failing test** — `internal/service/ipsession_test.go`:

```go
package service

import (
	"testing"
	"time"
)

func TestTouchIP(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	now := time.Unix(1000, 0)
	// first two distinct IPs under limit 2 -> allowed
	if ok, n := TouchIP(1, "1.1.1.1", 2, now); !ok || n != 1 {
		t.Fatalf("ip1 want allowed,1 got %v,%d", ok, n)
	}
	if ok, n := TouchIP(1, "2.2.2.2", 2, now); !ok || n != 2 {
		t.Fatalf("ip2 want allowed,2 got %v,%d", ok, n)
	}
	// same IP again -> allowed, no count increase
	if ok, n := TouchIP(1, "1.1.1.1", 2, now); !ok || n != 2 {
		t.Fatalf("ip1 repeat want allowed,2 got %v,%d", ok, n)
	}
	// new third IP over limit -> blocked, not recorded
	if ok, n := TouchIP(1, "3.3.3.3", 2, now); ok || n != 2 {
		t.Fatalf("ip3 want blocked,2 got %v,%d", ok, n)
	}
	if !RecentlyBlocked(1, time.Minute, now) {
		t.Fatal("expected RecentlyBlocked true after a block")
	}
	// limit 0 = unlimited -> always allowed
	if ok, _ := TouchIP(2, "9.9.9.9", 0, now); !ok {
		t.Fatal("limit 0 should always allow")
	}
}

func TestWindowExpiry(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	now := time.Unix(1000, 0)
	TouchIP(1, "1.1.1.1", 5, now)
	if DistinctActiveIPs(1, now) != 1 {
		t.Fatal("want 1 active")
	}
	later := now.Add(ipWindow + time.Second)
	if DistinctActiveIPs(1, later) != 0 {
		t.Fatal("want 0 active after window")
	}
}

func TestTrimOverLimit(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	base := time.Unix(1000, 0)
	// 3 IPs recorded at increasing times under a high limit
	TouchIP(1, "1.1.1.1", 9, base)
	TouchIP(1, "2.2.2.2", 9, base.Add(time.Second))
	TouchIP(1, "3.3.3.3", 9, base.Add(2*time.Second))
	// now lower the limit to 2 -> trims oldest 1, keeps 2 newest
	trimmed, distinct := TrimOverLimit(1, 2, base.Add(3*time.Second))
	if trimmed != 1 || distinct != 2 {
		t.Fatalf("want trimmed1,distinct2 got %d,%d", trimmed, distinct)
	}
	// the trimmed (oldest) IP is gone; a re-auth from it is now blocked
	if ok, _ := TouchIP(1, "1.1.1.1", 2, base.Add(4*time.Second)); ok {
		t.Fatal("trimmed IP should be blocked on re-auth at limit")
	}
}

func TestShouldAlertSharing(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	now := time.Unix(1000, 0)
	TouchIP(1, "1.1.1.1", 1, now) // create state
	if !ShouldAlertSharing(1, now) {
		t.Fatal("first alert should fire")
	}
	if ShouldAlertSharing(1, now.Add(time.Minute)) {
		t.Fatal("second alert within 30m should be suppressed")
	}
	if !ShouldAlertSharing(1, now.Add(alertEvery+time.Second)) {
		t.Fatal("alert after window should fire")
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined symbols):

```bash
GOTEST "./internal/service/ -run 'TestTouchIP|TestWindowExpiry|TestTrimOverLimit|TestShouldAlertSharing' -v"
```
Expected: compile error `undefined: ipStore` / `TouchIP` etc.

- [ ] **Step 3: Implement** — `internal/service/ipsession.go`:

```go
package service

import (
	"sort"
	"sync"
	"time"
)

const (
	ipWindow   = 15 * time.Minute // an IP is "active" if seen (re-authed) within this window
	alertEvery = 30 * time.Minute // per-user alert rate limit
)

type userIPState struct {
	ips         map[string]time.Time // ip -> last auth time
	lastAlert   time.Time
	lastBlocked time.Time
}

var (
	ipMu    sync.Mutex
	ipStore = map[uint]*userIPState{}
)

func ipStateLocked(userID uint) *userIPState {
	s := ipStore[userID]
	if s == nil {
		s = &userIPState{ips: map[string]time.Time{}}
		ipStore[userID] = s
	}
	return s
}

func pruneLocked(s *userIPState, now time.Time) {
	for ip, t := range s.ips {
		if now.Sub(t) > ipWindow {
			delete(s.ips, ip)
		}
	}
}

// TouchIP records an auth event. Returns whether the connection is allowed and the
// resulting distinct-IP count. A known IP always refreshes/allows; a new IP is allowed
// only when under the limit (0 = unlimited). A blocked attempt is NOT recorded but stamps
// lastBlocked for the enforcement loop to alert on.
func TouchIP(userID uint, ip string, limit int, now time.Time) (bool, int) {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStateLocked(userID)
	pruneLocked(s, now)
	if _, ok := s.ips[ip]; ok {
		s.ips[ip] = now
		return true, len(s.ips)
	}
	if limit == 0 || len(s.ips) < limit {
		s.ips[ip] = now
		return true, len(s.ips)
	}
	s.lastBlocked = now
	return false, len(s.ips)
}

// TrimOverLimit prunes stale IPs, then if the user holds more than limit IPs (e.g. the
// limit was just lowered) removes the oldest until len==limit. Returns how many were
// trimmed and the final distinct count.
func TrimOverLimit(userID uint, limit int, now time.Time) (int, int) {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil {
		return 0, 0
	}
	pruneLocked(s, now)
	if limit <= 0 || len(s.ips) <= limit {
		return 0, len(s.ips)
	}
	type kv struct {
		ip string
		t  time.Time
	}
	arr := make([]kv, 0, len(s.ips))
	for ip, t := range s.ips {
		arr = append(arr, kv{ip, t})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].t.Before(arr[j].t) }) // oldest first
	remove := len(arr) - limit
	for i := 0; i < remove; i++ {
		delete(s.ips, arr[i].ip)
	}
	return remove, len(s.ips)
}

func DistinctActiveIPs(userID uint, now time.Time) int {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil {
		return 0
	}
	pruneLocked(s, now)
	return len(s.ips)
}

func RecentlyBlocked(userID uint, within time.Duration, now time.Time) bool {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil || s.lastBlocked.IsZero() {
		return false
	}
	return now.Sub(s.lastBlocked) <= within
}

func ShouldAlertSharing(userID uint, now time.Time) bool {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil {
		return false
	}
	if !s.lastAlert.IsZero() && now.Sub(s.lastAlert) < alertEvery {
		return false
	}
	s.lastAlert = now
	return true
}
```

- [ ] **Step 4: Run, expect PASS** — same command as Step 2 → `ok  .../internal/service`.
- [ ] **Step 5: Commit** — `git add internal/service/ipsession.go internal/service/ipsession_test.go && git commit -m "feat(anti-sharing): in-memory per-user IP session store"`

---

## Task 2: Model fields `MaxIPs`

**Files:** Modify `internal/model/user.go`, `internal/model/plan.go`, `internal/handler/user.go`

- [ ] **Step 1:** In `internal/model/user.go`, add after the `TrafficUsed` line (currently line 21):

```go
	MaxIPs        int       `gorm:"default:0" json:"max_ips"` // 0 = unlimited; max distinct concurrent source IPs
```

- [ ] **Step 2:** In `internal/model/plan.go`, add after the `TrafficLimit` line (line 8):

```go
	MaxIPs        int    `gorm:"default:0" json:"max_ips"` // 0 = unlimited
```

- [ ] **Step 3:** In `internal/handler/user.go`, in `planUserSubscriptionUpdates` (the map returned ~line 58-73), add the entry so plan→user apply/sync copies it:

```go
		"max_ips":        p.MaxIPs,
```

- [ ] **Step 4: Verify build** (GORM AutoMigrate adds the columns at startup, non-destructive):

```bash
GOTEST "./internal/model/ ./internal/handler/ -run xxx" 2>&1 | tail -3   # compiles (no tests match; just typecheck)
```
Expected: `ok` or `no test files` — **no compile error**.

- [ ] **Step 5: Commit** — `git add internal/model/user.go internal/model/plan.go internal/handler/user.go && git commit -m "feat(anti-sharing): MaxIPs field on User+Plan, copied on plan apply"`

---

## Task 3: `GetNodeOnlineMap` + `KickUser` (`traffic.go`)

**Files:** Modify `internal/service/traffic.go`; Create `internal/service/traffic_test.go`

- [ ] **Step 1: Write the failing test** — `internal/service/traffic_test.go`:

```go
package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestGetNodeOnlineMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/online" {
			t.Errorf("want /online got %s", r.URL.Path)
		}
		w.Write([]byte(`{"alice":2,"bob":1}`))
	}))
	defer srv.Close()
	m, err := GetNodeOnlineMap(model.Node{TrafficAPI: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if m["alice"] != 2 || m["bob"] != 1 || len(m) != 2 {
		t.Fatalf("bad map: %v", m)
	}
}

func TestKickUser(t *testing.T) {
	var gotBody, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kick" || r.Method != http.MethodPost {
			t.Errorf("want POST /kick got %s %s", r.Method, r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()
	if err := KickUser(model.Node{TrafficAPI: srv.URL, TrafficSecret: "sek"}, "alice"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, "alice") {
		t.Fatalf("body missing username: %q", gotBody)
	}
	if gotAuth != "sek" {
		t.Fatalf("auth header want sek got %q", gotAuth)
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `GOTEST "./internal/service/ -run 'TestGetNodeOnlineMap|TestKickUser' -v"` → `undefined: GetNodeOnlineMap` / `KickUser`.

- [ ] **Step 3: Implement** — in `internal/service/traffic.go`, add `"bytes"` to imports, then append:

```go
// GetNodeOnlineMap returns HY2's per-user online connection counts (username -> count).
func GetNodeOnlineMap(node model.Node) (map[string]int, error) {
	if node.TrafficAPI == "" {
		return nil, fmt.Errorf("no traffic API configured")
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", node.TrafficAPI+"/online", nil)
	if err != nil {
		return nil, err
	}
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]int
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// KickUser disconnects all of a user's connections on a node via HY2 trafficStats /kick.
func KickUser(node model.Node, username string) error {
	if node.TrafficAPI == "" {
		return fmt.Errorf("no traffic API configured")
	}
	payload, _ := json.Marshal([]string{username})
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("POST", node.TrafficAPI+"/kick", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("kick status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 4: Run, expect PASS** — same as Step 2 → `ok`.
- [ ] **Step 5: Commit** — `git add internal/service/traffic.go internal/service/traffic_test.go && git commit -m "feat(anti-sharing): GetNodeOnlineMap + KickUser HY2 client"`

---

## Task 4: `Hy2Auth` hard block

**Files:** Modify `internal/handler/hy2auth.go`; Create `internal/handler/hy2auth_test.go`

- [ ] **Step 1: Write the failing test** — `internal/handler/hy2auth_test.go`:

```go
package handler

import "testing"

func TestIPFromAddr(t *testing.T) {
	cases := map[string]string{
		"1.2.3.4:5678":      "1.2.3.4",
		"1.2.3.4":           "1.2.3.4",
		"[2001:db8::1]:443": "2001:db8::1",
		"":                  "",
	}
	for in, want := range cases {
		if got := ipFromAddr(in); got != want {
			t.Errorf("ipFromAddr(%q)=%q want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `GOTEST "./internal/handler/ -run TestIPFromAddr -v"` → `undefined: ipFromAddr`.

- [ ] **Step 3: Implement** — rewrite `internal/handler/hy2auth.go`:

```go
package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

type Hy2AuthRequest struct {
	Addr string `json:"addr"`
	Auth string `json:"auth"`
	TX   int64  `json:"tx"`
}

// ipFromAddr strips the port from HY2's client address, leaving the bare host/IP.
func ipFromAddr(addr string) string {
	if addr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func Hy2Auth(c *gin.Context) {
	var req Hy2AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	var user model.User
	if database.DB.Where("hy2_password = ?", req.Auth).First(&user).Error != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	if !user.IsActive() {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	// Per-user IP concurrency limit (0 = unlimited). Record the source IP either way so the
	// admin UI can show current IP counts; only block when a limit is set and exceeded.
	if ip := ipFromAddr(req.Addr); ip != "" {
		allowed, _ := service.TouchIP(user.ID, ip, user.MaxIPs, time.Now())
		if user.MaxIPs > 0 && !allowed {
			c.JSON(http.StatusOK, gin.H{"ok": false})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "id": user.Username})
}
```

- [ ] **Step 4: Run, expect PASS** — same as Step 2 → `ok`.
- [ ] **Step 5: Commit** — `git add internal/handler/hy2auth.go internal/handler/hy2auth_test.go && git commit -m "feat(anti-sharing): Hy2Auth hard-block over-limit source IPs"`

---

## Task 5: Enforcement loop (`sharing.go`) + cache hook

**Files:** Create `internal/service/sharing.go`, `internal/service/sharing_test.go`; Modify `internal/service/cache.go`

- [ ] **Step 1: Write the failing test** — `internal/service/sharing_test.go`:

```go
package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestPlanSharingActions(t *testing.T) {
	users := []model.User{
		{Model: gormModel(1), Username: "alice", MaxIPs: 2},
		{Model: gormModel(2), Username: "bob", MaxIPs: 2},
	}
	snaps := []NodeSnapshot{
		{Name: "HK1", OnlineUsers: map[string]int{"alice": 1, "bob": 1}},
		{Name: "JP2", OnlineUsers: map[string]int{"alice": 1}},
	}
	// alice is over (trimmed>0) on both nodes; bob only had a blocked attempt (no kick).
	state := func(id uint) (trimmed, distinct int, blocked bool) {
		if id == 1 {
			return 1, 2, true
		}
		return 0, 2, true
	}
	acts := planSharingActions(users, snaps, state, func(uint) bool { return true })
	if len(acts) != 2 {
		t.Fatalf("want 2 actions got %d", len(acts))
	}
	var alice *sharingAction
	for i := range acts {
		if acts[i].User.Username == "alice" {
			alice = &acts[i]
		}
	}
	if alice == nil || len(alice.KickNodeIdx) != 2 || !alice.Alert {
		t.Fatalf("alice should kick 2 nodes and alert: %+v", alice)
	}
}

func TestFormatSharingAlert(t *testing.T) {
	s := formatSharingAlert("alice", 3, 2, []string{"HK1", "JP2"})
	if !strings.Contains(s, "alice") || !strings.Contains(s, "HK1") {
		t.Fatalf("alert text missing fields: %s", s)
	}
}
```

Add this tiny helper at the bottom of the test file (constructs a `gorm.Model` with an ID):
```go
func gormModel(id uint) gorm.Model { return gorm.Model{ID: id} }
```
and import `"gorm.io/gorm"`.

- [ ] **Step 2: Run, expect FAIL** — `GOTEST "./internal/service/ -run 'TestPlanSharingActions|TestFormatSharingAlert' -v"` → undefined symbols.

- [ ] **Step 3: Implement** — `internal/service/sharing.go`:

```go
package service

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

type sharingAction struct {
	User        model.User
	KickNodeIdx []int // indices into the snapshot slice where this user is online
	Distinct    int
	Alert       bool
}

// planSharingActions is the pure core: given limited users, the current node snapshots, a
// per-user state function (trimmed/distinct/blocked) and an alert-gate, decide who to kick
// where and whether to alert. A user is acted on if they were trimmed (over limit) or had a
// recent blocked attempt. Kicks target only the nodes where they're currently online.
func planSharingActions(
	users []model.User,
	snaps []NodeSnapshot,
	state func(userID uint) (trimmed, distinct int, blocked bool),
	shouldAlert func(userID uint) bool,
) []sharingAction {
	var out []sharingAction
	for _, u := range users {
		trimmed, distinct, blocked := state(u.ID)
		if trimmed == 0 && !blocked {
			continue
		}
		act := sharingAction{User: u, Distinct: distinct}
		if trimmed > 0 {
			for i := range snaps {
				if snaps[i].OnlineUsers[u.Username] > 0 {
					act.KickNodeIdx = append(act.KickNodeIdx, i)
				}
			}
		}
		act.Alert = shouldAlert(u.ID)
		out = append(out, act)
	}
	return out
}

func formatSharingAlert(username string, distinct, limit int, kicked []string) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "⚠️ 共享告警\n用户：%s\n当前不同IP：%d（上限 %d）\n", username, distinct, limit)
	if len(kicked) > 0 {
		fmt.Fprintf(b, "已踢节点：%s\n", strings.Join(kicked, ", "))
	} else {
		b.WriteString("有新IP尝试接入被拦截（疑似共享）\n")
	}
	return b.String()
}

func formatSharingUserNotice(limit int) string {
	return fmt.Sprintf(
		"⚠️ 安全提醒\n检测到你的账号从多个网络同时使用，已超过套餐上限（%d 个）。多余连接已断开。\n如非本人操作，请尽快修改密码。",
		limit,
	)
}

// enforceSharing runs each refresh cycle: trim/kick over-limit users, alert admin, notify user.
func enforceSharing(nodes []model.Node, snaps []NodeSnapshot) {
	var users []model.User
	if err := database.DB.Where("max_ips > 0").Find(&users).Error; err != nil {
		return
	}
	now := time.Now()
	state := func(id uint) (int, int, bool) {
		trimmed, distinct := TrimOverLimit(id, ipLimitFor(users, id), now)
		return trimmed, distinct, RecentlyBlocked(id, 90*time.Second, now)
	}
	for _, act := range planSharingActions(users, snaps, state, func(id uint) bool { return ShouldAlertSharing(id, now) }) {
		var kicked []string
		for _, idx := range act.KickNodeIdx {
			if err := KickUser(nodes[idx], act.User.Username); err == nil {
				kicked = append(kicked, snaps[idx].Name)
			}
		}
		if act.Alert {
			notifyAdminTG(formatSharingAlert(act.User.Username, act.Distinct, act.User.MaxIPs, kicked))
			notifyUserTG(act.User.TelegramID, formatSharingUserNotice(act.User.MaxIPs))
		}
	}
}

func ipLimitFor(users []model.User, id uint) int {
	for _, u := range users {
		if u.ID == id {
			return u.MaxIPs
		}
	}
	return 0
}

// notifyAdminTG sends to the bound admin (private), never the customer group.
func notifyAdminTG(text string) {
	if !config.C.HasTelegram() {
		return
	}
	adminID, ok := getAdminTelegramID()
	if !ok {
		return
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return
	}
	bot.Send(tgbotapi.NewMessage(adminID, text))
}

func notifyUserTG(tgID int64, text string) {
	if tgID == 0 || !config.C.HasTelegram() {
		return
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return
	}
	bot.Send(tgbotapi.NewMessage(tgID, text))
}
```

- [ ] **Step 4: Run, expect PASS** — same as Step 2 → `ok`.

- [ ] **Step 5: Wire into cache.go.** In `internal/service/cache.go`: (a) add `OnlineUsers map[string]int` to the `NodeSnapshot` struct; (b) replace the online goroutine body that calls `GetNodeOnline(n)` with the map variant; (c) call `enforceSharing` after `wg.Wait()`.

(b) — replace:
```go
				go func() {
					defer inner.Done()
					if o, err := GetNodeOnline(n); err == nil {
						snap.Online = o
					}
				}()
```
with:
```go
				go func() {
					defer inner.Done()
					if m, err := GetNodeOnlineMap(n); err == nil {
						snap.OnlineUsers = m
						snap.Online = len(m)
					}
				}()
```

(c) — immediately after the `wg.Wait()` that closes the per-node fan-out, add:
```go
		enforceSharing(nodes, results)
```

- [ ] **Step 6: Verify build + tests** — `GOTEST "./internal/service/ -v 2>&1 | tail -5"` → `ok`.
- [ ] **Step 7: Commit** — `git add internal/service/sharing.go internal/service/sharing_test.go internal/service/cache.go && git commit -m "feat(anti-sharing): enforcement loop (trim/kick/alert) wired into refresh"`

---

## Task 6: User & Plan request structs (`MaxIPs` in/out)

**Files:** Modify `internal/handler/user.go`, `internal/handler/plan.go`

- [ ] **Step 1:** `internal/handler/user.go` — add to `CreateUserRequest`:
```go
	MaxIPs       int    `json:"max_ips"`
```
and in `CreateUser`, in the `model.User{...}` literal, add:
```go
		MaxIPs:       req.MaxIPs,
```
Add to `UpdateUserRequest`:
```go
	MaxIPs       *int    `json:"max_ips"`
```
and in `UpdateUser`, alongside the other `if req.X != nil` blocks:
```go
	if req.MaxIPs != nil {
		updates["max_ips"] = *req.MaxIPs
	}
```

- [ ] **Step 2:** `internal/handler/plan.go` — add to `CreatePlanRequest`:
```go
	MaxIPs        int    `json:"max_ips"`
```
and in `CreatePlan`'s `model.Plan{...}` literal:
```go
		MaxIPs:        req.MaxIPs,
```
Add to `UpdatePlanRequest`:
```go
	MaxIPs        *int    `json:"max_ips"`
```
and in `UpdatePlan`'s updates map:
```go
	if req.MaxIPs != nil {
		updates["max_ips"] = *req.MaxIPs
	}
```

- [ ] **Step 3: Verify build** — `GOTEST "./internal/handler/ -run xxx 2>&1 | tail -3"` → compiles.
- [ ] **Step 4: Commit** — `git add internal/handler/user.go internal/handler/plan.go && git commit -m "feat(anti-sharing): MaxIPs in user/plan create+update APIs"`

---

## Task 7: Frontend editor fields

**Files:** Modify `web/src/components/UserEditModal.tsx`, `web/src/components/PlanEditModal.tsx`

- [ ] **Step 1: UserEditModal.** Add to the `EditableUser` interface:
```tsx
  max_ips?: number
```
In `AccessTab`'s `useState` form initializer, add:
```tsx
    max_ips: user.max_ips || 0,
```
In the `save` body object, add:
```tsx
        max_ips: form.max_ips,
```
After the Traffic Limit input block, add an identical input:
```tsx
      <div>
        <label className="text-xs text-ink-faint uppercase tracking-wider">
          同时在线 IP 上限 (0 = 不限)
        </label>
        <input
          type="number"
          min={0}
          value={form.max_ips}
          onChange={(e) => setForm({ ...form, max_ips: +e.target.value })}
          className={inputCls + " mt-1"}
        />
      </div>
```

- [ ] **Step 2: PlanEditModal.** Add to the `Plan` interface and `FormState`:
```tsx
  max_ips?: number
```
(FormState: `max_ips: number`). In the form initial state add `max_ips: plan?.max_ips || 0,` (match how other fields seed). In the edit-mode `body` object inside `save`, add:
```tsx
        max_ips: form.max_ips,
```
In the Basic tab grid (next to Duration days), add:
```tsx
      <input
        type="number"
        placeholder="IP 上限 (0=不限)"
        value={form.max_ips || ""}
        onChange={(e) => setForm((f) => ({ ...f, max_ips: +e.target.value }))}
        className={inputCls}
      />
```

- [ ] **Step 3: Commit** — `git add web/src/components/UserEditModal.tsx web/src/components/PlanEditModal.tsx && git commit -m "feat(anti-sharing): MaxIPs field in user/plan editors"`

---

## Task 8: Build, deploy, controlled live test

- [ ] **Step 1:** Full backend test pass — `GOTEST "./internal/... 2>&1 | tail -15"` → all `ok`, no failures.
- [ ] **Step 2:** Sync + tag rollback + build:
```bash
cd /root/ludandaye/ladder/hy2board
rsync -az --no-perms --omit-dir-times -e "sshpass -p 8011 ssh -o StrictHostKeyChecking=no" internal/ root@76.13.217.10:/opt/hy2board/internal/
rsync -az --no-perms --omit-dir-times -e "sshpass -p 8011 ssh -o StrictHostKeyChecking=no" web/src/ root@76.13.217.10:/opt/hy2board/web/src/
sshpass -p 8011 ssh -o StrictHostKeyChecking=no root@76.13.217.10 'cd /opt/hy2board && docker tag "$(docker inspect hy2board-hy2board-1 --format "{{.Image}}")" hy2board-hy2board:rollback-presharing && docker compose build > /tmp/buildshare.log 2>&1; echo EXIT $?; grep -nE "error|FAIL" /tmp/buildshare.log | head; tail -4 /tmp/buildshare.log'
```
Expected: `EXIT 0`, image built.
- [ ] **Step 3:** Up + verify migration column + auth still works:
```bash
sshpass -p 8011 ssh -o StrictHostKeyChecking=no root@76.13.217.10 'cd /opt/hy2board && docker compose up -d && sleep 8 && python3 -c "import sqlite3;print([r for r in sqlite3.connect(\"data/hy2board.db\").execute(\"select name from pragma_table_info(\x27users\x27) where name=\x27max_ips\x27\")])" && curl -s -X POST localhost:9000/api/admin/login -H "Content-Type: application/json" -d "{\"username\":\"ludandaye\",\"password\":\"8011\"}" | head -c 25'
```
Expected: `[('max_ips',)]` (column exists) and a token.
- [ ] **Step 4: Verify `/kick` works on the live HY2 version** (the one risk). Kick a dummy username (no-op, safe) on one node and confirm a 2xx:
```bash
sshpass -p 8011 ssh -o StrictHostKeyChecking=no root@76.13.217.10 'cd /opt/hy2board && python3 -c "
import sqlite3,urllib.request,json
api,sec=sqlite3.connect(\"data/hy2board.db\").execute(\"select traffic_api,traffic_secret from nodes where traffic_api like \x27http%\x27 limit 1\").fetchone()
req=urllib.request.Request(api+\"/kick\",data=json.dumps([\"__nonexistent__\"]).encode(),headers={\"Authorization\":sec,\"Content-Type\":\"application/json\"},method=\"POST\")
print(\"kick status\", urllib.request.urlopen(req,timeout=5).getcode())
"'
```
Expected: `kick status 200`. **If this 404s, HY2 is too old for `/kick`** — the hard-block + alert still work; note it and skip kicking (the loop's KickUser error is already non-fatal).
- [ ] **Step 5: Controlled end-to-end (you + me):** set your own test account `MaxIPs=1` in the User editor. Connect from one device (works), then a second network/IP (phone hotspot) — the second should fail to pass auth, you get a TG admin alert, and (if your account has a TG id) a user notice. Then set it back to `0`. Confirm normal users (limit 0) are unaffected throughout.
- [ ] **Step 6: Commit + push** — `git push origin HEAD:main`.
- [ ] **Step 7: Memory** — append to `/root/.claude/projects/-root-ludandaye-ladder/memory/hy2board-panel-deployment.md`: anti-sharing IP limit live (MaxIPs on User/Plan, ipsession.go store, Hy2Auth hard-block, enforceSharing kick+TG, rollback `rollback-presharing`, HY2-only).

---

## Self-Review

**1. Spec coverage:**
- Hard block at auth → Task 4. Kick excess → Task 3 (`KickUser`) + Task 5 (loop). Alert admin + notify user → Task 5 (`notifyAdminTG`/`notifyUserTG`, private admin via `getAdminTelegramID`). Distinct-IP basis + 15m window → Task 1. Per-plan template + per-user `MaxIPs`, 0=unlimited, copied on apply → Tasks 2/6. In-memory auth-driven store → Task 1. HY2-only → no VLESS/Trojan touched. Rate-limited alerts (30m) → Task 1 `ShouldAlertSharing`. Trim-on-limit-decrease convergence → Task 1 `TrimOverLimit` + Task 5. `/kick` version resilience → Task 8 Step 4 (non-fatal). Deploy/rollback/migration → Task 8. **All spec sections mapped.**

**2. Placeholder scan:** every code step has complete code; commands have expected output; no TBD/"handle errors" hand-waving. The one external unknown (HY2 `/kick` support) is an explicit verification step with a documented fallback, not a placeholder.

**3. Type/name consistency:** store API (`TouchIP`, `TrimOverLimit`, `DistinctActiveIPs`, `RecentlyBlocked`, `ShouldAlertSharing`, consts `ipWindow`/`alertEvery`, `ipStore`/`userIPState`) defined in Task 1 and used identically in Tasks 4–5. `GetNodeOnlineMap`/`KickUser` (Task 3) used in Task 5. `NodeSnapshot.OnlineUsers` added in Task 5 and read by `planSharingActions`. `MaxIPs` (model Task 2) ↔ `max_ips` JSON ↔ request structs (Task 6) ↔ editor `max_ips` (Task 7) ↔ `planUserSubscriptionUpdates` copy (Task 2) — all consistent. `getAdminTelegramID`/`config.C.HasTelegram`/`config.C.Telegram.BotToken` match the existing `telegram_admin.go`.
