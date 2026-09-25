/** 费用台账 7 类财务进度行的默认背景高亮颜色。 */
export function getDefaultRowColors(): {
  unbilled: string;
  unverifiedUninvoiced: string;
  invoicedUnverified: string;
  verifiedUninvoiced: string;
  completed: string;
  invoicedPartiallyVerified: string;
  partiallyVerifiedUninvoiced: string;
} {
  return {
    unbilled: '#FFF7E6',
    unverifiedUninvoiced: '#FFFBE6',
    invoicedUnverified: '#E6F4FF',
    verifiedUninvoiced: '#F9F0FF',
    completed: '#F6FFED',
    invoicedPartiallyVerified: '#E6F4FF',
    partiallyVerifiedUninvoiced: '#F9F0FF',
  };
}
