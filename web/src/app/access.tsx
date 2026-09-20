import { useMemo } from 'react';
import access from '@/access';
import { useInitialState } from './AppProvider';

// 按钮级与路由级权限统一入口：消费 access.ts 纯函数（权限真相源不变，
// 见 AGENTS.md「前端只消费 auth/me 权限集」约定）。
export function useAccess() {
  const { initialState } = useInitialState();
  const currentUser = initialState?.currentUser;
  return useMemo(() => access({ currentUser }), [currentUser]);
}
