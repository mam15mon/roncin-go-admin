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
import ContractCardList from './ContractCardList';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartnerContract: vi.fn(),
  partnerServiceListPartnerContracts: vi.fn(),
  partnerServiceUpdatePartnerContract: vi.fn(),
}));

const contract: API.PartnerContract = {
  id: 'contract-1',
  contractNo: 'CT2026102000001',
  name: '年度合同',
  status: PartnerContractStatus.PARTNER_CONTRACT_STATUS_ACTIVE,
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

describe('合同卡片日期提交', () => {
  it('创建通过真实模态表单提交 RFC3339 日期且不携带手工编号', async () => {
    renderWithApp(<ContractCardList partnerId="partner-1" />);
    fireEvent.click(await screen.findByText('添加合同'));
    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText('合同名称'), {
      target: { value: '新建年度合同' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: '确 定' }));
    await waitFor(() =>
      expect(partnerServiceCreatePartnerContract).toHaveBeenCalledOnce(),
    );
    const input = vi.mocked(partnerServiceCreatePartnerContract).mock
      .calls[0][1].contract;
    expect(input).not.toHaveProperty('contractNo');
    expect(input.startDate).toBe(dayjs().startOf('day').toISOString());
    expect(input.endDate).toBe(
      dayjs().add(1, 'year').startOf('day').toISOString(),
    );
  });

  it('编辑保留原合同编号并提交 RFC3339 日期', async () => {
    vi.mocked(partnerServiceListPartnerContracts).mockResolvedValue({
      data: [contract],
    });
    renderWithApp(<ContractCardList partnerId="partner-1" />);
    await screen.findByText('年度合同');
    fireEvent.click(screen.getByRole('button', { name: 'edit' }));
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
