# DoIt Frontend Redesign — Implementation Plan

> Design specification and implementation guide for updating the DoIt PWA frontend.
> Based on wireframes in this directory. `layout-shell.html` and `design-system.html` are the canonical style references.

---

## 1. Design Token Migration

**Priority: Do this first. Everything else depends on it.**

### 1.1 Update `web/src/constants.ts`

Replace the current palette with the evolved design system values:

```typescript
export const COLORS = {
  blue: '#3478F6',
  red: '#FF453A',
  orange: '#FF9F0A',
  yellow: '#FFD60A',
  green: '#30D158',
  teal: '#64D2FF',
  purple: '#BF5AF2',
  pink: '#FF375F',
  indigo: '#5E5CE6',
  brown: '#AC8E68',
  gray: '#8E8E93',
} as const

export const PRESET_COLORS = [
  COLORS.blue, COLORS.red, COLORS.orange, COLORS.green,
  COLORS.purple, COLORS.pink, COLORS.gray,
]

export const PRIORITY_COLORS: Partial<Record<0 | 1 | 2 | 3, string>> = {
  1: COLORS.green,   // low
  2: COLORS.orange,  // medium
  3: COLORS.red,     // high
}

export const UI = {
  accent: '#3478F6',
  accentHover: '#2563EB',
  danger: '#FF453A',
  success: '#30D158',
  warning: '#FF9F0A',
  textPrimary: '#1C1C1E',
  textSecondary: '#636366',
  textTertiary: '#AEAEB2',
  textQuaternary: '#C7C7CC',
  bgPrimary: '#FFFFFF',
  bgSecondary: '#F2F2F7',
  bgTertiary: '#E5E5EA',
  codeBg: '#F2F2F7',
} as const
```

### 1.2 Add Tailwind CSS Custom Properties

Create/update CSS custom properties in the global stylesheet. These tokens are the single source of truth — all components reference these, never raw hex values.

**Light mode (`:root`):**

```css
:root {
  /* Brand */
  --color-accent: #3478F6;
  --color-accent-hover: #2563EB;
  --color-accent-light: rgba(52, 120, 246, 0.10);
  --color-accent-medium: rgba(52, 120, 246, 0.18);

  /* Palette */
  --color-red: #FF453A;
  --color-orange: #FF9F0A;
  --color-yellow: #FFD60A;
  --color-green: #30D158;
  --color-teal: #64D2FF;
  --color-purple: #BF5AF2;
  --color-pink: #FF375F;
  --color-indigo: #5E5CE6;
  --color-brown: #AC8E68;
  --color-gray: #8E8E93;

  /* Semantic */
  --color-bg: #FFFFFF;
  --color-bg-secondary: #F2F2F7;
  --color-bg-tertiary: #E5E5EA;
  --color-bg-elevated: #FFFFFF;
  --color-text-primary: #1C1C1E;
  --color-text-secondary: #636366;
  --color-text-tertiary: #AEAEB2;
  --color-text-quaternary: #C7C7CC;
  --color-separator: rgba(60, 60, 67, 0.12);
  --color-separator-opaque: #D1D1D6;
  --color-danger: #FF453A;
  --color-success: #30D158;
  --color-warning: #FF9F0A;

  /* Priority */
  --priority-high: #FF453A;
  --priority-medium: #FF9F0A;
  --priority-low: #30D158;

  /* Typography */
  --font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'SF Pro Display', system-ui, sans-serif;
  --font-mono: 'SF Mono', SFMono-Regular, ui-monospace, monospace;

  /* Spacing (4px grid) */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;

  /* Radius */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --radius-xl: 20px;
  --radius-full: 9999px;

  /* Shadows (dual-layer) */
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.04);
  --shadow-card: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04);
  --shadow-popover: 0 4px 16px rgba(0,0,0,0.10), 0 1px 4px rgba(0,0,0,0.06);
  --shadow-modal: 0 12px 40px rgba(0,0,0,0.12), 0 4px 12px rgba(0,0,0,0.06);
  --shadow-fab: 0 4px 14px rgba(52,120,246,0.30), 0 2px 6px rgba(0,0,0,0.08);

  /* Transitions */
  --transition-fast: 0.15s ease;
  --transition-normal: 0.2s ease;
  --transition-slow: 0.3s ease;

  /* Z-index */
  --z-base: 1;
  --z-card: 10;
  --z-sticky: 20;
  --z-popover: 30;
  --z-modal: 40;
  --z-fab: 50;
  --z-toast: 60;

  /* Layout */
  --sidebar-width: 272px;
  --bottom-nav-height: 50px;
  --detail-panel-width: 380px;
}
```

**Dark mode (`[data-theme="dark"]` AND `@media (prefers-color-scheme: dark)`):**

```css
[data-theme="dark"],
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --color-accent: #4A9EFF;
    --color-accent-hover: #6BB3FF;
    --color-accent-light: rgba(74, 158, 255, 0.12);
    --color-accent-medium: rgba(74, 158, 255, 0.20);
    --color-red: #FF6961;
    --color-orange: #FFB340;
    --color-yellow: #FFD426;
    --color-green: #4ADE80;
    --color-teal: #5BC0EB;
    --color-purple: #C084FC;
    --color-pink: #FF6B8A;
    --color-indigo: #818CF8;
    --color-brown: #C4A882;
    --color-gray: #98989D;
    --color-bg: #1C1C1E;
    --color-bg-secondary: #2C2C2E;
    --color-bg-tertiary: #3A3A3C;
    --color-bg-elevated: #2C2C2E;
    --color-text-primary: #F5F5F7;
    --color-text-secondary: #98989D;
    --color-text-tertiary: #636366;
    --color-text-quaternary: #48484A;
    --color-separator: rgba(84, 84, 88, 0.65);
    --color-separator-opaque: #48484A;
    --color-danger: #FF6961;
    --color-success: #4ADE80;
    --color-warning: #FFB340;
    --priority-high: #FF6961;
    --priority-medium: #FFB340;
    --priority-low: #4ADE80;
    --shadow-sm: 0 1px 2px rgba(0,0,0,0.20);
    --shadow-card: 0 1px 4px rgba(0,0,0,0.30), 0 1px 2px rgba(0,0,0,0.20);
    --shadow-popover: 0 4px 20px rgba(0,0,0,0.40), 0 1px 6px rgba(0,0,0,0.25);
    --shadow-modal: 0 12px 48px rgba(0,0,0,0.50), 0 4px 16px rgba(0,0,0,0.30);
    --shadow-fab: 0 4px 16px rgba(74,158,255,0.25), 0 2px 6px rgba(0,0,0,0.20);
  }
}
```

### 1.3 Extend Tailwind Config

Map CSS custom properties to Tailwind utility classes so components can use `bg-bg-secondary`, `text-text-primary`, `shadow-card`, etc.

---

## 2. Layout Shell Redesign

**Reference wireframe:** `layout-shell.html`

### 2.1 Sidebar (`web/src/components/layout/Sidebar.tsx`)

| Property | Current | New |
|---|---|---|
| Width | 280px | 272px (`--sidebar-width`) |
| Background | `#f5f5f7` (flat) | `rgba(242,242,247,0.85)` + `backdrop-filter: blur(20px) saturate(180%)` (frosted glass) |
| Border | `border-gray-200` | `1px solid var(--color-separator)` |
| Nav item height | 44px (correct) | 44px (keep) |
| Nav item radius | `rounded-xl` | `rounded-[10px]` (`--radius-md`) |
| Nav item active | `bg-accent/10` with `motion.div layoutId` | `var(--color-accent-light)` with `nav-active-in` scale animation |
| Active text color | `text-accent` | `var(--color-accent)` font-weight 500 |
| Section headers | 11px uppercase | Keep, use `var(--color-text-tertiary)` |
| Footer items | 13px | Keep, add `var(--color-text-secondary)` |
| Labels section | Always visible | Collapsible with chevron toggle |
| Today badge | Blue count | Red badge (`var(--color-danger)`) to signal urgency |

### 2.2 Bottom Nav (`web/src/components/layout/BottomNav.tsx`)

| Property | Current | New |
|---|---|---|
| Background | `bg-white/95 backdrop-blur-sm` | `rgba(255,255,255,0.92)` + `backdrop-filter: blur(20px) saturate(180%)` |
| Height | 50px | 50px (keep) |
| Tabs | Inbox, Today, Upcoming, Done, More | Inbox, Today, Upcoming, Done, More (keep) |
| Tab icon size | 22px | 22px (keep) |
| Tab label | 10px | 10px (keep) |
| Today badge | Blue | Red (`--color-danger`) |
| Sync status | Pill above nav | Keep, update styling to match spec |

### 2.3 App Layout (`web/src/components/layout/AppLayout.tsx`)

| Property | Current | New |
|---|---|---|
| Mobile drawer | Basic slide-in | iOS-spring easing `cubic-bezier(0.32, 0.72, 0, 1)` 0.35s |
| Drawer overlay | `bg-black/30` | `rgba(0,0,0,0.35)` |
| FAB size | 56x56 | 56x56 (keep) |
| FAB shadow | `shadow-lg` | `var(--shadow-fab)` (accent-tinted) |
| FAB position (mobile) | `bottom-[calc(70px+env(safe-area))]` | `calc(50px + env(safe-area-inset-bottom) + 16px)` |
| Page transition | `motion.div` opacity 0.15s | Keep |
| Three-column layout | None | Add detail panel (380px) at >=1024px breakpoint |

### 2.4 Three-Column Layout (NEW)

At `>=1024px`, when a task is selected, show a detail panel on the right instead of a modal overlay:
- Panel width: 380px (440px at >=1440px)
- Border-left: `1px solid var(--color-separator)`
- Slide-in animation from right
- Task list remains visible and interactive on the left
- This replaces the centered modal `TaskDetail` on desktop only

---

## 3. Component Updates

### 3.1 Task Item (`web/src/components/tasks/TaskItem.tsx`)

**Reference wireframe:** `inbox.html`, `layout-shell.html`

Changes:
- **Priority bar**: 3px wide, left edge, rounded right side only, inset 12px top/bottom
- **Checkbox**: 22x22px circle, border `2px solid var(--color-text-quaternary)`, priority-aware border colors
- **Checkbox animation**: `checkbox-pop` 0.25s spring scale to 1.15, `check-draw` 0.2s stroke animation
- **Title**: 15px (`--text-subhead`), 2-line clamp with `-webkit-line-clamp: 2`
- **Completed state**: line-through, `var(--color-text-tertiary)`, row opacity 0.6
- **Metadata row**: 12px font, gap 8px, flex-wrap
- **Label pills**: `rgba(color, 0.12)` background, 11px font, `--radius-full`
- **Hover state (desktop)**: Reveal quick-action buttons (date, priority, list, delete) — 14px SVG icons
- **Drag handle**: Less prominent by default (opacity 0), visible on hover
- **Swipe gestures**: Keep existing, update colors to new palette

### 3.2 Task Detail — Desktop Panel (`web/src/components/tasks/TaskDetail.tsx`)

**Reference wireframe:** `task-detail.html` (Variant A)

At >=1024px, render as a side panel instead of modal:
- Width: 380px (440px at >=1440px)
- Slide in from right with spring animation
- Fixed header: title (editable) + close button
- Inline status bar: completion toggle + priority chip + due date + time
- Properties: list, recurrence, labels as pill/chip selectors
- Description: markdown editor with toolbar
- Subtasks: inline add, drag reorder, completion toggles
- Metadata footer: created date, last modified
- Action bar: Complete, Delete

### 3.3 Task Detail — Mobile Sheet (`web/src/components/tasks/TaskDetail.tsx`)

**Reference wireframe:** `task-detail.html` (Variant B)

At <768px, render as full-height bottom sheet:
- Drag handle at top
- Fixed header: Back + Title + Done
- iOS settings-style property rows (date, time, repeat, list, priority with chevrons)
- Fixed bottom bar: Complete + Delete
- Pickers open as additional bottom sheets

### 3.4 Quick Add (`web/src/components/tasks/QuickAdd.tsx`)

**Reference wireframe:** `quick-add.html`

Changes:
- **Inline (on pages)**: Always-visible input, not behind a button click. On focus, expand to show property chips (Date, Time, Repeat, List, Labels, Priority) as compact pill buttons with 14px SVG icons
- **Global modal (Cmd+N / FAB)**: Centered card with backdrop blur, spacious layout, keyboard shortcut hints
- **Mobile sheet (FAB)**: Bottom sheet, compact initial view (title + date + priority), "Show more" for full properties
- **Smart parsing hints**: Show subtle helper text for natural language input

### 3.5 Search Overlay (`web/src/components/common/SearchOverlay.tsx`)

**Reference wireframe:** `search.html`

Changes:
- **Command palette style**: Centered card (max 560px, max 70vh) instead of full-screen
- **Search input**: Magnifying glass icon + `Cmd+K` badge
- **Result categories**: Tasks, Lists, Labels with category headers
- **Task results**: Title, list name, due date, priority indicator
- **Recent searches**: Show when input is empty
- **Keyboard hints footer**: Up/Down Navigate, Enter Open, Esc Close
- **Mobile**: Full-screen with larger tap targets

### 3.6 Pickers (DatePicker, TimePicker, PriorityPicker, RecurrencePicker, ListSelect, LabelPicker)

**Reference wireframe:** `pickers.html`

All pickers:
- **Desktop**: Popovers with `--shadow-popover`, `--radius-lg`
- **Mobile**: Bottom sheets with drag handle
- **Consistent structure**: Header, options area, action area

Specific updates:
- **DatePicker**: Quick shortcuts row (Today, Tomorrow, Next Week, No Date) + mini calendar
- **TimePicker**: Grid of common times in 30-min increments + free-form input. Mobile: scroll wheel style
- **PriorityPicker**: Horizontal pills (None, Low/green, Medium/orange, High/red) with flag icons
- **RecurrencePicker**: Option list with icons (No repeat, Daily, Weekly, Monthly, Yearly)
- **ListSelect**: Dropdown with color swatches, search input, Inbox always at top
- **LabelPicker**: Multi-select with checkmarks, color dots, search, "Create new" action

---

## 4. Page Updates

### 4.1 Inbox Page (`web/src/pages/InboxPage.tsx`)

**Reference wireframe:** `inbox.html`

- Add filter/sort bar below header (sort by: date, priority, title; filter by: priority, labels)
- Quick-add always visible at top (not behind button)
- Empty state with illustration + "Inbox Zero!" message

### 4.2 Today Page (`web/src/pages/TodayPage.tsx`)

**Reference wireframe:** `today.html`

- Add progress indicator: "X of Y tasks done today" with animated SVG progress ring
- Overdue section: collapsible, red accent, count badge
- Optional "No date" section at bottom for unscheduled tasks

### 4.3 Upcoming Page (`web/src/pages/UpcomingPage.tsx`)

**Reference wireframe:** `upcoming.html`

- Tomorrow gets special treatment (larger header, accent dot)
- Vertical timeline line connecting day groups (desktop only)
- Per-day quick-add button on hover

### 4.4 Calendar Page (`web/src/pages/CalendarPage.tsx`)

**Reference wireframe:** `calendar.html`

- Add week view toggle (month/week tabs)
- Mobile: compact calendar with colored dots, scrollable task list below for selected day
- Desktop: full grid with task previews in cells
- Day click: opens detail popover with task list + add button
- Quick-add: tap day to add task with date pre-filled

### 4.5 Eisenhower Matrix (`web/src/pages/EisenhowerPage.tsx`)

**Reference wireframe:** `matrix.html`

- Desktop: true 2x2 grid with labeled axes (Important/Urgent)
- Color-themed quadrants: Do=red, Schedule=blue, Delegate=orange, Eliminate=gray
- Independent scroll per quadrant
- Mobile: swipeable pill tabs for each quadrant
- Summary pills at top showing counts

### 4.6 List Page (`web/src/pages/ListPage.tsx`)

**Reference wireframe:** `list-view.html`

- Color-accented header with list color bar
- Quick-add scoped to this list
- Filter/sort bar
- Drag-and-drop reordering with visible affordance

### 4.7 Label Page (`web/src/pages/LabelPage.tsx`)

**Reference wireframe:** `label-view.html`

- Label header with large color chip
- Tasks show which list they belong to (list badge on each item)
- Filter/sort options including list filter

### 4.8 Completed Page (`web/src/pages/CompletedPage.tsx`)

**Reference wireframe:** `completed.html`

- Stats summary card: "X tasks completed this month"
- Group by: Today, Yesterday, This Week, Earlier (Earlier collapsed)
- Tasks show completion timestamp
- Hover-reveal "Undo" per task
- Search/filter within completed

### 4.9 Trash Page (`web/src/pages/TrashPage.tsx`)

**Reference wireframe:** `trash.html`

- Warning banner: "Items permanently deleted after 30 days"
- Days-remaining badge per task (red when <7 days)
- Bulk actions: "Restore All" + "Empty Trash" with confirmation dialog
- Per-task restore/delete on hover or swipe

### 4.10 Login Page (`web/src/pages/LoginPage.tsx`)

**Reference wireframe:** `login.html`

- Glass-morphism centered card (backdrop blur)
- App icon + title
- Apple-style "Sign in with Google" button (dark, not Google's default)
- Background with subtle gradient
- Feature pills (Offline-first, Self-hosted, Safari PWA, CRDT sync)
- Mobile: PWA install hint

---

## 5. Implementation Order

Execute in this order. Each phase should be a separate PR.

### Phase 1: Foundation (tokens + layout shell)
1. Update `constants.ts` with new palette
2. Add CSS custom properties to global stylesheet
3. Extend Tailwind config with custom property mappings
4. Update `Sidebar.tsx` (frosted glass, collapsible labels, red Today badge)
5. Update `BottomNav.tsx` (frosted glass, updated styling)
6. Update `AppLayout.tsx` (iOS drawer animation, FAB shadow)
7. **Test**: All pages render correctly with new tokens. No visual regressions in existing functionality.

### Phase 2: Core components
1. Update `TaskItem.tsx` (priority bar, checkbox animation, hover actions, metadata layout)
2. Update `QuickAdd.tsx` (always-visible inline, property chips, smart parsing hints)
3. Update `SearchOverlay.tsx` (command palette style)
4. Update all pickers (desktop popovers, mobile bottom sheets)
5. **Test**: Task CRUD flows work. Pickers open/close correctly on both desktop and mobile.

### Phase 3: Task detail redesign
1. Implement desktop side panel variant for `TaskDetail.tsx`
2. Implement mobile bottom sheet variant
3. Wire three-column layout in `AppLayout.tsx` (>=1024px)
4. **Test**: Task detail opens correctly in both modes. Editing works. Transitions are smooth.

### Phase 4: Page-specific features
1. Inbox: filter/sort bar, empty state
2. Today: progress ring, collapsible sections
3. Upcoming: timeline, per-day add
4. Calendar: week view, mobile dot view, day detail popover
5. Matrix: 2x2 grid, mobile swipeable tabs
6. List/Label: color headers, scoped quick-add
7. Completed: stats card, time groups
8. Trash: days-remaining badges, bulk actions
9. Login: glass-morphism card redesign
10. **Test**: Each page matches wireframe. Run visual regression tests.

### Phase 5: Dark mode
1. Implement theme toggle (stored in IndexedDB user_config, respects system preference)
2. Apply `[data-theme]` attribute to document root
3. Verify all components render correctly in both modes
4. **Test**: Toggle between light/dark. System preference detection works.

---

## 6. Key Constraints (from CLAUDE.md)

- All reads via `useLiveQuery` from Dexie.js — no new state libraries
- Writes via `db/operations.ts` — never call API directly from components
- Safari/WebKit only — no Chromium-only APIs or CSS
- 44px minimum tap targets (Apple HIG)
- 16px minimum font on all text inputs (prevents iOS Safari zoom)
- `aria-label` on all icon-only buttons
- Tailwind CSS for styling
- Custom pickers (not native `<input type="date/time">`)
- Toast notifications for all user feedback

---

## 7. Wireframe Reference Index

| Wireframe | Implements | Key patterns to extract |
|---|---|---|
| `design-system.html` | Token reference | All CSS custom properties, component library |
| `layout-shell.html` | App shell, sidebar, bottom nav | Frosted glass, three-column layout, drawer animation |
| `inbox.html` | InboxPage | Filter/sort bar, always-visible quick-add, hover actions |
| `today.html` | TodayPage | Progress ring, collapsible overdue, no-date section |
| `upcoming.html` | UpcomingPage | Timeline, sticky headers, per-day add |
| `calendar.html` | CalendarPage | Week/month toggle, mobile dot view, day popover |
| `matrix.html` | EisenhowerPage | 2x2 grid, axis labels, mobile tabs |
| `list-view.html` | ListPage | Color header, drag reorder, scoped quick-add |
| `label-view.html` | LabelPage | Cross-list display, list badges |
| `completed.html` | CompletedPage | Stats card, time groups, undo |
| `trash.html` | TrashPage | Warning banner, days remaining, bulk actions |
| `login.html` | LoginPage | Glass-morphism, feature pills |
| `task-detail.html` | TaskDetail | Desktop panel + mobile sheet variants |
| `quick-add.html` | QuickAdd | Inline, modal, mobile sheet variants |
| `search.html` | SearchOverlay | Command palette, categories, keyboard hints |
| `pickers.html` | All pickers | Desktop popovers, mobile bottom sheets |
