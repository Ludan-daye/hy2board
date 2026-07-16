# Static-IP routing: AI→US / default→Taiwan, chosen users only

Date: 2026-07-16
Status: approved (brainstorming)

## Problem

The global US chain proxy (`config.ProxyChain` = 199.234.95.128) leaks into **every**
subscription: `EffectiveProxyChain(user)` falls back to the global chain when a user has no
per-user `ProxyHost`, and the `-AI` variants appear for any user with `rule_ai`/`chain_proxy`
on. So a plain user like `yyz` gets the operator's US static IP in their config. There is no
notion of "this static IP is only for chosen people," and no separate default exit — non-China,
non-AI traffic just goes `FINAL→Auto` (direct through the selected node).

The operator wants: static IPs (chains) available **only to a chosen set of 5 users**, and for
those users, **AI traffic → US IP, all other (non-China) traffic → Taiwan IP**, China → direct,
with a per-user override. Everyone else stays pure-direct (no chain in their config at all).

## Goal

1. Add a Taiwan chain (`202.160.86.52:8022` socks5 `scdrawqaee:ipimorpmwrmti`) as the **default
   exit** for chosen users.
2. Gate all chain/static-IP routing behind a **"static-IP user" flag** (`user.ChainProxy`).
   Chosen 5 = `ludandaye, sjb, qmf, yzp, xrx`. Everyone else (incl. `a100`) = off → no chain.
3. Chosen users' rule-based subscriptions (Clash/Surge/Shadowrocket): **AI→US, default→Taiwan,
   China→direct**, per-user override.
4. **No change for non-chosen users** except they lose the leaked `-AI`/US entries — their
   subscription becomes byte-identical minus those lines.

## Non-goals

- No new admin UI in this pass. The 5 users' flags + the Taiwan config are set via DB/API/config.
- URI/`v2ray`-base64/plain-`shadowrocket` formats carry **no routing rules** (client limitation),
  so AI→US/default→Taiwan does not apply there — those clients get direct nodes only. Chosen
  users must use Clash/Surge for the split to take effect. (Documented, not "fixed".)
- HY2/VLESS/Trojan node configs unchanged. No node-side changes.
- The broken `direct.miyaip.online` is dropped entirely.

## Decisions

- **Two global chains** in `config.yaml`:
  - `proxy_chain` = **US** `199.234.95.128` — the **AI** exit (unchanged).
  - `proxy_chain_default` = **Taiwan** `202.160.86.52` — the **default** exit (NEW).
- **"Static-IP user" = `user.ChainProxy == true`.** Chosen 5 → true; all others → false.
- **Per-user override ("除非特定设置")** = `user.ProxyHost` set → that replaces the Taiwan default
  for that user's default exit.
- **Resolver split** (`internal/service/proxy_resolver.go`):
  - `EffectiveAIChain(u)` → `config.ProxyChain` (US) with `ok=true` **iff `u.ChainProxy`**; else
    `ok=false`.
  - `EffectiveDefaultChain(u)` → `u.ProxyHost` override if set; else `config.ProxyChainDefault`
    (Taiwan) with `ok=true` **iff `u.ChainProxy`**; else `ok=false`.
  - The old `EffectiveProxyChain` is replaced by these two (call sites updated).

## Architecture / per-format changes

For each rule-capable generator — `GenerateSurgeWithCustomRules`, `GenerateClashWithCustomRules`,
`GenerateShadowrocketWithCustomRules` (`internal/service/subscription.go`):

1. Resolve `aiPC, hasAIChain := EffectiveAIChain(user)` and `defPC, hasDefChain := EffectiveDefaultChain(user)`.
2. **`-AI` proxies** (US): emit per node **iff `hasAIChain && rs.HasAI`** (was: `rs.HasAI && hasChainConfig`).
   Uses `aiPC.*`. → `AI` group (url-test over `-AI` names).
3. **`-TW` proxies** (Taiwan default): NEW. Emit per node **iff `hasDefChain`**, as
   `<node>-TW = socks5 defPC …, underlying-proxy=<node>` (Surge/Shadowrocket) /
   `dialer-proxy: <node>` (Clash). → NEW `Default` group (url-test over `-TW` names).
4. **FINAL/MATCH**: if `hasDefChain` → `FINAL,Default` (Taiwan); else `FINAL,Auto` (unchanged,
   direct). Keeps non-chosen users on `Auto`.
5. **AI-policy rewrite**: `aiPolicy = GroupAI` iff `hasAIChain && rs.HasAI`, else `GroupAuto`
   (existing pattern, now keyed on the split resolver).
6. China rules → `DIRECT` (unchanged). HongKong/Streaming/Global groups unchanged.
7. Add `GroupDefault = "Default"` constant (`rules.go`).

`generateURI` / base64 / plain-shadowrocket: **untouched** (no rules/groups — non-goal).

`config.go`: add `ProxyChainDefault ProxyChainConfig \`yaml:"proxy_chain_default"\`` + a
`HasProxyChainDefault()` guard.

## Data changes (one-time, via API/DB/config)

- `config.yaml`: add `proxy_chain_default` = Taiwan (`202.160.86.52:8022 socks5
  scdrawqaee:ipimorpmwrmti`). Keep `proxy_chain` = US.
- Users: set `chain_proxy=true` on `ludandaye, sjb, qmf, yzp, xrx`; `chain_proxy=false` on all
  others (notably `a100`). Clear the 5 users' `proxy_host` (the old per-user US override) so their
  default resolves to the global Taiwan chain (unless the operator later sets a custom override).
- Stale plan `proxy_host` values (7.7.7.7, 1.2.3.4 debug, etc.) are out of scope — left as-is.

## Error handling / edge cases

- **Non-chosen users must be byte-identical to before, minus the `-AI` lines + AI group.** This
  is the primary safety check (a normal user's browsing must not change).
- If a chosen user has `ChainProxy=true` but neither chain resolves (misconfig) → `hasDefChain`
  false → `FINAL,Auto` (falls back to direct; never emits a dangling group or breaks the config).
- `rule_ai` without `chain_proxy` (a non-chosen user with rule_ai) → no AI chain; AI rules resolve
  to `Auto` (direct) via the existing policy-rewrite. No leak.
- Per-user `ProxyHost` override applies to the **default** exit only; AI always uses the global
  US chain for chosen users.
- Taiwan/US chain proxy host is resolved by the **client**, not the panel (embedded in config),
  so panel-box DNS quirks don't matter.

## Testing

Go unit tests (like existing `*_test.go`, DB-free):
- `EffectiveAIChain` / `EffectiveDefaultChain`: chosen user → US/Taiwan with ok; per-user override
  wins for default; non-chosen → ok=false for both.
- Rendered-subscription tests (build a `model.User` + fake nodes, call each generator): chosen
  user's Clash/Surge contains `-AI` (US) + `-TW` (Taiwan), an `AI` group and a `Default` group,
  `FINAL→Default`, AI-rule→AI; non-chosen user contains **no** `-AI`/`-TW`, `FINAL→Auto`.
- **Live byte-diff before rollout**: fetch one **non-chosen** user's Clash+Surge before/after —
  diff must be only removed `-AI` lines/AI-group. Fetch one **chosen** user — confirm Taiwan
  `Default` group + `FINAL→Default` + `-AI` US present. Only then apply the data changes broadly.

## Deploy

Frontend-none; backend build + hot swap at `/opt/hy2board` (rollback image
`hy2board-hy2board:rollback-prerouting`). Config edit + user flag/proxy_host updates are reversible
(re-set `chain_proxy`/`proxy_host`, revert config). **Note the code-ship itself changes behavior**
(the new `EffectiveAIChain` gate drops the `-AI` leak from every `chain_proxy=false` user on
deploy, and no longer falls back to the global chain). Rollout order:
1. **Prep data first** so the code-ship lands cleanly: set `chain_proxy=false` on `a100` (and any
   non-chosen user currently true), confirm the 5 chosen have `chain_proxy=true`. Leave the 5's
   `proxy_host`=US for now (harmless: without the resolver split live yet it's still their chain).
2. **Ship code.** Intermediate state: leak fixed (only the 5 chosen still emit `-AI`, now US via
   `EffectiveAIChain`); Taiwan not configured yet → chosen users' default still `FINAL→Auto`
   (direct). Non-chosen users lose `-AI` only.
3. **Byte-diff verify** one non-chosen user (only `-AI`/AI-group removed) and confirm the 5 chosen
   still have US `-AI`.
4. **Enable Taiwan**: add `proxy_chain_default` (Taiwan) to `config.yaml`, clear the 5 chosen
   users' `proxy_host` (so default resolves to global Taiwan), restart. Re-verify a chosen user now
   shows the `Default` (Taiwan) group + `FINAL→Default`.
5. Announce nothing (transparent; users refresh subs to pick up the new routing).
