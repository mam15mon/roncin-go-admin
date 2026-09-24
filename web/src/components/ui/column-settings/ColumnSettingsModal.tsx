import { DownOutlined, SearchOutlined, UpOutlined } from '@ant-design/icons';
import {
  Button,
  Checkbox,
  Empty,
  Input,
  Modal,
  Radio,
  Space,
  Tabs,
  Tag,
  Tooltip,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import type { ColumnSettingsField, ColumnSettingsValue } from './types';

/** 弹窗内的可编辑草稿：字段元数据 + 当前可见性。 */
interface DraftItem extends ColumnSettingsField {
  visible: boolean;
}

type FieldFilterType = 'all' | 'shown' | 'hidden';

/** 由当前生效配置构建草稿：偏好顺序优先，未提及的新字段按可见兜底。 */
function buildDraft(
  fields: ColumnSettingsField[],
  value: ColumnSettingsValue,
): DraftItem[] {
  const fieldMap = new Map(fields.map((field) => [field.key, field]));
  const hiddenSet = new Set(value.hidden);
  const seen = new Set<string>();
  const draft: DraftItem[] = [];
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

function draftToValue(draft: DraftItem[]): ColumnSettingsValue {
  return {
    order: draft.map((item) => item.key),
    hidden: draft.filter((item) => !item.visible).map((item) => item.key),
  };
}

function zoneOf(fixed: ColumnSettingsField['fixed']): string {
  return fixed ?? 'main';
}

export interface ColumnSettingsModalProps {
  open: boolean;
  /** 全量可配置字段；无权限字段由调用方裁剪，不得传入。 */
  fields: ColumnSettingsField[];
  /** 打开弹窗时的草稿基线（当前生效配置）。 */
  value: ColumnSettingsValue;
  /** 「恢复默认」基准。 */
  defaultValue: ColumnSettingsValue;
  /** 弹窗标题，默认「列设置」。 */
  title?: string;
  /**
   * 增强版高级设置页签内容；提供后弹窗出现「列配置 / 高级设置」页签。
   * 高级内容的草稿状态由调用方自行维护。
   */
  advanced?: React.ReactNode;
  /** 保存提交中；加载期间保存按钮禁用。 */
  saving?: boolean;
  /**
   * 保存成功后由调用方关闭弹窗（open 置 false）；保存失败保持打开即可
   * 保留草稿继续调整或重试。
   */
  onSave: (value: ColumnSettingsValue) => void;
  onCancel: () => void;
  width?: number;
}

/**
 * 全站统一列设置弹窗：搜索列名、显隐勾选、排序（拖拽 + 上下移按钮）、
 * 恢复默认；全部操作只修改草稿，保存后才提交。
 * 搜索中禁用顺序调整并提示清空搜索后排序，避免以筛选后序号操作完整顺序。
 */
export default function ColumnSettingsModal({
  open,
  fields,
  value,
  defaultValue,
  title = '列设置',
  advanced,
  saving,
  onSave,
  onCancel,
  width = 640,
}: ColumnSettingsModalProps) {
  const [draft, setDraft] = useState<DraftItem[]>([]);
  const [keyword, setKeyword] = useState('');
  const [filterType, setFilterType] = useState<FieldFilterType>('all');
  const [draggingKey, setDraggingKey] = useState<string | null>(null);

  // 草稿只在弹窗打开瞬间取基线，编辑期间外部刷新不打断操作。
  useEffect(() => {
    if (!open) return;
    setDraft(buildDraft(fields, value));
    setKeyword('');
    setFilterType('all');
    setDraggingKey(null);
    // 仅在打开瞬间同步草稿基线，fields/value 的引用变化不重置用户编辑。
  }, [open]);

  const trimmedKeyword = keyword.trim().toLowerCase();
  const searching = trimmedKeyword.length > 0;

  const matchesKeyword = (item: DraftItem) =>
    !searching || item.title.toLowerCase().includes(trimmedKeyword);
  const matchesFilter = (item: DraftItem) => {
    if (filterType === 'shown') return item.visible;
    if (filterType === 'hidden') return !item.visible;
    return true;
  };
  const matchesView = (item: DraftItem) =>
    matchesKeyword(item) && matchesFilter(item);

  const visibleCount = draft.filter((item) => item.visible).length;
  const shownCount = draft.filter(matchesView).length;

  // 分区展示：左固定 / 动态列 / 右固定，排序只在同侧区域内进行。
  const zones = useMemo(
    () => [
      { key: 'left', label: '左侧固定列', items: draft.filter((item) => item.fixed === 'left') },
      { key: 'main', label: draft.some((item) => item.fixed) ? '动态列' : '', items: draft.filter((item) => !item.fixed) },
      { key: 'right', label: '右侧固定列', items: draft.filter((item) => item.fixed === 'right') },
    ],
    [draft],
  );

  const moveWithinZone = (key: string, direction: -1 | 1) => {
    if (searching) return;
    setDraft((prev) => {
      const index = prev.findIndex((item) => item.key === key);
      if (index < 0) return prev;
      const zone = zoneOf(prev[index].fixed);
      const canTarget = (item: DraftItem) =>
        zoneOf(item.fixed) === zone && matchesView(item);
      let target = index + direction;
      while (target >= 0 && target < prev.length && !canTarget(prev[target])) {
        target += direction;
      }
      if (target < 0 || target >= prev.length) return prev;
      const next = [...prev];
      const [item] = next.splice(index, 1);
      next.splice(target, 0, item);
      return next;
    });
  };

  const handleDrop = (targetKey: string) => {
    if (searching || !draggingKey || draggingKey === targetKey) return;
    setDraft((prev) => {
      const sourceIndex = prev.findIndex((item) => item.key === draggingKey);
      const targetIndex = prev.findIndex((item) => item.key === targetKey);
      if (sourceIndex < 0 || targetIndex < 0) return prev;
      if (zoneOf(prev[sourceIndex].fixed) !== zoneOf(prev[targetIndex].fixed)) {
        return prev;
      }
      const next = [...prev];
      const [item] = next.splice(sourceIndex, 1);
      next.splice(targetIndex, 0, item);
      return next;
    });
    setDraggingKey(null);
  };

  const toggleVisible = (key: string, visible: boolean) => {
    setDraft((prev) =>
      prev.map((item) =>
        item.key === key && !item.lockVisible ? { ...item, visible } : item,
      ),
    );
  };

  /** 可见列至少保留一列：唯一可见列不可取消勾选。 */
  const canHide = (item: DraftItem) =>
    !item.lockVisible && !(item.visible && visibleCount <= 1);

  const handleResetDraft = () => {
    setDraft(buildDraft(fields, defaultValue));
  };

  const renderRow = (item: DraftItem, zoneItems: DraftItem[]) => {
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

  const listNode = (
    <>
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
          style={{ width: 220 }}
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
      <div style={{ maxHeight: '46vh', overflowY: 'auto', paddingRight: 2 }}>
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
    </>
  );

  return (
    <Modal
      open={open}
      title={
        <span>
          {title}
          <Tag color="blue" style={{ marginLeft: 12 }}>
            已显示 {visibleCount} / {draft.length}
          </Tag>
        </span>
      }
      width={width}
      onCancel={onCancel}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Tooltip title="恢复默认列显隐与顺序，保存后生效">
            <Button onClick={handleResetDraft}>恢复默认</Button>
          </Tooltip>
          <Space>
            <Button onClick={onCancel}>取消</Button>
            <Button
              type="primary"
              loading={saving}
              onClick={() => onSave(draftToValue(draft))}
            >
              保存
            </Button>
          </Space>
        </div>
      }
    >
      {advanced ? (
        <Tabs
          items={[
            { key: 'columns', label: '列配置', children: listNode },
            { key: 'advanced', label: '高级设置', children: advanced },
          ]}
        />
      ) : (
        listNode
      )}
    </Modal>
  );
}
