import { createBrowserRouter } from 'react-router';
import { buildRouterConfig } from './adaptRoutes';
import { bindRouter } from './history';

// 全站唯一 router 实例；创建后立即绑定到 history shim，供 155 处
// history.push/replace/location 调用与 requestErrorConfig 的 401 重定向消费。
export const router = createBrowserRouter(buildRouterConfig());
bindRouter(router);
