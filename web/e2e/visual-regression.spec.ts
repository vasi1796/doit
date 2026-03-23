import { test, expect } from '@playwright/test'
import { mockApi, mockApiEmpty } from './helpers/mock-api'

async function waitForPage(page: import('@playwright/test').Page) {
  await page.locator('main h1').first().waitFor({ state: 'visible', timeout: 10_000 })
  await page.waitForTimeout(300)
}

test.describe('Visual regression — pages with data', () => {
  test.beforeEach(async ({ page }) => {
    await mockApi(page)
  })

  test('Login page', async ({ page }) => {
    await page.goto('/login')
    await page.locator('h1').waitFor({ state: 'visible' })
    await expect(page).toHaveScreenshot('login.png')
  })

  test('Inbox page', async ({ page }) => {
    await page.goto('/inbox')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('inbox.png')
  })

  test('Today page', async ({ page }) => {
    await page.goto('/today')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('today.png')
  })

  test('Upcoming page', async ({ page }) => {
    await page.goto('/upcoming')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('upcoming.png')
  })

  test('Completed page', async ({ page }) => {
    await page.goto('/completed')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('completed.png')
  })

  test('Trash page', async ({ page }) => {
    await page.goto('/trash')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('trash.png')
  })

  test('List page', async ({ page }) => {
    await page.goto('/lists/list-1')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('list-page.png')
  })

  test('Label page', async ({ page }) => {
    await page.goto('/labels/label-1')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('label-page.png')
  })

  test('Eisenhower Matrix page', async ({ page }) => {
    await page.goto('/matrix')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('matrix.png')
  })

  test('Calendar page', async ({ page }) => {
    await page.goto('/calendar')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('calendar.png')
  })
})

test.describe('Visual regression — interactive components', () => {
  test.beforeEach(async ({ page }) => {
    await mockApi(page)
  })

  test('QuickAdd expanded', async ({ page }) => {
    await page.goto('/inbox')
    await waitForPage(page)

    // Click the "New task..." button to expand QuickAdd
    const newTaskBtn = page.getByText('New task...')
    await newTaskBtn.waitFor({ state: 'visible', timeout: 15_000 })
    await newTaskBtn.click()
    // Wait for the expanded form to render
    await page.getByPlaceholder('Task name').waitFor({ state: 'visible', timeout: 10_000 })
    await page.waitForTimeout(300)

    await expect(page).toHaveScreenshot('quickadd-expanded.png')
  })

})

test.describe('Visual regression — empty states', () => {
  test.beforeEach(async ({ page }) => {
    await mockApiEmpty(page)
  })

  test('Inbox empty state', async ({ page }) => {
    await page.goto('/inbox')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('inbox-empty.png')
  })

  test('Today empty state', async ({ page }) => {
    await page.goto('/today')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('today-empty.png')
  })

  test('Trash empty state', async ({ page }) => {
    await page.goto('/trash')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('trash-empty.png')
  })

  test('Matrix empty state', async ({ page }) => {
    await page.goto('/matrix')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('matrix-empty.png')
  })

  test('Calendar empty state', async ({ page }) => {
    await page.goto('/calendar')
    await waitForPage(page)
    await expect(page).toHaveScreenshot('calendar-empty.png')
  })
})
