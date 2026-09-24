import { App } from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import type {
  ColumnSettingsField,
  ColumnSettingsValue,
} from '@/components/ui/column-settings';
import {
  ColumnSettingsModal,
  defaultColumnSettingsValue,
} from '@/components/ui/column-settings';
import { FEE_LEDGER_FIELDS, getDefaultRowColors } from './fields-meta';
import RowColorSettings, { type RowColorsConfig } from './RowColorSettings';
import type { FinanceLedgerViewConfig } from './types';

export interface TableColumnConfigModalProps {
  open: boolean;
  onClose: () => void;
  /** 当前生效的本地视图配置；缺省时使用默认列与默认配色。 */
  currentConfig?: FinanceLedgerViewConfig;
  /**
   * 持久化到浏览器本地；返回 false 表示写入失败，弹窗保持打开并保留草稿，
   * 不得声称已保存。
   */
  onSave: (config: FinanceLedgerViewConfig) => boolean;
}

/**
 * 费用台账列设置（增强版）：统一弹窗 + 「高级设置」页签（行配色）。
 * 全部草稿式编辑，保存时由调用方写入浏览器本地偏好，失败保留弹窗草稿。
 */
export function TableColumnConfigModal({
  open,
  onClose,
  currentConfig,
  onSave,
}: TableColumnConfigModalProps) {
  const { message } = App.useApp();
  const [rowColors, setRowColors] = useState<RowColorsConfig>(
    getDefaultRowColors(),
  );

  const fields: ColumnSettingsField[] = useMemo(
    () =>
      FEE_LEDGER_FIELDS.map((field) => ({
        key: field.key,
        title: field.name,
      })),
    [],
  );

  const defaultValue = useMemo(
    () => defaultColumnSettingsValue(fields),
    [fields],
  );

  // 打开弹窗时同步配色草稿；编辑期间外部刷新不重置。
  useEffect(() => {
    if (!open) return;
    if (currentConfig?.rowColors) {
      setRowColors({
        ...getDefaultRowColors(),
        ...currentConfig.rowColors,
      });
    } else {
      setRowColors(getDefaultRowColors());
    }
  }, [open, currentConfig]);

  // 存量配置按当前字段元数据解析：忽略未知 key，未提及列按默认顺序补齐。
  const value = useMemo(() => {
    const base = currentConfig?.columns;
    if (!base || base.order.length === 0) return defaultValue;
    const known = new Set(fields.map((field) => field.key));
    const order = base.order.filter((key) => known.has(key));
    for (const field of fields) {
      if (!order.includes(field.key)) order.push(field.key);
    }
    return {
      order,
      hidden: base.hidden.filter((key) => order.includes(key)),
    };
  }, [currentConfig, defaultValue, fields]);

  const handleSave = (next: ColumnSettingsValue) => {
    const persisted = onSave({ columns: next, rowColors });
    if (persisted === false) {
      // 写入失败：配置已在弹窗草稿中生效，保持打开便于重试或取消。
      message.error('列设置保存到本地浏览器失败，本次设置仅当前页面生效');
      return;
    }
    onClose();
  };

  const advanced = (
    <div style={{ paddingTop: 8 }}>
      <RowColorSettings
        rowColors={rowColors}
        onColorChange={(key, color) =>
          setRowColors((prev) => ({ ...prev, [key]: color }))
        }
        onResetColors={() => {
          setRowColors(getDefaultRowColors());
          message.success('行背景高亮颜色已重置为默认值');
        }}
      />
    </div>
  );

  return (
    <ColumnSettingsModal
      open={open}
      fields={fields}
      value={value}
      defaultValue={defaultValue}
      title="列设置"
      advanced={advanced}
      width={960}
      onSave={handleSave}
      onCancel={onClose}
    />
  );
}
