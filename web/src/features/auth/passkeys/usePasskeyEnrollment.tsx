import { useCallback, useRef, useState } from 'react';
import { PasskeyRegistrationModal } from './PasskeyRegistrationModal';
import { clearPasskeyEnrollment, shouldEnrollPasskey } from './preferences';

export function usePasskeyEnrollment() {
  const continuation = useRef<(() => void) | null>(null);
  const [userId, setUserId] = useState<string>();
  const completeLogin = useCallback(
    (onContinue: () => void, verifiedUserId: string) => {
      if (!shouldEnrollPasskey()) {
        onContinue();
        return;
      }
      continuation.current = onContinue;
      setUserId(verifiedUserId);
    },
    [],
  );
  const continueLogin = useCallback(() => {
    clearPasskeyEnrollment();
    setUserId(undefined);
    const callback = continuation.current;
    continuation.current = null;
    callback?.();
  }, []);
  return {
    completeLogin,
    isEnrollmentOpen: Boolean(userId),
    enrollmentPrompt: userId ? (
      <PasskeyRegistrationModal
        key={userId}
        open
        userId={userId}
        onRegistered={continueLogin}
        onSkip={continueLogin}
      />
    ) : null,
  };
}
