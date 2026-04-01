---
paths:
  - "web/**"
---

# React Frontend Constraints

## Data Flow (non-negotiable)
- All UI reads come from IndexedDB via `useLiveQuery` from Dexie.js
- Components must never call `api.*` directly — use `db/operations.ts` for writes
- No Redux, Zustand, Jotai, or other state libraries — Dexie.js is the sole
  state layer
- React Context is used only for layout-level computed data (task counts,
  quick-add ref)

## Offline-First Writes
- `db/operations.ts` writes optimistically to IndexedDB and queues a `SyncOp`
- The sync engine flushes the queue to `POST /api/v1/sync` periodically
- No rollback on failure — retry with exponential backoff (max 5 retries)

## Safari/WebKit Only
- No Chromium-only APIs (Background Sync, Web Bluetooth, Web USB)
- No native `<input type="date/time">` — use custom pickers (Safari PWA compat)
- All CSS must have WebKit support
- Test in Safari, not Chrome

## Accessibility & Touch
- Minimum 44x44px touch targets (Apple HIG)
- All text inputs >= 16px font size (prevents iOS Safari auto-zoom)
- `aria-label` on all icon-only buttons and placeholder-only inputs
- Run `npm run lint` (includes jsx-a11y) after changes

## Styling
- Tailwind CSS only — no CSS modules, styled-components, or inline styles
- Shared color constants in `constants.ts`
- Use fixed-position popovers for pickers, not native `<select>`

## Notifications
- Toast notifications for all user-facing feedback
- Toasts support action buttons (e.g., "Undo") for destructive operations
