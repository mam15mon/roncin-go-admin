import { createBrowserRouter } from 'react-router';
import { buildRouterConfig } from './adaptRoutes';

// 全站唯一 router 实例：模块级导出供 history shim 与 TagsView 等消费。
export const router = createBrowserRouter(buildRouterConfig());
