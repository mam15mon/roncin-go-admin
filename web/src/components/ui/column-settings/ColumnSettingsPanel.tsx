import {
  CheckCircleOutlined,
  HolderOutlined,
  QuestionCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import {
  Button,
  Checkbox,
  Empty,
  Input,
  InputNumber,
  Select,
  Tooltip,
} from 'antd';
import React, { useMemo, useState } from 'react';
import './ColumnSettingsPanel.less';
import {
  deriveColumnFreezeDefaults,
  orderColumnKeys,
  resolveColumnFreeze,
} from './freeze';
import type {
  ColumnFreezeValue,
  ColumnSettingsField,
  ColumnSettingsValue,
} from './types';

type Zone = 'left' | 'none' | 'right';
const ZONES: { key: Zone; title: string }[] = [
  { key: 'left', title: '左侧冻结' },
  { key: 'none', title: '普通列' },
  { key: 'right', title: '右侧冻结' },
];
const FREEZE_OPTIONS = [
  { value: 'left', label: '左冻结' },
  { value: 'none', label: '不冻结' },
  { value: 'right', label: '右冻结' },
];

export interface ColumnSettingsPanelProps {
  fields: ColumnSettingsField[];
  /** 不参与业务排序/显隐的结构列，保留其末尾锚点。 */
  structuralFields?: ColumnSettingsField[];
  value: ColumnSettingsValue;
  onChange: (value: ColumnSettingsValue) => void;
  onReset: () => void;
  onOpenAdvanced?: () => void;
}

/** 全量铺排并即时生效；编号为完整设置顺序，搜索不改变编号。 */
export default function ColumnSettingsPanel({
  fields,
  structuralFields = [],
  value,
  onChange,
  onReset,
  onOpenAdvanced,
}: ColumnSettingsPanelProps) {
  const [keyword, setKeyword] = useState('');
  const [draggingKey, setDraggingKey] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<{
    key?: string;
    zone: Zone;
    after?: boolean;
  } | null>(null);
  const searching = keyword.trim().length > 0;
  const fieldMap = useMemo(
    () => new Map(fields.map((field) => [field.key, field])),
    [fields],
  );
  const knownKeys = [...fields, ...structuralFields].map((field) => field.key);
  const freeze = resolveColumnFreeze(
    value.freeze,
    deriveColumnFreezeDefaults([...fields, ...structuralFields]),
    knownKeys,
  );
  const keys = [
    ...new Set([
      ...value.order.filter((key) => fieldMap.has(key)),
      ...fields.map((field) => field.key),
    ]),
  ];
  const ordered = orderColumnKeys(keys, freeze);
  const hidden = new Set(value.hidden);
  const visibleCount = fields.filter(
    (field) => field.lockVisible || !hidden.has(field.key),
  ).length;
  const allChecked = fields.length > 0 && visibleCount === fields.length;
  const zoneOf = (key: string): Zone =>
    freeze.left.includes(key)
      ? 'left'
      : freeze.right.includes(key)
        ? 'right'
        : 'none';
  const emit = (
    order: string[],
    nextFreeze: ColumnFreezeValue = freeze,
    nextHidden = value.hidden,
  ) =>
    onChange({
      order: orderColumnKeys(order, nextFreeze),
      hidden: nextHidden,
      freeze: nextFreeze,
    });
  const freezeAt = (key: string, zone: Zone): ColumnFreezeValue => ({
    left: [
      ...freeze.left.filter((item) => item !== key),
      ...(zone === 'left' ? [key] : []),
    ],
    right: [
      ...freeze.right.filter((item) => item !== key),
      ...(zone === 'right' ? [key] : []),
    ],
  });
  const clearDrag = () => {
    setDraggingKey(null);
    setDropTarget(null);
  };
  const move = (key: string, position: number) => {
    if (searching) return;
    const target =
      ordered[Math.max(0, Math.min(ordered.length - 1, position - 1))];
    if (!target || target === key) return;
    const next = ordered.filter((item) => item !== key);
    next.splice(Math.max(0, Math.min(next.length, position - 1)), 0, key);
    emit(next, freezeAt(key, zoneOf(target)));
  };
  const drop = (zone: Zone, target?: string, after = false) => {
    if (searching || !draggingKey || target === draggingKey) {
      clearDrag();
      return;
    }
    const next = ordered.filter((key) => key !== draggingKey);
    const index = target
      ? next.indexOf(target) + (after ? 1 : 0)
      : next.reduce(
          (last, key, index) => (zoneOf(key) === zone ? index + 1 : last),
          -1,
        );
    next.splice(index < 0 ? next.length : index, 0, draggingKey);
    emit(next, freezeAt(draggingKey, zone));
    clearDrag();
  };
  const toggleAll = () => {
    const locked = fields.some((field) => field.lockVisible);
    emit(
      ordered,
      freeze,
      allChecked
        ? fields
            .filter(
              (field) =>
                !field.lockVisible && (locked || field.key !== ordered[0]),
            )
            .map((field) => field.key)
        : [],
    );
  };
  const matches = (field: ColumnSettingsField) =>
    field.title.toLowerCase().includes(keyword.trim().toLowerCase());

  return (
    <div className="roncin-column-settings-panel">
      <div className="roncin-column-settings-toolbar">
        <Checkbox
          checked={allChecked}
          indeterminate={visibleCount > 0 && !allChecked}
          disabled={fields.every((field) => field.lockVisible)}
          onChange={toggleAll}
        >
          列展示
          <span className="roncin-column-settings-count">
            {visibleCount} / {fields.length}
          </span>
        </Checkbox>
        <Input
          allowClear
          size="small"
          prefix={<SearchOutlined />}
          placeholder="搜索列名"
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
        />
        <Button type="text" size="small" onClick={onReset}>
          恢复默认
        </Button>
      </div>
      <div className="roncin-column-settings-help">
        <span>勾选显示字段，拖动调整顺序</span>
        <Tooltip title="字段按从左到右、从上到下排列。拖到左右冻结区可固定字段；修改序号后按 Enter 或移开焦点可精确移位。">
          <Button type="text" size="small" icon={<QuestionCircleOutlined />}>
            排序说明
          </Button>
        </Tooltip>
      </div>
      {searching && (
        <div className="roncin-column-settings-search-tip">
          搜索中无法调整顺序，清空搜索后可拖拽或修改序号排序
        </div>
      )}
      <div className="roncin-column-settings-zones">
        {ZONES.map((zone) => {
          const items = ordered
            .filter((key) => zoneOf(key) === zone.key)
            .map((key) => fieldMap.get(key) as ColumnSettingsField);
          const matched = items.filter(matches);
          if (searching && !matched.length) return null;
          return (
            <section
              key={zone.key}
              className={`roncin-column-settings-zone${!items.length ? ' roncin-column-settings-zone-empty' : ''}${dropTarget?.zone === zone.key && !dropTarget.key ? ' roncin-column-settings-zone-target' : ''}`}
              data-freeze-zone={zone.key}
              aria-label={zone.title}
              onDragOver={(event) => {
                if (!draggingKey || searching) return;
                event.preventDefault();
                if (event.target === event.currentTarget)
                  setDropTarget({ zone: zone.key });
              }}
              onDrop={(event) => {
                event.preventDefault();
                drop(zone.key);
              }}
            >
              <div className="roncin-column-settings-zone-heading">
                {zone.title}
                <span>
                  {searching
                    ? `${matched.length} / ${items.length}`
                    : items.length}{' '}
                  列
                </span>
              </div>
              <div className="roncin-column-settings-grid">
                {matched.map((item) => {
                  const index = ordered.indexOf(item.key);
                  const visible = item.lockVisible || !hidden.has(item.key);
                  const canHide =
                    !item.lockVisible && !(visible && visibleCount <= 1);
                  return (
                    <div
                      key={item.key}
                      data-column-key={item.key}
                      data-column-position={index + 1}
                      className={`roncin-column-settings-row${!visible ? ' roncin-column-settings-row-hidden' : ''}${draggingKey === item.key ? ' roncin-column-settings-dragging' : ''}${dropTarget?.key === item.key ? (dropTarget.after ? ' roncin-column-settings-drop-after' : ' roncin-column-settings-drop-before') : ''}`}
                      onDragOver={(event) => {
                        if (!draggingKey || searching) return;
                        event.preventDefault();
                        event.stopPropagation();
                        const rect =
                          event.currentTarget.getBoundingClientRect();
                        setDropTarget({
                          key: item.key,
                          zone: zone.key,
                          after: event.clientX > rect.left + rect.width / 2,
                        });
                      }}
                      onDrop={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        const rect =
                          event.currentTarget.getBoundingClientRect();
                        drop(
                          zone.key,
                          item.key,
                          event.clientX > rect.left + rect.width / 2,
                        );
                      }}
                    >
                      <span
                        className="roncin-column-settings-handle"
                        draggable={!searching}
                        aria-label={`拖动${item.title}`}
                        role="img"
                        onDragStart={(event) => {
                          event.dataTransfer?.setData('text/plain', item.key);
                          if (event.dataTransfer)
                            event.dataTransfer.effectAllowed = 'move';
                          setDraggingKey(item.key);
                        }}
                        onDragEnd={clearDrag}
                      >
                        <HolderOutlined />
                      </span>
                      <InputNumber
                        key={`${item.key}:${index}`}
                        size="small"
                        variant="borderless"
                        className="roncin-column-settings-position"
                        aria-label={`${item.title}目标序号`}
                        min={1}
                        max={ordered.length}
                        controls={false}
                        defaultValue={index + 1}
                        disabled={searching}
                        onPressEnter={(event) => {
                          const next = Number(
                            (event.target as HTMLInputElement).value,
                          );
                          if (Number.isInteger(next)) move(item.key, next);
                        }}
                        onBlur={(event) => {
                          const next = Number(
                            (event.target as HTMLInputElement).value,
                          );
                          if (Number.isInteger(next) && next !== index + 1)
                            move(item.key, next);
                        }}
                      />
                      <Tooltip
                        title={
                          item.lockVisible
                            ? `${item.title}（必显）`
                            : item.title
                        }
                      >
                        <Checkbox
                          checked={visible}
                          disabled={!canHide}
                          onChange={(event) =>
                            emit(
                              ordered,
                              freeze,
                              event.target.checked
                                ? value.hidden.filter((key) => key !== item.key)
                                : [...value.hidden, item.key],
                            )
                          }
                        >
                          {item.title}
                        </Checkbox>
                      </Tooltip>
                      <Select
                        size="small"
                        variant="borderless"
                        className="roncin-column-settings-freeze"
                        aria-label={`${item.title}冻结位置`}
                        value={zone.key}
                        options={FREEZE_OPTIONS}
                        onChange={(next: Zone) =>
                          emit(ordered, freezeAt(item.key, next))
                        }
                      />
                    </div>
                  );
                })}
                {!matched.length && !searching && (
                  <div
                    className="roncin-column-settings-empty"
                    onDragOver={(event) => {
                      if (!draggingKey) return;
                      event.preventDefault();
                      setDropTarget({ zone: zone.key });
                    }}
                  >
                    {zone.key === 'none'
                      ? '拖入字段，取消冻结'
                      : `拖入字段，固定在表格${zone.key === 'left' ? '左侧' : '右侧'}`}
                  </div>
                )}
              </div>
            </section>
          );
        })}
      </div>
      {searching && !fields.some(matches) && (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="未找到匹配的列"
        />
      )}
      {structuralFields.length > 0 && (
        <div className="roncin-column-settings-structural">
          <span>辅助列 · 顺序固定</span>
          {structuralFields.map((field) => (
            <div
              key={field.key}
              className="roncin-column-settings-structural-field"
            >
              {field.title}
              <Select
                size="small"
                variant="borderless"
                aria-label={`${field.title}冻结位置`}
                value={zoneOf(field.key)}
                options={FREEZE_OPTIONS.filter(
                  (option) => option.value !== 'left',
                )}
                onChange={(next: Zone) =>
                  emit(ordered, freezeAt(field.key, next))
                }
              />
            </div>
          ))}
        </div>
      )}
      <div className="roncin-column-settings-footer">
        <span className="roncin-column-settings-status">
          <CheckCircleOutlined />
          修改即时生效，无需保存
        </span>
        {onOpenAdvanced && (
          <Button type="link" size="small" onClick={onOpenAdvanced}>
            更多设置
          </Button>
        )}
      </div>
    </div>
  );
}
