import { DownOutlined, RightOutlined } from '@ant-design/icons';
import { Card, Space, Typography } from 'antd';
import React, { useId, useState } from 'react';
import type { SectionCardProps } from './types';

const { Text } = Typography;

export const SectionCard: React.FC<SectionCardProps> = ({
  id,
  sectionKey,
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
  const contentId = useId();
  const [uncontrolledCollapsed, setUncontrolledCollapsed] =
    useState(defaultCollapsed);
  const isControlled = controlledCollapsed !== undefined;
  const isCollapsed = isControlled
    ? controlledCollapsed
    : uncontrolledCollapsed;

  const handleToggle = () => {
    if (!collapsible) return;
    const nextState = !isCollapsed;
    if (!isControlled) {
      setUncontrolledCollapsed(nextState);
    }
    onCollapseChange?.(nextState);
  };

  const TitleElement = collapsible ? 'button' : 'div';
  const titleNode = (
    <TitleElement
      type={collapsible ? 'button' : undefined}
      className={collapsible ? 'roncin-section-toggle' : undefined}
      aria-expanded={collapsible ? !isCollapsed : undefined}
      aria-controls={collapsible ? contentId : undefined}
      onClick={collapsible ? handleToggle : undefined}
      style={{
        border: 0,
        background: 'transparent',
        padding: 0,
        font: 'inherit',
        textAlign: 'left',
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
      <Text strong style={{ fontSize: 13.5, color: '#0f172a' }}>
        {title}
      </Text>
      {collapsible && (
        <span
          aria-hidden="true"
          style={{ fontSize: 12, color: '#94a3b8', marginLeft: 2 }}
        >
          {isCollapsed ? <RightOutlined /> : <DownOutlined />}
        </span>
      )}
    </TitleElement>
  );

  return (
    <Card
      id={id || (sectionKey ? `section-${sectionKey}` : undefined)}
      data-section-key={sectionKey}
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
        marginBottom: 'var(--roncin-page-section-gap)',
        backgroundColor: '#ffffff',
        borderRadius: 8,
        border: '1px solid #e2e8f0',
        boxShadow:
          '0 1px 3px 0 rgba(0, 0, 0, 0.03), 0 1px 2px -1px rgba(0, 0, 0, 0.02)',
        ...style,
      }}
      styles={{
        header: {
          padding: '10px 14px',
          minHeight: 40,
          borderBottom: isCollapsed ? 'none' : '1px solid #f1f5f9',
        },
        body: {
          padding: '14px 16px',
          display: isCollapsed ? 'none' : 'block',
          ...bodyStyle,
        },
      }}
    >
      <div id={contentId} hidden={collapsible && isCollapsed}>
        {children}
      </div>
    </Card>
  );
};
