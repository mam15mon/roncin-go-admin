import type access from '@/access';

// routes.ts 集中式路由配置的类型契约（唯一真相源保持 Umi 形状，见任务
// design.md D1）。access 键在此获得编译期校验：后端权限键改名后
// `pnpm --dir web tsc` 直接报错。
export type AccessState = ReturnType<typeof access>;
export type AccessKey = keyof AccessState;

export interface UmiRoute {
  path?: string;
  name?: string;
  icon?: string;
  access?: AccessKey;
  component?: string;
  redirect?: string;
  layout?: boolean;
  hideInMenu?: boolean;
  routes?: UmiRoute[];
}
