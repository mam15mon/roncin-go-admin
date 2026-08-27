/**
 * 全局统一智能下拉搜索过滤函数
 * 支持按 label、value、code、name、title、keywords 多维度不区分大小写匹配
 */
export const defaultSelectFilterOption = (
  input: string,
  option?: Record<string, any>,
): boolean => {
  if (!input?.trim()) return true;
  if (!option) return false;

  const target = input.trim().toLowerCase();

  // 1. 检查 label 字段
  if (option.label !== undefined && option.label !== null) {
    const labelStr =
      typeof option.label === 'string' ? option.label : String(option.label);
    if (labelStr.toLowerCase().includes(target)) return true;
  }

  // 2. 检查 value 字段
  if (option.value !== undefined && option.value !== null) {
    const valueStr = String(option.value);
    if (valueStr.toLowerCase().includes(target)) return true;
  }

  // 3. 检查附加搜索属性（code、name、title、keywords 等）
  const extraFields = ['code', 'name', 'title', 'key', 'search', 'keywords'];
  for (const field of extraFields) {
    if (option[field] !== undefined && option[field] !== null) {
      if (String(option[field]).toLowerCase().includes(target)) return true;
    }
  }

  // 4. 若为 JSX/字符串类型的 children
  if (
    typeof option.children === 'string' &&
    option.children.toLowerCase().includes(target)
  ) {
    return true;
  }

  return false;
};
