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
import { resolveRouteTitle } from './routeUtils';

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
  currentPath: string,
): string | null {
  if (currentPath !== closedKey) {
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
 * 现代化 Chrome 风格多页签导航组件
 */
export const TagsView: React.FC = () => {
  const location = useLocation();
  const currentPath = location.pathname;

  const [tags, setTags] = useState<TagItem[]>(() => {
    if (
      currentPath &&
      currentPath !== '/' &&
      currentPath !== '/welcome' &&
      !shouldIgnorePath(currentPath)
    ) {
      return [
        FIXED_TAB,
        {
          key: currentPath,
          path: currentPath,
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

    setTags((prevTags) => {
      const exists = prevTags.some((t) => t.key === currentPath);
      if (exists) return prevTags;
      return [
        ...prevTags,
        {
          key: currentPath,
          path: currentPath,
          title: resolveRouteTitle(currentPath),
          closable: true,
        },
      ];
    });
  }, [currentPath]);

  const handleTabClick = (tag: TagItem) => {
    if (currentPath === tag.path) return;
    history.push(tag.path);
  };

  const handleClose = (e: React.MouseEvent, tag: TagItem) => {
    e.stopPropagation();
    if (!tag.closable) return;

    const nextPath = computeNextActivePath(tags, tag.key, currentPath);
    setTags((prev) => prev.filter((t) => t.key !== tag.key));

    if (nextPath && nextPath !== currentPath) {
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
          if (currentPath === tag.path) {
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
          const nextPath = computeNextActivePath(tags, tag.key, currentPath);
          setTags((prev) => prev.filter((t) => t.key !== tag.key));
          if (nextPath && nextPath !== currentPath) {
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
          if (currentIndex < tags.findIndex((t) => t.key === currentPath)) {
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
          if (tags.findIndex((t) => t.key === currentPath) < currentIndex) {
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
          const isActive =
            tag.path === '/welcome'
              ? currentPath === '/welcome' || currentPath === '/'
              : currentPath === tag.path ||
                currentPath.startsWith(`${tag.path}/`);

          const isNextActive =
            index < tags.length - 1 &&
            (tags[index + 1].path === '/welcome'
              ? currentPath === '/welcome' || currentPath === '/'
              : currentPath === tags[index + 1].path ||
                currentPath.startsWith(`${tags[index + 1].path}/`));

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
                <div className="roncin-chrome-tab-content">
                  {getRouteIcon(tag.path)}
                  <span className="roncin-chrome-tab-title">{tag.title}</span>
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
                </div>

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
