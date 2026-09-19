import { RightOutlined } from '@ant-design/icons';
import { Button, Descriptions, Modal, Space, Tag, Typography } from 'antd';
import React from 'react';
import { MODAL_SIZE } from '@/components/ui';
import { OrderCommissionVisibilityMode } from '@/enums.generated';
import {
  buildCommissionFacts,
  formatCommissionAmount,
} from './OrderCommissionSummaryCell';

const { Text } = Typography;

/**
 * 订单列表提成摘要下钻弹窗：只展示该行 `commission_summary` 服务端已裁剪的
 * 聚合事实（状态、数量、金额），不做本地权限猜测，也不在列表页新造请求接口。
 * 语义铁律：
 * - 「已发」仅表示对应提成单已发放，不宣称整票或全部回款结清；
 * - 「待冲减」是待处理的冲减建议，不代表已实际扣回；
 * - 「预计」是尚未生成有效提成单的估算机会，非应发承诺；
 * - 金额为订单组织本位币口径，不同币种不合并。
 * 更多明细（按员工、来源单）前往现有提成台账页进一步查看。
 */

/** 提成摘要可见模式 → 页面文案；未知值不猜测，显示中性说明。 */
function visibilityMeta(mode?: number): {
  label: string;
  color: string;
  desc: string;
} {
  if (
    mode ===
    OrderCommissionVisibilityMode.ORDER_COMMISSION_VISIBILITY_MODE_EMPLOYEE
  ) {
    return {
      label: '本人视图',
      color: 'blue',
      desc: '仅显示本人在该票订单上的提成事实，不代表其他员工或整票情况。',
    };
  }
  if (
    mode ===
    OrderCommissionVisibilityMode.ORDER_COMMISSION_VISIBILITY_MODE_ORGANIZATION
  ) {
    return {
      label: '组织全员视图',
      color: 'geekblue',
      desc: '显示该票订单当前组织可见范围内的全员提成汇总事实。',
    };
  }
  return {
    label: '未知视图',
    color: 'default',
    desc: '服务端未返回明确的可见范围标识。',
  };
}

/** 单条事实的展示文案；预计机会无金额字段，仅展示数量。 */
function factDescription(
  fact: {
    amount?: string;
    count?: number;
  },
  baseCurrency?: string,
): string {
  const amountText = formatCommissionAmount(fact.amount, baseCurrency);
  const countText = `${fact.count ?? 0} 笔`;
  return amountText ? `${countText} · ${amountText}` : countText;
}

export interface OrderCommissionSummaryModalProps {
  open: boolean;
  orderNo?: string;
  summary?: API.OrderCommissionSummary;
  /** 是否允许跳转现有提成台账（复用路由级 access 判定结果）。 */
  canOpenLedger?: boolean;
  onClose: () => void;
  onOpenLedger?: () => void;
}

export function OrderCommissionSummaryModal({
  open,
  orderNo,
  summary,
  canOpenLedger = false,
  onClose,
  onOpenLedger,
}: OrderCommissionSummaryModalProps) {
  if (!open) {
    return null;
  }

  const meta = visibilityMeta(summary?.visibilityMode);
  const facts = summary ? buildCommissionFacts(summary) : [];
  const isEmployeeMode =
    summary?.visibilityMode ===
    OrderCommissionVisibilityMode.ORDER_COMMISSION_VISIBILITY_MODE_EMPLOYEE;
  const hasAmountCurrency = Boolean(summary?.baseCurrency);

  return (
    <Modal
      title={`提成摘要${orderNo ? ` · ${orderNo}` : ''}`}
      open={open}
      onCancel={onClose}
      footer={
        <Button type="primary" onClick={onClose}>
          关闭
        </Button>
      }
      width={MODAL_SIZE.SM}
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <div>
          <Space size={8} wrap>
            <Tag color={meta.color}>{meta.label}</Tag>
            {hasAmountCurrency && (
              <Text type="secondary">
                金额为订单组织本位币（{summary?.baseCurrency}
                ）口径，不同币种不合并汇总
              </Text>
            )}
          </Space>
          <div>
            <Text type="secondary">{meta.desc}</Text>
          </div>
        </div>

        {facts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            {isEmployeeMode ? (
              <>
                <Tag color="default">我暂无提成</Tag>
                <div style={{ marginTop: 8 }}>
                  <Text type="secondary">
                    本人在该票订单上暂无可显示的提成或预计可计提机会。
                  </Text>
                </div>
              </>
            ) : (
              <Text type="secondary">当前可见范围内暂无提成事实。</Text>
            )}
          </div>
        ) : (
          <Descriptions
            size="small"
            column={1}
            bordered
            styles={{ label: { width: 132 } }}
          >
            {facts.map((fact) => (
              <Descriptions.Item
                key={fact.key}
                label={
                  fact.risk ? <Tag color="orange">待冲减</Tag> : fact.label
                }
              >
                <Space size={8} wrap>
                  <span>{factDescription(fact, summary?.baseCurrency)}</span>
                  {fact.key === 'expected' && (
                    <Text type="secondary">
                      尚未生成有效提成单的估算机会，非应发承诺
                    </Text>
                  )}
                  {fact.key === 'draft' && (
                    <Text type="secondary">待财务确认，尚未发放</Text>
                  )}
                  {fact.key === 'confirmed' && (
                    <Text type="secondary">已确认待发放</Text>
                  )}
                  {fact.key === 'paid' && (
                    <Text type="secondary">
                      仅表示对应提成单已发放，不代表整票结清
                    </Text>
                  )}
                  {fact.key === 'decrease' && (
                    <Text type="secondary">冲减建议待处理，尚未实际扣回</Text>
                  )}
                </Space>
              </Descriptions.Item>
            ))}
          </Descriptions>
        )}

        {canOpenLedger && onOpenLedger && (
          <div>
            <Button type="link" size="small" onClick={onOpenLedger}>
              前往提成台账查看明细 <RightOutlined />
            </Button>
            <div>
              <Text type="secondary">
                如需按员工、来源单进一步核对，请在提成台账中查询。
              </Text>
            </div>
          </div>
        )}
      </Space>
    </Modal>
  );
}

export default OrderCommissionSummaryModal;
