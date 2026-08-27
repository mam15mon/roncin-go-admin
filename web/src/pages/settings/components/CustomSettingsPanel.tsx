import { DollarOutlined, ReloadOutlined } from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import {
  App,
  Button,
  Col,
  Row,
  Space,
  Spin,
  Switch,
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

  return (
    <Spin spinning={loading}>
      <SectionCard
        title="财务汇率设置"
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
    </Spin>
  );
}

export default CustomSettingsPanel;
