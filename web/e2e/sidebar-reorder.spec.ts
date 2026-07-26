import { test, expect, type Page, type Locator } from '@playwright/test'
import { mockApi } from './helpers/mock-api'

/** Mouse-drag a row's drag handle onto another row, exceeding the 8px
 * activation distance before travelling so the PointerSensor engages. */
async function dragByHandle(page: Page, handle: Locator, to: Locator) {
  const fromBox = await handle.boundingBox()
  const toBox = await to.boundingBox()
  if (!fromBox || !toBox) throw new Error('drag rows not visible')

  await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2)
  await page.mouse.down()
  await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2 + 12, { steps: 3 })
  await page.mouse.move(toBox.x + toBox.width / 2, toBox.y + toBox.height / 2 + 8, { steps: 10 })
  await page.mouse.up()
}

test.describe('Sidebar drag-and-drop reorder', () => {
  test('dragging a list below another persists across reload', async ({ page, isMobile }) => {
    test.skip(isMobile, 'mouse drag is the desktop entry point')

    await mockApi(page)
    await page.goto('/today')

    const listLinks = page.locator('aside a[href^="/lists/"]')
    await expect(listLinks.first()).toBeVisible()
    await expect(listLinks.nth(0)).toHaveAttribute('href', '/lists/list-1')
    await expect(listLinks.nth(1)).toHaveAttribute('href', '/lists/list-2')

    await dragByHandle(
      page,
      page.getByRole('button', { name: 'Drag to reorder Project Alpha' }),
      page.getByRole('link', { name: 'Groceries' }),
    )

    await expect(listLinks.nth(0)).toHaveAttribute('href', '/lists/list-2')
    await expect(listLinks.nth(1)).toHaveAttribute('href', '/lists/list-1')

    await page.reload()
    await expect(listLinks.first()).toBeVisible()
    await expect(listLinks.nth(0)).toHaveAttribute('href', '/lists/list-2')
    await expect(listLinks.nth(1)).toHaveAttribute('href', '/lists/list-1')
  })

  test('dragging a label below another persists across reload', async ({ page, isMobile }) => {
    test.skip(isMobile, 'mouse drag is the desktop entry point')

    await mockApi(page)
    await page.goto('/today')

    const labelLinks = page.locator('aside a[href^="/labels/"]')
    await expect(labelLinks.first()).toBeVisible()
    await expect(labelLinks.nth(0)).toHaveAttribute('href', '/labels/label-1')
    await expect(labelLinks.nth(1)).toHaveAttribute('href', '/labels/label-2')

    await dragByHandle(
      page,
      page.getByRole('button', { name: 'Drag to reorder Work' }),
      page.getByRole('link', { name: 'Personal' }),
    )

    await expect(labelLinks.nth(0)).toHaveAttribute('href', '/labels/label-2')
    await expect(labelLinks.nth(1)).toHaveAttribute('href', '/labels/label-1')

    await page.reload()
    await expect(labelLinks.first()).toBeVisible()
    await expect(labelLinks.nth(0)).toHaveAttribute('href', '/labels/label-2')
    await expect(labelLinks.nth(1)).toHaveAttribute('href', '/labels/label-1')
  })
})
