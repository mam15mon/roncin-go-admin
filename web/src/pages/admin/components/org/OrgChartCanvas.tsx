import { ApartmentOutlined, ProfileOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Space, Spin, Tag, Tooltip } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { buildOrgTree, type OrgTreeNode } from '../../organization-tree';
import OrgInspectorPanel from './OrgInspectorPanel';
import { getOrganizationKindMeta } from './types';

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

type BranchProps = {
  node: OrgTreeNode;
  selectedId: string;
  hasParent: boolean;
  direction: 'vertical' | 'horizontal';
  collapsedKeys: Set<string>;
  onToggleCollapse: (key: string) => void;
  onSelectNode: (id: string) => void;
  onOpenDrawer: () => void;
  isDragMoved: () => boolean;
};

type CardProps = {
  node: OrgTreeNode;
  isSelected: boolean;
  hasParent: boolean;
  hasChildren: boolean;
  direction: 'vertical' | 'horizontal';
  isDragMoved: () => boolean;
  onSelect: () => void;
};

function OrgCard({
  node,
  isSelected,
  hasParent,
  hasChildren,
  direction,
  isDragMoved,
  onSelect,
}: CardProps) {
  const kindMeta = getOrganizationKindMeta(node.kind);
  const childrenCount = node.children?.length ?? 0;
  const isVertical = direction === 'vertical';

  return (
    <div
      data-node-id={node.key}
      onClick={() => {
        if (!isDragMoved()) {
          onSelect();
        }
      }}
      style={{
        width: 220,
        height: 82,
        backgroundColor: '#ffffff',
        borderRadius: 8,
        border: isSelected ? '2px solid #1677ff' : '1px solid #e2e8f0',
        boxShadow: isSelected
          ? '0 4px 14px rgba(22, 119, 255, 0.22)'
          : '0 1px 3px rgba(0, 0, 0, 0.05)',
        padding: '10px 12px',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        cursor: 'pointer',
        boxSizing: 'border-box',
        position: 'relative',
        transition: 'border-color 0.15s, box-shadow 0.15s',
        userSelect: 'none',
        flexShrink: 0,
        zIndex: 2,
      }}
    >
      {/* Anchor Port Dot - Incoming from Parent */}
      {hasParent && (
        <div
          style={{
            position: 'absolute',
            ...(isVertical
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
            width: 7,
            height: 7,
            borderRadius: '50%',
            backgroundColor: '#1677ff',
            border: '2px solid #ffffff',
            boxShadow: '0 1px 2px rgba(0, 0, 0, 0.25)',
            zIndex: 5,
          }}
        />
      )}

      {/* Anchor Port Dot - Outgoing to Children */}
      {hasChildren && (
        <div
          style={{
            position: 'absolute',
            ...(isVertical
              ? {
                  bottom: -4,
                  left: '50%',
                  transform: 'translateX(-50%)',
                }
              : {
                  right: -4,
                  top: '50%',
                  transform: 'translateY(-50%)',
                }),
            width: 7,
            height: 7,
            borderRadius: '50%',
            backgroundColor: '#1677ff',
            border: '2px solid #ffffff',
            boxShadow: '0 1px 2px rgba(0, 0, 0, 0.25)',
            zIndex: 5,
          }}
        />
      )}

      {/* Top Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 8,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            minWidth: 0,
            flex: 1,
          }}
        >
          <ApartmentOutlined
            style={{
              color: isSelected ? '#1677ff' : 'rgba(0, 0, 0, 0.45)',
              fontSize: 14,
              flexShrink: 0,
            }}
          />
          <span
            style={{
              fontWeight: 600,
              fontSize: 13,
              color: isSelected ? '#1677ff' : 'rgba(0, 0, 0, 0.88)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
            title={node.title}
          >
            {node.title || '未命名组织'}
          </span>
        </div>
        {node.enabled ? (
          <Tag
            color="success"
            variant="filled"
            style={{
              margin: 0,
              fontSize: 10,
              lineHeight: '16px',
              padding: '0 4px',
              flexShrink: 0,
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
              flexShrink: 0,
            }}
          >
            停用
          </Tag>
        )}
      </div>

      {/* Bottom Meta */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          fontSize: 11,
          color: 'rgba(0, 0, 0, 0.45)',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            minWidth: 0,
          }}
        >
          {kindMeta && (
            <Tag
              color={kindMeta.color}
              variant="filled"
              style={{
                margin: 0,
                fontSize: 10,
                lineHeight: '16px',
                padding: '0 4px',
                flexShrink: 0,
              }}
            >
              {kindMeta.label}
            </Tag>
          )}
          <span
            style={{
              fontFamily: 'monospace',
              color: 'rgba(0, 0, 0, 0.45)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              maxWidth: 76,
            }}
            title={node.code}
          >
            {node.code || '-'}
          </span>
        </div>

        {childrenCount > 0 && (
          <span
            style={{
              color: '#1677ff',
              fontWeight: 500,
              fontSize: 11,
              flexShrink: 0,
            }}
          >
            {childrenCount} 个下级
          </span>
        )}
      </div>
    </div>
  );
}

function VerticalBranch({
  node,
  selectedId,
  hasParent,
  direction,
  collapsedKeys,
  onToggleCollapse,
  onSelectNode,
  onOpenDrawer,
  isDragMoved,
}: BranchProps) {
  const isSelected = node.key === selectedId;
  const children = node.children ?? [];
  const hasChildren = children.length > 0;
  const isCollapsed = collapsedKeys.has(node.key);

  return (
    <div
      style={{
        display: 'inline-flex',
        flexDirection: 'column',
        alignItems: 'center',
        verticalAlign: 'top',
      }}
    >
      <OrgCard
        node={node}
        isSelected={isSelected}
        hasParent={hasParent}
        hasChildren={hasChildren}
        direction={direction}
        isDragMoved={isDragMoved}
        onSelect={() => {
          onSelectNode(node.key);
          onOpenDrawer();
        }}
      />

      {/* Subtree when expanded */}
      {hasChildren && !isCollapsed && (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            width: '100%',
          }}
        >
          {/* Vertical stem down from parent */}
          <div
            style={{
              width: 2,
              height: 24,
              backgroundColor: '#94a3b8',
              position: 'relative',
            }}
          >
            <button
              type="button"
              data-interactive="true"
              onClick={(e) => {
                e.stopPropagation();
                onToggleCollapse(node.key);
              }}
              title="折叠下级"
              style={{
                position: 'absolute',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
                width: 16,
                height: 16,
                borderRadius: '50%',
                border: '1px solid #cbd5e1',
                backgroundColor: '#ffffff',
                color: '#64748b',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 12,
                lineHeight: 1,
                cursor: 'pointer',
                padding: 0,
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.08)',
                zIndex: 4,
              }}
            >
              -
            </button>
          </div>

          {/* Children container */}
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            {children.map((child, idx) => (
              <div
                key={child.key}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  position: 'relative',
                  padding: '0 16px',
                }}
              >
                {/* Horizontal branch line joining siblings */}
                {children.length > 1 && (
                  <div
                    style={{
                      position: 'absolute',
                      top: 0,
                      height: 2,
                      backgroundColor: '#94a3b8',
                      left: idx === 0 ? '50%' : 0,
                      right: idx === children.length - 1 ? '50%' : 0,
                    }}
                  />
                )}

                {/* Vertical stem down into this child */}
                <div
                  style={{
                    width: 2,
                    height: 20,
                    backgroundColor: '#94a3b8',
                  }}
                />

                {/* Recursive branch */}
                <VerticalBranch
                  node={child}
                  selectedId={selectedId}
                  hasParent={true}
                  direction={direction}
                  collapsedKeys={collapsedKeys}
                  onToggleCollapse={onToggleCollapse}
                  onSelectNode={onSelectNode}
                  onOpenDrawer={onOpenDrawer}
                  isDragMoved={isDragMoved}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Collapsed Indicator Button */}
      {hasChildren && isCollapsed && (
        <div
          style={{
            position: 'relative',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
          }}
        >
          <div
            style={{
              width: 2,
              height: 14,
              backgroundColor: '#94a3b8',
            }}
          />
          <button
            type="button"
            data-interactive="true"
            onClick={(e) => {
              e.stopPropagation();
              onToggleCollapse(node.key);
            }}
            title={`展开 ${children.length} 个下级`}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 2,
              padding: '2px 8px',
              fontSize: 11,
              fontWeight: 600,
              borderRadius: 12,
              border: '1px solid #93c5fd',
              backgroundColor: '#eff6ff',
              color: '#1677ff',
              cursor: 'pointer',
              boxShadow: '0 1px 3px rgba(22, 119, 255, 0.15)',
              zIndex: 4,
            }}
          >
            +{children.length}
          </button>
        </div>
      )}
    </div>
  );
}

function HorizontalBranch({
  node,
  selectedId,
  hasParent,
  direction,
  collapsedKeys,
  onToggleCollapse,
  onSelectNode,
  onOpenDrawer,
  isDragMoved,
}: BranchProps) {
  const isSelected = node.key === selectedId;
  const children = node.children ?? [];
  const hasChildren = children.length > 0;
  const isCollapsed = collapsedKeys.has(node.key);

  return (
    <div
      style={{
        display: 'inline-flex',
        flexDirection: 'row',
        alignItems: 'center',
      }}
    >
      <OrgCard
        node={node}
        isSelected={isSelected}
        hasParent={hasParent}
        hasChildren={hasChildren}
        direction={direction}
        isDragMoved={isDragMoved}
        onSelect={() => {
          onSelectNode(node.key);
          onOpenDrawer();
        }}
      />

      {/* Subtree when expanded */}
      {hasChildren && !isCollapsed && (
        <div
          style={{
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
          }}
        >
          {/* Horizontal stem right from parent */}
          <div
            style={{
              width: 28,
              height: 2,
              backgroundColor: '#94a3b8',
              position: 'relative',
              flexShrink: 0,
            }}
          >
            <button
              type="button"
              data-interactive="true"
              onClick={(e) => {
                e.stopPropagation();
                onToggleCollapse(node.key);
              }}
              title="折叠下级"
              style={{
                position: 'absolute',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
                width: 16,
                height: 16,
                borderRadius: '50%',
                border: '1px solid #cbd5e1',
                backgroundColor: '#ffffff',
                color: '#64748b',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 12,
                lineHeight: 1,
                cursor: 'pointer',
                padding: 0,
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.08)',
                zIndex: 4,
              }}
            >
              -
            </button>
          </div>

          {/* Children container */}
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
            }}
          >
            {children.map((child, idx) => (
              <div
                key={child.key}
                style={{
                  display: 'flex',
                  flexDirection: 'row',
                  alignItems: 'center',
                  position: 'relative',
                  padding: '12px 0',
                }}
              >
                {/* Vertical bus bar joining siblings */}
                {children.length > 1 && (
                  <div
                    style={{
                      position: 'absolute',
                      left: 0,
                      width: 2,
                      backgroundColor: '#94a3b8',
                      top: idx === 0 ? '50%' : 0,
                      bottom: idx === children.length - 1 ? '50%' : 0,
                    }}
                  />
                )}

                {/* Horizontal stem into this child */}
                <div
                  style={{
                    width: 24,
                    height: 2,
                    backgroundColor: '#94a3b8',
                    flexShrink: 0,
                  }}
                />

                {/* Recursive branch */}
                <HorizontalBranch
                  node={child}
                  selectedId={selectedId}
                  hasParent={true}
                  direction={direction}
                  collapsedKeys={collapsedKeys}
                  onToggleCollapse={onToggleCollapse}
                  onSelectNode={onSelectNode}
                  onOpenDrawer={onOpenDrawer}
                  isDragMoved={isDragMoved}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Collapsed Indicator Button */}
      {hasChildren && isCollapsed && (
        <div
          style={{
            position: 'relative',
            display: 'flex',
            alignItems: 'center',
            flexShrink: 0,
          }}
        >
          <div
            style={{
              width: 14,
              height: 2,
              backgroundColor: '#94a3b8',
            }}
          />
          <button
            type="button"
            data-interactive="true"
            onClick={(e) => {
              e.stopPropagation();
              onToggleCollapse(node.key);
            }}
            title={`展开 ${children.length} 个下级`}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 2,
              padding: '2px 8px',
              fontSize: 11,
              fontWeight: 600,
              borderRadius: 12,
              border: '1px solid #93c5fd',
              backgroundColor: '#eff6ff',
              color: '#1677ff',
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              boxShadow: '0 1px 3px rgba(22, 119, 255, 0.15)',
              zIndex: 4,
            }}
          >
            +{children.length}
          </button>
        </div>
      )}
    </div>
  );
}

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

  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [isPanning, setIsPanning] = useState(false);
  const [collapsedKeys, setCollapsedKeys] = useState<Set<string>>(new Set());
  const [inspectorOpen, setInspectorOpen] = useState(true);

  const offsetRef = useRef(offset);
  useEffect(() => {
    offsetRef.current = offset;
  }, [offset]);

  const isPanningRef = useRef(false);
  const dragRef = useRef({
    startX: 0,
    startY: 0,
    startOffsetX: 0,
    startOffsetY: 0,
    moved: false,
  });

  // Reconcile tree data: prefer treeData prop, fallback to reconstructing from graphData
  const effectiveTreeData = useMemo(() => {
    if (treeData && treeData.length > 0) return treeData;
    if (!graphData?.nodes || graphData.nodes.length === 0) return [];

    const orgs: API.AdminOrganization[] = graphData.nodes.map((n) => {
      const d = (n.data || n) as API.AdminOrganization;
      return {
        id: d.id,
        name: d.name,
        code: d.code,
        kind: d.kind,
        enabled: d.enabled,
        parentId: d.parentId,
      };
    });
    return buildOrgTree(orgs).treeData;
  }, [treeData, graphData]);

  // Derive selected organization when not explicitly provided
  const currentSelectedOrg = useMemo(() => {
    if (selectedOrg !== undefined) return selectedOrg;
    if (!selectedId) return null;
    const findNode = (nodes: OrgTreeNode[]): API.AdminOrganization | null => {
      for (const node of nodes) {
        if (node.key === selectedId) return node.raw;
        if (node.children) {
          const res = findNode(node.children);
          if (res) return res;
        }
      }
      return null;
    };
    return findNode(effectiveTreeData);
  }, [selectedOrg, selectedId, effectiveTreeData]);

  // Derive direct children when not explicitly provided
  const currentDirectChildren = useMemo(() => {
    if (directChildren !== undefined) return directChildren;
    if (!selectedId) return [];
    const findChildren = (nodes: OrgTreeNode[]): API.AdminOrganization[] => {
      for (const node of nodes) {
        if (node.key === selectedId) {
          return node.children?.map((c) => c.raw) ?? [];
        }
        if (node.children) {
          const res = findChildren(node.children);
          if (res.length > 0) return res;
        }
      }
      return [];
    };
    return findChildren(effectiveTreeData);
  }, [directChildren, selectedId, effectiveTreeData]);

  // Derive parent organization when not explicitly provided
  const currentParentOrg = useMemo(() => {
    if (parentOrg !== undefined) return parentOrg;
    if (!currentSelectedOrg?.parentId) return null;
    const findParent = (nodes: OrgTreeNode[]): API.AdminOrganization | null => {
      for (const node of nodes) {
        if (node.key === currentSelectedOrg.parentId) return node.raw;
        if (node.children) {
          const res = findParent(node.children);
          if (res) return res;
        }
      }
      return null;
    };
    return findParent(effectiveTreeData);
  }, [parentOrg, currentSelectedOrg, effectiveTreeData]);

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
      const findAncestors = (
        nodes: OrgTreeNode[],
        targetId: string,
        path: string[] = [],
      ): string[] | null => {
        for (const n of nodes) {
          if (n.key === targetId) return path;
          if (n.children && n.children.length > 0) {
            const res = findAncestors(n.children, targetId, [...path, n.key]);
            if (res) return res;
          }
        }
        return null;
      };

      const ancestors = findAncestors(effectiveTreeData, id);
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

      const panToElement = () => {
        if (!containerRef.current) return;
        const targetEl = containerRef.current.querySelector(
          `[data-node-id="${id}"]`,
        ) as HTMLElement | null;
        if (!targetEl) return;

        const containerRect = containerRef.current.getBoundingClientRect();
        const nodeRect = targetEl.getBoundingClientRect();
        const containerWidth = containerRect.width;
        const containerHeight = containerRect.height;

        // Reserve space for inspector panel on the right (390px) if open
        const rightReserve = inspectorOpen ? 390 : 0;
        const safeCenterX = (containerWidth - rightReserve) / 2;
        const safeCenterY = containerHeight / 2;

        const nodeCenterX =
          nodeRect.left + nodeRect.width / 2 - containerRect.left;
        const nodeCenterY =
          nodeRect.top + nodeRect.height / 2 - containerRect.top;

        const deltaX = safeCenterX - nodeCenterX;
        const deltaY = safeCenterY - nodeCenterY;

        setOffset((prev) => ({
          x: Math.round(prev.x + deltaX),
          y: Math.round(prev.y + deltaY),
        }));
      };

      requestAnimationFrame(() => {
        setTimeout(panToElement, 30);
      });
    },
    [effectiveTreeData, inspectorOpen],
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

  // Fit View logic with inspector safe area compensation
  const fitView = useCallback(() => {
    if (!containerRef.current || !contentRef.current) return;
    const containerWidth = containerRef.current.clientWidth;
    const containerHeight = containerRef.current.clientHeight;
    const contentWidth =
      contentRef.current.offsetWidth || contentRef.current.scrollWidth;
    const contentHeight =
      contentRef.current.offsetHeight || contentRef.current.scrollHeight;

    if (!containerWidth || !containerHeight || !contentWidth || !contentHeight)
      return;

    const rightReserve = inspectorOpen && currentSelectedOrg ? 390 : 0;
    const padding = 40;
    const availableWidth = Math.max(
      containerWidth - rightReserve - padding * 2,
      100,
    );
    const availableHeight = Math.max(containerHeight - padding * 2, 100);

    const scaleX = availableWidth / contentWidth;
    const scaleY = availableHeight / contentHeight;
    const fitScale = Math.min(Math.max(Math.min(scaleX, scaleY), 0.35), 1.15);

    const newOffsetX =
      (containerWidth - rightReserve - contentWidth * fitScale) / 2;
    const newOffsetY =
      chartDirection === 'vertical'
        ? Math.max(24, (containerHeight - contentHeight * fitScale) / 3)
        : (containerHeight - contentHeight * fitScale) / 2;

    setZoom(fitScale);
    setOffset({ x: Math.round(newOffsetX), y: Math.round(newOffsetY) });
  }, [chartDirection, inspectorOpen, currentSelectedOrg]);

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

  // Mouse wheel zoom listener (passive: false to prevent document scroll)
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();
      const rect = container.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const zoomFactor = e.deltaY < 0 ? 1.1 : 0.9;
      setZoom((prevZoom) => {
        const nextZoom = Math.min(Math.max(prevZoom * zoomFactor, 0.3), 2.5);
        if (nextZoom === prevZoom) return prevZoom;

        setOffset((prevOffset) => {
          const contentX = (mouseX - prevOffset.x) / prevZoom;
          const contentY = (mouseY - prevOffset.y) / prevZoom;
          return {
            x: mouseX - contentX * nextZoom,
            y: mouseY - contentY * nextZoom,
          };
        });
        return nextZoom;
      });
    };

    container.addEventListener('wheel', handleWheel, { passive: false });
    return () => {
      container.removeEventListener('wheel', handleWheel);
    };
  }, []);

  // Canvas Pan handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0 && e.button !== 1) return;
    const target = e.target as HTMLElement;
    if (target.closest('button, [data-interactive="true"]')) {
      return;
    }
    isPanningRef.current = true;
    setIsPanning(true);
    dragRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      startOffsetX: offsetRef.current.x,
      startOffsetY: offsetRef.current.y,
      moved: false,
    };
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isPanningRef.current) return;
      const dx = e.clientX - dragRef.current.startX;
      const dy = e.clientY - dragRef.current.startY;
      if (Math.abs(dx) > 4 || Math.abs(dy) > 4) {
        dragRef.current.moved = true;
      }
      setOffset({
        x: dragRef.current.startOffsetX + dx,
        y: dragRef.current.startOffsetY + dy,
      });
    };

    const handleMouseUp = () => {
      if (isPanningRef.current) {
        isPanningRef.current = false;
        setIsPanning(false);
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

  // Zoom helpers for floating controls
  const handleZoomIn = () => {
    if (!containerRef.current) return;
    const nextZoom = Math.min(zoom * 1.2, 2.5);
    const cx = containerRef.current.clientWidth / 2;
    const cy = containerRef.current.clientHeight / 2;
    setOffset({
      x: cx - ((cx - offset.x) / zoom) * nextZoom,
      y: cy - ((cy - offset.y) / zoom) * nextZoom,
    });
    setZoom(nextZoom);
  };

  const handleZoomOut = () => {
    if (!containerRef.current) return;
    const nextZoom = Math.max(zoom * 0.8, 0.3);
    const cx = containerRef.current.clientWidth / 2;
    const cy = containerRef.current.clientHeight / 2;
    setOffset({
      x: cx - ((cx - offset.x) / zoom) * nextZoom,
      y: cy - ((cy - offset.y) / zoom) * nextZoom,
    });
    setZoom(nextZoom);
  };

  const handleResetZoom = () => {
    if (!containerRef.current || !contentRef.current) return;
    const cw = contentRef.current.offsetWidth || contentRef.current.scrollWidth;
    const ch =
      contentRef.current.offsetHeight || contentRef.current.scrollHeight;
    const vw = containerRef.current.clientWidth;
    const vh = containerRef.current.clientHeight;
    setZoom(1);
    setOffset({
      x: Math.max(24, (vw - cw) / 2),
      y: chartDirection === 'vertical' ? 36 : Math.max(24, (vh - ch) / 2),
    });
  };

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
    const keys = new Set<string>();
    const traverse = (nodes: OrgTreeNode[]) => {
      for (const node of nodes) {
        if (node.children && node.children.length > 0) {
          keys.add(node.key);
          traverse(node.children);
        }
      }
    };
    traverse(effectiveTreeData);
    setCollapsedKeys(keys);
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
