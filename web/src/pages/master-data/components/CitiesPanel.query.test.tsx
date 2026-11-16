import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import CitiesPanel from './CitiesPanel';

const listRegions = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListAdministrativeRegions: listRegions,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ isSystemWorkspace: true }),
}));
vi.mock('@/components/ui/master-data-template/MasterDataTemplate', () => ({
  MasterDataTemplate: ({
    items,
  }: {
    items: Array<{ enabled: boolean; name: string }>;
  }) => (
    <div>
      {items.map((item) => (
        <output key={item.name} aria-label={item.name}>
          {String(item.enabled)}
        </output>
      ))}
    </div>
  ),
}));

describe('行政区划 Query 的真实 ProtoJSON 启停投影', () => {
  it.each([
    [undefined, false],
    [false, false],
    [true, true],
  ])('wire enabled=%s 映射为 %s', async (enabled, expected) => {
    listRegions.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'region',
          code: '310000000000',
          name: '上海市',
          level: 1,
          ...(enabled === undefined ? {} : { enabled }),
        },
      ],
    });
    renderWithClient(<CitiesPanel />);
    expect(await screen.findByLabelText('上海市')).toHaveTextContent(
      String(expected),
    );
  });
});
