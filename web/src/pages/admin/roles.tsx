import {
  EditOutlined,
  KeyOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { SearchFilterTemplate } from '@/components/ui';
import {
  App,
  Button,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import { useAccess } from '@umijs/max';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  adminServiceListOrganizations,
  adminServiceListPermissions,
  adminServiceListRoles,
} from '@/services/roncin/adminService';
import { toTableRequest, unwrapList } from '@/utils/api';
import RoleFormModal from './components/roles/RoleFormModal';
import {
  buildPermissionTree,
  filterPermissionTree,
} from './components/roles/permissionTree';
import {
  type OrderOrganizationAccess,
  dataScopeMap,
  dataScopeOptions,
} from './components/roles/roleConstants';

export default function RolesPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminRole>();
  const [permissions, setPermissions] = useState<API.AdminPermission[]>([]);
  const [organizations, setOrganizations] = useState<
    API.AdminOrganization[]
  >([]);
  const [orderOrganizationAccesses, setOrderOrganizationAccesses] = useState<
    OrderOrganizationAccess[]
  >([]);

  // Permission tree state inside modal
  const [selectedPermissionKeys, setSelectedPermissionKeys] = useState<string[]>([]);
  const [permissionKeyword, setPermissionKeyword] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);
  const canConfigureRoles =
    access.canReadPermissions && access.canReadOrganizations;
  const [searchParams, setSearchParams] = useState<{
    keyword?: string;
    dataScope?: number;
  }>({});

  // Load all permissions
  useEffect(() => {
    if (!access.canReadPermissions) return;
    adminServiceListPermissions()
      .then((response) => setPermissions(unwrapList(response)))
      .catch(() => message.error('加载权限字典失败'));
  }, [access.canReadPermissions, message]);

  useEffect(() => {
    if (!access.canReadOrganizations) return;
    adminServiceListOrganizations()
      .then((response) => setOrganizations(unwrapList(response)))
      .catch(() => message.error('加载组织列表失败'));
  }, [access.canReadOrganizations, message]);

  const companyOptions = useMemo(
    () =>
      organizations
        .filter(
          (organization) =>
            organization.enabled !== false && organization.kind === 2,
        )
        .map((organization) => ({
          label: organization.code
            ? `${organization.name} (${organization.code})`
            : organization.name ?? '',
          value: organization.id ?? '',
        })),
    [organizations],
  );

  const organizationNames = useMemo(
    () =>
      new Map(
        organizations.map((item) => [
          item.id,
          item.name ?? item.code ?? item.id ?? '',
        ]),
      ),
    [organizations],
  );

  const permissionTree = useMemo(
    () => buildPermissionTree(permissions),
    [permissions],
  );

  const filteredTreeData = useMemo(
    () => filterPermissionTree(permissionTree.treeData, permissionKeyword),
    [permissionKeyword, permissionTree.treeData],
  );

  const openCreate = () => {
    setEditing(undefined);
    setSelectedPermissionKeys([]);
    setOrderOrganizationAccesses([]);
    setPermissionKeyword('');
    setExpandedKeys(permissionTree.initialExpandedKeys);
    setAutoExpandParent(false);
    setModalOpen(true);
  };

  const openEdit = (role: API.AdminRole) => {
    setEditing(role);
    setSelectedPermissionKeys(role.permissionKeys ?? []);
    setOrderOrganizationAccesses(
      (role.orderOrganizationAccesses ?? []).map((access) => ({
        organizationId: access.organizationId as string,
        writable: access.writable ?? false,
      })),
    );
    setPermissionKeyword('');
    setExpandedKeys(permissionTree.initialExpandedKeys);
    setAutoExpandParent(false);
    setModalOpen(true);
  };

  const columns: ProColumns<API.AdminRole>[] = [
    {
      title: '角色标识 / 编码',
      dataIndex: 'code',
      width: 220,
      render: (_, r) => (
        <Space size={8}>
          <KeyOutlined style={{ color: '#1677ff', fontSize: 13 }} />
          <span style={{ fontWeight: 600, fontFamily: 'monospace', color: '#1e293b' }}>
            {r.code}
          </span>
        </Space>
      ),
    },
    {
      title: '角色名称',
      dataIndex: 'name',
      width: 200,
      render: (dom, r) => (
        <Space size={6}>
          <span style={{ fontWeight: 600, color: '#0f172a' }}>{dom}</span>
          {!r.enabled && (
            <Tag color="error" variant="filled" style={{ fontSize: 11, lineHeight: '18px' }}>
              已禁用
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: '数据访问范围',
      dataIndex: 'dataScope',
      width: 140,
      render: (_, r) => {
        const item = dataScopeMap.get(r.dataScope ?? 2);
        return (
          <Tooltip title={item?.description}>
            <Tag color={item?.color} variant="filled">
              {item?.label || `范围 ${r.dataScope}`}
            </Tag>
          </Tooltip>
        );
      },
    },
    {
      title: '跨公司订单范围',
      dataIndex: 'orderOrganizationAccesses',
      width: 260,
      render: (_, r) => {
        const accesses = r.orderOrganizationAccesses ?? [];
        if (accesses.length === 0) {
          return <span style={{ color: '#94a3b8', fontSize: 12 }}>仅当前公司</span>;
        }
        return (
          <Space size={4} wrap>
            {accesses.map((access) => {
              const label =
                organizationNames.get(access.organizationId ?? '') ||
                access.organizationId;
              return (
                <Tag
                  key={access.organizationId}
                  color={access.writable ? 'blue' : 'default'}
                  variant="filled"
                  style={{ fontSize: 11, margin: 0 }}
                >
                  {label} {access.writable ? '（可改）' : '（仅看）'}
                </Tag>
              );
            })}
          </Space>
        );
      },
    },
    {
      title: '功能权限集',
      dataIndex: 'permissionKeys',
      render: (_, r) => {
        const keys = r.permissionKeys ?? [];
        if (keys.length === 0) {
          return <span style={{ color: '#94a3b8', fontSize: 12 }}>暂无分配权限</span>;
        }
        return (
          <Space size={4} wrap>
            <Tag color="geekblue" variant="filled" style={{ fontWeight: 600 }}>
              共 {keys.length} 项
            </Tag>
            {keys.slice(0, 4).map((k) => (
              <Tag
                key={k}
                style={{
                  margin: 0,
                  fontSize: 11,
                  fontFamily: 'monospace',
                  backgroundColor: '#f8fafc',
                  border: '1px solid #e2e8f0',
                  color: '#475569',
                }}
              >
                {k}
              </Tag>
            ))}
            {keys.length > 4 && (
              <Tooltip
                title={
                  <div style={{ maxHeight: 200, overflowY: 'auto', padding: 4 }}>
                    {keys.map((k) => (
                      <div key={k} style={{ fontFamily: 'monospace', fontSize: 11 }}>
                        {k}
                      </div>
                    ))}
                  </div>
                }
              >
                <Tag style={{ margin: 0, fontSize: 11, cursor: 'pointer' }}>
                  +{keys.length - 4}...
                </Tag>
              </Tooltip>
            )}
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      render: (_, r) =>
        r.enabled ? (
          <Tag color="success" variant="filled">
            正常
          </Tag>
        ) : (
          <Tag color="error" variant="filled">
            禁用
          </Tag>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 120,
      render: (_, role) => [
        access.canUpdateRoles && canConfigureRoles ? (
          <Button
            key="edit"
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEdit(role)}
          >
            编辑
          </Button>
        ) : null,
      ],
    },
  ];

  return (
    <>
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder="搜索角色名称或角色编码..."
        quickFilters={[
          {
            name: 'dataScope',
            placeholder: '全部数据范围',
            width: 140,
            options: dataScopeOptions.map((opt) => ({
              label: opt.label,
              value: opt.value,
            })),
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
            {access.canCreateRoles && canConfigureRoles && (
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={openCreate}
              >
                新增角色
              </Button>
            )}
          </Space>
        }
      />

      <ProTable<API.AdminRole>
        headerTitle={
          <Space size={8}>
            <SafetyCertificateOutlined style={{ color: '#1677ff' }} />
            <span>角色与权限方案列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        search={false}
        pagination={false}
        request={async () => {
          const response = await adminServiceListRoles();
          let list = unwrapList(response);
          if (searchParams.keyword) {
            const kw = searchParams.keyword.toLowerCase().trim();
            list = list.filter(
              (r) =>
                r.name?.toLowerCase().includes(kw) ||
                r.code?.toLowerCase().includes(kw),
            );
          }
          if (searchParams.dataScope !== undefined) {
            list = list.filter((r) => r.dataScope === searchParams.dataScope);
          }
          return toTableRequest({ ...response, data: list });
        }}
        toolBarRender={false}
      />

      {/* Role Create/Edit Modal */}
      <RoleFormModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        editing={editing}
        formRef={formRef}
        companyOptions={companyOptions}
        allLeafKeys={permissionTree.allLeafKeys}
        allGroupKeys={permissionTree.allBranchKeys}
        filteredTreeData={filteredTreeData}
        requiresByPermission={permissionTree.requiresByPermission}
        permissionNameByKey={permissionTree.permissionNameByKey}
        selectedPermissionKeys={selectedPermissionKeys}
        setSelectedPermissionKeys={setSelectedPermissionKeys}
        orderOrganizationAccesses={orderOrganizationAccesses}
        setOrderOrganizationAccesses={setOrderOrganizationAccesses}
        expandedKeys={expandedKeys}
        setExpandedKeys={setExpandedKeys}
        autoExpandParent={autoExpandParent}
        setAutoExpandParent={setAutoExpandParent}
        permissionKeyword={permissionKeyword}
        setPermissionKeyword={setPermissionKeyword}
        onSuccess={() => actionRef.current?.reload()}
      />
    </>
  );
}
