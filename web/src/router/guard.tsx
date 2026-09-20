import { Result } from 'antd';
import type { ReactNode } from 'react';
import { useAccess } from '@/app/access';
import type { AccessKey } from './routeTypes';

// 路由级权限守卫：未命中的 access 键渲染 403（平移 Umi layout 插件的
// unAccessible 行为）。按钮级权限继续由 useAccess 消费，见 AGENTS.md。
export function AccessGuard({
  accessKey,
  children,
}: {
  accessKey: AccessKey;
  children: ReactNode;
}) {
  const accessState = useAccess();
  return accessState[accessKey] ? (
    children
  ) : (
    <Result status="403" title="403" subTitle="无权访问此页面" />
  );
}
