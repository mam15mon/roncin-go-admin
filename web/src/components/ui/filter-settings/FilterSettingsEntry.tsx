import { SettingOutlined } from '@ant-design/icons';
import { Button, Checkbox, Popover, Tooltip } from 'antd';
import React from 'react';
import type { FilterSettingsField } from './types';

export interface FilterSettingsEntryProps {
  fields: readonly FilterSettingsField[];
  visibleNames: readonly string[];
  onChange: (names: string[]) => void;
  onReset: () => void;
  disabled?: boolean;
}

/** 筛选设置入口：齿轮按钮打开勾选面板，勾选即时生效；移除字段会清除其条件并刷新列表。 */
export function FilterSettingsEntry({
  fields,
  visibleNames,
  onChange,
  onReset,
  disabled,
}: FilterSettingsEntryProps) {
  const visible = new Set(visibleNames);
  const content = (
    <div style={{ width: 360, background: '#ffffff' }}>
      <div style={{ color: '#5b6b82', fontSize: 12, marginBottom: 8 }}>
        取消勾选会同时清除该筛选条件并刷新列表
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
          gap: '6px 12px',
        }}
      >
        {fields.map((field) => (
          <Checkbox
            key={field.name}
            checked={visible.has(field.name)}
            disabled={field.required}
            onChange={(event) =>
              onChange(
                fields
                  .filter((f) =>
                    f.name === field.name
                      ? event.target.checked
                      : visible.has(f.name),
                  )
                  .map((f) => f.name),
              )
            }
          >
            {field.label}
          </Checkbox>
        ))}
      </div>
      <div style={{ marginTop: 10, textAlign: 'right' }}>
        <Button type="link" size="small" onClick={onReset}>
          恢复默认
        </Button>
      </div>
    </div>
  );
  return (
    <Popover
      trigger="click"
      placement="bottomRight"
      title="筛选设置"
      content={content}
    >
      <Tooltip title="筛选设置">
        <Button
          type="text"
          icon={<SettingOutlined />}
          aria-label="筛选设置"
          disabled={disabled}
        />
      </Tooltip>
    </Popover>
  );
}
