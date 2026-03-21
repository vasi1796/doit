import type { Task, Label, List } from '../../src/api/types'

export const MOCK_LABELS: Label[] = [
  { id: 'label-1', name: 'Work', colour: '#007aff' },
  { id: 'label-2', name: 'Personal', colour: '#4cd964' },
]

export const MOCK_LISTS: List[] = [
  {
    id: 'list-1',
    name: 'Project Alpha',
    colour: '#5856d6',
    position: '1',
    created_at: '2026-03-01T00:00:00Z',
    updated_at: '2026-03-01T00:00:00Z',
  },
  {
    id: 'list-2',
    name: 'Groceries',
    colour: '#ff9500',
    position: '2',
    created_at: '2026-03-02T00:00:00Z',
    updated_at: '2026-03-02T00:00:00Z',
  },
]

const today = new Date().toISOString().split('T')[0]
const tomorrow = new Date(Date.now() + 86_400_000).toISOString().split('T')[0]
const yesterday = new Date(Date.now() - 86_400_000).toISOString().split('T')[0]
const inThreeDays = new Date(Date.now() + 3 * 86_400_000).toISOString().split('T')[0]

export const MOCK_TASKS: Task[] = [
  {
    id: 'task-1',
    title: 'Review pull request',
    priority: 2,
    due_date: today,
    position: '1',
    is_completed: false,
    is_deleted: false,
    created_at: '2026-03-10T00:00:00Z',
    updated_at: '2026-03-10T00:00:00Z',
    labels: [MOCK_LABELS[0]],
    subtasks: [
      { id: 'sub-1', title: 'Check tests', is_completed: true, position: '1' },
      { id: 'sub-2', title: 'Review diff', is_completed: false, position: '2' },
    ],
  },
  {
    id: 'task-2',
    title: 'Buy groceries for dinner',
    priority: 1,
    due_date: tomorrow,
    list_id: 'list-2',
    position: '2',
    is_completed: false,
    is_deleted: false,
    created_at: '2026-03-11T00:00:00Z',
    updated_at: '2026-03-11T00:00:00Z',
    labels: [MOCK_LABELS[1]],
  },
  {
    id: 'task-3',
    title: 'Overdue report — finish ASAP',
    priority: 3,
    due_date: yesterday,
    list_id: 'list-1',
    position: '3',
    is_completed: false,
    is_deleted: false,
    created_at: '2026-03-08T00:00:00Z',
    updated_at: '2026-03-08T00:00:00Z',
  },
  {
    id: 'task-4',
    title: 'Plan team offsite',
    priority: 0,
    due_date: inThreeDays,
    list_id: 'list-1',
    position: '4',
    is_completed: false,
    is_deleted: false,
    created_at: '2026-03-12T00:00:00Z',
    updated_at: '2026-03-12T00:00:00Z',
    recurrence_rule: 'FREQ=WEEKLY',
  },
]

export const MOCK_COMPLETED_TASKS: Task[] = [
  {
    id: 'task-5',
    title: 'Write design document',
    priority: 2,
    position: '5',
    is_completed: true,
    completed_at: '2026-03-15T10:00:00Z',
    is_deleted: false,
    created_at: '2026-03-05T00:00:00Z',
    updated_at: '2026-03-15T10:00:00Z',
    labels: [MOCK_LABELS[0]],
  },
]

export const MOCK_DELETED_TASKS: Task[] = [
  {
    id: 'task-6',
    title: 'Old task to clean up',
    priority: 0,
    position: '6',
    is_completed: false,
    is_deleted: true,
    created_at: '2026-03-01T00:00:00Z',
    updated_at: '2026-03-14T00:00:00Z',
  },
]
