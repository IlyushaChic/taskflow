import React from 'react';
import { Input, Select, Space } from 'antd';
import { TaskFilter } from '../types/task';

const { Option } = Select;

interface Props {
  filters: TaskFilter;
  setFilters: React.Dispatch<React.SetStateAction<TaskFilter>>;
}

const TaskFilters: React.FC<Props> = ({ filters, setFilters }) => {
  const handleAssigneeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFilters((prev) => ({ ...prev, assignee: e.target.value }));
  };

  const handleStatusChange = (value: string | undefined) => {
    setFilters((prev) => ({ ...prev, status: value || '' }));
  };

  return (
    <Space style={{ marginBottom: 16 }}>
      <Input
        placeholder="Фильтр по исполнителю"
        value={filters.assignee}
        onChange={handleAssigneeChange}
        allowClear
      />
      <Select
        placeholder="Статус"
        style={{ width: 120 }}
        value={filters.status || undefined}
        onChange={handleStatusChange}
        allowClear
      >
        <Option value="new">Новая</Option>
        <Option value="in_progress">В работе</Option>
        <Option value="done">Выполнена</Option>
        <Option value="cancelled">Отменена</Option>
      </Select>
    </Space>
  );
};

export default TaskFilters;