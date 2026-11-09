/** 全站人员候选仅展示姓名，不拼接公司、部门、团队或登录账号。 */
export function formatPersonnelLabel(person: { displayName?: string }): string {
  return person.displayName || '未命名人员';
}
