import {
  AppstoreOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  NodeIndexOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  Button,
  Card,
  Empty,
  Input,
  Segmented,
  Space,
  Spin,
  Tag,
  Tree,
} from 'antd';
import type { DataNode } from 'antd/es/tree';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { adminServiceListOrganizations } from '@/services/roncin/adminService';
import OrgChartCanvas from './components/org/OrgChartCanvas';
import OrgCreateModal from './components/org/OrgCreateModal';
import OrgDetailCard from './components/org/OrgDetailCard';
import OrgDetailDrawer from './components/org/OrgDetailDrawer';
import OrgEditModal from './components/org/OrgEditModal';
import {
  getChildOrganizationKind,
  getOrganizationKindMeta,
  normalizeOrganizationKind,
} from './components/org/types';
import {
  buildOrgTree,
  filterOrgTree,
  getDirectChildren,
  getTotalDescendantCount,
  type OrgTreeNode,
} from './organization-tree';

export default function OrganizationsPanel() {
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

        const targetId = selectIdAfterLoad || selectedId;
        if (targetId && list.some((o) => o.id === targetId)) {
          setSelectedId(targetId);
        } else if (list.length > 0) {
          const rootOrg = list.find((o) => !o.parentId);
          setSelectedId(rootOrg ? (rootOrg.id ?? '') : (list[0].id ?? ''));
        }
      } catch {
        // Handled by service
      } finally {
        setLoading(false);
      }
    },
    [selectedId],
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Build Tree Data
  const { treeData, allKeys, orgMap } = useMemo(
    () => buildOrgTree(organizations),
    [organizations],
  );

  // Filtered Tree Data
  const { filteredTree, matchedKeys } = useMemo(
    () => filterOrgTree(treeData, searchKeyword),
    [treeData, searchKeyword],
  );

  // Auto-expand all when loaded
  useEffect(() => {
    if (searchKeyword.trim()) {
      setExpandedKeys(matchedKeys);
    } else if (allKeys.length > 0 && expandedKeys.length === 0) {
      setExpandedKeys(allKeys);
    }
  }, [allKeys, searchKeyword, matchedKeys]);

  const selectedOrg = useMemo(
    () => orgMap.get(selectedId) ?? null,
    [orgMap, selectedId],
  );

  const parentOrgOfSelected = useMemo(() => {
    if (!selectedOrg?.parentId) return null;
    return orgMap.get(selectedOrg.parentId) ?? null;
  }, [orgMap, selectedOrg]);

  const directChildren = useMemo(
    () => getDirectChildren(selectedId, organizations),
    [selectedId, organizations],
  );

  const totalDescendantCount = useMemo(
    () => getTotalDescendantCount(selectedId, organizations),
    [selectedId, organizations],
  );

  // Graph Data for Chart Mode
  const graphData = useMemo(() => {
    const validIds = new Set(organizations.map((o) => o.id).filter(Boolean));
    const nodes = organizations.map((org) => {
      const children = organizations.filter((o) => o.parentId === org.id);
      return {
        id: org.id ?? '',
        data: {
          id: org.id ?? '',
          name: org.name ?? '',
          code: org.code ?? '',
          kind: org.kind ?? 0,
          enabled: org.enabled ?? true,
          parentId: org.parentId,
          childrenCount: children.length,
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
    setCreateModalOpen(true);
  };

  const openEdit = (org?: API.AdminOrganization | null) => {
    if (!access.canUpdateOrganizations) return;
    const targetOrg = org || selectedOrg;
    if (!targetOrg) return;
    setEditingOrg(targetOrg);
    setEditModalOpen(true);
  };

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
        <OrgChartCanvas
          loading={loading}
          graphData={graphData}
          chartDirection={chartDirection}
          selectedId={selectedId}
          onSelectNode={setSelectedId}
          onOpenDrawer={() => setDrawerOpen(true)}
        />
      )}

      {/* Mode 2: Traditional Tree + Detail Table */}
      {viewMode === 'list' && (
        <ProCard split="vertical" headerBordered>
          {/* Left: Organization Tree */}
          <ProCard
            colSpan="380px"
            title={
              <Space size={6}>
                <TeamOutlined style={{ color: '#1677ff' }} />
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
          <OrgDetailCard
            selectedOrg={selectedOrg}
            parentOrgOfSelected={parentOrgOfSelected}
            directChildren={directChildren}
            totalDescendantCount={totalDescendantCount}
            canCreate={Boolean(access.canCreateOrganizations)}
            canUpdate={Boolean(access.canUpdateOrganizations)}
            onOpenCreateChild={openCreateChild}
            onOpenEdit={openEdit}
            onSelectOrg={setSelectedId}
          />
        </ProCard>
      )}

      {/* Drawer: Detailed view when a node is clicked in Chart mode */}
      <OrgDetailDrawer
        open={drawerOpen && viewMode === 'chart'}
        onClose={() => setDrawerOpen(false)}
        selectedOrg={selectedOrg}
        parentOrg={parentOrgOfSelected}
        directChildren={directChildren}
        totalDescendantCount={totalDescendantCount}
        canCreate={access.canCreateOrganizations}
        canUpdate={access.canUpdateOrganizations}
        onOpenCreateChild={openCreateChild}
        onOpenEdit={openEdit}
        onSelectNode={setSelectedId}
      />

      {/* Modal: Create Organization */}
      <OrgCreateModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        parentOrg={parentForCreate}
        onSuccess={async (createdId) => {
          await loadData(createdId || selectedId);
        }}
      />

      {/* Modal: Edit Organization */}
      <OrgEditModal
        open={editModalOpen}
        onOpenChange={setEditModalOpen}
        editingOrg={editingOrg}
        onSuccess={async (updatedId) => {
          await loadData(updatedId || selectedId);
        }}
      />
    </>
  );
}
