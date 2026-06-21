# VLESS+Reality TCP fallback (integrated into hy2board)

Date: 2026-06-21
Status: approved (brainstorming) — Phase 1 (pilot)

## Problem

Hysteria2 is UDP/QUIC only. Some users' networks (confirmed: an admin's home
broadband WiFi; cellular works fine) block or throttle UDP, so every HY2 node
fails for them while users on UDP-friendly networks are unaffected. The fix is a
**TCP-based protocol** as a fallback that works where UDP is blocked.

## Goal

Add **VLESS + Reality** on **TCP 443** on each node (coexists with HY2's UDP 443),
fully integrated into hy2board: per-user identity, traffic metered into
`users.traffic_used`, and disable/expiry/over-limit enforced — same controls as HY2.
Clients carry both: HY2 when UDP works (fast), VLESS-TCP as automatic fallback.

## Non-goals (this phase)

- Not rolling out to all nodes yet — **pilot on HK1 (38.47.108.14) only**, then a
  separate Phase 2 rolls out to the other 4 servers and folds setup into the deploy script.
- Not replacing HY2 or the panel. VLESS is an additive fallback.
- No client-app changes — standard Surge/Clash/v2rayN already speak VLESS+Reality.

## Why HK1 is the pilot

HK1 is heavily used by mainland users over HY2 (~6 GB/day), so its IP is provably
reachable from China — only UDP is the question. Adding VLESS-TCP to HK1 and
connecting from the UDP-blocked home WiFi directly validates the whole thesis.
VLESS on TCP 443 is a separate process/transport from HY2 (UDP 443), so it cannot
disrupt HK1's existing HY2 service.

## Architecture

### 1. Node side (HK1 first)

- **Xray-core**, one inbound: VLESS + Reality, listen TCP 443.
  - Reality: `dest` = a real steal-target (default `www.microsoft.com:443`),
    `serverNames` matching, an x25519 keypair generated on the node (private key
    stays on the node), one or more `shortIds`.
  - `clients`: starts empty; managed live via the API (no restart on user changes).
  - Xray **gRPC API** bound to localhost: `HandlerService` (AddUser/RemoveUser),
    `StatsService` (per-user uplink/downlink counters). Stats enabled per-user.
- **vless-agent** (small Go or shell+jq service; systemd unit):
  - Every ~30s: `GET https://vpn.linkbyfree.com/api/node/vless/clients` (node-secret
    auth) → desired active user set `{uuid, email}` → diff vs current Xray users →
    AddUser/RemoveUser via gRPC.
  - Serves `GET /vless/stats` (localhost or node-secret auth) translating Xray
    StatsService into JSON `{email: {tx, rx}}` cumulative counters, for the panel to poll.

### 2. Panel side (hy2board code)

- **Node model migration**: add columns `vless_enabled bool`, `vless_port int`,
  `reality_pubkey text`, `reality_shortid text`, `reality_sni text` (steal-target SNI
  clients use), `vless_stats_api text` (agent stats URL), `vless_stats_secret text`.
  Private key never touches the panel.
- **Client-list endpoint** `GET /api/node/vless/clients` (auth: shared node secret
  from config): returns active users `[{uuid, email}]` where `email = username` and
  `uuid = UUIDv5(NS, username)`. "Active" = `IsActive()` (enabled ∧ not expired ∧ not
  over traffic limit) ∧ user is allowed on this node by `EffectiveNodeIDs`.
- **Traffic metering**: panel polls each VLESS node's `vless_stats_api` (cumulative
  per-user tx/rx) and feeds the existing accrual (`trafficUsageDelta` →
  `persistTrafficUsage`). Same per-node-per-user delta logic already shipped for HY2,
  so VLESS and HY2 usage both accumulate into one `traffic_used`. Closes the
  limit-bypass hole.
- **Subscription generation** (`subscription.go`): for each `vless_enabled` healthy
  node the user may use, emit a VLESS+Reality proxy line (server=node host, port=
  vless_port, uuid=user UUID, security=reality, pbk=reality_pubkey, sid=
  reality_shortid, sni=reality_sni, flow=xtls-rprx-vision, fp=chrome). Add these node
  names into the existing Auto/Manual groups (after the HY2 entries) so clients
  failover UDP→TCP automatically. Cover Clash, Surge, and v2rayN URI formats.

### 3. Identity & enforcement

- `uuid = UUIDv5(fixed namespace, username)` — deterministic, reproducible on every
  node and in the panel, no new storage. Xray `email = username` aligns VLESS stats
  with the existing `traffic_logs` username keying, so the accrual is uniform.
- Enforcement: a disabled / expired / over-limit user drops out of
  `/api/node/vless/clients`; the agent removes them from Xray within one sync cycle
  (~30s). Same effect as HY2's `IsActive()` rejection at connect.

## Data flow

```
sync:     agent --GET /api/node/vless/clients--> panel ;  agent --gRPC Add/RemoveUser--> Xray   (~30s)
meter:    panel --poll vless_stats_api--> agent(Xray stats) --> trafficUsageDelta --> traffic_used  (~3s)
deliver:  client --GET /api/sub/:token--> panel  (subscription now includes HK1 VLESS node)
enforce:  disable/expire/over-limit -> IsActive()=false -> dropped from client-list -> agent removes from Xray
```

## Error handling

- Agent can't reach panel: keep last-known user set (fail-open for existing users;
  don't wipe Xray clients on a transient panel/network blip).
- Panel can't reach a node's stats API: that node contributes 0 delta that tick
  (no negative/spurious accrual — `trafficUsageDelta` already baselines on first sight).
- Counter resets (Xray restart): handled by the existing `counterDelta` reset guard.
- Reality misconfig: Xray fails fast on start; pilot is isolated to HK1.

## Testing

- **Panel (TDD, reuse in-memory sqlite pattern):**
  - UUID derivation is deterministic and stable per username.
  - `/api/node/vless/clients` returns only active+allowed users; excludes disabled/
    expired/over-limit; rejects wrong node secret.
  - VLESS subscription line generation (Clash/Surge/URI) for a user+node.
  - Traffic accrual extension: VLESS stats deltas increment `traffic_used` and a
    user crossing the limit flips `IsActive()` false (extends existing accrual tests).
- **Node/integration (manual on HK1):** Xray up on TCP 443; agent syncs a test user;
  stats flow to panel; subscription includes the VLESS node.
- **End-to-end success criterion:** from the UDP-blocked home WiFi, the HK1 VLESS-TCP
  node connects and passes traffic where HY2 fails; traffic shows up in `traffic_used`;
  disabling the user cuts VLESS within a sync cycle.

## Deploy

- Pilot setup on HK1 via a setup script (extends the existing deploy-hy2 pattern):
  install Xray, generate Reality keys, write config + agent + systemd units, open
  cloud-firewall TCP 443, print/register the node's reality params into the panel.
- Panel changes ship via the usual `docker compose build` + hot swap, with a rollback
  image tag. DB migration adds nullable columns (safe, additive).

## Phase 2 (later, separate spec/plan)

Roll out VLESS to JP2/JP3/SG1/MY; integrate VLESS setup into the one-click node deploy
script; optionally add a "VLESS enabled" toggle + reality params editor to the admin
Nodes page.
