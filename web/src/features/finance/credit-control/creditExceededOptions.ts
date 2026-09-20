import type { SelectOption } from '@/types/select-option';

/** 信用干预候选约束：契约字段标记已超出信用额度的客户候选（仅客户方向有意义）。 */
export type CreditCandidateOption = SelectOption & { creditExceeded?: boolean };

/**
 * 直接干预模式下把超额客户候选项置灰禁用；
 * 仅提醒模式原样返回，标签与禁用状态都经契约字段渲染，不污染 label。
 */
export function disableCreditExceededOptions<T extends CreditCandidateOption>(
  options: T[],
  interventionActive: boolean,
): T[] {
  if (!interventionActive) {
    return options;
  }
  return options.map((option) =>
    option.creditExceeded ? { ...option, disabled: true } : option,
  );
}
