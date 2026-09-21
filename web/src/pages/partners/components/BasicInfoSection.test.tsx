import { ProForm } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import BasicInfoSection from './BasicInfoSection';

const baseProps = {
  collapsed: false,
  onCollapseChange: vi.fn(),
  roleLabel: '客户',
  userSelectOptions: [
    { label: '张三', value: 'user-a' },
    { label: '李四', value: 'user-b' },
    { label: '王五', value: 'user-c' },
  ],
  aliases: [],
  onTianyanchaVerify: vi.fn(),
};

const COMMISSION_STAFF_LABELS = ['操作人员', '业务人员', '客服人员'] as const;

/** 打开指定下拉并点击候选项；过滤掉已关闭下拉残留的隐藏 DOM（antd 关闭后选项仍挂载）。 */
async function pickSelectOption(combobox: Element, name: string) {
  fireEvent.mouseDown(combobox);
  const option = await waitFor(() => {
    const candidates = screen.getAllByText(name).filter((element) => {
      const dropdown = element.closest('.ant-select-dropdown');
      return (
        dropdown !== null &&
        !dropdown.className.includes('-hidden') &&
        !dropdown.className.includes('ant-slide-up-leave')
      );
    });
    if (candidates.length === 0) {
      throw new Error(`选项 ${name} 尚未出现在打开的下拉中`);
    }
    return candidates[0];
  });
  fireEvent.click(option);
  // 等待下拉确认关闭（离开动画类已生效），下一次挑选才不会被残留 DOM 干扰。
  await waitFor(() =>
    expect(combobox).toHaveAttribute('aria-expanded', 'false'),
  );
}

function Host({
  roleType,
  roleTypes,
  onFinish,
}: {
  roleType: number;
  roleTypes: number[];
  onFinish: (values: unknown) => void;
}) {
  return (
    <ProForm
      layout="horizontal"
      submitter={false}
      onFinish={onFinish}
      initialValues={{
        legalName: '测试客户有限公司',
        roleTypes,
      }}
    >
      <BasicInfoSection {...baseProps} roleType={roleType} />
      <button type="submit">提交</button>
    </ProForm>
  );
}

function expectCommissionStaffRequired(required: boolean) {
  for (const label of COMMISSION_STAFF_LABELS) {
    const labelElement = screen.getByText(label).closest('label');
    expect(labelElement).not.toBeNull();
    if (required) {
      expect(labelElement).toHaveClass('ant-form-item-required');
    } else {
      expect(labelElement).not.toHaveClass('ant-form-item-required');
    }
  }
}

describe('BasicInfoSection 责任人员分配矩阵', () => {
  it('客户档案要求业务/操作/客服人员必填并显示星号', async () => {
    const onFinish = vi.fn();
    render(
      <Host
        roleType={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
        roleTypes={[PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER]}
        onFinish={onFinish}
      />,
    );

    expectCommissionStaffRequired(true);

    fireEvent.click(screen.getByRole('button', { name: '提交' }));
    expect(await screen.findByText('请选择业务人员')).toBeInTheDocument();
    expect(screen.getByText('请选择操作人员')).toBeInTheDocument();
    expect(screen.getByText('请选择客服人员')).toBeInTheDocument();
    expect(onFinish).not.toHaveBeenCalled();
  });

  it('客户档案补全三名责任人员后可提交', async () => {
    const onFinish = vi.fn();
    render(
      <Host
        roleType={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
        roleTypes={[PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER]}
        onFinish={onFinish}
      />,
    );

    const fillStaff = async (label: string, name: string) => {
      await pickSelectOption(screen.getByLabelText(label), name);
    };
    await fillStaff('业务人员', '张三');
    await fillStaff('操作人员', '李四');
    await fillStaff('客服人员', '王五');

    fireEvent.click(screen.getByRole('button', { name: '提交' }));
    await waitFor(() => expect(onFinish).toHaveBeenCalledTimes(1));
  });

  it('供应商档案不要求提成责任人员', async () => {
    const onFinish = vi.fn();
    render(
      <Host
        roleType={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
        roleTypes={[PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER]}
        onFinish={onFinish}
      />,
    );

    expectCommissionStaffRequired(false);

    fireEvent.click(screen.getByRole('button', { name: '提交' }));
    await waitFor(() => expect(onFinish).toHaveBeenCalledTimes(1));
  });
});
