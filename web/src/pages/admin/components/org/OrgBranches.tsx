import type { OrgTreeNode } from '../../organization-tree';
import OrgCard from './OrgCard';

export type BranchProps = {
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

/** 垂直组织树分支：父节点在上、子节点横向排列在下。 */
export function VerticalBranch({
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

/** 水平组织树分支：父节点在左、子节点纵向排列在右。 */
export function HorizontalBranch({
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
