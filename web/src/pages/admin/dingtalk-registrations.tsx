import { AuditOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Avatar, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  adminServiceListDingTalkRegistrations,
  adminServiceListTransferOrganizations,
} from '@/services/roncin/adminService';
import { toTableRequest, unwrapList } from '@/utils/api';
import RegistrationApproveModal from './components/dingtalk/RegistrationApproveModal';
import RegistrationRejectModal from './components/dingtalk/RegistrationRejectModal';
import RegistrationTransferModal from './components/dingtalk/RegistrationTransferModal';

const { Text } = Typography;

/**
 * 钉钉注册审批队列：待审批人员一站式审批（同意、转派兄弟公司、拒绝）；
 * 钉钉姓名/头像来自扫码返回，供管理员人工认领。
 */
export default function DingTalkRegistrationsPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const approveFormRef = useRef<ProFormInstance | undefined>(undefined);
  const rejectFormRef = useRef<ProFormInstance | undefined>(undefined);
  const transferFormRef = useRef<ProFormInstance | undefined>(undefined);
  const access = useAccess();
  const { initialState } = useInitialState();
  const [approving, setApproving] = useState<API.DingTalkRegistration>();
  const [rejecting, setRejecting] = useState<API.DingTalkRegistration>();
  const [transferring, setTransferring] = useState<API.DingTalkRegistration>();

  const { data: transferOrganizations = [] } = useQuery({
    queryKey: ['dingtalk-registration-transfer-organizations'],
    queryFn: async () =>
      unwrapList(await adminServiceListTransferOrganizations()),
    meta: { silent: true },
  });

  const columns: ProColumns<API.DingTalkRegistration>[] = [
    {
      title: '注册人',
      dataIndex: 'displayName',
      width: 220,
      render: (_, record) => {
        const initial = record.displayName
          ? record.displayName.charAt(0).toUpperCase()
          : '?';
        return (
          <Space size={10} align="center">
            <Avatar
              size={32}
              src={record.avatarUrl}
              style={{
                backgroundColor: '#1677ff',
                fontSize: 14,
                fontWeight: 600,
                flexShrink: 0,
              }}
            >
              {initial}
            </Avatar>
            <Text strong>{record.displayName || '未提供姓名'}</Text>
          </Space>
        );
      },
    },
    {
      title: '注册时间',
      dataIndex: 'registeredAt',
      valueType: 'dateTime',
      width: 170,
    },
    {
      title: '自选目标组织',
      dataIndex: 'requestedOrganizationName',
      width: 200,
      render: (_, record) =>
        record.requestedOrganizationId ? (
          <Text>{record.requestedOrganizationName || '未知组织'}</Text>
        ) : (
          <Tag color="orange" style={{ margin: 0 }}>
            系统管理收口
          </Tag>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 170,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            disabled={!record.requestedOrganizationId}
            title={
              !record.requestedOrganizationId ? '请先转派至目标公司' : undefined
            }
            type="link"
            size="small"
            style={{ padding: 0 }}
            onClick={() => setApproving(record)}
          >
            同意
          </Button>
          <Button
            type="link"
            size="small"
            style={{ padding: 0 }}
            onClick={() => setTransferring(record)}
          >
            转派
          </Button>
          <Button
            type="link"
            danger
            size="small"
            style={{ padding: 0 }}
            onClick={() => setRejecting(record)}
          >
            拒绝
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <ProTable<API.DingTalkRegistration>
        headerTitle={
          <Space size={8}>
            <AuditOutlined style={{ color: '#1677ff' }} />
            <span>待审批注册</span>
            <Text type="secondary">
              员工扫码提交的注册申请，按目标组织路由到这里
            </Text>
          </Space>
        }
        rowKey="userId"
        actionRef={actionRef}
        columns={columns}
        bordered
        search={false}
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        request={async (params) => {
          const response = await adminServiceListDingTalkRegistrations({
            page: params.current,
            pageSize: params.pageSize,
          });
          return toTableRequest(response);
        }}
        toolBarRender={false}
      />

      <RegistrationApproveModal
        registration={approving}
        open={Boolean(approving)}
        onOpenChange={(open) => {
          if (!open) setApproving(undefined);
        }}
        formRef={approveFormRef}
        currentOrganizationId={
          initialState?.currentUser?.currentOrganization?.id
        }
        canReadRoles={access.canReadRoles}
        onReload={() => actionRef.current?.reload()}
      />

      <RegistrationTransferModal
        registration={transferring}
        open={Boolean(transferring)}
        onOpenChange={(open) => {
          if (!open) setTransferring(undefined);
        }}
        formRef={transferFormRef}
        organizations={transferOrganizations}
        onReload={() => actionRef.current?.reload()}
      />

      <RegistrationRejectModal
        registration={rejecting}
        open={Boolean(rejecting)}
        onOpenChange={(open) => {
          if (!open) setRejecting(undefined);
        }}
        formRef={rejectFormRef}
        onReload={() => actionRef.current?.reload()}
      />
    </>
  );
}
