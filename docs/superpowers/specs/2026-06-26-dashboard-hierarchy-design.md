# Dashboard hierarchy redesign

Date: 2026-06-26
Status: approved (brainstorming) — sub-project 2 of 3 (after node-management UI; before global visual refresh)

## Problem

The admin Dashboard renders 12+ cards/charts flat on one page (5 stat cards, monthly
income, expiring-soon, near-limit, offline-nodes, live speed ranking, account-health
pie, top-traffic users, real-time traffic area chart, per-node live line charts, static
IPs, node traffic bar, traffic distribution pie). It's information-dense with no
hierarchy — fires (down nodes / at-risk users) sit alongside rarely-needed analysis
charts. The goal is to **restructure the hierarchy**, not change the data.

## Goal

Reorganize `web/src/pages/Dashboard.tsx` into four clear zones so the default view
answers "is everything OK?" first and demotes analysis charts behind a toggle. No data,
endpoints, or metrics are added or removed.

## Non-goals

- No new metrics/charts; no backend changes.
- No global visual restyle (that is sub-project 3). Only minimal layout/spacing here.

## The four zones (top → bottom)

1. **Health bar** (top, always visible) — replaces the 5 separate stat cards with one
   dense signal-first row:
   - Nodes `healthy/total` (amber/red when any unhealthy)
   - Online total
   - **Alerts** = `expiringSoon.length + nearLimit.length + deadNodes.length` as one
     number; amber/red when > 0; clicking navigates to `/alerts`
   - 本月收入 (monthly income)
   - Live network ▲up / ▼down (from the latest history point)

2. **需处理 / Needs attention** — the three actionable cards grouped: Expiring Soon,
   Near Traffic Limit, Offline Nodes. When all three are empty, render a single compact
   "✅ 一切正常" line instead of three empty cards.

3. **实时 / Live** — Real-time Traffic area chart (full width) + Live Speed Ranking
   (top 5). The "what's happening now" view.

4. **更多分析 / Analysis (collapsible, default collapsed)** — Account Health pie, Top
   Traffic Users, Per-Node Live Activity, Node Traffic Comparison, Traffic Distribution,
   Static IPs. Behind a `▸ 更多分析` toggle; **not rendered until expanded** (so the
   heavier charts don't run when collapsed).

## Layout sketch

```
[ 节点 9/10⚠  在线 25  告警 ③→/alerts  本月¥1,280  ▲2.1M/s ▼8.4M/s ]   ← health bar
[ 即将到期 2 | 接近限额 1 | 离线节点 1 ]   (or "✅ 一切正常")            ← needs attention
[ 实时流量面积图 (全宽) ][ 实时速度排行 top5 ]                          ← live
▸ 更多分析   (collapsed; expands to the 6 analysis cards)               ← analysis
```

## Architecture

- Single file `web/src/pages/Dashboard.tsx`. The data fetching and all `useMemo`/derived
  values (`totalOnline`, `expiringSoon`, `nearLimit`, `deadNodes`, `topTraffic`,
  `statusCounts`, `nodeBarData`, `pieData`, `history`) stay as-is — only the `return()`
  JSX is reorganized into the four zones.
- Add one state: `const [showAnalysis, setShowAnalysis] = useState(false)`; the analysis
  zone renders only when `showAnalysis`.
- Add a small `alertsTotal = expiringSoon.length + nearLimit.length + deadNodes.length`
  derived value for the health bar.
- The existing `StatCard`/`Card` components are reused. If the health bar needs a denser
  cell than `StatCard`, add a tiny local `HealthCell` presentational component in the
  same file (one clear responsibility: label + value + tone) rather than restyling
  StatCard globally.

## Error handling / edge cases

- Loading state unchanged (`if (!stats) return Loading`).
- Empty alerts → the compact "一切正常" line (no broken empty cards).
- Analysis collapsed by default means its charts (recharts) don't mount → less work on
  load; expanding mounts them.

## Testing

- No test runner for the frontend (vite/eslint only). Verify via `tsc -b && vite build`
  (type-check + build) and a live check: health bar shows correct counts; alerts number
  matches `/alerts`; clicking it navigates; needs-attention collapses to "一切正常" when
  empty; analysis toggle shows/hides the six charts.

## Deploy

Frontend-only; ships via `docker compose build` + hot swap with a rollback tag. No DB or
API change.
