// 角色权限树的勾选联动：勾选操作权限时自动补齐其依赖的基础权限，取消基础
// 权限时级联取消依赖它的权限。依赖关系来自 /admin/permissions 返回的 requires
// 字段，与后端 access.Manifest 是同一份真相。
export function applyPermissionLinkage(
  previousKeys: string[],
  nextKeys: string[],
  requiresByPermission: Record<string, string[]>,
): string[] {
  const previous = new Set(previousKeys);
  const result = new Set(nextKeys);

  // 仅对本次新勾选的键传递补齐依赖；保持勾选的键若依赖被显式取消，不自动回填。
  const expand = (key: string) => {
    for (const required of requiresByPermission[key] ?? []) {
      if (!result.has(required)) {
        result.add(required);
      }
      expand(required);
    }
  };
  for (const key of nextKeys) {
    if (!previous.has(key)) {
      expand(key);
    }
  }

  // 依赖链断裂的权限级联移除，直至集合稳定；每轮删除后重开迭代，避免遍历中
  // 修改集合。清单外的未知键视为无依赖，原样保留。
  let changed = true;
  while (changed) {
    changed = false;
    for (const key of result) {
      if ((requiresByPermission[key] ?? []).some((required) => !result.has(required))) {
        result.delete(key);
        changed = true;
        break;
      }
    }
  }

  return [...result];
}
