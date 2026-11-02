import { useCallback, useSyncExternalStore } from 'react';

const ENROLL_AFTER_LOGIN = 'roncin.passkey.enroll-after-login';
const PREFERENCE_CHANGED = 'roncin:passkey-enrollment-changed';

export function shouldEnrollPasskey(): boolean {
  return sessionStorage.getItem(ENROLL_AFTER_LOGIN) === 'true';
}

function notifyPreferenceChanged() {
  window.dispatchEvent(new Event(PREFERENCE_CHANGED));
}

function subscribePreference(listener: () => void) {
  window.addEventListener(PREFERENCE_CHANGED, listener);
  return () => window.removeEventListener(PREFERENCE_CHANGED, listener);
}

export function clearPasskeyEnrollment(): void {
  sessionStorage.removeItem(ENROLL_AFTER_LOGIN);
  notifyPreferenceChanged();
}

export function useEnrollmentPreference() {
  const enrollAfterLogin = useSyncExternalStore(
    subscribePreference,
    shouldEnrollPasskey,
    () => false,
  );
  const setEnrollAfterLogin = useCallback((value: boolean) => {
    if (value) {
      sessionStorage.setItem(ENROLL_AFTER_LOGIN, 'true');
      notifyPreferenceChanged();
    } else {
      clearPasskeyEnrollment();
    }
  }, []);
  return { enrollAfterLogin, setEnrollAfterLogin };
}
