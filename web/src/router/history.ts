import type { Location } from 'react-router';
import { router } from './index';

// history shim：平移 Umi 全局 history 单例的实测 API 面（push/replace/location），
// 业务侧 155 处调用零改动。业务仅消费 pathname/search/hash，故 router 尚未创建
// （模块顶层加载期）时回退 window.location 是安全的——无 state/key 依赖。
export const history = {
  push: (to: string) => router.navigate(to),
  replace: (to: string) => router.navigate(to, { replace: true }),
  get location(): Location {
    return router
      ? router.state.location
      : (window.location as unknown as Location);
  },
};
