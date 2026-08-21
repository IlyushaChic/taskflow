import React, { useEffect, useRef, useState } from 'react';
import { Table, Button, Modal, Form, Input, Select, DatePicker, Space, message } from 'antd';
import { getTasks, createTask, updateTask, deleteTask } from './api/tasks';
import { Task, CreateTaskData, UpdateTaskData, TaskFilter } from './types';
import * as Sentry from '@sentry/react';
import dayjs from 'dayjs';

const { Option } = Select;
import { notification } from 'antd';

const App: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [form] = Form.useForm();
  const [filters, setFilters] = useState<TaskFilter>({ status: '', assignee: '' });
  const wsRef = useRef<WebSocket | null>(null);


  const fetchTasks = async (params: TaskFilter = {}) => {
    setLoading(true);
    try {
      const res = await getTasks({ ...filters, ...params });
      setTasks(res.data.data);
    } catch (error) {
      message.error('Не удалось загрузить задачи');
      Sentry.captureException(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTasks();
  }, [filters]);


useEffect(() => {
  const ws = new WebSocket('ws://localhost:8080/ws');
  wsRef.current = ws;

  ws.onopen = () => {
    console.log('WebSocket connected');
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      // data: { type: 'task_created' | 'task_updated' | 'task_deleted', data: ... }
      notification.info({
        message: `Событие: ${data.type}`,
        description: data.type === 'task_deleted' 
          ? `Задача ${data.data.id} удалена` 
          : `Задача "${data.data.title}" обновлена`,
      });

      fetchTasks();
    } catch (err) {
      console.error('WebSocket message error', err);
    }
  };

  ws.onclose = () => {
    console.log('WebSocket disconnected');
  };

  return () => {
    ws.close();
  };
}, []);

  const handleCreate = () => {
    setEditingTask(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: Task) => {
    setEditingTask(record);
    form.setFieldsValue({
      ...record,
      due_date: record.due_date ? dayjs(record.due_date) : null,
    });
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

  const columns = [
    { title: 'Название', dataIndex: 'title', key: 'title' },
    { title: 'Статус', dataIndex: 'status', key: 'status' },
    { title: 'Исполнитель', dataIndex: 'assignee', key: 'assignee' },
    {
      title: 'Срок',
      dataIndex: 'due_date',
      key: 'due_date',
      render: (val: string | null) => (val ? new Date(val).toLocaleDateString() : '-'),
    },
    {
      title: 'Действия',
      key: 'action',
      render: (_: any, record: Task) => (
        <Space>
          <Button onClick={() => handleEdit(record)}>Редактировать</Button>
          <Button danger onClick={() => handleDelete(record.id)}>Удалить</Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <h1>TaskFlow</h1>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={handleCreate}>Создать задачу</Button>
        <Input
          placeholder="Фильтр по исполнителю"
          onChange={(e) => setFilters({ ...filters, assignee: e.target.value })}
        />
        <Select
          placeholder="Статус"
          style={{ width: 120 }}
          onChange={(val) => setFilters({ ...filters, status: val as string })}
          allowClear
        >
          <Option value="new">Новая</Option>
          <Option value="in_progress">В работе</Option>
          <Option value="done">Выполнена</Option>
          <Option value="cancelled">Отменена</Option>
        </Select>
      </Space>

      <Table
        dataSource={tasks}
        columns={columns}
        loading={loading}
        rowKey="id"
      />

      <Modal
        title={editingTask ? 'Редактировать задачу' : 'Создать задачу'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="title" label="Название" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Описание">
            <Input.TextArea />
          </Form.Item>
          <Form.Item name="status" label="Статус">
            <Select>
              <Option value="new">Новая</Option>
              <Option value="in_progress">В работе</Option>
              <Option value="done">Выполнена</Option>
              <Option value="cancelled">Отменена</Option>
            </Select>
          </Form.Item>
          <Form.Item name="assignee" label="Исполнитель">
            <Input />
          </Form.Item>
          <Form.Item name="due_date" label="Срок">
            <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Sentry.withProfiler(App);
