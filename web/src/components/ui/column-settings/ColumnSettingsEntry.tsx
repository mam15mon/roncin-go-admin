import { SettingOutlined } from '@ant-design/icons';
import { Button, Modal, Popover, Tooltip } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import ColumnSettingsPanel from './ColumnSettingsPanel';
import type {
  ColumnFreezeTargetTitles,
  ColumnSettingsField,
  ColumnSettingsValue,
} from './types';

export interface ColumnSettingsEntryProps {
  /** 全量可配置字段；无权限字段由调用方裁剪，不得传入。 */
  fields: ColumnSettingsField[];
  /** 当前生效配置（即时真相源）。 */
  value: ColumnSettingsValue;
  /** 显隐和排序后实际首尾名称，包含末尾操作列。 */
  freezeTargets?: ColumnFreezeTargetTitles;
  /** 显隐或排序变更回调：浮层内每次交互立即触发，由调用方应用并持久化。 */
  onChange: (value: ColumnSettingsValue) => void;
  /** 恢复默认回调。 */
  onReset: () => void;
  /**
   * 增强版高级设置内容（如行配色）：提供时浮层底部出现「更多设置」，
   * 点击关闭浮层并打开二级弹窗展示该内容；草稿状态由调用方自行维护。
   */
  advanced?: React.ReactNode;
  /** 二级弹窗标题，默认「高级设置」。 */
  advancedTitle?: string;
  /** 二级弹窗宽度，默认 880。 */
  advancedWidth?: number;
  /** 禁用入口（如编辑进行中）；配合 disabledReason 提示原因。 */
  disabled?: boolean;
  disabledReason?: string;
  size?: 'small' | 'middle';
}

/**
 * 全站统一列设置入口：齿轮按钮锚定浮层，表格全程可见、操作即时生效；
 * 偏好持久化由调用方在 onChange/onReset 中完成，本组件不承担存储。
 * 外层结构在禁用切换时保持稳定，避免触发按钮 DOM 重建。
 */
export function ColumnSettingsEntry({
  fields,
  value,
  freezeTargets,
  onChange,
  onReset,
  advanced,
  advancedTitle,
  advancedWidth,
  disabled,
  disabledReason,
  size = 'middle',
}: ColumnSettingsEntryProps) {
  const [open, setOpen] = useState(false);
  // 每次打开递增 key 重挂浮层内容，使搜索与筛选从干净状态开始。
  const [session, setSession] = useState(0);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  // Popover 关闭动画可能保留旧内容与回调，禁用校验读取当前值。
  const disabledRef = useRef(disabled);
  disabledRef.current = disabled;
  useEffect(() => {
    if (disabled) {
      setOpen(false);
      setAdvancedOpen(false);
    }
  }, [disabled]);

  return (
    <>
      <Popover
        trigger="click"
        placement="bottom"
        open={open && !disabled}
        onOpenChange={(next) => {
          if (next && disabledRef.current) return;
          setOpen(next);
          if (next) setSession((prev) => prev + 1);
        }}
        content={
          <ColumnSettingsPanel
            key={session}
            fields={fields}
            value={value}
            freezeTargets={freezeTargets}
            onChange={(next) => {
              if (!disabledRef.current) onChange(next);
            }}
            onReset={() => {
              if (!disabledRef.current) onReset();
            }}
            onOpenAdvanced={
              advanced
                ? () => {
                    if (disabledRef.current) return;
                    setOpen(false);
                    setAdvancedOpen(true);
                  }
                : undefined
            }
          />
        }
      >
        <span>
          <Tooltip title={disabled ? (disabledReason ?? '列设置') : '列设置'}>
            <Button
              type="text"
              size={size}
              icon={<SettingOutlined />}
              aria-label="列设置"
              disabled={disabled}
            />
          </Tooltip>
        </span>
      </Popover>
      <Modal
        open={advancedOpen && !disabled}
        title={advancedTitle ?? '高级设置'}
        width={advancedWidth ?? 880}
        footer={null}
        onCancel={() => setAdvancedOpen(false)}
      >
        {advanced}
      </Modal>
    </>
  );
}
