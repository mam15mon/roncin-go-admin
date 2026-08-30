import type { ProColumns } from '@ant-design/pro-components';
import { ProFormText } from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { Tag, Typography } from 'antd';
import React, { forwardRef } from 'react';
import {
  SubEntityDrawerTemplate,
  type SubEntityDrawerRef,
} from '@/components/ui/sub-entity-drawer';
import {
  orderPersonnelRoleOptions,
  orderPersonnelRoleValueEnum,
} from '../../common';
import {
  orderPersonnelServiceAssignPersonnel,
  orderPersonnelServiceListPersonnel,
  orderPersonnelServiceRemovePersonnel,
} from '@/services/roncin/orderPersonnelService';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

export type PersonnelDrawerRef = SubEntityDrawerRef<API.Order>;

type PersonnelDrawerProps = {
  canAssign: boolean;
  canRemove: boolean;
};

type PersonnelFormValues = {
  userId: string;
  organizationId: string;
  role: number;
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
    render: (_, record) => formatDate(record.assignedAt),
  },
];

const PersonnelDrawer = forwardRef<PersonnelDrawerRef, PersonnelDrawerProps>(
  function PersonnelDrawer({ canAssign, canRemove }, ref) {
    return (
      <SubEntityDrawerTemplate<
        API.OrderPersonnel,
        API.Order,
        PersonnelFormValues
      >
        ref={ref}
        entityName="协作人员"
        drawerTitle={(order) =>
          order
            ? `订单协作团队 - ${order.orderNo || order.id}`
            : '订单协作团队'
        }
        drawerWidth={820}
        canCreate={canAssign}
        canUpdate={false}
        canRemove={canRemove}
        columns={columns}
        fetchList={(order) =>
          orderPersonnelServiceListPersonnel({
            orderId: order.id as string,
          })
        }
        createItem={(values, order) =>
          orderPersonnelServiceAssignPersonnel(
            { orderId: order.id as string },
            {
              orderId: order.id as string,
              userId: values.userId.trim(),
              organizationId: values.organizationId.trim(),
              role: Number(values.role),
            },
          )
        }
        removeItem={(record, order) =>
          orderPersonnelServiceRemovePersonnel({
            orderId: order.id as string,
            id: (record.id as string) || (record.userId as string),
          })
        }
        modalWidth={520}
        renderFormItems={() => (
          <>
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
          </>
        )}
      />
    );
  },
);

export default PersonnelDrawer;
