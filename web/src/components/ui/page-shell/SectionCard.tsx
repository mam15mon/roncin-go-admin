import { DownOutlined, RightOutlined } from '@ant-design/icons';
import { Card, Space, Typography } from 'antd';
import React, { useState } from 'react';
import type { SectionCardProps } from './types';

const { Text } = Typography;

export const SectionCard: React.FC<SectionCardProps> = ({
  title,
  extra,
  children,
  collapsible = false,
  collapsed: controlledCollapsed,
  defaultCollapsed = false,
  onCollapseChange,
  style,
  bodyStyle,
  className,
}) => {
  const [uncontrolledCollapsed, setUncontrolledCollapsed] = useState(defaultCollapsed);
  const isControlled = controlledCollapsed !== undefined;
  const isCollapsed = isControlled ? controlledCollapsed : uncontrolledCollapsed;

  const handleToggle = () => {
    if (!collapsible) return;
    const nextState = !isCollapsed;
    if (!isControlled) {
      setUncontrolledCollapsed(nextState);
    }
    onCollapseChange?.(nextState);
  };

  const titleNode = (
    <div
      onClick={collapsible ? handleToggle : undefined}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 8,
        cursor: collapsible ? 'pointer' : 'default',
        userSelect: 'none',
      }}
    >
      <span
        style={{
          width: 3,
          height: 15,
          borderRadius: 2,
          backgroundColor: '#1677ff',
          display: 'inline-block',
          flexShrink: 0,
        }}
      />
      <Text strong style={{ fontSize: 14, color: 'rgba(0, 0, 0, 0.88)' }}>
        {title}
      </Text>
      {collapsible && (
        <span style={{ fontSize: 12, color: 'rgba(0, 0, 0, 0.45)', marginLeft: 2 }}>
          {isCollapsed ? <RightOutlined /> : <DownOutlined />}
        </span>
      )}
    </div>
  );

  return (
    <Card
      size="small"
      title={titleNode}
      extra={
        extra ? (
          <Space size={8} align="center">
            {extra}
          </Space>
        ) : null
      }
      variant="borderless"
      className={`roncin-section-card ${className || ''}`}
      style={{
        width: '100%',
        marginBottom: 12,
        backgroundColor: '#ffffff',
        borderRadius: 6,
        border: '1px solid #f0f0f0',
        boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.02)',
        ...style,
      }}
      styles={{
        header: {
          padding: '10px 14px',
          minHeight: 40,
          borderBottom: isCollapsed ? 'none' : '1px solid #f0f0f0',
        },
        body: {
          padding: '14px 16px',
          display: isCollapsed ? 'none' : 'block',
          ...bodyStyle,
        },
      }}
    >
      {children}
    </Card>
  );
};
