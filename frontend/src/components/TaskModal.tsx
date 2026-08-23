import React, { useEffect } from 'react';
import { Modal, Form, Input, Select, DatePicker } from 'antd';
import dayjs from 'dayjs';
import { Task } from '../types/task';

const { Option } = Select;

interface Props {
  open: boolean;
  editingTask: Task | null;
  onFinish: (values: any) => void;
  onCancel: () => void;
}

const TaskModal: React.FC<Props> = ({ open, editingTask, onFinish, onCancel }) => {
  const [form] = Form.useForm();

  useEffect(() => {
    if (editingTask) {
      form.setFieldsValue({
        ...editingTask,
        due_date: editingTask.due_date ? dayjs(editingTask.due_date) : null,
      });
    } else {
      form.resetFields();
    }
  }, [editingTask, form]);

  const handleOk = () => {
    form.submit();
  };

  return (
    <Modal
      title={editingTask ? 'Редактировать задачу' : 'Создать задачу'}
      open={open}
      onCancel={onCancel}
      onOk={handleOk}
      destroyOnClose
    >
      <Form form={form} layout="vertical" onFinish={onFinish}>
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
  );
};

export default TaskModal;