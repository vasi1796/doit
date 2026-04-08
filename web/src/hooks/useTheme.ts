import { useEffect } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'

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

  const setTheme = async (next: Theme) => {
    await db.userPreferences.put({ key: THEME_KEY, value: next })
  }

  return { theme, setTheme }
}

/**
 * Applies the current theme to the document root by setting the
 * `data-theme` attribute. When the theme is 'system' the attribute is
 * removed so the `prefers-color-scheme` media query in index.css takes
 * over.
 *
 * Must be mounted once at the app root (AppLayout) so every page picks
 * up theme changes live.
 */
export function useApplyTheme(theme: Theme) {
  useEffect(() => {
    const root = document.documentElement
    if (theme === 'system') {
      root.removeAttribute('data-theme')
    } else {
      root.setAttribute('data-theme', theme)
    }
  }, [theme])
}
