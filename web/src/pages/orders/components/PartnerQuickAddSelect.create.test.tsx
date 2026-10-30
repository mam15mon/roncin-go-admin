import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PartnerAssignmentRole, PartnerRoleType } from '@/enums.generated';
import {
  fillCommissionStaff,
  openDropdown,
  partnerServiceCreatePartner,
  pickSelectOption,
  SupplierHost,
  savePartner,
  searchPartners,
  TestHost,
} from './PartnerQuickAddSelect.fixtures';

describe('PartnerQuickAddSelect', () => {
  it('保存仅提交公司抬头与单一角色，成功后回填字段并注入本地选项', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-1',
        legalName: '新测试单位',
        code: 'P1',
      } as never,
    });
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    const input = await screen.findByLabelText('公司抬头');
    fireEvent.change(input, { target: { value: '  新测试单位  ' } });
    await fillCommissionStaff();
    await savePartner();

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '新测试单位',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
          isCasual: true,
          assignments: [
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
              userId: 'user-a',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
              userId: 'user-b',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
              userId: 'user-c',
            },
          ],
        },
        { signal: expect.any(AbortSignal) },
      ),
    );

    // 回填后选中项显示公司抬头（而非裸 ID），下拉中出现新建单位且与远程结果去重
    await waitFor(() =>
      expect(screen.getAllByTitle('新测试单位').length).toBeGreaterThanOrEqual(
        1,
      ),
    );
    expect(screen.getByTestId('customer-id')).toHaveTextContent('partner-1');
    expect(screen.getByTestId('customer-code')).toHaveTextContent('P1');
    // 弹窗关闭动画期间人员下拉 combobox 仍挂载，主字段按标签精确定位
    fireEvent.mouseDown(screen.getByLabelText('委托单位'));
    await waitFor(() => {
      expect(screen.getAllByText('既有单位').length).toBe(1);
    });
  });

  it('响应缺少伙伴 ID 时报错、弹窗保持打开且输入保留', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {},
    } as never);
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    const input = await screen.findByLabelText('公司抬头');
    fireEvent.change(input, { target: { value: '缺 ID 单位' } });
    await fillCommissionStaff();
    await savePartner();

    expect(
      await screen.findByText('创建结果缺少伙伴 ID，请重试'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('缺 ID 单位');
  });

  it('当前创建请求失败时仍由 QuickCreateModal 展示原始错误', async () => {
    vi.mocked(partnerServiceCreatePartner).mockRejectedValue(
      new Error('无权创建往来单位'),
    );
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '创建被拒绝单位' },
    });
    await fillCommissionStaff();
    await savePartner();

    expect(await screen.findByText('无权创建往来单位')).toBeInTheDocument();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('创建被拒绝单位');
  });

  it('远程返回本地同 ID 时只保留本地名称', async () => {
    searchPartners
      .mockResolvedValueOnce([
        { label: '既有单位', value: 'partner-0', code: 'P0' },
      ])
      .mockResolvedValue([
        { label: '远程旧名称', value: 'partner-1', code: 'P1' },
      ]);
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-1', legalName: '新测试单位', code: 'P1' } as never,
    });
    render(<TestHost />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: /新增 委托单位/ }));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '新测试单位' },
    });
    await fillCommissionStaff();
    await savePartner();

    await waitFor(() =>
      expect(screen.getByTestId('customer-id')).toHaveTextContent('partner-1'),
    );
    // 弹窗关闭动画期间人员下拉 combobox 仍挂载，主字段按标签精确定位
    fireEvent.mouseDown(screen.getByLabelText('委托单位'));
    await waitFor(() => {
      const dropdownOptions = screen
        .getAllByText('新测试单位')
        .filter((element) => element.closest('.ant-select-item-option'));
      expect(dropdownOptions).toHaveLength(1);
      expect(screen.queryByText('远程旧名称')).not.toBeInTheDocument();
    });
    expect(screen.getAllByTitle('新测试单位').length).toBeGreaterThanOrEqual(1);
  });

  it('支持取消单次合作勾选，且散客候选项渲染散客标签', async () => {
    searchPartners.mockResolvedValue([
      { label: '散客货代', value: 'partner-c', code: 'PC', isCasual: true },
      { label: '正式客户', value: 'partner-r', code: 'PR', isCasual: false },
    ]);
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-new',
        legalName: '正式新单位',
        code: 'PN',
        isCasual: false,
      } as never,
    });
    render(<TestHost />);
    fireEvent.mouseDown(screen.getByRole('combobox'));

    // 下拉应渲染散客标签
    await screen.findByText('散客货代');
    expect(screen.getByText('散客')).toBeInTheDocument();

    // 打开快捷新增
    fireEvent.click(screen.getByRole('button', { name: /新增 委托单位/ }));
    const legalNameInput = await screen.findByLabelText('公司抬头');
    fireEvent.change(legalNameInput, { target: { value: '正式新单位' } });

    const checkbox = screen.getByLabelText('单次合作往来单位（散客）');
    expect(checkbox).toBeChecked();
    fireEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();

    await fillCommissionStaff();
    await savePartner();

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '正式新单位',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
          isCasual: false,
          assignments: [
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
              userId: 'user-a',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
              userId: 'user-b',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
              userId: 'user-c',
            },
          ],
        },
        { signal: expect.any(AbortSignal) },
      ),
    );
  });

  it('客户快建未选或部分选择提成岗位时均可保存，仅提交已选岗位', async () => {
    vi.mocked(partnerServiceCreatePartner)
      .mockResolvedValueOnce({
        data: { id: 'partner-bare', legalName: '缺责任人单位', code: 'PB' },
      } as never)
      .mockResolvedValueOnce({
        data: { id: 'partner-partial', legalName: '部分责任人单位' },
      } as never);

    // 第一轮：完全不选提成岗位，保存不再被拦截。
    const { unmount } = render(<TestHost />);
    await openDropdown();
    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '缺责任人单位' },
    });
    await savePartner();

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '缺责任人单位',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
          isCasual: true,
          assignments: [],
        },
        { signal: expect.any(AbortSignal) },
      ),
    );
    await waitFor(() =>
      expect(screen.getByTestId('customer-id')).toHaveTextContent(
        'partner-bare',
      ),
    );
    unmount();

    // 第二轮：仅选业务人员，只提交已选岗位。
    render(<TestHost />);
    await openDropdown();
    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '部分责任人单位' },
    });
    const comboboxes = ['业务人员', '操作人员', '客服人员'].map((label) =>
      screen.getByLabelText(label),
    );
    expect(
      screen
        .getByLabelText('公司抬头')
        .closest('[role="dialog"]')
        ?.querySelectorAll('[role="combobox"]'),
    ).toHaveLength(3);
    await pickSelectOption(comboboxes[0], '张三 · 公司');
    await savePartner();

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenLastCalledWith(
        {
          legalName: '部分责任人单位',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
          isCasual: true,
          assignments: [
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
              userId: 'user-a',
            },
          ],
        },
        { signal: expect.any(AbortSignal) },
      ),
    );
  });

  it('非客户角色快建不要求提成责任岗位', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-s1', legalName: '新订舱代理', code: 'PS1' } as never,
    });
    render(<SupplierHost />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('新增 订舱代理'));

    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '新订舱代理' },
    });
    expect(screen.queryByLabelText('业务人员')).not.toBeInTheDocument();
    await savePartner();

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '新订舱代理',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
              enabled: true,
            },
          ],
          isCasual: true,
          assignments: undefined,
        },
        { signal: expect.any(AbortSignal) },
      ),
    );
  });
});
