import { useEffect } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'
import { setUserPreference } from '../db/operations'

export type Theme = 'light' | 'dark' | 'system'

const THEME_KEY = 'theme'
const DEFAULT_THEME: Theme = 'system'

/** Valid theme values — anything else is treated as 'system'. */
function parseTheme(value: string | undefined): Theme {
  return value === 'light' || value === 'dark' || value === 'system' ? value : DEFAULT_THEME
}

/**
 * Read/write the user's theme preference from IndexedDB.
 * Preferences are device-local and never sync to the server.
 */
export function useTheme() {
  const pref = useLiveQuery(() => db.userPreferences.get(THEME_KEY), [])
  const theme = parseTheme(pref?.value)

  const setTheme = (next: Theme) => setUserPreference(THEME_KEY, next)

  return { theme, setTheme }
}

/** Matches the bg tokens in index.css — used to keep Safari's URL-bar
 *  tint (meta[name="theme-color"]) in sync with the user's chosen theme. */
const THEME_COLOR_LIGHT = '#FFFFFF'
const THEME_COLOR_DARK = '#1C1C1E'

function setThemeColorMeta(color: string) {
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  if (meta) meta.content = color
}

/**
 * Applies the current theme to the document root by setting the
 * `data-theme` attribute. When the theme is 'system' the attribute is
 * removed so the `prefers-color-scheme` media query in index.css takes
 * over — and a matchMedia listener keeps Safari's `theme-color` meta in
 * sync with the OS as it flips.
 *
 * When the user picks an explicit theme we also pin the meta tag to
 * the matching bg color, so the iOS Safari URL-bar / status-bar area
 * follows the in-app choice even if it disagrees with the OS.
 *
 * Must be mounted once at the app root so every route picks up changes.
 */
export function useApplyTheme(theme: Theme) {
  useEffect(() => {
    const root = document.documentElement

    if (theme === 'dark') {
      root.setAttribute('data-theme', 'dark')
      setThemeColorMeta(THEME_COLOR_DARK)
      return
    }
    if (theme === 'light') {
      root.setAttribute('data-theme', 'light')
      setThemeColorMeta(THEME_COLOR_LIGHT)
      return
    }

    // theme === 'system': drop the attribute, follow the OS via media
    // query, and keep the Safari chrome tint live-updating with it.
    root.removeAttribute('data-theme')
    const mql = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = (dark: boolean) => setThemeColorMeta(dark ? THEME_COLOR_DARK : THEME_COLOR_LIGHT)
    apply(mql.matches)
    const handler = (e: MediaQueryListEvent) => apply(e.matches)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [theme])
}
