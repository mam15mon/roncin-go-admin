import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { partnerServiceUpdatePartner } from '@/services/roncin/partnerService';
import RoleSwitchModal from './RoleSwitchModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceUpdatePartner: vi.fn(),
}));
beforeEach(() => vi.clearAllMocks());
it('客商角色转换失败保留选中身份和弹窗，完整实体成功后才关闭刷新', async () => {
  vi.mocked(partnerServiceUpdatePartner)
    .mockResolvedValueOnce(undefined as never)
    .mockResolvedValueOnce({ success: false, data: { id: 'p-1' } })
    .mockResolvedValueOnce({ success: true, data: {} })
    .mockResolvedValueOnce({ success: true, data: { id: 'p-1' } });
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  renderWithApp(
    <RoleSwitchModal
      open
      partner={{
        id: 'p-1',
        legalName: '测试客商',
        roles: [{ type: 1, enabled: true }],
      }}
      onClose={onClose}
      onSuccess={onSuccess}
    />,
  );
  const supplier = screen.getByRole('checkbox', { name: '供应商' });
  fireEvent.click(supplier);
  for (let n = 1; n <= 3; n++) {
    fireEvent.click(screen.getByRole('button', { name: '确认转换' }));
    await waitFor(() =>
      expect(partnerServiceUpdatePartner).toHaveBeenCalledTimes(n),
    );
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '确认转换' })).not.toHaveClass(
        'ant-btn-loading',
      ),
    );
    expect(supplier).toBeChecked();
    expect(onClose).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();
  }
  fireEvent.click(screen.getByRole('button', { name: '确认转换' }));
  await waitFor(() => expect(onSuccess).toHaveBeenCalledOnce());
  expect(onClose).toHaveBeenCalledOnce();
});
