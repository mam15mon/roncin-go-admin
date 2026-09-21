/** 人员候选仅展示姓名和当前公司内的部门归属，不使用登录账号。 */
export function formatPersonnelLabel(person: {
  displayName?: string;
  departmentNames?: string[];
}): string {
  return `${person.displayName || '未命名人员'} · ${person.departmentNames?.length ? person.departmentNames.join('、') : '公司'}`;
}
