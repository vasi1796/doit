import { test, expect } from '@playwright/test'
import { mockApi } from './helpers/mock-api'
import { dragByHandle } from './helpers/drag'

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
