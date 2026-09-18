import { WarningOutlined } from '@ant-design/icons';
import { Space, Tag } from 'antd';
import React from 'react';
import { OrderCommissionVisibilityMode } from '@/enums.generated';
import { formatAmount } from '@/utils/format';

/**
 * 订单列表【提成】列单元格：渲染服务端按当前用户权限裁剪后的
 * `commission_summary` 可并存事实集合。前端不做任何本地权限猜测：
 * - 摘要缺省（`commission_summary` 未返回）时仅渲染占位符，不渲染空壳；
 * - 本人（EMPLOYEE）视图无任何事实时渲染「我暂无提成」仅针对本人的空态，
 *   不暗示整票或其他员工有无提成；
 * - 组织（ORGANIZATION）视图渲染全员汇总事实；
 * - 「待冲减」是独立风险标识（橙色 + 警示图标），与基础状态并列，不表示已扣回；
 * - 「已发」仅表示对应提成单已发放，不宣称整票结清。
 */

interface CommissionFact {
  key: string;
  label: string;
  count?: number;
  amount?: string;
  color: string;
  risk?: boolean;
}

/** 按服务端契约顺序抽取为 true 的事实；bool 字段按仓库规范用 `=== true` 判定。 */
export function buildCommissionFacts(
  summary: API.OrderCommissionSummary,
): CommissionFact[] {
  const facts: CommissionFact[] = [];
  if (summary.hasExpectedOpportunity === true) {
    facts.push({
      key: 'expected',
      label: '预计',
      count: summary.expectedOpportunityCount,
      color: 'cyan',
    });
  }
  if (summary.hasDraftCommission === true) {
    facts.push({
      key: 'draft',
      label: '待确认',
      count: summary.draftCommissionCount,
      amount: summary.draftCommissionAmount || undefined,
      color: 'default',
    });
  }
  if (summary.hasConfirmedCommission === true) {
    facts.push({
      key: 'confirmed',
      label: '待发',
      count: summary.confirmedCommissionCount,
      amount: summary.confirmedCommissionAmount || undefined,
      color: 'processing',
    });
  }
  if (summary.hasPaidCommission === true) {
    facts.push({
      key: 'paid',
      label: '已发',
      count: summary.paidCommissionCount,
      amount: summary.paidCommissionAmount || undefined,
      color: 'success',
    });
  }
  if (summary.hasPendingDecrease === true) {
    facts.push({
      key: 'decrease',
      label: '待冲减',
      count: summary.pendingDecreaseCount,
      amount: summary.pendingDecreaseAmount || undefined,
      color: 'orange',
      risk: true,
    });
  }
  return facts;
}

/** 金额为订单组织本位币口径；CNY 使用 ¥ 前缀，其他币种显式跟随代码，不合并。 */
export function formatCommissionAmount(
  amount?: string,
  baseCurrency?: string,
): string | undefined {
  if (!amount) return undefined;
  const text = formatAmount(amount);
  if (text === '-') return undefined;
  if (!baseCurrency || baseCurrency === 'CNY') return `¥${text}`;
  return `${text} ${baseCurrency}`;
}

export interface OrderCommissionSummaryCellProps {
  /** 行上的服务端裁剪投影；来自 `rawRecord.commissionSummary`，可能缺省。 */
  summary?: API.OrderCommissionSummary;
  /** 点击事实 Tag 时的下钻回调。 */
  onOpen?: (summary: API.OrderCommissionSummary) => void;
}

export function OrderCommissionSummaryCell({
  summary,
  onOpen,
}: OrderCommissionSummaryCellProps) {
  if (!summary) {
    return <span style={{ color: '#bfbfbf' }}>-</span>;
  }

  const facts = buildCommissionFacts(summary);
  const isEmployeeMode =
    summary.visibilityMode ===
    OrderCommissionVisibilityMode.ORDER_COMMISSION_VISIBILITY_MODE_EMPLOYEE;

  if (facts.length === 0) {
    // 仅本人视图提供「我暂无提成」空态；组织视图与未知视图保持中性占位。
    if (isEmployeeMode) {
      return <Tag color="default">我暂无提成</Tag>;
    }
    return <span style={{ color: '#bfbfbf' }}>-</span>;
  }

  return (
    <Space size={4} wrap>
      {facts.map((fact) => {
        const amountText = formatCommissionAmount(
          fact.amount,
          summary.baseCurrency,
        );
        return (
          <Tag
            key={fact.key}
            color={fact.color}
            icon={fact.risk ? <WarningOutlined /> : undefined}
            style={{ cursor: onOpen ? 'pointer' : undefined }}
            onClick={onOpen ? () => onOpen(summary) : undefined}
          >
            {fact.label} {amountText ?? fact.count ?? 0}
          </Tag>
        );
      })}
    </Space>
  );
}

export default OrderCommissionSummaryCell;
