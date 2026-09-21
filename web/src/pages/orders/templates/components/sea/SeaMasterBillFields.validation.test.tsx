import {
  ProForm,
  type ProFormInstance,
  ProFormText,
} from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as orderService from '@/services/roncin/orderService';
import { SeaMasterBillFields } from './SeaTransportSection';

const field = 'seaMasterBillMasterNo';
const pendingMessage = '正在核对已有主单，请稍候';
type MatchResult = Awaited<
  ReturnType<typeof orderService.orderServiceMatchSeaMasterBillCandidate>
>;

function setup(isDetail = false, existingMbl?: API.SeaMasterBillSummary) {
  const formRef: { current: ProFormInstance | undefined } = {
    current: undefined,
  };
  render(
    <ProForm
      formRef={formRef}
      submitter={false}
      initialValues={{
        shippingLineId: 'carrier-1',
        [field]: 'MBL123',
        seaMasterBill: existingMbl,
      }}
    >
      <ProFormText name="shippingLineId" hidden />
      <SeaMasterBillFields isDetail={isDetail} />
    </ProForm>,
  );
  return formRef;
}

async function setupPending(isDetail = false) {
  const deferred = Promise.withResolvers<MatchResult>();
  const match = vi
    .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
    .mockReturnValue(deferred.promise);
  const form = setup(isDetail);
  await waitFor(() => expect(match).toHaveBeenCalledOnce());
  await act(async () => {
    await form.current?.validateFields([field]).catch(() => undefined);
  });
  expect(form.current?.getFieldError(field)).toContain(pendingMessage);
  return { ...deferred, form };
}

const candidate: API.SeaMasterBillCandidate = {
  id: 'mbl-1',
  masterNo: 'MBL123',
  version: '1',
  transportExecutions: [{ id: 'te-1', version: '1', vesselName: '测试船' }],
};

afterEach(() => vi.restoreAllMocks());

describe('主单候选核对与字段校验同步', () => {
  it('核对未命中后无需再次输入就清除等待提示，并允许保存', async () => {
    const { resolve, form } = await setupPending();
    await act(async () => resolve({ matched: false }));
    await waitFor(() => expect(form.current?.getFieldError(field)).toEqual([]));
    await waitFor(() =>
      expect(screen.queryByText(pendingMessage)).not.toBeInTheDocument(),
    );
    await act(async () => {
      await expect(form.current?.validateFields([field])).resolves.toEqual({
        [field]: 'MBL123',
      });
    });
  });

  it('核对失败后把等待错误替换为请求错误，继续阻止保存', async () => {
    const { reject, form } = await setupPending();
    await act(async () => reject(new Error('核对请求失败')));
    await waitFor(() =>
      expect(form.current?.getFieldError(field)).toEqual(['核对请求失败']),
    );
  });

  it('命中航程冲突后保留真实阻断错误', async () => {
    const { resolve, form } = await setupPending();
    await act(async () =>
      resolve({
        matched: true,
        candidate,
        conflicts: [{ field: 'vesselName', message: '船名不同' }],
      }),
    );
    await waitFor(() =>
      expect(form.current?.getFieldError(field)).toEqual([
        '航程信息与已有主单冲突，不能确认关联',
      ]),
    );
  });

  it('新建单自动关联成功后清除等待错误', async () => {
    const { resolve, form } = await setupPending();
    await act(async () => resolve({ matched: true, candidate }));
    await waitFor(() => {
      expect(form.current?.getFieldValue('seaMasterBillCandidateId')).toBe(
        'mbl-1',
      );
      expect(form.current?.getFieldError(field)).toEqual([]);
      expect(screen.getByText('运输执行：')).toBeInTheDocument();
    });
  });

  it('待确认关联后写入候选 ID 会重新校验并解除阻断', async () => {
    const { resolve, form } = await setupPending(true);
    await act(async () => resolve({ matched: true, candidate }));
    await waitFor(() =>
      expect(form.current?.getFieldError(field)).toEqual([
        '发现已有主单，请明确确认关联后再保存',
      ]),
    );
    await act(async () => {
      form.current?.setFieldValue('seaMasterBillCandidateId', candidate.id);
    });
    await waitFor(() => expect(form.current?.getFieldError(field)).toEqual([]));
  });

  it('核对期间改为非法主单号后仍保留格式错误', async () => {
    const { resolve, form } = await setupPending();
    fireEvent.change(screen.getByPlaceholderText('请输入主单号'), {
      target: { value: 'MBL-123' },
    });
    await act(async () => resolve({ matched: false }));
    await waitFor(() =>
      expect(form.current?.getFieldError(field)).toEqual([
        '主单号仅允许包含英文字母和数字，禁止包含空格或特殊字符',
      ]),
    );
  });

  it('详情主单身份未改变时无需查询，也不产生核对阻断', async () => {
    const match = vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    );
    const form = setup(true, {
      masterNo: 'MBL123',
      shippingLineId: 'carrier-1',
      memberCount: 1,
    });
    await act(async () => {
      await expect(form.current?.validateFields([field])).resolves.toEqual({
        [field]: 'MBL123',
      });
    });
    expect(match).not.toHaveBeenCalled();
    expect(form.current?.getFieldError(field)).toEqual([]);
  });
});
