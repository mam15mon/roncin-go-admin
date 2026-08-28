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
  ProFormRadio,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { history, useAccess, useLocation } from '@umijs/max';
import { App, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import {
  partnerServiceExportPartners,
  partnerServiceImportPartners,
  partnerServiceListPartners,
  partnerServiceSetSupplierBlacklist,
} from '@/services/roncin/partnerService';
import PartnerSecondary from './partner-secondary';

const { Text } = Typography;

const roleOptions = [
  { label: '客户', value: 1, color: 'blue' },
  { label: '供应商', value: 2, color: 'green' },
  { label: '国外代理', value: 3, color: 'purple' },
  { label: '承运人', value: 4, color: 'orange' },
];

const partnerViews: Record<
  string,
  { title: string; roleType: number; codeExample: string; description: string }
> = {
  '/partners/customers': {
    title: '客户',
    roleType: 1,
    codeExample: 'CUST001',
    description: '维护客户企业档案、联系人、合同与结算资料',
  },
  '/partners/suppliers': {
    title: '供应商',
    roleType: 2,
    codeExample: 'SUPP001',
    description: '维护供应商企业档案、联系人、合同与黑名单',
  },
  '/partners/foreign-agents': {
    title: '国外代理',
    roleType: 3,
    codeExample: 'AGENT001',
    description: '维护国外代理企业档案、联系人、合同与结算资料',
  },
};

const roleMap = new Map(roleOptions.map((opt) => [opt.value, opt]));

const roleLabels: Record<number, string> = Object.fromEntries(
  roleOptions.map((option) => [option.value, option.label]),
);

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
      <Tag key={role.type} color={color} variant="filled">
        {roleLabels[role.type ?? 0] ?? '未知'}
        {!role.enabled ? ' (停用)' : ''}
        {role.blacklisted ? ' [黑名单]' : ''}
      </Tag>
    );
  });
}

export default function Partners() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const blacklistFormRef = useRef<ProFormInstance | undefined>(undefined);
  const importFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const location = useLocation();
  const currentView = partnerViews[location.pathname] ?? partnerViews['/partners/customers'];
  const [blacklistModalOpen, setBlacklistModalOpen] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [blacklistPartner, setBlacklistPartner] = useState<API.Partner>();
  const [secondaryPartner, setSecondaryPartner] = useState<API.Partner>();

  const openCreate = () => {
    history.push(`${location.pathname}/create`);
  };

  const openImport = () => {
    importFormRef.current?.resetFields();
    setImportModalOpen(true);
  };

  const handleExport = async () => {
    try {
      setExporting(true);
      const response = await partnerServiceExportPartners({ role: currentView.roleType });
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
      link.setAttribute('download', `${currentView.title}.csv`);
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
    history.push(`${location.pathname}/${partner.id}`);
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
      search: false,
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
          <Tag variant="filled">
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
          <Tag variant="filled">
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

  const [searchParams, setSearchParams] = useState<{ keyword?: string; enabled?: boolean }>({});

  return (
    <PageContainer
      title={currentView.title}
      subTitle={currentView.description}
    >
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder={`搜索${currentView.title}代码、名称、拼音或税号...`}
        quickFilters={[
          {
            name: 'enabled',
            placeholder: '全部状态',
            width: 120,
            options: [
              { label: '启用', value: true },
              { label: '停用', value: false },
            ],
          },
        ]}
        onSearch={(values) => {
          setSearchParams(values);
          actionRef.current?.reload();
        }}
        onReset={() => {
          setSearchParams({});
          actionRef.current?.reload();
        }}
        extraRight={
          <Space size={8}>
            <Button
              key="refresh"
              icon={<ReloadOutlined />}
              onClick={() => actionRef.current?.reload()}
            >
              刷新
            </Button>
            <Button
              key="export"
              icon={<ExportOutlined />}
              loading={exporting}
              onClick={handleExport}
            >
              导出 CSV
            </Button>
            {access.canManagePartners && (
              <Button
                key="import"
                icon={<ImportOutlined />}
                onClick={openImport}
              >
                导入数据
              </Button>
            )}
            {access.canManagePartners && (
              <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                新增{currentView.title}
              </Button>
            )}
          </Space>
        }
      />
      <ProTable<API.Partner>
        key={location.pathname}
        headerTitle={
          <Space size={8}>
            <ContactsOutlined style={{ color: '#1677ff' }} />
            <span>{currentView.title}档案列表</span>
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
            keyword: searchParams.keyword,
            role: currentView.roleType,
            enabled: searchParams.enabled,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
            total: response.total ?? 0,
          };
        }}
        search={false}
        toolBarRender={false}
      />

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
