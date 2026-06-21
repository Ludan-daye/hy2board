# Trojan TCP fallback for Surge/Shadowrocket

Date: 2026-06-21
Status: approved (brainstorming)

## Problem

The VLESS+Reality fallback only works in clients that support VLESS (Clash/mihomo,
v2rayN, sing-box, URI imports). **Surge has no `vless` proxy type** (and Shadowrocket
`.conf` doesn't take our VLESS line either), so Surge/Shadowrocket users — including
the admin — have no TCP fallback when their network blocks HY2's UDP. Surge *does*
support **Trojan** over TCP. Add Trojan so those users get a native fallback.

## Goal

Run a **Trojan inbound** on each VLESS node (same Xray), fully integrated into
hy2board the same way VLESS is (per-user credential, traffic metered, disable/expiry
enforced). Emit it in the Surge and Shadowrocket subscription formats as the
`<node>-T` node, so every client gets one TCP fallback node it can actually use.

## Reuse (no new infrastructure)

- **Same Xray** on each node — add a third inbound (`api`, `vless-reality`, +`trojan`).
- **Same `vless-agent`** — its sync gains a trojan-inbound update; its stats endpoint
  already reports per-user traffic across *all* inbounds, so **Trojan bytes are
  metered with zero panel/poller changes**.
- **Same node-secret client-list endpoint** — extended to also return a per-user
  Trojan password.
- **Same enforcement** — a disabled/expired/over-limit user drops from the endpoint;
  the agent removes them from *both* inbounds.

## Node side (HK1/SG1/JP2, via updated setup script)

- Add Xray **Trojan inbound on TCP 8443** (free; HY2-obfs uses *UDP* 8443).
- TLS: generate a **self-signed cert** (CN = the node's SNI) into
  `/usr/local/etc/xray/trojan.{crt,key}`, readable by the `nobody` user Xray runs as
  (key chmod 644 — single-tenant throwaway cert). Clients connect with
  `skip-cert-verify` (same as the existing HY2 setup).
- The agent (`vless-agent.sh sync`) additionally sets the trojan inbound's clients to
  `[{password, email}]` from the panel list; restart Xray on change (existing behavior).
- ufw: open **TCP 8443** (clients). Setup script's ufw block extended.

## Panel side (hy2board code)

- **Node model**: add `trojan_enabled bool`, `trojan_port int`, `trojan_sni text`.
- **`util.TrojanPassword(username)`**: deterministic password derived from username
  (e.g. hex of `sha256(namespace+username)`), separate from `hy2_password`.
- **Client-list endpoint** (`GET /api/node/vless/clients`): each active user object
  gains `"password": TrojanPassword(username)` alongside `uuid`/`email`. (Additive;
  the VLESS agent ignores the new field.)
- **Subscription generation**:
  - Surge (`GenerateSurgeWithCustomRules`) and Shadowrocket
    (`GenerateShadowrocketWithCustomRules`): for each `trojan_enabled` healthy node,
    emit a Trojan proxy named `VlessName(n)` = `<node>-T`, and add that name to the
    Auto/Manual groups (after the HY2 names) for UDP→TCP failover.
  - Clash/URI: unchanged (keep VLESS, which those clients prefer).
  - The `-T` name is therefore the same across formats — "the TCP fallback node" —
    realized as VLESS (Clash/URI) or Trojan (Surge/Shadowrocket).
- **Helpers** (`internal/service/trojan_subscription.go`):
  - `NodeHasTrojan(n) = n.TrojanEnabled && n.TrojanPort>0`.
  - `TrojanSurgeLine(u,n)` → `"<node>-T = trojan, host, port, password=<pw>, sni=<sni>, skip-cert-verify=true"`.
  - `TrojanShadowrocketLine(u,n)` → Shadowrocket `.conf` key=value form
    (`<node>-T=trojan,host,port,password=<pw>,...`). (Shadowrocket DOES support trojan.)

## Data flow (all reused except the two emit points + password field)

```
sync:    agent --GET clients(uuid,password,email)--> panel ; agent sets vless+trojan inbound clients
meter:   panel --poll stats--> agent(Xray stats, all inbounds) --> traffic_used   (unchanged)
deliver: client --GET /api/sub/:token--> panel  (Surge/Shadowrocket now include <node>-T trojan)
enforce: inactive user -> dropped from clients -> agent removes from vless AND trojan inbounds
```

## Error handling

- Same fail-open agent (don't wipe clients on a transient panel error).
- Trojan cert generation is idempotent in the setup script (regenerate only if absent).
- If `trojan_enabled` but the node has no Trojan inbound yet, the Surge line still
  renders; it simply won't connect until the node is set up — operationally we set the
  node up first, then flip `trojan_enabled` in the panel (same order as VLESS).

## Testing (TDD, reuse in-memory sqlite pattern)

- `TrojanPassword` deterministic + stable per username.
- Client-list endpoint returns `password` for active users; still excludes inactive.
- Trojan Surge line: valid `trojan,` syntax, includes password/sni/skip-cert-verify,
  name `<node>-T`.
- Subscription: with no trojan node, Surge/Shadowrocket output unchanged (byte-identical
  guarantee). With a trojan node, the `-T` entry appears and joins Auto/Manual.
- Surge "Unknown proxy type" regression: assert the emitted line is `trojan` (a type
  Surge supports), never `vless`.

## Stealth note (accepted tradeoff)

Self-signed Trojan is more detectable than VLESS-Reality (active probing reveals a
self-signed cert; no real-site masquerade). It is the best Surge-compatible option and
matches the existing HY2 self-signed approach. VLESS-Reality remains the primary, more
stealthy path for Clash/URI clients. A real-domain cert per node could be a later
hardening step.

## Rollout

Update `deploy/vless-pilot-setup.sh` to also create the Trojan inbound + cert + ufw
rule (or a small `deploy/trojan-add.sh` re-run). Apply to HK1/SG1/JP2, set
`trojan_enabled` in the panel per node, verify panel→node TCP 8443. Panel code ships
via `docker compose build` + hot swap with a rollback tag.
