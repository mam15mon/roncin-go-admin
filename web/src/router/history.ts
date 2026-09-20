import type { createBrowserRouter, Location } from 'react-router';

// react-router v8 导出的 Router 是组件值而非类型，实例类型用返回值推导。
type AppRouter = ReturnType<typeof createBrowserRouter>;

// history shim：平移 Umi 全局 history 单例的实测 API 面（push/replace/location），
// 业务侧 155 处调用零改动。业务仅消费 pathname/search/hash，故 router 尚未
// 绑定（模块顶层加载期）时回退 window.location 是安全的——无 state/key 依赖。
//
// 通过 bindRouter 注入而非直接 import router 实例：history 必须是依赖图的
// 叶子，否则会形成 requestErrorConfig → history → router → guard → access →
// AppProvider → authService → requestClient → requestErrorConfig 的模块求值环，
// 导致 errorConfig 在 TDZ 阶段被读取（测试与生产运行时均会崩溃）。
let routerRef: AppRouter | undefined;

export function bindRouter(router: AppRouter) {
  routerRef = router;
}

export const history = {
  push: (to: string) => {
    if (routerRef) routerRef.navigate(to);
  },
  replace: (to: string) => {
    if (routerRef) routerRef.navigate(to, { replace: true });
  },
  get location(): Location {
    return routerRef
      ? routerRef.state.location
      : (window.location as unknown as Location);
  },
};
