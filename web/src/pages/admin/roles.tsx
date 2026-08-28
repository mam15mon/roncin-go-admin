import {
  AppstoreOutlined,
  CheckSquareOutlined,
  DownOutlined,
  EditOutlined,
  KeyOutlined,
  MinusSquareOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
  SettingOutlined,
  UpOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect, SearchFilterTemplate } from '@/components/ui';
import {
  App,
  Button,
  Col,
  Empty,
  Input,
  Row,
  Select,
  Space,
  Tag,
  Tooltip,
  Tree,
  Typography,
} from 'antd';
import type { DataNode } from 'antd/es/tree';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  adminServiceCreateRole,
  adminServiceListOrganizations,
  adminServiceListPermissions,
  adminServiceListRoles,
  adminServiceUpdateRole,
} from '@/services/roncin/adminService';

const { Text } = Typography;

const dataScopeOptions = [
  {
    label: '全部组织',
    value: 1,
    color: 'purple',
    description: '可跨越组织边界访问全平台业务与管理数据',
  },
  {
    label: '当前组织',
    value: 2,
    color: 'orange',
    description: '仅能访问用户当前所在组织的业务数据',
  },
  {
    label: '组织树',
    value: 3,
    color: 'cyan',
    description: '可访问当前组织及所有直属或深层下级组织的业务数据',
  },
  {
    label: '仅本人',
    value: 4,
    color: 'default',
    description: '仅能访问本人创建或直接参与指派的单据与业务数据',
  },
];

const dataScopeMap = new Map(dataScopeOptions.map((item) => [item.value, item]));

type RoleFormValues = {
  code?: string;
  name?: string;
  dataScope?: number;
  permissionKeys?: string[];
  enabled?: boolean;
};

type PermissionLeafNode = {
  key: string;
  title: string;
  name: string;
  group: string;
  description?: string;
  isLeaf: boolean;
};

type PermissionGroupNode = {
  key: string;
  title: string;
  groupName: string;
  isLeaf: boolean;
  children: PermissionLeafNode[];
};

export default function RolesPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminRole>();
  const [permissions, setPermissions] = useState<API.AdminPermission[]>([]);
  const [organizations, setOrganizations] = useState<
    API.AdminOrganization[]
  >([]);
  const [orderOrganizationAccesses, setOrderOrganizationAccesses] = useState<
    API.OrderOrganizationAccess[]
  >([]);

  // Permission tree state inside modal
  const [selectedPermissionKeys, setSelectedPermissionKeys] = useState<string[]>([]);
  const [permissionKeyword, setPermissionKeyword] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // Load all permissions
  useEffect(() => {
    adminServiceListPermissions().then((response) => setPermissions(response.data ?? []));
  }, []);

  useEffect(() => {
    adminServiceListOrganizations().then((response) => {
      setOrganizations(response.data ?? []);
    });
  }, []);

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
    () => new Map(organizations.map((item) => [item.id, item.name ?? item.code ?? item.id ?? ''])),
    [organizations],
  );

  // Construct permission tree by group
  const { fullTreeData, allGroupKeys, allLeafKeys } = useMemo(() => {
    const groupMap = new Map<string, API.AdminPermission[]>();
    const leafKeys: string[] = [];
    const groupKeys: string[] = [];

    for (const perm of permissions) {
      const group = perm.group || '通用权限';
      const arr = groupMap.get(group) ?? [];
      arr.push(perm);
      groupMap.set(group, arr);
      if (perm.key) leafKeys.push(perm.key);
    }

    const tree: PermissionGroupNode[] = [];
    for (const [groupName, perms] of groupMap.entries()) {
      const groupKey = `group:${groupName}`;
      groupKeys.push(groupKey);
      tree.push({
        key: groupKey,
        title: groupName,
        groupName,
        isLeaf: false,
        children: perms.map((p) => ({
          key: p.key ?? '',
          title: p.name ?? p.key ?? '',
          name: p.name ?? '',
          group: groupName,
          description: p.description,
          isLeaf: true,
        })),
      });
    }

    return { fullTreeData: tree, allGroupKeys: groupKeys, allLeafKeys: leafKeys };
  }, [permissions]);

  // Filter permission tree by keyword
  const filteredTreeData = useMemo(() => {
    const kw = permissionKeyword.trim().toLowerCase();
    if (!kw) return fullTreeData;

    const result: PermissionGroupNode[] = [];
    for (const groupNode of fullTreeData) {
      const groupMatches = groupNode.groupName.toLowerCase().includes(kw);
      const matchedChildren = groupNode.children.filter(
        (child) =>
          child.name.toLowerCase().includes(kw) ||
          child.key.toLowerCase().includes(kw) ||
          child.description?.toLowerCase().includes(kw),
      );

      if (groupMatches || matchedChildren.length > 0) {
        result.push({
          ...groupNode,
          children: groupMatches ? groupNode.children : matchedChildren,
        });
      }
    }
    return result;
  }, [fullTreeData, permissionKeyword]);

  const openCreate = () => {
    setEditing(undefined);
    setSelectedPermissionKeys([]);
    setOrderOrganizationAccesses([]);
    setPermissionKeyword('');
    setExpandedKeys(allGroupKeys);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (role: API.AdminRole) => {
    setEditing(role);
    setSelectedPermissionKeys(role.permissionKeys ?? []);
    setOrderOrganizationAccesses(role.orderOrganizationAccesses ?? []);
    setPermissionKeyword('');
    setExpandedKeys(allGroupKeys);
    setModalOpen(true);
  };

  // Permission selection handlers
  const handleSelectAll = () => {
    setSelectedPermissionKeys([...allLeafKeys]);
  };

  const handleClearAll = () => {
    setSelectedPermissionKeys([]);
  };

  const handleExpandAll = () => {
    setExpandedKeys(allGroupKeys);
  };

  const handleCollapseAll = () => {
    setExpandedKeys([]);
  };

  const handleTreeCheck = (checked: React.Key[] | { checked: React.Key[]; halfChecked: React.Key[] }) => {
    const checkedArr = Array.isArray(checked) ? checked : checked.checked;
    // Keep only leaf permission keys (filter out group nodes)
    const leafOnly = checkedArr
      .map(String)
      .filter((k) => !k.startsWith('group:'));
    setSelectedPermissionKeys(leafOnly);
  };

  const columns: ProColumns<API.AdminRole>[] = [
    {
      title: '角色编码',
      dataIndex: 'code',
      width: 180,
      render: (code) => (
        <Text copyable style={{ fontFamily: 'monospace', fontWeight: 500 }}>
          {code}
        </Text>
      ),
    },
    {
      title: '角色名称',
      dataIndex: 'name',
      width: 200,
      render: (text) => (
        <Space size={6}>
          <SettingOutlined style={{ color: '#1677ff' }} />
          <Text strong>{text}</Text>
        </Space>
      ),
    },
    {
      title: '数据访问范围',
      dataIndex: 'dataScope',
      width: 160,
      valueEnum: Object.fromEntries(
        dataScopeOptions.map((item) => [item.value, { text: item.label }]),
      ),
      render: (_, record) => {
        const item = dataScopeMap.get(record.dataScope ?? 0);
        if (!item) return <Tag>未设置</Tag>;
        return (
          <Tooltip title={item.description}>
            <Tag
              color={item.color}
              variant="filled"
              style={{ fontSize: 12, padding: '2px 8px', cursor: 'help' }}
            >
              {item.label}
            </Tag>
          </Tooltip>
        );
      },
    },
    {
      title: '已授权权限',
      dataIndex: 'permissionKeys',
      width: 140,
      search: false,
      render: (_, record) => {
        const count = record.permissionKeys?.length ?? 0;
        return (
          <Tag
            variant="filled"
            style={{
              margin: 0,
              backgroundColor: count > 0 ? '#e6f4ff' : '#fafafa',
              color: count > 0 ? '#1677ff' : 'rgba(0, 0, 0, 0.45)',
              fontSize: 12,
              padding: '2px 8px',
            }}
          >
            <KeyOutlined style={{ marginRight: 4 }} />
            {count} 项
          </Tag>
        );
      },
    },
    {
      title: '跨公司订单范围',
      dataIndex: 'orderOrganizationAccesses',
      width: 220,
      search: false,
      render: (_, record) => {
        const accesses = record.orderOrganizationAccesses ?? [];
        if (accesses.length === 0) return <Text type="secondary">仅当前公司</Text>;
        return (
          <Space size={[4, 4]} wrap>
            {accesses.map((access) => (
              <Tag color={access.writable ? 'blue' : 'default'} key={access.organizationId}>
                {organizationNames.get(access.organizationId) ?? access.organizationId}
                {access.writable ? '（可修改）' : '（仅查看）'}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
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
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          style={{ padding: 0 }}
          onClick={() => openEdit(record)}
        >
          编辑
        </Button>
      ),
    },
  ];

  const [searchParams, setSearchParams] = useState<{ keyword?: string; dataScope?: number }>({});

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
            options: dataScopeOptions.map((opt) => ({ label: opt.label, value: opt.value })),
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
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={openCreate}
            >
              新增角色
            </Button>
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
          let list = response.data ?? [];
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
          return { data: list, success: response.success ?? true };
        }}
        toolBarRender={false}
      />

      {/* Role Create/Edit Modal */}
      <ModalForm<RoleFormValues>
        title={editing ? `编辑角色：${editing.name} (${editing.code})` : '新增角色'}
        open={modalOpen}
        formRef={formRef}
        initialValues={editing ? { ...editing, permissionKeys: editing.permissionKeys } : { dataScope: 2, enabled: true }}
        modalProps={{
          destroyOnClose: true,
          width: 800,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          try {
            if (editing?.id) {
              await adminServiceUpdateRole(
                { id: editing.id },
                {
                  id: editing.id,
                  name: values.name?.trim() ?? '',
                  dataScope: values.dataScope ?? 2,
                  enabled: values.enabled ?? true,
                  permissionKeys: selectedPermissionKeys,
                  orderOrganizationAccesses,
                },
              );
              message.success('角色已成功更新');
            } else {
              await adminServiceCreateRole({
                code: values.code?.trim() ?? '',
                name: values.name?.trim() ?? '',
                dataScope: values.dataScope ?? 2,
                permissionKeys: selectedPermissionKeys,
                orderOrganizationAccesses,
              });
              message.success('角色已成功创建');
            }
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          } catch {
            message.error('保存角色失败，请重试');
            return false;
          }
        }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <ProFormText
              name="code"
              label="角色编码"
              placeholder="例如：ROLE_OPS_LEAD"
              disabled={Boolean(editing)}
              rules={[
                { required: true, message: '请输入角色编码' },
                { pattern: /^[A-Za-z0-9_-]+$/, message: '编码仅支持英文字母、数字、下划线及连字符' },
              ]}
            />
          </Col>
          <Col span={12}>
            <ProFormText
              name="name"
              label="角色名称"
              placeholder="例如：操作主管 / 财务专员"
              rules={[{ required: true, message: '请输入角色名称' }]}
            />
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={16}>
            <ProFormSearchableSelect
              name="dataScope"
              label="数据访问范围"
              options={dataScopeOptions.map((opt) => ({
                label: `${opt.label} —— ${opt.description}`,
                value: opt.value,
              }))}
              rules={[{ required: true, message: '请选择数据范围' }]}
            />
          </Col>
          <Col span={8}>
            {editing && (
              <ProFormSwitch
                name="enabled"
                label="角色状态"
                extra="停用后关联用户将失去此角色权限"
              />
            )}
          </Col>
        </Row>

        <div
          style={{
            marginTop: 8,
            marginBottom: 16,
            border: '1px solid #d9d9d9',
            borderRadius: 6,
            padding: 12,
          }}
        >
          <Text strong>跨公司订单范围</Text>
          <div style={{ marginTop: 8 }}>
            <Text type="secondary">指定公司订单默认仅查看；勾选可修改后，仍需同时拥有对应的订单操作权限。</Text>
          </div>
          <div style={{ marginTop: 12 }}>
            <Text>可查看的公司</Text>
            <Select
              allowClear
              mode="multiple"
              options={companyOptions}
              placeholder="不选择时仅可访问当前公司订单"
              style={{ display: 'block', width: '100%', marginTop: 4 }}
              value={orderOrganizationAccesses.map((access) => access.organizationId)}
              onChange={(organizationIds: string[]) => {
                setOrderOrganizationAccesses((previous) =>
                  organizationIds.map((organizationId) => ({
                    organizationId,
                    writable:
                      previous.find((access) => access.organizationId === organizationId)
                        ?.writable ?? false,
                  })),
                );
              }}
            />
          </div>
          <div style={{ marginTop: 12 }}>
            <Text>其中允许修改的公司</Text>
            <Select
              allowClear
              mode="multiple"
              options={companyOptions.filter((option) =>
                orderOrganizationAccesses.some(
                  (access) => access.organizationId === option.value,
                ),
              )}
              placeholder="不选择时跨公司订单均为仅查看"
              style={{ display: 'block', width: '100%', marginTop: 4 }}
              value={orderOrganizationAccesses
                .filter((access) => access.writable)
                .map((access) => access.organizationId)}
              onChange={(organizationIds: string[]) => {
                const writableOrganizationIDs = new Set(organizationIds);
                setOrderOrganizationAccesses((previous) =>
                  previous.map((access) => ({
                    ...access,
                    writable: writableOrganizationIDs.has(access.organizationId),
                  })),
                );
              }}
            />
          </div>
        </div>

        {/* Permission Configuration Tree Panel */}
        <div style={{ marginTop: 8 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 8,
            }}
          >
            <Space size={8}>
              <Text strong style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.88)' }}>
                功能权限配置
              </Text>
              <Tag color="blue" variant="filled">
                已选 {selectedPermissionKeys.length} / {allLeafKeys.length} 项
              </Tag>
            </Space>
            <Space size={6}>
              <Button size="small" icon={<CheckSquareOutlined />} onClick={handleSelectAll}>
                全选
              </Button>
              <Button size="small" icon={<MinusSquareOutlined />} onClick={handleClearAll}>
                清空
              </Button>
              <Button size="small" icon={<DownOutlined />} onClick={handleExpandAll}>
                展开全部
              </Button>
              <Button size="small" icon={<UpOutlined />} onClick={handleCollapseAll}>
                收起全部
              </Button>
            </Space>
          </div>

          <div
            style={{
              border: '1px solid #f0f0f0',
              borderRadius: 6,
              padding: 12,
              backgroundColor: '#fafafa',
            }}
          >
            <Input
              placeholder="搜索权限名称、权限码或说明..."
              prefix={<SearchOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />}
              allowClear
              size="small"
              value={permissionKeyword}
              onChange={(e) => {
                setPermissionKeyword(e.target.value);
                setAutoExpandParent(true);
                if (e.target.value.trim()) {
                  setExpandedKeys(allGroupKeys);
                }
              }}
              style={{ marginBottom: 10, backgroundColor: '#ffffff' }}
            />

            <div
              style={{
                maxHeight: 280,
                overflowY: 'auto',
                backgroundColor: '#ffffff',
                border: '1px solid #f1f5f9',
                borderRadius: 4,
                padding: '8px 10px',
              }}
            >
              {filteredTreeData.length > 0 ? (
                <Tree
                  checkable
                  blockNode
                  treeData={filteredTreeData as unknown as DataNode[]}
                  checkedKeys={selectedPermissionKeys}
                  expandedKeys={expandedKeys}
                  autoExpandParent={autoExpandParent}
                  onExpand={(keys) => {
                    setExpandedKeys(keys);
                    setAutoExpandParent(false);
                  }}
                  onCheck={handleTreeCheck}
                  titleRender={(nodeData) => {
                    const node = nodeData as unknown as (PermissionGroupNode | PermissionLeafNode);
                    if ('children' in node && Array.isArray(node.children)) {
                      // Group Node
                      const groupLeaves = node.children;
                      const checkedInGroup = groupLeaves.filter((c) =>
                        selectedPermissionKeys.includes(c.key),
                      ).length;
                      return (
                        <span
                          style={{
                            fontWeight: 600,
                            fontSize: 13,
                            color: '#1e293b',
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 6,
                          }}
                        >
                          <AppstoreOutlined style={{ color: '#1677ff', fontSize: 13 }} />
                          <span>{node.title}</span>
                          <span
                            style={{
                              fontSize: 11,
                              fontWeight: 400,
                              color: checkedInGroup > 0 ? '#1677ff' : '#94a3b8',
                            }}
                          >
                            ({checkedInGroup}/{groupLeaves.length})
                          </span>
                        </span>
                      );
                    }

                    // Leaf Permission Node
                    const leaf = node as PermissionLeafNode;
                    const kw = permissionKeyword.trim().toLowerCase();
                    const isMatched =
                      kw &&
                      (leaf.name.toLowerCase().includes(kw) ||
                        leaf.key.toLowerCase().includes(kw) ||
                        leaf.description?.toLowerCase().includes(kw));

                    return (
                      <span
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: 8,
                          fontSize: 12,
                          padding: '1px 0',
                        }}
                      >
                        <span
                          style={{
                            color: isMatched ? '#1677ff' : '#334155',
                            fontWeight: isMatched ? 600 : 400,
                          }}
                        >
                          {leaf.name}
                        </span>
                        <Tag
                          variant="filled"
                          style={{
                            margin: 0,
                            fontSize: 10,
                            lineHeight: '16px',
                            padding: '0 4px',
                            backgroundColor: '#f1f5f9',
                            color: '#64748b',
                            fontFamily: 'monospace',
                          }}
                        >
                          {leaf.key}
                        </Tag>
                        {leaf.description && (
                          <span
                            style={{
                              fontSize: 11,
                              color: '#94a3b8',
                              maxWidth: 240,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {leaf.description}
                          </span>
                        )}
                      </span>
                    );
                  }}
                />
              ) : (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={permissionKeyword ? '未找到匹配的权限项' : '暂无可用权限'}
                  style={{ margin: '20px 0' }}
                />
              )}
            </div>
          </div>
        </div>
      </ModalForm>
    </>
  );
}
