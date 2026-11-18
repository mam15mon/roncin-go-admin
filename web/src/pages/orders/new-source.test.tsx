import { act, cleanup, fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  getFormDraftKey,
  getFormDraftScope,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  deferred,
  getOrder,
  input,
  latestProps,
  mount,
  navigate,
  permissions,
  ready,
  reference,
  sourceOrder,
} from './new-source.fixtures';

describe('连续新增分单的来源与资源生命周期', () => {
  it('本次来源读取成功前不挂表单，只预填主单共享身份，不复制本票资料', async () => {
    const request = deferred<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(request.promise);
    mount();
    expect(screen.getByText('正在读取来源主单...')).toBeInTheDocument();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    await act(async () => request.resolve({ data: sourceOrder('A') }));
    await ready();
    expect(input('主单号')).toHaveValue('MBL-A');
    expect(input('船公司')).toHaveValue('line-1');
    for (const label of ['客户', '货物', '目的地'])
      expect(input(label)).toHaveValue('');
    expect(latestProps.seaHouseOrderCreateSource).toEqual({
      orderId: 'A',
      orderNo: 'SE-A',
      masterBillId: 'mbl-A',
      masterNo: 'MBL-A',
      shippingLineId: 'line-1',
      shippingLineName: '来源船公司',
      transportExecutionId: 'te-A',
    });
    expect(getOrder).toHaveBeenCalledWith(
      { id: 'A' },
      { skipErrorHandler: true },
    );
  });

  it('来源A的迟到读取不能回填已经进入的B表单', async () => {
    const request = deferred<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(request.promise);
    const { router } = mount();
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    fireEvent.change(reference(), { target: { value: 'B本次输入' } });
    await act(async () => request.resolve({ data: sourceOrder('A') }));
    expect(reference()).toHaveValue('B本次输入');
    expect(input('主单号')).toHaveValue('MBL-B');
    expect(latestProps.seaHouseOrderCreateSource?.orderId).toBe('B');
  });

  it('无读取/创建/当前公司能力及空重复来源参数都阻断且不请求来源', async () => {
    for (const operation of ['read', 'create', 'organization'] as const) {
      permissions[operation] = false;
      mount();
      expect(
        screen.getByText(
          operation === 'create'
            ? '无权新建此类订单'
            : '无权读取来源订单或新建分单',
        ),
      ).toBeInTheDocument();
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      cleanup();
      permissions[operation] = true;
    }
    for (const search of [
      '?sourceOrderId=',
      '?sourceOrderId=A&sourceOrderId=A',
    ]) {
      mount(`/orders/sea-export/new${search}`);
      expect(screen.getByText('无法从此订单新增分单')).toBeInTheDocument();
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      cleanup();
    }
    expect(getOrder).not.toHaveBeenCalled();
  });

  it('来源读取失败给出重试，DIRECT及跨公司来源均不退化为空白建单', async () => {
    getOrder.mockRejectedValueOnce(new Error('来源已失效'));
    mount();
    await screen.findByText('来源已失效');
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '重新加载来源订单' }));
    await ready();
    cleanup();
    for (const order of [
      {
        ...sourceOrder('A'),
        seaDocumentStructure:
          SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
      },
      { ...sourceOrder('A'), organizationId: 'org-2' },
    ]) {
      getOrder.mockResolvedValueOnce({ data: order });
      mount();
      await screen.findByText('来源订单加载失败');
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      cleanup();
    }
  });

  it('普通新建及来源A/B分别恢复草稿，无关query变化保留当前实例', async () => {
    const { router } = mount('/orders/sea-export/new');
    await ready();
    fireEvent.change(reference(), { target: { value: '普通草稿' } });
    await navigate(router, '/orders/sea-export/new?sourceOrderId=A');
    expect(reference()).toHaveValue('');
    fireEvent.change(reference(), { target: { value: 'A草稿' } });
    await navigate(router, '/orders/sea-export/new?extra=1&sourceOrderId=A');
    expect(reference()).toHaveValue('A草稿');
    expect(getOrder).toHaveBeenCalledTimes(1);
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    expect(reference()).toHaveValue('');
    expect(input('主单号')).toHaveValue('MBL-B');
    fireEvent.change(reference(), { target: { value: 'B草稿' } });
    await navigate(router, '/orders/sea-export/new?sourceOrderId=A');
    expect(reference()).toHaveValue('A草稿');
    await navigate(router, '/orders/sea-export/new');
    expect(reference()).toHaveValue('普通草稿');
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    expect(reference()).toHaveValue('B草稿');
  });

  it('来源加载完成后恢复本来源草稿，后台重查成功或失败均不覆盖输入和初始来源', async () => {
    saveFormDraft(
      getFormDraftKey(
        '/orders/sea-export/new',
        '/orders/sea-export/new?sourceOrderId=A',
        getFormDraftScope('user-1', 'org-1'),
      ),
      {
        customerReferenceNo: 'A恢复草稿',
        seaMasterBillMasterNo: '草稿改选主单',
      },
    );
    const { client } = mount();
    await ready();
    expect(reference()).toHaveValue('A恢复草稿');
    expect(input('主单号')).toHaveValue('草稿改选主单');
    fireEvent.change(reference(), { target: { value: '本次输入' } });
    getOrder.mockResolvedValueOnce({ data: sourceOrder('A', '后台新主单') });
    await act(async () => {
      await client.invalidateQueries({
        queryKey: ['orders', 'sea-house-create-source'],
      });
    });
    expect(reference()).toHaveValue('本次输入');
    expect(input('主单号')).toHaveValue('草稿改选主单');
    expect(latestProps.seaHouseOrderCreateSource?.masterNo).toBe('MBL-A');
    getOrder.mockRejectedValueOnce(new Error('后台重查失败'));
    await act(async () => {
      await client.invalidateQueries({
        queryKey: ['orders', 'sea-house-create-source'],
      });
    });
    expect(reference()).toHaveValue('本次输入');
  });

  it('返回已有缓存来源仍等待本次GET；失败阻断，重试使用当前来源身份', async () => {
    const { router } = mount();
    await ready();
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    const request = deferred<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(request.promise);
    await act(async () => {
      await router.navigate('/orders/sea-export/new?sourceOrderId=A');
    });
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    await act(async () => request.reject(new Error('缓存来源已失效')));
    await screen.findByText('缓存来源已失效');
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    getOrder.mockResolvedValueOnce({
      data: {
        ...sourceOrder('A', '最新主单'),
        seaMasterBill: {
          ...sourceOrder('A').seaMasterBill,
          masterNo: '最新主单',
          transportExecutionId: 'te-new',
        },
      },
    });
    fireEvent.click(screen.getByRole('button', { name: '重新加载来源订单' }));
    await ready();
    expect(input('主单号')).toHaveValue('最新主单');
    expect(latestProps.seaHouseOrderCreateSource?.transportExecutionId).toBe(
      'te-new',
    );
  });
});
