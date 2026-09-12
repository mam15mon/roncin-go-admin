import { DingdingOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess, useModel } from '@umijs/max';
import { App, Button, Popconfirm, Space, Typography } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import {
  dingTalkInvitationStatusMeta,
  makeValueEnum,
  statusTag,
} from '@/constants/statusMeta';
import { DingTalkInvitationStatus } from '@/enums.generated';
import {
  adminServiceListDingTalkInvitations,
  adminServiceListOrganizations,
  adminServiceRevokeDingTalkInvitation,
} from '@/services/roncin/adminService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { organizationSelectOptions } from './components/dingtalk/constants';
import InvitationFormModal from './components/dingtalk/InvitationFormModal';

const { Text } = Typography;

const invitationStatusValueEnum = makeValueEnum(dingTalkInvitationStatusMeta);

/** 把 ProTable 搜索表单的状态值规范化为后端枚举参数（空值不发送）。 */
export function normalizeInvitationStatusFilter(
  value: unknown,
): number | undefined {
  if (value === undefined || value === null || value === '') return undefined;
  return Number(value);
}

/**
 * 钉钉扫码邀请面板（通道 A）：按手机号预建邀请，员工扫码匹配后自动激活，
 * 无需人工审批；手机号由服务端脱敏返回，前端不接触完整号码。
 */
export default function DingTalkInvitationsPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useModel('@@initialState');
  const [createOpen, setCreateOpen] = useState(false);
  const [allOrganizations, setAllOrganizations] = useState<
    API.AdminOrganization[]
  >([]);
  const currentOrganization = initialState?.currentUser?.currentOrganization;
  const currentOrganizationId = currentOrganization?.id;

  useEffect(() => {
    // 全组织列表仅限具备全局组织读取权限的管理员；普通组织管理员的目标组织
    // 就是当前组织，由服务端按可写范围最终裁决。
    if (access.canReadOrganizations) {
      adminServiceListOrganizations().then((response) =>
        setAllOrganizations(unwrapList(response)),
      );
    }
  }, [access.canReadOrganizations]);

  // 邀请目标组织候选：全局管理员可任选可写组织，普通管理员固定当前组织。
  const invitationOrganizations: API.AdminOrganization[] =
    access.canReadOrganizations
      ? allOrganizations
      : currentOrganization?.id
        ? [
            {
              id: currentOrganization.id,
              name: currentOrganization.name,
              code: currentOrganization.code,
            },
          ]
        : [];

  const handleRevoke = async (record: API.DingTalkInvitation) => {
    if (!record.id) return;
    await adminServiceRevokeDingTalkInvitation({ id: record.id });
    message.success('邀请已撤销，该手机号扫码不再自动激活');
    actionRef.current?.reload();
  };

  // 组织筛选仅对具备全局组织读取权限的管理员开放（多组织可选）。
  const organizationFilterColumn: ProColumns<API.DingTalkInvitation>[] =
    access.canReadOrganizations
      ? [
          {
            title: '目标组织',
            dataIndex: 'organizationId',
            hideInTable: true,
            valueType: 'select',
            fieldProps: {
              options: organizationSelectOptions(allOrganizations),
              placeholder: '按组织筛选',
            },
          },
        ]
      : [];

  const columns: ProColumns<API.DingTalkInvitation>[] = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 96,
      valueEnum: invitationStatusValueEnum,
      render: (_, record) =>
        statusTag(dingTalkInvitationStatusMeta, record.status ?? 0, '未知'),
    },
    ...organizationFilterColumn,
    {
      title: '手机号',
      dataIndex: 'mobileMasked',
      width: 130,
      search: false,
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace' }}>
          {record.mobileMasked || '-'}
        </Text>
      ),
    },
    {
      title: '备注姓名',
      dataIndex: 'displayName',
      width: 120,
      ellipsis: true,
      search: false,
      render: (_, record) =>
        record.displayName || <Text type="secondary">-</Text>,
    },
    {
      title: '目标组织',
      dataIndex: 'organizationName',
      width: 200,
      ellipsis: true,
      search: false,
      render: (_, record) => record.organizationName || '-',
    },
    {
      title: '初始角色',
      dataIndex: 'roleName',
      width: 140,
      ellipsis: true,
      search: false,
      render: (_, record) => record.roleName || <Text type="secondary">-</Text>,
    },
    {
      title: '已激活',
      dataIndex: 'consumedName',
      width: 150,
      ellipsis: true,
      search: false,
      render: (_, record) =>
        record.consumedName || <Text type="secondary">-</Text>,
    },
    {
      title: '邀请人',
      dataIndex: 'inviterName',
      width: 110,
      ellipsis: true,
      search: false,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 150,
      search: false,
    },
    {
      title: '过期时间',
      dataIndex: 'expiresAt',
      valueType: 'dateTime',
      width: 150,
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      fixed: 'right',
      render: (_, record) =>
        record.status ===
        DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_PENDING ? (
          <Popconfirm
            title="确定撤销该邀请？"
            description="撤销后该手机号扫码不再自动激活；撤销操作不可恢复。"
            okText="撤销"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => handleRevoke(record)}
          >
            <Button type="link" danger size="small" style={{ padding: 0 }}>
              撤销
            </Button>
          </Popconfirm>
        ) : null,
    },
  ];

  return (
    <>
      <ProTable<API.DingTalkInvitation>
        headerTitle={
          <Space size={8}>
            <DingdingOutlined style={{ color: '#1677ff' }} />
            <span>扫码邀请</span>
            <Text type="secondary">按手机号预建，员工扫码自动激活</Text>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        request={async (params) => {
          const response = await adminServiceListDingTalkInvitations({
            page: params.current,
            pageSize: params.pageSize,
            status: normalizeInvitationStatusFilter(params.status),
            organizationId:
              typeof params.organizationId === 'string' &&
              params.organizationId !== ''
                ? params.organizationId
                : undefined,
          });
          return toTableRequest(response);
        }}
        toolBarRender={() => [
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            新建邀请
          </Button>,
        ]}
      />

      <InvitationFormModal
        open={createOpen}
        onOpenChange={setCreateOpen}
        formRef={formRef}
        organizations={invitationOrganizations}
        currentOrganizationId={currentOrganizationId}
        canReadRoles={access.canReadRoles}
        onReload={() => actionRef.current?.reload()}
      />
    </>
  );
}
