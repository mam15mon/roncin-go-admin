import { ApartmentOutlined } from '@ant-design/icons';
import { Tag } from 'antd';
import type { OrgTreeNode } from '../../organization-tree';
import { getOrganizationKindMeta } from './types';

export type CardProps = {
  node: OrgTreeNode;
  isSelected: boolean;
  hasParent: boolean;
  hasChildren: boolean;
  direction: 'vertical' | 'horizontal';
  isDragMoved: () => boolean;
  onSelect: () => void;
};

/** 组织树画布中的单个组织节点卡片。 */
export default function OrgCard({
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
        width: 'max-content',
        minWidth: 220,
        maxWidth: 360,
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
