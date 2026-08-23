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
  
  export type CreateTaskData = Omit<Task, 'id' | 'version' | 'created_at' | 'updated_at' | 'deleted_at'> & {
    due_date?: string | null;
  };
  
  export type UpdateTaskData = Partial<Omit<Task, 'id' | 'created_at' | 'updated_at' | 'deleted_at'>> & {
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