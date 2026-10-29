import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as orderService from '@/services/roncin/orderService';
import {
  candidateResponse,
  masterField,
  renderVoyageForm,
  voyage,
} from './SeaMasterBillFields.test-utils';

afterEach(() => vi.restoreAllMocks());

describe('共享主单航程继承', () => {
  it('单航程自动带入并只读，保留所有独立资料且只核对一次', async () => {
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValue(candidateResponse());
    const independent = {
      customerId: 'customer',
      seaHouseBill: { houseNo: 'HBL-NEW', content: { packageCount: 10 } },
      totalPackages: 20,
      totalGrossWeightKg: '300',
      totalVolumeCbm: '4',
      destinationLocationId: 'destination',
      fees: [{ amount: '100' }],
    };
    const { form } = renderVoyageForm(independent);
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe('te-1'),
    );
    expect(form.getFieldValue('seaMasterBillExpectedCandidateVersion')).toBe(
      '5',
    );
    expect(form.getFieldValue('seaMasterBillExpectedCandidateTeVersion')).toBe(
      '3',
    );
    expect(form.getFieldsValue(true)).toMatchObject(independent);
    expect(form.getFieldValue('originLocationId')).toBe('origin');
    expect(form.getFieldValue('dischargeLocationId')).toBe('discharge');
    expect(form.getFieldValue('transitLocationId')).toBe('transit');
    expect(form.getFieldValue('etd').valueOf()).toBe(
      dayjs(voyage.etd).valueOf(),
    );
    for (const label of [
      '船名航次',
      '起运港',
      '卸货港',
      '中转港',
      'ETD',
      'ETA',
    ])
      expect(screen.getByLabelText(label)).toBeDisabled();
    expect(screen.getByLabelText('目的港')).not.toBeDisabled();
    expect(screen.getByLabelText('船公司')).not.toBeDisabled();
    expect(
      screen.queryByRole('combobox', { name: '本票共用航程' }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText('本票保存后将加入该主单，共 3 票。'),
    ).toBeInTheDocument();
    await act(async () => {
      await form.validateFields([masterField]);
    });
    expect(match).toHaveBeenCalledOnce();
  });

  it('非空差异先保留，点击采用后覆盖为共享值且解除保存阻断', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(candidateResponse());
    const { form } = renderVoyageForm({
      vesselVoyage: 'OTHER / 002W',
      originLocationId: 'other-origin',
      etd: dayjs('2026-11-01'),
    });
    await screen.findByRole(
      'button',
      { name: '采用主单航程' },
      // 等待真实候选防抖查询，不依赖满载时默认 1s 的完成窗口。
      { timeout: 5000 },
    );
    expect(form.getFieldValue('vesselVoyage')).toBe('OTHER / 002W');
    expect(form.getFieldValue('originLocationId')).toBe('other-origin');
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBeUndefined();
    expect(
      screen.getByText(/起运港：本票 宁波；主单 上海/),
    ).toBeInTheDocument();
    await act(async () => {
      await expect(form.validateFields([masterField])).rejects.toMatchObject({
        errorFields: [
          expect.objectContaining({
            errors: ['航程信息与已有主单冲突，请确认采用主单航程'],
          }),
        ],
      });
    });
    fireEvent.click(screen.getByRole('button', { name: '采用主单航程' }));
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-1'),
    );
    expect(form.getFieldValue('vesselVoyage')).toBe('EVER TEST / 001W');
    expect(form.getFieldValue('originLocationId')).toBe('origin');
    await act(async () => {
      await expect(form.validateFields([masterField])).resolves.toEqual({
        [masterField]: 'MBL123',
      });
    });
  });

  it('主单缺失值也列入差异，采用不会把新票资料当成共享补全', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(
      candidateResponse({
        transportExecutions: [
          {
            ...voyage,
            transitLocationId: undefined,
            transitLocationName: undefined,
            eta: undefined,
          },
        ],
      }),
    );
    const { form } = renderVoyageForm({
      transitLocationId: 'transit',
      eta: dayjs('2026-10-25'),
    });
    await screen.findByRole(
      'button',
      { name: '采用主单航程' },
      // 与上一场景一致，等待真实防抖查询在全量并发下完成。
      { timeout: 5000 },
    );
    expect(
      screen.getByText(/中转港：本票 当前已选地点；主单 未填写/),
    ).toBeInTheDocument();
    expect(screen.getByText('已有主单的航程资料尚未填全')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'SE-A' })).toHaveAttribute(
      'target',
      '_blank',
    );
    fireEvent.click(screen.getByRole('button', { name: '采用主单航程' }));
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-1'),
    );
    expect(form.getFieldValue('transitLocationId')).toBeUndefined();
    expect(form.getFieldValue('eta')).toBeUndefined();
    await act(async () => {
      await expect(form.validateFields(['eta'])).rejects.toMatchObject({
        errorFields: [
          expect.objectContaining({ errors: ['请选择 ETA（预计到港时间）'] }),
        ],
      });
    });
  });

  it('详情共用航程只读并显示真实票数，单票更正保留原能力', async () => {
    const shared = renderVoyageForm(
      {
        seaMasterBill: {
          ...voyage,
          masterNo: 'MBL123',
          shippingLineId: 'carrier-1',
          memberCount: 2,
        },
      },
      { isDetail: true },
    );
    await waitFor(() =>
      expect(screen.getByLabelText('船名航次')).toBeDisabled(),
    );
    expect(screen.getByText('共用主单 MBL123，共 2 票')).toBeInTheDocument();
    expect(screen.getByText(/更多操作 → 修改共享航程/)).toBeInTheDocument();
    shared.unmount();
    renderVoyageForm(
      {
        seaMasterBill: {
          masterNo: 'MBL123',
          shippingLineId: 'carrier-1',
          memberCount: 1,
        },
      },
      { isDetail: true },
    );
    expect(screen.getByLabelText('船名航次')).not.toBeDisabled();
    expect(screen.getByLabelText('ETD')).not.toBeDisabled();
    await act(async () => {});
  });

  it('父表单业务禁用状态不会被共享字段的可编辑默认值覆盖', async () => {
    renderVoyageForm({ shippingLineId: undefined }, { disabled: true });
    for (const label of [
      '船名航次',
      '起运港',
      '卸货港',
      '中转港',
      'ETD',
      'ETA',
    ])
      expect(screen.getByLabelText(label)).toBeDisabled();
    await act(async () => {});
  });

  it('共享港口缺少名称时明确显示读取失败，同时保留真实地点ID', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(
      candidateResponse({
        transportExecutions: [
          {
            ...voyage,
            originLocationName: undefined,
            dischargeLocationName: undefined,
            transitLocationName: undefined,
          },
        ],
      }),
    );
    const { form } = renderVoyageForm();
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe('te-1'),
    );
    for (const label of ['起运港', '卸货港', '中转港']) {
      expect(
        screen.getByLabelText(label).closest('.ant-select'),
      ).toHaveTextContent(`${label}名称读取失败`);
    }
    expect(form.getFieldValue('originLocationId')).toBe('origin');
    expect(form.getFieldValue('dischargeLocationId')).toBe('discharge');
    expect(form.getFieldValue('transitLocationId')).toBe('transit');
  });
});
