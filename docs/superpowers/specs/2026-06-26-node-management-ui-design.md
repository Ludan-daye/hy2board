# Node management UI — VLESS / Trojan / Reality

Date: 2026-06-26
Status: approved (brainstorming) — sub-project 1 of 3 (then Dashboard redesign, then global visual refresh)

## Problem

The VLESS+Reality and Trojan node fields (`vless_enabled`, `vless_port`,
`reality_pubkey/shortid/sni`, `vless_stats_api/secret`, `trojan_enabled`,
`trojan_port`, `trojan_sni`) have **no UI**. Every node was enabled/configured with
raw `UPDATE nodes SET ...` SQL. The admin needs to do this from the Nodes page.

## Goal

Manage all VLESS/Trojan/Reality node fields in the admin **Nodes** page — both by
**pasting the setup-script `REGISTER` output** (auto-fill) and by **manual form
fields**. No SQL required to add/enable a fallback-capable node.

## Non-goals (this sub-project)

- Dashboard redesign and the global visual/UX refresh are separate sub-projects.
- No change to the on-node scripts or the subscription/agent logic — those already
  produce and consume these fields. This is purely panel CRUD UI + handler plumbing.

## Backend (`internal/handler/node.go`)

- Extend `NodeRequest` with the new fields:
  `VlessEnabled *bool`, `VlessPort int`, `RealityPubkey/RealityShortID/RealitySNI string`,
  `VlessStatsAPI string`, `VlessStatsSecret string`, `TrojanEnabled *bool`,
  `TrojanPort int`, `TrojanSNI string`. (Bools are `*bool` so "not sent" ≠ "false".)
- `CreateNode`: map the new fields onto `model.Node` (bools default false when nil).
- `UpdateNode`: add the new fields to the `updates` map. Booleans go in unconditionally
  when the pointer is non-nil. **`vless_stats_secret` is write-only**: include it in
  the update only when the request value is non-empty (blank = leave unchanged), so the
  UI never has to read it back.
- `ListNodes` already returns the model JSON (secrets are `json:"-"`, so the stats
  secret is never sent to the browser).

## Frontend

### `web/src/utils/nodeRegister.ts` (new, pure + unit-testable)

`parseRegister(text): Partial<NodeForm>` — parses the two `REGISTER` formats the setup
scripts print and returns the matching form fields:

- VLESS block (multi-line `key = value`): `vless_enabled=1`, `vless_port`,
  `reality_pubkey`, `reality_shortid`, `reality_sni`, `vless_stats_api`,
  `vless_stats_secret`.
- Trojan line (inline `key=value`): `trojan_enabled=1`, `trojan_port`, `trojan_sni`.

Tolerant of spacing/casing; ignores unrelated lines; pasting only one of the two blocks
fills only that block's fields.

### `web/src/pages/Nodes.tsx`

- Extend the node form state (`emptyForm`) and the add/edit form with a collapsible
  **"TCP 回落 (VLESS / Trojan)"** section:
  - `☑ 启用 VLESS-Reality` toggle → reveals `vless_port`, `reality_pubkey`,
    `reality_shortid`, `reality_sni`, `vless_stats_api`, `vless_stats_secret`
    (secret shows placeholder "已设置 / 留空不变"; never pre-filled).
  - `☑ 启用 Trojan` toggle → reveals `trojan_port`, `trojan_sni`.
  - A **"📋 粘贴注册信息"** control: a small textarea + apply button that runs
    `parseRegister` and merges the result into the form (sets toggles + fields).
- `startEdit(n)` pre-fills the new fields from the node (except the write-only secret).
- `add`/`save` send the new fields in the POST/PUT body (the secret only if the user
  typed one).

### Layout

```
▼ TCP 回落 (VLESS / Trojan)            [📋 粘贴注册信息]
  ☑ 启用 VLESS-Reality   端口[443]
    reality 公钥 [______]  short-id [____]  SNI [www.apple.com]
    stats API [http://IP:25415/]  stats 密钥 [已设置/留空不变]
  ☑ 启用 Trojan   端口[8443]   SNI [www.apple.com]
```

## Data flow

`Nodes.tsx` ← `GET /admin/nodes` (already includes the fields) → form. Save →
`PUT/POST /admin/nodes[/:id]` with the extended body. Paste → `parseRegister` → merge
into form state (client-only, no request until Save).

## Error handling

- `parseRegister` on unrecognized text returns an empty object (no fields changed) and
  the UI shows a brief "未识别到注册信息" message.
- Invalid port (non-numeric) is ignored by the parser; the form's number inputs guard.
- Disabling a toggle sends `vless_enabled/trojan_enabled = false`; the backend persists
  it (map update handles the zero value), so the node stops appearing in subscriptions.

## Testing

- **Backend (TDD, in-memory sqlite like existing handler tests):** UpdateNode with the
  new fields persists them; `vless_enabled=false` is stored (not dropped); empty
  `vless_stats_secret` leaves the existing one unchanged; non-empty replaces it.
- **Frontend:** `nodeRegister.ts` is a pure function — add Vitest IF a runner is wired,
  else verify via `tsc -b && vite build` + a live check (paste a real REGISTER block,
  confirm fields fill). No test runner exists today (vite/eslint only); a unit-test
  harness is out of scope — verify by build + manual.

## Deploy

Backend + frontend ship together via `docker compose build` + hot swap with a rollback
tag. DB unchanged (columns already exist). No impact on existing nodes/subscriptions —
the form just exposes fields that were previously SQL-only.
