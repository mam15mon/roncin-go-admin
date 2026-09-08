import { ArrowLeftOutlined } from '@ant-design/icons';
import { Link } from '@umijs/max';
import { Button, Tooltip, Typography } from 'antd';
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
  const hasBreadcrumbs = Boolean(breadcrumbs && breadcrumbs.length > 0);

  return (
    <div
      className={`roncin-page-header-shell ${hasBreadcrumbs ? 'roncin-page-header-has-breadcrumbs' : ''} ${className || ''}`}
      style={{
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
      {/* 1. 顶层面包屑行 */}
      {hasBreadcrumbs && (
        <div className="roncin-page-header-breadcrumbs">
          {breadcrumbs?.map((crumb, idx) => {
            const isLast = idx === (breadcrumbs?.length ?? 0) - 1;
            return (
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
                  <span className="roncin-page-header-crumb-text">
                    {crumb.label}
                  </span>
                )}
                {!isLast && (
                  <span className="roncin-page-header-crumb-sep">/</span>
                )}
              </React.Fragment>
            );
          })}
        </div>
      )}

      {/* 2. 底层主体行 */}
      <div className="roncin-page-header-main-row">
        <div className="roncin-page-header-title-area">
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

          <Text strong className="roncin-page-header-heading-title">
            {title}
          </Text>

          {tags && <div className="roncin-page-header-tags">{tags}</div>}

          {subTitle && (
            <Text className="roncin-page-header-subtitle">
              {subTitle}
            </Text>
          )}
        </div>

        {extra && <div className="roncin-page-header-extra">{extra}</div>}
      </div>
    </div>
  );
};
