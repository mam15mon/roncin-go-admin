import { PlusOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm } from '@ant-design/pro-components';
import { Button, Col, Empty, Form, Row, Space, Tag } from 'antd';
import React, { useRef, useState, type ReactNode } from 'react';

export interface SubEntityCardGridProps<TItem, TFormValues = any> {
  entityName: string;
  items: TItem[];
  title?: ReactNode;
  canCreate?: boolean;
  canUpdate?: boolean;
  canDelete?: boolean;
  modalWidth?: number;
  colSpan?: {
    xs?: number;
    sm?: number;
    md?: number;
    lg?: number;
    xl?: number;
  };
  emptyText?: string;
  renderCard: (
    item: TItem,
    index: number,
    helpers: {
      openEdit: () => void;
      deleteItem: () => void;
    },
  ) => ReactNode;
  initialValues?: (item?: TItem, index?: number) => TFormValues;
  renderFormItems: (
    item?: TItem,
    form?: any,
    index?: number,
  ) => ReactNode;
  onSave: (
    values: TFormValues,
    editingItem?: TItem,
    editingIndex?: number,
  ) => Promise<boolean | undefined> | boolean | undefined;
  onDelete?: (item: TItem, index: number) => Promise<void> | void;
  extraHeader?: ReactNode;
}

export function SubEntityCardGrid<TItem, TFormValues = any>({
  entityName,
  items = [],
  title,
  canCreate = true,
  modalWidth = 540,
  colSpan = { xs: 24, sm: 12, md: 8, lg: 6 },
  emptyText,
  renderCard,
  initialValues,
  renderFormItems,
  onSave,
  onDelete,
  extraHeader,
}: SubEntityCardGridProps<TItem, TFormValues>) {
  const [modalOpen, setModalOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | undefined>(undefined);
  const [editingItem, setEditingItem] = useState<TItem | undefined>(undefined);
  const [form] = Form.useForm();
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  const handleOpenAdd = () => {
    setEditingIndex(undefined);
    setEditingItem(undefined);
    form.resetFields();
    if (initialValues) {
      form.setFieldsValue(initialValues(undefined, undefined));
    }
    setModalOpen(true);
  };

  const handleOpenEdit = (item: TItem, index: number) => {
    setEditingIndex(index);
    setEditingItem(item);
    form.resetFields();
    if (initialValues) {
      form.setFieldsValue(initialValues(item, index));
    }
    setModalOpen(true);
  };

  const handleDelete = (item: TItem, index: number) => {
    onDelete?.(item, index);
  };

  return (
    <div>
      {(title || canCreate || extraHeader) && (
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 16,
          }}
        >
          <Space size={8}>
            {typeof title === 'string' ? (
              <span style={{ fontSize: 14, fontWeight: 600 }}>{title}</span>
            ) : (
              title
            )}
            <Tag color="blue" variant="filled">
              {items.length}
            </Tag>
          </Space>
          <Space size={8}>
            {extraHeader}
            {canCreate && (
              <Button
                type="primary"
                size="small"
                icon={<PlusOutlined />}
                onClick={handleOpenAdd}
              >
                添加{entityName}
              </Button>
            )}
          </Space>
        </div>
      )}

      {items.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={emptyText || `暂无${entityName}信息`}
          style={{ margin: '32px 0' }}
        >
          {canCreate && (
            <Button
              type="dashed"
              icon={<PlusOutlined />}
              onClick={handleOpenAdd}
            >
              添加第一条{entityName}
            </Button>
          )}
        </Empty>
      ) : (
        <Row gutter={[16, 16]}>
          {items.map((item, index) => (
            <Col {...colSpan} key={(item as any)?.id || index}>
              {renderCard(item, index, {
                openEdit: () => handleOpenEdit(item, index),
                deleteItem: () => handleDelete(item, index),
              })}
            </Col>
          ))}
        </Row>
      )}

      <ModalForm<TFormValues>
        title={editingItem ? `编辑${entityName}` : `添加${entityName}`}
        open={modalOpen}
        form={form}
        formRef={formRef}
        modalProps={{
          destroyOnHidden: true,
          width: modalWidth,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const success = await onSave(values, editingItem, editingIndex);
          if (success !== false) {
            setModalOpen(false);
            return true;
          }
          return false;
        }}
      >
        {renderFormItems(editingItem, form, editingIndex)}
      </ModalForm>
    </div>
  );
}

export default SubEntityCardGrid;
