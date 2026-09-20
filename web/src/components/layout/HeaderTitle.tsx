import { useLocation } from 'react-router';
import React from 'react';
import { resolveRouteTitle } from './routeUtils';

/**
 * 顶栏左侧路由标题组件
 */
export const HeaderTitle: React.FC = () => {
  const location = useLocation();
  const title = resolveRouteTitle(location.pathname);

  if (!title) return null;

  return (
    <div className="roncin-header-title">
      <span className="roncin-header-title-text">{title}</span>
    </div>
  );
};
