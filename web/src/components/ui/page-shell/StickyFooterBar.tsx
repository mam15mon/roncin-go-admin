import { Space } from 'antd';
import React from 'react';
import type { StickyFooterBarProps } from './types';

export const StickyFooterBar: React.FC<StickyFooterBarProps> = ({
  info,
  children,
  style,
  className,
}) => {
  return (
    <div
      className={`roncin-sticky-footer-bar ${className || ''}`}
      style={{
        position: 'sticky',
        bottom: 0,
        zIndex: 15,
        display: 'flex',
        alignItems: 'center',
        justifyContent: info ? 'space-between' : 'center',
        padding: '10px 24px',
        backgroundColor: '#ffffff',
        borderTop: '1px solid #f0f0f0',
        boxShadow: '0 -2px 8px 0 rgba(0, 0, 0, 0.04)',
        marginTop: 16,
        ...style,
      }}
    >
      {info && <div>{info}</div>}
      <Space size={12} align="center">
        {children}
      </Space>
    </div>
  );
};
