import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { App, Button, Drawer, Popconfirm, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderPersonnelRoleOptions,
  orderPersonnelRoleValueEnum,
} from '../../common';
import {
  orderPersonnelServiceAssignPersonnel,
  orderPersonnelServiceListPersonnel,
  orderPersonnelServiceRemovePersonnel,
} from '@/services/roncin/orderPersonnelService';

const { Text } = Typography;

export type PersonnelDrawerRef = {
  open: (order: API.Order) => void;
};

type PersonnelDrawerProps = {
  canAssign: boolean;
  canRemove: boolean;
};

type PersonnelFormValues = {
  userId: string;
  organizationId: string;
  role: number;
};

const PersonnelDrawer = forwardRef<PersonnelDrawerRef, PersonnelDrawerProps>(
  function PersonnelDrawer({ canAssign, canRemove }, ref) {
    const { message } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
      },
    }));

    const openAssignPersonnel = () => {
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const columns: ProColumns<API.OrderPersonnel>[] = [
      {
        title: '用户 ID',
        dataIndex: 'userId',
        copyable: true,
        ellipsis: true,
        render: (_, record) => (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.userId || '-'}
          </Text>
        ),
      },
      {
        title: '协作角色',
        dataIndex: 'role',
        valueType: 'select',
        valueEnum: orderPersonnelRoleValueEnum,
        render: (_, record) =>
          record.role !== undefined &&
          orderPersonnelRoleValueEnum[record.role] ? (
            <Tag color="blue" variant="filled">
              {orderPersonnelRoleValueEnum[record.role]?.text}
            </Tag>
          ) : (
            '-'
          ),
      },
      {
        title: '指派时间',
        dataIndex: 'assignedAt',
        valueType: 'dateTime',
        width: 180,
        render: (_, record) =>
          record.assignedAt
            ? dayjs(record.assignedAt).format('YYYY-MM-DD HH:mm:ss')
            : '-',
      },
      {
        title: '操作',
        valueType: 'option',
        width: 80,
        render: (_, record) => {
          if (!canRemove) return null;
          return (
            <Popconfirm
              title="确定移除该协作人员？"
              onConfirm={async () => {
                if (!order?.id || !record.id) return;
                await orderPersonnelServiceRemovePersonnel({
                  orderId: order.id,
                  id: record.id,
                });
                message.success('移除协作人员成功');
                actionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small">
                删除
              </Button>
            </Popconfirm>
          );
        },
      },
    ];

    return (
      <>
        <Drawer
          title={
            order
              ? `订单协作团队 - ${order.orderNo || order.id}`
              : '订单协作团队'
          }
          open={drawerOpen}
          onClose={() => {
            setDrawerOpen(false);
            setOrder(undefined);
          }}
          size={820}
          destroyOnHidden
        >
          {order?.id && (
            <ProTable<API.OrderPersonnel>
              actionRef={actionRef}
              rowKey={(record) => record.id || `${record.userId}-${record.role}`}
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderPersonnelServiceListPersonnel({
                  orderId: order.id as string,
                });
                return {
                  data: response.data ?? [],
                  success: response.success ?? true,
                };
              }}
              toolBarRender={() => [
                canAssign && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openAssignPersonnel}
                  >
                    分配协作人员
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<PersonnelFormValues>
          title="分配协作人员"
          open={modalOpen}
          formRef={formRef}
          modalProps={{
            destroyOnHidden: true,
            width: 520,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onFinish={async (values) => {
            if (!order?.id) return false;
            await orderPersonnelServiceAssignPersonnel(
              { orderId: order.id },
              {
                orderId: order.id,
                userId: values.userId.trim(),
                organizationId: values.organizationId.trim(),
                role: Number(values.role),
              },
            );
            message.success('分配协作人员成功');
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
          <ProFormText
            name="userId"
            label="用户 UUID"
            placeholder="请输入组织内用户 UUID"
            rules={[{ required: true, message: '请输入用户 UUID' }]}
          />
          <ProFormText
            name="organizationId"
            label="所属公司 UUID"
            placeholder="请输入人员所属公司 UUID"
            rules={[{ required: true, message: '请输入人员所属公司 UUID' }]}
          />
          <ProFormSearchableSelect
            name="role"
            label="担当角色"
            rules={[{ required: true, message: '请选择角色' }]}
            options={orderPersonnelRoleOptions}
            placeholder="请选择角色"
          />
        </ModalForm>
      </>
    );
  },
);

export default PersonnelDrawer;
