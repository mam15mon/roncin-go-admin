import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CitiesPanel from './CitiesPanel';
import CurrenciesPanel from './CurrenciesPanel';

const services = vi.hoisted(() => ({ currencies: vi.fn(), regions: vi.fn() }));

vi.mock('@/app/access', () => ({
  useAccess: () => ({ isSystemWorkspace: true }),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: services.currencies,
  masterDataServiceListAdministrativeRegions: services.regions,
}));

beforeEach(() => {
  services.currencies.mockReset().mockResolvedValue({ success: true });
  services.regions.mockReset().mockResolvedValue({ success: true });
});

describe('全量主数据查询失败恢复', () => {
  it.each([
    [
      CurrenciesPanel,
      '货币主数据加载失败',
      services.currencies,
      '币种数据加载失败',
    ],
    [
      CitiesPanel,
      '行政区划数据加载失败',
      services.regions,
      '行政区划数据加载失败',
    ],
  ])(
    '%s 首次响应缺失时不展示空数据统计，重试后才接受合法空列表',
    async (Panel, title, request, detail) => {
      request.mockResolvedValueOnce(undefined);
      renderWithApp(<Panel />);
      const alert = await screen.findByRole('alert');
      expect(alert).toHaveTextContent(title);
      expect(alert).toHaveTextContent(detail);
      expect(screen.queryByText('标准货币主库')).not.toBeInTheDocument();
      expect(screen.queryByText('省级行政区')).not.toBeInTheDocument();
      expect(screen.queryByText(/Cannot read/)).not.toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
      await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
      await waitFor(() =>
        expect(screen.queryByText(title)).not.toBeInTheDocument(),
      );
      expect(screen.getAllByRole('table').length).toBeGreaterThan(0);
    },
  );
});
