import { history } from '@umijs/max';
import { Space } from 'antd';
import React, { type ReactNode } from 'react';

export interface DocumentDetailBreadcrumbItem {
  label: string;
  path?: string;
}

export interface DocumentDetailLayoutProps {
  breadcrumbs?: DocumentDetailBreadcrumbItem[];
  code?: string;
  extraBreadcrumb?: ReactNode;
  actions?: ReactNode;
  timeline?: ReactNode;
  statusSection?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
}

export function DocumentDetailLayout({
  breadcrumbs = [],
  code,
  extraBreadcrumb,
  actions,
  timeline,
  statusSection,
  footer,
  children,
}: DocumentDetailLayoutProps) {
  return (
    <div style={{ marginBottom: 24 }}>
      {/* 顶部面包屑 */}
      {breadcrumbs.length > 0 && (
        <div style={{ padding: '8px 16px', fontSize: 13, color: '#64748b' }}>
          <Space size={6}>
            {breadcrumbs.map((item, index) => {
              const isLast = index === breadcrumbs.length - 1;
              return (
                <React.Fragment key={item.label}>
                  {index > 0 && <span>&gt;</span>}
                  {item.path && !isLast ? (
                    <a
                      style={{ color: '#64748b' }}
                      onClick={() => history.push(item.path as string)}
                    >
                      {item.label}
                    </a>
                  ) : (
                    <span
                      style={{
                        color: isLast ? '#1677ff' : '#64748b',
                        fontWeight: isLast ? 500 : 400,
                      }}
                    >
                      {item.label}
                    </span>
                  )}
                </React.Fragment>
              );
            })}
            {code && (
              <span
                style={{
                  color: '#0f172a',
                  fontWeight: 600,
                  marginLeft: 8,
                  fontFamily: 'monospace',
                }}
              >
                ({code})
              </span>
            )}
            {extraBreadcrumb}
          </Space>
        </div>
      )}

      {/* 顶部操作工具栏 */}
      {actions && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-start',
            flexWrap: 'nowrap',
            overflowX: 'auto',
            backgroundColor: '#ffffff',
            padding: '8px 16px',
            borderRadius: 6,
            border: '1px solid #e2e8f0',
            boxShadow: '0 1px 2px rgba(0, 0, 0, 0.02)',
            marginBottom: 12,
          }}
        >
          <Space size={8} wrap={false}>
            {actions}
          </Space>
        </div>
      )}

      {/* 状态与生命周期流转 */}
      {statusSection}

      {/* 审核 / 流转时间轴 */}
      {timeline}

      {/* 详情主内容区 */}
      {children}

      {/* 底部悬浮/操作栏 */}
      {footer}
    </div>
  );
}

export default DocumentDetailLayout;
