import { DownOutlined, UpOutlined } from '@ant-design/icons';
import { Button, Checkbox, Input, Modal, Space, Tag, Tooltip } from 'antd';
import React, { useEffect, useState } from 'react';
import {
  defaultFeeColumnPreference,
  effectiveFeeColumnDefs,
  type FeeColumnDef,
  type FeeColumnKey,
  type FeeColumnPreference,
} from './feeColumnPreference';

/** 弹窗内的可编辑草稿：列元数据 + 当前可见性。 */
interface DraftColumn extends FeeColumnDef {
  visible: boolean;
}

interface FeeColumnSettingsModalProps {
  open: boolean;
  /** 是否具备财务费用读取权限（决定账单关联列是否可选）。 */
  financeAvailable: boolean;
  /** 当前已生效的偏好；打开弹窗时作为草稿基线。 */
  value: FeeColumnPreference;
  onCancel: () => void;
  onConfirm: (next: FeeColumnPreference) => void;
}

/** 按偏好草稿构建完整列顺序（含被隐藏列，保持其相对位置便于再次调整）。 */
function buildDraft(
  defs: FeeColumnDef[],
  preference: Pick<FeeColumnPreference, 'order' | 'hidden'>,
): DraftColumn[] {
  const defMap = new Map<FeeColumnKey, FeeColumnDef>(
    defs.map((def) => [def.key, def]),
  );
  const hiddenSet = new Set<string>(preference.hidden);
  const seen = new Set<FeeColumnKey>();
  const draft: DraftColumn[] = [];
  for (const key of preference.order) {
    const def = defMap.get(key);
    if (def && !seen.has(key)) {
      draft.push({
        ...def,
        visible: def.lockVisible || !hiddenSet.has(key),
      });
      seen.add(key);
    }
  }
  for (const def of defs) {
    if (!seen.has(def.key)) {
      draft.push({
        ...def,
        visible:
          def.lockVisible || (def.defaultVisible && !hiddenSet.has(def.key)),
      });
    }
  }
  return draft;
}

/**
 * 订单费用表列设置弹窗：搜索列名、显隐、上移/下移排序、恢复默认。
 * 确定后才提交并持久化；取消不改变表格；必显列（录入必备）不可隐藏。
 */
export default function FeeColumnSettingsModal({
  open,
  financeAvailable,
  value,
  onCancel,
  onConfirm,
}: FeeColumnSettingsModalProps) {
  const [draft, setDraft] = useState<DraftColumn[]>([]);
  const [keyword, setKeyword] = useState('');

  useEffect(() => {
    if (!open) return;
    setDraft(buildDraft(effectiveFeeColumnDefs(financeAvailable), value));
    setKeyword('');
  }, [open, financeAvailable, value]);

  const filteredKeys = new Set(
    draft
      .filter((column) =>
        column.title.toLowerCase().includes(keyword.trim().toLowerCase()),
      )
      .map((column) => column.key),
  );

  const moveColumn = (key: FeeColumnKey, offset: -1 | 1) => {
    setDraft((prev) => {
      const index = prev.findIndex((column) => column.key === key);
      const target = index + offset;
      if (index < 0 || target < 0 || target >= prev.length) return prev;
      const next = [...prev];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  };

  const toggleVisible = (key: FeeColumnKey, visible: boolean) => {
    setDraft((prev) =>
      prev.map((column) =>
        column.key === key && !column.lockVisible
          ? { ...column, visible }
          : column,
      ),
    );
  };

  const resetToDefault = () => {
    setDraft(
      buildDraft(
        effectiveFeeColumnDefs(financeAvailable),
        defaultFeeColumnPreference(financeAvailable),
      ),
    );
  };

  const handleConfirm = () => {
    onConfirm({
      order: draft.map((column) => column.key),
      hidden: draft
        .filter((column) => !column.visible && !column.lockVisible)
        .map((column) => column.key),
    });
  };

  const visibleDraft = draft.filter((column) => filteredKeys.has(column.key));

  return (
    <Modal
      open={open}
      title="列设置"
      width={480}
      onCancel={onCancel}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button onClick={resetToDefault}>恢复默认</Button>
          <Space>
            <Button onClick={onCancel}>取消</Button>
            <Button type="primary" onClick={handleConfirm}>
              确定
            </Button>
          </Space>
        </div>
      }
    >
      <Input
        allowClear
        placeholder="搜索列名"
        value={keyword}
        onChange={(event) => setKeyword(event.target.value)}
        style={{ marginBottom: 12 }}
      />
      <div style={{ maxHeight: 360, overflowY: 'auto' }}>
        {visibleDraft.map((column, index) => (
          <div
            key={column.key}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '6px 4px',
              borderBottom: '1px solid #f0f0f0',
            }}
          >
            <span>
              <Tooltip
                title={column.lockVisible ? '录入必备列，不可隐藏' : undefined}
              >
                <Checkbox
                  checked={column.visible}
                  disabled={column.lockVisible}
                  onChange={(event) =>
                    toggleVisible(column.key, event.target.checked)
                  }
                >
                  {column.title}
                </Checkbox>
              </Tooltip>
              {column.lockVisible && (
                <Tag style={{ marginLeft: 8 }} color="blue">
                  必显
                </Tag>
              )}
            </span>
            <Space size={4}>
              <Button
                type="text"
                size="small"
                aria-label={`上移${column.title}`}
                icon={<UpOutlined />}
                disabled={index === 0}
                onClick={() => moveColumn(column.key, -1)}
              />
              <Button
                type="text"
                size="small"
                aria-label={`下移${column.title}`}
                icon={<DownOutlined />}
                disabled={index === visibleDraft.length - 1}
                onClick={() => moveColumn(column.key, 1)}
              />
            </Space>
          </div>
        ))}
        {visibleDraft.length === 0 && (
          <div
            style={{ padding: '24px 0', textAlign: 'center', color: '#8c8c8c' }}
          >
            未找到匹配的列
          </div>
        )}
      </div>
    </Modal>
  );
}
