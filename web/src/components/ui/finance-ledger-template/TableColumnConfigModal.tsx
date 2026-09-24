import { App, Card, Col, Radio, Row, Select } from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import {
  ColumnSettingsModal,
  defaultColumnSettingsValue,
} from '@/components/ui/column-settings';
import type {
  ColumnSettingsField,
  ColumnSettingsValue,
} from '@/components/ui/column-settings';
import {
  settlementServiceUpdateFeeLedgerPreference,
} from '@/services/roncin/settlementService';
import {
  ALL_153_FINANCE_FIELDS,
  getDefaultRowColors,
} from './fields-meta';
import RowColorSettings, { type RowColorsConfig } from './RowColorSettings';

export interface TableColumnConfigModalProps {
  open: boolean;
  onClose: () => void;
  currentPreference?: API.FeeLedgerPreference;
  onSaved: (preference: API.FeeLedgerPreference) => void;
}

/** 历史偏好中的字段 key 规范化：与表格列应用逻辑保持同一映射。 */
function normalizeFieldKey(key: string): string {
  if (key === 'financial_progress') return 'financialProgress';
  if (key === 'customerName') return 'customerId';
  return key;
}

/** 由服务端偏好构建统一列配置：忽略未知 key，未提及列按默认显隐兜底。 */
function preferenceToColumnValue(
  preference?: API.FeeLedgerPreference,
): ColumnSettingsValue | null {
  const columns = preference?.columns ?? [];
  if (columns.length === 0) return null;
  const known = new Set(ALL_153_FINANCE_FIELDS.map((field) => field.key));
  const order: string[] = [];
  const hidden: string[] = [];
  const seen = new Set<string>();
  for (const item of columns) {
    if (!item.fieldKey) continue;
    const key = normalizeFieldKey(item.fieldKey);
    if (!known.has(key) || seen.has(key)) continue;
    order.push(key);
    seen.add(key);
    if (!item.visible) hidden.push(key);
  }
  for (const field of ALL_153_FINANCE_FIELDS) {
    if (!seen.has(field.key)) {
      order.push(field.key);
      if (!field.defaultVisible) hidden.push(field.key);
    }
  }
  return { order, hidden };
}

/**
 * 财务费用台账列设置（增强版）：统一弹窗 + 「高级设置」页签。
 * 高级页承载分页、默认排序与 7 类行配色；列配置沿用统一交互，
 * 全部草稿式编辑，保存时一并提交服务端，失败保留弹窗草稿。
 */
export function TableColumnConfigModal({
  open,
  onClose,
  currentPreference,
  onSaved,
}: TableColumnConfigModalProps) {
  const { message } = App.useApp();

  const [saving, setSaving] = useState(false);
  const [pageSize, setPageSize] = useState<number>(40);
  const [sortField, setSortField] = useState<string>('');
  const [sortDirection, setSortDirection] = useState<'ASC' | 'DESC'>('DESC');
  const [rowColors, setRowColors] = useState<RowColorsConfig>(
    getDefaultRowColors(),
  );

  const fields: ColumnSettingsField[] = useMemo(
    () =>
      ALL_153_FINANCE_FIELDS.map((field) => ({
        key: field.key,
        title: field.name,
      })),
    [],
  );

  const defaultValue = useMemo(
    () =>
      defaultColumnSettingsValue(
        fields,
        ALL_153_FINANCE_FIELDS.filter((field) => !field.defaultVisible).map(
          (field) => field.key,
        ),
      ),
    [fields],
  );

  const value = useMemo(() => {
    const resolved = preferenceToColumnValue(currentPreference);
    if (!resolved) return defaultValue;
    // 存量偏好按当前字段元数据解析，未知 key 与必显约束在此收敛。
    const order = resolved.order.filter((key) =>
      fields.some((field) => field.key === key),
    );
    for (const field of fields) {
      if (!order.includes(field.key)) order.push(field.key);
    }
    return {
      order,
      hidden: resolved.hidden.filter((key) => order.includes(key)),
    };
  }, [currentPreference, defaultValue, fields]);

  // 打开弹窗时同步高级设置草稿；编辑期间外部刷新不重置。
  useEffect(() => {
    if (!open) return;
    if (currentPreference?.rowColors) {
      setRowColors({
        ...getDefaultRowColors(),
        ...(currentPreference.rowColors as RowColorsConfig),
      });
    } else {
      setRowColors(getDefaultRowColors());
    }
    setPageSize(currentPreference?.pageSize ?? 40);
    setSortField(currentPreference?.sortField ?? '');
    setSortDirection(
      (currentPreference?.sortDirection as 'ASC' | 'DESC') ?? 'DESC',
    );
  }, [open, currentPreference]);

  const visibleKeys = useMemo(
    () => value.order.filter((key) => !value.hidden.includes(key)),
    [value],
  );

  const handleSave = async (next: ColumnSettingsValue) => {
    setSaving(true);
    try {
      const response = await settlementServiceUpdateFeeLedgerPreference({
        columns: next.order.map((key) => ({
          fieldKey: key,
          visible: !next.hidden.includes(key),
        })),
        rowColors,
        pageSize,
        sortField: sortField || undefined,
        sortDirection,
      });
      if (!response.data) {
        message.error('保存偏好配置失败');
        return;
      }
      message.success('表格字段偏好与视图配置已保存');
      onSaved(response.data);
      onClose();
    } catch {
      // 保存失败保持弹窗打开，草稿保留供修正后重试。
      message.error('保存偏好配置失败');
    } finally {
      setSaving(false);
    }
  };

  const advanced = (
    <div style={{ paddingTop: 8 }}>
      <Card
        size="small"
        title="基础分页与排序设置"
        style={{ marginBottom: 16, background: '#fafafa' }}
      >
        <Row gutter={24} align="middle">
          <Col span={8}>
            <div style={{ marginBottom: 6, fontWeight: 500, fontSize: 13 }}>
              每页展示行数：
            </div>
            <Select
              value={pageSize}
              onChange={setPageSize}
              style={{ width: '100%' }}
              options={[
                { label: '40 行 / 页 (默认推荐)', value: 40 },
                { label: '60 行 / 页 (高密度)', value: 60 },
                { label: '100 行 / 页 (大屏宽表)', value: 100 },
                { label: '200 行 / 页 (全量极限)', value: 200 },
              ]}
            />
          </Col>
          <Col span={10}>
            <div style={{ marginBottom: 6, fontWeight: 500, fontSize: 13 }}>
              默认排序字段：
            </div>
            <Select
              value={sortField}
              onChange={setSortField}
              style={{ width: '100%' }}
              allowClear
              placeholder="请选择默认排序字段（默认费用时间）"
              options={[
                { label: '无特定排序（按录入与费用时间）', value: '' },
                ...ALL_153_FINANCE_FIELDS.filter((field) =>
                  visibleKeys.includes(field.key),
                ).map((field) => ({
                  label: `${field.name} (${field.key})`,
                  value: field.key,
                })),
              ]}
            />
          </Col>
          <Col span={6}>
            <div style={{ marginBottom: 6, fontWeight: 500, fontSize: 13 }}>
              排序方式：
            </div>
            <Radio.Group
              value={sortDirection}
              onChange={(event) => setSortDirection(event.target.value)}
              optionType="button"
              buttonStyle="solid"
              options={[
                { label: '降序 DESC', value: 'DESC' },
                { label: '升序 ASC', value: 'ASC' },
              ]}
            />
          </Col>
        </Row>
      </Card>
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
      saving={saving}
      width={960}
      onSave={handleSave}
      onCancel={onClose}
    />
  );
}
