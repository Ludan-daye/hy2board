# Claude "warm paper" Reskin — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reskin the hy2board admin panel from its cold dark zinc theme to Claude's warm cream "paper" look (cream `#FAF9F5`, near-black ink `#141413`, single clay `#C96442` accent, Newsreader serif headings + Inter UI), with **zero behavior/data change**.

**Architecture:** Centralize the palette in `web/src/index.css` via a Tailwind v4 `@theme` block that (a) defines semantic tokens and (b) **remaps the default `zinc` scale** to warm light values tuned to each shade's dominant use — this flips ~700 zinc class usages app-wide at once. A small set of **global `sed` sweeps** then handles the non-zinc neutrals (`text-white`, `bg-black`, primary buttons `bg-white text-black`, white-alpha inline borders, brand hexes). The shell (`Layout.tsx`) and the chart-heavy files (`Dashboard.tsx`, `Finance.tsx`) are hand-edited because they carry bespoke inline hex/recharts colors the sweeps deliberately skip. Rollout is phased: Phase 1 makes the whole app warm + fully polishes the Dashboard for a user gate; Phase 2 polishes the remaining pages; Phase 3 reskins Login + global a11y/contrast polish.

**Tech Stack:** React 19 + Vite + Tailwind CSS **v4** (`@tailwindcss/vite`, no config file — theme lives in CSS) + recharts + react-router-dom + lucide-react. Build/deploy via Docker (`docker compose build` runs `tsc -b && vite build` in the image) + hot swap. No frontend test runner.

**Sync note:** edit in `/root/ludandaye/ladder/hy2board` (commit/push), then **scp every changed file to `/opt/hy2board`** before `docker compose build`. The two trees are separate; `/opt/hy2board` is what actually builds/serves.

---

## File structure

| File | Change |
|---|---|
| `web/index.html` | add Google Fonts `<link>` (Inter, Newsreader, JetBrains Mono) |
| `web/src/index.css` | **the token layer** — `@theme` semantic tokens + zinc remap + `body` defaults (currently just `@import "tailwindcss"`) |
| `web/src/components/Layout.tsx` | hand rewrite shell: cream bg, paper sidebar, clay active-nav pill, serif wordmark, paper top-bar |
| `web/src/pages/Dashboard.tsx` | hand: `Card`/`HealthCell`/`tooltipStyle` inline styles → tokens; recolor `COLORS`/`STATUS_COLORS`/all chart hex to the chart palette; health-bar numerals → serif |
| `web/src/pages/Finance.tsx` | hand: recolor recharts strokes (`#60a5fa/#f87171/#4ade80`, axis greys) to chart palette |
| all other `web/src/**/*.tsx` | global `sed` sweep (Task 2) + per-page residue fixes (Phase 2) |
| `web/src/pages/Login.tsx` | hand: its self-contained `:root` (electric `#2b47ff`/`#ffae00` + grain + animation) → paper palette |
| `docs/superpowers/plans/...` | this file |

---

## Migration Reference (DRY — referenced by every task)

### A. Semantic tokens (defined in `index.css`, Task 1)

| token | value | role | Tailwind class |
|---|---|---|---|
| `--color-paper` | `#FAF9F5` | app bg | `bg-paper` |
| `--color-paper-rail` | `#F0ECE0` | sidebar / zebra | `bg-paper-rail` |
| `--color-surface` | `#FFFFFF` | cards/modals/inputs | `bg-surface` |
| `--color-surface-2` | `#F4F1EA` | hover / secondary-button surface | `bg-surface-2` |
| `--color-border` | `#E5E0D6` | hairline border | `border-border` |
| `--color-border-strong` | `#D9D3C6` | input/emphasis border | `border-border-strong` |
| `--color-ink` | `#141413` | primary text | `text-ink` |
| `--color-ink-muted` | `#6B6862` | secondary text | `text-ink-muted` |
| `--color-ink-faint` | `#9A968C` | labels/captions | `text-ink-faint` |
| `--color-clay` | `#C96442` | **accent** | `bg-clay` `text-clay` `border-clay` |
| `--color-clay-hover` | `#B5573A` | hover/pressed | `bg-clay-hover` |
| `--color-success` | `#3F8A4D` | online/active | `text-success` |
| `--color-warning` | `#C2843E` | expiring/near-limit | `text-warning` |
| `--color-danger` | `#BE3A31` | offline/over-limit | `text-danger` |
| `--color-chart-1..4` | `#C96442 #4C7A87 #C2843E #6E8B6A` | chart series | (used as hex in recharts) |

Clay tint for badges/active pills: use `bg-clay/10` (Tailwind opacity modifier on `--color-clay`).

### B. Zinc remap (in the SAME `@theme`, Task 1) — tuned to each shade's dominant use

In Tailwind v4, overriding `--color-zinc-N` re-points every `bg-/text-/border-zinc-N` utility. The mapping is intentionally **non-monotonic** (text shades pulled dark, surface/border shades pushed light) because the code uses shades semantically, not by ordering:

| shade | dominant use (count) | new value | rationale |
|---|---|---|---|
| `zinc-950` | bg ×1 | `#FAF9F5` | page surface |
| `zinc-900` | bg ×53 | `#FFFFFF` | card surface |
| `zinc-800` | border ×81 / bg ×35 | `#E5E0D6` | light warm border **and** an acceptable light secondary-button surface |
| `zinc-700` | border ×43 / bg ×25 | `#D9D3C6` | stronger border / button-hover surface |
| `zinc-600` | text ×45 | `#8B8780` | faint text |
| `zinc-500` | text ×162 | `#73706A` | muted text (most common) |
| `zinc-400` | text ×89 | `#5C5950` | secondary text (darker → more prominent, mirrors dark-theme brightness) |
| `zinc-300` | text ×32 | `#3A3833` | prominent secondary text |
| `zinc-200` | bg ×23 | `#E5E0D6` | light chip surface |
| `zinc-100` | text ×6 | `#2A2825` | near-primary text |

**Known minor casualties** (fixed during per-page review, not blockers): `text-zinc-200` (×7) and `border-zinc-500/600` (×13) inherit a value tuned for the other use; spot-fix any that look wrong.

### C. Global `sed` sweeps (Task 2) — the non-zinc neutrals. Run from `web/src`, in THIS ORDER.

```bash
cd /root/ludandaye/ladder/hy2board/web/src
# 1. body/foreground white text -> ink  (do FIRST, before re-introducing white on clay)
grep -rl 'text-white' . | xargs sed -i 's/\btext-white\b/text-ink/g'
# 2. primary "white pill" buttons -> clay (re-introduces white ONLY on clay)
grep -rl 'bg-white text-black' . | xargs sed -i 's/bg-white text-black/bg-clay text-white/g'
grep -rl 'hover:bg-zinc-200' . | xargs sed -i 's/hover:bg-zinc-200/hover:bg-clay-hover/g'
# 3. leftover black text/surfaces
grep -rl 'text-black' . | xargs sed -i 's/\btext-black\b/text-ink/g'
grep -rl 'bg-black/60' . | xargs sed -i 's#bg-black/60#bg-ink/40#g'   # modal backdrops -> warm, lighter
grep -rl 'bg-black/40' . | xargs sed -i 's#bg-black/40#bg-ink/30#g'
grep -rl 'bg-black' . | xargs sed -i 's#bg-black\([^/]\)#bg-surface\1#g; s#bg-black$#bg-surface#g'
grep -rl 'bg-white' . | xargs sed -i 's/\bbg-white\b/bg-surface/g'    # remaining non-button bg-white
# 4. white-alpha inline borders/tints -> warm ink-alpha (inverts them onto cream)
grep -rl 'rgba(255,255,255,0.06)' . | xargs sed -i 's/rgba(255,255,255,0.06)/rgba(20,20,19,0.08)/g'
grep -rl 'rgba(255,255,255,0.04)' . | xargs sed -i 's/rgba(255,255,255,0.04)/rgba(20,20,19,0.06)/g'
grep -rl 'rgba(255,255,255,0.03)' . | xargs sed -i 's/rgba(255,255,255,0.03)/rgba(20,20,19,0.04)/g'
grep -rl 'rgba(255,255,255,0.02)' . | xargs sed -i 's/rgba(255,255,255,0.02)/rgba(20,20,19,0.03)/g'
# 5. inline brand/zinc hex on the 11 NON-chart files (skip Dashboard+Finance+Login — handled by hand)
for f in $(grep -rl '#1a2b6b\|#243b8f\|#111318\|#18181b\|#27272a\|#71717a\|#8900ff\|#eee6ff' . \
           | grep -v -e 'Dashboard.tsx' -e 'Finance.tsx' -e 'Login.tsx'); do
  sed -i 's/#1a2b6b/#C96442/g; s/#243b8f/#B5573A/g; s/#111318/#FFFFFF/g; s/#18181b/#FAF9F5/g; \
          s/#27272a/#E5E0D6/g; s/#71717a/#6B6862/g; s/#8900ff/#C96442/g; s/#eee6ff/#F4E9E4/g' "$f"
done
```

### D. Status / accent palette classes (handled in per-page review, Phase 2)

Tailwind `green/red/amber` status utilities (`text-green-400`, `bg-red-400/10`, etc.) **stay** — they read fine on cream and keep semantic meaning. Decorative non-status accents get remapped per page:
- `text-blue-300 / text-sky-400 / bg-blue-*` → `text-clay` / chart-2 teal as appropriate
- `bg-purple-500/10 text-purple-400` (the "AI"/chain badge) → `bg-clay/10 text-clay`

---

## Phase 1 — Foundation + whole-app warm baseline + Dashboard polish (USER GATE)

### Task 1: Token layer (`index.css` + `index.html`)

**Files:** Modify `web/index.html`; Modify `web/src/index.css`

- [ ] **Step 1:** Add fonts to `web/index.html` `<head>` (after the `<meta viewport>` line):

```html
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Newsreader:opsz,wght@6..72,400;6..72,500;6..72,600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet" />
```

- [ ] **Step 2:** Replace the entire contents of `web/src/index.css` with the token layer:

```css
@import "tailwindcss";

@theme {
  /* fonts */
  --font-sans: "Inter", ui-sans-serif, system-ui, -apple-system, sans-serif;
  --font-serif: "Newsreader", ui-serif, Georgia, "Times New Roman", serif;
  --font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, monospace;

  /* semantic palette (Claude warm paper) */
  --color-paper: #FAF9F5;
  --color-paper-rail: #F0ECE0;
  --color-surface: #FFFFFF;
  --color-surface-2: #F4F1EA;
  --color-border: #E5E0D6;
  --color-border-strong: #D9D3C6;
  --color-ink: #141413;
  --color-ink-muted: #6B6862;
  --color-ink-faint: #9A968C;
  --color-clay: #C96442;
  --color-clay-hover: #B5573A;
  --color-success: #3F8A4D;
  --color-warning: #C2843E;
  --color-danger: #BE3A31;
  --color-chart-1: #C96442;
  --color-chart-2: #4C7A87;
  --color-chart-3: #C2843E;
  --color-chart-4: #6E8B6A;

  /* zinc remapped to warm-paper scale (dominant-use tuned; see plan §B) */
  --color-zinc-950: #FAF9F5;
  --color-zinc-900: #FFFFFF;
  --color-zinc-800: #E5E0D6;
  --color-zinc-700: #D9D3C6;
  --color-zinc-600: #8B8780;
  --color-zinc-500: #73706A;
  --color-zinc-400: #5C5950;
  --color-zinc-300: #3A3833;
  --color-zinc-200: #E5E0D6;
  --color-zinc-100: #2A2825;
}

html, body, #root {
  background: var(--color-paper);
  color: var(--color-ink);
  font-family: var(--font-sans);
  -webkit-font-smoothing: antialiased;
}

/* visible keyboard focus in clay */
:focus-visible {
  outline: 2px solid var(--color-clay);
  outline-offset: 2px;
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
}
```

- [ ] **Step 3: Commit** — `git add web/index.html web/src/index.css && git commit -m "reskin: token layer (Claude warm paper) + fonts"`

### Task 2: Global neutral sweep

**Files:** Modify many under `web/src/` (mechanical)

- [ ] **Step 1:** Run the sweep commands from **Migration Reference §C** verbatim, in order.
- [ ] **Step 2: Verify no white-on-color regressed.** Run:

```bash
cd /root/ludandaye/ladder/hy2board/web/src
grep -rn 'bg-clay text-ink' . || echo "OK: no clay buttons lost their white text"
grep -rn 'text-white' . | grep -v 'bg-clay'   # any remaining text-white should be Layout (hand-done next) only
```

Expected: the first grep prints `OK`; the second prints only `Layout.tsx` hits (rewritten in Task 3) or nothing.

- [ ] **Step 3: Commit** — `git commit -am "reskin: global neutral sweep (text/bg/borders -> warm tokens)"`

### Task 3: Shell (`Layout.tsx`) — hand rewrite

**Files:** Modify `web/src/components/Layout.tsx`

- [ ] **Step 1:** Replace the outer wrapper + sidebar + logo. Change the root `<div>` (line 37) and `<aside>` (39-41) and logo block (44-58) to:

```tsx
    <div className="min-h-screen flex bg-paper">
      {/* Sidebar */}
      <aside
        className="w-[240px] flex flex-col relative shrink-0 bg-paper-rail"
        style={{ borderRight: "1px solid var(--color-border)" }}
      >
        {/* Logo */}
        <div className="px-5 py-6" style={{ borderBottom: "1px solid var(--color-border)" }}>
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-lg flex items-center justify-center bg-clay">
              <Zap size={16} className="text-white" />
            </div>
            <div>
              <h1 className="text-[17px] font-semibold tracking-tight text-ink font-serif">
                hy2board
              </h1>
              <p className="text-[10px] tracking-[0.15em] text-ink-faint uppercase">Admin Panel</p>
            </div>
          </div>
        </div>
```

- [ ] **Step 2:** Replace the nav-link styling (62-101). The active state becomes a clay pill; keep structure:

```tsx
        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1">
          <p className="px-3 mb-3 text-[10px] font-semibold tracking-[0.2em] text-ink-faint uppercase">Navigation</p>
          {nav.map(({ to, icon: Icon, label, desc }) => {
            const active = location.pathname === to
            return (
              <Link
                key={to}
                to={to}
                className={"group flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all duration-200 relative " +
                  (active ? "bg-clay/10" : "hover:bg-surface-2")}
              >
                {active && (
                  <div className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 rounded-r-full bg-clay" />
                )}
                <div className={"w-8 h-8 rounded-lg flex items-center justify-center transition-all duration-200 shrink-0 " +
                  (active ? "bg-clay/15 text-clay" : "bg-surface-2 text-ink-faint")}>
                  <Icon size={15} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className={"text-[13px] font-medium " + (active ? "text-clay" : "text-ink-muted group-hover:text-ink")}>
                    {label}
                  </p>
                  <p className="text-[10px] text-ink-faint truncate">{desc}</p>
                </div>
                {active && <ChevronRight size={12} className="text-clay" />}
              </Link>
            )
          })}
        </nav>
```

- [ ] **Step 3:** The bottom section (104-146): the Service Status link + system box keep green semantics (`text-green-…` stays). Update only the neutral inline styles — change `rgba(255,255,255,0.02)`/`0.04` (System box, line 123) to `var(--color-surface)` bg + `var(--color-border)` border, and `text-zinc-*` already remapped. Replace the System box opening div (121-124) with:

```tsx
          <div
            className="mx-2 px-3 py-3 rounded-lg bg-surface"
            style={{ border: "1px solid var(--color-border)" }}
          >
```

(The Service Status green link, the animated dot, and Sign Out button need no change — green/red utilities are kept.)

- [ ] **Step 4:** Top bar + page wrapper (152-177). Replace the top-bar `style` and the gradient avatar:

```tsx
        <div
          className="sticky top-0 z-30 flex items-center justify-between px-8 py-4"
          style={{
            background: "rgba(250, 249, 245, 0.8)",
            backdropFilter: "blur(12px)",
            borderBottom: "1px solid var(--color-border)",
          }}
        >
          <div>
            <h2 className="text-[16px] font-semibold text-ink font-serif">
              {nav.find((n) => n.to === location.pathname)?.label || "Page"}
            </h2>
            <p className="text-[11px] text-ink-muted">
              {nav.find((n) => n.to === location.pathname)?.desc || ""}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-[11px] text-ink-faint font-mono">{new Date().toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" })}</div>
            <div className="w-8 h-8 rounded-full flex items-center justify-center text-[11px] font-bold text-white bg-clay">
              A
            </div>
          </div>
        </div>

        {/* Page content */}
        <div className="p-8 text-ink">{children}</div>
```

- [ ] **Step 5: Commit** — `git commit -am "reskin: Layout shell to warm paper (clay nav pill, serif wordmark)"`

### Task 4: Dashboard charts + cards (`Dashboard.tsx`) — hand

**Files:** Modify `web/src/pages/Dashboard.tsx`

- [ ] **Step 1:** Recolor the constants (lines 43-44):

```tsx
const COLORS = ["#C96442", "#4C7A87", "#C2843E", "#6E8B6A", "#B5573A", "#8B6F47"]
const STATUS_COLORS = { active: "#3F8A4D", expired: "#C2843E", over: "#BE3A31", disabled: "#9A968C" }
```

- [ ] **Step 2:** `tooltipStyle` (64-67) → paper:

```tsx
const tooltipStyle = {
  contentStyle: { background: "#FFFFFF", border: "1px solid #E5E0D6", borderRadius: 8, fontSize: 12, color: "#141413" },
  itemStyle: { color: "#6B6862" },
}
```

- [ ] **Step 3:** `Card` component (72-83) → surface/border tokens:

```tsx
    <div className="rounded-xl p-5 bg-surface" style={{ border: "1px solid var(--color-border)" }}>
```

- [ ] **Step 4:** `HealthCell` (90-95) → surface/border + **serif numerals** (the signature). Replace the two inner divs:

```tsx
    <div onClick={onClick}
      className={"px-3 py-2.5 bg-surface border border-border rounded-lg" + (onClick ? " cursor-pointer hover:border-border-strong" : "")}>
      <div className="text-[11px] text-ink-faint mb-0.5">{label}</div>
      <div className={"text-xl font-serif font-semibold " + tone}>{value}</div>
```

- [ ] **Step 5:** Health-bar tones (216-224) — replace the light-on-dark tints with legible ink/clay/status tones:

```tsx
        <HealthCell label="节点 healthy/total" value={`${stats.healthy_nodes}/${stats.total_nodes}`}
          tone={stats.healthy_nodes < stats.total_nodes ? "text-warning" : "text-ink"} />
        <HealthCell label="在线" value={totalOnline} tone="text-ink" />
        <HealthCell label="告警" value={totalAlerts}
          tone={totalAlerts > 0 ? "text-danger" : "text-ink-faint"} onClick={() => navigate("/alerts")} />
        <HealthCell label="本月收入" value={incomeCur ? yuan(incomeCur.total_cents) : "¥0.00"}
          onClick={() => navigate("/finance")} tone="text-ink" />
        <HealthCell label="实时 ↑" value={lastPoint ? fmtSpeed(lastPoint.tx) : "-"} tone="text-clay" />
        <HealthCell label="实时 ↓" value={lastPoint ? fmtSpeed(lastPoint.rx) : "-"} tone="text-[#4C7A87]" />
```

(`HealthCell` default `tone` param stays `text-white`→ change its default to `text-ink`: line 87 `tone = "text-white"` → `tone = "text-ink"`.)

- [ ] **Step 6:** Chart series hex — replace every `#1a2b6b`→`#C96442` and `#d4a017`→`#4C7A87` in the chart JSX (gradients 301-306, areas 313-314, legend dots 319-320, speed bars 344-349, bar chart 496-497). Also the CartesianGrid strokes `rgba(255,255,255,0.04)` (309, 492) → `#E5E0D6`, and axis tick fills `#555`/`#666` (310-311, 493-494) → `#9A968C`. The per-node line (440) `#22c55e`/`#ef4444` stays (status). The "Top Traffic" bar bg `rgba(43, 71, 107, 0.25)` (403) → `rgba(201,100,66,0.12)`. The AI badge (408) `bg-purple-500/10 text-purple-400` → `bg-clay/10 text-clay`.

Use targeted edits (each hex appears in a known line above). Verify after with:
```bash
grep -nE '#1a2b6b|#d4a017|rgba\(255,255,255|purple-' web/src/pages/Dashboard.tsx
```
Expected: no matches.

- [ ] **Step 7: Commit** — `git commit -am "reskin: Dashboard charts + cards + serif KPI numerals"`

### Task 5: Build, deploy, USER GATE

- [ ] **Step 1:** scp the changed files to `/opt/hy2board`:

```bash
cd /root/ludandaye/ladder/hy2board
scp web/index.html ml:/opt/hy2board/web/index.html
scp web/src/index.css ml:/opt/hy2board/web/src/index.css
scp web/src/components/Layout.tsx ml:/opt/hy2board/web/src/components/Layout.tsx
# scp ALL files the sweep touched (safer to rsync the whole src):
rsync -a web/src/ ml:/opt/hy2board/web/src/
```

- [ ] **Step 2:** Tag rollback + build + hot swap (ssh to `ml` = `76.13.217.10`):

```bash
ssh ml 'cd /opt/hy2board && docker tag "$(docker inspect hy2board-hy2board-1 --format "{{.Image}}")" hy2board-hy2board:rollback-predesign && docker compose build && docker compose up -d && sleep 8'
```

- [ ] **Step 3: Verify build clean** — the build log shows `✓ built` with **no `error TS`** (a moved/renamed class that broke JSX surfaces here). Confirm the new Dashboard chunk and auth:

```bash
ssh ml 'docker exec hy2board-hy2board-1 sh -c "ls /app/web/dist/assets/Dashboard-*.js" && curl -s -X POST localhost:9000/api/admin/login -d "{\"username\":\"ludandaye\",\"password\":\"8011\"}" -H "Content-Type: application/json" | head -c 80'
```
Expected: a `Dashboard-*.js` path prints; login returns `{"ok":true...` / a token.

- [ ] **Step 4: USER GATE (you):** open the panel. Confirm: cream paper bg; paper sidebar with **clay pill on the active item** + serif `hy2board` wordmark; Dashboard health bar in **serif numerals**, readable, clay `实时↑`; charts in clay/teal (no blue/gold); cards are white with warm hairline borders; "✅ 一切正常" / alert cards legible. **Stop here for user approval before Phase 2.**

- [ ] **Step 5: Commit + push** — `git push origin HEAD:main`.

---

## Phase 2 — Remaining pages (after user approves Phase 1)

The global sweep (Task 2) already warmed every page. Phase 2 is **polish + verify** per file: catch sweep casualties, recolor any page-specific accents/charts, eyeball it. Do these as separate small tasks; each is the same shape.

**Per-page procedure (apply to each file below):**

- [ ] **Step 1:** Residue grep — `grep -nE '#1a2b6b|#d4a017|#8900ff|rgba\(255,255,255|text-zinc-200|bg-zinc-[78]00 .*text-ink|blue-[0-9]|sky-[0-9]|purple-[0-9]' web/src/pages/<File>.tsx`. Fix each hit per Migration Reference §A/§D (brand→clay, white-alpha→ink-alpha, decorative blue/purple→clay, light-chip text-zinc-200→`text-ink`).
- [ ] **Step 2:** Open the page in the running panel; check text contrast, button readability (primary = clay, secondary = light gray w/ ink text), table borders, badges.
- [ ] **Step 3:** scp the file + `rsync -a web/src/` already covers it; rebuild only once at the end of Phase 2.
- [ ] **Step 4: Commit** — `git commit -am "reskin: <File> residue polish"`.

**Files (each its own task):**

### Task 6: `Finance.tsx` (has charts — hand recolor)
Recolor recharts: `stroke="#27272a"` (137, grid) → `#E5E0D6`; `stroke="#71717a"` (138-139, axes) → `#9A968C`; series `#60a5fa`→`#C96442`, `#f87171`→`#BE3A31`, `#4ade80`→`#3F8A4D` (141-143). Then the per-page procedure.

### Task 7: `Users.tsx` — per-page procedure (largest table; check expand rows, edit modal trigger, payment dialog button)
### Task 8: `Nodes.tsx` — per-page procedure (live columns, "TCP 回落" section toggles, register paste)
### Task 9: `Alerts.tsx` — per-page procedure (the three risk tables; clickable rows)
### Task 10: `Plans.tsx` — per-page procedure
### Task 11: `Rules.tsx` — per-page procedure (dropdown selectors)
### Task 12: `StaticIPs.tsx` — per-page procedure (IP health dots, latency)
### Task 13: `AuditLogs.tsx` — per-page procedure (read-only table, pagination)
### Task 14: `TelegramBot.tsx` — per-page procedure (status indicator)
### Task 15: `UserPortal.tsx` — per-page procedure (user self-service view)
### Task 16: `Downloads.tsx` — per-page procedure (card layout)
### Task 17: Shared modals — `UserEditModal.tsx`, `PlanEditModal.tsx`, `PaymentDialog.tsx`
Per-page procedure; verify modal backdrop is the warm `bg-ink/40` (not heavy black), inputs are `bg-surface` w/ `border-border-strong`, primary action = clay.

### Task 18: Phase 2 build + deploy
- [ ] `rsync -a web/src/ ml:/opt/hy2board/web/src/`
- [ ] `ssh ml 'cd /opt/hy2board && docker compose build && docker compose up -d && sleep 8'` — build clean (no `error TS`).
- [ ] Click through every page once; no leftover dark panel, no invisible text, no blue/gold accent.
- [ ] `git commit -am "reskin: Phase 2 pages polished" && git push origin HEAD:main`.

---

## Phase 3 — Login + global polish

### Task 19: `Login.tsx` reskin
**Files:** Modify `web/src/pages/Login.tsx`
- [ ] **Step 1:** In its local `:root` (lines 135-142) change `--accent: #2b47ff` → `#C96442`, `--gold: #ffae00` → `#C2843E`; set `--font-head` → `"Newsreader", serif`. Change the page background to `#FAF9F5` and text to `#141413` (find the dark bg in this file via `grep -nE '#08090d|#0c0d12|background|grain' web/src/pages/Login.tsx`).
- [ ] **Step 2:** Recolor remaining electric hexes (`#00c2ff`, `#00d8ff`, `#9499ff`, `#9135ff`, `#2b47ff`) → clay/teal; lower the grain overlay opacity so it reads on cream. Keep the loading animation (it's recolored, not removed) — `prefers-reduced-motion` already handled globally in Task 1.
- [ ] **Step 3:** Visual check the login + loading screen at `/login` (logged out). **Commit** — `git commit -am "reskin: Login to warm paper"`.

### Task 20: Global a11y + contrast pass
- [ ] **Step 1:** Full-app grep for any survivor: `grep -rnE '#08090d|#0c0d12|#111318|bg-black|text-white' web/src | grep -v 'bg-clay'` → fix any non-clay hit.
- [ ] **Step 2:** Eyeball contrast on the faintest text (`text-ink-faint` labels) on `bg-paper`; if a label is too light, bump `--color-ink-faint` to `#8A867C` in `index.css`. Confirm status colors (success/warning/danger) are distinguishable from clay on small badges.
- [ ] **Step 3:** Tab through the Dashboard + a form — confirm the clay `:focus-visible` ring shows. Confirm mobile width (`<sm`) doesn't break the sidebar/grids.
- [ ] **Step 4:** Final build/deploy: `rsync -a web/src/ ml:/opt/hy2board/web/src/ && ssh ml 'cd /opt/hy2board && docker compose build && docker compose up -d && sleep 8'`.
- [ ] **Step 5: Commit + push + memory** — `git commit -am "reskin: global a11y/contrast polish" && git push origin HEAD:main`. Update `/root/.claude/projects/-root-ludandaye-ladder/memory/hy2board-panel-deployment.md` marking ③ done with rollback `rollback-predesign`.

---

## Self-Review

**1. Spec coverage:**
- Light "paper" palette + tokens → Task 1 (§A). Type (Newsreader/Inter/JetBrains) → Task 1 fonts. Clay accent → §A + Layout/Dashboard. Chart palette → Task 4/§A. Signature (serif KPI numerals) → Task 4 Step 4-5. Chrome (sidebar pill, flat cards, top bar, buttons, tables, badges, inputs/modals) → Task 3 + §C buttons + Task 17. Login → Task 19. Zinc-remap + overload handling → §B + §C. Phased rollout w/ Dashboard gate → Phase 1 Task 5 Step 4. Contrast/focus/reduced-motion → Task 1 + Task 20. No-behavior-change → only color/font/class edits; verified by build + click-through. Deploy/rollback → Task 5/18/20. **All spec sections mapped.**

**2. Placeholder scan:** No "TBD"/"handle edge cases". Every code step shows exact code or exact `sed`/`grep`/`ssh` commands. Per-page Phase 2 tasks reference a concrete shared procedure + named greps (not "similar to Task N" hand-waving — the procedure IS the content, and chart files Finance/Dashboard get explicit hex edits). ✓

**3. Type/name consistency:** token class names (`bg-paper`, `bg-surface`, `bg-surface-2`, `border-border`, `border-border-strong`, `text-ink`, `text-ink-muted`, `text-ink-faint`, `bg-clay`, `text-clay`, `bg-clay-hover`, `text-success/warning/danger`) are defined once in §A/Task 1 and used identically in Tasks 3-4 and the sweeps. `--color-*` var names in `index.css` match the `*` Tailwind suffixes. Chart hexes `#C96442/#4C7A87/#C2843E/#6E8B6A` consistent between §A, Task 4, Task 6. `rollback-predesign` image name consistent (Task 5 creates, Task 20 references). ✓

**Risk note:** the zinc value-remap is the one approximate lever; §B lists the known minor casualties and Phase 2's per-page residue grep (`text-zinc-200`, overloaded button checks) catches them. Build failures = a class typo; the `error TS` / chunk checks gate each phase.
