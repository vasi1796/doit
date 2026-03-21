import type { Page } from '@playwright/test'
import {
  MOCK_TASKS,
  MOCK_COMPLETED_TASKS,
  MOCK_DELETED_TASKS,
  MOCK_LISTS,
  MOCK_LABELS,
} from '../fixtures/mock-data'

/**
 * Intercept all /api/v1/* calls and return mock data.
 * Call this before navigating to any authenticated page.
 */
export async function mockApi(page: Page) {
  // Use ** to match across path separators (single task: /tasks/{id})
  await page.route('**/api/v1/tasks**', (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname

    // Single task GET: /api/v1/tasks/{id}
    const singleTaskMatch = path.match(/\/api\/v1\/tasks\/([^/]+)$/)
    if (singleTaskMatch) {
      const id = singleTaskMatch[1]
      const allTasks = [...MOCK_TASKS, ...MOCK_COMPLETED_TASKS, ...MOCK_DELETED_TASKS]
      const task = allTasks.find((t) => t.id === id)
      if (task) {
        return route.fulfill({ json: task })
      }
      return route.fulfill({ status: 404, json: { error: 'task not found' } })
    }

    // List tasks: /api/v1/tasks?params
    const params = url.searchParams
    let tasks = [...MOCK_TASKS]

    if (params.get('is_completed') === 'true') {
      tasks = [...MOCK_COMPLETED_TASKS]
    } else if (params.get('is_deleted') === 'true') {
      tasks = [...MOCK_DELETED_TASKS]
    } else {
      if (params.get('inbox') === 'true') {
        tasks = tasks.filter((t) => !t.list_id)
      }
      if (params.get('list_id')) {
        tasks = tasks.filter((t) => t.list_id === params.get('list_id'))
      }
      if (params.get('label_id')) {
        const labelId = params.get('label_id')
        tasks = tasks.filter((t) => t.labels?.some((l) => l.id === labelId))
      }
    }

    return route.fulfill({ json: tasks })
  })

  await page.route('**/api/v1/lists', (route) => {
    return route.fulfill({ json: MOCK_LISTS })
  })

  await page.route('**/api/v1/labels', (route) => {
    return route.fulfill({ json: MOCK_LABELS })
  })
}

/**
 * Mock API to return empty data for testing empty states.
 */
export async function mockApiEmpty(page: Page) {
  await page.route('**/api/v1/tasks**', (route) => {
    return route.fulfill({ json: [] })
  })

  await page.route('**/api/v1/lists', (route) => {
    return route.fulfill({ json: [] })
  })

  await page.route('**/api/v1/labels', (route) => {
    return route.fulfill({ json: [] })
  })
}
