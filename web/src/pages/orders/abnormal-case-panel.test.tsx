import type { ProColumns } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { createRef, type ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderAbnormalCaseStatus } from '@/enums.generated';
import { masterDataServiceSearchOptions } from '@/services/roncin/masterDataService';
import {
  orderAbnormalCaseServiceListAbnormalCases,
  orderAbnormalCaseServiceRemoveAbnormalCase,
  orderAbnormalCaseServiceResolveAbnormalCase,
} from '@/services/roncin/orderAbnormalCaseService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';

const tableState = vi.hoisted(() => ({ reload: vi.fn() }));

vi.mock('@/components/ui/column-settings', () => ({
  useColumnSettings: ({
    columns,
  }: {
    columns: ProColumns<API.OrderAbnormalCase>[];
  }) => ({ columns }),
}));
vi.mock('@/components/ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/components/ui')>();
  const { useEffect, useState } = await import('react');
  return {
    ...actual,
    ResponsiveProTable: ({
      columns,
      request,
      toolBarRender,
      actionRef,
    }: {
      columns: ProColumns<API.OrderAbnormalCase>[];
      request: () => Promise<{ data?: API.OrderAbnormalCase[] }>;
      toolBarRender: () => ReactNode[];
      actionRef: { current: unknown };
    }) => {
      const [rows, setRows] = useState<API.OrderAbnormalCase[]>([]);
      useEffect(() => {
        actionRef.current = { reload: tableState.reload };
        void request().then((response) => setRows(response.data || []));
      }, [actionRef, request]);
      return (
        <div>
          {toolBarRender()}
          {rows.map((row) => (
            <div key={row.id}>
              {columns.map((column, index) => (
                <div key={String(column.key || column.dataIndex || index)}>
                  {
                    column.render?.(
                      '-',
                      row,
                      index,
                      {} as never,
                      {} as never,
                    ) as ReactNode
                  }
                </div>
              ))}
            </div>
          ))}
        </div>
      );
    },
  };
});

vi.mock('@/services/roncin/orderAbnormalCaseService', () => ({
  orderAbnormalCaseServiceListAbnormalCases: vi.fn(),
  orderAbnormalCaseServiceMarkAbnormalCase: vi.fn(),
  orderAbnormalCaseServiceRemoveAbnormalCase: vi.fn(),
  orderAbnormalCaseServiceResolveAbnormalCase: vi.fn(),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceSearchOptions: vi.fn(),
}));

const listCases = vi.mocked(orderAbnormalCaseServiceListAbnormalCases);
const resolveCase = vi.mocked(orderAbnormalCaseServiceResolveAbnormalCase);
const removeCase = vi.mocked(orderAbnormalCaseServiceRemoveAbnormalCase);
const listTypes = vi.mocked(masterDataServiceSearchOptions);

function renderPanel(
  props: Partial<React.ComponentProps<typeof AbnormalCasePanel>> = {},
) {
  const ref = createRef<AbnormalCasePanelRef>();
  renderWithApp(
    <AbnormalCasePanel
      ref={ref}
      canRead
      canCreate={false}
      canResolve={false}
      canDelete={false}
      {...props}
    />,
  );
  act(() => ref.current?.open({ id: 'order-1', orderNo: 'SE001' }));
  return ref;
}

describe('订单异常独立权限与历史名称', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listCases.mockResolvedValue({
      data: [
        {
          id: 'case-1',
          abnormalCaseId: 'disabled-type-uuid',
          abnormalCaseName: '已停用的货损类型',
          status: OrderAbnormalCaseStatus.ORDER_ABNORMAL_CASE_STATUS_ACTIVE,
          markedAt: '2026-10-23T08:30:00Z',
        },
      ],
    });
    listTypes.mockResolvedValue({
      data: [{ id: 'type-2', name: '新异常类型', code: 'EX2' }],
    });
  });

  it('仅查看权限可以打开并读取历史类型名称，不加载登记候选', async () => {
    renderPanel();
    expect(await screen.findByText('已停用的货损类型')).toBeVisible();
    expect(screen.queryByText('disabled-type-uuid')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '标记异常' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '解决' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '移除' }),
    ).not.toBeInTheDocument();
    expect(listTypes).not.toHaveBeenCalled();
    expect(listCases).toHaveBeenCalledWith({ orderId: 'order-1' });
  });

  it('仅解决权限可办理，失败保留确认，成功才刷新', async () => {
    resolveCase.mockResolvedValueOnce(undefined as never);
    resolveCase.mockResolvedValueOnce({ data: { id: 'case-1' } });
    renderPanel({ canResolve: true });
    fireEvent.click(await screen.findByRole('button', { name: '解决' }));
    await act(async () => {
      fireEvent.click(await screen.findByText('确 定'));
    });
    await waitFor(() => expect(resolveCase).toHaveBeenCalledTimes(1));
    expect(tableState.reload).not.toHaveBeenCalled();
    expect(screen.getByText('确定解决该异常？')).toBeInTheDocument();
    await act(async () => {
      fireEvent.click(screen.getByText('确 定'));
    });
    await waitFor(() => expect(tableState.reload).toHaveBeenCalledTimes(1));
    expect(
      screen.queryByRole('button', { name: '移除' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '标记异常' }),
    ).not.toBeInTheDocument();
  });

  it('仅删除权限显示移除并调用原删除接口', async () => {
    removeCase.mockResolvedValue({ success: true });
    renderPanel({ canDelete: true });
    fireEvent.click(await screen.findByRole('button', { name: '移除' }));
    fireEvent.click(await screen.findByText('确 定'));
    await waitFor(() =>
      expect(removeCase).toHaveBeenCalledWith({
        orderId: 'order-1',
        id: 'case-1',
      }),
    );
    await waitFor(() => expect(tableState.reload).toHaveBeenCalledTimes(1));
    expect(
      screen.queryByRole('button', { name: '解决' }),
    ).not.toBeInTheDocument();
  });

  it('有登记权限也仅在打开登记表单时加载候选，不依赖历史名称候选', async () => {
    renderPanel({ canCreate: true });
    await screen.findByText('已停用的货损类型');
    expect(listTypes).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: /标记异常/ }));
    await waitFor(() =>
      expect(listTypes).toHaveBeenCalledWith({
        kind: 10,
        keyword: undefined,
        page: 1,
        pageSize: 50,
      }),
    );
    fireEvent.mouseDown(await screen.findByRole('combobox'));
    expect(await screen.findByText('新异常类型 (EX2)')).toBeInTheDocument();
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '新异常' },
    });
    await waitFor(() =>
      expect(listTypes).toHaveBeenCalledWith(
        expect.objectContaining({ keyword: '新异常' }),
      ),
    );
    expect(
      screen.queryByRole('button', { name: '解决' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '移除' }),
    ).not.toBeInTheDocument();
  });

  it('业务锁定只阻止写入，允许查看并展示原因', async () => {
    renderPanel({
      canCreate: true,
      canResolve: true,
      canDelete: true,
      businessWriteBlockedReason: '订单已锁定，请先解锁',
    });
    expect(await screen.findByText('已停用的货损类型')).toBeVisible();
    expect(screen.getByText('订单已锁定，请先解锁')).toBeVisible();
    for (const name of ['标记异常', '解决', '移除']) {
      expect(
        screen.getByRole('button', { name: new RegExp(name) }),
      ).toBeDisabled();
    }
    expect(listTypes).not.toHaveBeenCalled();
    expect(resolveCase).not.toHaveBeenCalled();
    expect(removeCase).not.toHaveBeenCalled();
  });

  it('没有查看权限时命令式打开无效，不请求异常数据', () => {
    renderPanel({ canRead: false, canCreate: true });
    expect(screen.queryByText('订单异常协同 - SE001')).not.toBeInTheDocument();
    expect(listCases).not.toHaveBeenCalled();
    expect(listTypes).not.toHaveBeenCalled();
  });
});
