import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Form, Input } from 'antd';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrderReferenceType } from '@/enums.generated';
import * as orderService from '@/services/roncin/orderService';
import { SeaOrderReferenceField } from './SeaBasicInfoSection';

type CheckResult = Awaited<
  ReturnType<typeof orderService.orderServiceCheckOrderReference>
>;

function setup(
  name: 'customerReferenceNo' | 'internalReferenceNo' = 'internalReferenceNo',
) {
  render(
    <Form initialValues={{ customerId: 'customer-1' }}>
      <Form.Item name="customerId">
        <Input aria-label="委托单位" />
      </Form.Item>
      <SeaOrderReferenceField name={name} />
    </Form>,
  );
  return screen.getByPlaceholderText('请输入');
}

afterEach(() => vi.restoreAllMocks());

describe('订单参考编号自动查重', () => {
  it('输入后自动核对并显示通过对勾，清空后移除反馈', async () => {
    const check = vi
      .spyOn(orderService, 'orderServiceCheckOrderReference')
      .mockResolvedValue({ data: { duplicate: false } });
    const input = setup();
    expect(screen.queryByText('重复校验')).not.toBeInTheDocument();
    expect(check).not.toHaveBeenCalled();
    fireEvent.change(input, { target: { value: 'INTERNAL-001' } });
    await waitFor(() =>
      expect(check).toHaveBeenCalledWith(
        {
          referenceType: OrderReferenceType.ORDER_REFERENCE_TYPE_INTERNAL,
          referenceNo: 'INTERNAL-001',
          customerId: undefined,
        },
        { skipErrorHandler: true },
      ),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('img', { name: 'check-circle' }),
      ).toBeInTheDocument(),
    );
    fireEvent.change(input, { target: { value: '' } });
    await waitFor(() =>
      expect(
        screen.queryByRole('img', { name: 'check-circle' }),
      ).not.toBeInTheDocument(),
    );
  });

  it('重复编号在字段下显示订单提示，不显示通过对勾', async () => {
    vi.spyOn(orderService, 'orderServiceCheckOrderReference').mockResolvedValue(
      { data: { duplicate: true, orderNo: 'SE-001' } },
    );
    fireEvent.change(setup(), { target: { value: 'DUPLICATE' } });
    expect(
      await screen.findByText('企业内部编号已用于订单 SE-001'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('img', { name: 'check-circle' }),
    ).not.toBeInTheDocument();
  });

  it('客户编号按委托单位查重，切换委托单位会重新核对', async () => {
    const check = vi
      .spyOn(orderService, 'orderServiceCheckOrderReference')
      .mockResolvedValue({ data: { duplicate: false } });
    fireEvent.change(setup('customerReferenceNo'), {
      target: { value: 'CUSTOMER-001' },
    });
    await waitFor(() =>
      expect(check).toHaveBeenCalledWith(
        expect.objectContaining({ customerId: 'customer-1' }),
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('img', { name: 'check-circle' }),
      ).toBeInTheDocument(),
    );
    fireEvent.change(screen.getByLabelText('委托单位'), {
      target: { value: 'customer-2' },
    });
    await waitFor(() =>
      expect(check).toHaveBeenLastCalledWith(
        expect.objectContaining({ customerId: 'customer-2' }),
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('img', { name: 'check-circle' }),
      ).toBeInTheDocument(),
    );
  });

  it('未选择委托单位时给出提示，不发客户编号查重请求', async () => {
    const check = vi.spyOn(orderService, 'orderServiceCheckOrderReference');
    const input = setup('customerReferenceNo');
    fireEvent.change(screen.getByLabelText('委托单位'), {
      target: { value: '' },
    });
    fireEvent.change(input, { target: { value: 'CUSTOMER-001' } });
    expect(await screen.findByText('请先选择委托单位')).toBeInTheDocument();
    expect(check).not.toHaveBeenCalled();
    expect(
      screen.queryByRole('img', { name: 'check-circle' }),
    ).not.toBeInTheDocument();
  });

  it('旧编号的迟到核对结果不能覆盖新编号的重复提示', async () => {
    const oldRequest = Promise.withResolvers<CheckResult>();
    const check = vi
      .spyOn(orderService, 'orderServiceCheckOrderReference')
      .mockReturnValueOnce(oldRequest.promise)
      .mockResolvedValue({ data: { duplicate: true, orderNo: 'SE-002' } });
    const input = setup();
    fireEvent.change(input, { target: { value: 'OLD' } });
    await waitFor(() => expect(check).toHaveBeenCalledOnce());
    fireEvent.change(input, { target: { value: 'NEW' } });
    expect(
      await screen.findByText('企业内部编号已用于订单 SE-002'),
    ).toBeInTheDocument();
    await act(async () => oldRequest.resolve({ data: { duplicate: false } }));
    expect(
      screen.getByText('企业内部编号已用于订单 SE-002'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('img', { name: 'check-circle' }),
    ).not.toBeInTheDocument();
  });

  it('接口未返回核对数据时提示失败，不能误显示通过', async () => {
    vi.spyOn(orderService, 'orderServiceCheckOrderReference').mockResolvedValue(
      { success: false },
    );
    fireEvent.change(setup(), { target: { value: 'INTERNAL-001' } });
    expect(
      await screen.findByText('企业内部编号查重失败，请重试'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('img', { name: 'check-circle' }),
    ).not.toBeInTheDocument();
  });
});
