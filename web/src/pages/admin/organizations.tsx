import { OrganizationChart } from '@ant-design/graphs';
import {
  ApartmentOutlined,
  AppstoreOutlined,
  EditOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  NodeIndexOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProCard,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Input,
  Segmented,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Tree,
  Typography,
} from 'antd';
import type { DataNode } from 'antd/es/tree';
import { useAccess } from '@umijs/max';
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
import {
  buildOrgTree,
  filterOrgTree,
  getDirectChildren,
  getTotalDescendantCount,
  type OrgTreeNode,
} from './organization-tree';

const { Text, Title } = Typography;

const organizationKindMeta = {
  1: { label: '总部', color: 'purple' },
  2: { label: '公司', color: 'blue' },
  3: { label: '部门', color: 'cyan' },
  4: { label: '组', color: 'gold' },
} as const;

function getOrganizationKindMeta(kind?: number) {
  return organizationKindMeta[kind as keyof typeof organizationKindMeta];
}

function normalizeOrganizationKind(
  kind: API.AdminOrganization['kind'],
): number {
  if (typeof kind === 'number') return kind;
  switch (String(kind)) {
    case 'ORGANIZATION_KIND_HEADQUARTERS':
      return 1;
    case 'ORGANIZATION_KIND_COMPANY':
      return 2;
    case 'ORGANIZATION_KIND_DEPARTMENT':
      return 3;
    case 'ORGANIZATION_KIND_TEAM':
      return 4;
    default:
      return 0;
  }
}

function getChildOrganizationKind(kind?: number): 2 | 3 | 4 | undefined {
  if (kind === 1) return 2;
  if (kind === 2) return 3;
  if (kind === 3) return 4;
  return undefined;
}

type CreateFormValues = {
  code?: string;
  name?: string;
  baseCurrency?: string;
};

type EditFormValues = {
  name?: string;
  enabled?: boolean;
  baseCurrency?: string;
};

export default function OrganizationsPanel() {
  const { message } = App.useApp();
  const access = useAccess();
  const [loading, setLoading] = useState(false);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>(
    [],
  );
  const [selectedId, setSelectedId] = useState<string>('');
  const [searchKeyword, setSearchKeyword] = useState<string>('');
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // View Mode: 'chart' (拓扑架构图) | 'list' (树表列表)
  const [viewMode, setViewMode] = useState<'chart' | 'list'>('chart');
  const [chartDirection, setChartDirection] = useState<
    'vertical' | 'horizontal'
  >('vertical');
  const [drawerOpen, setDrawerOpen] = useState(false);

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
  const graphRef = useRef<any>(null);

  // Load organizations from backend
  const loadData = useCallback(
    async (selectIdAfterLoad?: string) => {
      setLoading(true);
      try {
        const response = await adminServiceListOrganizations();
        const list = (response.data ?? []).map((organization) => ({
          ...organization,
          kind: normalizeOrganizationKind(organization.kind),
        }));
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

  // Construct organization tree structure
  const { treeData, allKeys, orgMap } = useMemo(
    () => buildOrgTree(organizations),
    [organizations],
  );

  // Default expand all nodes on initial load
  useEffect(() => {
    if (expandedKeys.length === 0 && allKeys.length > 0) {
      setExpandedKeys(allKeys);
    }
  }, [allKeys, expandedKeys.length]);

  // Selected organization object
  const selectedOrg = useMemo(() => {
    if (!selectedId) return null;
    return orgMap.get(selectedId) ?? null;
  }, [selectedId, orgMap]);

  // Parent organization of currently selected organization
  const parentOrgOfSelected = useMemo(() => {
    if (!selectedOrg?.parentId) return null;
    return orgMap.get(selectedOrg.parentId) ?? null;
  }, [selectedOrg, orgMap]);

  // Direct sub-organizations of selected organization
  const directChildren = useMemo(
    () => getDirectChildren(selectedId, organizations),
    [selectedId, organizations],
  );

  // Recursive total count of all sub-organizations
  const totalDescendantCount = useMemo(
    () => getTotalDescendantCount(selectedId, organizations),
    [selectedId, organizations],
  );

  // Filtered tree data
  const { filteredTree } = useMemo(
    () => filterOrgTree(treeData, searchKeyword),
    [treeData, searchKeyword],
  );

  // Graph data for @ant-design/graphs OrganizationChart
  const graphData = useMemo(() => {
    const validIds = new Set(organizations.map((o) => o.id));
    const directChildrenMap = new Map<string, number>();

    for (const org of organizations) {
      if (org.parentId && validIds.has(org.parentId)) {
        directChildrenMap.set(
          org.parentId,
          (directChildrenMap.get(org.parentId) ?? 0) + 1,
        );
      }
    }

    const nodes = organizations.map((org) => {
      const id = org.id ?? '';
      return {
        id,
        data: {
          id,
          name: org.name ?? '未命名组织',
          code: org.code ?? '',
          enabled: org.enabled ?? true,
          kind: org.kind ?? 0,
          parentId: org.parentId,
          childrenCount: directChildrenMap.get(id) ?? 0,
        },
      };
    });

    const edges: {
      source: string;
      target: string;
      sourcePort: string;
      targetPort: string;
    }[] = [];
    for (const org of organizations) {
      if (org.parentId && validIds.has(org.parentId) && org.id) {
        edges.push({
          source: org.parentId,
          target: org.id,
          sourcePort: 'out',
          targetPort: 'in',
        });
      }
    }

    return { nodes, edges };
  }, [organizations]);

  // Handle keyword search
  const handleSearch = (value: string) => {
    setSearchKeyword(value);
    setAutoExpandParent(true);
    if (value.trim()) {
      const { matchedKeys: keysToExpand } = filterOrgTree(treeData, value);
      setExpandedKeys((prev) =>
        Array.from(new Set([...prev, ...keysToExpand])),
      );
    }
  };

  // Node title renderer for standard Tree
  const renderTreeTitle = (nodeData: DataNode) => {
    const node = nodeData as unknown as OrgTreeNode;
    const kw = searchKeyword.trim().toLowerCase();
    const title = node.title || '未命名组织';
    const code = node.code || '';
    const kindMeta = getOrganizationKindMeta(node.kind);
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
            color: isMatched ? '#1677ff' : 'rgba(0, 0, 0, 0.88)',
          }}
        >
          {title}
        </span>
        <Tag
          variant="filled"
          style={{
            margin: 0,
            fontSize: 11,
            lineHeight: '18px',
            padding: '0 4px',
            fontFamily: 'monospace',
            backgroundColor: '#fafafa',
            color: 'rgba(0, 0, 0, 0.45)',
          }}
        >
          {code}
        </Tag>
        {kindMeta && (
          <Tag color={kindMeta.color} variant="filled" style={{ margin: 0 }}>
            {kindMeta.label}
          </Tag>
        )}
        {!node.enabled && (
          <Tag
            color="default"
            variant="filled"
            style={{
              margin: 0,
              fontSize: 11,
              lineHeight: '18px',
              padding: '0 4px',
            }}
          >
            已停用
          </Tag>
        )}
      </span>
    );
  };

  const openCreateChild = (parent?: API.AdminOrganization | null) => {
    if (!access.canCreateOrganizations) return;
    const targetParent = parent || selectedOrg;
    if (!targetParent || !getChildOrganizationKind(targetParent.kind)) return;
    setParentForCreate(targetParent);
    createFormRef.current?.resetFields();
    if (getChildOrganizationKind(targetParent.kind) === 2) {
      createFormRef.current?.setFieldValue(
        'baseCurrency',
        targetParent.baseCurrency,
      );
    }
    setCreateModalOpen(true);
  };

  const openEdit = (org?: API.AdminOrganization | null) => {
    if (!access.canUpdateOrganizations) return;
    const targetOrg = org || selectedOrg;
    if (!targetOrg) return;
    setEditingOrg(targetOrg);
    editFormRef.current?.setFieldsValue({
      name: targetOrg.name,
      enabled: targetOrg.enabled ?? true,
      baseCurrency:
        targetOrg.kind === 1 || targetOrg.kind === 2
          ? targetOrg.baseCurrency
          : undefined,
    });
    setEditModalOpen(true);
  };

  const childKindForCreate = getChildOrganizationKind(parentForCreate?.kind);
  const childKindMetaForCreate = getOrganizationKindMeta(childKindForCreate);

  return (
    <>
      <Card
        styles={{ body: { padding: '12px 16px' } }}
        style={{ marginBottom: 12 }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <Space size={12} align="center">
            <Segmented
              value={viewMode}
              onChange={(val) => setViewMode(val as 'chart' | 'list')}
              options={[
                {
                  value: 'chart',
                  icon: <NodeIndexOutlined />,
                  label: '拓扑架构图',
                },
                {
                  value: 'list',
                  icon: <AppstoreOutlined />,
                  label: '树表列表',
                },
              ]}
            />
            {viewMode === 'chart' && (
              <Segmented
                value={chartDirection}
                onChange={(val) =>
                  setChartDirection(val as 'vertical' | 'horizontal')
                }
                options={[
                  { value: 'vertical', label: '垂直上下' },
                  { value: 'horizontal', label: '水平左右' },
                ]}
              />
            )}
            <Tag color="blue" variant="filled">
              共 {organizations.length} 个组织节点
            </Tag>
          </Space>

          <Space size={8}>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => loadData(selectedId)}
              loading={loading}
            >
              刷新数据
            </Button>
          </Space>
        </div>
      </Card>

      {/* Mode 1: Interactive Organization Chart */}
      {viewMode === 'chart' && (
        <Card
          styles={{ body: { padding: 0 } }}
          style={{
            minHeight: 640,
            overflow: 'hidden',
            position: 'relative',
          }}
        >
          {/* Org Chart Container */}
          <Spin spinning={loading}>
            {graphData.nodes.length > 0 ? (
              <div
                style={{ height: 'calc(100vh - 270px)', minHeight: 600 }}
                onMouseDown={(e) => {
                  if (e.button === 1) {
                    e.preventDefault();
                  }
                }}
              >
                <OrganizationChart
                  ref={graphRef}
                  data={graphData}
                  direction={chartDirection}
                  autoFit="center"
                  node={{
                    style: {
                      size: [210, 80],
                      ports:
                        chartDirection === 'vertical'
                          ? [
                              { key: 'in', placement: 'top' },
                              { key: 'out', placement: 'bottom' },
                            ]
                          : [
                              { key: 'in', placement: 'left' },
                              { key: 'out', placement: 'right' },
                            ],
                      component: (nodeData: Record<string, unknown>) => {
                        const item = (nodeData.data || nodeData) as {
                          id: string;
                          name: string;
                          code: string;
                          kind: number;
                          enabled: boolean;
                          parentId?: string;
                          childrenCount?: number;
                        };
                        const isCurrentSelected = item.id === selectedId;

                        return (
                          <div
                            style={{
                              width: 210,
                              height: 80,
                              backgroundColor: '#ffffff',
                              borderRadius: 8,
                              border: isCurrentSelected
                                ? '2px solid #1677ff'
                                : '1px solid #e2e8f0',
                              boxShadow: isCurrentSelected
                                ? '0 4px 14px rgba(22, 119, 255, 0.25)'
                                : '0 2px 6px rgba(0, 0, 0, 0.04)',
                              padding: '9px 12px',
                              display: 'flex',
                              flexDirection: 'column',
                              justifyContent: 'space-between',
                              cursor: 'pointer',
                              boxSizing: 'border-box',
                              position: 'relative',
                              transition: 'all 0.15s ease',
                            }}
                          >
                            {/* Inlet Anchor / 上级入口连接桩（有上级组织时显示） */}
                            {item.parentId && (
                              <div
                                style={{
                                  position: 'absolute',
                                  ...(chartDirection === 'vertical'
                                    ? {
                                        top: -4,
                                        left: '50%',
                                        transform: 'translateX(-50%)',
                                      }
                                    : {
                                        left: -4,
                                        top: '50%',
                                        transform: 'translateY(-50%)',
                                      }),
                                  width: 8,
                                  height: 8,
                                  borderRadius: '50%',
                                  backgroundColor: '#1677ff',
                                  border: '2px solid #ffffff',
                                  boxShadow: '0 1px 3px rgba(0, 0, 0, 0.2)',
                                  zIndex: 5,
                                }}
                              />
                            )}

                            {/* Card Top Row: Name + Status */}
                            <div
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                              }}
                            >
                              <div
                                style={{
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 6,
                                  minWidth: 0,
                                }}
                              >
                                <ApartmentOutlined
                                  style={{
                                    color: isCurrentSelected
                                      ? '#1677ff'
                                      : 'rgba(0, 0, 0, 0.45)',
                                    fontSize: 14,
                                  }}
                                />
                                <span
                                  style={{
                                    fontWeight: 600,
                                    fontSize: 13,
                                    color: isCurrentSelected
                                      ? '#1677ff'
                                      : 'rgba(0, 0, 0, 0.88)',
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                    maxWidth: 120,
                                  }}
                                  title={item.name}
                                >
                                  {item.name}
                                </span>
                              </div>
                              {item.enabled ? (
                                <Tag
                                  color="success"
                                  variant="filled"
                                  style={{
                                    margin: 0,
                                    fontSize: 10,
                                    lineHeight: '16px',
                                    padding: '0 4px',
                                  }}
                                >
                                  启用
                                </Tag>
                              ) : (
                                <Tag
                                  color="default"
                                  variant="filled"
                                  style={{
                                    margin: 0,
                                    fontSize: 10,
                                    lineHeight: '16px',
                                    padding: '0 4px',
                                  }}
                                >
                                  停用
                                </Tag>
                              )}
                            </div>

                            {/* Card Bottom Row: Code + Count Badge */}
                            <div
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                fontSize: 11,
                              }}
                            >
                              <span
                                style={{
                                  fontFamily: 'monospace',
                                  color: 'rgba(0, 0, 0, 0.45)',
                                  maxWidth: 110,
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                  whiteSpace: 'nowrap',
                                }}
                                title={item.code}
                              >
                                {item.code}
                              </span>

                              {getOrganizationKindMeta(item.kind) && (
                                <Tag
                                  color={
                                    getOrganizationKindMeta(item.kind)?.color
                                  }
                                  variant="filled"
                                  style={{
                                    margin: 0,
                                    fontSize: 10,
                                    lineHeight: '16px',
                                    padding: '0 4px',
                                  }}
                                >
                                  {getOrganizationKindMeta(item.kind)?.label}
                                </Tag>
                              )}

                              {(item.childrenCount ?? 0) > 0 ? (
                                <Tag
                                  color="blue"
                                  variant="filled"
                                  style={{
                                    margin: 0,
                                    fontSize: 10,
                                    lineHeight: '16px',
                                    padding: '0 4px',
                                  }}
                                >
                                  {item.childrenCount} 个下级
                                </Tag>
                              ) : (
                                <span style={{ color: 'rgba(0, 0, 0, 0.25)' }}>
                                  末级
                                </span>
                              )}
                            </div>

                            {access.canCreateOrganizations && getChildOrganizationKind(item.kind) && (
                              <Tooltip
                                title={`在「${item.name}」下新增${getOrganizationKindMeta(getChildOrganizationKind(item.kind))?.label}`}
                              >
                                <div
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    openCreateChild(item);
                                  }}
                                  style={{
                                    position: 'absolute',
                                    ...(chartDirection === 'vertical'
                                      ? {
                                          bottom: -11,
                                          left: '50%',
                                          transform: 'translateX(-50%)',
                                        }
                                      : {
                                          right: -11,
                                          top: '50%',
                                          transform: 'translateY(-50%)',
                                        }),
                                    width: 22,
                                    height: 22,
                                    borderRadius: '50%',
                                    backgroundColor: '#1677ff',
                                    color: '#ffffff',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    boxShadow:
                                      '0 2px 6px rgba(22, 119, 255, 0.4)',
                                    cursor: 'pointer',
                                    zIndex: 10,
                                    border: '2px solid #ffffff',
                                    transition: 'all 0.15s ease',
                                  }}
                                >
                                  <PlusOutlined style={{ fontSize: 10 }} />
                                </div>
                              </Tooltip>
                            )}
                          </div>
                        );
                      },
                    },
                  }}
                  edge={{
                    type:
                      chartDirection === 'vertical'
                        ? 'cubic-vertical'
                        : 'cubic-horizontal',
                    style: {
                      stroke: '#1677ff',
                      lineWidth: 1.5,
                      endArrow: true,
                      endArrowType: 'vee',
                    },
                  }}
                  behaviors={[
                    {
                      key: 'drag-canvas',
                      type: 'drag-canvas',
                      enable: (event: any) => {
                        const btn =
                          event.button ?? event.nativeEvent?.button;
                        if (btn === 1) return false;
                        const buttons =
                          event.buttons ?? event.nativeEvent?.buttons;
                        if (
                          typeof buttons === 'number' &&
                          (buttons & 4) === 4
                        ) {
                          return false;
                        }
                        if ('targetType' in event) {
                          return event.targetType === 'canvas';
                        }
                        return true;
                      },
                    },
                    'click-select',
                  ]}
                  onReady={(graph: any) => {
                    graphRef.current = graph;
                    void graph.fitCenter();
                    graph.off('node:click');
                    graph.on('node:click', (evt: any) => {
                      const id =
                        evt.target?.id || evt.itemId || evt.target?.cfg?.id;
                      if (id) {
                        setSelectedId(id);
                        setDrawerOpen(true);
                      }
                    });
                  }}
                />
              </div>
            ) : (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="暂无总部数据，请先执行管理员初始化"
                style={{ padding: '80px 0' }}
              />
            )}
          </Spin>
        </Card>
      )}

      {/* Mode 2: Classic Tree & Table Mode */}
      {viewMode === 'list' && (
        <ProCard
          split="vertical"
          variant="outlined"
          headerBordered
          style={{ minHeight: 600 }}
        >
          {/* Left: Tree */}
          <ProCard
            colSpan={{ xs: 24, md: '340px' }}
            title={
              <Space size={8}>
                <ApartmentOutlined style={{ color: '#1677ff' }} />
                <span>组织架构树</span>
                <Tag
                  color="blue"
                  variant="filled"
                  style={{ margin: 0, fontSize: 11 }}
                >
                  {organizations.length} 个节点
                </Tag>
              </Space>
            }
          >
            {/* Search Input */}
            <div style={{ marginBottom: 12 }}>
              <Input
                placeholder="搜索组织名称或编码..."
                prefix={
                  <SearchOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />
                }
                allowClear
                value={searchKeyword}
                onChange={(e) => handleSearch(e.target.value)}
              />
            </div>

            {/* Tree View */}
            <Spin spinning={loading}>
              <div
                style={{
                  maxHeight: 'calc(100vh - 340px)',
                  minHeight: 400,
                  overflowY: 'auto',
                  paddingRight: 4,
                }}
              >
                {filteredTree.length > 0 ? (
                  <Tree
                    showIcon
                    blockNode
                    treeData={filteredTree as unknown as DataNode[]}
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
                      return (
                        <TeamOutlined
                          style={{ color: 'rgba(0, 0, 0, 0.45)' }}
                        />
                      );
                    }}
                  />
                ) : (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={
                      searchKeyword ? '未找到匹配的组织机构' : '暂无组织数据'
                    }
                    style={{ margin: '40px 0' }}
                  />
                )}
              </div>
            </Spin>
          </ProCard>

          {/* Right: Details & Children Table */}
          <ProCard
            title={
              selectedOrg ? (
                <Space align="center" size={10}>
                  <Title
                    level={4}
                    style={{ margin: 0, color: 'rgba(0, 0, 0, 0.88)' }}
                  >
                    {selectedOrg.name}
                  </Title>
                  <Tag
                    variant="filled"
                    style={{
                      margin: 0,
                      fontFamily: 'monospace',
                      fontSize: 12,
                      padding: '2px 8px',
                      backgroundColor: '#fafafa',
                      color: 'rgba(0, 0, 0, 0.65)',
                    }}
                  >
                    {selectedOrg.code}
                  </Tag>
                  {selectedOrg.enabled ? (
                    <Tag color="success">正常启用</Tag>
                  ) : (
                    <Tag color="default">已停用</Tag>
                  )}
                </Space>
              ) : (
                '组织详细信息'
              )
            }
            extra={
              selectedOrg ? (
                <Space size={8}>
                  {access.canCreateOrganizations && getChildOrganizationKind(selectedOrg.kind) && (
                    <Button
                      icon={<PlusOutlined />}
                      onClick={() => openCreateChild(selectedOrg)}
                    >
                      新增
                      {
                        getOrganizationKindMeta(
                          getChildOrganizationKind(selectedOrg.kind),
                        )?.label
                      }
                    </Button>
                  )}
                  {access.canUpdateOrganizations && (
                    <Button
                      type="primary"
                      icon={<EditOutlined />}
                      onClick={() => openEdit(selectedOrg)}
                    >
                      编辑组织
                    </Button>
                  )}
                </Space>
              ) : null
            }
            headerBordered
          >
            {selectedOrg ? (
              <Space vertical size={20} style={{ width: '100%' }}>
                <Descriptions
                  bordered
                  size="middle"
                  column={{ xs: 1, sm: 2, lg: 3 }}
                >
                  <Descriptions.Item label="组织编码">
                    <Text copyable style={{ fontFamily: 'monospace' }}>
                      {selectedOrg.code}
                    </Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="组织名称">
                    <Text strong>{selectedOrg.name}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="本币">
                    <Tag color="blue">{selectedOrg.baseCurrency}</Tag>
                    {selectedOrg.kind === 3 || selectedOrg.kind === 4 ? (
                      <Text type="secondary">从所属公司继承</Text>
                    ) : null}
                  </Descriptions.Item>
                  <Descriptions.Item label="上级组织">
                    {parentOrgOfSelected ? (
                      <Button
                        type="link"
                        size="small"
                        style={{ padding: 0, height: 'auto' }}
                        onClick={() =>
                          setSelectedId(parentOrgOfSelected.id ?? '')
                        }
                      >
                        {parentOrgOfSelected.name} ({parentOrgOfSelected.code})
                      </Button>
                    ) : (
                      <Text type="secondary">根组织（无上级）</Text>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="运营状态">
                    {selectedOrg.enabled ? (
                      <Tag color="success">正常启用</Tag>
                    ) : (
                      <Tag color="default">已停用</Tag>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="直属下级机构">
                    <Text strong style={{ color: '#1677ff' }}>
                      {directChildren.length}
                    </Text>{' '}
                    个
                  </Descriptions.Item>
                  <Descriptions.Item label="全部后代机构">
                    <Text strong style={{ color: '#1677ff' }}>
                      {totalDescendantCount}
                    </Text>{' '}
                    个
                  </Descriptions.Item>
                </Descriptions>

                <Card
                  title={
                    <Space size={6}>
                      <TeamOutlined style={{ color: '#1677ff' }} />
                      <span style={{ fontSize: 14, fontWeight: 600 }}>
                        直属下级组织列表
                      </span>
                      <Tag
                        color="blue"
                        variant="filled"
                        style={{ margin: 0, fontSize: 11 }}
                      >
                        {directChildren.length}
                      </Tag>
                    </Space>
                  }
                  size="small"
                  extra={access.canCreateOrganizations ? (
                    <Button
                      type="link"
                      size="small"
                      icon={<PlusOutlined />}
                      onClick={() => openCreateChild(selectedOrg)}
                    >
                      添加下级组织
                    </Button>
                  ) : null}
                >
                  <Table<API.AdminOrganization>
                    rowKey="id"
                    size="middle"
                    pagination={
                      directChildren.length > 5
                        ? { pageSize: 5, showSizeChanger: false }
                        : false
                    }
                    dataSource={directChildren}
                    columns={[
                      {
                        title: '组织名称',
                        dataIndex: 'name',
                        render: (name, record) => (
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
                            {name}
                          </Button>
                        ),
                      },
                      {
                        title: '组织编码',
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
                        width: 100,
                        render: (enabled) =>
                          enabled ? (
                            <Tag color="success">启用</Tag>
                          ) : (
                            <Tag color="default">停用</Tag>
                          ),
                      },
                      {
                        title: '操作',
                        width: 100,
                        render: (_, record) => (
                          <Button
                            type="link"
                            size="small"
                            style={{ padding: 0 }}
                            onClick={() => setSelectedId(record.id ?? '')}
                          >
                            切换查看
                          </Button>
                        ),
                      },
                    ]}
                    locale={{
                      emptyText: (
                        <Empty
                          image={Empty.PRESENTED_IMAGE_SIMPLE}
                          description="当前组织暂无直属下级机构"
                          style={{ margin: '16px 0' }}
                        />
                      ),
                    }}
                  />
                </Card>
              </Space>
            ) : (
              <div
                style={{
                  minHeight: 400,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Empty description="请从左侧组织架构树选择一个节点查看详细信息" />
              </div>
            )}
          </ProCard>
        </ProCard>
      )}

      {/* Drawer: Detailed view when a node is clicked in Chart mode */}
      <Drawer
        title={
          selectedOrg ? (
            <Space>
              <span>{selectedOrg.name}</span>
              <Tag
                color={selectedOrg.enabled ? 'success' : 'default'}
                variant="filled"
              >
                {selectedOrg.enabled ? '正常启用' : '已停用'}
              </Tag>
            </Space>
          ) : (
            '组织详情'
          )
        }
        open={drawerOpen && viewMode === 'chart'}
        onClose={() => setDrawerOpen(false)}
        size={420}
        extra={
          selectedOrg && (
            <Space>
              {access.canCreateOrganizations && getChildOrganizationKind(selectedOrg.kind) && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={() => openCreateChild(selectedOrg)}
                >
                  新增
                  {
                    getOrganizationKindMeta(
                      getChildOrganizationKind(selectedOrg.kind),
                    )?.label
                  }
                </Button>
              )}
              {access.canUpdateOrganizations && (
                <Button
                  type="primary"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openEdit(selectedOrg)}
                >
                  编辑
                </Button>
              )}
            </Space>
          )
        }
      >
        {selectedOrg ? (
          <Space vertical size={16} style={{ width: '100%' }}>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="节点类型">
                {getOrganizationKindMeta(selectedOrg.kind)?.label}
              </Descriptions.Item>
              <Descriptions.Item label="组织编码">
                <Text copyable style={{ fontFamily: 'monospace' }}>
                  {selectedOrg.code}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="组织名称">
                <Text strong>{selectedOrg.name}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="本币">
                <Tag color="blue">{selectedOrg.baseCurrency}</Tag>
                {selectedOrg.kind === 3 || selectedOrg.kind === 4 ? (
                  <Text type="secondary">从所属公司继承</Text>
                ) : null}
              </Descriptions.Item>
              <Descriptions.Item label="所属上级">
                {parentOrgOfSelected ? (
                  <span>
                    {parentOrgOfSelected.name} ({parentOrgOfSelected.code})
                  </span>
                ) : (
                  <Text type="secondary">根组织（无上级）</Text>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="直属下级">
                <Text strong style={{ color: '#1677ff' }}>
                  {directChildren.length}
                </Text>{' '}
                个组织
              </Descriptions.Item>
              <Descriptions.Item label="全部后代">
                <Text strong style={{ color: '#1677ff' }}>
                  {totalDescendantCount}
                </Text>{' '}
                个组织
              </Descriptions.Item>
            </Descriptions>

            <Card
              size="small"
              title={`直属下级组织 (${directChildren.length})`}
              styles={{ body: { padding: 0 } }}
            >
              <Table<API.AdminOrganization>
                rowKey="id"
                size="small"
                pagination={false}
                dataSource={directChildren}
                columns={[
                  {
                    title: '名称',
                    dataIndex: 'name',
                    render: (name, record) => (
                      <Button
                        type="link"
                        size="small"
                        style={{ padding: 0 }}
                        onClick={() => setSelectedId(record.id ?? '')}
                      >
                        {name}
                      </Button>
                    ),
                  },
                  {
                    title: '编码',
                    dataIndex: 'code',
                    render: (code) => (
                      <Text style={{ fontFamily: 'monospace', fontSize: 11 }}>
                        {code}
                      </Text>
                    ),
                  },
                ]}
                locale={{
                  emptyText: (
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description="无直属下级"
                      style={{ margin: '12px 0' }}
                    />
                  ),
                }}
              />
            </Card>
          </Space>
        ) : (
          <Empty description="请点击节点查看详情" />
        )}
      </Drawer>

      {/* Modal: Create Organization */}
      <ModalForm<CreateFormValues>
        title={`新增${childKindMetaForCreate?.label ?? ''}（所属上级：${parentForCreate?.name ?? ''}）`}
        open={createModalOpen}
        formRef={createFormRef}
        modalProps={{
          destroyOnClose: true,
          width: 520,
          onCancel: () => setCreateModalOpen(false),
        }}
        onOpenChange={setCreateModalOpen}
        onFinish={async (values) => {
          if (!parentForCreate?.id || !childKindForCreate) return false;
          try {
            const response = await adminServiceCreateOrganization({
              code: values.code?.trim() ?? '',
              name: values.name?.trim() ?? '',
              parentId: parentForCreate.id,
              kind: childKindForCreate,
              baseCurrency:
                childKindForCreate === 2
                  ? values.baseCurrency?.trim().toUpperCase()
                  : undefined,
            });
            message.success(`${childKindMetaForCreate?.label}已成功创建`);
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
        {parentForCreate && childKindMetaForCreate && (
          <Alert
            showIcon
            type="info"
            title={`当前上级：${parentForCreate.name}；本次创建：${childKindMetaForCreate.label}`}
            style={{ marginBottom: 16 }}
          />
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
        {childKindForCreate === 2 && (
          <ProFormText
            name="baseCurrency"
            label="本币"
            placeholder="例如 CNY、USD"
            fieldProps={{ maxLength: 3 }}
            rules={[
              { required: true, message: '请输入公司本币' },
              { pattern: /^[A-Za-z]{3}$/, message: '请输入 3 位币种代码' },
            ]}
          />
        )}
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
                baseCurrency:
                  editingOrg.kind === 1 || editingOrg.kind === 2
                    ? values.baseCurrency?.trim().toUpperCase()
                    : undefined,
              },
            );
            message.success('组织已成功更新');
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
        {(editingOrg?.kind === 1 || editingOrg?.kind === 2) && (
          <ProFormText
            name="baseCurrency"
            label="本币"
            placeholder="例如 CNY、USD"
            fieldProps={{ maxLength: 3 }}
            rules={[
              { required: true, message: '请输入组织本币' },
              { pattern: /^[A-Za-z]{3}$/, message: '请输入 3 位币种代码' },
            ]}
            extra="修改本币不会改变已保存费用的汇率快照"
          />
        )}
      </ModalForm>
    </>
  );
}
