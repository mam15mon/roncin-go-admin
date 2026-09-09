import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React, { useEffect } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  modalFinish: undefined as undefined | ((values: any) => Promise<boolean>),
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: any) => {
    useEffect(() => {
      void props.request?.({});
    }, [props]);
    return (
      <div>
        {props.headerTitle}
        {props.toolBarRender?.()}
      </div>
    );
  },
  ModalForm: (props: any) => {
    mocks.modalFinish = props.onFinish;
    return props.open ? <div>{props.children}</div> : null;
  },
  ProFormSelect: ({ label }: any) => <span>{label}</span>,
  ProFormSwitch: ({ label }: any) => <span>{label}</span>,
  ProFormText: ({ label }: any) => <span>{label}</span>,
  ProFormTextArea: ({ label }: any) => <span>{label}</span>,
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartnerAccounts: mocks.list,
  partnerServiceCreatePartnerAccount: mocks.create,
  partnerServiceUpdatePartnerAccount: mocks.update,
}));

import AccountsPanel from './AccountsPanel';

describe('AccountsPanel 往来结算账户', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.modalFinish = undefined;
    mocks.list.mockResolvedValue({ data: [] });
  });

  it('不再以客户角色作为账户管理门槛，并展示完整账户建档字段', async () => {
    render(
      <App>
        <AccountsPanel
          partner={{
            id: 'supplier-1',
            legalName: '仅供应商单位',
            roles: [{ type: 2, enabled: true }],
          }}
          canRead
          canCreate
          canUpdate
        />
      </App>,
    );

    expect(await screen.findByText('结算账户列表')).toBeInTheDocument();
    expect(mocks.list).toHaveBeenCalledWith({ partnerId: 'supplier-1' });
    const createButton = screen.getByRole('button', { name: /新增结算账户/ });
    expect(createButton).toBeInTheDocument();
    fireEvent.click(createButton);
    expect(await screen.findByText('账户名称')).toBeInTheDocument();
    expect(screen.getByText('账户户名')).toBeInTheDocument();
    expect(screen.getByText('允许用途')).toBeInTheDocument();
    expect(screen.getByText('设为应收默认账户')).toBeInTheDocument();
    expect(screen.getByText('设为应付默认账户')).toBeInTheDocument();
    expect(screen.queryByText('无需结算账户配置')).not.toBeInTheDocument();
  });

  it('读取、创建与更新入口分别由账户专属权限控制', async () => {
    const { rerender } = render(
      <App>
        <AccountsPanel
          partner={{ id: 'partner-1', legalName: '单位甲' }}
          canRead={false}
          canCreate
          canUpdate
        />
      </App>,
    );
    expect(screen.getByText('暂无结算账户查看权限。')).toBeInTheDocument();
    expect(mocks.list).not.toHaveBeenCalled();

    rerender(
      <App>
        <AccountsPanel
          partner={{ id: 'partner-1', legalName: '单位甲' }}
          canRead
          canCreate={false}
          canUpdate={false}
        />
      </App>,
    );
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(1));
    expect(
      screen.queryByRole('button', { name: /新增结算账户/ }),
    ).not.toBeInTheDocument();
  });

  it('新增时提交完整账户资料与两类默认标记', async () => {
    mocks.create.mockResolvedValue({ data: { id: 'account-1' } });
    render(
      <App>
        <AccountsPanel
          partner={{ id: 'partner-1', legalName: '单位甲' }}
          canRead
          canCreate
          canUpdate
        />
      </App>,
    );
    fireEvent.click(
      await screen.findByRole('button', { name: /新增结算账户/ }),
    );

    await waitFor(() => expect(mocks.modalFinish).toBeTypeOf('function'));
    await act(async () => {
      await mocks.modalFinish?.({
        name: '收付主账户',
        accountHolder: '单位甲有限公司',
        bankName: '测试银行上海分行',
        accountNo: '6222020012345678',
        currency: 'USD',
        swiftCode: 'ICBKCNBS',
        usage: 3,
        isDefaultReceivable: true,
        isDefaultPayable: true,
        enabled: true,
      });
    });

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        { partnerId: 'partner-1' },
        {
          partnerId: 'partner-1',
          account: expect.objectContaining({
            name: '收付主账户',
            accountHolder: '单位甲有限公司',
            bankName: '测试银行上海分行',
            accountNo: '6222020012345678',
            currency: 'USD',
            swiftCode: 'ICBKCNBS',
            usage: 3,
            isDefaultReceivable: true,
            isDefaultPayable: true,
            enabled: true,
          }),
        },
      ),
    );
  });
});
