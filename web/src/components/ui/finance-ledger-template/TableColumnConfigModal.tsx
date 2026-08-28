import {
  ReloadOutlined,
  SearchOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import {
  App,
  Button,
  Card,
  Col,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Row,
  Select,
  Space,
  Tag,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import { FieldConfigCard } from './FieldConfigCard';
import {
  ALL_153_FINANCE_FIELDS,
  getDefaultColumnPreferences,
  getDefaultRowColors,
  type FinanceFieldMeta,
} from './fields-meta';
import RowColorSettings, {
  type RowColorsConfig,
} from './RowColorSettings';
import {
  settlementServiceResetFeeLedgerPreference,
  settlementServiceUpdateFeeLedgerPreference,
} from '@/services/roncin/settlementService';

export interface TableColumnConfigModalProps {
  open: boolean;
  onClose: () => void;
  currentPreference?: API.FeeLedgerPreference;
  onSaved: (preference: API.FeeLedgerPreference) => void;
}

export function TableColumnConfigModal({
  open,
  onClose,
  currentPreference,
  onSaved,
}: TableColumnConfigModalProps) {
  const { message } = App.useApp();

  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'checked' | 'unchecked'>(
    'all',
  );

  // 153 个字段的排序列（用户偏好的自定义字段顺序）
  const [fieldList, setFieldList] = useState<FinanceFieldMeta[]>(
    ALL_153_FINANCE_FIELDS,
  );

  // 字段勾选状态 Map (key -> visible boolean)
  const [columnMap, setColumnMap] = useState<Map<string, boolean>>(() => {
    const map = new Map<string, boolean>();
    const defaultCols = getDefaultColumnPreferences();
    for (const c of defaultCols) {
      map.set(c.fieldKey, c.visible);
    }
    return map;
  });

  // 7 类进度高亮配色
  const [rowColors, setRowColors] = useState<RowColorsConfig>(
    getDefaultRowColors(),
  );

  // 基础分页与排序
  const [pageSize, setPageSize] = useState<number>(40);
  const [sortField, setSortField] = useState<string>('');
  const [sortDirection, setSortDirection] = useState<'ASC' | 'DESC'>('DESC');

  // 保存与重置 loading
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);

  // 拖拽源 key
  const [draggingKey, setDraggingKey] = useState<string | null>(null);

  // 同步外部传入的偏好
  useEffect(() => {
    if (open) {
      if (currentPreference?.columns && currentPreference.columns.length > 0) {
        const keyMap = new Map(ALL_153_FINANCE_FIELDS.map((f) => [f.key, f]));
        const ordered: FinanceFieldMeta[] = [];
        for (const item of currentPreference.columns) {
          if (item.fieldKey) {
            const f = keyMap.get(item.fieldKey);
            if (f) {
              ordered.push(f);
              keyMap.delete(item.fieldKey);
            }
          }
        }
        for (const f of keyMap.values()) {
          ordered.push(f);
        }
        setFieldList(ordered);

        const map = new Map<string, boolean>();
        for (const c of currentPreference.columns) {
          if (c.fieldKey) {
            map.set(c.fieldKey, Boolean(c.visible));
          }
        }
        setColumnMap(map);
      } else {
        setFieldList(ALL_153_FINANCE_FIELDS);
        const map = new Map<string, boolean>();
        for (const c of getDefaultColumnPreferences()) {
          map.set(c.fieldKey, c.visible);
        }
        setColumnMap(map);
      }

      if (currentPreference?.rowColors) {
        setRowColors({
          ...getDefaultRowColors(),
          ...(currentPreference.rowColors as RowColorsConfig),
        });
      } else {
        setRowColors(getDefaultRowColors());
      }

      if (currentPreference?.pageSize) {
        setPageSize(currentPreference.pageSize);
      }
      if (currentPreference?.sortField !== undefined) {
        setSortField(currentPreference.sortField);
      }
      if (currentPreference?.sortDirection) {
        setSortDirection(currentPreference.sortDirection as 'ASC' | 'DESC');
      }
    }
  }, [open, currentPreference]);

  // 已选字段数量
  const selectedCount = useMemo(() => {
    let count = 0;
    for (const val of columnMap.values()) {
      if (val) count++;
    }
    return count;
  }, [columnMap]);

  // 过滤后的字段
  const filteredFields = useMemo(() => {
    const kw = searchKeyword.trim().toLowerCase();
    return fieldList.filter((f) => {
      const isChecked = Boolean(columnMap.get(f.key));
      if (filterType === 'checked' && !isChecked) return false;
      if (filterType === 'unchecked' && isChecked) return false;
      if (kw) {
        return (
          f.name.toLowerCase().includes(kw) || f.key.toLowerCase().includes(kw)
        );
      }
      return true;
    });
  }, [fieldList, columnMap, filterType, searchKeyword]);

  // 字段序号 Map (基于当前整个 153 排序列)
  const fieldOrderMap = useMemo(() => {
    const map = new Map<string, number>();
    fieldList.forEach((f, idx) => {
      map.set(f.key, idx + 1);
    });
    return map;
  }, [fieldList]);

  // 切换单个字段勾选
  const handleToggleColumn = (key: string) => {
    setColumnMap((prev) => {
      const next = new Map(prev);
      next.set(key, !next.get(key));
      return next;
    });
  };

  // 全选
  const handleSelectAll = () => {
    setColumnMap((prev) => {
      const next = new Map(prev);
      for (const f of fieldList) {
        next.set(f.key, true);
      }
      return next;
    });
  };

  // 反选
  const handleInvertSelect = () => {
    setColumnMap((prev) => {
      const next = new Map(prev);
      for (const f of fieldList) {
        next.set(f.key, !next.get(f.key));
      }
      return next;
    });
  };

  // 重置为推荐字段
  const handleResetDefaultFields = () => {
    const map = new Map<string, boolean>();
    for (const c of getDefaultColumnPreferences()) {
      map.set(c.fieldKey, c.visible);
    }
    setColumnMap(map);
    message.success('已恢复为系统推荐显示的常用字段');
  };

  // 将已选字段一键排到最前
  const handleSortCheckedFirst = () => {
    const checkedList: FinanceFieldMeta[] = [];
    const uncheckedList: FinanceFieldMeta[] = [];
    for (const f of fieldList) {
      if (columnMap.get(f.key)) {
        checkedList.push(f);
      } else {
        uncheckedList.push(f);
      }
    }
    setFieldList([...checkedList, ...uncheckedList]);
    message.success('已将勾选字段全部排至最前');
  };

  // 置顶
  const handleMoveToTop = (key: string) => {
    setFieldList((prev) => {
      const idx = prev.findIndex((f) => f.key === key);
      if (idx <= 0) return prev;
      const target = prev[idx];
      const next = [...prev];
      next.splice(idx, 1);
      next.unshift(target);
      return next;
    });
  };

  // 上移一位
  const handleMoveUp = (key: string) => {
    setFieldList((prev) => {
      const idx = prev.findIndex((f) => f.key === key);
      if (idx <= 0) return prev;
      const next = [...prev];
      const temp = next[idx - 1];
      next[idx - 1] = next[idx];
      next[idx] = temp;
      return next;
    });
  };

  // 下移一位
  const handleMoveDown = (key: string) => {
    setFieldList((prev) => {
      const idx = prev.findIndex((f) => f.key === key);
      if (idx < 0 || idx >= prev.length - 1) return prev;
      const next = [...prev];
      const temp = next[idx + 1];
      next[idx + 1] = next[idx];
      next[idx] = temp;
      return next;
    });
  };

  // 拖拽开始
  const handleDragStart = (key: string) => {
    setDraggingKey(key);
  };

  // 拖拽放置
  const handleDrop = (targetKey: string) => {
    if (!draggingKey || draggingKey === targetKey) return;
    setFieldList((prev) => {
      const sourceIdx = prev.findIndex((f) => f.key === draggingKey);
      const targetIdx = prev.findIndex((f) => f.key === targetKey);
      if (sourceIdx < 0 || targetIdx < 0) return prev;
      const next = [...prev];
      const [item] = next.splice(sourceIdx, 1);
      next.splice(targetIdx, 0, item);
      return next;
    });
    setDraggingKey(null);
  };

  // 重置颜色
  const handleResetColors = () => {
    setRowColors(getDefaultRowColors());
    message.success('行背景高亮颜色已重置为默认值');
  };

  // 彻底重置为系统默认配置
  const handleResetToSystemDefault = async () => {
    setResetting(true);
    try {
      const res = await settlementServiceResetFeeLedgerPreference({});
      if (res.data) {
        onSaved(res.data);
        message.success('已恢复系统初始偏好设置');
        onClose();
      }
    } catch {
      message.error('恢复默认设置失败');
    } finally {
      setResetting(false);
    }
  };

  // 保存配置
  const handleSave = async () => {
    setSaving(true);
    try {
      const columnsPayload: API.FeeLedgerColumnPreference[] = fieldList.map((f) => ({
        fieldKey: f.key,
        visible: Boolean(columnMap.get(f.key)),
      }));

      const body: API.UpdateFeeLedgerPreferenceRequest = {
        columns: columnsPayload,
        rowColors,
        pageSize,
        sortField: sortField || undefined,
        sortDirection,
      };

      const res = await settlementServiceUpdateFeeLedgerPreference(body);
      if (res.data) {
        message.success('表格字段偏好与视图配置已保存');
        onSaved(res.data);
        onClose();
      }
    } catch {
      message.error('保存偏好配置失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      width={1120}
      title={
        <Space size={8}>
          <SettingOutlined style={{ color: '#1677ff' }} />
          <span>全量业务字段配置与表头偏好</span>
          <Tag color="blue" style={{ marginLeft: 8 }}>
            已启用 {selectedCount} / 153 项
          </Tag>
        </Space>
      }
      footer={[
        <Popconfirm
          key="reset-default"
          title="恢复系统初始配置？"
          description="将清空当前用户的个性化表头、排序及配色方案，彻底恢复为系统默认设置。"
          onConfirm={handleResetToSystemDefault}
        >
          <Button danger loading={resetting} style={{ float: 'left' }}>
            <ReloadOutlined /> 恢复系统默认
          </Button>
        </Popconfirm>,
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Button
          key="save"
          type="primary"
          loading={saving}
          onClick={handleSave}
        >
          确定并保存配置
        </Button>,
      ]}
    >
      <div style={{ maxHeight: '72vh', overflowY: 'auto', paddingRight: 4 }}>
        {/* 1. 基础分页与排序设置 */}
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
                  ...fieldList
                    .filter((f) => columnMap.get(f.key))
                    .map((f) => ({
                      label: `${f.name} (${f.key})`,
                      value: f.key,
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
                onChange={(e) => setSortDirection(e.target.value)}
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

        {/* 2. 字段选择与 153 项拖拽网格 */}
        <div style={{ marginBottom: 12 }}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 10,
              flexWrap: 'wrap',
              gap: 8,
            }}
          >
            <Space size={8} wrap>
              <span style={{ fontWeight: 600, fontSize: 14 }}>
                全量业务字段配置
              </span>
              <Radio.Group
                size="small"
                value={filterType}
                onChange={(e) => setFilterType(e.target.value)}
                optionType="button"
                buttonStyle="solid"
                options={[
                  { label: `全部 (${fieldList.length})`, value: 'all' },
                  { label: `已启用 (${selectedCount})`, value: 'checked' },
                  {
                    label: `未启用 (${fieldList.length - selectedCount})`,
                    value: 'unchecked',
                  },
                ]}
              />
              <Button
                size="small"
                type="dashed"
                onClick={handleSortCheckedFirst}
              >
                已启用一键置顶
              </Button>
              <Button size="small" onClick={handleSelectAll}>
                全选
              </Button>
              <Button size="small" onClick={handleInvertSelect}>
                反选
              </Button>
              <Button size="small" onClick={handleResetDefaultFields}>
                重置默认推荐
              </Button>
            </Space>
            <Input
              allowClear
              prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              placeholder="搜索字段名或标识..."
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              style={{ width: 240 }}
            />
          </div>

          <div
            style={{
              border: '1px solid #f0f0f0',
              borderRadius: 6,
              padding: 12,
              background: '#fff',
              maxHeight: 340,
              overflowY: 'auto',
            }}
          >
            <Row gutter={[8, 8]}>
              {filteredFields.map((field) => (
                <Col span={6} key={field.key}>
                  <FieldConfigCard
                    field={field}
                    checked={Boolean(columnMap.get(field.key))}
                    globalIndex={fieldOrderMap.get(field.key) || 1}
                    onToggle={() => handleToggleColumn(field.key)}
                    onDragStart={handleDragStart}
                    onDrop={handleDrop}
                    onMoveToTop={handleMoveToTop}
                    onMoveUp={handleMoveUp}
                    onMoveDown={handleMoveDown}
                  />
                </Col>
              ))}
            </Row>
            {filteredFields.length === 0 && (
              <div
                style={{
                  textAlign: 'center',
                  padding: '24px 0',
                  color: '#8c8c8c',
                }}
              >
                未搜索到匹配的字段
              </div>
            )}
          </div>
        </div>

        {/* 3. 7 类状态行背景高亮颜色设置 */}
        <RowColorSettings
          rowColors={rowColors}
          onColorChange={(key, color) =>
            setRowColors((prev) => ({ ...prev, [key]: color }))
          }
          onResetColors={handleResetColors}
        />
      </div>
    </Modal>
  );
}
