import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React, { useEffect, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  organizations: vi.fn(),
  candidates: vi.fn(),
  preview: vi.fn(),
  create: vi.fn(),
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
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceOrganizationOptions: mocks.organizations,
  settlementServiceListBillCreationCandidates: mocks.candidates,
  settlementServicePreviewBillBatch: mocks.preview,
  settlementServiceCreateBillBatch: mocks.create,
  settlementServiceConfirmBillBatch: vi.fn(),
}));

import BillCreationWorkbench from './BillCreationWorkbench';

describe('BillCreationWorkbench 建账候选组织范围', () => {
  beforeEach(() => {
    mocks.organizations.mockReset();
    mocks.candidates.mockReset();
    mocks.preview.mockReset();
    mocks.create.mockReset();
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
  });
  it('未选组织不请求，A 慢响应切 B 后不回填且清空选择', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.candidates.mockImplementation(({ organizationId }) =>
      organizationId === 'A'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            data: [{ id: 'fee-B', feeName: '费用B', status: 2 }],
          }),
    );
    render(
      <App>
        <BillCreationWorkbench open onClose={vi.fn()} />
      </App>,
    );
    await waitFor(() =>
      expect(mocks.organizations).toHaveBeenCalledWith({ purpose: 11 }),
    );
    expect(mocks.candidates).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByLabelText('所属公司'));
    fireEvent.click(await screen.findByText('公司A'));
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
      ),
    );
    fireEvent.mouseDown(screen.getByLabelText('所属公司'));
    fireEvent.click(await screen.findByText('公司B'));
    await waitFor(() => {
      expect(
        screen.queryByRole('button', { name: '费用A' }),
      ).not.toBeInTheDocument();
      expect(screen.getByRole('button', { name: '费用B' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.preview).not.toHaveBeenCalled();
    resolveA?.({ data: [{ id: 'fee-A', feeName: '费用A', status: 2 }] });
    await waitFor(() => {
      expect(
        screen.queryByRole('button', { name: '费用A' }),
      ).not.toBeInTheDocument();
      expect(screen.getByRole('button', { name: '费用B' })).toBeInTheDocument();
    });
  });
  it('预览后的对账抬头默认结算单位名称，且不请求 Partner 服务', async () => {
    mocks.preview.mockResolvedValue({
      previewToken: 'token',
      data: [{ groupKey: 'g', settlementPartyName: '结算单位甲', fees: [] }],
    });
    render(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    expect(mocks.preview).toHaveBeenCalledWith(
      expect.objectContaining({ organizationId: 'A' }),
      expect.anything(),
    );
    fireEvent.click(screen.getByRole('button', { name: /原子生成 1 张账单/ }));
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
        expect.anything(),
      ),
    );
  });

  it('固定来源从 A 切到 B 后丢弃迟到的 A 预览', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.preview.mockImplementation(({ organizationId }) =>
      organizationId === 'A'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            previewToken: 'token-b',
            data: [
              {
                groupKey: 'group-b',
                settlementPartyName: '结算单位B',
                fees: [],
              },
            ],
          }),
    );
    const view = render(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
        expect.anything(),
      ),
    );

    view.rerender(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-B']}
          initialOrganizationId="B"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('结算单位B')).toBeInTheDocument();

    resolveA?.({
      previewToken: 'token-a',
      data: [
        {
          groupKey: 'group-a',
          settlementPartyName: '结算单位A',
          fees: [],
        },
      ],
    });
    await waitFor(() => {
      expect(screen.queryByDisplayValue('结算单位A')).not.toBeInTheDocument();
      expect(screen.getByDisplayValue('结算单位B')).toBeInTheDocument();
    });
  });
});
