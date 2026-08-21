import { CloseOutlined, HomeOutlined } from '@ant-design/icons';
import { history, useLocation } from '@umijs/max';
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
 * 简洁多页签导航组件
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

  return (
    <nav className="roncin-tags-view" aria-label="多页签导航">
      <div className="roncin-tags-view-container" role="tablist">
        {tags.map((tag) => {
          const isActive =
            tag.path === '/welcome'
              ? currentPath === '/welcome' || currentPath === '/'
              : currentPath === tag.path ||
                currentPath.startsWith(`${tag.path}/`);

          return (
            <div
              role="tab"
              tabIndex={0}
              aria-selected={isActive}
              key={tag.key}
              className={`roncin-tag-item ${
                isActive ? 'roncin-tag-item-active' : ''
              }`}
              onClick={() => handleTabClick(tag)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  handleTabClick(tag);
                }
              }}
            >
              {tag.key === '/welcome' ? (
                <HomeOutlined className="roncin-tag-icon" />
              ) : (
                <span className="roncin-tag-dot" />
              )}
              <span className="roncin-tag-title">{tag.title}</span>
              {tag.closable && (
                <button
                  type="button"
                  aria-label={`关闭 ${tag.title}`}
                  className="roncin-tag-close"
                  onClick={(e) => handleClose(e, tag)}
                >
                  <CloseOutlined />
                </button>
              )}
            </div>
          );
        })}
      </div>
    </nav>
  );
};
