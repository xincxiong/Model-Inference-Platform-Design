# Phase 3 — UI Review

**Audited:** 2026-04-30
**Baseline:** Abstract 6-pillar standards (no UI-SPEC.md exists)
**Screenshots:** Captured (desktop, tablet, mobile) at localhost:3000

Pages audited: `/observability`, `/datalab`, `/costs`, `/compare`
Baseline comparison: existing pages (`/usage`, `/models`, `/endpoints`) and `globals.css` design tokens

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 3/4 | Chinese labels consistent; no generic "OK/Cancel" in new pages; but some inline styles lack aria-labels |
| 2. Visuals | 3/4 | Clear visual hierarchy via cards/tables; SVG charts well-structured; but mobile/tablet views show sidebar-only with blank content area |
| 3. Color | 3/4 | Consistent CSS variable usage; 60/30/10 roughly respected; but 12 hardcoded hex colors in data model (acceptable as data, not styling) |
| 4. Typography | 2/4 | Only `text-xs`/`text-sm`/`text-xl`/`text-[10px]`/`text-[11px]` used — good restraint; but `text-[10px]` and `text-[11px]` are arbitrary values outside the design system scale |
| 5. Spacing | 2/4 | Uses Tailwind scale consistently; but `max-w-[200px]`, `max-w-[300px]`, `min-w-[500px]` arbitrary values; chart SVGs hardcoded to `viewBox="0 0 800 ..."` limiting responsiveness |
| 6. Experience Design | 2/4 | Hover states and disabled states present on interactive elements; but NO loading states, NO error/empty states on any of the 4 new pages — all assume data exists |

**Overall: 15/24**

---

## Top 3 Priority Fixes

1. **No loading/error/empty states on any new page** — If the API returns no data or fails, the user sees a blank page with no feedback — Add loading skeletons to all 4 pages; add empty-state illustrations with guidance text; add error boundary with retry button
2. **Charts hardcoded to 800px viewBox, break on mobile** — The SVG charts use `viewBox="0 0 800 {h}"` with `min-w-[500px]`, which causes horizontal scroll on screens <500px and looks terrible on mobile — Change viewBox to be dynamic based on container width, or use percentage-based widths with `preserveAspectRatio`
3. **Arbitrary font sizes `text-[10px]` and `text-[11px]` violate the type scale** — The design system defines sizes via Tailwind scale (`xs`/`sm`/`base`), but 4 pages use bracket notation for sub-label text — Replace `text-[10px]` with `text-[10px]` mapped to a design token like `--text-xs` or standardize to `text-xs` with CSS transform

---

## Detailed Findings

### Pillar 1: Copywriting (3/4)

**What's good:**
- All Chinese labels are contextually appropriate (可观测性, 成本分析, 模型对比, 数据实验室)
- No generic English labels like "OK", "Cancel", "Save" found in the 4 new pages
- Table headers use concise Chinese terms (占比, 成本, 延迟)

**Issues:**
- SVG chart text labels are hardcoded in English in `charts.tsx:42` (`fill="var(--text-muted)"`) — acceptable since they render numbers, but the `yAxisLabel` prop is passed as `"ms"`, `"%"`, `"$"` — should be localized
- Icon-only elements in SVG charts have no `aria-label` attributes (e.g., legend lines in `charts.tsx:75`)
- The SQL query editor button uses `▶ 运行` — the emoji/symbol is not a standard icon, consider using an SVG play icon instead

### Pillar 2: Visuals (3/4)

**What's good:**
- Strong card-based hierarchy consistent with `globals.css` `.card` class
- Tables use consistent column alignment (left for text, right for numbers)
- Model color coding is consistent across pages (DeepSeek=blue, Qwen=orange, etc.)
- Donut-style SVG circles in `/compare` provide clear visual scoring

**Issues:**
- **Mobile view (screenshot):** Sidebar takes full width, content area is blank white — no responsive layout collapse. The `grid-cols-2 md:grid-cols-4` pattern in `/costs` stat cards is correct, but the sidebar doesn't collapse on mobile
- **Tablet view (screenshot):** Shows "Loading models..." in content area — this is from an existing page (`/models`), not the new pages, but confirms the layout issue: sidebar is fixed at 240px regardless of viewport
- Chart areas have no Y-axis title (only labels), making them ambiguous without context
- The stacked bar chart in `/costs` renders overlapping colored rectangles with random heights (`Math.random()`) — in production this would be replaced with real data, but the visual structure needs fixed segment ordering

### Pillar 3: Color (3/4)

**What's good:**
- All new pages use CSS variables from `globals.css` (`var(--accent)`, `var(--text)`, `var(--text-muted)`, `var(--bg-secondary)`)
- No new hardcoded color values introduced in class attributes — colors are only defined as data properties in model arrays
- 60/30/10 rule roughly followed: white backgrounds dominate, gray secondary surfaces, blue accent sparingly

**Issues:**
- 12 hardcoded hex color values exist in data models (`compare/page.tsx:24-29`, `costs/page.tsx:8-13`) — these are model color identifiers, not styling per se, but they bypass the CSS variable system. If these colors need to change, they require a code deploy
- `charts.tsx:60` uses hardcoded `stopOpacity="0.12"` and `stopOpacity="0"` — acceptable for gradient aesthetics but worth noting
- `charts.tsx:126` uses `opacity="0.8"` hardcoded — could be a CSS variable

### Pillar 4: Typography (2/4)

**What's good:**
- Only 4 distinct sizes in use across new pages: `text-xs`, `text-sm`, `text-xl`, and two arbitrary sizes
- Font weights limited to `font-medium`, `font-semibold`, and one `font-bold` — restrained

**Issues:**
- `text-[10px]` used 4 times in `compare/page.tsx` (lines 137, 151, 182, 219) — outside the Tailwind type scale
- `text-[11px]` used 4 times in `costs/page.tsx:194,196` and `observability/page.tsx:119,121` — also outside the scale
- `globals.css` imports `Google Sans` (400/500/700) and `Inter` (400/500/600/700), but new pages don't specify `font-family` on specific elements, relying on inheritance — this is correct but fragile
- `charts.tsx:42,49` specify `fontFamily="monospace"` inline for axis labels — inconsistent with the Inter/Google Sans system

### Pillar 5: Spacing (2/4)

**What's good:**
- Consistent use of `gap-2`, `gap-3`, `gap-4` for card grids
- Padding follows a clear pattern: `py-2.5` for table rows, `px-3`/`py-1.5` for buttons
- `mb-1`, `mb-3`, `mb-4`, `mb-5`, `mb-6` for vertical section spacing — systematic

**Issues:**
- `max-w-[200px]` in `datalab/page.tsx:151` and `max-w-[300px]` in `datalab/page.tsx:316` — arbitrary constraints that won't adapt to content changes
- `min-w-[500px]` in `charts.tsx:37,113` and `costs/page.tsx:220` — forces horizontal scroll on narrow viewports
- SVG chart `viewBox` is hardcoded to `800` width — the chart doesn't truly respond to container size; it just scales down via CSS `w-full`
- `charts.tsx` SVG width is `w-800` (800px fixed in viewBox), which means on a 1440px desktop the chart occupies only ~55% of available width

### Pillar 6: Experience Design (2/4)

**What's good:**
- Hover states present on all interactive elements (`hover:bg-[var(--bg-secondary)]`, `hover:border-[var(--accent)]`)
- Disabled states on pagination buttons (`disabled:opacity-40`)
- Transition classes on buttons and tabs (`transition-colors`, `transition-all`)
- Tab switching uses clear active/inactive visual differentiation

**Issues:**
- **NO loading states** — All 4 pages use `useState` with mock data generators that run synchronously. In production with real API calls, there are no skeleton screens, spinners, or loading indicators
- **NO error states** — No `try/catch` blocks around data fetching; no error boundary; no error message display. If the API fails, the page renders with empty/undefined data
- **NO empty states** — If `modelCosts` is empty (line 69 `costs/page.tsx`), the division by zero in `avgDaily` calculation would produce `NaN`. Tables would render with zero rows but no "no data" message
- **NO confirmation for destructive actions** — The `删除` (delete) button in `datalab/page.tsx:272` has no confirmation dialog
- The `编辑` (edit) button in `costs/page.tsx:321` is non-functional (no onClick handler) — a dead UI element
- SVG charts have no tooltip or hover interaction — users cannot get precise values from the visualizations

---

## Automated Screenshot Verification

Desktop screenshots captured for all 4 new pages confirm:
- Cards render correctly with proper border and shadow
- Tables display with correct column alignment
- Charts render within card containers
- Sidebar navigation highlights active page correctly

Mobile/tablet screenshots show the sidebar-only view (no responsive content area detected in screenshot capture), confirming the fixed sidebar layout issue noted in Pillar 2.

---

## Registry Safety

No `components.json` found — no shadcn or third-party UI registries in use.
No registry safety audit required.

---

## Files Audited

- `model-inference-platform/frontend/src/app/observability/page.tsx` (280 lines)
- `model-inference-platform/frontend/src/app/datalab/page.tsx` (329 lines)
- `model-inference-platform/frontend/src/app/costs/page.tsx` (350 lines)
- `model-inference-platform/frontend/src/app/compare/page.tsx` (375+ lines)
- `model-inference-platform/frontend/src/components/charts.tsx` (137 lines)
- `model-inference-platform/frontend/src/components/Sidebar.tsx` (updated)
- `model-inference-platform/frontend/src/app/globals.css` (design system baseline)
