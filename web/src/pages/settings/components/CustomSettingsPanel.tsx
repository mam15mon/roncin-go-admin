import {
  DollarOutlined,
  FileTextOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import {
  App,
  Button,
  Checkbox,
  Col,
  Divider,
  Row,
  Space,
  Spin,
  Switch,
  Tag,
  Typography,
} from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { SectionCard } from '@/components/ui';
import {
  exchangeRateServiceGetExchangeRateCustomSetting,
  exchangeRateServiceUpdateExchangeRateCustomSetting,
} from '@/services/roncin/exchangeRateService';
import {
  settlementServiceGetBilledFeeEditPolicy,
  settlementServiceUpdateBilledFeeEditPolicy,
} from '@/services/roncin/settlementService';
import { formatDate } from '@/utils/format';

const { Text, Paragraph } = Typography;

/** 账单创建后允许修改费用的字段枚举映射 */
export const BILLED_FEE_EDITABLE_FIELD = {
  FEE_NAME: 1,
  CURRENCY: 2,
  EXCHANGE_RATE: 3,
  QUANTITY: 4,
  UNIT_PRICE: 5,
  TAX_RATE: 6,
} as const;

export const BILLED_FEE_FIELD_OPTIONS = [
  { label: '费用名称', value: BILLED_FEE_EDITABLE_FIELD.FEE_NAME },
  { label: '币种', value: BILLED_FEE_EDITABLE_FIELD.CURRENCY },
  { label: '汇率', value: BILLED_FEE_EDITABLE_FIELD.EXCHANGE_RATE },
  { label: '数量', value: BILLED_FEE_EDITABLE_FIELD.QUANTITY },
  { label: '单价', value: BILLED_FEE_EDITABLE_FIELD.UNIT_PRICE },
  { label: '税率', value: BILLED_FEE_EDITABLE_FIELD.TAX_RATE },
];

export function CustomSettingsPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [savingRate, setSavingRate] = useState(false);
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [rateSetting, setRateSetting] = useState<API.ExchangeRateCustomSetting>();
  const [billedFeePolicy, setBilledFeePolicy] = useState<API.BilledFeeEditPolicy>();

  const loadAllSettings = useCallback(async () => {
    setLoading(true);
    try {
      const [rateRes, policyRes] = await Promise.all([
        exchangeRateServiceGetExchangeRateCustomSetting(),
        settlementServiceGetBilledFeeEditPolicy(),
      ]);
      if (!rateRes.data || !policyRes.data) {
        throw new Error('自定义设置响应不完整');
      }
      setRateSetting(rateRes.data);
      setBilledFeePolicy(policyRes.data);
    } catch (e: any) {
      message.error(e.message || '获取自定义设置失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadAllSettings();
  }, [loadAllSettings]);

  // 1. 保存汇率继承策略
  const handleToggleRateInheritance = async (checked: boolean) => {
    setSavingRate(true);
    try {
      const response = await exchangeRateServiceUpdateExchangeRateCustomSetting({
        inheritBaseCurrencyRate: checked,
        expectedVersion: rateSetting?.version ?? '0',
      });
      setRateSetting(response.data);
      message.success(
        checked
          ? '已开启：专用汇率未配置时继承折本币汇率'
          : '已关闭：专用汇率未配置时继承折本币汇率',
      );
    } catch (e: any) {
      message.error(e.message || '更新汇率设置失败，请刷新重试');
      try {
        const res = await exchangeRateServiceGetExchangeRateCustomSetting();
        if (res.data) setRateSetting(res.data);
      } catch (reloadError: any) {
        message.error(reloadError.message || '重新加载汇率设置失败');
      }
    } finally {
      setSavingRate(false);
    }
  };

  // 2. 保存账单费用修改总开关
  const handleToggleBilledFeePolicy = async (checked: boolean) => {
    setSavingPolicy(true);
    try {
      const response = await settlementServiceUpdateBilledFeeEditPolicy({
        enabled: checked,
        editableFields: billedFeePolicy?.editableFields ?? [],
        expectedVersion: billedFeePolicy?.version ?? '0',
      });
      setBilledFeePolicy(response.data);
      message.success(
        checked
          ? '已开启：账单创建后允许修改费用'
          : '已关闭：账单创建后允许修改费用',
      );
    } catch (e: any) {
      message.error(e.message || '更新账单费用修改策略失败，请刷新重试');
      try {
        const res = await settlementServiceGetBilledFeeEditPolicy();
        if (res.data) setBilledFeePolicy(res.data);
      } catch (reloadError: any) {
        message.error(reloadError.message || '重新加载账单费用策略失败');
      }
    } finally {
      setSavingPolicy(false);
    }
  };

  // 3. 保存可修改字段选择
  const handleChangeEditableFields = async (checkedFields: number[]) => {
    setSavingPolicy(true);
    try {
      const response = await settlementServiceUpdateBilledFeeEditPolicy({
        enabled: billedFeePolicy?.enabled ?? false,
        editableFields: checkedFields,
        expectedVersion: billedFeePolicy?.version ?? '0',
      });
      setBilledFeePolicy(response.data);
      message.success('已更新允许修改的费用字段');
    } catch (e: any) {
      message.error(e.message || '更新可修改字段失败，请刷新重试');
      try {
        const res = await settlementServiceGetBilledFeeEditPolicy();
        if (res.data) setBilledFeePolicy(res.data);
      } catch (reloadError: any) {
        message.error(reloadError.message || '重新加载账单费用策略失败');
      }
    } finally {
      setSavingPolicy(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Space vertical size={12} style={{ width: '100%' }}>
        {/* 1. 财务汇率设置 */}
        <SectionCard
          title="财务汇率设置"
          extra={
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={loadAllSettings}
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
                <Space vertical size={6} style={{ width: '100%' }}>
                  <Space align="center" size={8} wrap>
                    <DollarOutlined style={{ fontSize: 16, color: '#1677ff' }} />
                    <Text strong style={{ fontSize: 15, color: 'rgba(0, 0, 0, 0.88)' }}>
                      专用汇率未配置时继承折本币汇率
                    </Text>
                    {rateSetting?.inheritBaseCurrencyRate ? (
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
                  {rateSetting?.updatedAt && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      最近修改时间：{formatDate(rateSetting.updatedAt)}
                      {rateSetting.updatedBy ? `（操作人：${rateSetting.updatedBy}）` : ''}
                    </Text>
                  )}
                </Space>
              </Col>
              <Col xs={24} md={6} style={{ textAlign: 'right' }}>
                <Switch
                  checkedChildren="已开启"
                  unCheckedChildren="已关闭"
                  checked={Boolean(rateSetting?.inheritBaseCurrencyRate)}
                  loading={savingRate}
                  disabled={!access.canUpdateExchangeRates || loading}
                  onChange={handleToggleRateInheritance}
                  style={{ minWidth: 70 }}
                />
              </Col>
            </Row>
          </div>
        </SectionCard>

        {/* 2. 账单费用修改策略 */}
        <SectionCard title="账单费用修改策略">
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
                <Space vertical size={6} style={{ width: '100%' }}>
                  <Space align="center" size={8} wrap>
                    <FileTextOutlined style={{ fontSize: 16, color: '#1677ff' }} />
                    <Text strong style={{ fontSize: 15, color: 'rgba(0, 0, 0, 0.88)' }}>
                      账单创建后允许修改费用
                    </Text>
                    {billedFeePolicy?.enabled ? (
                      <Tag color="success">已开启修改</Tag>
                    ) : (
                      <Tag>默认关闭</Tag>
                    )}
                  </Space>
                  <Paragraph
                    type="secondary"
                    style={{ margin: 0, fontSize: 13, lineHeight: '22px' }}
                  >
                    开启后，当费用所属账单仍处于「草稿」状态时，允许按下方勾选的字段对已建账单费用进行修改，并同步更新账单快照与总金额。已确认或锁定的账单不可修改。
                  </Paragraph>
                  {billedFeePolicy?.updatedAt && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      最近修改时间：{formatDate(billedFeePolicy.updatedAt)}
                      {billedFeePolicy.updatedBy ? `（操作人：${billedFeePolicy.updatedBy}）` : ''}
                    </Text>
                  )}
                </Space>
              </Col>
              <Col xs={24} md={6} style={{ textAlign: 'right' }}>
                <Switch
                  checkedChildren="已开启"
                  unCheckedChildren="已关闭"
                  checked={Boolean(billedFeePolicy?.enabled)}
                  loading={savingPolicy}
                  disabled={!access.canUpdateFinanceBills || loading}
                  onChange={handleToggleBilledFeePolicy}
                  style={{ minWidth: 70 }}
                />
              </Col>
            </Row>

            <Divider style={{ margin: '14px 0 12px 0' }} />

            <div>
              <Space align="center" style={{ marginBottom: 8 }}>
                <Text strong style={{ fontSize: 13 }}>
                  允许修改的字段范围：
                </Text>
                {!billedFeePolicy?.enabled && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    （需先开启总开关）
                  </Text>
                )}
              </Space>
              <div>
                <Checkbox.Group
                  options={BILLED_FEE_FIELD_OPTIONS}
                  value={billedFeePolicy?.editableFields ?? []}
                  disabled={
                    !billedFeePolicy?.enabled ||
                    !access.canUpdateFinanceBills ||
                    savingPolicy ||
                    loading
                  }
                  onChange={(values) => handleChangeEditableFields(values as number[])}
                />
              </div>
            </div>
          </div>
        </SectionCard>
      </Space>
    </Spin>
  );
}

export default CustomSettingsPanel;
