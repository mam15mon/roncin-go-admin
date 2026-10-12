/** 审批意见按 Unicode 码点计数，与服务端 rune 校验保持一致。 */
export const feeReviewCommentFieldProps = {
  count: {
    max: 1000,
    show: true,
    strategy: (value: string) => Array.from(value).length,
  },
};

export const feeReviewCommentRule = {
  validator: (_: unknown, value?: string) =>
    Array.from(value ?? '').length <= 1000
      ? Promise.resolve()
      : Promise.reject(new Error('审批意见最多填写 1000 个字符')),
};
