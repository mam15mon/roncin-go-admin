import {
  ApartmentOutlined,
  EditOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  InfoCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Input,
  Row,
  Space,
  Spin,
  Table,
  Tag,
  Tree,
  Typography,
} from 'antd';
import type { DataNode } from 'antd/es/tree';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  adminServiceCreateOrganization,
  adminServiceListOrganizations,
  adminServiceUpdateOrganization,
} from '@/services/roncin/adminService';

const { Text } = Typography;

type OrgNode = {
  key: string;
  title: string;
  code: string;
  enabled: boolean;
  raw: API.AdminOrganization;
  children?: OrgNode[];
};

type CreateFormValues = {
  code?: string;
  name?: string;
};

type EditFormValues = {
  name?: string;
  enabled?: boolean;
};

export default function OrganizationsPanel() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>(
    [],
  );
  const [selectedId, setSelectedId] = useState<string>('');
  const [searchKeyword, setSearchKeyword] = useState<string>('');
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // Modal states
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [parentForCreate, setParentForCreate] =
    useState<API.AdminOrganization | null>(null);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingOrg, setEditingOrg] = useState<API.AdminOrganization | null>(
    null,
  );

  const createFormRef = useRef<ProFormInstance | undefined>(undefined);
  const editFormRef = useRef<ProFormInstance | undefined>(undefined);

  // Load data
  const loadData = useCallback(
    async (selectIdAfterLoad?: string) => {
      setLoading(true);
      try {
        const response = await adminServiceListOrganizations();
        const list = response.data ?? [];
        setOrganizations(list);
        if (selectIdAfterLoad) {
          setSelectedId(selectIdAfterLoad);
        } else if (list.length > 0) {
          setSelectedId((prev) => {
            if (prev && list.some((item) => item.id === prev)) return prev;
            return list[0].id ?? '';
          });
        } else {
          setSelectedId('');
        }
      } catch {
        message.error('加载组织列表失败');
      } finally {
        setLoading(false);
      }
    },
    [message],
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Index map of organizations
  const orgMap = useMemo(() => {
    const map = new Map<string, API.AdminOrganization>();
    for (const org of organizations) {
      if (org.id) map.set(org.id, org);
    }
    return map;
  }, [organizations]);

  // Build tree data
  const { treeData, allKeys } = useMemo(() => {
    const childrenMap = new Map<string, API.AdminOrganization[]>();
    const roots: API.AdminOrganization[] = [];

    for (const org of organizations) {
      const pId = org.parentId;
      if (!pId || !orgMap.has(pId)) {
        roots.push(org);
      } else {
        const arr = childrenMap.get(pId) ?? [];
        arr.push(org);
        childrenMap.set(pId, arr);
      }
    }

    const keys: string[] = [];
    const buildNodes = (list: API.AdminOrganization[]): OrgNode[] => {
      return list.map((item) => {
        const id = item.id ?? '';
        keys.push(id);
        const children = childrenMap.get(id);
        return {
          key: id,
          title: item.name ?? '',
          code: item.code ?? '',
          enabled: item.enabled ?? true,
          raw: item,
          children:
            children && children.length > 0 ? buildNodes(children) : undefined,
        };
      });
    };

    return { treeData: buildNodes(roots), allKeys: keys };
  }, [organizations, orgMap]);

  // Set default expanded keys when tree is first loaded
  useEffect(() => {
    if (expandedKeys.length === 0 && allKeys.length > 0) {
      setExpandedKeys(allKeys);
    }
  }, [allKeys, expandedKeys.length]);

  // Currently selected organization
  const selectedOrg = useMemo(() => {
    if (!selectedId) return null;
    return orgMap.get(selectedId) ?? null;
  }, [selectedId, orgMap]);

  // Direct sub-organizations of selected organization
  const directChildren = useMemo(() => {
    if (!selectedId) return [];
    return organizations.filter((org) => org.parentId === selectedId);
  }, [selectedId, organizations]);

  // Recursive count of all descendants
  const totalDescendantCount = useMemo(() => {
    if (!selectedId) return 0;
    let count = 0;
    const stack = [...directChildren];
    while (stack.length > 0) {
      const current = stack.pop();
      if (!current?.id) continue;
      count += 1;
      const sub = organizations.filter((org) => org.parentId === current.id);
      stack.push(...sub);
    }
    return count;
  }, [selectedId, directChildren, organizations]);

  // Filtered tree data & search highlight
  const filteredTreeData = useMemo(() => {
    const kw = searchKeyword.trim().toLowerCase();
    if (!kw) return treeData;

    const filterNode = (node: OrgNode): OrgNode | null => {
      const matchesSelf =
        node.title.toLowerCase().includes(kw) ||
        node.code.toLowerCase().includes(kw);
      const filteredChildren = node.children
        ?.map(filterNode)
        .filter((child): child is OrgNode => Boolean(child));

      if (matchesSelf || (filteredChildren && filteredChildren.length > 0)) {
        return {
          ...node,
          children:
            filteredChildren && filteredChildren.length > 0
              ? filteredChildren
              : undefined,
        };
      }
      return null;
    };

    return treeData
      .map(filterNode)
      .filter((node): node is OrgNode => Boolean(node));
  }, [treeData, searchKeyword]);

  // When searching, expand all matched paths
  const handleSearch = (value: string) => {
    setSearchKeyword(value);
    setAutoExpandParent(true);
    if (!value.trim()) return;

    const kw = value.trim().toLowerCase();
    const matchedKeys: string[] = [];
    for (const org of organizations) {
      if (
        org.name?.toLowerCase().includes(kw) ||
        org.code?.toLowerCase().includes(kw)
      ) {
        if (org.id) matchedKeys.push(org.id);
      }
    }
    setExpandedKeys((prev) => Array.from(new Set([...prev, ...matchedKeys])));
  };

  // Node title renderer with search keyword highlight
  const renderTreeTitle = (nodeData: DataNode) => {
    const node = nodeData as unknown as OrgNode;
    const kw = searchKeyword.trim().toLowerCase();
    const title = node.title || '未命名组织';
    const code = node.code || '';
    const isMatched =
      kw &&
      (title.toLowerCase().includes(kw) || code.toLowerCase().includes(kw));

    return (
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          padding: '2px 0',
          fontSize: 13,
        }}
      >
        <span
          style={{
            fontWeight: node.children && node.children.length > 0 ? 600 : 400,
            color: isMatched ? '#1677ff' : undefined,
          }}
        >
          {title}
        </span>
        <Tag
          bordered={false}
          style={{
            margin: 0,
            fontSize: 11,
            lineHeight: '18px',
            padding: '0 4px',
            color: '#64748b',
            backgroundColor: '#f1f5f9',
            fontFamily: 'monospace',
          }}
        >
          {code}
        </Tag>
        {!node.enabled && (
          <Tag
            color="default"
            bordered={false}
            style={{
              margin: 0,
              fontSize: 11,
              lineHeight: '18px',
              padding: '0 4px',
            }}
          >
            停用
          </Tag>
        )}
      </span>
    );
  };

  const openCreateRoot = () => {
    setParentForCreate(null);
    createFormRef.current?.resetFields();
    setCreateModalOpen(true);
  };

  const openCreateChild = () => {
    if (!selectedOrg) return;
    setParentForCreate(selectedOrg);
    createFormRef.current?.resetFields();
    setCreateModalOpen(true);
  };

  const openEdit = () => {
    if (!selectedOrg) return;
    setEditingOrg(selectedOrg);
    editFormRef.current?.setFieldsValue({
      name: selectedOrg.name,
      enabled: selectedOrg.enabled ?? true,
    });
    setEditModalOpen(true);
  };

  const parentOrgOfSelected = selectedOrg?.parentId
    ? orgMap.get(selectedOrg.parentId)
    : null;

  return (
    <div style={{ minHeight: 600 }}>
      <Row gutter={[16, 16]}>
        {/* Left: Organization Tree Card */}
        <Col xs={24} md={9} lg={8} xl={7}>
          <Card
            styles={{
              body: { padding: '12px' },
              header: { padding: '0 12px', minHeight: 48 },
            }}
            title={
              <Space size={6}>
                <ApartmentOutlined style={{ color: '#1677ff' }} />
                <span>组织架构</span>
                <Tag style={{ margin: 0, fontSize: 11 }}>
                  {organizations.length} 个
                </Tag>
              </Space>
            }
            extra={
              <Space size={4}>
                <Button
                  size="small"
                  icon={<ReloadOutlined />}
                  onClick={() => loadData(selectedId)}
                  loading={loading}
                  title="刷新组织树"
                />
                <Button
                  type="primary"
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={openCreateRoot}
                >
                  新增根组织
                </Button>
              </Space>
            }
          >
            {/* Search Input */}
            <div style={{ marginBottom: 12 }}>
              <Input
                placeholder="搜索组织名称或编码"
                prefix={<SearchOutlined style={{ color: '#94a3b8' }} />}
                allowClear
                size="middle"
                onChange={(e) => handleSearch(e.target.value)}
              />
            </div>

            {/* Tree */}
            <Spin spinning={loading}>
              <div
                style={{
                  maxHeight: 'calc(100vh - 310px)',
                  minHeight: 420,
                  overflowY: 'auto',
                  paddingRight: 4,
                }}
              >
                {filteredTreeData.length > 0 ? (
                  <Tree
                    showIcon
                    blockNode
                    treeData={filteredTreeData as unknown as DataNode[]}
                    selectedKeys={selectedId ? [selectedId] : []}
                    expandedKeys={expandedKeys}
                    autoExpandParent={autoExpandParent}
                    onExpand={(keys) => {
                      setExpandedKeys(keys);
                      setAutoExpandParent(false);
                    }}
                    onSelect={(keys) => {
                      if (keys.length > 0) {
                        setSelectedId(String(keys[0]));
                      }
                    }}
                    titleRender={renderTreeTitle}
                    icon={({ expanded, isLeaf }) => {
                      if (!isLeaf) {
                        return expanded ? (
                          <FolderOpenOutlined style={{ color: '#1677ff' }} />
                        ) : (
                          <FolderOutlined style={{ color: '#1677ff' }} />
                        );
                      }
                      return <TeamOutlined style={{ color: '#64748b' }} />;
                    }}
                  />
                ) : (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={searchKeyword ? '无匹配组织' : '暂无组织数据'}
                    style={{ margin: '40px 0' }}
                  />
                )}
              </div>
            </Spin>
          </Card>
        </Col>

        {/* Right: Selected Organization Details */}
        <Col xs={24} md={15} lg={16} xl={17}>
          {selectedOrg ? (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Card
                styles={{
                  body: { padding: 20 },
                  header: { padding: '0 20px', minHeight: 52 },
                }}
                title={
                  <Space align="center" size={8}>
                    <Text strong style={{ fontSize: 16 }}>
                      {selectedOrg.name}
                    </Text>
                    {selectedOrg.enabled ? (
                      <Tag color="success">启用</Tag>
                    ) : (
                      <Tag>已停用</Tag>
                    )}
                    <Tag
                      bordered={false}
                      style={{
                        fontFamily: 'monospace',
                        color: '#475569',
                        backgroundColor: '#f1f5f9',
                      }}
                    >
                      {selectedOrg.code}
                    </Tag>
                  </Space>
                }
                extra={
                  <Space size={8}>
                    <Button icon={<PlusOutlined />} onClick={openCreateChild}>
                      新增子组织
                    </Button>
                    <Button
                      type="primary"
                      icon={<EditOutlined />}
                      onClick={openEdit}
                    >
                      编辑组织
                    </Button>
                  </Space>
                }
              >
                <Descriptions
                  bordered
                  size="small"
                  column={{ xs: 1, sm: 2, md: 2, lg: 3 }}
                  styles={{ label: { width: 110, color: '#64748b' } }}
                >
                  <Descriptions.Item label="组织名称">
                    <Text strong>{selectedOrg.name}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="组织编码">
                    <Text copyable style={{ fontFamily: 'monospace' }}>
                      {selectedOrg.code}
                    </Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="组织状态">
                    {selectedOrg.enabled ? (
                      <Tag color="success">启用中</Tag>
                    ) : (
                      <Tag color="default">已停用</Tag>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="上级组织">
                    {parentOrgOfSelected ? (
                      <Space size={4}>
                        <Button
                          type="link"
                          size="small"
                          style={{ padding: 0, height: 'auto' }}
                          onClick={() =>
                            setSelectedId(parentOrgOfSelected.id ?? '')
                          }
                        >
                          {parentOrgOfSelected.name}
                        </Button>
                        <Text
                          type="secondary"
                          style={{ fontSize: 11, fontFamily: 'monospace' }}
                        >
                          ({parentOrgOfSelected.code})
                        </Text>
                      </Space>
                    ) : (
                      <Tag color="blue" bordered={false}>
                        根组织 / 顶级
                      </Tag>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="直接子组织">
                    <Text strong>{directChildren.length}</Text> 个
                  </Descriptions.Item>
                  <Descriptions.Item label="下级组织总数">
                    <Text strong>{totalDescendantCount}</Text> 个
                  </Descriptions.Item>
                  <Descriptions.Item label="组织唯一标识" span={3}>
                    <Text
                      copyable
                      type="secondary"
                      style={{ fontSize: 12, fontFamily: 'monospace' }}
                    >
                      {selectedOrg.id}
                    </Text>
                  </Descriptions.Item>
                </Descriptions>
              </Card>

              {/* Sub-organizations Table */}
              <Card
                title="下级子组织"
                styles={{
                  body: { padding: 0 },
                  header: { padding: '0 20px', minHeight: 48 },
                }}
                extra={
                  <Button
                    size="small"
                    type="link"
                    icon={<PlusOutlined />}
                    onClick={openCreateChild}
                  >
                    添加子组织
                  </Button>
                }
              >
                <Table
                  rowKey="id"
                  size="small"
                  pagination={false}
                  dataSource={directChildren}
                  locale={{ emptyText: '当前组织下暂无子组织' }}
                  columns={[
                    {
                      title: '组织名称',
                      dataIndex: 'name',
                      render: (text, record) => (
                        <Button
                          type="link"
                          size="small"
                          style={{
                            padding: 0,
                            height: 'auto',
                            fontWeight: 500,
                          }}
                          onClick={() => setSelectedId(record.id ?? '')}
                        >
                          {text}
                        </Button>
                      ),
                    },
                    {
                      title: '编码',
                      dataIndex: 'code',
                      render: (code) => (
                        <Text
                          copyable
                          style={{ fontFamily: 'monospace', fontSize: 12 }}
                        >
                          {code}
                        </Text>
                      ),
                    },
                    {
                      title: '状态',
                      dataIndex: 'enabled',
                      width: 90,
                      render: (enabled) =>
                        enabled ? (
                          <Tag color="success">启用</Tag>
                        ) : (
                          <Tag>停用</Tag>
                        ),
                    },
                    {
                      title: '操作',
                      width: 90,
                      render: (_, record) => (
                        <Button
                          type="link"
                          size="small"
                          style={{ padding: 0 }}
                          onClick={() => {
                            setSelectedId(record.id ?? '');
                            setEditingOrg(record);
                            editFormRef.current?.setFieldsValue({
                              name: record.name,
                              enabled: record.enabled ?? true,
                            });
                            setEditModalOpen(true);
                          }}
                        >
                          编辑
                        </Button>
                      ),
                    },
                  ]}
                />
              </Card>

              {/* Architecture info callout */}
              <Alert
                showIcon
                icon={<InfoCircleOutlined />}
                type="info"
                message="组织架构权限联动机制"
                description="组织用于划分企业多租户或多层级业务边界。调整组织启用状态时，停用的组织及其成员的数据访问权限将受到相应限制。"
                style={{ backgroundColor: '#f8fafc', borderColor: '#e2e8f0' }}
              />
            </Space>
          ) : (
            <Card
              style={{
                minHeight: 400,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Empty description="请从左侧组织树选择一个节点查看详细信息，或点击上方新增根组织">
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openCreateRoot}
                >
                  新增根组织
                </Button>
              </Empty>
            </Card>
          )}
        </Col>
      </Row>

      {/* Modal: Create Organization */}
      <ModalForm<CreateFormValues>
        title={
          parentForCreate
            ? `新增子组织（所属上级：${parentForCreate.name}）`
            : '新增根组织'
        }
        open={createModalOpen}
        formRef={createFormRef}
        modalProps={{
          destroyOnClose: true,
          width: 520,
          onCancel: () => setCreateModalOpen(false),
        }}
        onOpenChange={setCreateModalOpen}
        onFinish={async (values) => {
          try {
            const response = await adminServiceCreateOrganization({
              code: values.code?.trim() ?? '',
              name: values.name?.trim() ?? '',
              parentId: parentForCreate?.id,
            });
            message.success('组织已创建');
            setCreateModalOpen(false);
            const createdId = response.data?.id;
            await loadData(createdId || selectedId);
            return true;
          } catch {
            message.error('创建组织失败，请重试');
            return false;
          }
        }}
      >
        {parentForCreate && (
          <div style={{ marginBottom: 16 }}>
            <Text
              type="secondary"
              style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
            >
              所属上级组织
            </Text>
            <Tag
              color="processing"
              style={{ fontSize: 13, padding: '2px 8px' }}
            >
              {parentForCreate.name} ({parentForCreate.code})
            </Tag>
          </div>
        )}
        <ProFormText
          name="code"
          label="组织编码"
          placeholder="例如：SH_BRANCH 或 LOGISTICS_HQ"
          rules={[
            { required: true, message: '请输入组织编码' },
            {
              pattern: /^[A-Za-z0-9_-]+$/,
              message: '编码仅支持英文字母、数字、下划线及连字符',
            },
          ]}
        />
        <ProFormText
          name="name"
          label="组织名称"
          placeholder="例如：上海分公司 / 华东海运中心"
          rules={[{ required: true, message: '请输入组织名称' }]}
        />
      </ModalForm>

      {/* Modal: Edit Organization */}
      <ModalForm<EditFormValues>
        title={`编辑组织：${editingOrg?.name ?? ''}`}
        open={editModalOpen}
        formRef={editFormRef}
        modalProps={{
          destroyOnClose: true,
          width: 520,
          onCancel: () => setEditModalOpen(false),
        }}
        onOpenChange={setEditModalOpen}
        onFinish={async (values) => {
          if (!editingOrg?.id) return false;
          try {
            await adminServiceUpdateOrganization(
              { id: editingOrg.id },
              {
                id: editingOrg.id,
                name: values.name?.trim() ?? '',
                enabled: values.enabled ?? true,
              },
            );
            message.success('组织已更新');
            setEditModalOpen(false);
            await loadData(editingOrg.id);
            return true;
          } catch {
            message.error('更新组织失败，请重试');
            return false;
          }
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Text
            type="secondary"
            style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
          >
            组织编码（不可变更）
          </Text>
          <Text strong style={{ fontFamily: 'monospace', fontSize: 14 }}>
            {editingOrg?.code}
          </Text>
        </div>
        <ProFormText
          name="name"
          label="组织名称"
          rules={[{ required: true, message: '请输入组织名称' }]}
        />
        <ProFormSwitch
          name="enabled"
          label="启用状态"
          extra="停用后该组织及其关联成员将无法进行业务操作"
        />
      </ModalForm>
    </div>
  );
}
