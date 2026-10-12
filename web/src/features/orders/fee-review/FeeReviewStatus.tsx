import { Tag } from 'antd';
import React from 'react';

export const feeReviewStatusOptions = [
  { label: '未提交', value: 'UNSUBMITTED' },
  { label: '待审核', value: 'PENDING' },
  { label: '已批准', value: 'APPROVED' },
  { label: '已退回', value: 'REJECTED' },
  { label: '已撤销批准', value: 'REVOKED' },
];

const colors: Record<string, string> = {
  PENDING: 'processing',
  APPROVED: 'success',
  REJECTED: 'error',
  REVOKED: 'warning',
};

/** 订单和费用台账共同展示所属订单的审核事实。 */
export function FeeReviewStatus({
  status,
  pendingSupplement,
}: {
  status?: string;
  pendingSupplement?: boolean;
}) {
  return (
    <>
      <Tag color={colors[status ?? '']}>
        {feeReviewStatusOptions.find((item) => item.value === status)?.label ??
          '未提交'}
      </Tag>
      {pendingSupplement && <Tag color="warning">有补录待审核</Tag>}
    </>
  );
}
