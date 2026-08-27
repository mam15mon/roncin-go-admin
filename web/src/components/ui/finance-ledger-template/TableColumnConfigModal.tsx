import {
  CheckOutlined,
  ReloadOutlined,
  SearchOutlined,
  SettingOutlined,
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
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import {
  ALL_153_FINANCE_FIELDS,
  getDefaultColumnPreferences,
  getDefaultRowColors,
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

export function TableColumnConfigModal({
  open,
  onClose,
  currentPreference,
  onSaved,
}: TableColumnConfigModalProps) {
  const { message } = App.useApp();
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);

  // 1. 字段显隐与顺序映射
  const [columnMap, setColumnMap] = useState<Map<string, boolean>>(new Map());
  const [searchKeyword, setSearchKeyword] = useState('');

  // 2. 基础分页与排序
  const [pageSize, setPageSize] = useState<number>(40);
  const [sortField, setSortField] = useState<string>('');
  const [sortDirection, setSortDirection] = useState<'ASC' | 'DESC'>('DESC');

  // 3. 5 类状态行背景高亮颜色
  const [rowColors, setRowColors] = useState({
    unbilled: '#FFF7E6',
    unverifiedUninvoiced: '#FFFBE6',
    invoicedUnverified: '#E6F4FF',
    verifiedUninvoiced: '#F9F0FF',
    completed: '#F6FFED',
  });

  // 初始化数据
  useEffect(() => {
    if (!open) return;
    const defaultCols = getDefaultColumnPreferences();
    const map = new Map<string, boolean>();

    if (currentPreference?.columns && currentPreference.columns.length > 0) {
      currentPreference.columns.forEach((c) => {
        if (c.fieldKey) map.set(c.fieldKey, Boolean(c.visible));
      });
      // 补充可能未包含的新字段
      defaultCols.forEach((d) => {
        if (!map.has(d.fieldKey)) {
          map.set(d.fieldKey, d.visible);
        }
      });
    } else {
      defaultCols.forEach((d) => {
        map.set(d.fieldKey, d.visible);
      });
    }
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
      });
    } else {
      setRowColors(getDefaultRowColors());
    }
    setSearchKeyword('');
  }, [open, currentPreference]);

  // 过滤后的 153 字段
  const filteredFields = useMemo(() => {
    if (!searchKeyword.trim()) return ALL_153_FINANCE_FIELDS;
    const kw = searchKeyword.trim().toLowerCase();
    return ALL_153_FINANCE_FIELDS.filter(
      (f) =>
        f.name.toLowerCase().includes(kw) ||
        f.key.toLowerCase().includes(kw) ||
        String(f.id).includes(kw),
    );
  }, [searchKeyword]);

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
      ALL_153_FINANCE_FIELDS.forEach((f) => {
        next.set(f.key, true);
      });
      return next;
    });
  };

  // 反选
  const handleInvertSelect = () => {
    setColumnMap((prev) => {
      const next = new Map(prev);
      ALL_153_FINANCE_FIELDS.forEach((f) => {
        next.set(f.key, !prev.get(f.key));
      });
      return next;
    });
  };

  // 重置默认字段显隐
  const handleResetDefaultFields = () => {
    const defaultCols = getDefaultColumnPreferences();
    const map = new Map<string, boolean>();
    defaultCols.forEach((d) => {
      map.set(d.fieldKey, d.visible);
    });
    setColumnMap(map);
    message.success('已恢复系统默认推荐显示字段');
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
    } catch (e: any) {
      message.error(e.message || '重置系统默认配置失败');
    } finally {
      setResetting(false);
    }
  };

  // 提交保存配置
  const handleSave = async () => {
    if (selectedCount === 0) {
      message.warning('请至少保留一个可见字段');
      return;
    }
    setSaving(true);
    try {
      // 保持 153 项全量顺序
      const columnsPayload = ALL_153_FINANCE_FIELDS.map((f) => ({
        fieldKey: f.key,
        visible: Boolean(columnMap.get(f.key)),
      }));

      const res = await settlementServiceUpdateFeeLedgerPreference({
        columns: columnsPayload,
        pageSize,
        sortField: sortField || undefined,
        sortDirection: sortField ? sortDirection : undefined,
        rowColors,
        version: String(currentPreference?.version || '0'),
      });

      if (res.data) {
        message.success('表头与列表个性化配置已成功保存');
        onSaved(res.data);
        onClose();
      }
    } catch (e: any) {
      message.error(e.message || '保存配置失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <SettingOutlined style={{ color: '#1677ff', fontSize: 18 }} />
          <span style={{ fontWeight: 600, fontSize: 16 }}>表头设置与排序</span>
          <Tag color="blue" style={{ marginLeft: 4 }}>
            已选 {selectedCount} / 153 项
          </Tag>
        </div>
      }
      open={open}
      width={1020}
      destroyOnHidden
      onCancel={onClose}
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
                  ...ALL_153_FINANCE_FIELDS.filter((f) =>
                    columnMap.get(f.key),
                  ).map((f) => ({
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
                disabled={!sortField}
                value={sortDirection}
                onChange={(e) => setSortDirection(e.target.value)}
              >
                <Radio value="DESC">降序 (DESC)</Radio>
                <Radio value="ASC">升序 (ASC)</Radio>
              </Radio.Group>
            </Col>
          </Row>
        </Card>

        {/* 2. 153 项全量字段列表 */}
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
            <Space size={8}>
              <span style={{ fontWeight: 600, fontSize: 14 }}>
                全量业务字段配置（153 项）
              </span>
              <Button size="small" onClick={handleSelectAll}>
                全选
              </Button>
              <Button size="small" onClick={handleInvertSelect}>
                反选
              </Button>
              <Button size="small" onClick={handleResetDefaultFields}>
                重置默认推荐字段
              </Button>
            </Space>
            <Input
              allowClear
              prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              placeholder="请输入搜索字段名、字段标识或序号..."
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              style={{ width: 280 }}
            />
          </div>

          {/* 字段选择网格 */}
          <div
            style={{
              border: '1px solid #f0f0f0',
              borderRadius: 6,
              padding: 12,
              background: '#fff',
              maxHeight: 280,
              overflowY: 'auto',
            }}
          >
            <Row gutter={[8, 8]}>
              {filteredFields.map((field) => {
                const checked = Boolean(columnMap.get(field.key));
                return (
                  <Col span={6} key={field.key}>
                    <div
                      onClick={() => handleToggleColumn(field.key)}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: '5px 8px',
                        borderRadius: 4,
                        border: `1px solid ${checked ? '#91caff' : '#f0f0f0'}`,
                        background: checked ? '#e6f4ff' : '#fafafa',
                        cursor: 'pointer',
                        transition: 'all 0.15s ease',
                      }}
                    >
                      <Space size={6} style={{ overflow: 'hidden' }}>
                        <span
                          style={{
                            fontSize: 11,
                            color: '#8c8c8c',
                            minWidth: 22,
                            display: 'inline-block',
                          }}
                        >
                          #{field.id}
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
                      <Checkbox
                        checked={checked}
                        style={{ pointerEvents: 'none' }}
                      />
                    </div>
                  </Col>
                );
              })}
            </Row>
            {filteredFields.length === 0 && (
              <div
                style={{ textAlign: 'center', padding: '24px 0', color: '#8c8c8c' }}
              >
                未搜索到匹配的字段
              </div>
            )}
          </div>
        </div>

        {/* 3. 列表颜色设置（按费用状态高亮行背景） */}
        <Card
          size="small"
          title={
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <span>列表颜色设置（按费用业务状态高亮行背景）</span>
              <a
                style={{ fontSize: 12, fontWeight: 400 }}
                onClick={handleResetColors}
              >
                重置列表颜色
              </a>
            </div>
          }
          style={{ background: '#fafafa' }}
        >
          <Row gutter={[16, 12]}>
            {[
              {
                key: 'unbilled',
                label: '账单未建立',
                desc: '草稿或未生成对账单',
              },
              {
                key: 'unverifiedUninvoiced',
                label: '未核销未开票',
                desc: '已建账单但未开票未核销',
              },
              {
                key: 'invoicedUnverified',
                label: '已开票未核销',
                desc: '已开具发票待收款/付款核销',
              },
              {
                key: 'verifiedUninvoiced',
                label: '已核销未开票',
                desc: '款项已结清但未补齐发票',
              },
              {
                key: 'completed',
                label: '已完成',
                desc: '发票开具且款项全额核销完毕',
              },
            ].map((st) => {
              const currentColor = (rowColors as any)[st.key] || '#FFFFFF';
              return (
                <Col span={12} key={st.key}>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '6px 10px',
                      borderRadius: 6,
                      border: '1px solid #e8e8e8',
                      background: currentColor,
                    }}
                  >
                    <div>
                      <div style={{ fontWeight: 600, fontSize: 12 }}>
                        {st.label}
                      </div>
                      <div style={{ fontSize: 11, color: '#595959' }}>
                        {st.desc}
                      </div>
                    </div>
                    <Space size={6}>
                      {/* 快捷预设色块 */}
                      <Space size={2}>
                        {PRESET_COLORS.slice(0, 5).map((color) => (
                          <div
                            key={color}
                            onClick={() =>
                              setRowColors((prev) => ({
                                ...prev,
                                [st.key]: color,
                              }))
                            }
                            style={{
                              width: 16,
                              height: 16,
                              borderRadius: 3,
                              background: color,
                              border: `1px solid ${currentColor === color ? '#1677ff' : '#d9d9d9'}`,
                              cursor: 'pointer',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                            }}
                          >
                            {currentColor === color && (
                              <CheckOutlined
                                style={{ fontSize: 10, color: '#1677ff' }}
                              />
                            )}
                          </div>
                        ))}
                      </Space>
                      {/* 自定义拾色器 */}
                      <input
                        type="color"
                        value={currentColor}
                        onChange={(e) => {
                          const val = e.target.value.toUpperCase();
                          setRowColors((prev) => ({
                            ...prev,
                            [st.key]: val,
                          }));
                        }}
                        style={{
                          width: 26,
                          height: 24,
                          padding: 0,
                          border: 'none',
                          background: 'none',
                          cursor: 'pointer',
                        }}
                      />
                    </Space>
                  </div>
                </Col>
              );
            })}
          </Row>
        </Card>
      </div>
    </Modal>
  );
}
