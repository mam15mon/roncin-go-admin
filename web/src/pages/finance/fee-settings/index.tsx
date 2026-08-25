import { history } from '@umijs/max';
import React, { useEffect } from 'react';

export { FeeSettingsPanel } from '../../settings/components/FeeSettingsPanel';

export default function FeeSettingsRedirect() {
  useEffect(() => {
    history.replace('/settings?tab=fee-settings');
  }, []);
  return null;
}
