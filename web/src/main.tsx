import { QueryClientProvider } from '@tanstack/react-query';
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import { HelmetProvider } from 'react-helmet-async';
import { RouterProvider } from 'react-router';
import { createRoot } from 'react-dom/client';
import { themeConfig } from '../config/theme';
import { AppProvider } from './app/AppProvider';
import { router } from './router';
import { AppFeedbackBridge } from './utils/appFeedback';
import { queryClient } from './utils/queryClient';
// Umi 隐式全局入口在此显式化：39KB 全局高密度样式 + Tailwind 基础层。
import './global.less';
import '../tailwind.css';

dayjs.locale('zh-cn');

const container = document.getElementById('root');
if (!container) {
  throw new Error('未找到 #root 挂载节点');
}

// Provider 层级与 Umi 运行时等价：antd App 上下文（appFeedback 桥依赖）→
// React Query → 初始状态（阻塞首帧直至 getInitialState 完成）→ 路由。
createRoot(container).render(
  <HelmetProvider>
    <ConfigProvider locale={zhCN} theme={themeConfig}>
      <AntdApp>
        <AppFeedbackBridge />
        <QueryClientProvider client={queryClient}>
          <AppProvider>
            <RouterProvider router={router} />
          </AppProvider>
        </QueryClientProvider>
      </AntdApp>
    </ConfigProvider>
  </HelmetProvider>,
);
