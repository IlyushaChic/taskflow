import React, { useState, useEffect, useCallback } from 'react';
import { Button, message, notification } from 'antd';
import { getTasks, createTask, updateTask, deleteTask } from './api/tasks';
import { Task, CreateTaskData, UpdateTaskData, TaskFilter } from './types/task';
import { useWebSocket } from './hooks/useWebSocket';
import TaskFilters from './components/TaskFilters';
import TaskTable from './components/TaskTable';
import TaskModal from './components/TaskModal';

const App: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFilters] = useState<TaskFilter>({ status: '', assignee: '' });
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);

  const fetchTasks = useCallback(
    async (params: TaskFilter = {}) => {
      setLoading(true);
      try {
        const res = await getTasks({ ...filters, ...params });
        setTasks(res.data.data);
      } catch (error) {
        message.error('Не удалось загрузить задачи');
      } finally {
        setLoading(false);
      }
    },
    [filters]
  );

  // Загрузка при изменении фильтров
  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  // WebSocket
  useWebSocket('ws://localhost:8080/ws', (data) => {
    notification.info({
      message: `Событие: ${data.type}`,
      description:
        data.type === 'task_deleted'
          ? `Задача ${data.data.id} удалена`
          : `Задача "${data.data.title}" обновлена`,
    });
    fetchTasks();
  });

  const handleCreate = () => {
    setEditingTask(null);
    setModalVisible(true);
  };

  const handleEdit = (task: Task) => {
    setEditingTask(task);
    setModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteTask(id);
      message.success('Задача удалена');
      fetchTasks();
    } catch (error) {
      message.error('Ошибка удаления');
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (editingTask) {
        const payload: UpdateTaskData = {
          ...values,
          version: editingTask.version,
          due_date: values.due_date ? values.due_date.toISOString() : null,
        };
        await updateTask(editingTask.id, payload);
        message.success('Задача обновлена');
      } else {
        const payload: CreateTaskData = {
          ...values,
          due_date: values.due_date ? values.due_date.toISOString() : null,
        };
        await createTask(payload);
        message.success('Задача создана');
      }
      setModalVisible(false);
      fetchTasks();
    } catch (error) {
      message.error('Операция не удалась');
    }
  };

  const handleCancel = () => {
    setModalVisible(false);
  };

  return (
    <div style={{ padding: 24 }}>
      <h1>TaskFlow</h1>
      <Button type="primary" onClick={handleCreate} style={{ marginBottom: 16 }}>
        Создать задачу
      </Button>

      <TaskFilters filters={filters} setFilters={setFilters} />

      <TaskTable
        tasks={tasks}
        loading={loading}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />

      <TaskModal
        open={modalVisible}
        editingTask={editingTask}
        onFinish={handleSubmit}
        onCancel={handleCancel}
      />
    </div>
  );
};

export default App;