import { ProFormTextArea } from '@ant-design/pro-components';
import { Button, Form, Tag } from 'antd';
import React, { useEffect, useState } from 'react';
import { onSeaFormErrorReveal } from '../../sea-form-error-reveal';

/**
 * 就近折叠备注字段：空内容初始收起、有内容初始展开，展开后直接编辑。
 *
 * - 折叠只隐藏输入框（display:none），字段与校验规则常驻 Form store，
 *   值不丢失；收起有内容时展示「有备注」标记。
 * - 初始展开态由数据（详情回填 / 草稿恢复）到达后自动确定；用户手动
 *   收起或展开后，同单普通重绘尊重该选择，切单由页面重挂载重置。
 * - 提交校验失败时经 sea-form-error-reveal 链路强制展开以便滚动定位。
 */
export function SeaCollapsibleNotesField({
  name,
  label,
  placeholder,
  disabled,
}: {
  name: string;
  label: string;
  placeholder?: string;
  disabled?: boolean;
}) {
  const form = Form.useFormInstance();
  const value = Form.useWatch(name, form) as string | undefined;
  const hasContent = Boolean(typeof value === 'string' && value.trim());
  // null 表示用户尚未手动干预，展开态跟随是否有内容。
  const [manualExpanded, setManualExpanded] = useState<boolean | null>(null);
  const expanded = manualExpanded ?? hasContent;

  useEffect(
    () =>
      onSeaFormErrorReveal((fields) => {
        if (fields.some((field) => field.name[0] === name)) {
          setManualExpanded(true);
        }
      }),
    [name],
  );

  return (
    <div data-testid={`sea-notes-${name}`} style={{ width: '100%' }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          minHeight: 24,
          marginBottom: expanded ? 4 : 0,
        }}
      >
        <span
          style={{
            fontSize: 13,
            fontWeight: 500,
            color: 'rgba(0, 0, 0, 0.88)',
          }}
        >
          {label}
        </span>
        {hasContent && !expanded ? <Tag color="blue">有备注</Tag> : null}
        <Button
          type="link"
          size="small"
          disabled={disabled}
          onClick={() => setManualExpanded(!expanded)}
          style={{ padding: 0, height: 'auto', fontSize: 12 }}
        >
          {expanded ? '收起' : '展开'}
        </Button>
      </div>
      <div style={expanded ? undefined : { display: 'none' }}>
        <ProFormTextArea
          name={name}
          placeholder={placeholder}
          disabled={disabled}
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />
      </div>
    </div>
  );
}
