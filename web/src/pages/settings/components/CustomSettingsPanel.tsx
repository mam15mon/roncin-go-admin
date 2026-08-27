import {
  CheckCircleOutlined,
  DollarOutlined,
  InfoCircleOutlined,
  ReloadOutlined,
  SwapRightOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Col,
  Row,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useState } from 'react';
import { SectionCard } from '@/components/ui';
import {
  exchangeRateServiceGetExchangeRateCustomSetting,
  exchangeRateServiceUpdateExchangeRateCustomSetting,
} from '@/services/roncin/exchangeRateService';

const { Text, Paragraph } = Typography;

const businessNodeData = [
  {
    key: 'BILL',
    node: '建立账单',
    specialRate: '账单汇率',
    fallbackRate: '自动采用同期折本币汇率',
  },
  {
    key: 'INVOICE',
    node: '登记开票',
    specialRate: '开票汇率',
    fallbackRate: '自动采用同期折本币汇率',
  },
  {
    key: 'SETTLEMENT',
    node: '收付结算',
    specialRate: '结算汇率',
    fallbackRate: '自动采用同期折本币汇率',
  },
  {
    key: 'WRITE_OFF',
    node: '资金核销',
    specialRate: '核销汇率',
    fallbackRate: '自动采用同期折本币汇率',
  },
];

export function CustomSettingsPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [setting, setSetting] = useState<API.ExchangeRateCustomSetting>();

  const loadSetting = useCallback(async () => {
    setLoading(true);
    try {
      const response = await exchangeRateServiceGetExchangeRateCustomSetting();
      setSetting(response.data);
    } catch (e: any) {
      message.error(e.message || '获取汇率设置失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadSetting();
  }, [loadSetting]);

  const handleToggleInheritance = async (checked: boolean) => {
    setSaving(true);
    try {
      const response = await exchangeRateServiceUpdateExchangeRateCustomSetting({
        inheritBaseCurrencyRate: checked,
        expectedVersion: setting?.version ?? '0',
      });
      setSetting(response.data);
      message.success(
        checked
          ? '已开启：专用汇率未配置时自动继承折本币汇率'
          : '已关闭：专用汇率未配置时继承折本币汇率',
      );
    } catch (e: any) {
      message.error(e.message || '更新汇率设置失败，请刷新重试');
      loadSetting();
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    {
      title: '业务环节',
      dataIndex: 'node',
      key: 'node',
      width: 140,
      render: (val: string) => <Text strong>{val}</Text>,
    },
    {
      title: '专属汇率类型',
      dataIndex: 'specialRate',
      key: 'specialRate',
      width: 160,
      render: (val: string) => <Tag color="blue">{val}</Tag>,
    },
    {
      title: '未配置专属汇率时的取值规则',
      dataIndex: 'fallbackRate',
      key: 'fallbackRate',
      render: (val: string) => (
        <Space size={6}>
          <SwapRightOutlined style={{ color: '#1677ff' }} />
          <Text style={{ color: '#1677ff', fontWeight: 500 }}>{val}</Text>
        </Space>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        {/* 1. 核心开关卡片 */}
        <SectionCard
          title="财务汇率继承设置"
          extra={
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={loadSetting}
              loading={loading}
            >
              刷新
            </Button>
          }
        >
          <div
            style={{
              padding: '16px 20px',
              backgroundColor: '#fafafa',
              borderRadius: 8,
              border: '1px solid #f0f0f0',
            }}
          >
            <Row align="middle" justify="space-between" gutter={[16, 16]}>
              <Col xs={24} md={18}>
                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <Space align="center" size={8} wrap>
                    <DollarOutlined style={{ fontSize: 16, color: '#1677ff' }} />
                    <Text strong style={{ fontSize: 15, color: 'rgba(0, 0, 0, 0.88)' }}>
                      专用汇率未配置时，继承折本币汇率
                    </Text>
                    {setting?.inheritBaseCurrencyRate ? (
                      <Tag color="success">已开启继承</Tag>
                    ) : (
                      <Tag>默认关闭</Tag>
                    )}
                  </Space>
                  <Paragraph
                    type="secondary"
                    style={{ margin: 0, fontSize: 13, lineHeight: '22px' }}
                  >
                    开启后，账单、开票、结算和核销的专用汇率未配置时，系统将使用同一业务日期的折本币汇率。已配置的专用汇率始终优先。
                  </Paragraph>
                  {setting?.updatedAt && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      最近修改时间：{dayjs(setting.updatedAt).format('YYYY-MM-DD HH:mm:ss')}
                      {setting.updatedBy ? `（操作人：${setting.updatedBy}）` : ''}
                    </Text>
                  )}
                </Space>
              </Col>
              <Col xs={24} md={6} style={{ textAlign: 'right' }}>
                <Switch
                  checkedChildren="已开启"
                  unCheckedChildren="已关闭"
                  checked={Boolean(setting?.inheritBaseCurrencyRate)}
                  loading={saving}
                  disabled={!access.canUpdateExchangeRates || loading}
                  onChange={handleToggleInheritance}
                  style={{ minWidth: 70 }}
                />
              </Col>
            </Row>
          </div>
        </SectionCard>

        {/* 2. 业务生效规则与优先级 */}
        <SectionCard title="汇率生效与优先级规则">
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {/* 三步取值优先级 */}
            <Row gutter={[12, 12]}>
              <Col xs={24} md={8}>
                <Card
                  size="small"
                  style={{
                    height: '100%',
                    backgroundColor: '#fafafa',
                    borderColor: '#f0f0f0',
                  }}
                >
                  <Space direction="vertical" size={4}>
                    <Text strong style={{ color: '#1677ff', fontSize: 13 }}>
                      1. 优先使用专属汇率
                    </Text>
                    <Paragraph type="secondary" style={{ margin: 0, fontSize: 12, lineHeight: '20px' }}>
                      只要在「汇率设置」中配置了对应环节的专属汇率（如开票汇率、结算汇率等），系统始终优先采用专属汇率。
                    </Paragraph>
                  </Space>
                </Card>
              </Col>
              <Col xs={24} md={8}>
                <Card
                  size="small"
                  style={{
                    height: '100%',
                    backgroundColor: '#fafafa',
                    borderColor: '#f0f0f0',
                  }}
                >
                  <Space direction="vertical" size={4}>
                    <Text strong style={{ color: '#52c41a', fontSize: 13 }}>
                      2. 未设专属时继承折本币
                    </Text>
                    <Paragraph type="secondary" style={{ margin: 0, fontSize: 12, lineHeight: '20px' }}>
                      当某个环节未配置专属汇率且本开关开启时，系统自动匹配同币种、同业务日期的折本币汇率。
                    </Paragraph>
                  </Space>
                </Card>
              </Col>
              <Col xs={24} md={8}>
                <Card
                  size="small"
                  style={{
                    height: '100%',
                    backgroundColor: '#fafafa',
                    borderColor: '#f0f0f0',
                  }}
                >
                  <Space direction="vertical" size={4}>
                    <Text strong style={{ color: '#fa8c16', fontSize: 13 }}>
                      3. 严谨防错与拦截
                    </Text>
                    <Paragraph type="secondary" style={{ margin: 0, fontSize: 12, lineHeight: '20px' }}>
                      若折本币汇率也未维护，系统将明确提示“汇率未配置”，不会随意按 1.0 折算，确保财务数据准确。
                    </Paragraph>
                  </Space>
                </Card>
              </Col>
            </Row>

            {/* 业务环节表格 */}
            <div>
              <Text strong style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>
                适用的业务环节与汇率对照
              </Text>
              <Table
                columns={columns}
                dataSource={businessNodeData}
                pagination={false}
                size="small"
                bordered
              />
            </div>
          </Space>
        </SectionCard>
      </Space>
    </Spin>
  );
}

export default CustomSettingsPanel;
