import { DownOutlined, SearchOutlined, UpOutlined } from '@ant-design/icons';
import {
  Button,
  Checkbox,
  Empty,
  Input,
  Radio,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import React, { useMemo, useState } from 'react';
import type { ColumnSettingsField, ColumnSettingsValue } from './types';

/** 受控渲染条目：字段元数据 + 当前可见性。 */
interface PanelItem extends ColumnSettingsField {
  visible: boolean;
}

type FieldFilterType = 'all' | 'shown' | 'hidden';

/** 由当前生效配置构建展示条目：偏好顺序优先，未提及的新字段按可见兜底。 */
function buildDraft(
  fields: ColumnSettingsField[],
  value: ColumnSettingsValue,
): PanelItem[] {
  const fieldMap = new Map(fields.map((field) => [field.key, field]));
  const hiddenSet = new Set(value.hidden);
  const seen = new Set<string>();
  const draft: PanelItem[] = [];
  for (const key of value.order) {
    const field = fieldMap.get(key);
    if (field && !seen.has(key)) {
      draft.push({
        ...field,
        visible: field.lockVisible || !hiddenSet.has(key),
      });
      seen.add(key);
    }
  }
  for (const field of fields) {
    if (!seen.has(field.key)) {
      draft.push({ ...field, visible: true });
    }
  }
  return draft;
}

function draftToValue(draft: PanelItem[]): ColumnSettingsValue {
  return {
    order: draft.map((item) => item.key),
    hidden: draft.filter((item) => !item.visible).map((item) => item.key),
  };
}

function zoneOf(fixed: ColumnSettingsField['fixed']): string {
  return fixed ?? 'main';
}

export interface ColumnSettingsPanelProps {
  /** 全量可配置字段；无权限字段由调用方裁剪，不得传入。 */
  fields: ColumnSettingsField[];
  /** 当前生效配置（即时真相源），变更经 onChange 回流。 */
  value: ColumnSettingsValue;
  /** 显隐或排序变更回调：每次交互立即触发，无草稿与保存步骤。 */
  onChange: (value: ColumnSettingsValue) => void;
  /** 恢复默认回调：由调用方提交默认配置并持久化。 */
  onReset: () => void;
  /** 提供时底部出现「更多设置」按钮（行配色等高级内容入口）。 */
  onOpenAdvanced?: () => void;
}

/**
 * 统一列设置浮层内容：搜索列名、显隐勾选、排序（拖拽 + 上下移按钮）与
 * 恢复默认；全部操作即时回调生效。搜索中禁用顺序调整并提示清空搜索后
 * 排序，避免以筛选后序号操作完整顺序。
 */
export default function ColumnSettingsPanel({
  fields,
  value,
  onChange,
  onReset,
  onOpenAdvanced,
}: ColumnSettingsPanelProps) {
  const [keyword, setKeyword] = useState('');
  const [filterType, setFilterType] = useState<FieldFilterType>('all');
  const [draggingKey, setDraggingKey] = useState<string | null>(null);

  const draft = useMemo(() => buildDraft(fields, value), [fields, value]);

  const trimmedKeyword = keyword.trim().toLowerCase();
  const searching = trimmedKeyword.length > 0;

  const matchesKeyword = (item: PanelItem) =>
    !searching || item.title.toLowerCase().includes(trimmedKeyword);
  const matchesFilter = (item: PanelItem) => {
    if (filterType === 'shown') return item.visible;
    if (filterType === 'hidden') return !item.visible;
    return true;
  };
  const matchesView = (item: PanelItem) =>
    matchesKeyword(item) && matchesFilter(item);

  const visibleCount = draft.filter((item) => item.visible).length;
  const shownCount = draft.filter(matchesView).length;

  // 分区展示：左固定 / 动态列 / 右固定，排序只在同侧区域内进行。
  const zones = useMemo(
    () => [
      {
        key: 'left',
        label: '左侧固定列',
        items: draft.filter((item) => item.fixed === 'left'),
      },
      {
        key: 'main',
        label: draft.some((item) => item.fixed) ? '动态列' : '',
        items: draft.filter((item) => !item.fixed),
      },
      {
        key: 'right',
        label: '右侧固定列',
        items: draft.filter((item) => item.fixed === 'right'),
      },
    ],
    [draft],
  );

  const emit = (next: PanelItem[]) => onChange(draftToValue(next));

  const moveWithinZone = (key: string, direction: -1 | 1) => {
    if (searching) return;
    const index = draft.findIndex((item) => item.key === key);
    if (index < 0) return;
    const zone = zoneOf(draft[index].fixed);
    const canTarget = (item: PanelItem) =>
      zoneOf(item.fixed) === zone && matchesView(item);
    let target = index + direction;
    while (target >= 0 && target < draft.length && !canTarget(draft[target])) {
      target += direction;
    }
    if (target < 0 || target >= draft.length) return;
    const next = [...draft];
    const [item] = next.splice(index, 1);
    next.splice(target, 0, item);
    emit(next);
  };

  const handleDrop = (targetKey: string) => {
    if (searching || !draggingKey || draggingKey === targetKey) return;
    const sourceIndex = draft.findIndex((item) => item.key === draggingKey);
    const targetIndex = draft.findIndex((item) => item.key === targetKey);
    if (sourceIndex < 0 || targetIndex < 0) return;
    if (zoneOf(draft[sourceIndex].fixed) !== zoneOf(draft[targetIndex].fixed)) {
      return;
    }
    const next = [...draft];
    const [item] = next.splice(sourceIndex, 1);
    next.splice(targetIndex, 0, item);
    emit(next);
    setDraggingKey(null);
  };

  const toggleVisible = (key: string, visible: boolean) => {
    emit(
      draft.map((item) =>
        item.key === key && !item.lockVisible ? { ...item, visible } : item,
      ),
    );
  };

  /** 可见列至少保留一列：唯一可见列不可取消勾选。 */
  const canHide = (item: PanelItem) =>
    !item.lockVisible && !(item.visible && visibleCount <= 1);

  const renderRow = (item: PanelItem, zoneItems: PanelItem[]) => {
    const zoneIndex = zoneItems.findIndex((row) => row.key === item.key);
    const checkboxDisabled = !canHide(item);
    const checkboxTooltip = item.lockVisible
      ? '必显列，不可隐藏'
      : checkboxDisabled
        ? '至少保留一列显示'
        : undefined;
    const moveDisabled = searching;
    return (
      <div
        key={item.key}
        draggable={!searching}
        onDragStart={() => setDraggingKey(item.key)}
        onDragEnd={() => setDraggingKey(null)}
        onDragOver={(event) => {
          if (!searching) event.preventDefault();
        }}
        onDrop={() => handleDrop(item.key)}
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '5px 8px',
          borderBottom: '1px solid #f0f0f0',
          opacity: draggingKey === item.key ? 0.5 : 1,
          cursor: searching ? 'default' : 'grab',
        }}
      >
        <Tooltip title={checkboxTooltip}>
          <Checkbox
            checked={item.visible}
            disabled={checkboxDisabled}
            onChange={(event) => toggleVisible(item.key, event.target.checked)}
          >
            {item.title}
          </Checkbox>
        </Tooltip>
        {item.lockVisible && (
          <Tag color="blue" style={{ marginRight: 8 }}>
            必显
          </Tag>
        )}
        <Space size={2}>
          <Button
            type="text"
            size="small"
            aria-label={`上移${item.title}`}
            icon={<UpOutlined />}
            disabled={moveDisabled || zoneIndex <= 0}
            onClick={() => moveWithinZone(item.key, -1)}
          />
          <Button
            type="text"
            size="small"
            aria-label={`下移${item.title}`}
            icon={<DownOutlined />}
            disabled={moveDisabled || zoneIndex >= zoneItems.length - 1}
            onClick={() => moveWithinZone(item.key, 1)}
          />
        </Space>
      </div>
    );
  };

  return (
    <div style={{ width: 320 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 12,
          flexWrap: 'wrap',
          marginBottom: 12,
        }}
      >
        <Input
          allowClear
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          placeholder="搜索列名"
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          style={{ width: 140 }}
        />
        <Radio.Group
          size="small"
          value={filterType}
          optionType="button"
          buttonStyle="solid"
          onChange={(event) =>
            setFilterType(event.target.value as FieldFilterType)
          }
          options={[
            { label: `全部 (${draft.length})`, value: 'all' },
            { label: `已显示 (${visibleCount})`, value: 'shown' },
            {
              label: `已隐藏 (${draft.length - visibleCount})`,
              value: 'hidden',
            },
          ]}
        />
      </div>
      {searching && (
        <div style={{ marginBottom: 8, color: '#8c8c8c', fontSize: 12 }}>
          搜索中无法调整顺序，清空搜索后可拖拽或使用箭头排序
        </div>
      )}
      <div style={{ maxHeight: 420, overflowY: 'auto', paddingRight: 2 }}>
        {zones.map((zone) => {
          const zoneItems = zone.items.filter(matchesView);
          if (zoneItems.length === 0) return null;
          return (
            <div key={zone.key}>
              {zone.label && (
                <div
                  style={{
                    padding: '6px 8px 4px',
                    color: '#8c8c8c',
                    fontSize: 12,
                  }}
                >
                  {zone.label}
                </div>
              )}
              {zoneItems.map((item) => renderRow(item, zone.items))}
            </div>
          );
        })}
        {shownCount === 0 && (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="未找到匹配的列"
            style={{ padding: '24px 0' }}
          />
        )}
      </div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginTop: 8,
          paddingTop: 8,
          borderTop: '1px solid #f0f0f0',
        }}
      >
        <Tag style={{ marginInlineEnd: 0 }}>
          已显示 {visibleCount} / {draft.length}
        </Tag>
        <Space size={4}>
          {onOpenAdvanced && (
            <Button
              type="link"
              size="small"
              style={{ paddingInline: 4 }}
              onClick={onOpenAdvanced}
            >
              更多设置
            </Button>
          )}
          <Tooltip title="恢复默认列显隐与顺序，立即生效">
            <Button size="small" onClick={onReset}>
              恢复默认
            </Button>
          </Tooltip>
        </Space>
      </div>
    </div>
  );
}
