import React from 'react';

/**
 * 订单模板卡内的小节标题：品牌蓝竖标 + 分组名，不套子卡片。
 * 海运/空运出口表单的运输卡共用同一分组视觉。
 */
export function TemplateFieldGroup({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div style={{ display: 'grid', gap: 12, width: '100%' }}>
      <div style={{ display: 'flex', alignItems: 'center' }}>
        <div
          style={{
            width: 3,
            height: 14,
            backgroundColor: '#1677ff',
            borderRadius: 2,
            marginRight: 8,
          }}
        />
        <span style={{ fontSize: 13, fontWeight: 600, color: '#1f2329' }}>
          {title}
        </span>
      </div>
      {children}
    </div>
  );
}
