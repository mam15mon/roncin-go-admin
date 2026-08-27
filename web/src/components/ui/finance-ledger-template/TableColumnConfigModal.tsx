import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CheckOutlined,
  HolderOutlined,
  ReloadOutlined,
  SearchOutlined,
  SettingOutlined,
  VerticalAlignTopOutlined,
} from '@ant-design/icons';
import {
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Row,
  Select,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import {
  ALL_153_FINANCE_FIELDS,
  getDefaultColumnPreferences,
  getDefaultRowColors,
  type FinanceFieldMeta,
} from './fields-meta';
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

const PRESET_COLORS = [
  '#FFF7E6', // 浅橙黄
  '#FFFBE6', // 浅黄
  '#E6F4FF', // 浅蓝
  '#F9F0FF', // 浅紫
  '#F6FFED', // 浅绿
  '#FFF0F6', // 浅粉
  '#F0F5FF', // 浅靛
  '#FCFFE6', // 浅柠
  '#FFFFFF', // 纯白（无高亮）
];

interface FieldConfigCardProps {
  field: FinanceFieldMeta;
  checked: boolean;
  globalIndex: number;
  onToggle: () => void;
  onDragStart: (key: string) => void;
  onDrop: (key: string) => void;
  onMoveToTop: (key: string) => void;
  onMoveUp: (key: string) => void;
  onMoveDown: (key: string) => void;
}

// 采用 React.memo 隔离 153 个卡片的渲染，消除全量 diff，大幅提升拖拽帧率
const FieldConfigCard = React.memo(function FieldConfigCard({
  field,
  checked,
  globalIndex,
  onToggle,
  onDragStart,
  onDrop,
  onMoveToTop,
  onMoveUp,
  onMoveDown,
}: FieldConfigCardProps) {
  const [isDragTarget, setIsDragTarget] = useState(false);

  return (
    <div
      draggable
      onDragStart={(e) => {
        onDragStart(field.key);
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', field.key);
        (e.currentTarget as HTMLElement).style.opacity = '0.4';
      }}
      onDragEnd={(e) => {
        (e.currentTarget as HTMLElement).style.opacity = '1';
        setIsDragTarget(false);
      }}
      onDragOver={(e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
      }}
      onDragEnter={() => setIsDragTarget(true)}
      onDragLeave={() => setIsDragTarget(false)}
      onDrop={(e) => {
        e.preventDefault();
        setIsDragTarget(false);
        onDrop(field.key);
      }}
      onClick={onToggle}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '6px 8px',
        borderRadius: 4,
        border: isDragTarget
          ? '2px dashed #1677ff'
          : `1px solid ${checked ? '#91caff' : '#f0f0f0'}`,
        background: isDragTarget
          ? '#e6f4ff'
          : checked
            ? '#e6f4ff'
            : '#fafafa',
        cursor: 'pointer',
        transition: 'border-color 0.12s, background-color 0.12s, box-shadow 0.12s',
        userSelect: 'none',
        willChange: 'transform',
        transform: 'translateZ(0)',
      }}
    >
      <Space size={6} style={{ overflow: 'hidden', flex: 1 }}>
        <Tooltip title="按住拖拽可调整前后顺序">
          <span
            style={{
              cursor: 'grab',
              color: '#bfbfbf',
              display: 'flex',
              alignItems: 'center',
            }}
            onMouseDown={(e) => e.stopPropagation()}
          >
            <HolderOutlined />
          </span>
        </Tooltip>
        <span
          style={{
            fontSize: 11,
            fontWeight: 600,
            color: checked ? '#1677ff' : '#8c8c8c',
            minWidth: 26,
            display: 'inline-block',
          }}
        >
          #{globalIndex}
        </span>
        <span
          style={{
            fontSize: 12,
            fontWeight: checked ? 500 : 400,
            color: checked ? '#1677ff' : '#262626',
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis',
            overflow: 'hidden',
          }}
        >
          {field.name}
        </span>
      </Space>

      {/* 右侧微调按钮与 Checkbox */}
      <Space size={4} style={{ marginLeft: 4 }}>
        <Tooltip title="置顶">
          <VerticalAlignTopOutlined
            style={{ fontSize: 11, color: '#8c8c8c' }}
            onClick={(e) => {
              e.stopPropagation();
              onMoveToTop(field.key);
            }}
          />
        </Tooltip>
        <Tooltip title="上移">
          <ArrowUpOutlined
            style={{ fontSize: 11, color: '#8c8c8c' }}
            onClick={(e) => {
              e.stopPropagation();
              onMoveUp(field.key);
            }}
          />
        </Tooltip>
        <Tooltip title="下移">
          <ArrowDownOutlined
            style={{ fontSize: 11, color: '#8c8c8c' }}
            onClick={(e) => {
              e.stopPropagation();
              onMoveDown(field.key);
            }}
          />
        </Tooltip>
        <Checkbox
          checked={checked}
          style={{ pointerEvents: 'none' }}
        />
      </Space>
    </div>
  );
});

export function TableColumnConfigModal({
  open,
  onClose,
  currentPreference,
  onSaved,
}: TableColumnConfigModalProps) {
  const { message } = App.useApp();
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);

  // 1. 字段有序列表与勾选状态
  const [fieldList, setFieldList] = useState<FinanceFieldMeta[]>(ALL_153_FINANCE_FIELDS);
  const [columnMap, setColumnMap] = useState<Map<string, boolean>>(new Map());
  const [searchKeyword, setSearchKeyword] = useState('');

  // 全局实时序号映射（确保搜索/分类过滤后编号依然精准反映真实全局先后顺序）
  const fieldOrderMap = useMemo(() => {
    const map = new Map<string, number>();
    fieldList.forEach((f, idx) => {
      map.set(f.key, idx + 1);
    });
    return map;
  }, [fieldList]);

  // 2. 基础分页与排序
  const [pageSize, setPageSize] = useState<number>(40);
  const [sortField, setSortField] = useState<string>('');
  const [sortDirection, setSortDirection] = useState<'ASC' | 'DESC'>('DESC');

  // 3. 状态行背景高亮颜色
  const [rowColors, setRowColors] = useState(getDefaultRowColors());

  // 初始化数据与顺序
  useEffect(() => {
    if (!open) return;
    const defaultCols = getDefaultColumnPreferences();
    const map = new Map<string, boolean>();
    const metaMap = new Map<string, FinanceFieldMeta>();
    ALL_153_FINANCE_FIELDS.forEach((f) => {
      metaMap.set(f.key, f);
    });

    const normalizeKey = (k: string) => {
      if (k === 'status') return 'financialProgress';
      if (k === 'customerName') return 'customerId';
      return k;
    };

    const ordered: FinanceFieldMeta[] = [];

    if (currentPreference?.columns && currentPreference.columns.length > 0) {
      currentPreference.columns.forEach((c) => {
        if (c.fieldKey) {
          const normKey = normalizeKey(c.fieldKey);
          map.set(normKey, Boolean(c.visible));
          const meta = metaMap.get(normKey);
          if (meta) {
            ordered.push(meta);
            metaMap.delete(normKey);
          }
        }
      });
      // 补充可能未包含的新字段
      metaMap.forEach((meta) => {
        ordered.push(meta);
        if (!map.has(meta.key)) {
          map.set(meta.key, false);
        }
      });
    } else {
      defaultCols.forEach((d) => {
        map.set(d.fieldKey, d.visible);
      });
      ordered.push(...ALL_153_FINANCE_FIELDS);
    }

    setFieldList(ordered);
    setColumnMap(map);

    setPageSize(currentPreference?.pageSize || 40);
    setSortField(currentPreference?.sortField || '');
    setSortDirection(
      (currentPreference?.sortDirection as 'ASC' | 'DESC') || 'DESC',
    );

    if (currentPreference?.rowColors) {
      setRowColors({
        unbilled: currentPreference.rowColors.unbilled || '#FFF7E6',
        unverifiedUninvoiced:
          currentPreference.rowColors.unverifiedUninvoiced || '#FFFBE6',
        invoicedUnverified:
          currentPreference.rowColors.invoicedUnverified || '#E6F4FF',
        verifiedUninvoiced:
          currentPreference.rowColors.verifiedUninvoiced || '#F9F0FF',
        completed: currentPreference.rowColors.completed || '#F6FFED',
        invoicedPartiallyVerified:
          currentPreference.rowColors.invoicedPartiallyVerified || '#E6F4FF',
        partiallyVerifiedUninvoiced:
          currentPreference.rowColors.partiallyVerifiedUninvoiced || '#F9F0FF',
      });
    } else {
      setRowColors(getDefaultRowColors());
    }
    setSearchKeyword('');
  }, [open, currentPreference]);

  const [filterType, setFilterType] = useState<'all' | 'checked' | 'unchecked'>('all');

  // 一键将所有已启用字段整齐置顶
  const handleSortCheckedFirst = () => {
    setFieldList((prev) => {
      const checkedList = prev.filter((f) => columnMap.get(f.key));
      const uncheckedList = prev.filter((f) => !columnMap.get(f.key));
      return [...checkedList, ...uncheckedList];
    });
    message.success('已将全部启用字段按顺序整齐置顶');
  };

  // 过滤后的 153 字段
  const filteredFields = useMemo(() => {
    let list = fieldList;
    if (filterType === 'checked') {
      list = list.filter((f) => columnMap.get(f.key));
    } else if (filterType === 'unchecked') {
      list = list.filter((f) => !columnMap.get(f.key));
    }

    if (!searchKeyword.trim()) return list;
    const kw = searchKeyword.trim().toLowerCase();
    return list.filter(
      (f) =>
        f.name.toLowerCase().includes(kw) ||
        f.key.toLowerCase().includes(kw),
    );
  }, [fieldList, filterType, columnMap, searchKeyword]);

  const selectedCount = useMemo(() => {
    let count = 0;
    columnMap.forEach((v) => {
      if (v) count++;
    });
    return count;
  }, [columnMap]);

  // 切换单列勾选
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
      fieldList.forEach((f) => {
        next.set(f.key, true);
      });
      return next;
    });
  };

  // 反选
  const handleInvertSelect = () => {
    setColumnMap((prev) => {
      const next = new Map(prev);
      fieldList.forEach((f) => {
        next.set(f.key, !prev.get(f.key));
      });
      return next;
    });
  };

  // 重置默认字段显隐与顺序
  const handleResetDefaultFields = () => {
    const defaultCols = getDefaultColumnPreferences();
    const map = new Map<string, boolean>();
    defaultCols.forEach((d) => {
      map.set(d.fieldKey, d.visible);
    });
    setFieldList(ALL_153_FINANCE_FIELDS);
    setColumnMap(map);
    message.success('已恢复系统默认推荐显示字段与初始顺序');
  };

  // 拖拽源 key（仅在 dragstart/drop/dragend 时更新一次）
  const draggedKeyRef = React.useRef<string | null>(null);

  // 拖拽排序事件（基于 key，0 冗余 re-render）
  const handleDragStart = (key: string) => {
    draggedKeyRef.current = key;
  };

  const handleDrop = (targetKey: string) => {
    const srcKey = draggedKeyRef.current;
    if (!srcKey || srcKey === targetKey) {
      draggedKeyRef.current = null;
      return;
    }
    setFieldList((prev) => {
      const next = [...prev];
      const srcIdx = next.findIndex((f) => f.key === srcKey);
      const tgtIdx = next.findIndex((f) => f.key === targetKey);
      if (srcIdx === -1 || tgtIdx === -1) return prev;
      const [item] = next.splice(srcIdx, 1);
      next.splice(tgtIdx, 0, item);
      return next;
    });
    draggedKeyRef.current = null;
  };

  // 快捷置顶（基于 key，全局安全）
  const handleMoveToTop = (key: string) => {
    setFieldList((prev) => {
      const idx = prev.findIndex((f) => f.key === key);
      if (idx <= 0) return prev;
      const next = [...prev];
      const [item] = next.splice(idx, 1);
      next.unshift(item);
      return next;
    });
  };

  // 快捷上移（基于 key，全局安全）
  const handleMoveUp = (key: string) => {
    setFieldList((prev) => {
      const idx = prev.findIndex((f) => f.key === key);
      if (idx <= 0) return prev;
      const next = [...prev];
      const [item] = next.splice(idx, 1);
      next.splice(idx - 1, 0, item);
      return next;
    });
  };

  // 快捷下移（基于 key，全局安全）
  const handleMoveDown = (key: string) => {
    setFieldList((prev) => {
      const idx = prev.findIndex((f) => f.key === key);
      if (idx === -1 || idx >= prev.length - 1) return prev;
      const next = [...prev];
      const [item] = next.splice(idx, 1);
      next.splice(idx + 1, 0, item);
      return next;
    });
  };

  // 重置颜色
  const handleResetColors = () => {
    setRowColors(getDefaultRowColors());
    message.success('已恢复默认列表高亮配色');
  };

  // 恢复系统默认（调用后端 reset 接口）
  const handleResetToSystemDefault = async () => {
    setResetting(true);
    try {
      const res = await settlementServiceResetFeeLedgerPreference({});
      if (res.data) {
        message.success('已彻底重置为系统默认配置');
        onSaved(res.data);
        onClose();
      }
    } catch {
      message.error('重置偏好配置失败');
    } finally {
      setResetting(false);
    }
  };

  // 提交保存配置
  const handleSave = async () => {
    setSaving(true);
    try {
      const columnsPayload = fieldList.map((field) => ({
        fieldKey: field.key,
        visible: Boolean(columnMap.get(field.key)),
      }));

      const res = await settlementServiceUpdateFeeLedgerPreference({
        pageSize,
        sortField: sortField || undefined,
        sortDirection: sortDirection || 'DESC',
        columns: columnsPayload,
        rowColors,
      });

      if (res.data) {
        message.success('表头偏好与排序已成功保存');
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
                  ...fieldList.filter((f) => columnMap.get(f.key)).map((f) => ({
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
                  { label: `未启用 (${fieldList.length - selectedCount})`, value: 'unchecked' },
                ]}
              />
              <Button size="small" type="dashed" onClick={handleSortCheckedFirst}>
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
        <Card
          size="small"
          title={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span>7 类费用财务进度 行背景高亮颜色设置</span>
              <Button size="small" onClick={handleResetColors}>
                重置默认颜色
              </Button>
            </div>
          }
          style={{ background: '#fafafa' }}
        >
          <Row gutter={[16, 12]}>
            {/* 1. 账单未建立 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.unbilled,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="gold">账单未建立</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （草稿/未出账单）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({ ...prev, unbilled: c }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.unbilled === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.unbilled === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>

            {/* 2. 未核销未开票 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.unverifiedUninvoiced,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="orange">未核销未开票</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （已确认待处理）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({
                          ...prev,
                          unverifiedUninvoiced: c,
                        }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.unverifiedUninvoiced === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.unverifiedUninvoiced === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>

            {/* 3. 已开票未核销 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.invoicedUnverified,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="blue">已开票未核销</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （发票已开待收付款）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({
                          ...prev,
                          invoicedUnverified: c,
                        }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.invoicedUnverified === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.invoicedUnverified === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>

            {/* 4. 已开票部分核销 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.invoicedPartiallyVerified,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="cyan">已开票部分核销</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （已开发票且部分收付款）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({
                          ...prev,
                          invoicedPartiallyVerified: c,
                        }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.invoicedPartiallyVerified === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.invoicedPartiallyVerified === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>

            {/* 5. 部分核销未开票 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.partiallyVerifiedUninvoiced,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="geekblue">部分核销未开票</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （已部分收付款待开票）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({
                          ...prev,
                          partiallyVerifiedUninvoiced: c,
                        }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.partiallyVerifiedUninvoiced === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.partiallyVerifiedUninvoiced === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>

            {/* 6. 已核销未开票 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.verifiedUninvoiced,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="purple">已核销未开票</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （资金已全收付待开票）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({
                          ...prev,
                          verifiedUninvoiced: c,
                        }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.verifiedUninvoiced === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.verifiedUninvoiced === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>

            {/* 7. 已完成 */}
            <Col span={12}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: rowColors.completed,
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <Space>
                  <Tag color="green">已完成</Tag>
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    （已全额核销并开票完毕）
                  </span>
                </Space>
                <Space size={4}>
                  {PRESET_COLORS.map((c) => (
                    <div
                      key={c}
                      onClick={() =>
                        setRowColors((prev) => ({ ...prev, completed: c }))
                      }
                      style={{
                        width: 18,
                        height: 18,
                        background: c,
                        border:
                          rowColors.completed === c
                            ? '2px solid #1677ff'
                            : '1px solid #d9d9d9',
                        borderRadius: 3,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {rowColors.completed === c && (
                        <CheckOutlined
                          style={{ fontSize: 10, color: '#1677ff' }}
                        />
                      )}
                    </div>
                  ))}
                </Space>
              </div>
            </Col>
          </Row>
        </Card>
      </div>
    </Modal>
  );
}
