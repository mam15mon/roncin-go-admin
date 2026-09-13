import { Navigate, useLocation } from '@umijs/max';
import React from 'react';

/**
 * RegisterRedirect
 * 钉钉登录与入职注册已统一收敛至登录页闭环（老员工直接登录，新员工扫码自动进入入职审批），
 * 历史 /user/register 访问统一 replace 重定向至 /user/login 并透传所有 query 参数（如 ?invite=...）。
 */
export default function RegisterRedirect() {
  const location = useLocation();
  return <Navigate to={`/user/login${location.search}`} replace />;
}
