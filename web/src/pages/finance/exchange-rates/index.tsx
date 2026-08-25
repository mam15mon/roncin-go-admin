import { history } from '@umijs/max';
import React, { useEffect } from 'react';

export { ExchangeRatesPanel } from '../../settings/components/ExchangeRatesPanel';

export default function ExchangeRatesRedirect() {
  useEffect(() => {
    history.replace('/settings?tab=exchange-rates');
  }, []);
  return null;
}
