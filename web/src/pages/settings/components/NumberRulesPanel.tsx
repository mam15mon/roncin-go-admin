import {
  AppstoreOutlined,
  CopyOutlined,
  EditOutlined,
  NumberOutlined,
  PlusOutlined,
  ReloadOutlined,
  TableOutlined,
} from '@ant-design/icons';
import {
  ModalForm,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import {
  App,
  Badge,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Radio,
  Row,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useState } from 'react';
import {
  masterDataServiceCreateNumberRule,
  masterDataServiceListNumberRules,
  masterDataServiceUpdateNumberRule,
} from '@/services/roncin/masterDataService';

const { Text } = Typography;

// 1. 单据类型枚举映射 (支持字符串与数字双向兼容)
export interface DocTypeMeta {
  key: string;
  numValue: number;
  label: string;
  shortLabel: string;
  color: string;
  defaultPrefix: string;
  businessCodes?: string[];
}

export const DOC_TYPES: DocTypeMeta[] = [
  { key: 'DOCUMENT_TYPE_ORDER', numValue: 1, label: '订单编号设置', shortLabel: '订单', color: 'blue', defaultPrefix: '', businessCodes: ['SE', 'SI', 'AE', 'AI'] },
];

const docTypeMap = new Map<string | number, DocTypeMeta>();
DOC_TYPES.forEach((t) => {
  docTypeMap.set(t.key, t);
  docTypeMap.set(t.numValue, t);
  docTypeMap.set(String(t.numValue), t);
  docTypeMap.set(t.key.replace('DOCUMENT_TYPE_', ''), t);
});

export function filterVisibleNumberRules(rules: API.NumberRule[]): API.NumberRule[] {
  return rules.filter((rule) => docTypeMap.has(rule.documentType as any));
}

// 2. 日期格式枚举映射 (支持字符串与数字双向兼容)
const DATE_FORMATS: Record<string | number, { label: string; formatStr: string; numValue: number }> = {
  DATE_FORMAT_YYYYMMDD: { label: '年月日 (yyyyMMdd)', formatStr: 'YYYYMMDD', numValue: 1 },
  1: { label: '年月日 (yyyyMMdd)', formatStr: 'YYYYMMDD', numValue: 1 },
  DATE_FORMAT_YYYYMM: { label: '年月 (yyyyMM)', formatStr: 'YYYYMM', numValue: 2 },
  2: { label: '年月 (yyyyMM)', formatStr: 'YYYYMM', numValue: 2 },
  DATE_FORMAT_YYYY: { label: '年 (yyyy)', formatStr: 'YYYY', numValue: 3 },
  3: { label: '年 (yyyy)', formatStr: 'YYYY', numValue: 3 },
  DATE_FORMAT_NONE: { label: '无日期', formatStr: '', numValue: 4 },
  4: { label: '无日期', formatStr: '', numValue: 4 },
  DATE_FORMAT_UNSPECIFIED: { label: '无日期', formatStr: '', numValue: 4 },
  0: { label: '无日期', formatStr: '', numValue: 4 },
};

// 3. 重置周期枚举映射 (支持字符串与数字双向兼容)
const RESET_POLICIES: Record<string | number, { label: string; numValue: number }> = {
  RESET_POLICY_DAILY: { label: '每日重置', numValue: 1 },
  1: { label: '每日重置', numValue: 1 },
  RESET_POLICY_MONTHLY: { label: '每月重置', numValue: 2 },
  2: { label: '每月重置', numValue: 2 },
  RESET_POLICY_YEARLY: { label: '每年重置', numValue: 3 },
  3: { label: '每年重置', numValue: 3 },
  RESET_POLICY_NEVER: { label: '永不重置', numValue: 4 },
  4: { label: '永不重置', numValue: 4 },
  RESET_POLICY_UNSPECIFIED: { label: '永不重置', numValue: 4 },
  0: { label: '永不重置', numValue: 4 },
};

// 工具函数：解析生成规则示例预览
function generatePreviewNumber(rule: API.NumberRule): string {
  const meta = docTypeMap.get(rule.documentType as any);
  const prefix = `${rule.prefix || ''}${meta?.businessCodes?.[0] || ''}`;
  const now = dayjs();
  const dateMeta = DATE_FORMATS[rule.dateFormat as any] || DATE_FORMATS[4];
  const dateStr = dateMeta.formatStr ? now.format(dateMeta.formatStr) : '';
  const len = Number(rule.sequenceLength) || 4;
  const seq = '1'.padStart(len, '0');
  return `${prefix}${dateStr}${seq}`;
}

export function NumberRulesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<API.NumberRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [viewMode, setViewMode] = useState<'card' | 'table'>('card');

  // Dialog State
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<API.NumberRule | null>(null);
  const [form] = Form.useForm();

  // Load Rules
  const fetchRules = useCallback(async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListNumberRules({});
      setData(
        filterVisibleNumberRules(res.data || []).sort((left, right) => {
          const leftOrder = docTypeMap.get(left.documentType as any)?.numValue ?? Number.MAX_SAFE_INTEGER;
          const rightOrder = docTypeMap.get(right.documentType as any)?.numValue ?? Number.MAX_SAFE_INTEGER;
          return leftOrder - rightOrder;
        }),
      );
    } catch {
      message.error('单号规则加载失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  // Handle Edit Click
  const handleOpenEdit = (rule: API.NumberRule) => {
    setEditingItem(rule);
    const docMeta = docTypeMap.get(rule.documentType as any);
    const dateMeta = DATE_FORMATS[rule.dateFormat as any];
    const resetMeta = RESET_POLICIES[rule.resetPolicy as any];

    form.resetFields();
    form.setFieldsValue({
      documentType: docMeta?.numValue || rule.documentType,
      prefix: rule.prefix,
      dateFormat: dateMeta?.numValue || 1,
      sequenceLength: rule.sequenceLength || 4,
      resetPolicy: resetMeta?.numValue || 1,
      enabled: rule.enabled ?? true,
    });
    setModalOpen(true);
  };

  // Toggle active directly on card
  const handleToggleRuleActive = async (rule: API.NumberRule, checked: boolean) => {
    if (!rule.id) return;
    try {
      const dateMeta = DATE_FORMATS[rule.dateFormat as any];
      const resetMeta = RESET_POLICIES[rule.resetPolicy as any];
      await masterDataServiceUpdateNumberRule(
        { id: rule.id },
        {
          id: rule.id,
          prefix: rule.prefix,
          dateFormat: dateMeta?.numValue || 1,
          sequenceLength: rule.sequenceLength || 4,
          resetPolicy: resetMeta?.numValue || 1,
          enabled: checked,
        },
      );
      setData((prev) =>
        prev.map((r) => (r.id === rule.id ? { ...r, enabled: checked } : r)),
      );
      const docMeta = docTypeMap.get(rule.documentType as any);
      message.success(`${docMeta?.shortLabel || '单号规则'}已${checked ? '启用' : '停用'}`);
    } catch {
      message.error('状态更新失败');
    }
  };

  // Submit Create / Edit
  const handleFormFinish = async (values: any) => {
    try {
      if (editingItem?.id) {
        await masterDataServiceUpdateNumberRule(
          { id: editingItem.id },
          {
            id: editingItem.id,
            prefix: values.prefix?.trim(),
            dateFormat: values.dateFormat,
            sequenceLength: values.sequenceLength,
            resetPolicy: values.resetPolicy,
            enabled: values.enabled,
          },
        );
        message.success('单号规则已更新');
      } else {
        await masterDataServiceCreateNumberRule({
          documentType: values.documentType,
          prefix: values.prefix?.trim(),
          dateFormat: values.dateFormat,
          sequenceLength: values.sequenceLength,
          resetPolicy: values.resetPolicy,
        });
        message.success('单号规则创建成功');
      }
      setModalOpen(false);
      await fetchRules();
      return true;
    } catch {
      message.error('操作失败');
      return false;
    }
  };

  // Copy Preview Number
  const handleCopyPreview = (num: string) => {
    navigator.clipboard?.writeText(num);
    message.success(`已复制单号示例: ${num}`);
  };

  // Table Columns (for Table Mode)
  const columns: ColumnsType<API.NumberRule> = [
    {
      title: '单据类型',
      dataIndex: 'documentType',
      key: 'documentType',
      width: 180,
      render: (docType) => {
        const meta = docTypeMap.get(docType);
        return (
          <Tag color={meta?.color || 'blue'} style={{ fontSize: 12, padding: '2px 8px' }}>
            {meta?.label || `单据 ${docType}`}
          </Tag>
        );
      },
    },
    {
      title: '前缀代码',
      dataIndex: 'prefix',
      key: 'prefix',
      width: 110,
      render: (prefix) => (
        <Tag style={{ fontFamily: 'monospace', fontWeight: 600, color: '#1677ff', margin: 0 }}>
          {prefix || '无'}
        </Tag>
      ),
    },
    {
      title: '日期格式',
      dataIndex: 'dateFormat',
      key: 'dateFormat',
      width: 170,
      render: (format) => {
        const meta = DATE_FORMATS[format as any];
        return meta?.label || '-';
      },
    },
    {
      title: '流水号位数',
      dataIndex: 'sequenceLength',
      key: 'sequenceLength',
      width: 110,
      render: (len) => `${len || 4} 位`,
    },
    {
      title: '重置策略',
      dataIndex: 'resetPolicy',
      key: 'resetPolicy',
      width: 120,
      render: (policy) => {
        const meta = RESET_POLICIES[policy as any];
        return meta?.label || '-';
      },
    },
    {
      title: '单号示例预览',
      key: 'preview',
      width: 220,
      render: (_, record) => {
        const sample = generatePreviewNumber(record);
        return (
          <Space size={4}>
            <Tag
              style={{
                fontFamily: 'monospace',
                fontSize: 12,
                fontWeight: 600,
                backgroundColor: '#f6ffed',
                borderColor: '#b7eb8f',
                color: '#389e0d',
                padding: '2px 8px',
                margin: 0,
              }}
            >
              {sample}
            </Tag>
            <Tooltip title="复制示例单号">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />}
                onClick={() => handleCopyPreview(sample)}
                style={{ width: 20, height: 20, padding: 0 }}
              />
            </Tooltip>
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 90,
      render: (enabled) => (
        <Badge
          status={enabled ? 'success' : 'default'}
          text={enabled ? '启用' : '停用'}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 90,
      align: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          style={{ padding: 0 }}
          onClick={() => handleOpenEdit(record)}
        >
          编辑
        </Button>
      ),
    },
  ];

  return (
    <div>
      {/* 1. Header Toolbar */}
      <Card
        size="small"
        bordered={false}
        style={{
          borderRadius: 8,
          marginBottom: 16,
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
        }}
        styles={{ body: { padding: '14px 20px' } }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
          <Space size={10} align="center">
            <div
              style={{
                width: 38,
                height: 38,
                borderRadius: 8,
                backgroundColor: '#e6f4ff',
                color: '#1677ff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 20,
              }}
            >
              <NumberOutlined />
            </div>
            <div>
              <div style={{ fontSize: 16, fontWeight: 600, color: 'rgba(0, 0, 0, 0.88)' }}>
                订单编号规则设置
              </div>
              <Text type="secondary" style={{ fontSize: 12 }}>
                当前仅订单编号已接入业务流程，按 SE、SI、AE、AI 自动拼接业务代码
              </Text>
            </div>
          </Space>

          {/* Action Tools */}
          <Space size={10}>
            {/* View Switcher: Card vs Table */}
            <Radio.Group
              value={viewMode}
              onChange={(e) => setViewMode(e.target.value)}
              buttonStyle="solid"
              size="middle"
            >
              <Radio.Button value="card">
                <AppstoreOutlined style={{ marginRight: 4 }} />
                卡片视图
              </Radio.Button>
              <Radio.Button value="table">
                <TableOutlined style={{ marginRight: 4 }} />
                表格视图
              </Radio.Button>
            </Radio.Group>

            <Button icon={<ReloadOutlined />} onClick={fetchRules}>
              刷新
            </Button>
            {data.length === 0 && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingItem(null);
                  form.resetFields();
                  form.setFieldsValue({
                    documentType: 1,
                    prefix: '',
                    dateFormat: 1,
                    sequenceLength: 5,
                    resetPolicy: 1,
                    enabled: true,
                  });
                  setModalOpen(true);
                }}
              >
                新建订单编号规则
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* 2. Main Content: Card Grid View or Table View */}
      <Spin spinning={loading}>
        {data.length === 0 ? (
          <Card bordered={false} style={{ borderRadius: 8, padding: '40px 0' }}>
            <Empty description="暂无单号规则配置" />
          </Card>
        ) : viewMode === 'card' ? (
          /* Card Grid View */
          <Row gutter={[16, 16]}>
            {data.map((rule) => {
              const meta: DocTypeMeta = docTypeMap.get(rule.documentType as any) || {
                key: String(rule.documentType),
                numValue: Number(rule.documentType) || 0,
                label: `单据类型 ${rule.documentType}`,
                shortLabel: '单据',
                color: 'blue',
                defaultPrefix: rule.prefix || 'DOC',
              };
              const dateMeta = DATE_FORMATS[rule.dateFormat as any] || { label: '无日期', formatStr: '' };
              const resetMeta = RESET_POLICIES[rule.resetPolicy as any] || { label: '永不重置' };
              const sampleNumber = generatePreviewNumber(rule);

              return (
                <Col xs={24} sm={12} md={8} lg={6} key={rule.id || rule.documentType}>
                  <Card
                    size="small"
                    hoverable
                    style={{
                      borderRadius: 8,
                      border: '1px solid #e8e8e8',
                      overflow: 'hidden',
                      height: '100%',
                      display: 'flex',
                      flexDirection: 'column',
                      boxShadow: '0 2px 6px rgba(0, 0, 0, 0.02)',
                      transition: 'all 0.3s cubic-bezier(0.645, 0.045, 0.355, 1)',
                    }}
                    styles={{
                      body: {
                        padding: '16px',
                        display: 'flex',
                        flexDirection: 'column',
                        flex: 1,
                        justifyContent: 'space-between',
                      },
                    }}
                  >
                    {/* Card Top Strip */}
                    <div>
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'flex-start',
                          marginBottom: 12,
                        }}
                      >
                        <div>
                          <Tag
                            color={meta.color}
                            style={{
                              fontSize: 12,
                              fontWeight: 600,
                              padding: '2px 8px',
                              marginBottom: 4,
                            }}
                          >
                            {meta.shortLabel}
                          </Tag>
                          <div style={{ fontSize: 14, fontWeight: 600, color: '#262626' }}>
                            {meta.label}
                          </div>
                        </div>

                        {/* Status Switch */}
                        <Switch
                          size="small"
                          checked={rule.enabled ?? true}
                          onChange={(checked) => handleToggleRuleActive(rule, checked)}
                        />
                      </div>

                      {/* Rule Attributes Table List */}
                      <div
                        style={{
                          backgroundColor: '#fafafa',
                          borderRadius: 6,
                          padding: '8px 12px',
                          fontSize: 12,
                          marginBottom: 14,
                          display: 'flex',
                          flexDirection: 'column',
                          gap: 6,
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                          <span style={{ color: '#8c8c8c' }}>前缀代码：</span>
                          <span style={{ fontFamily: 'monospace', fontWeight: 600, color: '#1677ff' }}>
                            {rule.prefix || '无'}
                          </span>
                        </div>
                        {meta.businessCodes && (
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span style={{ color: '#8c8c8c' }}>业务编号：</span>
                            <span style={{ fontFamily: 'monospace', color: '#262626' }}>
                              {meta.businessCodes.join(' / ')}
                            </span>
                          </div>
                        )}
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                          <span style={{ color: '#8c8c8c' }}>日期格式：</span>
                          <span style={{ color: '#262626' }}>{dateMeta.label}</span>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                          <span style={{ color: '#8c8c8c' }}>流水位数：</span>
                          <span style={{ color: '#262626' }}>{rule.sequenceLength || 4} 位数字</span>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                          <span style={{ color: '#8c8c8c' }}>重置策略：</span>
                          <span style={{ color: '#262626' }}>{resetMeta.label}</span>
                        </div>
                      </div>
                    </div>

                    {/* Card Bottom: Live Preview Box & Actions */}
                    <div>
                      <div style={{ fontSize: 11, color: '#8c8c8c', marginBottom: 4 }}>
                        生成单号示例预览：
                      </div>
                      <div
                        style={{
                          backgroundColor: '#f6ffed',
                          border: '1px dashed #b7eb8f',
                          borderRadius: 6,
                          padding: '6px 10px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          marginBottom: 12,
                        }}
                      >
                        <span
                          style={{
                            fontFamily: 'monospace',
                            fontSize: 13,
                            fontWeight: 600,
                            color: '#389e0d',
                            letterSpacing: '0.5px',
                          }}
                        >
                          {sampleNumber}
                        </span>
                        <Tooltip title="复制示例">
                          <Button
                            type="text"
                            size="small"
                            icon={<CopyOutlined style={{ fontSize: 12, color: '#52c41a' }} />}
                            onClick={() => handleCopyPreview(sampleNumber)}
                            style={{ height: 22, width: 22, padding: 0 }}
                          />
                        </Tooltip>
                      </div>

                      {/* Edit Button */}
                      <Button
                        type="default"
                        block
                        icon={<EditOutlined />}
                        onClick={() => handleOpenEdit(rule)}
                        style={{ borderRadius: 6 }}
                      >
                        修改规则
                      </Button>
                    </div>
                  </Card>
                </Col>
              );
            })}
          </Row>
        ) : (
          /* Table View */
          <Card
            size="small"
            bordered={false}
            style={{
              borderRadius: 8,
              backgroundColor: '#ffffff',
              boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
            }}
            styles={{ body: { padding: 0 } }}
          >
            <Table
              columns={columns}
              dataSource={data}
              rowKey="id"
              pagination={false}
              size="middle"
            />
          </Card>
        )}
      </Spin>

      {/* 3. Create / Edit Modal Form */}
      <ModalForm
        title={editingItem ? '编辑订单编号规则' : '新建订单编号规则'}
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleFormFinish}
        modalProps={{
          destroyOnClose: true,
          maskClosable: false,
          width: 520,
        }}
        layout="horizontal"
        grid
      >
        <Col span={24}>
          <ProFormSelect
            name="documentType"
            label="单据类型"
            options={DOC_TYPES.map((t) => ({ label: t.label, value: t.numValue }))}
            placeholder="请选择单据类型"
            rules={[{ required: true, message: '请选择单据类型' }]}
            disabled={Boolean(editingItem)}
          />
        </Col>
        <Col span={24}>
          <ProFormText
            name="prefix"
            label="前缀代码"
            placeholder="可选，例如：OR"
            rules={[
              { pattern: /^[A-Za-z0-9_-]*$/, message: '仅支持字母、数字与下划线' },
            ]}
          />
        </Col>
        <Col span={24}>
          <ProFormSelect
            name="dateFormat"
            label="日期格式"
            options={[
              { label: '年月日 (yyyyMMdd 示例: 20260823)', value: 1 },
              { label: '年月 (yyyyMM 示例: 202608)', value: 2 },
              { label: '年 (yyyy 示例: 2026)', value: 3 },
              { label: '无日期 (仅前缀+流水号)', value: 4 },
            ]}
            rules={[{ required: true, message: '请选择日期格式' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormDigit
            name="sequenceLength"
            label="流水号位数"
            min={1}
            max={12}
            placeholder="例如：4 (生成 0001)"
            rules={[{ required: true, message: '请输入流水号位数 (1-12)' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormSelect
            name="resetPolicy"
            label="重置周期"
            options={[
              { label: '每日重置 (推荐配合年月日)', value: 1 },
              { label: '每月重置 (推荐配合年月)', value: 2 },
              { label: '每年重置 (推荐配合年)', value: 3 },
              { label: '永不重置 (递增累加)', value: 4 },
            ]}
            rules={[{ required: true, message: '请选择重置周期' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormSwitch
            name="enabled"
            label="启用状态"
            checkedChildren="启用"
            unCheckedChildren="停用"
          />
        </Col>
      </ModalForm>
    </div>
  );
}

export default NumberRulesPanel;
