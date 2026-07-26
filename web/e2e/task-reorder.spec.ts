import { test, expect, type Page } from '@playwright/test'
import { mockApi } from './helpers/mock-api'
import { dragByHandle } from './helpers/drag'

function taskRows(page: Page) {
  return page.locator('main div[role="button"]')
}

test.describe('Task drag-and-drop reorder', () => {
  test('dragging a task down one slot lands after the target and persists', async ({ page, isMobile }) => {
    test.skip(isMobile, 'mouse drag is the desktop entry point')

    await mockApi(page)
    await page.goto('/inbox')
    await page.getByText('Review pull request').waitFor({ state: 'visible' })

    await expect(taskRows(page).nth(0)).toContainText('Review pull request')
    await expect(taskRows(page).nth(1)).toContainText('Prepare quarterly review')
    await expect(taskRows(page).nth(2)).toContainText('Reply to vendor email')

    // The one-slot downward drag is the regression case: the old insertAt
    // arithmetic made it a silent no-op.
    await dragByHandle(
      page,
      page.locator('main').getByRole('button', { name: 'Drag to reorder', exact: true }).first(),
      taskRows(page).nth(1),
    )

    await expect(taskRows(page).nth(0)).toContainText('Prepare quarterly review')
    await expect(taskRows(page).nth(1)).toContainText('Review pull request')

    await page.reload()
    await page.getByText('Review pull request').waitFor({ state: 'visible' })
    await expect(taskRows(page).nth(0)).toContainText('Prepare quarterly review')
    await expect(taskRows(page).nth(1)).toContainText('Review pull request')
  })
})
