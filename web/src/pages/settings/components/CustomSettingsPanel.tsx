import {
  FileTextOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
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
  settlementServiceGetBilledFeeEditPolicy,
  settlementServiceGetCreditLimitControlPolicy,
  settlementServiceUpdateBilledFeeEditPolicy,
  settlementServiceUpdateCreditLimitControlPolicy,
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
  const { message } = App.useApp();
  const [loadingPolicy, setLoadingPolicy] = useState(false);
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [billedFeePolicy, setBilledFeePolicy] =
    useState<API.BilledFeeEditPolicy>();
  const [canUpdateBilledFeePolicy, setCanUpdateBilledFeePolicy] =
    useState(false);
  const [creditPolicy, setCreditPolicy] =
    useState<API.CreditLimitControlPolicy>();
  const [canUpdateCreditPolicy, setCanUpdateCreditPolicy] = useState(false);
  const [savingCreditPolicy, setSavingCreditPolicy] = useState(false);

  const loadCreditPolicy = useCallback(async () => {
    try {
      const policyRes = await settlementServiceGetCreditLimitControlPolicy({});
      if (!policyRes.data) {
        throw new Error('信用额度管控策略响应不完整');
      }
      setCreditPolicy(policyRes.data);
      setCanUpdateCreditPolicy(Boolean(policyRes.canUpdate));
    } catch (e: any) {
      setCreditPolicy(undefined);
      setCanUpdateCreditPolicy(false);
      message.error(e.message || '读取信用额度管控策略失败');
    }
  }, [message]);

  const loadBilledFeePolicy = useCallback(async () => {
    setLoadingPolicy(true);
    try {
      const policyRes = await settlementServiceGetBilledFeeEditPolicy();
      if (!policyRes.data) {
        throw new Error('自定义设置响应不完整');
      }
      setBilledFeePolicy(policyRes.data);
      setCanUpdateBilledFeePolicy(Boolean(policyRes.canUpdate));
    } catch (e: any) {
      setBilledFeePolicy(undefined);
      setCanUpdateBilledFeePolicy(false);
      message.error(e.message || '当前公司无此设置权限');
    } finally {
      setLoadingPolicy(false);
    }
  }, [message]);

  useEffect(() => {
    void loadBilledFeePolicy();
    void loadCreditPolicy();
  }, [loadBilledFeePolicy, loadCreditPolicy]);

  // 1. 保存账单费用修改总开关
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
      await loadBilledFeePolicy();
    } finally {
      setSavingPolicy(false);
    }
  };

  // 2. 保存可修改字段选择
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
      await loadBilledFeePolicy();
    } finally {
      setSavingPolicy(false);
    }
  };

  // 3. 保存信用额度管控开关（true 仅提醒；false 直接干预拦截）
  const handleToggleCreditPolicy = async (checked: boolean) => {
    setSavingCreditPolicy(true);
    try {
      const response = await settlementServiceUpdateCreditLimitControlPolicy({
        allowSelectionWhenCreditExceeded: checked,
        expectedVersion: creditPolicy?.version ?? '0',
      });
      setCreditPolicy(response.data);
      message.success(
        checked
          ? '已开启：超额往来单位仅提醒，仍可选择与录单'
          : '已关闭：超额往来单位将被置灰禁用并在保存时拦截',
      );
    } catch (e: any) {
      message.error(e.message || '更新信用额度管控策略失败，请刷新重试');
      await loadCreditPolicy();
    } finally {
      setSavingCreditPolicy(false);
    }
  };

  return (
    <Spin spinning={loadingPolicy}>
      <Space vertical size={12} style={{ width: '100%' }}>
        {/* 账单费用修改策略 */}
        <SectionCard
          title="账单费用修改策略"
          extra={
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => void loadBilledFeePolicy()}
              loading={loadingPolicy}
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
                    <FileTextOutlined
                      style={{ fontSize: 16, color: '#1677ff' }}
                    />
                    <Text
                      strong
                      style={{ fontSize: 15, color: 'rgba(0, 0, 0, 0.88)' }}
                    >
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
                      {billedFeePolicy.updatedBy
                        ? `（操作人：${billedFeePolicy.updatedBy}）`
                        : ''}
                    </Text>
                  )}
                </Space>
              </Col>
              <Col xs={24} md={6} style={{ textAlign: 'right' }}>
                <Switch
                  checkedChildren="已开启"
                  unCheckedChildren="已关闭"
                  aria-label="账单费用修改开关"
                  checked={Boolean(billedFeePolicy?.enabled)}
                  loading={savingPolicy}
                  disabled={!canUpdateBilledFeePolicy || loadingPolicy}
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
                    !canUpdateBilledFeePolicy ||
                    savingPolicy ||
                    loadingPolicy
                  }
                  onChange={(values) =>
                    handleChangeEditableFields(values as number[])
                  }
                />
              </div>
            </div>
          </div>
        </SectionCard>

        {/* 往来单位信用额度管控策略 */}
        <SectionCard
          title="往来单位信用额度管控策略"
          extra={
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => void loadCreditPolicy()}
              loading={savingCreditPolicy}
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
                    <SafetyCertificateOutlined
                      style={{ fontSize: 16, color: '#1677ff' }}
                    />
                    <Text
                      strong
                      style={{ fontSize: 15, color: 'rgba(0, 0, 0, 0.88)' }}
                    >
                      往来户超信用额度后是否可以选择
                    </Text>
                    {creditPolicy?.allowSelectionWhenCreditExceeded ? (
                      <Tag color="warning">仅提醒模式</Tag>
                    ) : (
                      <Tag color="error">直接干预模式</Tag>
                    )}
                  </Space>
                  <Paragraph
                    type="secondary"
                    style={{ margin: 0, fontSize: 13, lineHeight: '22px' }}
                  >
                    开启后（默认），当往来单位应收未核销金额（本币）超出信用额度时，系统仅在选择与建账时提供黄色预警提醒，仍允许选择与录单；关闭后，超额往来单位将被直接干预拦截，在下拉选择器中置灰禁用且无法选择该往来户，订单保存与应收费用录入时服务端也会拒绝入库。
                  </Paragraph>
                  {creditPolicy?.updatedAt && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      最近修改时间：{formatDate(creditPolicy.updatedAt)}
                      {creditPolicy.updatedBy
                        ? `（操作人：${creditPolicy.updatedBy}）`
                        : ''}
                    </Text>
                  )}
                </Space>
              </Col>
              <Col xs={24} md={6} style={{ textAlign: 'right' }}>
                <Switch
                  checkedChildren="仅提醒"
                  unCheckedChildren="直接干预"
                  aria-label="信用额度管控开关"
                  checked={Boolean(
                    creditPolicy?.allowSelectionWhenCreditExceeded,
                  )}
                  loading={savingCreditPolicy}
                  disabled={!canUpdateCreditPolicy || savingCreditPolicy}
                  onChange={handleToggleCreditPolicy}
                  style={{ minWidth: 70 }}
                />
              </Col>
            </Row>
          </div>
        </SectionCard>
      </Space>
    </Spin>
  );
}

export default CustomSettingsPanel;
