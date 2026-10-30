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

/** 多控件字段使用原生分组标题，避免生成无法关联单个输入框的 label。 */
export function TemplateControlGroup({
  title,
  required = false,
  children,
}: {
  title: React.ReactNode;
  required?: boolean;
  children: React.ReactNode;
}) {
  return (
    <fieldset style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
      <legend
        style={{
          padding: 0,
          border: 0,
          width: 'auto',
          marginBottom: 4,
          fontSize: 13,
          lineHeight: '22px',
          color: 'rgba(0, 0, 0, 0.88)',
        }}
      >
        {required && (
          <span aria-hidden="true" style={{ color: '#ff4d4f', marginRight: 4 }}>
            *
          </span>
        )}
        {title}
      </legend>
      {children}
    </fieldset>
  );
}
