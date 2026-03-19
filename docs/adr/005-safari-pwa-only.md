# ADR-005: Safari PWA Instead of Native Apps

**Status:** Accepted

## Context

DoIt targets the Apple ecosystem exclusively — macOS, iOS, and iPadOS. It is
used by 1-3 users (personal/family). We need to decide between:

1. **Native apps**: Swift/SwiftUI for iOS/iPadOS and macOS.
2. **Cross-platform native**: React Native or Flutter.
3. **Progressive Web App (PWA)**: Web application with offline support via
   service workers.

Considerations:
- The user base is 1-3 people. App Store distribution is unnecessary overhead.
- All target devices run Safari/WebKit.
- The application needs offline support for task management.
- Development is done by a solo developer as a learning project.
- Maintaining native code for multiple platforms increases scope significantly.

## Decision

We will build a **Progressive Web App** targeting **Safari on macOS, iOS, and
iPadOS**.

- The frontend is a React SPA served as a PWA with a web app manifest.
- **Workbox** manages the service worker for asset caching and offline support.
- Users add the app to their home screen via Safari's "Add to Home Screen".
- All UI follows Apple Human Interface Guidelines (44px tap targets, safe area
  insets, system font preferences).
- Only Safari/WebKit APIs are targeted. No Chromium-only features.

## Consequences

**Benefits:**
- **Single codebase** for all three platforms (macOS, iOS, iPadOS).
- **No App Store** — no review process, no certificates, no provisioning profiles.
- **Instant updates** — deploy to the server, service worker picks up changes.
- **Lower complexity** — web technologies only, no Swift/native build tooling.
- **Reuses existing skills** — React/TypeScript knowledge applies directly.

**Costs:**
- **No Background Sync API** in Safari — sync can only happen when the app is
  in the foreground with network access. The sync engine must be resilient to
  interrupted sync operations.
- **Storage eviction risk** — Safari may evict IndexedDB data under storage
  pressure (especially on iOS). The app must handle graceful re-sync from the
  server when local data is lost.
- **Push notifications are best-effort** — Safari push notifications have
  limitations compared to native. They require user opt-in and may not be as
  reliable.
- **No access to native APIs** — no Siri integration, no widgets, no Apple Watch
  complications. Acceptable for a task management app.
- **Safari PWA quirks** — Safari's PWA support has historically lagged behind
  Chrome. Some features (e.g., standalone display mode behavior) may have subtle
  differences across iOS versions.
