import type access from '@/access';

// routes.ts 集中式路由配置的类型契约（唯一真相源保持 Umi 形状，见任务
// design.md D1）。access 键在此获得编译期校验：后端权限键改名后
// `pnpm --dir web tsc` 直接报错。
export type AccessState = ReturnType<typeof access>;
export type AccessKey = keyof AccessState;

export interface UmiRoute {
  path?: string;
  name?: string;
  /** 菜单短名与页面标题不同时显式声明；动态标题只解释路由参数。 */
  pageTitle?:
    | string
    | ((params: Readonly<Record<string, string | undefined>>) => string);
  icon?: string;
  access?: AccessKey;
  component?: string;
  redirect?: string;
  layout?: boolean;
  hideInMenu?: boolean;
  routes?: UmiRoute[];
  /** 组织切换只保留功能位置，记录页显式声明其所属列表。 */
  organizationSwitch?: { keep: true } | { list: string };
}
