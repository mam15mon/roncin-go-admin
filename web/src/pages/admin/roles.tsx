import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Popconfirm, Space, Tag, Tooltip } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import {
  adminServiceDeleteRole,
  adminServiceListPermissions,
  adminServiceListRoles,
} from '@/services/roncin/adminService';
import { toTableRequest, unwrapList } from '@/utils/api';
import {
  buildPermissionTree,
  filterPermissionTree,
} from './components/roles/permissionTree';
import RoleFormModal from './components/roles/RoleFormModal';
import {
  ROLE_BASE_PERMISSION_KEY,
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

  // Permission tree state inside modal
  const [selectedPermissionKeys, setSelectedPermissionKeys] = useState<
    string[]
  >([]);
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
    // 默认勾选基础权限，避免漏勾后该角色用户登录无任何可进页面。
    setSelectedPermissionKeys([ROLE_BASE_PERMISSION_KEY]);
    setPermissionKeyword('');
    setExpandedKeys(permissionTree.initialExpandedKeys);
    setAutoExpandParent(false);
    setModalOpen(true);
  };

  const openEdit = (role: API.AdminRole) => {
    setEditing(role);
    setSelectedPermissionKeys(role.permissionKeys ?? []);
    setPermissionKeyword('');
    setExpandedKeys(permissionTree.initialExpandedKeys);
    setAutoExpandParent(false);
    setModalOpen(true);
  };

  // 删除失败时由全局错误处理器展示后端中文原因（如角色已分配成员），
  // 这里只处理成功分支并刷新列表。
  const handleDeleteRole = async (role: API.AdminRole) => {
    if (!role.id) return;
    await adminServiceDeleteRole({ id: role.id });
    message.success(`角色「${role.name || role.code}」已删除`);
    actionRef.current?.reload();
  };

  const columns: ProColumns<API.AdminRole>[] = [
    {
      title: '角色名称',
      dataIndex: 'name',
      width: 200,
      render: (dom, r) => (
        <Space size={6}>
          <span style={{ fontWeight: 600, color: '#0f172a' }}>{dom}</span>
          {!r.enabled && (
            <Tag
              color="error"
              variant="filled"
              style={{ fontSize: 11, lineHeight: '18px' }}
            >
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
      title: '功能权限集',
      dataIndex: 'permissionKeys',
      render: (_, r) => {
        const keys = r.permissionKeys ?? [];
        if (keys.length === 0) {
          return (
            <span style={{ color: '#94a3b8', fontSize: 12 }}>暂无分配权限</span>
          );
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
                  <div
                    style={{ maxHeight: 200, overflowY: 'auto', padding: 4 }}
                  >
                    {keys.map((k) => (
                      <div
                        key={k}
                        style={{ fontFamily: 'monospace', fontSize: 11 }}
                      >
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
      width: 170,
      render: (_, role) => {
        const assignedCount = role.assignmentsCount ?? 0;
        const canDelete =
          access.canDeleteRoles &&
          canConfigureRoles &&
          role.code !== 'administrator';
        return [
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
          canDelete ? (
            <Popconfirm
              key="delete"
              title="删除角色"
              description={`确认删除角色「${role.name || role.code}」？删除后不可恢复。`}
              okText="确认删除"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              onConfirm={() => handleDeleteRole(role)}
            >
              {assignedCount > 0 ? (
                <Tooltip title="该角色已分配成员，请先移除后重试">
                  <span>
                    <Button
                      type="link"
                      size="small"
                      danger
                      disabled
                      icon={<DeleteOutlined />}
                    >
                      删除
                    </Button>
                  </span>
                </Tooltip>
              ) : (
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                >
                  删除
                </Button>
              )}
            </Popconfirm>
          ) : null,
        ];
      },
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
        permissions={permissions}
        allLeafKeys={permissionTree.allLeafKeys}
        allGroupKeys={permissionTree.allBranchKeys}
        filteredTreeData={filteredTreeData}
        requiresByPermission={permissionTree.requiresByPermission}
        permissionNameByKey={permissionTree.permissionNameByKey}
        selectedPermissionKeys={selectedPermissionKeys}
        setSelectedPermissionKeys={setSelectedPermissionKeys}
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
