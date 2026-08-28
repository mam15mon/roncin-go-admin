import {
  AppstoreOutlined,
  CopyOutlined,
  EditOutlined,
  NumberOutlined,
  PlusOutlined,
  ReloadOutlined,
  TableOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
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
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import {
  masterDataServiceCreateNumberRule,
  masterDataServiceListNumberRules,
  masterDataServiceUpdateNumberRule,
} from '@/services/roncin/masterDataService';
import NumberRuleCard from './number-rules/NumberRuleCard';
import NumberRuleEditModal from './number-rules/NumberRuleEditModal';
import {
  DATE_FORMATS,
  DOC_TYPES,
  type DocTypeMeta,
  RESET_POLICIES,
  docTypeMap,
  filterVisibleNumberRules,
  generatePreviewNumber,
} from './number-rules/numberRulesConstants';

export {
  DOC_TYPES,
  DATE_FORMATS,
  RESET_POLICIES,
  docTypeMap,
  filterVisibleNumberRules,
  generatePreviewNumber,
  type DocTypeMeta,
};

const { Text } = Typography;

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
          const leftOrder =
            docTypeMap.get(left.documentType as any)?.numValue ??
            Number.MAX_SAFE_INTEGER;
          const rightOrder =
            docTypeMap.get(right.documentType as any)?.numValue ??
            Number.MAX_SAFE_INTEGER;
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
      enabled: Boolean(rule.enabled),
    });
    setModalOpen(true);
  };

  // Toggle active directly on card
  const handleToggleRuleActive = async (
    rule: API.NumberRule,
    checked: boolean,
  ) => {
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
      message.success(
        `${docMeta?.shortLabel || '单号规则'}已${checked ? '启用' : '停用'}`,
      );
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
          <Tag
            color={meta?.color || 'blue'}
            style={{ fontSize: 12, padding: '2px 8px' }}
          >
            {meta?.label || `单据 ${docType}`}
          </Tag>
        );
      },
    },
    {
      title: '前缀代码',
      dataIndex: 'prefix',
      key: 'prefix',
      width: 140,
      render: (prefix, record) => {
        const meta = docTypeMap.get(record.documentType as any);
        if (meta?.numValue === 1) {
          return (
            <Tag
              color="blue"
              style={{
                fontFamily: 'ui-monospace, monospace',
                fontWeight: 600,
                margin: 0,
              }}
            >
              {prefix ? `${prefix}-{业务代码}` : '{业务代码}'}
            </Tag>
          );
        }
        return (
          <Tag
            style={{
              fontFamily: 'ui-monospace, monospace',
              fontWeight: 600,
              color: prefix ? '#1677ff' : '#8c8c8c',
              margin: 0,
            }}
          >
            {prefix || '无'}
          </Tag>
        );
      },
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
      width: 240,
      render: (_, record) => {
        const sample = generatePreviewNumber(record);
        return (
          <Space size={4}>
            <Tag
              style={{
                fontFamily:
                  'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                fontSize: 12,
                fontWeight: 600,
                backgroundColor: '#f6ffed',
                borderColor: '#b7eb8f',
                color: '#389e0d',
                padding: '2px 8px',
                margin: 0,
                letterSpacing: '0.5px',
              }}
            >
              {sample.text}
            </Tag>
            <Tooltip title="复制示例单号">
              <Button
                type="text"
                size="small"
                icon={
                  <CopyOutlined
                    style={{ fontSize: 11, color: '#8c8c8c' }}
                  />
                }
                onClick={() => handleCopyPreview(sample.text)}
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
        variant="borderless"
        style={{
          borderRadius: 8,
          marginBottom: 16,
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
        }}
        styles={{ body: { padding: '14px 20px' } }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
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
              <div
                style={{
                  fontSize: 16,
                  fontWeight: 600,
                  color: 'rgba(0, 0, 0, 0.88)',
                }}
              >
                单据自动编号规则
              </div>
              <Text type="secondary" style={{ fontSize: 12 }}>
                集中维护订单、账单、批次、发票、核销、流水及提成等全链路 13
                类业务单据的自动出号规则
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
            {data.length < DOC_TYPES.length && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingItem(null);
                  const existingTypes = new Set(
                    data.map(
                      (r) =>
                        docTypeMap.get(r.documentType as any)?.numValue,
                    ),
                  );
                  const firstUnused =
                    DOC_TYPES.find((t) => !existingTypes.has(t.numValue)) ||
                    DOC_TYPES[0];
                  form.resetFields();
                  form.setFieldsValue({
                    documentType: firstUnused.numValue,
                    prefix: firstUnused.defaultPrefix,
                    dateFormat: 1,
                    sequenceLength: 5,
                    resetPolicy: 1,
                    enabled: true,
                  });
                  setModalOpen(true);
                }}
              >
                新建编号规则
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* 2. Main Content: Card Grid View or Table View */}
      <Spin spinning={loading}>
        {data.length === 0 ? (
          <Card
            variant="borderless"
            style={{ borderRadius: 8, padding: '40px 0' }}
          >
            <Empty description="暂无单号规则配置" />
          </Card>
        ) : viewMode === 'card' ? (
          /* Card Grid View */
          <Row gutter={[16, 16]}>
            {data.map((rule) => {
              const meta: DocTypeMeta = docTypeMap.get(
                rule.documentType as any,
              ) || {
                key: String(rule.documentType),
                numValue: Number(rule.documentType) || 0,
                label: `单据类型 ${rule.documentType}`,
                shortLabel: '单据',
                color: 'blue',
                defaultPrefix: rule.prefix || 'DOC',
              };

              return (
                <Col xs={24} sm={12} md={8} lg={6} key={rule.id}>
                  <NumberRuleCard
                    rule={rule}
                    meta={meta}
                    onToggleActive={handleToggleRuleActive}
                    onOpenEdit={handleOpenEdit}
                    onCopyPreview={handleCopyPreview}
                  />
                </Col>
              );
            })}
          </Row>
        ) : (
          /* Table View */
          <Card
            size="small"
            variant="borderless"
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
      <NumberRuleEditModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        editingItem={editingItem || undefined}
        data={data}
        form={form}
        onFinish={handleFormFinish}
      />
    </div>
  );
}

export default NumberRulesPanel;
