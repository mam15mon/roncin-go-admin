import { Alert } from 'antd';
import React from 'react';

export type BillTermsCreditWarningsProps = {
  direction?: string;
  isCasual?: boolean;
  paymentTermsDays?: number | null;
  /** 信用额度比对信息（折本位币口径），来自服务端预览或伙伴主档结算规则。 */
  creditLimitAmount?: string;
  creditCurrency?: string;
  currentUnsettledAmount?: string;
  isCreditExceeded?: boolean;
  /** 本单金额：建账工作台为叶子总额，草稿编辑为账单总额。 */
  billAmount?: string;
  billCurrency?: string;
};

/**
 * 散客账期与信用超额的统一预警组装：创建工作台与草稿编辑共用同一套文案与判定。
 * 两个预警均为软提示（黄色 Alert），不阻断提交——服务端只在建账预览返回信息，
 * 直接干预拦截仅作用于订单与应收费用写入路径。
 */
export default function BillTermsCreditWarnings({
  direction,
  isCasual,
  paymentTermsDays,
  creditLimitAmount,
  creditCurrency,
  currentUnsettledAmount,
  isCreditExceeded,
  billAmount,
  billCurrency,
}: BillTermsCreditWarningsProps) {
  const hasCasualTermsWarning =
    isCasual === true &&
    direction === 'RECEIVABLE' &&
    typeof paymentTermsDays === 'number' &&
    paymentTermsDays > 0;

  return (
    <>
      {hasCasualTermsWarning ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          description={`该客户为单次合作散客，建议现结；当前已设置 ${paymentTermsDays} 天账期，请注意资金回款风险`}
        />
      ) : null}
      {isCreditExceeded ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          description={`该客户应收未核销余额（本币）${currentUnsettledAmount ?? '-'} 已超出约定信用额度 ${creditLimitAmount ?? '-'}${creditCurrency ? ` ${creditCurrency}` : ''}；本单金额 ${billAmount ?? '-'}${billCurrency ? ` ${billCurrency}` : ''}。录入后请注意资金回款风险。`}
        />
      ) : null}
    </>
  );
}
