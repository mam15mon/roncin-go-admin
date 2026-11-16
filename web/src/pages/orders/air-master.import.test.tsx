import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  masterResponse,
  masterServices,
  renderMaster,
  setupMasterTests,
} from './air-master-test.fixtures';

setupMasterTests();
describe('AI共享主单真实类型与路由', () => {
  it('只有AI能力仍可读取AI主单并按真实模式继续AI建票', async () => {
    masterServices.canOrder.mockImplementation(
      (type) => type === OrderBusinessType.BUSINESS_TYPE_AI,
    );
    masterServices.get.mockResolvedValue(
      masterResponse('4', { businessType: 'AI', documentMode: 'HOUSE' }),
    );
    renderMaster('air-import');
    const button = await screen.findByRole('button', {
      name: '从本主单继续建票',
    });
    await waitFor(() => expect(button).toBeEnabled());
    fireEvent.click(button);
    expect(masterServices.push).toHaveBeenCalledWith(
      '/orders/air-import/new?masterId=master&documentMode=HOUSE',
    );
    expect(
      masterServices.canOrder.mock.calls.every(
        ([type]) => type === OrderBusinessType.BUSINESS_TYPE_AI,
      ),
    ).toBe(true);
  });
  it('只有AE能力不能读取AI共享主单，不发任何主单请求', () => {
    masterServices.canOrder.mockImplementation(
      (type) => type === OrderBusinessType.BUSINESS_TYPE_AE,
    );
    renderMaster('air-import');
    expect(screen.getByText('无权查看空运主单')).toBeVisible();
    expect(masterServices.get).not.toHaveBeenCalled();
  });
  it.each([
    { businessType: 'AE' },
    { businessType: undefined },
    { organizationId: undefined },
    { organizationId: 'other' },
    { id: 'other' },
  ])('AI入口严格拒绝错误或缺失主单身份 %j', async (patch) => {
    masterServices.get.mockResolvedValue(
      masterResponse('4', { businessType: 'AI', ...patch }),
    );
    renderMaster('air-import');
    expect(await screen.findByText('主单加载失败')).toBeVisible();
    expect(screen.queryByText('78112345675')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '从本主单继续建票' }),
    ).not.toBeInTheDocument();
  });
});
