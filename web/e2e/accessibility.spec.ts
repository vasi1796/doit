import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { mockApi } from './helpers/mock-api'

// Known a11y violations in the existing codebase.
// Remove rules from this list as they are fixed.
const KNOWN_VIOLATIONS = [
  'button-name',           // Some icon-only buttons lack aria-label
  'color-contrast',        // Some text/background combos below AA ratio
  'landmark-one-main',     // Login page has no <main> landmark
  'landmark-unique',       // Duplicate nav landmarks in layout
  'nested-interactive',    // Nested interactive elements in task items
  'region',                // Content outside landmark regions
  'scrollable-region-focusable', // Calendar grid scroll — tabIndex conflicts with jsx-a11y linter
]

async function waitForPage(page: import('@playwright/test').Page) {
  // Wait for the page content h1 inside <main>, not the sidebar "DoIt" h1
  await page.locator('main h1').first().waitFor({ state: 'visible', timeout: 10_000 })
  await page.waitForTimeout(300)
}

function a11yTest(name: string, path: string) {
  test(name, async ({ page }) => {
    await mockApi(page)
    await page.goto(path)
    await waitForPage(page)

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'best-practice'])
      .disableRules(KNOWN_VIOLATIONS)
      .analyze()

    const violations = results.violations.map((v) => ({
      id: v.id,
      impact: v.impact,
      description: v.description,
      nodes: v.nodes.length,
    }))

    expect(violations, `a11y violations on ${path}:\n${JSON.stringify(violations, null, 2)}`).toEqual([])
  })
}

test.describe('Accessibility — all pages', () => {
  test('Login page', async ({ page }) => {
    await page.goto('/login')
    await page.locator('h1').waitFor({ state: 'visible' })

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'best-practice'])
      .disableRules(KNOWN_VIOLATIONS)
      .analyze()

    const violations = results.violations.map((v) => ({
      id: v.id,
      impact: v.impact,
      description: v.description,
      nodes: v.nodes.length,
    }))

    expect(violations, `a11y violations on /login:\n${JSON.stringify(violations, null, 2)}`).toEqual([])
  })

  a11yTest('Inbox page', '/inbox')
  a11yTest('Today page', '/today')
  a11yTest('Upcoming page', '/upcoming')
  a11yTest('Completed page', '/completed')
  a11yTest('Trash page', '/trash')
  a11yTest('List page', '/lists/list-1')
  a11yTest('Label page', '/labels/label-1')
  a11yTest('Matrix page', '/matrix')
  a11yTest('Calendar page', '/calendar')
})

test.describe('Accessibility — font sizes', () => {
  test('Text inputs are at least 16px to prevent iOS auto-zoom', async ({ page }) => {
    await mockApi(page)
    await page.goto('/inbox')
    await waitForPage(page)

    const tooSmall = await page.evaluate(() => {
      const inputs = document.querySelectorAll('input, textarea, select')
      const violations: string[] = []

      inputs.forEach((el) => {
        const style = window.getComputedStyle(el)
        const fontSize = parseFloat(style.fontSize)
        if (fontSize < 16) {
          const name = el.getAttribute('name') || el.getAttribute('placeholder') || el.tagName
          violations.push(`${name}: ${fontSize}px`)
        }
      })

      return violations
    })

    expect(tooSmall, `Inputs below 16px font-size:\n${tooSmall.join('\n')}`).toEqual([])
  })
})
