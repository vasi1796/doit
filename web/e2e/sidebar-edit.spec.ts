import { test, expect } from '@playwright/test'
import { mockApi } from './helpers/mock-api'

test.describe('Sidebar list/label context menu', () => {
  test('right-click opens menu and rename reflects in the sidebar', async ({ page, isMobile }) => {
    test.skip(isMobile, 'right-click is the desktop entry point')

    await mockApi(page)
    await page.goto('/today')
    const listLink = page.getByRole('link', { name: 'Project Alpha' })
    await listLink.waitFor({ state: 'visible' })

    await listLink.click({ button: 'right' })
    const menu = page.getByRole('menu')
    await expect(menu).toBeVisible()

    await menu.getByRole('menuitem', { name: 'Edit' }).click()
    const dialog = page.getByRole('dialog', { name: 'Edit List' })
    await expect(dialog).toBeVisible()

    const nameInput = dialog.getByRole('textbox', { name: 'Name' })
    await nameInput.fill('Project Beta')
    await dialog.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByRole('link', { name: 'Project Beta' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Project Alpha' })).toHaveCount(0)
  })

  test('hover options button opens menu with Edit and Delete', async ({ page, isMobile }) => {
    test.skip(isMobile, 'hover affordance is desktop-only')

    await mockApi(page)
    await page.goto('/today')
    const listLink = page.getByRole('link', { name: 'Project Alpha' })
    await listLink.waitFor({ state: 'visible' })

    await listLink.hover()
    await page.getByRole('button', { name: 'Project Alpha list options' }).click()

    const menu = page.getByRole('menu')
    await expect(menu.getByRole('menuitem', { name: 'Edit' })).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: 'Delete' })).toBeVisible()
  })

  test('touch long-press opens the context menu', async ({ page, isMobile }) => {
    test.skip(!isMobile, 'long-press is the touch entry point')

    await mockApi(page)
    await page.goto('/today')

    // Sidebar lives in the mobile drawer — open it via the More tab
    await page.getByRole('button', { name: 'More' }).click()
    const listLink = page.getByRole('link', { name: 'Project Alpha' })
    await listLink.waitFor({ state: 'visible' })

    // Synthesise a touch long-press: pointerdown with pointerType 'touch',
    // held past the 500ms threshold with no movement.
    await listLink.dispatchEvent('pointerdown', {
      pointerType: 'touch',
      clientX: 100,
      clientY: 300,
      bubbles: true,
    })
    const menu = page.getByRole('menu')
    await expect(menu).toBeVisible({ timeout: 5000 })

    // Release the finger: Safari fires a synthetic click at the release
    // coordinates, which now hit the menu backdrop — the menu must survive it.
    await listLink.dispatchEvent('pointerup', { pointerType: 'touch', bubbles: true })
    await page.evaluate(() => {
      document.elementFromPoint(100, 300)?.dispatchEvent(
        new MouseEvent('click', { bubbles: true, cancelable: true, clientX: 100, clientY: 300 })
      )
    })
    await expect(menu).toBeVisible()
    await menu.getByRole('menuitem', { name: 'Edit' }).click()
    await expect(page.getByRole('dialog', { name: 'Edit List' })).toBeVisible()
  })
})
