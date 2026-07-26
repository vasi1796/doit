import { expect, type Page, type Locator } from '@playwright/test'

/** Mouse-drag a row's drag handle onto another row, exceeding the 8px
 * activation distance before travelling so the PointerSensor engages.
 * Each phase waits on dnd-kit's own live-region announcement rather than a
 * fixed delay — under CPU contention the pointer stream can otherwise outrun
 * the sensor and the drag silently becomes a no-op. */
export async function dragByHandle(page: Page, handle: Locator, to: Locator) {
  const fromBox = await handle.boundingBox()
  const toBox = await to.boundingBox()
  if (!fromBox || !toBox) throw new Error('drag rows not visible')

  const announcement = page.locator('[role="status"]').last()

  await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2)
  await page.mouse.down()
  await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2 + 12, { steps: 3 })

  await expect(announcement).toContainText('Draggable item')
  const beforeTravel = await announcement.textContent()

  await page.mouse.move(toBox.x + toBox.width / 2, toBox.y + toBox.height / 2 + 8, { steps: 10 })

  // Collision detection has re-run over the destination
  await expect.poll(() => announcement.textContent()).not.toBe(beforeTravel)

  await page.mouse.up()
}
