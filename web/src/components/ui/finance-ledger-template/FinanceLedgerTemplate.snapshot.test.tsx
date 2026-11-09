import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { FinanceLedgerTemplate } from './FinanceLedgerTemplate';

type Row = { id: string; name: string };
type Result = {
  data: Row[];
  total: number;
  summary: { metric: string };
  success?: boolean;
  matchedSummary: {
    count: number;
    amounts: { direction: string; currency: string; amount: string }[];
  };
};
function page(name: string, amount: string): Result {
  return {
    data: [{ id: name, name }],
    total: 20,
    summary: { metric: `${name}-${amount}` },
    matchedSummary: {
      count: 20,
      amounts: [{ direction: '', currency: 'USD', amount }],
    },
  };
}
const config = {
  groups: [{ direction: '', label: '金额' }],
  projectRow: () => ({ direction: '', currency: 'USD', amount: '1' }),
};
function setup(request: () => Promise<Result>) {
  const actionRef = { current: undefined as ActionType | undefined };
  const accepted = vi.fn();
  const download = vi
    .spyOn(URL, 'createObjectURL')
    .mockReturnValue('blob:ledger');
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
  function Harness() {
    const [metric, setMetric] = useState('初始未加载');
    return (
      <App>
        <FinanceLedgerTemplate<Row, Record<string, unknown>, { metric: string }>
          actionRef={actionRef}
          columns={[{ title: '名称', dataIndex: 'name' }]}
          exportColumns={[{ header: '名称', value: (row) => row.name }]}
          request={request}
          search={false}
          summaryConfig={config}
          metricCards={[{ key: 'metric', title: '业务指标', value: metric }]}
          onAcceptedSnapshot={(snapshot) => {
            accepted(snapshot);
            setMetric(snapshot.summary?.metric ?? '缺少业务指标');
          }}
        />
      </App>
    );
  }
  render(<Harness />);
  return { actionRef, accepted, download };
}

describe('财务台账官方采纳快照', () => {
  it('首次请求失败使用中文提示，不显示成功零汇总或程序异常文本', async () => {
    const request = vi
      .fn()
      .mockRejectedValue(new TypeError('Cannot read properties of undefined'));
    const { accepted } = setup(request);
    await waitFor(() =>
      expect(screen.getByText('财务列表加载失败，请稍后重试')).toBeVisible(),
    );
    expect(screen.getByRole('status', { name: '金额合计' })).toHaveTextContent(
      '汇总尚未加载',
    );
    expect(
      screen.queryByText(/Cannot read properties/),
    ).not.toBeInTheDocument();
    expect(accepted).not.toHaveBeenCalled();
    vi.restoreAllMocks();
  });

  it('列表独立处理请求错误时保留服务端中文原因', async () => {
    const request = vi.fn().mockRejectedValue(
      Object.assign(new Error('Request failed with status code 403'), {
        response: { data: { message: '当前公司无权读取账单' } },
      }),
    );
    const { accepted } = setup(request);
    await waitFor(() =>
      expect(screen.getByText('当前公司无权读取账单')).toBeVisible(),
    );
    expect(
      screen.queryByText('财务列表加载失败，请稍后重试'),
    ).not.toBeInTheDocument();
    expect(accepted).not.toHaveBeenCalled();
    vi.restoreAllMocks();
  });

  it('B先返回后A迟到，表格、顶部、筛选底栏与导出均保持B', async () => {
    let finishA!: (value: Result) => void;
    const pending = new Promise<Result>((resolve) => {
      finishA = resolve;
    });
    const request = vi
      .fn()
      .mockReturnValueOnce(pending)
      .mockResolvedValue(page('新B', '200'));
    const { actionRef, accepted, download } = setup(request);
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
    expect(screen.queryByText('初始未加载')).not.toBeInTheDocument();
    await act(async () => {
      await actionRef.current?.reload();
    });
    await screen.findByText('新B');
    expect(screen.getByText('新B-200')).toBeVisible();
    expect(screen.getByRole('status', { name: '金额合计' })).toHaveTextContent(
      '200.00 USD',
    );
    await act(async () => {
      finishA(page('旧A', '100'));
    });
    expect(screen.queryByText('旧A')).not.toBeInTheDocument();
    expect(screen.queryByText('旧A-100')).not.toBeInTheDocument();
    expect(accepted).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: /导出当前页/ }));
    expect(await (download.mock.calls[0][0] as Blob).text()).toContain('新B');
    vi.restoreAllMocks();
  });

  it('失败结果携带金额也不覆盖上次成功快照', async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce(page('成功B', '200'))
      .mockResolvedValue({ ...page('失败A', '999'), success: false });
    const { actionRef, accepted } = setup(request);
    await screen.findByText('成功B');
    await act(async () => {
      await actionRef.current?.reload();
    });
    expect(screen.queryByText('失败A')).not.toBeInTheDocument();
    expect(screen.getByText('成功B-200')).toBeVisible();
    expect(screen.getByRole('status', { name: '金额合计' })).toHaveTextContent(
      '200.00 USD',
    );
    expect(accepted).toHaveBeenCalledOnce();
    vi.restoreAllMocks();
  });
});
