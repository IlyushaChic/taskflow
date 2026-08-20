export interface Task {
  id: string;
  title: string;
  description: string | null;
  status: 'new' | 'in_progress' | 'done' | 'cancelled';
  assignee: string | null;
  due_date: string | null;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export type CreateTaskData = {
  title: string;
  description?: string | null;
  assignee?: string | null;
  due_date?: string | null;
};

export type UpdateTaskData = {
  title?: string | null;
  description?: string | null;
  status?: Task['status'];
  assignee?: string | null;
  due_date?: string | null;
  version: number;
};

export interface TaskFilter {
  status?: string;
  assignee?: string;
  due_date_from?: string;
  due_date_to?: string;
  offset?: number;
  limit?: number;
}

export interface ListResponse {
  data: Task[];
  total: number;
}