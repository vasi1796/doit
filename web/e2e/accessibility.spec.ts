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
})

test.describe('Accessibility — touch targets', () => {
  // Known undersized elements in the existing codebase.
  // Remove entries as they are fixed.
  const KNOWN_UNDERSIZED = [
    'button[Mark complete] is 22x22px',  // Checkbox visual is 22px, needs min-h/w-[44px]
  ]

  test('Interactive elements in main content meet 44px minimum', async ({ page }) => {
    await mockApi(page)
    await page.goto('/inbox')
    await waitForPage(page)

    // Only check elements inside <main> — sidebar has its own sizing rules
    const tooSmall = await page.evaluate(() => {
      const main = document.querySelector('main')
      if (!main) return ['<main> element not found']

      const interactiveElements = main.querySelectorAll('button, a, [role="button"], input, select, textarea')
      const violations: string[] = []

      interactiveElements.forEach((el) => {
        const rect = el.getBoundingClientRect()
        if (rect.width === 0 || rect.height === 0) return
        if (rect.width < 44 || rect.height < 44) {
          const tag = el.tagName.toLowerCase()
          const text = (el as HTMLElement).innerText?.slice(0, 30) || el.getAttribute('aria-label') || ''
          violations.push(`${tag}[${text}] is ${Math.round(rect.width)}x${Math.round(rect.height)}px`)
        }
      })

      return violations
    })

    // Filter out known issues to catch only new regressions
    const newViolations = tooSmall.filter((v) => !KNOWN_UNDERSIZED.includes(v))
    expect(newViolations, `New elements below 44px touch target:\n${newViolations.join('\n')}`).toEqual([])
  })
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
