import { ProfileOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Space, Spin, Tooltip } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { OrgTreeNode } from '../../organization-tree';
import { HorizontalBranch, VerticalBranch } from './OrgBranches';
import OrgInspectorPanel from './OrgInspectorPanel';
import {
  collectBranchKeys,
  findAncestorKeys,
  findChildRawsByKey,
  findNodeRawByKey,
  resolveTreeData,
} from './orgChartLayout';
import { useOrgChartViewport } from './useOrgChartViewport';

export type OrgChartCanvasProps = {
  loading: boolean;
  graphData?: { nodes: any[]; edges: any[] };
  treeData?: OrgTreeNode[];
  chartDirection: 'vertical' | 'horizontal';
  selectedId: string;
  onSelectNode: (id: string) => void;
  selectedOrg?: API.AdminOrganization | null;
  parentOrg?: API.AdminOrganization | null;
  directChildren?: API.AdminOrganization[];
  totalDescendantCount?: number;
  canCreate?: boolean;
  canUpdate?: boolean;
  onOpenCreateChild?: (org: API.AdminOrganization) => void;
  onOpenEdit?: (org: API.AdminOrganization) => void;
  onOpenDrawer?: () => void;
};

export default function OrgChartCanvas({
  loading,
  graphData,
  treeData,
  chartDirection,
  selectedId,
  onSelectNode,
  selectedOrg,
  parentOrg,
  directChildren,
  totalDescendantCount,
  canCreate = false,
  canUpdate = false,
  onOpenCreateChild,
  onOpenEdit,
  onOpenDrawer,
}: OrgChartCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  const [collapsedKeys, setCollapsedKeys] = useState<Set<string>>(new Set());
  const [inspectorOpen, setInspectorOpen] = useState(true);

  // Reconcile tree data: prefer treeData prop, fallback to reconstructing from graphData
  const effectiveTreeData = useMemo(
    () => resolveTreeData(treeData, graphData),
    [treeData, graphData],
  );

  // Derive selected organization when not explicitly provided
  const currentSelectedOrg = useMemo(() => {
    if (selectedOrg !== undefined) return selectedOrg;
    if (!selectedId) return null;
    return findNodeRawByKey(effectiveTreeData, selectedId);
  }, [selectedOrg, selectedId, effectiveTreeData]);

  // Derive direct children when not explicitly provided
  const currentDirectChildren = useMemo(() => {
    if (directChildren !== undefined) return directChildren;
    if (!selectedId) return [];
    return findChildRawsByKey(effectiveTreeData, selectedId);
  }, [directChildren, selectedId, effectiveTreeData]);

  // Derive parent organization when not explicitly provided
  const currentParentOrg = useMemo(() => {
    if (parentOrg !== undefined) return parentOrg;
    if (!currentSelectedOrg?.parentId) return null;
    return findNodeRawByKey(effectiveTreeData, currentSelectedOrg.parentId);
  }, [parentOrg, currentSelectedOrg, effectiveTreeData]);

  const {
    offset,
    zoom,
    isPanning,
    dragRef,
    fitView,
    centerNode,
    handleMouseDown,
    handleZoomIn,
    handleZoomOut,
    handleResetZoom,
  } = useOrgChartViewport({
    containerRef,
    contentRef,
    chartDirection,
    inspectorOpen,
    selectedOrg: currentSelectedOrg,
  });

  const handleSelectNode = useCallback(
    (id: string) => {
      onSelectNode(id);
      setInspectorOpen(true);
      if (onOpenDrawer) {
        onOpenDrawer();
      }
    },
    [onSelectNode, onOpenDrawer],
  );

  // Focus on a specific node and smoothly center it in the safe visible area
  const focusNode = useCallback(
    (id: string) => {
      if (!containerRef.current || !contentRef.current) return;

      // Expand any collapsed ancestors of target node
      const ancestors = findAncestorKeys(effectiveTreeData, id);
      if (ancestors && ancestors.length > 0) {
        setCollapsedKeys((prev) => {
          let changed = false;
          const next = new Set(prev);
          for (const a of ancestors) {
            if (next.has(a)) {
              next.delete(a);
              changed = true;
            }
          }
          return changed ? next : prev;
        });
      }

      requestAnimationFrame(() => {
        setTimeout(() => centerNode(id), 30);
      });
    },
    [centerNode, effectiveTreeData],
  );

  // Keyboard shortcut: Esc to collapse inspector panel when open
  useEffect(() => {
    if (!inspectorOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !e.defaultPrevented) {
        // Only close inspector if no modal is currently visible
        const hasOpenModal = document.querySelector(
          '.ant-modal-wrap:not([style*="display: none"])',
        );
        if (!hasOpenModal) {
          setInspectorOpen(false);
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [inspectorOpen]);

  // Auto fit on data load or direction switch
  useEffect(() => {
    if (effectiveTreeData.length > 0 && !loading) {
      const timer = setTimeout(() => {
        fitView();
      }, 50);
      return () => {
        clearTimeout(timer);
      };
    }
    return undefined;
  }, [effectiveTreeData, chartDirection, loading, fitView]);

  // Branch collapse / expand
  const handleToggleCollapse = (key: string) => {
    setCollapsedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const handleExpandAll = () => {
    setCollapsedKeys(new Set());
  };

  const handleCollapseAll = () => {
    setCollapsedKeys(collectBranchKeys(effectiveTreeData));
  };

  return (
    <Card
      styles={{ body: { padding: 0 } }}
      style={{
        minHeight: 640,
        overflow: 'hidden',
        position: 'relative',
        backgroundColor: '#f8fafc',
      }}
    >
      <Spin spinning={loading}>
        {effectiveTreeData.length > 0 ? (
          <div
            ref={containerRef}
            style={{
              height: 'calc(100vh - 270px)',
              minHeight: 600,
              overflow: 'hidden',
              position: 'relative',
              cursor: isPanning ? 'grabbing' : 'grab',
              backgroundColor: '#f8fafc',
            }}
            onMouseDown={handleMouseDown}
          >
            <div
              ref={contentRef}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                transform: `translate(${offset.x}px, ${offset.y}px) scale(${zoom})`,
                transformOrigin: '0 0',
                transition: isPanning
                  ? 'none'
                  : 'transform 0.25s cubic-bezier(0.2, 0, 0, 1)',
                display: 'inline-flex',
                flexDirection: chartDirection === 'vertical' ? 'row' : 'column',
                gap: chartDirection === 'vertical' ? 48 : 36,
                padding: 60,
              }}
            >
              {effectiveTreeData.map((rootNode) =>
                chartDirection === 'vertical' ? (
                  <VerticalBranch
                    key={rootNode.key}
                    node={rootNode}
                    selectedId={selectedId}
                    hasParent={false}
                    direction={chartDirection}
                    collapsedKeys={collapsedKeys}
                    onToggleCollapse={handleToggleCollapse}
                    onSelectNode={handleSelectNode}
                    onOpenDrawer={onOpenDrawer ?? (() => {})}
                    isDragMoved={() => dragRef.current.moved}
                  />
                ) : (
                  <HorizontalBranch
                    key={rootNode.key}
                    node={rootNode}
                    selectedId={selectedId}
                    hasParent={false}
                    direction={chartDirection}
                    collapsedKeys={collapsedKeys}
                    onToggleCollapse={handleToggleCollapse}
                    onSelectNode={handleSelectNode}
                    onOpenDrawer={onOpenDrawer ?? (() => {})}
                    isDragMoved={() => dragRef.current.moved}
                  />
                ),
              )}
            </div>
          </div>
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="暂无组织架构数据"
            style={{ margin: '80px 0' }}
          />
        )}
      </Spin>

      {/* Floating Canvas Controls (Bottom Left) */}
      <div
        style={{
          position: 'absolute',
          bottom: 16,
          left: 16,
          zIndex: 10,
          backgroundColor: 'rgba(255, 255, 255, 0.96)',
          backdropFilter: 'blur(4px)',
          padding: '4px 8px',
          borderRadius: 8,
          boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)',
          border: '1px solid #e2e8f0',
          display: 'flex',
          alignItems: 'center',
        }}
      >
        <Space size={4}>
          <Tooltip title="自适应画布居中">
            <Button size="small" type="text" onClick={fitView}>
              居中适界
            </Button>
          </Tooltip>
          <Tooltip title="全部展开">
            <Button size="small" type="text" onClick={handleExpandAll}>
              全部展开
            </Button>
          </Tooltip>
          <Tooltip title="全部折叠">
            <Button size="small" type="text" onClick={handleCollapseAll}>
              全部折叠
            </Button>
          </Tooltip>
          <div
            style={{
              width: 1,
              height: 16,
              backgroundColor: '#e2e8f0',
              margin: '0 4px',
            }}
          />
          <Tooltip title="缩小">
            <Button size="small" type="text" onClick={handleZoomOut}>
              -
            </Button>
          </Tooltip>
          <span
            style={{
              fontSize: 12,
              color: 'rgba(0, 0, 0, 0.65)',
              minWidth: 42,
              textAlign: 'center',
              fontFamily: 'monospace',
            }}
          >
            {Math.round(zoom * 100)}%
          </span>
          <Tooltip title="放大">
            <Button size="small" type="text" onClick={handleZoomIn}>
              +
            </Button>
          </Tooltip>
          <Tooltip title="重置缩放到 100%">
            <Button size="small" type="text" onClick={handleResetZoom}>
              1:1
            </Button>
          </Tooltip>
        </Space>
      </div>

      {/* Re-expand Inspector Toggle Button when Collapsed */}
      {!inspectorOpen && currentSelectedOrg && (
        <Button
          type="default"
          icon={<ProfileOutlined style={{ color: '#1677ff' }} />}
          onClick={() => setInspectorOpen(true)}
          style={{
            position: 'absolute',
            top: 12,
            right: 12,
            zIndex: 10,
            backgroundColor: 'rgba(255, 255, 255, 0.96)',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.08)',
            border: '1px solid #e2e8f0',
            borderRadius: 6,
          }}
        >
          组织详情
        </Button>
      )}

      {/* Embedded Floating Inspector Panel */}
      {currentSelectedOrg && (
        <OrgInspectorPanel
          open={inspectorOpen}
          onClose={() => setInspectorOpen(false)}
          selectedOrg={currentSelectedOrg}
          parentOrg={currentParentOrg}
          directChildren={currentDirectChildren}
          totalDescendantCount={totalDescendantCount ?? 0}
          canCreate={canCreate}
          canUpdate={canUpdate}
          onOpenCreateChild={onOpenCreateChild ?? (() => {})}
          onOpenEdit={onOpenEdit ?? (() => {})}
          onSelectNode={handleSelectNode}
          onLocateNode={focusNode}
        />
      )}
    </Card>
  );
}
