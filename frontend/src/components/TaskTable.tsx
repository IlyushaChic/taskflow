import React from 'react';
import { Table, Button, Space } from 'antd';
import { Task } from '../types/task';

interface Props {
  tasks: Task[];
  loading: boolean;
  onEdit: (task: Task) => void;
  onDelete: (id: string) => void;
}

const TaskTable: React.FC<Props> = ({ tasks, loading, onEdit, onDelete }) => {
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
          <Button onClick={() => onEdit(record)}>Редактировать</Button>
          <Button danger onClick={() => onDelete(record.id)}>Удалить</Button>
        </Space>
      ),
    },
  ];

  return <Table dataSource={tasks} columns={columns} loading={loading} rowKey="id" />;
};

export default TaskTable;