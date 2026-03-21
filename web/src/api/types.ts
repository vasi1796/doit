// Re-exports from generated OpenAPI types.
// Source of truth: api/openapi.yaml
// Regenerate: npm run generate:types
import type { components } from './types.gen'

// Priority is narrower than the generated `number` — kept as a branded type
// for frontend use. The spec enforces min:0 max:3.
export type Priority = 0 | 1 | 2 | 3

// Narrow the `priority` field from `number` to the stricter Priority union type.
export type Task = Omit<components['schemas']['Task'], 'priority'> & { priority: Priority }
export type Subtask = components['schemas']['Subtask']
export type Label = components['schemas']['Label']
export type List = components['schemas']['List']
export type CreateTaskRequest = Omit<components['schemas']['CreateTaskRequest'], 'priority'> & { priority: Priority }
export type UpdateTaskRequest = Omit<components['schemas']['UpdateTaskRequest'], 'priority'> & { priority?: Priority }
export type CreateListRequest = components['schemas']['CreateListRequest']
export type CreateLabelRequest = components['schemas']['CreateLabelRequest']
