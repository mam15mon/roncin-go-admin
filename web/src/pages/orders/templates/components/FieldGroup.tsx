import React from 'react';

/**
 * 订单模板卡内的小节以中等字重与留白分组，蓝色竖标只保留在外层卡片。
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
      <div
        style={{
          fontSize: 13,
          fontWeight: 500,
          color: '#475569',
          lineHeight: '20px',
        }}
      >
        {title}
      </div>
      {children}
    </div>
  );
}
