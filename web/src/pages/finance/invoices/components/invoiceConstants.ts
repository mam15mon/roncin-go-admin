export const invoiceStates: Record<
  string,
  { text: string; color: string }
> = {
  DRAFT: { text: '草稿', color: 'gold' },
  ISSUED: { text: '已开具', color: 'green' },
  CANCELLED: { text: '已作废', color: 'default' },
  RED_FLUSHED: { text: '已红冲', color: 'error' },
};
