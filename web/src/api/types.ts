export interface Task {
  id: string
  list_id?: string
  title: string
  description?: string
  priority: number
  due_date?: string
  due_time?: string
  position: string
  is_completed: boolean
  completed_at?: string
  recurrence_rule?: string
  is_deleted: boolean
  created_at: string
  updated_at: string
  subtasks?: Subtask[]
  labels?: Label[]
}

export interface Subtask {
  id: string
  title: string
  is_completed: boolean
  position: string
}

export interface Label {
  id: string
  name: string
  colour?: string
  created_at?: string
}

export interface List {
  id: string
  name: string
  colour?: string
  icon?: string
  position: string
  created_at: string
  updated_at: string
}

export interface CreateTaskRequest {
  title: string
  description?: string
  priority: number
  due_date?: string
  due_time?: string
  list_id?: string
  position: string
}

export interface UpdateTaskRequest {
  title?: string
  description?: string
  priority?: number
  due_date?: string
  due_time?: string
  recurrence_rule?: string
  list_id?: string
  position?: string
}

export interface CreateListRequest {
  name: string
  colour: string
  icon?: string
  position: string
}

export interface CreateLabelRequest {
  name: string
  colour: string
}
