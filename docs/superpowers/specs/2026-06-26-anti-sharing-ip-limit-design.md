# Anti-account-sharing: per-user IP concurrency limit

Date: 2026-06-26
Status: approved (brainstorming)

## Problem

The panel has no defense against account sharing/reselling. A single paid account can be
used from many places at once with no limit — the biggest revenue leak for a VPN reseller.
The auth path (`Hy2Auth`) currently only checks `IsActive()` (enabled / expiry / traffic);
it never looks at concurrency. HY2's per-node `/online` API already exposes a per-user
**connection-count** map (confirmed live, e.g. `{"fangzhenghao":1,"ludandaye":2}`) but the
panel collapses it to `len()`. Crucially, `/online` gives counts, **not IPs** — the only
place the panel can observe a connection's real source IP is the auth callback's `addr`.

## Goal

Limit how many **distinct source IPs** a user may use concurrently. Over the limit →
**reject the new connection at auth (hard block)** + **kick the user's excess connections**
+ **notify the user and alert the admin** over Telegram. Per-plan default limit, per-user
override, `0 = unlimited`. Ship with zero impact on existing users until limits are set.

## Decisions (from brainstorming)

- **Enforcement = both:** hard block at auth (`Hy2Auth` returns `ok:false`) **and** a
  background loop that kicks already-connected excess + alerts.
- **Basis = distinct source IPs** (same public IP across many nodes counts as 1 — naturally
  immune to the "one device url-tests many nodes" inflation; a household NAT counts as 1).
- **Action = kick excess + notify user (TG) + alert admin (TG).**
- **Config = per-plan template + per-user value, `0 = unlimited`;** existing users default 0.

## Non-goals (v1)

- **HY2 only.** VLESS/Trojan (the TCP fallback for the UDP-blocked minority) don't pass
  through `Hy2Auth` and Xray doesn't hand the panel per-connection IPs — deferred to a later
  phase (would need the on-node agent to parse Xray access logs).
- **No DB-persisted session store.** In-memory; rebuilds from auth events after a restart
  (brief under-enforcement window accepted). Panel is single-instance.
- **Node-whitelist enforcement is NOT part of this** (it's a separate known gap / feature).
  We touch `Hy2Auth` only to add the IP check, preserving all existing behavior.
- No per-IP selective kick (HY2 `/kick` is by username = kicks all that user's conns on a
  node; that's fine — legit devices reconnect and re-auth admits them up to the limit).

## Architecture

IP data is only available at the auth callback, so the design is **panel-side**: an
in-memory per-user active-IP table fed by `Hy2Auth`, plus a background enforcement loop that
reuses the existing node-stats poll.

```
client → HY2 node → POST /api/auth/hy2 {addr, auth, tx}
                       └→ Hy2Auth: IsActive? → parse IP → ipsession.Touch(userID, ip, MaxIPs)
                            ├ known/refreshed IP, or under limit → record → ok:true
                            └ new IP that would exceed MaxIPs   → ok:false (+ rate-limited alert)

every stats-refresh cycle:
  enforcement loop → for each user with MaxIPs>0 and distinctActiveIPs > MaxIPs:
      → HY2 /kick {user} on nodes where /online shows them
      → TG alert admin  +  TG notify user (if TelegramID set)   [rate-limited per user]
```

## Components

### 1. Data model (`internal/model/user.go`, `plan.go`)
Add `MaxIPs int gorm:"default:0" json:"max_ips"` to **both** `User` and `Plan`. `0 =
unlimited`. Runtime authority is `user.MaxIPs`; `plan.MaxIPs` is the template copied to the
user on plan create/apply — identical to how `TrafficLimit`/`NodeIDs` already work (so the
apply-to-user and create paths that copy plan fields must also copy `MaxIPs`). GORM
AutoMigrate adds the column (default 0) — non-destructive, existing rows unaffected.

### 2. IP session store (`internal/service/ipsession.go`, new)
In-memory, mutex-guarded:
```
type ipEntry struct { lastSeen time.Time }
type userIPs struct { ips map[string]time.Time; lastAlert time.Time }
var store map[uint]*userIPs        // userID → state
const Window = 15 * time.Minute
```
- `Touch(userID uint, ip string, limit int) (allowed bool, distinct int)` — prune entries
  older than `Window`; if `ip` already present → refresh, `allowed=true`; else if
  `limit==0` (unlimited) or `len(ips) < limit` → add, `allowed=true`; else `allowed=false`
  (do **not** add the rejected IP). Returns post-op distinct count.
- `DistinctActive(userID) int` — pruned count (for the loop + UI).
- `ShouldAlert(userID, every time.Duration) bool` — rate-limit alerts (sets lastAlert).
- `Prune()` — periodic GC of stale users/IPs (called from the loop).
Tracks **all** online users (tiny: a few IPs each) so the admin UI can show current IP
counts even for unlimited users; **enforcement only acts when `MaxIPs > 0`**.

### 3. `Hy2Auth` change (`internal/handler/hy2auth.go`)
After the existing `IsActive()` check passes, parse the IP from `req.Addr`
(`net.SplitHostPort`, fall back to the raw string; on parse failure → allow, never break
auth), then `allowed, n := ipsession.Touch(user.ID, ip, user.MaxIPs)`. If `!allowed` →
`ok:false` and fire a rate-limited admin alert ("hard block"). Otherwise `ok:true` as today.
No other existing behavior changes.

### 4. Enforcement loop (`internal/service/sharing_enforce.go`, new; called from the
existing stats-refresh in `cache.go`)
For each user with `MaxIPs > 0` where `DistinctActive(userID) > MaxIPs`: collect the nodes
where `/online` lists the username, call HY2 `/kick` for that user on each, and — if
`ShouldAlert(userID, 30m)` — send the admin alert + user notify. Failures (kick error, no
TG) are logged and never block the loop.

### 5. HY2 `/kick` client (`internal/service/traffic.go` alongside `GetNodeOnline`)
`KickUser(node, username) error` — `POST node.TrafficAPI + "/kick"` with the trafficStats
secret in `Authorization` and the documented payload (JSON array of auth ids). On
implementation, verify the deployed HY2 version supports `/kick` by testing with a dummy
username; if unsupported, the loop logs and relies on hard-block + alert (still functional).

### 6. Telegram notify (reuse existing bot)
- **Admin alert:** `⚠️ 共享告警: <user> 当前 <n> 个 IP (上限 <limit>), 节点 [<names>], 已踢。`
- **User notify (if `user.TelegramID > 0`):** friendly zh message — account used from N
  networks over the plan limit, excess disconnected, change password if not you. Skip
  silently when `TelegramID == 0`.

### 7. Admin UI (`web/src`)
- `MaxIPs` number field (label "同时在线 IP 上限 (0=不限)") in the Plan editor
  (`PlanEditModal`) and User editor (`UserEditModal`) — wired through the existing
  create/update request bodies that already carry `traffic_limit` etc.
- Optional: a "当前IP" column in the Users table from `DistinctActive` (a small read-only
  endpoint or fold into the existing users payload) so the admin can spot sharers at a
  glance. Minimal v1 may ship just the editor field; the column is a nice-to-have.

## Effective-limit resolution

Runtime reads `user.MaxIPs` directly (`0` = unlimited). `plan.MaxIPs` is only a template
copied to the user when a plan is created-from/applied-to the user, mirroring the existing
`TrafficLimit` flow. No runtime plan lookup on the hot auth path.

## Error handling / edge cases

- `req.Addr` unparseable / empty → treat as no-IP, **allow** (auth must never break on this).
- Multiple connections from one IP → counts as **1** (the whole point of IP-basis).
- User exactly at limit: existing IP → allowed (refresh); new IP → rejected.
- `MaxIPs == 0` → never block, never kick (still tracked for visibility).
- `/kick` unsupported/old HY2 → logged; hard-block + alert still enforce.
- `user.TelegramID == 0` → skip user notify; admin alert still sent.
- Panel restart → empty store → rebuilds from new auths; brief under-enforcement accepted.
- Alert storm → per-user 30-min rate-limit on alerts/notifies.
- Long-lived connection whose IP entry expires after `Window` → may transiently free a slot;
  the periodic kick (which forces a reconnect+re-auth and resyncs the table) keeps sharing
  impractical. This is a **deterrent**, documented, not an absolute lock.

## Testing

No new infra; Go unit tests like `traffic_accrual_test.go`:
- `ipsession`: same-IP within window doesn't increment; new IP increments; over-limit →
  `allowed=false` and not recorded; window expiry prunes; `DistinctActive` correct;
  `ShouldAlert` rate-limits.
- `Hy2Auth`: active user, new IP over limit → `ok:false`; existing IP → `ok:true`;
  `MaxIPs==0` → `ok:true`; unparseable addr → `ok:true`.
- enforcement selection: given a store + `/online` map, the right users/nodes are chosen.

## Deploy

Hot deploy via `docker compose build` + up at `/opt/hy2board` (rsync from local). DB column
added by GORM AutoMigrate (non-destructive). Rollback image `hy2board-hy2board:rollback-presharing`.
Roll out with all limits at 0 (no behavior change), then set per-plan limits when ready.
Verify with a controlled test (set your own account to a low limit, connect from 2 IPs,
confirm block+kick+alert).
