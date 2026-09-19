import { Space } from 'antd';
import React, { type ReactNode } from 'react';

export interface DocumentDetailBreadcrumbItem {
  label: string;
  path?: string;
}

export interface DocumentDetailLayoutProps {
  /** @deprecated 面包屑统一由 PageHeaderShell 提供，此处不再渲染 */
  breadcrumbs?: DocumentDetailBreadcrumbItem[];
  /** @deprecated 单号由 PageHeaderShell title 提供 */
  code?: string;
  /** @deprecated 面包屑扩展项已废弃 */
  extraBreadcrumb?: ReactNode;
  actions?: ReactNode;
  timeline?: ReactNode;
  statusSection?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
}

export function DocumentDetailLayout({
  actions,
  timeline,
  statusSection,
  footer,
  children,
}: DocumentDetailLayoutProps) {
  return (
    <div style={{ marginBottom: 24 }}>
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
