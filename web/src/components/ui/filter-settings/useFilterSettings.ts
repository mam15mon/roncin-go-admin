import { App } from 'antd';
import { useContext, useMemo, useRef, useState } from 'react';
import { appInitialStateContext } from '@/app/initialStateContext';
import {
  clearFilterSettingsPreference,
  isFilterSettingsScopeReady,
  loadFilterSettingsPreference,
  resolveFilterVisibleNames,
  saveFilterSettingsPreference,
} from './preference';
import type {
  FilterSettingsField,
  FilterSettingsPreference,
  FilterSettingsScope,
} from './types';

export interface UseFilterSettingsOptions {
  /** 页面标识，承担「页面」隔离维度。 */
  pageKey: string;
  fields: readonly FilterSettingsField[];
  /** 缺省取认证用户与当前公司。 */
  scope?: FilterSettingsScope;
}

export interface UseFilterSettingsResult {
  visibleNames: string[];
  setVisible: (names: readonly string[]) => void;
  reset: () => void;
  /** 最近一次写入本地存储是否失败（设置仍对当前页面生效）。 */
  persistFailed: boolean;
}

/** 筛选项显隐设置：按用户+公司+页面保存在本浏览器，写入失败提示一次且当次仍生效。 */
export function useFilterSettings({
  pageKey,
  fields,
  scope: explicitScope,
}: UseFilterSettingsOptions): UseFilterSettingsResult {
  const { message } = App.useApp();
  const initialState = useContext(appInitialStateContext)?.initialState;
  const scope: FilterSettingsScope = explicitScope ?? {
    userId: initialState?.currentUser?.id,
    organizationId: initialState?.currentUser?.currentOrganization?.id,
  };
  const identity = JSON.stringify([
    pageKey,
    scope.userId ?? null,
    scope.organizationId ?? null,
  ]);
  const [local, setLocal] = useState<{
    identity: string;
    value: FilterSettingsPreference | null;
  }>(() => ({ identity, value: loadFilterSettingsPreference(pageKey, scope) }));
  const [failedIdentity, setFailedIdentity] = useState<string>();
  const notifiedRef = useRef<string | null>(null);

  // 切换作用域的当前渲染直接读取新偏好，不暴露旧用户/公司的配置。
  const stored =
    local.identity === identity
      ? local.value
      : loadFilterSettingsPreference(pageKey, scope);
  const resolved = resolveFilterVisibleNames(fields, stored);
  const signature = resolved.join('\u0000');
  // biome-ignore lint/correctness/useExhaustiveDependencies: signature 即 resolved 的稳定标识
  const visibleNames = useMemo(() => resolved, [signature]);

  const report = (persisted: boolean) => {
    if (persisted) {
      notifiedRef.current = null;
      setFailedIdentity(undefined);
      return;
    }
    setFailedIdentity(identity);
    if (notifiedRef.current !== identity) {
      notifiedRef.current = identity;
      message.error('筛选设置保存到本地浏览器失败，本次设置仅当前页面生效');
    }
  };

  const setVisible = (names: readonly string[]) => {
    const wanted = new Set(names);
    const value: FilterSettingsPreference = {
      visible: fields
        .filter((f) => f.required || wanted.has(f.name))
        .map((f) => f.name),
      hidden: fields
        .filter((f) => !f.required && !wanted.has(f.name))
        .map((f) => f.name),
    };
    setLocal({ identity, value });
    // 身份未就绪只在内存生效，不读不写也不提示。
    if (isFilterSettingsScopeReady(scope)) {
      report(saveFilterSettingsPreference(pageKey, scope, value));
    }
  };

  const reset = () => {
    setLocal({ identity, value: null });
    if (isFilterSettingsScopeReady(scope)) {
      report(clearFilterSettingsPreference(pageKey, scope));
    }
  };

  return {
    visibleNames,
    setVisible,
    reset,
    persistFailed: failedIdentity === identity,
  };
}
