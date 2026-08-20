import {
  EditOutlined,
  FolderOpenOutlined,
  PlusOutlined,
  ReloadOutlined,
  StopOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormList,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Space, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  partnerServiceCreatePartner,
  partnerServiceListPartners,
  partnerServiceSetSupplierBlacklist,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import PartnerSecondary from './partner-secondary';

const roleOptions = [
  { label: '客户', value: 1 },
  { label: '供应商', value: 2 },
  { label: '代理', value: 3 },
  { label: '承运人', value: 4 },
];

const roleLabels: Record<number, string> = Object.fromEntries(
  roleOptions.map((option) => [option.value, option.label]),
);

type PartnerFormValues = {
  code?: string;
  legalName?: string;
  unifiedSocialCreditCode?: string;
  registeredAddress?: string;
  enabled?: boolean;
  roles?: API.PartnerRoleInput[];
  contacts?: API.PartnerContactInput[];
  aliases?: API.PartnerAliasInput[];
};

type BlacklistFormValues = {
  blacklisted?: boolean;
  reason?: string;
};

function roleTags(roles?: API.PartnerRole[]) {
  return (roles ?? []).map((role) => (
    <Tag key={role.type} color={role.blacklisted ? 'error' : undefined}>
      {roleLabels[role.type ?? 0] ?? '未知'}
      {!role.enabled ? '（停用）' : ''}
      {role.blacklisted ? '（黑名单）' : ''}
    </Tag>
  ));
}

export default function Partners() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const blacklistFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const [modalOpen, setModalOpen] = useState(false);
  const [blacklistModalOpen, setBlacklistModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.Partner>();
  const [blacklistPartner, setBlacklistPartner] = useState<API.Partner>();
  const [secondaryPartner, setSecondaryPartner] = useState<API.Partner>();

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (partner: API.Partner) => {
    setEditing(partner);
    setModalOpen(true);
  };

  const openBlacklist = (partner: API.Partner) => {
    setBlacklistPartner(partner);
    blacklistFormRef.current?.resetFields();
    setBlacklistModalOpen(true);
  };

  const columns: ProColumns<API.Partner>[] = [
    {
      title: '编码',
      dataIndex: 'code',
      width: 130,
      fixed: 'left',
      copyable: true,
    },
    {
      title: '法人名称',
      dataIndex: 'legalName',
      width: 240,
      ellipsis: true,
    },
    {
      title: '角色',
      dataIndex: 'role',
      width: 260,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        roleOptions.map((option) => [option.value, { text: option.label }]),
      ),
      render: (_, record) => <Space wrap>{roleTags(record.roles)}</Space>,
    },
    {
      title: '联系人',
      dataIndex: 'contacts',
      width: 90,
      search: false,
      render: (_, record) => record.contacts?.length ?? 0,
    },
    {
      title: '别名',
      dataIndex: 'aliases',
      width: 80,
      search: false,
      render: (_, record) => record.aliases?.length ?? 0,
    },
    {
      title: '统一社会信用代码',
      dataIndex: 'unifiedSocialCreditCode',
      width: 190,
      search: false,
      copyable: true,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 90,
      valueType: 'select',
      valueEnum: {
        true: { text: '启用', status: 'Success' },
        false: { text: '停用', status: 'Default' },
      },
      render: (_, record) =>
        record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 270,
      fixed: 'right',
      render: (_, record) => (
        <Space size={0} wrap>
          <Button
            type="link"
            size="small"
            icon={<FolderOpenOutlined />}
            onClick={() => setSecondaryPartner(record)}
          >
            账户/合同
          </Button>
          {access.canManagePartners ? (
            <>
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
              {record.roles?.some((role) => role.type === 2) ? (
                <Button
                  type="link"
                  size="small"
                  icon={<StopOutlined />}
                  onClick={() => openBlacklist(record)}
                >
                  黑名单
                </Button>
              ) : null}
            </>
          ) : null}
        </Space>
      ),
    },
  ];

  return (
    <>
      <ProTable<API.Partner>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1400 }}
        request={async (params) => {
          const response = await partnerServiceListPartners({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            role: params.role,
            enabled: params.enabled,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
            total: response.total ?? 0,
          };
        }}
        search={{ labelWidth: 'auto', defaultCollapsed: false }}
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => actionRef.current?.reload()}
          >
            刷新
          </Button>,
          access.canManagePartners ? (
            <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新增往来单位
            </Button>
          ) : null,
        ].filter(Boolean) as React.ReactNode[]}
      />

      <ModalForm<PartnerFormValues>
        title={editing ? '编辑往来单位' : '新增往来单位'}
        open={modalOpen}
        formRef={formRef}
        modalProps={{
          destroyOnClose: true,
          width: 760,
          onCancel: () => setModalOpen(false),
        }}
        initialValues={
          editing
            ? {
                ...editing,
                roles: editing.roles?.map((role) => ({
                  type: role.type,
                  enabled: role.enabled,
                })),
              }
            : {
                roles: [{ type: 1, enabled: true }],
                contacts: [],
                aliases: [],
              }
        }
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const roles = values.roles ?? [];
          const contacts = values.contacts ?? [];
          const aliases = values.aliases ?? [];
          if (editing?.id) {
            await partnerServiceUpdatePartner(
              { id: editing.id },
              {
                id: editing.id,
                legalName: values.legalName ?? '',
                unifiedSocialCreditCode: values.unifiedSocialCreditCode,
                registeredAddress: values.registeredAddress,
                enabled: values.enabled ?? true,
                roles,
                contacts,
                aliases,
              },
            );
            message.success('往来单位已更新');
          } else {
            await partnerServiceCreatePartner({
              code: values.code ?? '',
              legalName: values.legalName ?? '',
              unifiedSocialCreditCode: values.unifiedSocialCreditCode,
              registeredAddress: values.registeredAddress,
              roles,
              contacts,
              aliases,
            });
            message.success('往来单位已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="编码"
          placeholder="请输入组织内唯一编码"
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请输入往来单位编码' }]}
        />
        <ProFormText
          name="legalName"
          label="法人名称"
          placeholder="请输入完整法人名称"
          rules={[{ required: true, message: '请输入法人名称' }]}
        />
        <ProFormText name="unifiedSocialCreditCode" label="统一社会信用代码" />
        <ProFormTextArea name="registeredAddress" label="注册地址" fieldProps={{ rows: 2 }} />
        <ProFormSwitch name="enabled" label="启用状态" initialValue />
        <ProFormList
          name="roles"
          label="业务角色"
          creatorButtonProps={{ creatorButtonText: '添加角色' }}
          min={1}
        >
          <Space align="start">
            <ProFormSelect
              name="type"
              label="角色"
              options={roleOptions}
              width="sm"
              rules={[{ required: true, message: '请选择角色' }]}
            />
            <ProFormSwitch name="enabled" label="启用" initialValue />
          </Space>
        </ProFormList>
        <ProFormList
          name="contacts"
          label="联系人"
          creatorButtonProps={{ creatorButtonText: '添加联系人' }}
        >
          <Space align="start" wrap>
            <ProFormText name="name" label="姓名" rules={[{ required: true, message: '请输入联系人姓名' }]} />
            <ProFormText name="phone" label="电话" />
            <ProFormText name="email" label="邮箱" rules={[{ type: 'email', message: '请输入正确的邮箱地址' }]} />
            <ProFormSwitch name="isPrimary" label="主联系人" />
            <ProFormText name="note" label="备注" />
          </Space>
        </ProFormList>
        <ProFormList
          name="aliases"
          label="别名"
          creatorButtonProps={{ creatorButtonText: '添加别名' }}
        >
          <Space align="start">
            <ProFormText name="aliasName" label="别名" rules={[{ required: true, message: '请输入别名' }]} />
            <ProFormDigit name="sortOrder" label="排序" min={0} fieldProps={{ precision: 0 }} />
          </Space>
        </ProFormList>
      </ModalForm>

      <ModalForm<BlacklistFormValues>
        title={`供应商黑名单 - ${blacklistPartner?.legalName ?? ''}`}
        open={blacklistModalOpen}
        formRef={blacklistFormRef}
        initialValues={{
          blacklisted: Boolean(
            blacklistPartner?.roles?.find((role) => role.type === 2)?.blacklisted,
          ),
        }}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setBlacklistModalOpen(false),
        }}
        onOpenChange={setBlacklistModalOpen}
        onFinish={async (values) => {
          if (!blacklistPartner?.id) return false;
          await partnerServiceSetSupplierBlacklist(
            { id: blacklistPartner.id },
            {
              id: blacklistPartner.id,
              blacklisted: values.blacklisted ?? false,
              reason: values.reason?.trim() ?? '',
            },
          );
          message.success(values.blacklisted ? '已加入供应商黑名单' : '已移出供应商黑名单');
          setBlacklistModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSwitch name="blacklisted" label="列入黑名单" />
        <ProFormTextArea
          name="reason"
          label="变更原因"
          rules={[{ required: true, message: '请输入黑名单变更原因' }]}
          fieldProps={{ rows: 4, maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <PartnerSecondary
        partner={secondaryPartner}
        open={Boolean(secondaryPartner)}
        canManage={access.canManagePartners}
        onClose={() => setSecondaryPartner(undefined)}
      />
    </>
  );
}
