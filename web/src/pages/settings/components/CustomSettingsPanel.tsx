import {
  DollarOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
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

const inheritanceRulesData = [
  {
    key: 'BILL',
    node: '建立账单',
    rateType: 'BILL (账单汇率)',
    fallback: '查询同期 BASE_CURRENCY (折本币) 汇率',
    sourceTag: 'INHERITED_BASE_CURRENCY',
  },
  {
    key: 'INVOICE',
    node: '登记开票',
    rateType: 'INVOICE (开票汇率)',
    fallback: '查询同期 BASE_CURRENCY (折本币) 汇率',
    sourceTag: 'INHERITED_BASE_CURRENCY',
  },
  {
    key: 'SETTLEMENT',
    node: '收付结算',
    rateType: 'SETTLEMENT (结算汇率)',
    fallback: '查询同期 BASE_CURRENCY (折本币) 汇率',
    sourceTag: 'INHERITED_BASE_CURRENCY',
  },
  {
    key: 'WRITE_OFF',
    node: '资金核销',
    rateType: 'WRITE_OFF (核销汇率)',
    fallback: '查询同期 BASE_CURRENCY (折本币) 汇率',
    sourceTag: 'INHERITED_BASE_CURRENCY',
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
      message.error(e.message || '获取汇率自定义设置失败');
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
          ? '已成功开启「专用汇率未配置时继承折本币汇率」'
          : '已关闭「专用汇率未配置时继承折本币汇率」',
      );
    } catch (e: any) {
      message.error(e.message || '更新汇率自定义设置失败');
      // 冲突或失败时重新拉取最新版本
      loadSetting();
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    {
      title: '业务节点',
      dataIndex: 'node',
      key: 'node',
      width: 140,
      render: (val: string) => <Text strong>{val}</Text>,
    },
    {
      title: '专用汇率类型',
      dataIndex: 'rateType',
      key: 'rateType',
      width: 180,
      render: (val: string) => <Tag color="blue">{val}</Tag>,
    },
    {
      title: '专用汇率未配置时处理',
      dataIndex: 'fallback',
      key: 'fallback',
      render: (val: string) => <Text style={{ color: '#1677ff' }}>{val}</Text>,
    },
    {
      title: '财务追溯标识 (exchange_rate_source)',
      dataIndex: 'sourceTag',
      key: 'sourceTag',
      width: 260,
      render: (val: string) => <Tag color="cyan">{val}</Tag>,
    },
  ];

  return (
    <Spin spinning={loading}>
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        {/* 1. 财务汇率自定义配置卡片 */}
        <SectionCard
          title="财务汇率自定义规则"
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
                      专用汇率未配置时继承折本币汇率
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
                      最近更新时间：{dayjs(setting.updatedAt).format('YYYY-MM-DD HH:mm:ss')}
                      {setting.updatedBy ? `（操作人：${setting.updatedBy}）` : ''}
                      {setting.version ? ` · 策略版本号: v${setting.version}` : ''}
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

        {/* 2. 汇率继承机制与财务追溯说明 */}
        <SectionCard title="继承规则与财务追溯说明">
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Alert
              type="info"
              showIcon
              icon={<SafetyCertificateOutlined />}
              message="汇率解析与审计追溯保障"
              description={
                <ul style={{ margin: '6px 0 0 0', paddingLeft: 18, lineHeight: '22px' }}>
                  <li>
                    <strong>专用汇率绝对优先</strong>
                    ：系统优先按业务节点的日期和专用汇率类型查询，只要命中专用汇率则始终直接采用。
                  </li>
                  <li>
                    <strong>继承受控查询</strong>
                    ：仅当专用汇率返回“未命中”，且本开关已打开时，才查询同一币种、同一日期、同一收付方向的折本币汇率（BASE_CURRENCY）。
                  </li>
                  <li>
                    <strong>无静默默认值</strong>
                    ：若折本币汇率也未配置，将明确报错提示“汇率未配置”，不会静默使用 1.0 或其他默认值。
                  </li>
                  <li>
                    <strong>冲突严谨暴露</strong>
                    ：若专用汇率存在生效区间冲突，将继续返回冲突错误，绝不通过继承折本币掩盖数据异常。
                  </li>
                  <li>
                    <strong>组织本币直通</strong>
                    ：费用币种若本来就是组织本币，直接采用汇率 1.0，与此开关无关。
                  </li>
                  <li>
                    <strong>完整财务追溯</strong>
                    ：继承结果在账单、发票、收付流水与核销记录中明确标记{' '}
                    <code>exchange_rate_source = INHERITED_BASE_CURRENCY</code> 并记录折本币汇率设置
                    ID，保障财务审计透明追溯。
                  </li>
                </ul>
              }
            />

            <div>
              <Text strong style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>
                支持继承的业务节点与专用汇率类型映射
              </Text>
              <Table
                columns={columns}
                dataSource={inheritanceRulesData}
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
