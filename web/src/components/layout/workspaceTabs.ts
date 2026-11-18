import type { AccessState } from '@/router/routeTypes';
import { clearWorkspaceDrafts } from './formDraft';
import {
  canRestoreWorkspaceTab,
  resolveTabKey,
  resolveWorkspaceTabTitle,
} from './routeUtils';

export interface TagItem {
  key: string;
  path: string;
  title: string;
  closable: boolean;
}

export const FIXED_TAB: TagItem = {
  key: '/welcome',
  path: '/welcome',
  title: '工作台',
  closable: false,
};

const tabStorageKey = (scope: string) => `roncin:open-tabs:${scope}`;

function readPaths(scope?: string): string[] {
  if (!scope) return [];
  try {
    const paths: unknown = JSON.parse(
      sessionStorage.getItem(tabStorageKey(scope)) ?? '[]',
    );
    return Array.isArray(paths)
      ? paths.filter((path): path is string => typeof path === 'string')
      : [];
  } catch {
    return [];
  }
}

export function readWorkspaceTabs(
  scope: string | undefined,
  access: AccessState,
): TagItem[] {
  const tabs = new Map([[FIXED_TAB.key, FIXED_TAB]]);
  for (const path of readPaths(scope)) {
    if (!canRestoreWorkspaceTab(path, access)) continue;
    const key = resolveTabKey(path);
    if (key !== FIXED_TAB.key) {
      tabs.set(key, {
        key,
        path,
        title: resolveWorkspaceTabTitle(path),
        closable: true,
      });
    }
  }
  return [...tabs.values()];
}

export function getStoredWorkspaceTabKeys(scope?: string): string[] {
  return readPaths(scope).map(resolveTabKey);
}

export function storeWorkspaceTabs(
  scope: string | undefined,
  tags: TagItem[],
): void {
  if (!scope) return;
  try {
    sessionStorage.setItem(
      tabStorageKey(scope),
      JSON.stringify(tags.filter((tag) => tag.closable).map((tag) => tag.path)),
    );
  } catch {
    // 浏览器禁止存储时，页签和关闭确认仍可使用实时状态。
  }
}

export function closeWorkspaceTabs(scope?: string): void {
  if (!scope) return;
  clearWorkspaceDrafts(scope);
  try {
    sessionStorage.removeItem(tabStorageKey(scope));
  } catch {
    // 浏览器禁止存储。
  }
}
