import { ArrowLeftOutlined } from '@ant-design/icons';
import { Button, Divider, Space, Typography } from 'antd';
import React from 'react';
import type { PageHeaderShellProps } from './types';

const { Text } = Typography;

export const PageHeaderShell: React.FC<PageHeaderShellProps> = ({
  title,
  subTitle,
  onBack,
  backText = '返回列表',
  breadcrumbs,
  tags,
  extra,
  sticky = true,
  style,
  className,
}) => {
  return (
    <div
      className={`roncin-page-header-shell ${className || ''}`}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: 12,
        height: 52,
        padding: '0 16px',
        backgroundColor: '#ffffff',
        borderBottom: '1px solid #f0f0f0',
        marginBottom: 12,
        borderRadius: 6,
        boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03)',
        ...(sticky
          ? {
              position: 'sticky',
              top: 84, // 48px Header + 36px TagsView
              zIndex: 18,
            }
          : {}),
        ...style,
      }}
    >
      {/* Left: Navigation & Title */}
      <Space size={8} align="center" style={{ minWidth: 0, flex: 1 }}>
        {onBack && (
          <>
            <Button
              type="text"
              size="small"
              icon={<ArrowLeftOutlined />}
              onClick={onBack}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                fontWeight: 500,
                color: 'rgba(0, 0, 0, 0.65)',
                padding: '2px 6px',
              }}
            >
              {backText}
            </Button>
            <Divider vertical style={{ margin: '0 4px' }} />
          </>
        )}

        {breadcrumbs &&
          breadcrumbs.length > 0 &&
          breadcrumbs.map((crumb, idx) => (
            <React.Fragment key={crumb.label || idx}>
              {crumb.onClick ? (
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0, color: 'rgba(0, 0, 0, 0.45)', height: 'auto' }}
                  onClick={crumb.onClick}
                >
                  {crumb.label}
                </Button>
              ) : (
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {crumb.label}
                </Text>
              )}
              <span style={{ color: 'rgba(0, 0, 0, 0.3)' }}>/</span>
            </React.Fragment>
          ))}

        <Text strong style={{ fontSize: 15, color: 'rgba(0, 0, 0, 0.88)' }}>
          {title}
        </Text>

        {subTitle && (
          <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
            {subTitle}
          </Text>
        )}

        {tags && <span style={{ marginLeft: 4 }}>{tags}</span>}
      </Space>

      {/* Right: Actions */}
      {extra && <Space size={8} align="center">{extra}</Space>}
    </div>
  );
};
