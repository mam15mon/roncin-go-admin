import { ArrowLeftOutlined } from '@ant-design/icons';
import { Link } from '@umijs/max';
import { Button, Space, Tooltip, Typography } from 'antd';
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
      <Space size={4} align="center" style={{ minWidth: 0, flex: 1 }}>
        {onBack && (
          <Tooltip title={backText}>
            <Button
              type="text"
              size="small"
              className="roncin-page-header-back-btn"
              icon={<ArrowLeftOutlined />}
              onClick={onBack}
              aria-label={backText}
            />
          </Tooltip>
        )}

        {breadcrumbs &&
          breadcrumbs.length > 0 &&
          breadcrumbs.map((crumb, idx) => (
            <React.Fragment key={crumb.label || idx}>
              {crumb.href ? (
                <Link
                  to={crumb.href}
                  className="roncin-page-header-crumb-link"
                  onClick={crumb.onClick}
                >
                  {crumb.label}
                </Link>
              ) : crumb.onClick ? (
                <Button
                  type="link"
                  size="small"
                  className="roncin-page-header-crumb-btn"
                  onClick={crumb.onClick}
                >
                  {crumb.label}
                </Button>
              ) : (
                <Text className="roncin-page-header-crumb-text">
                  {crumb.label}
                </Text>
              )}
              <span className="roncin-page-header-crumb-sep">/</span>
            </React.Fragment>
          ))}

        <Text strong className="roncin-page-header-title">
          {title}
        </Text>

        {subTitle && (
          <Text className="roncin-page-header-subtitle">
            {subTitle}
          </Text>
        )}

        {tags && <span style={{ marginLeft: 6 }}>{tags}</span>}
      </Space>

      {/* Right: Actions */}
      {extra && <Space size={8} align="center">{extra}</Space>}
    </div>
  );
};
