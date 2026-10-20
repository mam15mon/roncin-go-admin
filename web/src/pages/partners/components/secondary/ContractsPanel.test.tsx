import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerContractStatus } from '@/enums.generated';
import {
  partnerServiceCreatePartnerContract,
  partnerServiceListPartnerContracts,
  partnerServiceUpdatePartnerContract,
} from '@/services/roncin/partnerService';
import ContractsPanel from './ContractsPanel';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartnerContract: vi.fn(),
  partnerServiceListPartnerContracts: vi.fn(),
  partnerServiceUpdatePartnerContract: vi.fn(),
}));

const contract: API.PartnerContract = {
  id: 'contract-1',
  contractNo: 'CT2026102000001',
  name: '年度合同',
  status: PartnerContractStatus.PARTNER_CONTRACT_STATUS_PENDING,
  allowedStatuses: [PartnerContractStatus.PARTNER_CONTRACT_STATUS_PENDING],
  startDate: dayjs('2026-10-20').startOf('day').toISOString(),
  endDate: dayjs('2027-10-20').startOf('day').toISOString(),
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(partnerServiceListPartnerContracts).mockResolvedValue({ data: [] });
  vi.mocked(partnerServiceCreatePartnerContract).mockResolvedValue({
    data: contract,
  });
  vi.mocked(partnerServiceUpdatePartnerContract).mockResolvedValue({
    data: contract,
  });
});

describe('独立合同列表日期提交', () => {
  it('创建通过真实日期选择与模态表单提交 RFC3339 日期', async () => {
    renderWithApp(<ContractsPanel partner={{ id: 'partner-1' }} canManage />);
    fireEvent.click(await screen.findByRole('button', { name: /新增合同/ }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText('合同名称'), {
      target: { value: '新建年度合同' },
    });
    const start = within(dialog).getByLabelText('合同有效起止期限');
    const end = dialog.querySelectorAll('.ant-picker-input input')[1];
    fireEvent.focus(start);
    fireEvent.change(start, { target: { value: '2026-10-20' } });
    fireEvent.keyDown(start, { key: 'Enter', code: 'Enter' });
    fireEvent.focus(end);
    fireEvent.change(end, { target: { value: '2027-10-20' } });
    fireEvent.keyDown(end, { key: 'Enter', code: 'Enter' });
    fireEvent.click(within(dialog).getByRole('button', { name: '确 定' }));
    await waitFor(() =>
      expect(partnerServiceCreatePartnerContract).toHaveBeenCalledOnce(),
    );
    const input = vi.mocked(partnerServiceCreatePartnerContract).mock
      .calls[0][1].contract;
    expect(input).not.toHaveProperty('contractNo');
    expect(input.startDate).toBe(contract.startDate);
    expect(input.endDate).toBe(contract.endDate);
  });

  it('编辑通过真实模态表单保留编号并提交 RFC3339 日期', async () => {
    vi.mocked(partnerServiceListPartnerContracts).mockResolvedValue({
      data: [contract],
    });
    renderWithApp(<ContractsPanel partner={{ id: 'partner-1' }} canManage />);
    fireEvent.click(await screen.findByRole('button', { name: /编\s*辑/ }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByLabelText('合同编号')).toHaveValue(
      contract.contractNo,
    );
    fireEvent.click(within(dialog).getByRole('button', { name: '确 定' }));
    await waitFor(() =>
      expect(partnerServiceUpdatePartnerContract).toHaveBeenCalledOnce(),
    );
    expect(partnerServiceUpdatePartnerContract).toHaveBeenCalledWith(
      { partnerId: 'partner-1', id: contract.id },
      expect.objectContaining({
        contract: expect.objectContaining({
          startDate: contract.startDate,
          endDate: contract.endDate,
        }),
      }),
    );
  });
});
