import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import * as XLSX from 'xlsx';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { PartnerRoleType } from '@/enums.generated';
import PartnerExcelImportModal from './PartnerExcelImportModal';

const mocks = vi.hoisted(() => ({ importPartners: vi.fn() }));
vi.mock('xlsx', () => ({
  read: vi.fn(() => ({ SheetNames: ['sheet'], Sheets: { sheet: {} } })),
  utils: {
    sheet_to_json: vi.fn(() => [{ 单位编码: 'CUST', 企业名称: '测试客商' }]),
  },
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceImportPartners: mocks.importPartners,
}));
beforeEach(() => {
  window.history.replaceState({}, '', '/partners/customers');
  vi.clearAllMocks();
});
it('Excel实际载入后保护，提交失败保留，取消清理', async () => {
  mocks.importPartners.mockResolvedValue(undefined);
  const onClose = vi.fn();
  render(
    <App>
      <PartnerExcelImportModal
        open
        onClose={onClose}
        onSuccess={vi.fn()}
        currentRoleType={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
        currentRoleLabel="客户"
      />
    </App>,
  );
  expect(getAnyDirtyTabKey()).toBeUndefined();
  const input = document.querySelector('input[type="file"]');
  if (!input) throw new Error('缺少Excel文件输入');
  const file = new File(['data'], 'partners.xlsx');
  Object.defineProperty(file, 'arrayBuffer', {
    value: async () => new ArrayBuffer(1),
  });
  await act(async () => {
    fireEvent.change(input, { target: { files: [file] } });
  });
  await screen.findByText('测试客商');
  expect(getAnyDirtyTabKey()).toBe('/partners/customers');
  fireEvent.click(screen.getByRole('button', { name: /确认导入/ }));
  await waitFor(() => expect(mocks.importPartners).toHaveBeenCalledOnce());
  expect(getAnyDirtyTabKey()).toBe('/partners/customers');
  expect(onClose).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
  expect(getAnyDirtyTabKey()).toBeUndefined();
});
it('用户修改导入模式属于待提交配置，关闭后清理', () => {
  const view = render(
    <App>
      <PartnerExcelImportModal
        open
        onClose={vi.fn()}
        onSuccess={vi.fn()}
        currentRoleType={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
        currentRoleLabel="客户"
      />
    </App>,
  );
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.click(screen.getByRole('radio', { name: /仅新增/ }));
  expect(getAnyDirtyTabKey()).toBe('/partners/customers');
  view.unmount();
  expect(getAnyDirtyTabKey()).toBeUndefined();
});

it('文件解析无有效行时不误提示，导入模式删除文件时一起恢复', async () => {
  vi.mocked(XLSX.utils.sheet_to_json).mockReturnValueOnce([]);
  render(
    <App>
      <PartnerExcelImportModal
        open
        onClose={vi.fn()}
        onSuccess={vi.fn()}
        currentRoleType={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
        currentRoleLabel="客户"
      />
    </App>,
  );
  const input = document.querySelector('input[type="file"]');
  if (!input) throw new Error('缺少Excel文件输入');
  const file = new File(['data'], 'empty.xlsx');
  Object.defineProperty(file, 'arrayBuffer', {
    value: async () => new ArrayBuffer(1),
  });
  await act(async () => {
    fireEvent.change(input, { target: { files: [file] } });
  });
  await screen.findByText('未在 Excel 中解析到任何有效数据行');
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.click(screen.getByRole('radio', { name: /仅新增/ }));
  expect(getAnyDirtyTabKey()).toBe('/partners/customers');
  fireEvent.click(screen.getByRole('button', { name: 'Remove file' }));
  expect(getAnyDirtyTabKey()).toBeUndefined();
  expect(screen.getByRole('radio', { name: /存在则更新/ })).toBeChecked();
});
