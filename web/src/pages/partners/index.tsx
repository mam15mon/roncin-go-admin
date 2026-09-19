import {
  ContactsOutlined,
  DownOutlined,
  EditOutlined,
  FileExcelOutlined,
  FolderOpenOutlined,
  ImportOutlined,
  PlusOutlined,
  ReloadOutlined,
  StopOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormSelect,
  ProFormSwitch,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { history, useAccess, useLocation } from '@umijs/max';
import {
  App,
  Button,
  Dropdown,
  type MenuProps,
  Space,
  Tag,
  Typography,
} from 'antd';
import React, { useRef, useState } from 'react';
import * as XLSX from 'xlsx';
import { SearchFilterTemplate } from '@/components/ui';
import { PartnerRoleType } from '@/enums.generated';
import {
  partnerServiceListPartners,
  partnerServiceSetPartnerRoleBlacklist,
} from '@/services/roncin/partnerService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import PartnerExcelImportModal from './components/PartnerExcelImportModal';
import RoleSwitchModal from './components/RoleSwitchModal';
import PartnerSecondary from './partner-secondary';

const { Text } = Typography;

const roleOptions = [
  {
    label: '客户',
    value: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
    color: 'blue',
  },
  {
    label: '供应商',
    value: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
    color: 'green',
  },
  {
    label: '国外代理',
    value: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
    color: 'purple',
  },
];

const currentViewMeta: Record<
  string,
  { roleType: PartnerRoleType; title: string; description: string }
> = {
  '/partners/customers': {
    roleType: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
    title: '客户',
    description: '维护客户企业档案、联系人、合同与结算资料',
  },
  '/partners/suppliers': {
    roleType: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
    title: '供应商',
    description: '维护供应商企业档案、联系人、合同与结算资料',
  },
  '/partners/foreign-agents': {
    roleType: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
    title: '国外代理',
    description: '维护国外代理企业档案、联系人、合同与结算资料',
  },
};

const roleMap = new Map<number, (typeof roleOptions)[number]>(
  roleOptions.map((option) => [option.value, option]),
);

const roleLabels: Record<number, string> = Object.fromEntries(
  roleOptions.map((option) => [option.value, option.label]),
);

type BlacklistFormValues = {
  roleType?: PartnerRoleType;
  blacklisted?: boolean;
  reason?: string;
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
  const { message } = App.useApp();
  const location = useLocation();
  const access = useAccess();
  const actionRef = useRef<ActionType>(null);
  const blacklistFormRef = useRef<
    ProFormInstance<BlacklistFormValues> | undefined
  >(undefined);

  const [blacklistModalOpen, setBlacklistModalOpen] = useState(false);
  const [blacklistPartner, setBlacklistPartner] = useState<API.Partner | null>(
    null,
  );
  const [secondaryPartner, setSecondaryPartner] = useState<
    API.Partner | undefined
  >(undefined);
  const [roleSwitchPartner, setRoleSwitchPartner] =
    useState<API.Partner | null>(null);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [exporting, setExporting] = useState(false);

  const currentView =
    currentViewMeta[location.pathname] ||
    currentViewMeta['/partners/customers'];
  const defaultBlacklistRoleType = blacklistPartner?.roles?.some(
    (role) => role.type === currentView.roleType,
  )
    ? currentView.roleType
    : blacklistPartner?.roles?.find((role) => roleMap.has(role.type ?? 0))
        ?.type;
  const defaultBlacklistRole = blacklistPartner?.roles?.find(
    (role) => role.type === defaultBlacklistRoleType,
  );

  const openCreate = () => {
    history.push(`${location.pathname}/create`);
  };

  const openImport = () => {
    setImportModalOpen(true);
  };

  const isCustomerView =
    currentView.roleType === PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER;

  const handleExport = async () => {
    try {
      setExporting(true);
      const response = await partnerServiceListPartners({
        page: 1,
        pageSize: 2000,
        role: currentView.roleType,
      });
      const data = unwrapList(response);
      if (data.length === 0) {
        message.warning('没有可导出的数据');
        return;
      }
      const headers = [
        '单位编码',
        '企业名称',
        '统一社会信用代码',
        '注册地址',
        '启用状态',
        '业务角色',
        '主要联系人',
        '更新时间',
      ];
      const formatRolesForExcel = (roles?: API.PartnerRole[]): string => {
        if (!roles || roles.length === 0) return '';
        return roles
          .map((r) => {
            const label = roleLabels[r.type ?? 0] ?? '未知';
            const statusSuffix = !r.enabled
              ? '(停用)'
              : r.blacklisted
                ? '(黑名单)'
                : '';
            return `${label}${statusSuffix}`;
          })
          .join(';');
      };
      const rows = data.map((item) => [
        item.code ?? '',
        item.legalName ?? '',
        item.unifiedSocialCreditCode ?? '',
        item.registeredAddress ?? '',
        item.enabled ? '启用' : '停用',
        formatRolesForExcel(item.roles),
        (item.contacts ?? [])
          .map((c) => `${c.name || ''}${c.phone ? `(${c.phone})` : ''}`)
          .filter(Boolean)
          .join('; ') || '',
        item.updatedAt ? new Date(item.updatedAt).toLocaleString() : '',
      ]);

      const ws = XLSX.utils.aoa_to_sheet([headers, ...rows]);
      ws['!cols'] = [
        { wch: 14 },
        { wch: 28 },
        { wch: 22 },
        { wch: 32 },
        { wch: 10 },
        { wch: 20 },
        { wch: 24 },
        { wch: 20 },
      ];
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, `${currentView.title}档案`);
      XLSX.writeFile(wb, `${currentView.title}档案列表.xlsx`);
      message.success(`成功导出 ${data.length} 条数据至 Excel`);
    } catch (err) {
      message.error(getErrorMessage(err, '导出 Excel 失败'));
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
        <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>{code}</Text>
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
      render: (_, record) => (
        <Space wrap size={[4, 4]}>
          {roleTags(record.roles)}
        </Space>
      ),
    },
    {
      title: '合作类型',
      dataIndex: 'isCasual',
      width: 100,
      search: false,
      hideInTable: !isCustomerView,
      render: (_, record) =>
        record.isCasual ? (
          <Tag color="warning">散客</Tag>
        ) : (
          <Tag color="default">正式</Tag>
        ),
    },
    {
      title: '联系人',
      dataIndex: 'contacts',
      width: 100,
      search: false,
      render: (_, record) => {
        const count = record.contacts?.length ?? 0;
        return <Tag variant="filled">{count} 位</Tag>;
      },
    },
    {
      title: '常用别名',
      dataIndex: 'aliases',
      width: 90,
      search: false,
      render: (_, record) => {
        const count = record.aliases?.length ?? 0;
        return <Tag variant="filled">{count} 个</Tag>;
      },
    },
    {
      title: '统一社会信用代码',
      dataIndex: 'unifiedSocialCreditCode',
      width: 200,
      search: false,
      copyable: true,
      hideInTable:
        currentView.roleType ===
        PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
      render: (code) =>
        code ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{code}</Text>
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
      width: 120,
      fixed: 'right',
      render: (_, record) => {
        const moreItems: MenuProps['items'] = [];

        moreItems.push({
          key: 'secondary',
          icon: <FolderOpenOutlined />,
          label: '账户/合同',
          onClick: () => setSecondaryPartner(record),
        });

        if (access.canManagePartners || access.canUpdatePartners) {
          moreItems.push({
            key: 'role-switch',
            icon: <SwapOutlined />,
            label: '转角色',
            onClick: () => setRoleSwitchPartner(record),
          });
        }

        if (
          (access.canManagePartners || access.canBlacklistPartners) &&
          record.roles?.some((role) => roleMap.has(role.type ?? 0))
        ) {
          moreItems.push({
            type: 'divider',
          });
          moreItems.push({
            key: 'blacklist',
            icon: <StopOutlined />,
            label: '黑名单',
            danger: true,
            onClick: () => openBlacklist(record),
          });
        }

        return (
          <Space size={8}>
            {(access.canManagePartners || access.canUpdatePartners) && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                style={{ padding: 0 }}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
            {moreItems.length > 0 && (
              <Dropdown menu={{ items: moreItems }} trigger={['click']}>
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0 }}
                  onClick={(e) => e.preventDefault()}
                >
                  更多 <DownOutlined style={{ fontSize: 10 }} />
                </Button>
              </Dropdown>
            )}
          </Space>
        );
      },
    },
  ];

  const [searchParams, setSearchParams] = useState<{
    keyword?: string;
    enabled?: boolean;
    isCasual?: boolean;
  }>({});

  return (
    <PageContainer
      title={false}
      breadcrumbRender={false}
      header={{
        title: false,
        breadcrumb: undefined,
        style: { padding: 0 },
      }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder={`搜索单位编码、名称、拼音或税号...`}
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
          ...(isCustomerView
            ? [
                {
                  name: 'isCasual',
                  placeholder: '合作类型',
                  width: 120,
                  options: [
                    { label: '正式伙伴', value: false },
                    { label: '散客', value: true },
                  ],
                },
              ]
            : []),
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
            {(access.canManagePartners || access.canExportPartners) && (
              <Button
                key="export"
                icon={<FileExcelOutlined />}
                loading={exporting}
                onClick={handleExport}
              >
                导出 Excel
              </Button>
            )}
            {(access.canManagePartners || access.canImportPartners) && (
              <Button
                key="import"
                icon={<ImportOutlined />}
                onClick={openImport}
              >
                导入 Excel
              </Button>
            )}
            {(access.canManagePartners || access.canCreatePartners) && (
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={openCreate}
              >
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
        cardProps={{
          style: {
            borderRadius: 8,
            border: '1px solid #f0f0f0',
          },
        }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1480 }}
        request={async (params) => {
          const response = await partnerServiceListPartners({
            page: params.current,
            pageSize: params.pageSize,
            keyword: searchParams.keyword,
            role: currentView.roleType,
            enabled: searchParams.enabled,
            isCasual: searchParams.isCasual,
          });
          return toTableRequest(response);
        }}
        search={false}
        toolBarRender={false}
      />

      <PartnerExcelImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onSuccess={() => actionRef.current?.reload()}
        currentRoleType={currentView.roleType}
        currentRoleLabel={currentView.title}
      />

      <RoleSwitchModal
        open={Boolean(roleSwitchPartner)}
        partner={roleSwitchPartner}
        onClose={() => setRoleSwitchPartner(null)}
        onSuccess={() => actionRef.current?.reload()}
      />

      <ModalForm<BlacklistFormValues>
        title={`角色黑名单管理 - ${blacklistPartner?.legalName ?? ''}`}
        open={blacklistModalOpen}
        formRef={blacklistFormRef}
        initialValues={{
          roleType: defaultBlacklistRoleType,
          blacklisted: Boolean(defaultBlacklistRole?.blacklisted),
        }}
        modalProps={{
          destroyOnClose: true,
          width: 520,
          onCancel: () => setBlacklistModalOpen(false),
        }}
        onOpenChange={setBlacklistModalOpen}
        onValuesChange={(changedValues) => {
          if (changedValues.roleType === undefined) return;
          const role = blacklistPartner?.roles?.find(
            (item) => item.type === changedValues.roleType,
          );
          blacklistFormRef.current?.setFieldValue(
            'blacklisted',
            Boolean(role?.blacklisted),
          );
        }}
        onFinish={async (values) => {
          if (!blacklistPartner?.id) return false;
          if (!values.roleType) return false;
          await partnerServiceSetPartnerRoleBlacklist(
            { id: blacklistPartner.id },
            {
              id: blacklistPartner.id,
              roleType: values.roleType,
              blacklisted: values.blacklisted ?? false,
              reason: values.reason?.trim() ?? '',
            },
          );
          message.success(
            values.blacklisted
              ? `已将${roleLabels[values.roleType]}角色加入黑名单`
              : `已将${roleLabels[values.roleType]}角色移出黑名单`,
          );
          setBlacklistModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="roleType"
          label="目标角色"
          options={(blacklistPartner?.roles ?? [])
            .filter((role) => roleMap.has(role.type ?? 0))
            .map((role) => ({
              label: roleLabels[role.type ?? 0],
              value: role.type,
            }))}
          rules={[{ required: true, message: '请选择目标角色' }]}
        />
        <ProFormSwitch name="blacklisted" label="列入该角色黑名单" />
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
        canReadAccounts={access.canReadPartnerAccounts}
        canCreateAccounts={access.canCreatePartnerAccounts}
        canUpdateAccounts={access.canUpdatePartnerAccounts}
        canManage={access.canManagePartners}
        onClose={() => setSecondaryPartner(undefined)}
      />
    </PageContainer>
  );
}
