import { SettingOutlined } from '@ant-design/icons';
import { Button, ConfigProvider, Modal, Tooltip } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import ColumnSettingsPanel from './ColumnSettingsPanel';
import type { ColumnSettingsField, ColumnSettingsValue } from './types';

// 脱离 ProTable 的局部挂载点；全屏时浮层仍需位于浏览器全屏元素内。
function getSettingsPopupContainer(): HTMLElement {
  const fullscreenElement = document.fullscreenElement;
  return fullscreenElement instanceof HTMLElement
    ? fullscreenElement
    : document.body;
}

export interface ColumnSettingsEntryProps {
  /** 全量可配置字段；无权限字段由调用方裁剪，不得传入。 */
  fields: ColumnSettingsField[];
  /** 当前生效配置（即时真相源）。 */
  value: ColumnSettingsValue;
  /** 不参与普通字段排序的结构列冻结配置。 */
  structuralFields?: ColumnSettingsField[];
  /** 显隐或排序变更回调：弹窗内每次交互立即触发，由调用方应用并持久化。 */
  onChange: (value: ColumnSettingsValue) => void;
  /** 恢复默认回调。 */
  onReset: () => void;
  /**
   * 增强版高级设置内容（如行配色）：提供时弹窗底部出现「更多设置」，
   * 点击关闭弹窗并打开二级弹窗展示该内容；草稿状态由调用方自行维护。
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
 * 全站统一列设置入口：齿轮按钮打开全量铺排弹窗、操作即时生效；
 * 偏好持久化由调用方在 onChange/onReset 中完成，本组件不承担存储。
 * 外层结构在禁用切换时保持稳定，避免触发按钮 DOM 重建。
 */
export function ColumnSettingsEntry({
  fields,
  value,
  structuralFields,
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
  // 每次打开递增 key 重挂弹窗内容，使搜索与筛选从干净状态开始。
  const [session, setSession] = useState(0);
  const sessionRef = useRef(0);
  const activeSessionRef = useRef<number | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  // 关闭动画可能保留旧内容；只允许当前打开批次的回调写入偏好。
  const disabledRef = useRef(disabled);
  disabledRef.current = disabled;
  if (disabled) activeSessionRef.current = null;
  useEffect(
    () => () => {
      activeSessionRef.current = null;
    },
    [],
  );
  useEffect(() => {
    if (disabled) {
      setOpen(false);
      setAdvancedOpen(false);
    }
  }, [disabled]);

  return (
    <>
      <Tooltip title={disabled ? (disabledReason ?? '列设置') : '列设置'}>
        <Button
          type="text"
          size={size}
          icon={<SettingOutlined />}
          aria-label="列设置"
          disabled={disabled}
          onClick={() => {
            if (disabledRef.current) return;
            sessionRef.current += 1;
            activeSessionRef.current = sessionRef.current;
            setSession(sessionRef.current);
            setOpen(true);
          }}
        />
      </Tooltip>
      <ConfigProvider getPopupContainer={getSettingsPopupContainer}>
        <Modal
          title="列设置"
          open={open && !disabled}
          width="calc(100vw - 24px)"
          style={{ top: 12, paddingBottom: 12 }}
          styles={{ container: { padding: 16 } }}
          footer={null}
          onCancel={() => {
            activeSessionRef.current = null;
            setOpen(false);
          }}
          destroyOnHidden
        >
          <ColumnSettingsPanel
            key={session}
            fields={fields}
            structuralFields={structuralFields}
            value={value}
            onChange={(next) => {
              if (!disabledRef.current && activeSessionRef.current === session)
                onChange(next);
            }}
            onReset={() => {
              if (!disabledRef.current && activeSessionRef.current === session)
                onReset();
            }}
            onOpenAdvanced={
              advanced
                ? () => {
                    if (
                      disabledRef.current ||
                      activeSessionRef.current !== session
                    )
                      return;
                    activeSessionRef.current = null;
                    setOpen(false);
                    setAdvancedOpen(true);
                  }
                : undefined
            }
          />
        </Modal>
        <Modal
          open={advancedOpen && !disabled}
          title={advancedTitle ?? '高级设置'}
          width={advancedWidth ?? 880}
          footer={null}
          onCancel={() => setAdvancedOpen(false)}
        >
          {advanced}
        </Modal>
      </ConfigProvider>
    </>
  );
}
