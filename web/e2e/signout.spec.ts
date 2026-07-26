import { test, expect, type Page } from '@playwright/test'
import { mockApi } from './helpers/mock-api'

/** Rows in the local 'tasks' store; 0 when the database or store is absent. */
function countStoredTasks(page: Page): Promise<number> {
  return page.evaluate(async () => {
    const dbs = await indexedDB.databases()
    if (!dbs.some((d) => d.name === 'doit')) return 0
    return new Promise<number>((resolve, reject) => {
      const open = indexedDB.open('doit')
      open.onerror = () => reject(open.error)
      open.onsuccess = () => {
        const database = open.result
        if (!database.objectStoreNames.contains('tasks')) {
          database.close()
          resolve(0)
          return
        }
        const count = database.transaction('tasks').objectStore('tasks').count()
        count.onsuccess = () => {
          database.close()
          resolve(count.result)
        }
        count.onerror = () => {
          database.close()
          reject(count.error)
        }
      }
    })
  })
}

test.describe('Sign out', () => {
  test('wipes local data and lands on the login page', async ({ page, isMobile }) => {
    test.skip(isMobile, 'sidebar sign-out is the desktop entry point')

    await mockApi(page)
    await page.route('**/auth/logout', (route) => route.fulfill({ json: { status: 'logged out' } }))
    await page.goto('/today')
    await page.getByRole('link', { name: 'Project Alpha' }).waitFor({ state: 'visible' })

    // Seeded data must actually be present, or the post-wipe assertion is vacuous
    await expect.poll(() => countStoredTasks(page)).toBeGreaterThan(0)

    await page.getByRole('button', { name: 'Sign out' }).click()
    await page.waitForURL('**/login')

    // Assert emptiness, not absence: the root useTheme() live query reopens an
    // empty 'doit' database on the login page, so absence is a race.
    await expect.poll(() => countStoredTasks(page)).toBe(0)
  })
})
