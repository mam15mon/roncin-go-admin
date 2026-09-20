import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import {
  createContext,
  useContext,
  useEffect,
  useState,
} from 'react';
import type { ReactNode } from 'react';
import { history } from '@/router/history';
import { getRequestErrorStatus } from '@/requestErrorConfig';
import { authServiceMe } from '@/services/roncin/authService';
import { DEV_MOCK_USER, isDevMockEnabled } from '@/utils/devMockUser';
import defaultSettings from '../../config/defaultSettings';

export interface InitialState {
  settings?: Partial<LayoutSettings>;
  currentUser?: API.CurrentUser;
  fetchUserInfo?: () => Promise<API.CurrentUser | undefined>;
}

export const LOGIN_PATH = '/user/login';
export const PUBLIC_AUTH_PATHS = new Set([
  LOGIN_PATH,
  '/user/register',
  '/user/login/wecom/callback',
  '/user/login/dingtalk/callback',
]);

const layoutSettings = defaultSettings as Partial<LayoutSettings>;

// 初始状态逻辑自 src/app.tsx 的 getInitialState 原样平移（含 devMock 兜底、
// 公开路径短路与 401 重定向）；差异仅在执行时机：由「渲染前插件钩子」变为
// Provider 首帧前的阻塞 effect。
async function getInitialState(): Promise<InitialState> {
  const fetchUserInfo = async () => {
    try {
      const response = await authServiceMe({ skipErrorHandler: true });
      return response.data;
    } catch (error) {
      if (isDevMockEnabled()) {
        return DEV_MOCK_USER;
      }
      throw error;
    }
  };

  if (PUBLIC_AUTH_PATHS.has(history.location.pathname)) {
    if (isDevMockEnabled()) {
      return {
        fetchUserInfo,
        currentUser: DEV_MOCK_USER,
        settings: layoutSettings,
      };
    }
    return { fetchUserInfo, settings: layoutSettings };
  }

  try {
    const user = await fetchUserInfo();
    return {
      fetchUserInfo,
      currentUser: user,
      settings: layoutSettings,
    };
  } catch (error) {
    if (isDevMockEnabled()) {
      return {
        fetchUserInfo,
        currentUser: DEV_MOCK_USER,
        settings: layoutSettings,
      };
    }
    if (getRequestErrorStatus(error) === 401) {
      const { pathname, search, hash } = history.location;
      history.replace(
        `${LOGIN_PATH}?redirect=${encodeURIComponent(pathname + search + hash)}`,
      );
      return { fetchUserInfo, settings: layoutSettings };
    }
    throw error;
  }
}

const initialStateContext = createContext<InitialState | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  const [initialState, setInitialState] = useState<InitialState | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let alive = true;
    getInitialState()
      .then((state) => {
        if (alive) setInitialState(state);
      })
      .catch((error) => {
        // 非预期初始化失败（如网络不可达）：落到未登录态由布局守卫接管跳转
        // 登录页，错误保留在控制台便于排查，不中断整页渲染。
        console.error('[AppProvider] 初始状态获取失败', error);
      })
      .finally(() => {
        if (alive) setReady(true);
      });
    return () => {
      alive = false;
    };
  }, []);

  // 阻塞首帧直至初始状态就绪（对齐 Umi 在渲染前执行 getInitialState 的语义）。
  if (!ready) return null;

  return (
    <initialStateContext.Provider value={initialState}>
      {children}
    </initialStateContext.Provider>
  );
}

export function useInitialState(): InitialState {
  const value = useContext(initialStateContext);
  if (!value) {
    throw new Error('useInitialState 必须在 AppProvider 内使用');
  }
  return value;
}
