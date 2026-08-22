import {
  EditOutlined,
  NumberOutlined,
  PlusOutlined,
  ReloadOutlined,
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
  Form,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import {
  masterDataServiceCreateNumberRule,
  masterDataServiceListNumberRules,
  masterDataServiceUpdateNumberRule,
} from '@/services/roncin/masterDataService';

const { Text } = Typography;

const documentTypeOptions = [
  { label: '订单 (Order)', value: 1, color: 'blue' },
  { label: '海运订舱 (Booking)', value: 2, color: 'cyan' },
  { label: '分单 HBL', value: 3, color: 'purple' },
  { label: '主单 MBL', value: 4, color: 'geekblue' },
  { label: '应收应付账单 (Bill)', value: 5, color: 'orange' },
  { label: '对账单 (Statement)', value: 6, color: 'gold' },
  { label: '收付款申请 (Payment)', value: 7, color: 'green' },
  { label: '发票 (Invoice)', value: 8, color: 'volcano' },
];

const documentTypeMap = new Map(
  documentTypeOptions.map((item) => [item.value, item]),
);

const dateFormatOptions = [
  { label: 'yyyyMMdd (年月日 20260822)', value: 1 },
  { label: 'yyyyMM (年月 202608)', value: 2 },
  { label: 'yyyy (年 2026)', value: 3 },
  { label: '无日期', value: 4 },
];

const dateFormatLabels = Object.fromEntries(
  dateFormatOptions.map((item) => [item.value, item.label]),
);

const resetPolicyOptions = [
  { label: '每日重置', value: 1 },
  { label: '每月重置', value: 2 },
  { label: '每年重置', value: 3 },
  { label: '永不重置', value: 4 },
];

const resetPolicyLabels = Object.fromEntries(
  resetPolicyOptions.map((item) => [item.value, item.label]),
);

export default function NumberRulesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<API.NumberRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<API.NumberRule | null>(null);
  const [form] = Form.useForm();

  const fetchRules = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListNumberRules({});
      setData(res.data || []);
    } catch {
      // Mock fallback
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleCreateOrUpdate = async (values: any) => {
    try {
      if (editingItem?.id) {
        await masterDataServiceUpdateNumberRule(
          { id: editingItem.id },
          {
            id: editingItem.id,
            prefix: values.prefix,
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
          prefix: values.prefix,
          dateFormat: values.dateFormat,
          sequenceLength: values.sequenceLength,
          resetPolicy: values.resetPolicy,
        });
        message.success('单号规则创建成功');
      }
      setModalOpen(false);
      fetchRules();
      return true;
    } catch {
      message.error('操作失败');
      return false;
    }
  };

  const columns: ColumnsType<API.NumberRule> = [
    {
      title: '单据类型',
      dataIndex: 'documentType',
      key: 'documentType',
      width: 180,
      render: (docType: number) => {
        const conf = documentTypeMap.get(docType);
        return (
          <Tag color={conf?.color || 'default'} style={{ fontSize: 12, padding: '2px 8px' }}>
            {conf?.label || `类型 ${docType}`}
          </Tag>
        );
      },
    },
    {
      title: '前缀',
      dataIndex: 'prefix',
      key: 'prefix',
      width: 120,
      render: (prefix) => (
        <Tag style={{ fontFamily: 'monospace', fontWeight: 600, color: '#1677ff', margin: 0 }}>
          {prefix}
        </Tag>
      ),
    },
    {
      title: '日期格式',
      dataIndex: 'dateFormat',
      key: 'dateFormat',
      width: 180,
      render: (format: number) => dateFormatLabels[format] || '-',
    },
    {
      title: '流水号位数',
      dataIndex: 'sequenceLength',
      key: 'sequenceLength',
      width: 110,
      render: (len) => `${len} 位`,
    },
    {
      title: '编号示例预览',
      key: 'preview',
      render: (_, record) => {
        const now = dayjs();
        let datePart = '';
        if (record.dateFormat === 1) datePart = now.format('YYYYMMDD');
        else if (record.dateFormat === 2) datePart = now.format('YYYYMM');
        else if (record.dateFormat === 3) datePart = now.format('YYYY');
        const seq = '1'.padStart(record.sequenceLength || 4, '0');
        const sample = `${record.prefix || ''}${datePart}${seq}`;
        return (
          <Tag
            style={{
              fontFamily: 'monospace',
              fontSize: 12,
              fontWeight: 600,
              backgroundColor: '#f6ffed',
              borderColor: '#b7eb8f',
              color: '#389e0d',
              padding: '2px 8px',
            }}
          >
            {sample}
          </Tag>
        );
      },
    },
    {
      title: '重置策略',
      dataIndex: 'resetPolicy',
      key: 'resetPolicy',
      width: 120,
      render: (policy: number) => resetPolicyLabels[policy] || '-',
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
      width: 100,
      align: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          style={{ padding: 0 }}
          onClick={() => {
            setEditingItem(record);
            form.setFieldsValue({ ...record });
            setModalOpen(true);
          }}
        >
          编辑
        </Button>
      ),
    },
  ];

  return (
    <div>
      {/* Header */}
      <Card
        size="small"
        bordered={false}
        style={{
          borderRadius: 8,
          marginBottom: 12,
          backgroundColor: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
        }}
        styles={{ body: { padding: '14px 20px' } }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
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
                单据编号规则设置
              </div>
              <Text type="secondary" style={{ fontSize: 12 }}>
                配置货代订单、海运提单、订舱单、财务账单等业务单据的自动流水号生成规则
              </Text>
            </div>
          </Space>

          <Space size={8}>
            <Button icon={<ReloadOutlined />} onClick={fetchRules}>
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingItem(null);
                form.resetFields();
                form.setFieldsValue({
                  dateFormat: 1,
                  sequenceLength: 4,
                  resetPolicy: 1,
                  enabled: true,
                });
                setModalOpen(true);
              }}
            >
              新建规则
            </Button>
          </Space>
        </div>
      </Card>

      {/* Table */}
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
        <Spin spinning={loading}>
          <Table
            columns={columns}
            dataSource={data}
            rowKey="id"
            pagination={false}
            size="middle"
          />
        </Spin>
      </Card>

      {/* Create / Edit Modal */}
      <ModalForm
        title={editingItem ? '编辑单号规则' : '新建单号生成规则'}
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleCreateOrUpdate}
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
            options={documentTypeOptions}
            placeholder="请选择要编号的业务单据"
            rules={[{ required: true, message: '请选择单据类型' }]}
            disabled={Boolean(editingItem)}
          />
        </Col>
        <Col span={24}>
          <ProFormText
            name="prefix"
            label="前缀代码"
            placeholder="例如：ORD、BKG、INV"
            rules={[
              { required: true, message: '请输入前缀代码' },
              { pattern: /^[A-Za-z0-9_-]+$/, message: '仅支持字母、数字与下划线' },
            ]}
          />
        </Col>
        <Col span={24}>
          <ProFormSelect
            name="dateFormat"
            label="日期格式"
            options={dateFormatOptions}
            rules={[{ required: true, message: '请选择日期格式' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormDigit
            name="sequenceLength"
            label="流水号位数"
            min={3}
            max={8}
            placeholder="例如：4 (生成 0001)"
            rules={[{ required: true, message: '请输入流水号位数' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormSelect
            name="resetPolicy"
            label="重置周期"
            options={resetPolicyOptions}
            rules={[{ required: true, message: '请选择重置周期' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormSwitch
            name="enabled"
            label="是否启用"
            checkedChildren="启用"
            unCheckedChildren="停用"
          />
        </Col>
      </ModalForm>
    </div>
  );
}
