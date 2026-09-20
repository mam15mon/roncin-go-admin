import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import React, { useEffect, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  organizations: vi.fn(),
  candidates: vi.fn(),
  preview: vi.fn(),
  create: vi.fn(),
  accounts: vi.fn(),
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: any) => {
    const [data, setData] = useState<any[]>([]);
    useEffect(() => {
      if (props.request)
        void props
          .request({ current: 1, pageSize: 20 })
          .then((r: any) => setData(r.data || []));
    }, []);
    return (
      <div>
        {data.map((row) => (
          <button
            key={row.id}
            type="button"
            onClick={() => props.rowSelection?.onChange([row.id], [row])}
          >
            {row.feeName}
          </button>
        ))}
      </div>
    );
  },
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({ hasAction: () => true }),
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceOrganizationOptions: mocks.organizations,
  settlementServiceListBillCreationCandidates: mocks.candidates,
  settlementServicePreviewBillBatch: mocks.preview,
  settlementServiceCreateBillBatch: mocks.create,
  settlementServiceConfirmBillBatch: vi.fn(),
  settlementServiceListBillSettlementAccountCandidates: mocks.accounts,
}));

vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([
    { label: 'CNY - 人民币', value: 'CNY' },
    { label: 'USD - 美元', value: 'USD' },
  ]),
}));

import BillCreationWorkbench from './BillCreationWorkbench';

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/** 防抖窗口断言仍用真实定时器；包进 act 让窗口期落地的预览/账户回调在 act 内更新状态。 */
const sleepInAct = async (ms: number) => {
  await act(async () => {
    await sleep(ms);
  });
};

function resetWorkbenchMocks() {
  mocks.organizations.mockReset();
  mocks.candidates.mockReset();
  mocks.preview.mockReset();
  mocks.create.mockReset();
  mocks.accounts.mockReset();
  mocks.create.mockResolvedValue({ data: { id: 'batch-1', bills: [] } });
  mocks.organizations.mockResolvedValue({
    data: [
      { id: 'A', name: '公司A' },
      { id: 'B', name: '公司B' },
    ],
  });
  mocks.candidates.mockImplementation(({ organizationId }) =>
    Promise.resolve({
      data: [
        {
          id: `fee-${organizationId}`,
          feeName: `费用${organizationId}`,
          status: 2,
        },
      ],
    }),
  );
  mocks.accounts.mockResolvedValue({
    data: [
      {
        id: 'account-default',
        name: '默认结算账户',
        bankName: '测试银行',
        accountNo: '6222',
        currency: 'CNY',
        isDefault: true,
      },
      {
        id: 'account-manual',
        name: '次要手工账户',
        bankName: '测试银行2',
        accountNo: '6333',
        currency: 'CNY',
        isDefault: false,
      },
    ],
  });
}

describe('BillCreationWorkbench 规模压测与异常阻断', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
  });

  it('14 家单位 40 笔费用：层级导航、groupKey 状态保持、分币种汇总与整批提交', async () => {
    const chineseUnits = [
      '一',
      '二',
      '三',
      '四',
      '五',
      '六',
      '七',
      '八',
      '九',
      '十',
      '十一',
      '十二',
      '十三',
      '十四',
    ];
    const groups: any[] = [];
    const addGroup = (
      unitIndex: number,
      currency: string,
      feeCount: number,
      orderNo: string,
    ) => {
      const unitName = `第${chineseUnits[unitIndex]}单位`;
      const key = `g-${unitIndex}-${currency}-${orderNo}`;
      groups.push({
        groupKey: key,
        settlementPartyId: `partner-${unitIndex}`,
        settlementPartyName: unitName,
        direction: 'RECEIVABLE',
        currency,
        baseCurrency: 'CNY',
        orderNo,
        totalAmount: `${feeCount * 100}.00`,
        baseCurrencyAmount: `${feeCount * 100}.00`,
        estimatedInvoiceCurrency: currency,
        estimatedInvoiceRate: '1',
        estimatedInvoiceAmount: `${feeCount * 100}.00`,
        configurationComplete: true,
        fees: Array.from({ length: feeCount }, (_, index) => ({
          id: `${key}-fee-${index}`,
          feeName: `${unitName}费用${index + 1}`,
          status: 2,
        })),
      });
    };
    // 10 家单位各 1 张 CNY 叶子（2 笔费用），共 20 笔。
    for (let unitIndex = 0; unitIndex < 10; unitIndex += 1) {
      addGroup(unitIndex, 'CNY', 2, 'SE0001');
    }
    // 第 11、12 家单位各有 CNY + USD 两张叶子；同一单位的不同叶子使用不同订单号，
    // 便于按卡片订单标题定位唯一叶子。
    addGroup(10, 'CNY', 2, 'SE1101');
    addGroup(10, 'USD', 2, 'SE1102');
    addGroup(11, 'CNY', 2, 'SE1201');
    addGroup(11, 'USD', 2, 'SE1202');
    // 第 13 家单位两张 USD 叶子对应不同订单。
    addGroup(12, 'USD', 2, 'SE1301');
    addGroup(12, 'USD', 3, 'SE1302');
    // 第 14 家单位一张 EUR 叶子承接剩余 7 笔费用。
    addGroup(13, 'EUR', 7, 'SE1401');
    expect(groups).toHaveLength(17);
    expect(groups.reduce((sum, group) => sum + group.fees.length, 0)).toBe(40);

    mocks.preview.mockResolvedValue({
      previewToken: 'token-scale',
      data: groups,
    });
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(
      await screen.findByDisplayValue('第一单位', {}, { timeout: 10000 }),
    ).toBeInTheDocument();
    // 17 个叶子全部完成默认账户回填与防抖重预览。
    await waitFor(() => expect(mocks.accounts).toHaveBeenCalledTimes(17), {
      timeout: 10000,
    });
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2), {
      timeout: 10000,
    });
    await sleepInAct(700);

    // 批次汇总按币种分别列示，禁止跨币种相加。
    expect(screen.getByText('2400 CNY')).toBeInTheDocument();
    expect(screen.getByText('900 USD')).toBeInTheDocument();
    expect(screen.getByText('700 EUR')).toBeInTheDocument();
    expect(screen.queryByText(/4000/)).not.toBeInTheDocument();
    // 40 笔费用全部落在某个叶子上。
    const feeTagSum = screen
      .queryAllByText(/^\d+ 笔费用$/)
      .reduce(
        (sum, node) => sum + Number(node.textContent?.match(/\d+/)?.[0]),
        0,
      );
    expect(feeTagSum).toBe(40);

    // 层级导航：结算单位 → 币种 → 订单 定位第 11 家单位的 USD 叶子。
    fireEvent.click(screen.getByText('第十一单位 · 200.00 USD'));
    const usdCard = screen
      .getByText('订单 SE1102')
      .closest('.ant-card') as HTMLElement;
    await waitFor(() =>
      expect(within(usdCard).getByDisplayValue('第十一单位')).toBeVisible(),
    );
    fireEvent.change(within(usdCard).getByDisplayValue('第十一单位'), {
      target: { value: '抬头十一USD' },
    });

    // 切换到其他叶子再切回，groupKey 草稿不丢失（不按数组下标串状态）。
    fireEvent.click(screen.getByText('第一单位 · 200.00 CNY'));
    await waitFor(() =>
      expect(screen.getByDisplayValue('第一单位')).toBeVisible(),
    );
    expect(screen.getByDisplayValue('抬头十一USD')).not.toBeVisible();
    fireEvent.click(screen.getByText('第十一单位 · 200.00 USD'));
    await waitFor(() =>
      expect(screen.getByDisplayValue('抬头十一USD')).toBeVisible(),
    );

    // 提交的 groups 数量与服务端预览叶子数一致。
    const createBtn = await screen.findByRole('button', {
      name: /原子生成 17 张账单/,
    });
    await waitFor(() => expect(createBtn).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(createBtn);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    const createInput = mocks.create.mock.calls[0][0];
    expect(createInput.groups).toHaveLength(17);
    expect(createInput.previewToken).toBe('token-scale');
    expect(
      new Set(createInput.groups.map((group: any) => group.groupKey)).size,
    ).toBe(17);
  }, 45000);

  it('任一叶子缺少结算账户时阻断整批创建并自动跳到该叶子', async () => {
    const groups: any[] = [
      {
        groupKey: 'g-ok',
        settlementPartyId: 'partner-ok',
        settlementPartyName: '配置完整单位',
        direction: 'RECEIVABLE',
        currency: 'CNY',
        baseCurrency: 'CNY',
        totalAmount: '100.00',
        baseCurrencyAmount: '100.00',
        estimatedInvoiceCurrency: 'CNY',
        estimatedInvoiceRate: '1',
        estimatedInvoiceAmount: '100.00',
        configurationComplete: true,
        fees: [],
      },
      {
        groupKey: 'g-missing',
        settlementPartyId: 'partner-missing',
        settlementPartyName: '缺账户单位',
        direction: 'RECEIVABLE',
        currency: 'EUR',
        baseCurrency: 'CNY',
        totalAmount: '800.00',
        baseCurrencyAmount: '800.00',
        estimatedInvoiceCurrency: 'EUR',
        estimatedInvoiceRate: '1',
        estimatedInvoiceAmount: '800.00',
        configurationComplete: true,
        fees: [],
      },
    ];
    mocks.preview.mockResolvedValue({
      previewToken: 'token-missing',
      data: groups,
    });
    mocks.accounts.mockImplementation(({ settlementPartyId }) =>
      settlementPartyId === 'partner-missing'
        ? Promise.resolve({ data: [] })
        : Promise.resolve({
            data: [
              {
                id: 'account-default',
                name: '默认结算账户',
                bankName: '测试银行',
                accountNo: '6222',
                currency: 'CNY',
                isDefault: true,
              },
            ],
          }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('配置完整单位')).toBeInTheDocument();
    await waitFor(() =>
      expect(mocks.accounts).toHaveBeenCalledWith(
        expect.objectContaining({ settlementPartyId: 'partner-missing' }),
      ),
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await sleepInAct(700);

    fireEvent.click(screen.getByRole('button', { name: /原子生成 2 张账单/ }));
    await waitFor(() =>
      expect(
        screen.getByText('请先补齐首个标记叶子的日期和结算账户'),
      ).toBeInTheDocument(),
    );
    // 整批被阻断时自动跳到缺配置叶子，其他叶子不再置前展示。
    await waitFor(() =>
      expect(screen.getByDisplayValue('缺账户单位')).toBeVisible(),
    );
    expect(screen.getByDisplayValue('配置完整单位')).not.toBeVisible();
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
