import type { Page, Locator } from '@playwright/test'

/** Mouse-drag a row's drag handle onto another row, exceeding the 8px
 * activation distance before travelling so the PointerSensor engages. */
export async function dragByHandle(page: Page, handle: Locator, to: Locator) {
  const fromBox = await handle.boundingBox()
  const toBox = await to.boundingBox()
  if (!fromBox || !toBox) throw new Error('drag rows not visible')

  await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2)
  await page.mouse.down()
  await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2 + 12, { steps: 3 })
  await page.mouse.move(toBox.x + toBox.width / 2, toBox.y + toBox.height / 2 + 8, { steps: 10 })
  await page.mouse.up()
}
