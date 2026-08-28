import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  HolderOutlined,
  VerticalAlignTopOutlined,
} from '@ant-design/icons';
import { Checkbox, Space, Tooltip } from 'antd';
import React, { useState } from 'react';
import type { FinanceFieldMeta } from './fields-meta';

export interface FieldConfigCardProps {
  field: FinanceFieldMeta;
  checked: boolean;
  globalIndex: number;
  onToggle: () => void;
  onDragStart: (key: string) => void;
  onDrop: (key: string) => void;
  onMoveToTop: (key: string) => void;
  onMoveUp: (key: string) => void;
  onMoveDown: (key: string) => void;
}

// 采用 React.memo 隔离 153 个卡片的渲染，消除全量 diff，大幅提升拖拽帧率
export const FieldConfigCard = React.memo(function FieldConfigCard({
  field,
  checked,
  globalIndex,
  onToggle,
  onDragStart,
  onDrop,
  onMoveToTop,
  onMoveUp,
  onMoveDown,
}: FieldConfigCardProps) {
  const [isDragTarget, setIsDragTarget] = useState(false);

  return (
    <div
      draggable
      onDragStart={(e) => {
        onDragStart(field.key);
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', field.key);
        (e.currentTarget as HTMLElement).style.opacity = '0.4';
      }}
      onDragEnd={(e) => {
        (e.currentTarget as HTMLElement).style.opacity = '1';
        setIsDragTarget(false);
      }}
      onDragOver={(e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
      }}
      onDragEnter={() => setIsDragTarget(true)}
      onDragLeave={() => setIsDragTarget(false)}
      onDrop={(e) => {
        e.preventDefault();
        setIsDragTarget(false);
        onDrop(field.key);
      }}
      onClick={onToggle}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '6px 8px',
        borderRadius: 4,
        border: isDragTarget
          ? '2px dashed #1677ff'
          : `1px solid ${checked ? '#91caff' : '#f0f0f0'}`,
        background: isDragTarget
          ? '#e6f4ff'
          : checked
            ? '#e6f4ff'
            : '#fafafa',
        cursor: 'pointer',
        transition:
          'border-color 0.12s, background-color 0.12s, box-shadow 0.12s',
        userSelect: 'none',
        willChange: 'transform',
        transform: 'translateZ(0)',
      }}
    >
      <Space size={6} style={{ overflow: 'hidden', flex: 1 }}>
        <Tooltip title="按住拖拽可调整前后顺序">
          <span
            style={{
              cursor: 'grab',
              color: '#bfbfbf',
              display: 'flex',
              alignItems: 'center',
            }}
            onMouseDown={(e) => e.stopPropagation()}
          >
            <HolderOutlined />
          </span>
        </Tooltip>
        <span
          style={{
            fontSize: 11,
            fontWeight: 600,
            color: checked ? '#1677ff' : '#8c8c8c',
            minWidth: 26,
            display: 'inline-block',
          }}
        >
          #{globalIndex}
        </span>
        <span
          style={{
            fontSize: 12,
            fontWeight: checked ? 500 : 400,
            color: checked ? '#1677ff' : '#262626',
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis',
            overflow: 'hidden',
          }}
        >
          {field.name}
        </span>
      </Space>

      {/* 右侧微调按钮与 Checkbox */}
      <Space size={4} style={{ marginLeft: 4 }}>
        <Tooltip title="置顶">
          <VerticalAlignTopOutlined
            style={{ fontSize: 11, color: '#8c8c8c' }}
            onClick={(e) => {
              e.stopPropagation();
              onMoveToTop(field.key);
            }}
          />
        </Tooltip>
        <Tooltip title="上移">
          <ArrowUpOutlined
            style={{ fontSize: 11, color: '#8c8c8c' }}
            onClick={(e) => {
              e.stopPropagation();
              onMoveUp(field.key);
            }}
          />
        </Tooltip>
        <Tooltip title="下移">
          <ArrowDownOutlined
            style={{ fontSize: 11, color: '#8c8c8c' }}
            onClick={(e) => {
              e.stopPropagation();
              onMoveDown(field.key);
            }}
          />
        </Tooltip>
        <Checkbox checked={checked} style={{ pointerEvents: 'none' }} />
      </Space>
    </div>
  );
});
