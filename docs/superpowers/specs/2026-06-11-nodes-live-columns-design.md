# Nodes page — live connection columns

Date: 2026-06-11
Status: approved (brainstorming)

## Goal

Show each node's **real-time connection status** directly on the admin **Nodes**
page: current online client count and live up/down throughput. Node latency and
probe status are already shown in the existing "Monitor" column.

## Non-goals

- No backend / DB / API changes. Both endpoints already expose what we need.
- No Telegram work. Node-down (`🚨 节点异常`) and node-recovery (`✅ 节点恢复`)
  alerts already exist in `maybeAlertNodeProbe` and the admin telegram_id is bound.
- No Dashboard refactor. Only a small shared format helper is extracted.

## Data sources (both already exist)

- `GET /admin/nodes` — probe status, latency, last_checked (Nodes page already polls this).
- `GET /admin/stats` — per node: `online` (live client count) and `traffic`
  (`{username: {tx, rx}}` cumulative byte counters).

## Design (frontend only: `web/src/pages/Nodes.tsx`)

1. On each refresh, fetch `/admin/nodes` and `/admin/stats` together; index the
   stats node list by node id.
2. **Live speed is computed client-side.** Keep the previous snapshot
   (`{nodeId: {tx, rx}}` + timestamp) in a `useRef`. Per node, sum all users'
   tx/rx to a node total; `delta = max(0, cur - prev)` (the `max(0, …)` guards
   against node counter resets); `speed = delta / elapsedSeconds`.
3. Poll interval: **10s → 3s** so throughput reads as live. (Probe data only
   changes every 30s server-side; latency simply repeats — harmless.)
4. Table gets two new columns: **在线** (online count) and **实时 ↑/↓** (up/down
   speed). The existing Monitor/Name/Host/SNI/Obfs/Traffic API/Actions columns and
   all management actions are unchanged.
5. Add an **Online** summary card next to the existing status cards.
6. Extract a tiny `web/src/lib/format.ts` (`fmtBytes`, `fmtSpeed`) used by Nodes.
   Dashboard keeps its local copies (no unrelated churn).

## Data flow

`Nodes.tsx` —(every 3s)→ `GET /admin/nodes` + `GET /admin/stats`
→ merge by id + diff vs prev snapshot → render table.

## Edge cases

- First poll (no prev snapshot) → speed shows `-`.
- Node present in `/admin/nodes` but absent from `/admin/stats` → online `0`, speed `-`.
- Counter reset (node restart) → `max(0, cur-prev)` yields one undercounted tick, never a negative/huge spike.

## Deploy & verify

- Frontend is built into the image (`Dockerfile` node stage → `web/dist`), so
  shipping requires `docker compose build` + `up -d` (the same low-impact hot swap
  used for the traffic fix). No data risk.
- No frontend test runner exists (vite/eslint only); a unit-test harness is out of
  scope. Verify via `tsc -b && vite build` (type-check + build pass) and a live
  check in the admin UI that the columns populate and speed updates each tick.
