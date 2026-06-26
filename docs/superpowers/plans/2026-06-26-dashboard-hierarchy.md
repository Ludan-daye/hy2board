# Dashboard Hierarchy Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize `web/src/pages/Dashboard.tsx`'s `return()` into 4 zones (health bar → needs-attention → live → collapsible analysis) without changing any data or metrics.

**Architecture:** Pure JSX reorganization of one file. Keep all data fetching and derived values; add `showAnalysis` state + `alertsTotal` derived; replace the stat-card row with a dense health bar; group the 3 actionable cards (with an empty-state); put Real-time Traffic + Live Speed Ranking in a Live zone; move the remaining 6 charts behind a `▸ 更多分析` toggle that mounts them only when expanded.

**Tech Stack:** React 19 + Vite + recharts + react-router-dom. No test runner (verify via `tsc -b && vite build` + live check). Build/deploy via Docker image build (`tsc -b && vite build`) + hot swap.

**Sync note:** edit in `/root/ludandaye/ladder/hy2board` (commit/push) AND scp `web/src/pages/Dashboard.tsx` to `/opt/hy2board` before building.

---

## File structure

| File | Change |
|---|---|
| `web/src/pages/Dashboard.tsx` | reorganize `return()`; add `showAnalysis` state, `alertsTotal`, `HealthCell` local component; relocate existing `<Card>` blocks into zones |

The current `return()` has these wrappers (do NOT lose any inner content when moving):
- W1 `grid grid-cols-2 lg:grid-cols-6` → 5 `<StatCard>` + the 本月收入 card
- W2 `grid grid-cols-1 lg:grid-cols-3` → `Expiring Soon`, `Near Traffic Limit`, `Offline Nodes`
- W3 `grid grid-cols-1 lg:grid-cols-12` → `Live Speed Ranking`, `Account Health`, `Top Traffic Users`
- W4 `mb-4` → `Real-time Traffic`
- W5 `mb-4` → `Per-Node Live Activity`
- W6 (below) → `Static IPs`, `Node Traffic Comparison`, `Traffic Distribution`

---

## Task 1: State + derived values + HealthCell

**Files:** `web/src/pages/Dashboard.tsx`

- [ ] **Step 1:** Add `useState` import is already present. Add the state near the other `useState`s in the component:

```tsx
  const [showAnalysis, setShowAnalysis] = useState(false)
```

- [ ] **Step 2:** After `const deadNodes = ...` (where the derived values are computed, ~line 156), add:

```tsx
  const alertsTotal = expiringSoon.length + nearLimit.length + deadNodes.length
  const lastPoint = history.length ? history[history.length - 1] : null
```

- [ ] **Step 3:** Add a small presentational `HealthCell` near the existing `StatCard`/`Card` defs (top of the file, after `Card`):

```tsx
function HealthCell({ label, value, tone = "text-white", to }: {
  label: string; value: React.ReactNode; tone?: string; to?: string
}) {
  const inner = (
    <div className="px-3 py-2.5 bg-zinc-900 border border-zinc-800 rounded-lg">
      <div className="text-[11px] text-zinc-500 mb-0.5">{label}</div>
      <div className={`text-lg font-semibold ${tone}`}>{value}</div>
    </div>
  )
  return to ? <Link to={to} className="block hover:border-zinc-600">{inner}</Link> : inner
}
```

- [ ] **Step 4:** Ensure `Link` is imported. If `import { Link } from "react-router-dom"` is not already at the top, add it. Build check happens in Task 5.
- [ ] **Step 5: Commit** — `git commit -am "dashboard: add showAnalysis state, alertsTotal, HealthCell"`

---

## Task 2: Health bar (replace the stat-card row)

**Files:** `web/src/pages/Dashboard.tsx`

- [ ] **Step 1:** Replace wrapper **W1** (the `<div className="grid grid-cols-2 lg:grid-cols-6 gap-3 mb-4">` block containing the 5 `<StatCard>`s and the 本月收入 card) with the health bar. Keep the `cur`/income computation that the 本月收入 card used — reference the same value via a local `incomeText`. Insert this in place of W1:

```tsx
      {/* Zone 1: health bar */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-4">
        <HealthCell label="节点 healthy/total"
          value={`${stats.healthy_nodes}/${stats.total_nodes}`}
          tone={stats.healthy_nodes < stats.total_nodes ? "text-amber-400" : "text-white"} />
        <HealthCell label="在线" value={totalOnline} tone="text-blue-300" />
        <HealthCell label="告警" to="/alerts"
          value={alertsTotal}
          tone={alertsTotal > 0 ? "text-red-400" : "text-zinc-500"} />
        <HealthCell label="本月收入" value={incomeText} />
        <HealthCell label="实时 ↑" value={lastPoint ? fmt(lastPoint.speed_tx) + "/s" : "-"} tone="text-sky-400" />
        <HealthCell label="实时 ↓" value={lastPoint ? fmt(lastPoint.speed_rx) + "/s" : "-"} tone="text-emerald-400" />
      </div>
```

- [ ] **Step 2:** Compute `incomeText` next to the other derived values (Task 1 area). The original 本月收入 card used `yuan(cur.total_cents)` where `cur` is the current-month income row. Add:

```tsx
  const incomeText = (() => {
    const cur = income.find(i => i.month === new Date().toISOString().slice(0, 7))
    return cur ? yuan(cur.total_cents) : yuan(0)
  })()
```

> If the existing code already derives `cur`/income differently, reuse that exact value instead of re-deriving — the goal is the same number the old 本月收入 card showed. Confirm `yuan(...)` and the `income` array names by reading the current file; adjust if the helper/array names differ.

- [ ] **Step 3:** Confirm `HistoryPoint` has `speed_tx`/`speed_rx` (it does — used by the Real-time Traffic chart). If the field names differ, use the actual ones.
- [ ] **Step 4: Commit** — `git commit -am "dashboard: zone 1 health bar replaces stat cards"`

---

## Task 3: Needs-attention zone with empty-state

**Files:** `web/src/pages/Dashboard.tsx`

- [ ] **Step 1:** Wrap wrapper **W2** (the 3-column grid with Expiring Soon / Near Traffic Limit / Offline Nodes) in a conditional: show the grid when there is anything to handle, else a single compact line. Replace the opening of W2 so it reads:

```tsx
      {/* Zone 2: needs attention */}
      {alertsTotal === 0 ? (
        <div className="mb-4 px-4 py-3 bg-zinc-900 border border-zinc-800 rounded-lg text-sm text-emerald-400">
          ✅ 一切正常 — 没有到期/超限用户或离线节点
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 mb-4">
          {/* ...existing Expiring Soon / Near Traffic Limit / Offline Nodes Cards unchanged... */}
        </div>
      )}
```

Keep the three existing `<Card>` blocks exactly as they are inside the `else` grid.

- [ ] **Step 2: Commit** — `git commit -am "dashboard: zone 2 needs-attention with all-clear empty state"`

---

## Task 4: Live zone

**Files:** `web/src/pages/Dashboard.tsx`

- [ ] **Step 1:** Build a Live zone containing the **Real-time Traffic** card (currently W4) and the **Live Speed Ranking** card (currently the first card in W3). Move the `Live Speed Ranking` `<Card>...</Card>` block out of W3 and place it beside Real-time Traffic:

```tsx
      {/* Zone 3: live */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 mb-4">
        <div className="lg:col-span-2">
          {/* ...existing Real-time Traffic <Card> block (from W4) unchanged... */}
        </div>
        <div>
          {/* ...existing Live Speed Ranking <Card> block (moved from W3) unchanged... */}
        </div>
      </div>
```

- [ ] **Step 2:** Remove the now-empty W4 wrapper and the Live Speed Ranking slot from W3 (W3 now only has Account Health + Top Traffic Users, which move to analysis in Task 5).
- [ ] **Step 3: Commit** — `git commit -am "dashboard: zone 3 live (realtime traffic + speed ranking)"`

---

## Task 5: Collapsible analysis zone

**Files:** `web/src/pages/Dashboard.tsx`

- [ ] **Step 1:** Wrap the remaining charts — **Account Health** + **Top Traffic Users** (rest of W3), **Per-Node Live Activity** (W5), **Static IPs**, **Node Traffic Comparison**, **Traffic Distribution** (W6) — in a toggle that only renders when expanded. Place this after the Live zone, replacing those wrappers:

```tsx
      {/* Zone 4: analysis (collapsed by default) */}
      <button onClick={() => setShowAnalysis(s => !s)}
        className="mb-3 text-sm text-zinc-400 hover:text-white flex items-center gap-1">
        <span>{showAnalysis ? "▾" : "▸"}</span> 更多分析
      </button>
      {showAnalysis && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            {/* ...Account Health <Card> + Top Traffic Users <Card> (moved from W3)... */}
          </div>
          {/* ...Per-Node Live Activity <Card> (W5)... */}
          {/* ...Static IPs block... */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            {/* ...Node Traffic Comparison <Card> + Traffic Distribution <Card> (W6)... */}
          </div>
        </div>
      )}
```

Move the existing `<Card>` blocks verbatim into these slots. Do not duplicate any block — each existing card appears exactly once across all zones.

- [ ] **Step 2: Commit** — `git commit -am "dashboard: zone 4 collapsible analysis"`

---

## Task 6: Build, deploy, verify

- [ ] **Step 1:** scp `web/src/pages/Dashboard.tsx` to `/opt/hy2board/web/src/pages/Dashboard.tsx`.
- [ ] **Step 2:** Tag rollback + build (runs `tsc -b && vite build`; **a type error here means a moved block lost a brace or a derived name is wrong — fix and rebuild**) + hot swap:

```bash
cd /opt/hy2board
docker tag "$(docker inspect hy2board-hy2board-1 --format '{{.Image}}')" hy2board-hy2board:rollback-dashui
docker compose build && docker compose up -d && sleep 8
```

- [ ] **Step 3: Verify** — `docker compose build` printed `✓ built` with no `error TS`; auth `ok:true`; the Dashboard chunk built (`docker exec hy2board-hy2board-1 sh -c "ls /app/web/dist/assets/Dashboard-*.js"`).
- [ ] **Step 4: Live check (you):** open the Dashboard — health bar shows nodes/online/alerts/income/live; the 告警 cell number equals the count on `/alerts` and clicking it navigates there; when no alerts, "✅ 一切正常" shows instead of the three cards; Real-time Traffic + Speed Ranking sit together; `▸ 更多分析` is collapsed by default and expands to the six analysis charts.
- [ ] **Step 5: Commit + push** — `git push origin HEAD:main`.

---

## Self-review

- **Spec coverage:** 4 zones — health bar (T2), needs-attention + empty state (T3), live (T4), collapsible analysis that mounts on expand (T5); `showAnalysis`/`alertsTotal`/`HealthCell` (T1); deploy/verify (T6). All spec sections mapped. No data/metric added or removed — only relocation + one toggle.
- **Placeholders:** the `{/* ...existing <Card> ... */}` markers denote relocating already-existing JSX (not new code to invent); all NEW code (HealthCell, health bar, empty-state, toggle, derived values) is written in full. Implementation reads the current file and moves the marked blocks verbatim.
- **Type consistency:** `alertsTotal`/`incomeText`/`lastPoint`/`showAnalysis` defined in T1–T2 and used in T2/T5; `HealthCell` props (`label/value/tone/to`) consistent; relies on existing `fmt`, `yuan`, `income`, `history`, `totalOnline`, `stats`, `expiringSoon/nearLimit/deadNodes` — verify these exact names in the file during T1 and adjust the two derived snippets if a helper name differs.
