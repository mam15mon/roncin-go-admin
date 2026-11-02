import { useCallback, useState } from 'react';

const REMEMBER_LOGIN_KEY = 'roncin.login.remember';

// 只保存非敏感界面偏好，登录凭证始终由服务端 HttpOnly Cookie 管理。
export function getRememberLogin(): boolean {
  return localStorage.getItem(REMEMBER_LOGIN_KEY) !== 'false';
}

export function useRememberLogin() {
  const [rememberLogin, setRememberLoginState] = useState(getRememberLogin);
  const setRememberLogin = useCallback((value: boolean) => {
    localStorage.setItem(REMEMBER_LOGIN_KEY, String(value));
    setRememberLoginState(value);
  }, []);
  return { rememberLogin, setRememberLogin };
}
