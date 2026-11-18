import {
  AppstoreOutlined,
  ArrowLeftOutlined,
  ArrowRightOutlined,
  CloseCircleOutlined,
  CloseOutlined,
  DatabaseOutlined,
  HomeOutlined,
  InboxOutlined,
  ReloadOutlined,
  SettingOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { Dropdown } from 'antd';
import React, { useEffect, useLayoutEffect, useState } from 'react';
import { useLocation } from 'react-router';
import access from '@/access';
import { useInitialState } from '@/app/AppProvider';
import { useClearListQueries } from '@/components/ui/list-query-state';
import { history } from '@/router/history';
import { clearTabDrafts, getFormDraftScope } from './formDraft';
import {
  isRouteWithoutLayout,
  resolveTabKey,
  resolveWorkspaceTabTitle,
} from './routeUtils';
import { confirmIfTabsDirty } from './tabCloseGuard';

import {
  FIXED_TAB,
  readWorkspaceTabs,
  storeWorkspaceTabs,
  type TagItem,
} from './workspaceTabs';

export { FIXED_TAB, type TagItem } from './workspaceTabs';

/**
 * 判断路径是否无需加入多页签
 */
export function shouldIgnorePath(pathname: string): boolean {
  if (!pathname || pathname === '/') return false;
  return isRouteWithoutLayout(pathname);
}

/**
 * 当关闭页签时，计算下一个激活的路径
 */
export function computeNextActivePath(
  tags: TagItem[],
  closedKey: string,
  currentKey: string,
): string | null {
  if (currentKey !== closedKey) {
    return null;
  }
  const index = tags.findIndex((t) => t.key === closedKey);
  if (index === -1) return null;

  // 优先激活前一个，否则激活后一个，最后退回工作台
  if (index > 0) {
    return tags[index - 1].path;
  }
  if (index + 1 < tags.length) {
    return tags[index + 1].path;
  }
  return FIXED_TAB.path;
}

/**
 * 根据路径匹配 Tab 图标
 */
function getRouteIcon(path: string) {
  if (path === '/welcome' || path === '/') {
    return <HomeOutlined className="roncin-chrome-tab-icon" />;
  }
  if (path.startsWith('/orders')) {
    return <InboxOutlined className="roncin-chrome-tab-icon" />;
  }
  if (path.startsWith('/partners')) {
    return <TeamOutlined className="roncin-chrome-tab-icon" />;
  }
  if (path.startsWith('/master-data')) {
    return <DatabaseOutlined className="roncin-chrome-tab-icon" />;
  }
  if (path.startsWith('/admin')) {
    return <SettingOutlined className="roncin-chrome-tab-icon" />;
  }
  return <AppstoreOutlined className="roncin-chrome-tab-icon" />;
}

/**
 * 现代化 Chrome 风格多页签导航组件 (Chrome Modern Tab)
 */
export const TagsView: React.FC = () => {
  const clearQueries = useClearListQueries();
  const location = useLocation();
  const { initialState } = useInitialState();
  const draftScope = getFormDraftScope(
    initialState?.currentUser?.id,
    initialState?.currentUser?.currentOrganization?.id,
  );
  const currentPath = location.pathname;
  const fullPath = `${location.pathname}${location.search || ''}${location.hash || ''}`;
  const currentTabKey = resolveTabKey(currentPath);

  const [tags, setTags] = useState(() =>
    updateCurrentTab(readWorkspaceTabs(draftScope, access(initialState))),
  );

  function updateCurrentTab(previousTags: TagItem[]): TagItem[] {
    if (
      !currentPath ||
      shouldIgnorePath(currentPath) ||
      currentTabKey === FIXED_TAB.key
    ) {
      return previousTags;
    }
    const existingTag = previousTags.find((tag) => tag.key === currentTabKey);
    const tag: TagItem = {
      key: currentTabKey,
      path: fullPath,
      title:
        existingTag?.path.split(/[?#]/, 1)[0] === currentPath
          ? existingTag.title
          : resolveWorkspaceTabTitle(currentPath),
      closable: true,
    };
    return existingTag
      ? previousTags.map((item) => (item.key === tag.key ? tag : item))
      : [...previousTags, tag];
  }

  useLayoutEffect(() => {
    setTags(updateCurrentTab);
  }, [currentPath, currentTabKey, fullPath]);

  useLayoutEffect(() => {
    storeWorkspaceTabs(draftScope, tags);
  }, [draftScope, tags]);

  // 监听动态更新页签标题事件（如详情页加载完真实单号后更新 Tab 标题）
  useEffect(() => {
    const handleUpdateTabTitle = (
      e: Event & { detail?: { path: string; title: string } },
    ) => {
      if (!e.detail?.path || !e.detail?.title) return;
      const { path, title } = e.detail;
      const eventPathname = path.split('?')[0].split('#')[0];
      const targetKey = resolveTabKey(eventPathname);

      setTags((prev) =>
        prev.map((t) => {
          if (t.key !== targetKey) return t;
          const currentTagPathname = t.path.split('?')[0].split('#')[0];
          if (currentTagPathname === eventPathname) {
            return { ...t, title };
          }
          return t;
        }),
      );
    };

    window.addEventListener(
      'roncin:update-tab-title',
      handleUpdateTabTitle as EventListener,
    );
    return () => {
      window.removeEventListener(
        'roncin:update-tab-title',
        handleUpdateTabTitle as EventListener,
      );
    };
  }, []);

  const handleTabClick = (tag: TagItem) => {
    if (fullPath === tag.path) return;
    history.push(tag.path);
  };

  const requestCloseTags = (
    closingTags: TagItem[],
    nextPath?: string | null,
  ) => {
    const closingKeys = closingTags.map((tag) => tag.key);
    confirmIfTabsDirty(
      closingKeys,
      () => {
        for (const key of closingKeys) clearTabDrafts(key, draftScope);
        clearQueries(closingKeys);
        const remainingTags = tags.filter(
          (tag) => !closingKeys.includes(tag.key),
        );
        storeWorkspaceTabs(draftScope, remainingTags);
        setTags((previous) =>
          previous.filter((tag) => !closingKeys.includes(tag.key)),
        );
        if (nextPath && nextPath !== fullPath) return history.push(nextPath);
        return undefined;
      },
      undefined,
      draftScope,
    );
  };

  const closeTag = (tag: TagItem) => {
    if (tag.closable) {
      requestCloseTags(
        [tag],
        computeNextActivePath(tags, tag.key, currentTabKey),
      );
    }
  };

  const handleClose = (e: React.MouseEvent, tag: TagItem) => {
    e.stopPropagation();
    closeTag(tag);
  };

  // 右键快捷菜单逻辑
  const getContextMenuItems = (tag: TagItem): MenuProps['items'] => {
    const currentIndex = tags.findIndex((t) => t.key === tag.key);
    const hasLeft = currentIndex > 1; // index 0 为固定工作台
    const hasRight = currentIndex < tags.length - 1;
    const hasOther =
      tags.length > 2 || (tags.length === 2 && tag.key === '/welcome');

    return [
      {
        key: 'refresh',
        icon: <ReloadOutlined />,
        label: '重新加载',
        onClick: () => {
          if (fullPath === tag.path) {
            window.location.reload();
          } else {
            history.push(tag.path);
          }
        },
      },
      { type: 'divider' },
      {
        key: 'close-current',
        icon: <CloseOutlined />,
        label: '关闭标签页',
        disabled: !tag.closable,
        onClick: () => closeTag(tag),
      },
      {
        key: 'close-other',
        icon: <CloseCircleOutlined />,
        label: '关闭其他标签页',
        disabled: !hasOther,
        onClick: () =>
          requestCloseTags(
            tags.filter((item) => item.closable && item.key !== tag.key),
            tag.path,
          ),
      },
      {
        key: 'close-all',
        icon: <CloseCircleOutlined />,
        label: '关闭全部标签页',
        disabled: !tags.some((item) => item.closable),
        onClick: () =>
          requestCloseTags(
            tags.filter((item) => item.closable),
            FIXED_TAB.path,
          ),
      },
      {
        key: 'close-right',
        icon: <ArrowRightOutlined />,
        label: '关闭右侧标签页',
        disabled: !hasRight,
        onClick: () => {
          const closing = tags.slice(currentIndex + 1);
          requestCloseTags(
            closing,
            closing.some((item) => item.key === currentTabKey)
              ? tag.path
              : null,
          );
        },
      },
      {
        key: 'close-left',
        icon: <ArrowLeftOutlined />,
        label: '关闭左侧标签页',
        disabled: !hasLeft,
        onClick: () => {
          const closing = tags.slice(1, currentIndex);
          requestCloseTags(
            closing,
            closing.some((item) => item.key === currentTabKey)
              ? tag.path
              : null,
          );
        },
      },
    ];
  };

  return (
    <nav className="roncin-tags-view" aria-label="多页签导航">
      <div className="roncin-tags-view-container" role="tablist">
        {tags.map((tag, index) => {
          const isActive = currentTabKey === tag.key;
          const isNextActive =
            index < tags.length - 1 && currentTabKey === tags[index + 1].key;

          return (
            <Dropdown
              key={tag.key}
              menu={{ items: getContextMenuItems(tag) }}
              trigger={['contextMenu']}
            >
              <div
                role="tab"
                tabIndex={0}
                aria-selected={isActive}
                className={`roncin-chrome-tab ${
                  isActive ? 'roncin-chrome-tab-active' : ''
                }`}
                onClick={() => handleTabClick(tag)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handleTabClick(tag);
                  }
                }}
              >
                {/* 标签主体内容 */}
                {getRouteIcon(tag.key)}
                <span className="roncin-chrome-tab-title">{tag.title}</span>
                {tag.closable && (
                  <button
                    type="button"
                    aria-label={`关闭 ${tag.title}`}
                    className="roncin-chrome-tab-close"
                    onClick={(e) => handleClose(e, tag)}
                    onKeyDown={(e) => e.stopPropagation()}
                  >
                    <CloseOutlined style={{ fontSize: 9 }} />
                  </button>
                )}

                {/* 未激活标签之间的细分割线 */}
                {!isActive && !isNextActive && (
                  <div className="roncin-chrome-tab-divider" />
                )}
              </div>
            </Dropdown>
          );
        })}
      </div>
    </nav>
  );
};
