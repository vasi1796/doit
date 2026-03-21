import type {
  Task,
  Label,
  List,
  CreateTaskRequest,
  UpdateTaskRequest,
  CreateListRequest,
  CreateLabelRequest,
} from './types'

const BASE = '/api/v1'

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (res.status === 401) {
    window.location.href = '/login'
    throw new ApiError(401, 'Unauthorized')
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'Unknown error' }))
    throw new ApiError(res.status, body.error || 'Request failed')
  }

  if (res.status === 204) return undefined as T

  return res.json()
}

// Normalize due_date from "2026-03-25T00:00:00Z" to "2026-03-25"
function normalizeTasks(tasks: Task[]): Task[] {
  return tasks.map(normalizeTask)
}

function normalizeTask(task: Task): Task {
  return {
    ...task,
    due_date: task.due_date?.includes('T') ? task.due_date.split('T')[0] : task.due_date,
    due_time: task.due_time && task.due_time.length > 5 ? task.due_time.slice(0, 5) : task.due_time,
  }
}

export const api = {
  // Tasks
  async listTasks(params?: Record<string, string>): Promise<Task[]> {
    const qs = params ? '?' + new URLSearchParams(params).toString() : ''
    const tasks = await request<Task[]>(`/tasks${qs}`)
    return normalizeTasks(tasks)
  },

  async getTask(id: string): Promise<Task> {
    const task = await request<Task>(`/tasks/${id}`)
    return normalizeTask(task)
  },

  createTask(data: CreateTaskRequest): Promise<{ id: string }> {
    return request('/tasks', { method: 'POST', body: JSON.stringify(data) })
  },

  updateTask(id: string, data: UpdateTaskRequest): Promise<void> {
    return request(`/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(data) })
  },

  completeTask(id: string): Promise<void> {
    return request(`/tasks/${id}/complete`, { method: 'POST' })
  },

  uncompleteTask(id: string): Promise<void> {
    return request(`/tasks/${id}/uncomplete`, { method: 'POST' })
  },

  deleteTask(id: string): Promise<void> {
    return request(`/tasks/${id}`, { method: 'DELETE' })
  },

  addLabel(taskId: string, labelId: string): Promise<void> {
    return request(`/tasks/${taskId}/labels`, { method: 'POST', body: JSON.stringify({ label_id: labelId }) })
  },

  removeLabel(taskId: string, labelId: string): Promise<void> {
    return request(`/tasks/${taskId}/labels/${labelId}`, { method: 'DELETE' })
  },

  createSubtask(taskId: string, data: { title: string; position: string }): Promise<{ id: string }> {
    return request(`/tasks/${taskId}/subtasks`, { method: 'POST', body: JSON.stringify(data) })
  },

  completeSubtask(taskId: string, subtaskId: string): Promise<void> {
    return request(`/tasks/${taskId}/subtasks/${subtaskId}/complete`, { method: 'POST' })
  },

  uncompleteSubtask(taskId: string, subtaskId: string): Promise<void> {
    return request(`/tasks/${taskId}/subtasks/${subtaskId}/uncomplete`, { method: 'POST' })
  },

  updateSubtaskTitle(taskId: string, subtaskId: string, title: string): Promise<void> {
    return request(`/tasks/${taskId}/subtasks/${subtaskId}`, { method: 'PATCH', body: JSON.stringify({ title }) })
  },

  restoreTask(id: string): Promise<void> {
    return request(`/tasks/${id}/restore`, { method: 'POST' })
  },

  // Lists
  listLists(): Promise<List[]> {
    return request<List[]>('/lists')
  },

  createList(data: CreateListRequest): Promise<{ id: string }> {
    return request('/lists', { method: 'POST', body: JSON.stringify(data) })
  },

  deleteList(id: string): Promise<void> {
    return request(`/lists/${id}`, { method: 'DELETE' })
  },

  // Labels
  listLabels(): Promise<Label[]> {
    return request<Label[]>('/labels')
  },

  createLabel(data: CreateLabelRequest): Promise<{ id: string }> {
    return request('/labels', { method: 'POST', body: JSON.stringify(data) })
  },

  deleteLabel(id: string): Promise<void> {
    return request(`/labels/${id}`, { method: 'DELETE' })
  },
}
