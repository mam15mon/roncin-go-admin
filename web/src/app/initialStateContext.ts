import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import { createContext, type Dispatch, type SetStateAction } from 'react';

export interface InitialState {
  settings?: Partial<LayoutSettings>;
  currentUser?: API.CurrentUser;
  fetchUserInfo?: () => Promise<API.CurrentUser | undefined>;
}

export interface InitialStateModel {
  initialState?: InitialState;
  setInitialState: Dispatch<SetStateAction<InitialState | undefined>>;
  refresh: () => void;
  loading: boolean;
}

/** 认证状态唯一上下文；独立定义让通用展示组件只读取已有状态。 */
export const appInitialStateContext = createContext<
  InitialStateModel | undefined
>(undefined);
