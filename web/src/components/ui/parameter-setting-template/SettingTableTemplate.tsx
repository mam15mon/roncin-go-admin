import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ModalForm, ProTable } from '@ant-design/pro-components';
import { App, Button, Card } from 'antd';
import React, { useMemo, useRef, useState } from 'react';
import type { SettingTableTemplateProps } from './types';

export function SettingTableTemplate<
  TRecord extends Record<string, any> = Record<string, any>,
  TFormValues extends Record<string, any> = Record<string, any>,
>({
  entityName,
  rowKey = 'id',
  columns,
  renderFormItems,
  query,
  createItem,
  updateItem,
  canCreate = true,
  canUpdate = true,
  initialValues,
  beforeSubmit,
  modalWidth = 520,
  grid = false,
  search = false,
  pagination = false,
  rowSelection,
  scroll,
  extraToolBarButtons = [],
  cardStyle,
  actionRef: externalActionRef,
}: SettingTableTemplateProps<TRecord, TFormValues>) {
  const { message } = App.useApp();
  const internalActionRef = useRef<ActionType | undefined>(undefined);
  const actionRef = externalActionRef || internalActionRef;

  const [modalOpen, setModalOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<TRecord | undefined>(undefined);

  // 1. 处理操作列（若未自定义 option 列且具备修改权限，自动追加标准编辑列）
  const tableColumns = useMemo<ProColumns<TRecord>[]>(() => {
    const hasOptionColumn = columns.some((col) => col.valueType === 'option');
    if (hasOptionColumn || !canUpdate || !updateItem) {
      return columns;
    }

    const actionColumn: ProColumns<TRecord> = {
      title: '操作',
      valueType: 'option',
      width: 90,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setEditingRecord(record);
            setModalOpen(true);
          }}
        >
          编辑
        </Button>
      ),
    };

    return [...columns, actionColumn];
  }, [columns, canUpdate, updateItem]);

  // 2. 初始表单值
  const currentInitialValues = useMemo<Partial<TFormValues>>(() => {
    if (initialValues) {
      return initialValues(editingRecord);
    }
    return {} as Partial<TFormValues>;
  }, [initialValues, editingRecord]);

  // 3. 提交处理
  const handleFinish = async (values: TFormValues) => {
    try {
      const submitData = beforeSubmit ? beforeSubmit(values, editingRecord) : values;

      if (editingRecord) {
        if (updateItem) {
          await updateItem(editingRecord, submitData);
          message.success(`${entityName}更新成功`);
        }
      } else {
        if (createItem) {
          await createItem(submitData);
          message.success(`${entityName}创建成功`);
        }
      }

      setModalOpen(false);
      actionRef.current?.reload();
      return true;
    } catch {
      // 错误由全局拦截器或底层抛出处理
      return false;
    }
  };

  return (
    <Card
      bordered={false}
      style={{
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
        ...cardStyle,
      }}
      styles={{ body: { padding: '12px 16px' } }}
    >
      <ProTable<TRecord>
        actionRef={actionRef}
        rowKey={rowKey}
        columns={tableColumns}
        search={search ? undefined : false}
        pagination={
          pagination === false
            ? false
            : typeof pagination === 'object'
            ? pagination
            : undefined
        }
        rowSelection={rowSelection || undefined}
        scroll={scroll}
        request={async (params) => {
          const res = await query(params);
          return {
            data: res.data ?? [],
            success: res.success ?? true,
            total: res.total,
          };
        }}
        toolBarRender={() => [
          ...extraToolBarButtons,
          ...(canCreate && createItem
            ? [
                <Button
                  key="create-btn"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setEditingRecord(undefined);
                    setModalOpen(true);
                  }}
                >
                  新建{entityName}
                </Button>,
              ]
            : []),
        ]}
      />

      <ModalForm<TFormValues>
        title={editingRecord ? `编辑${entityName}` : `新建${entityName}`}
        open={modalOpen}
        initialValues={currentInitialValues}
        layout="horizontal"
        labelAlign="right"
        labelCol={{ flex: '96px' }}
        wrapperCol={{ flex: 'auto' }}
        grid={grid}
        modalProps={{
          destroyOnHidden: true,
          width: modalWidth,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={handleFinish}
      >
        {renderFormItems(editingRecord)}
      </ModalForm>
    </Card>
  );
}

export default SettingTableTemplate;
