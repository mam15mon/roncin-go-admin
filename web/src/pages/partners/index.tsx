import {
  ContactsOutlined,
  EditOutlined,
  ExportOutlined,
  FolderOpenOutlined,
  ImportOutlined,
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
  PageContainer,
  ProFormDigit,
  ProFormList,
  ProFormRadio,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  partnerServiceCreatePartner,
  partnerServiceExportPartners,
  partnerServiceImportPartners,
  partnerServiceListPartners,
  partnerServiceSetSupplierBlacklist,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import PartnerSecondary from './partner-secondary';

const { Text } = Typography;

const roleOptions = [
  { label: '客户', value: 1, color: 'blue' },
  { label: '供应商', value: 2, color: 'green' },
  { label: '代理', value: 3, color: 'purple' },
  { label: '承运人', value: 4, color: 'orange' },
];

const roleMap = new Map(roleOptions.map((opt) => [opt.value, opt]));

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

type PartnerImportFormValues = {
  source?: string;
  mode?: number;
  items?: string;
};

function roleTags(roles?: API.PartnerRole[]) {
  return (roles ?? []).map((role) => {
    const item = roleMap.get(role.type ?? 0);
    const color = role.blacklisted ? 'error' : item?.color || 'default';
    return (
      <Tag key={role.type} color={color} bordered={false}>
        {roleLabels[role.type ?? 0] ?? '未知'}
        {!role.enabled ? ' (停用)' : ''}
        {role.blacklisted ? ' [黑名单]' : ''}
      </Tag>
    );
  });
}

export default function Partners() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const blacklistFormRef = useRef<ProFormInstance | undefined>(undefined);
  const importFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const [modalOpen, setModalOpen] = useState(false);
  const [blacklistModalOpen, setBlacklistModalOpen] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [editing, setEditing] = useState<API.Partner>();
  const [blacklistPartner, setBlacklistPartner] = useState<API.Partner>();
  const [secondaryPartner, setSecondaryPartner] = useState<API.Partner>();

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openImport = () => {
    importFormRef.current?.resetFields();
    setImportModalOpen(true);
  };

  const handleExport = async () => {
    try {
      setExporting(true);
      const response = await partnerServiceExportPartners({});
      const data = response.data ?? [];
      if (data.length === 0) {
        message.warning('没有可导出的数据');
        return;
      }
      const headers = [
        'code',
        'legal_name',
        'unified_social_credit_code',
        'registered_address',
        'enabled',
        'roles',
      ];
      const escapeCsvCell = (val: unknown): string => {
        if (val === null || val === undefined) return '';
        const str = Array.isArray(val) ? val.join(';') : String(val);
        if (
          str.includes(',') ||
          str.includes('"') ||
          str.includes('\n') ||
          str.includes('\r')
        ) {
          return `"${str.replace(/"/g, '""')}"`;
        }
        return str;
      };
      const rows = data.map((item) => [
        escapeCsvCell(item.code ?? ''),
        escapeCsvCell(item.legalName ?? ''),
        escapeCsvCell(item.unifiedSocialCreditCode ?? ''),
        escapeCsvCell(item.registeredAddress ?? ''),
        escapeCsvCell(item.enabled ?? false),
        escapeCsvCell(item.roles ?? []),
      ]);
      const csvContent =
        '\uFEFF' +
        [headers.join(','), ...rows.map((row) => row.join(','))].join('\r\n');
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'partners.csv');
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      message.success(`成功导出 ${data.length} 条数据`);
    } catch (err: any) {
      message.error(err?.message || '导出失败');
    } finally {
      setExporting(false);
    }
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
      title: '单位编码',
      dataIndex: 'code',
      width: 140,
      fixed: 'left',
      copyable: true,
      render: (code) => (
        <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>
          {code}
        </Text>
      ),
    },
    {
      title: '法人实体名称',
      dataIndex: 'legalName',
      width: 240,
      ellipsis: true,
      render: (name) => <Text strong>{name}</Text>,
    },
    {
      title: '业务角色身份',
      dataIndex: 'role',
      width: 240,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        roleOptions.map((option) => [option.value, { text: option.label }]),
      ),
      render: (_, record) => <Space wrap size={[4, 4]}>{roleTags(record.roles)}</Space>,
    },
    {
      title: '联系人',
      dataIndex: 'contacts',
      width: 100,
      search: false,
      render: (_, record) => {
        const count = record.contacts?.length ?? 0;
        return (
          <Tag bordered={false}>
            {count} 位
          </Tag>
        );
      },
    },
    {
      title: '常用别名',
      dataIndex: 'aliases',
      width: 90,
      search: false,
      render: (_, record) => {
        const count = record.aliases?.length ?? 0;
        return (
          <Tag bordered={false}>
            {count} 个
          </Tag>
        );
      },
    },
    {
      title: '统一社会信用代码',
      dataIndex: 'unifiedSocialCreditCode',
      width: 200,
      search: false,
      copyable: true,
      render: (code) =>
        code ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {code}
          </Text>
        ) : (
          <Text type="secondary">-</Text>
        ),
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
        record.enabled ? (
          <Tag color="success">启用</Tag>
        ) : (
          <Tag color="default">停用</Tag>
        ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space size={8}>
          <Button
            type="link"
            size="small"
            icon={<FolderOpenOutlined />}
            style={{ padding: 0 }}
            onClick={() => setSecondaryPartner(record)}
          >
            账户/合同
          </Button>
          {access.canManagePartners && (
            <>
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                style={{ padding: 0 }}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
              {record.roles?.some((role) => role.type === 2) && (
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<StopOutlined />}
                  style={{ padding: 0 }}
                  onClick={() => openBlacklist(record)}
                >
                  黑名单
                </Button>
              )}
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      title="往来单位"
      subTitle="统一维护客户、供应商、车队、船东及海外代理档案，管理银行账户、合同条款与黑名单"
    >
      <ProTable<API.Partner>
        headerTitle={
          <Space size={8}>
            <ContactsOutlined style={{ color: '#1677ff' }} />
            <span>往来单位档案列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1300 }}
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
          <Button
            key="export"
            icon={<ExportOutlined />}
            loading={exporting}
            onClick={handleExport}
          >
            导出 CSV
          </Button>,
          access.canManagePartners ? (
            <Button
              key="import"
              icon={<ImportOutlined />}
              onClick={openImport}
            >
              导入数据
            </Button>
          ) : null,
          access.canManagePartners ? (
            <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新增往来单位
            </Button>
          ) : null,
        ].filter(Boolean) as React.ReactNode[]}
      />

      <ModalForm<PartnerFormValues>
        title={editing ? `编辑往来单位：${editing.legalName} (${editing.code})` : '新增往来单位'}
        open={modalOpen}
        formRef={formRef}
        modalProps={{
          destroyOnClose: true,
          width: 780,
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
            message.success('往来单位已成功更新');
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
            message.success('往来单位已成功创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="单位唯一编码"
          placeholder="请输入组织内唯一编码（如 CUST001、SUPP001）"
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请输入往来单位编码' }]}
        />
        <ProFormText
          name="legalName"
          label="企业法人全称"
          placeholder="请输入完整企业法人营业执照名称"
          rules={[{ required: true, message: '请输入法人名称' }]}
        />
        <ProFormText name="unifiedSocialCreditCode" label="统一社会信用代码" placeholder="18位统一社会信用代码" />
        <ProFormTextArea name="registeredAddress" label="注册办公地址" fieldProps={{ rows: 2 }} placeholder="请输入企业法定注册地址" />
        <ProFormSwitch name="enabled" label="企业合作状态" initialValue />
        <ProFormList
          name="roles"
          label="业务角色身份分配"
          creatorButtonProps={{ creatorButtonText: '添加业务角色' }}
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
            <ProFormSwitch name="enabled" label="启用状态" initialValue />
          </Space>
        </ProFormList>
        <ProFormList
          name="contacts"
          label="联系人通讯录"
          creatorButtonProps={{ creatorButtonText: '添加联系人' }}
        >
          <Space align="start" wrap>
            <ProFormText name="name" label="姓名" rules={[{ required: true, message: '请输入联系人姓名' }]} />
            <ProFormText name="phone" label="联系电话" />
            <ProFormText name="email" label="电子邮箱" rules={[{ type: 'email', message: '请输入正确的邮箱地址' }]} />
            <ProFormSwitch name="isPrimary" label="主要联系人" />
            <ProFormText name="note" label="职务/备注" />
          </Space>
        </ProFormList>
        <ProFormList
          name="aliases"
          label="常用企业简称 / 别名"
          creatorButtonProps={{ creatorButtonText: '添加别名' }}
        >
          <Space align="start">
            <ProFormText name="aliasName" label="简称/别名" rules={[{ required: true, message: '请输入别名' }]} />
            <ProFormDigit name="sortOrder" label="排序权重" min={0} fieldProps={{ precision: 0 }} />
          </Space>
        </ProFormList>
      </ModalForm>

      <ModalForm<PartnerImportFormValues>
        title="批量导入往来单位数据"
        open={importModalOpen}
        formRef={importFormRef}
        modalProps={{
          destroyOnClose: true,
          width: 700,
          onCancel: () => setImportModalOpen(false),
        }}
        initialValues={{
          source: 'manual',
          mode: 1,
        }}
        onOpenChange={setImportModalOpen}
        onFinish={async (values) => {
          const source = values.source?.trim();
          if (!source) {
            message.error('请输入数据来源');
            return false;
          }

          let items: API.PartnerImportItemInput[];
          try {
            const raw = typeof values.items === 'string' ? values.items.trim() : '';
            if (!raw) {
              message.error('导入数据不能为空');
              return false;
            }
            const parsed = JSON.parse(raw);
            if (!Array.isArray(parsed)) {
              message.error('导入数据格式错误: 必须为 JSON 数组');
              return false;
            }
            if (parsed.length === 0) {
              message.error('导入数据不能为空数组');
              return false;
            }
            if (parsed.length > 500) {
              message.error(
                `导入数据超过限制: 最多支持 500 条，当前共 ${parsed.length} 条`,
              );
              return false;
            }
            items = parsed;
          } catch (err: any) {
            message.error(
              `JSON 解析失败: ${err?.message || '请检查 JSON 格式'}`,
            );
            return false;
          }

          let modeNumber = 1;
          if (values.mode === 2) {
            modeNumber = 2;
          }

          const response = await partnerServiceImportPartners({
            source,
            mode: modeNumber,
            items,
          });

          message.success(
            `导入成功: 新增 ${response.createdCount ?? 0} 条，更新 ${response.updatedCount ?? 0} 条`,
          );
          setImportModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="source"
          label="数据来源标识"
          placeholder="请输入数据来源，例如 ERP / EXCEL"
          rules={[{ required: true, message: '请输入数据来源' }]}
        />
        <ProFormRadio.Group
          name="mode"
          label="数据导入模式"
          options={[
            { label: '仅新增 (若存在则忽略或跳过)', value: 1 },
            { label: '存在则更新 (Upsert 按编码匹配覆盖)', value: 2 },
          ]}
          rules={[{ required: true, message: '请选择导入模式' }]}
        />
        <ProFormTextArea
          name="items"
          label="导入数据载荷 (JSON Array)"
          placeholder={`请输入 JSON 数组，例如：
[
  {
    "code": "CUST001",
    "legalName": "示例客户有限公司",
    "unifiedSocialCreditCode": "91310000XXXXXXXXXX",
    "registeredAddress": "上海市浦东新区...",
    "roles": [{ "type": 1, "enabled": true }]
  }
]`}
          rules={[{ required: true, message: '请输入导入数据 JSON' }]}
          fieldProps={{ rows: 10 }}
        />
      </ModalForm>

      <ModalForm<BlacklistFormValues>
        title={`供应商黑名单管理 - ${blacklistPartner?.legalName ?? ''}`}
        open={blacklistModalOpen}
        formRef={blacklistFormRef}
        initialValues={{
          blacklisted: Boolean(
            blacklistPartner?.roles?.find((role) => role.type === 2)?.blacklisted,
          ),
        }}
        modalProps={{
          destroyOnClose: true,
          width: 520,
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
        <ProFormSwitch name="blacklisted" label="列入供应商黑名单" />
        <ProFormTextArea
          name="reason"
          label="变更原因与说明"
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
    </PageContainer>
  );
}
