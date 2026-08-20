import axios from 'axios';
import { Task, CreateTaskData, UpdateTaskData, TaskFilter, ListResponse } from '../types';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
});

export const getTasks = (params?: TaskFilter) =>
  api.get<ListResponse>('/tasks', { params });

export const createTask = (data: CreateTaskData) =>
  api.post<Task>('/tasks', data);

export const updateTask = (id: string, data: UpdateTaskData) =>
  api.put<Task>(`/tasks/${id}`, data);

export const deleteTask = (id: string) =>
  api.delete(`/tasks/${id}`);
