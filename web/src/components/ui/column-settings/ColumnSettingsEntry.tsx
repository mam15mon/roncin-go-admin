import { SettingOutlined } from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import React from 'react';

export interface ColumnSettingsEntryProps {
  onClick: () => void;
  /** 禁用入口（如编辑进行中）；配合 disabledReason 提示原因。 */
  disabled?: boolean;
  disabledReason?: string;
  size?: 'small' | 'middle';
}

/** 全站统一列设置入口：齿轮图标 + 「列设置」同名提示与可访问名称。 */
export function ColumnSettingsEntry({
  onClick,
  disabled,
  disabledReason,
  size = 'middle',
}: ColumnSettingsEntryProps) {
  const button = (
    <Button
      type="text"
      size={size}
      icon={<SettingOutlined />}
      aria-label="列设置"
      disabled={disabled}
      onClick={onClick}
    />
  );
  return (
    <Tooltip title={disabled ? (disabledReason ?? '列设置') : '列设置'}>
      <span>{button}</span>
    </Tooltip>
  );
}
