import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ModalForm, ProTable } from '@ant-design/pro-components';
import { App, Button, Drawer, Popconfirm, Space } from 'antd';
import React, {
  forwardRef,
  useImperativeHandle,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { toTableRequest } from '@/utils/api';

export type SubEntityDrawerRef<TParent = any> = {
  open: (parent: TParent) => void;
  close: () => void;
  reload: () => void;
};

export interface SubEntityDrawerTemplateProps<
  TItem extends { id?: string | number },
  TParent extends { id?: string | number } = any,
  TFormValues = any,
> {
  entityName: string;
  drawerTitle?: string | ((parent?: TParent) => ReactNode);
  drawerWidth?: number | string;
  canCreate?: boolean;
  canUpdate?: boolean;
  canRemove?: boolean;
  columns: ProColumns<TItem>[];
  hideDefaultActions?: boolean;
  actionColumnWidth?: number;
  fetchList: (
    parent: TParent,
  ) => Promise<{ data?: TItem[]; success?: boolean }>;
  createItem?: (values: TFormValues, parent: TParent) => Promise<any>;
  updateItem?: (
    item: TItem,
    values: TFormValues,
    parent: TParent,
  ) => Promise<any>;
  removeItem?: (item: TItem, parent: TParent) => Promise<any>;
  initialValues?: (
    item?: TItem,
    parent?: TParent,
  ) => TFormValues | Promise<TFormValues>;
  renderFormItems: (
    item?: TItem,
    formRef?: React.RefObject<ProFormInstance | undefined>,
    parent?: TParent,
  ) => ReactNode;
  modalWidth?: number;
  onOpen?: (parent: TParent) => void;
  extraToolbar?: (
    parent?: TParent,
    actionRef?: React.RefObject<ActionType | undefined>,
  ) => ReactNode[];
}

export function SubEntityDrawerTemplateInner<
  TItem extends { id?: string | number },
  TParent extends { id?: string | number } = any,
  TFormValues = any,
>(
  {
    entityName,
    drawerTitle,
    drawerWidth = 920,
    canCreate = true,
    canUpdate = true,
    canRemove = true,
    columns,
    hideDefaultActions = false,
    actionColumnWidth = 120,
    fetchList,
    createItem,
    updateItem: updateItemProp,
    removeItem,
    initialValues,
    renderFormItems,
    modalWidth = 560,
    onOpen,
    extraToolbar,
  }: SubEntityDrawerTemplateProps<TItem, TParent, TFormValues>,
  ref: React.ForwardedRef<SubEntityDrawerRef<TParent>>,
) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [parentRecord, setParentRecord] = useState<TParent>();
  const [editingItem, setEditingItem] = useState<TItem>();

  useImperativeHandle(ref, () => ({
    open: (parent: TParent) => {
      setParentRecord(parent);
      setDrawerOpen(true);
      onOpen?.(parent);
    },
    close: () => {
      setDrawerOpen(false);
      setParentRecord(undefined);
    },
    reload: () => {
      actionRef.current?.reload();
    },
  }));

  const openCreate = () => {
    setEditingItem(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: TItem) => {
    setEditingItem(record);
    if (initialValues) {
      const initVals = initialValues(record, parentRecord);
      if (initVals instanceof Promise) {
        initVals.then((vals) => formRef.current?.setFieldsValue(vals));
      } else {
        formRef.current?.setFieldsValue(initVals);
      }
    }
    setModalOpen(true);
  };

  const resolvedColumns = React.useMemo(() => {
    if (hideDefaultActions || (!canUpdate && !canRemove)) {
      return columns;
    }

    const actionCol: ProColumns<TItem> = {
      title: '操作',
      valueType: 'option',
      width: actionColumnWidth,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          {canUpdate && updateItemProp && (
            <Button
              type="link"
              size="small"
              onClick={() => openEdit(record)}
            >
              编辑
            </Button>
          )}
          {canRemove && removeItem && (
            <Popconfirm
              title={`确定移除该${entityName}？`}
              onConfirm={async () => {
                if (!parentRecord) return;
                const removed = await removeItem(record, parentRecord);
                if (removed === false) return;
                message.success(`移除${entityName}成功`);
                actionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small">
                删除
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    };

    return [...columns, actionCol];
  }, [
    columns,
    hideDefaultActions,
    canUpdate,
    canRemove,
    actionColumnWidth,
    updateItemProp,
    removeItem,
    entityName,
    parentRecord,
    message,
  ]);

  const computedTitle = React.useMemo(() => {
    if (typeof drawerTitle === 'function') {
      return drawerTitle(parentRecord);
    }
    if (drawerTitle) return drawerTitle;
    return `${entityName}管理`;
  }, [drawerTitle, entityName, parentRecord]);

  return (
    <>
      <Drawer
        title={computedTitle}
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setParentRecord(undefined);
        }}
        size={typeof drawerWidth === 'number' ? drawerWidth : 920}
        destroyOnHidden
      >
        {parentRecord && (
          <ProTable<TItem>
            actionRef={actionRef}
            rowKey="id"
            columns={resolvedColumns}
            bordered
            search={false}
            pagination={false}
            request={async () => {
              const response = await fetchList(parentRecord);
              return toTableRequest(response);
            }}
            toolBarRender={() => [
              ...(extraToolbar ? extraToolbar(parentRecord, actionRef) : []),
              canCreate && createItem && (
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openCreate}
                >
                  添加{entityName}
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<TFormValues>
        title={editingItem ? `编辑${entityName}` : `添加${entityName}`}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingItem && initialValues
            ? (initialValues(editingItem, parentRecord) as any)
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: modalWidth,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (!parentRecord) return false;
          if (editingItem && updateItemProp) {
            await updateItemProp(editingItem, values, parentRecord);
            message.success(`更新${entityName}成功`);
          } else if (createItem) {
            await createItem(values, parentRecord);
            message.success(`添加${entityName}成功`);
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        {renderFormItems(editingItem, formRef, parentRecord)}
      </ModalForm>
    </>
  );
}

export const SubEntityDrawerTemplate = forwardRef(
  SubEntityDrawerTemplateInner,
) as <
  TItem extends { id?: string | number },
  TParent extends { id?: string | number } = any,
  TFormValues = any,
>(
  props: SubEntityDrawerTemplateProps<TItem, TParent, TFormValues> & {
    ref?: React.ForwardedRef<SubEntityDrawerRef<TParent>>;
  },
) => React.ReactElement;

export default SubEntityDrawerTemplate;
