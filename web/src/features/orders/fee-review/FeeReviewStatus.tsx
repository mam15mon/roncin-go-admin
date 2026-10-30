import { Tag } from 'antd';
import React from 'react';

export const feeReviewStatusOptions = [
  { label: '未审核', value: 'UNSUBMITTED' },
  { label: '已审核', value: 'APPROVED' },
];

const colors: Record<string, string> = { APPROVED: 'success' };

/** 订单和费用台账共同展示所属订单的审核事实。 */
export function FeeReviewStatus({
  status,
  pendingSupplement,
  pendingCorrection,
}: {
  status?: string;
  pendingSupplement?: boolean;
  pendingCorrection?: boolean;
}) {
  return (
    <>
      <Tag color={colors[status ?? '']}>
        {feeReviewStatusOptions.find((item) => item.value === status)?.label ??
          '未审核'}
      </Tag>
      {(pendingSupplement || pendingCorrection) && (
        <Tag color="warning">有补录待补审</Tag>
      )}
    </>
  );
}
