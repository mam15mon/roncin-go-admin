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
import { history, useLocation } from '@umijs/max';
import type { MenuProps } from 'antd';
import { Dropdown } from 'antd';
import React, { useEffect, useState } from 'react';
import { EllipsisTooltip } from '@/components/ui';
import { resolveRouteTitle, resolveTabKey } from './routeUtils';

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

const IGNORED_PREFIXES = ['/user/login', '/user', '/login'];

/**
 * 判断路径是否无需加入多页签
 */
export function shouldIgnorePath(pathname: string): boolean {
  if (!pathname || pathname === '/') return false;
  return IGNORED_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  );
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
  const location = useLocation();
  const currentPath = location.pathname;
  const fullPath = `${location.pathname}${location.search || ''}${location.hash || ''}`;
  const currentTabKey = resolveTabKey(currentPath);

  const [tags, setTags] = useState<TagItem[]>(() => {
    if (
      currentPath &&
      currentPath !== '/' &&
      currentPath !== '/welcome' &&
      !shouldIgnorePath(currentPath)
    ) {
      const initialKey = resolveTabKey(currentPath);
      if (initialKey === '/welcome') {
        return [FIXED_TAB];
      }
      return [
        FIXED_TAB,
        {
          key: initialKey,
          path: fullPath,
          title: resolveRouteTitle(currentPath),
          closable: true,
        },
      ];
    }
    return [FIXED_TAB];
  });

  useEffect(() => {
    if (
      shouldIgnorePath(currentPath) ||
      currentPath === '/' ||
      currentPath === '/welcome'
    ) {
      return;
    }

    const tabKey = resolveTabKey(currentPath);
    if (tabKey === '/welcome') {
      return;
    }

    setTags((prevTags) => {
      const index = prevTags.findIndex((t) => t.key === tabKey);
      if (index !== -1) {
        const prevTag = prevTags[index];
        const prevPathname = prevTag.path.split('?')[0].split('#')[0];
        const title =
          prevPathname === currentPath
            ? prevTag.title
            : resolveRouteTitle(currentPath);

        const updated = [...prevTags];
        updated[index] = {
          ...prevTag,
          path: fullPath,
          title,
        };
        return updated;
      }

      return [
        ...prevTags,
        {
          key: tabKey,
          path: fullPath,
          title: resolveRouteTitle(currentPath),
          closable: true,
        },
      ];
    });
  }, [currentPath, fullPath]);

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

  const handleClose = (e: React.MouseEvent, tag: TagItem) => {
    e.stopPropagation();
    if (!tag.closable) return;

    const nextPath = computeNextActivePath(tags, tag.key, currentTabKey);
    setTags((prev) => prev.filter((t) => t.key !== tag.key));

    if (nextPath && nextPath !== fullPath) {
      history.push(nextPath);
    }
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
        onClick: () => {
          const nextPath = computeNextActivePath(tags, tag.key, currentTabKey);
          setTags((prev) => prev.filter((t) => t.key !== tag.key));
          if (nextPath && nextPath !== fullPath) {
            history.push(nextPath);
          }
        },
      },
      {
        key: 'close-other',
        icon: <CloseCircleOutlined />,
        label: '关闭其他标签页',
        disabled: !hasOther,
        onClick: () => {
          if (tag.key === '/welcome') {
            setTags([FIXED_TAB]);
            history.push('/welcome');
          } else {
            setTags([FIXED_TAB, tag]);
            history.push(tag.path);
          }
        },
      },
      {
        key: 'close-right',
        icon: <ArrowRightOutlined />,
        label: '关闭右侧标签页',
        disabled: !hasRight,
        onClick: () => {
          setTags((prev) => prev.slice(0, currentIndex + 1));
          const activeIndex = tags.findIndex((t) => t.key === currentTabKey);
          if (currentIndex < activeIndex) {
            history.push(tag.path);
          }
        },
      },
      {
        key: 'close-left',
        icon: <ArrowLeftOutlined />,
        label: '关闭左侧标签页',
        disabled: !hasLeft,
        onClick: () => {
          setTags((prev) => [FIXED_TAB, ...prev.slice(currentIndex)]);
          const activeIndex = tags.findIndex((t) => t.key === currentTabKey);
          if (activeIndex > 0 && activeIndex < currentIndex) {
            history.push(tag.path);
          }
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
            index < tags.length - 1 &&
            currentTabKey === tags[index + 1].key;

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
                <span className="roncin-chrome-tab-title">
                  <EllipsisTooltip autoDetect maxWidth="100%">
                    {tag.title}
                  </EllipsisTooltip>
                </span>
                {tag.closable && (
                  <button
                    type="button"
                    aria-label={`关闭 ${tag.title}`}
                    className="roncin-chrome-tab-close"
                    onClick={(e) => handleClose(e, tag)}
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
