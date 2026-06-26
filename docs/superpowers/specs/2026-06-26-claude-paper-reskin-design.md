# Claude "warm paper" global reskin

Date: 2026-06-26
Status: approved direction (light "paper" mode chosen by user) — sub-project 3 of 3 (global visual/UX refresh; follows node-management UI + Dashboard hierarchy)

## Problem

The hy2board admin panel is a **cold dark** UI: shell `#08090d`, sidebar `#0c0d12`,
cards `#111318`, neutral **zinc** grays throughout, brand accents deep-blue `#1a2b6b`
+ gold `#d4a017`, charts in blue/gold. It works but looks like a generic dark
dashboard. The user wants it reskinned to **Claude's signature look** — the warm cream
"paper" aesthetic people picture when they think of Claude.

The frontend-design skill normally flags warm-cream + terracotta as one of the three
"AI-generated default" looks to avoid. Here it is the **explicit brief** ("做 Claude 那
种米色纸感的 UI"), so per that skill's own rule — *the brief's words win, including when
it asks for one of these looks* — we follow it deliberately. The real design work is
adapting that calm, airy, editorial aesthetic to a **data-dense ops panel** (tables,
live charts, KPI cells) without it becoming cramped or losing legibility.

## Goal

Reskin every page to a light, warm "paper" theme derived from Claude's design language —
a centralized token system (one CSS file drives the palette), serif editorial headings,
one clay accent — **without changing any behavior, data, endpoints, or layout structure**.
Same panel, same features; new skin.

## Non-goals

- No functional/data/endpoint/routing changes. No new pages, metrics, or charts.
- No layout restructure (the Dashboard 4-zone work is already done; this is *visual*).
- No dark-mode toggle in this pass. Ship **light paper only**. The token layer is built so
  a future warm-dark toggle is cheap, but YAGNI — not now.
- No component-library migration (stay on Tailwind v4 + recharts; no shadcn/MUI).

## Design direction: "warm paper"

A calm cream sheet with near-black warm ink and a single terracotta/clay accent doing all
the emphasis. Flat cards defined by hairline warm borders (Claude leans on borders, not
drop shadows). Serif numerals/titles give it the editorial, "written" feel; the dense data
stays in a clean sans. Spend the boldness in **one** place — the clay accent + serif
headline numerals — and keep everything else quiet.

### Color tokens (light "paper")

| token | value | role |
|---|---|---|
| `--paper` | `#FAF9F5` | app background |
| `--paper-rail` | `#F0ECE0` | sidebar / rail / table zebra |
| `--surface` | `#FFFFFF` | cards, modals, inputs |
| `--border` | `#E5E0D6` | hairline borders / dividers |
| `--border-strong` | `#D9D3C6` | input borders, emphasis dividers |
| `--ink` | `#141413` | primary text (warm near-black) |
| `--ink-muted` | `#73706A` | secondary text |
| `--ink-faint` | `#9A968C` | labels, captions, placeholders |
| `--clay` | `#C96442` | **the accent** — primary buttons, active nav, key chart series, links |
| `--clay-hover` | `#B5573A` | hover/pressed clay |
| `--clay-soft` | `rgba(201,100,66,0.10)` | active-nav pill, clay badges, highlights |
| `--success` | `#3F8A4D` | online / active (deeper green for contrast on cream) |
| `--warning` | `#C2843E` | expiring / near-limit (ochre) |
| `--danger` | `#BE3A31` | offline / over-limit (brick red — kept distinct from clay's orange) |

**Status vs accent:** clay `#C96442` is orange-terracotta; danger `#BE3A31` is a redder
brick. They must stay visually distinguishable on small badges — clay is *never* used for
alarm states, only for neutral emphasis/branding.

### Chart palette (recharts — replaces blue/gold)

A warm, harmonious set with enough separation for multi-series charts:

| series role | value |
|---|---|
| primary (e.g. tx / income / main bar) | `--clay` `#C96442` |
| secondary (e.g. rx / second series) | `#4C7A87` (calm desaturated teal — distinct from clay) |
| tertiary | `#C2843E` (ochre) |
| quaternary | `#6E8B6A` (sage) |
| grid / axis | `--border` `#E5E0D6` lines, `--ink-faint` labels |
| pie/status segments | success/warning/danger tokens above |

### Typography

Loaded via Google Fonts in `index.html` (`<link>` with `display=swap`).

| role | family | usage |
|---|---|---|
| display / serif | **Newsreader** | logo wordmark, page titles, headline KPI numerals (Tiempos/Copernicus editorial feel). Used with restraint — headings + hero numbers only, never body. |
| UI / body | **Inter** | all UI text, tables, forms, labels — legible at the panel's small sizes (`text-xs`/`text-[11px]`). |
| mono | **JetBrains Mono** | IDs, traffic byte counts, IPs, code-ish values. |

Tailwind exposes these as `font-serif` (Newsreader), `font-sans` (Inter, default),
`font-mono` (JetBrains Mono) via `@theme`.

### Chrome / component specs

- **Sidebar** (`Layout.tsx`): `--paper-rail` bg, `--border` right edge; logo wordmark in
  Newsreader serif; nav items in Inter; **active item = clay text on a `--clay-soft` rounded
  pill**; inactive = `--ink-muted`, hover `--ink`.
- **Top bar**: cream `rgba(250,249,245,0.8)` + `backdrop-blur`, `--border` bottom.
- **Cards**: `--surface` bg, 1px `--border`, `rounded-xl` (~12px), **no heavy shadow** (an
  optional `shadow-[0_1px_2px_rgba(20,20,19,0.04)]` whisper only). Section titles in serif.
- **Buttons**: primary = `--clay` bg + white text, hover `--clay-hover`; secondary =
  `--surface` bg + `--ink` text + `--border`. (Current `bg-white text-black` primary would
  vanish on cream — must become clay.)
- **Tables**: header row `--ink-faint` uppercase Inter; body `--ink`; row borders `--border`;
  hover/zebra `--paper-rail`. Numeric columns in mono.
- **Badges**: tinted by semantic token at ~10% bg + solid token text (e.g. online =
  `--success` text on `rgba(63,138,77,0.10)`).
- **Inputs / modals**: `--surface` bg, `--border-strong` border, `--ink` text, clay focus
  ring (`ring-2 ring-clay/40`). Modal backdrop `rgba(20,20,19,0.35)` (lighter than the
  current `bg-black/60`, which reads heavy on a light app).
- **Login page** (`Login.tsx`): self-contained — its own `:root` (electric-blue `--accent`,
  grain overlay, character animation). Reskin to paper: cream bg, grain at low opacity, clay
  accent, Newsreader wordmark. The loading animation stays (motion respected) but recolored.

### Signature element

The one memorable thing: **clay-on-paper editorial KPIs** — the Dashboard health-bar
numbers and key card figures set in **Newsreader serif**, with the clay accent as the only
color that "speaks." Everything else (tables, controls, charts grid) is quiet warm neutrals.
This is what makes it read unmistakably "Claude" rather than a recolored generic dashboard.

## Architecture / engineering strategy

The colors currently live in **three** places: Tailwind utility classes (`bg-zinc-900`,
`text-zinc-400`, `border-zinc-800`, `text-white`, `bg-black`), hardcoded inline `style={{}}`
hex (shell, charts, modals), and Login's local `:root`. The strategy centralizes as much as
possible and surgically handles the rest.

1. **Token layer in `index.css`** (currently just `@import "tailwindcss"`). Add a Tailwind
   v4 `@theme` block defining semantic colors (`--color-paper`, `--color-surface`,
   `--color-border`, `--color-ink`, `--color-ink-muted`, `--color-ink-faint`, `--color-clay`,
   `--color-clay-soft`, `--color-success/warning/danger`, chart colors) → usable as
   `bg-paper`, `text-ink`, `border-border`, `bg-clay`, etc. Set the page `body`/`#root` to
   `bg-paper text-ink` and the default font to Inter.

2. **Zinc-ramp override (the big lever).** Most pages use `bg-zinc-900` (dark surface),
   `border-zinc-800` (border), `text-zinc-300/400/500/600` (muted text). Redefine the zinc
   scale in `@theme` to *light-mode-appropriate warm values* so those hundreds of existing
   classes flip automatically without editing each page:
   - `--color-zinc-950/900/800` → light surfaces/rails (`#FFFFFF`/`#FAF9F5`/`#F0ECE0`)
   - `--color-zinc-700/800` (as borders) → warm light borders (`#E5E0D6`/`#D9D3C6`)
   - `--color-zinc-300/400/500/600` (as text) → warm grays dark enough for cream
     (`#9A968C`→`#73706A`→`#5C5950`…)
   This is a **non-monotonic remap** (text shades pulled dark, surface shades pushed light)
   — valid because the code uses these shades *semantically* (400 = muted text, 900 = surface),
   not by relative ordering.

   **Honest caveat — overloaded shades.** The remap is not a clean silver bullet: some shades
   carry *two* meanings. `zinc-800` is used both as a border (`border-zinc-800`) **and** as the
   secondary-button surface (`bg-zinc-800 text-white`) — remap it to a light border color and the
   buttons turn invisible (light bg + white text). So the override is safe for the *unambiguous*
   shades (the `text-zinc-300/400/500/600` muted-text family, and card/page background shades) but
   the overloaded ones must be resolved in the surgical pass. **Phase-1 task: audit each zinc shade's
   actual uses (grep `bg-/text-/border-` separately) and decide override-vs-replace per shade**;
   secondary buttons (`bg-zinc-800 text-white`) restyle to `bg-surface text-ink border-border`
   alongside the primary-button change in step 3.

3. **Surgical find/replace for what the ramp can't reach** — `text-white` (×53, primary text
   → `text-ink`), `bg-black` (×36, dark surface → `bg-surface`/`bg-paper`), primary buttons
   (`bg-white text-black` → `bg-clay text-white`). Done per-file with review, not blind sed.

4. **Inline-hex spots** — `Layout.tsx` shell backgrounds, the recharts series/grid colors in
   `Dashboard.tsx`/`Finance.tsx`, modal backdrops, brand `#1a2b6b`/`#d4a017`. Replaced with
   token references (CSS var or the new Tailwind classes).

5. **Login.tsx** — reskinned as its own unit (local `:root` → paper palette, recolor grain +
   animation).

6. **Fonts** — `<link>` in `index.html`; `@theme` `--font-*` mappings.

This makes the bulk of the reskin a handful of central edits (index.css + index.html +
Layout shell + chart constants) plus a bounded find/replace, rather than rewriting 13 pages.

## Rollout (de-risked, phased — matches the project's cautious deploy pattern)

- **Phase 1 — Foundation + shell + Dashboard pilot.** Token layer, fonts, zinc override,
  `Layout.tsx` shell, and `Dashboard.tsx` (incl. chart recolor). Build, deploy behind a
  `rollback-predesign` image, **user verifies the look on the live Dashboard before we go
  further.** This is the "先看一个再决定" gate.
- **Phase 2 — Remaining pages.** Users, Nodes, Finance, Plans, Alerts, Rules, AuditLogs,
  StaticIPs, TelegramBot, UserPortal, Downloads — each swept for `text-white`/`bg-black`/
  button/inline-hex residue and visually checked.
- **Phase 3 — Login + polish.** Reskin Login; final pass for contrast, focus rings, any
  missed cold-gray, reduced-motion.

## Error handling / edge cases

- **Contrast:** every text token must hit WCAG AA on its background (`--ink-muted #73706A`
  on `--paper #FAF9F5` ≈ 4.7:1 ✓; verify faint/labels — bump darker if a label fails).
- **Status legibility:** online/expiring/offline must stay instantly distinguishable on cream;
  danger brick kept distinct from clay orange.
- **No behavior change:** subscriptions, API calls, computed values, routing untouched —
  purely presentational. Verify the app still builds and every page renders the same data.
- **Accessibility floor:** visible `:focus-visible` clay ring; `prefers-reduced-motion`
  respected on the Login animation; responsive down to mobile preserved.
- **Charts:** recharts needs explicit hex (can't read CSS vars) — pass the chart-palette hex
  constants; ensure series stay distinguishable for color-vision-deficient users (clay vs
  teal vs ochre differ in lightness, not just hue).

## Testing

No frontend test runner (vite + tsc only). Verify via:
- `tsc -b && vite build` — type-check + build clean (a moved/renamed class that breaks JSX
  shows here).
- **Live visual check per page** — open each page; confirm cream paper, clay accent, serif
  headings, readable text, no leftover dark/zinc panel, charts recolored, badges legible.
- **No-op behavior check** — same data renders; a subscription/endpoint smoke check confirms
  nothing functional moved.

## Deploy

Frontend-only. Per page-phase: scp changed files to `/opt/hy2board`, tag rollback image
(`hy2board-hy2board:rollback-predesign` before Phase 1), `docker compose build && up -d`,
verify the new Dashboard chunk + auth `ok:true`. Commit + push each phase (secrets stay out:
no config.yaml/data/backups). Update the memory file when ③ lands.
