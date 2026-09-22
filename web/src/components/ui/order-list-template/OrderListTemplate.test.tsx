import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderListSearchFilter } from './OrderListSearchFilter';
import { OrderListTemplate } from './OrderListTemplate';
import type { OrderListItem } from './types';

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

describe('OrderListTemplate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('不渲染面包屑和状态切签，并正确渲染标题、工具栏与数据表格', async () => {
    const mockQuery = vi.fn().mockResolvedValue({
      data: [
        {
          id: 'order-1',
          orderNo: 'ORD202608260001',
          customerName: '阿里巴巴国际站',
          customerReferenceNo: 'CUST-REF-888',
          vesselVoyage: 'COSCO STAR / 024W',
          masterBlNo: 'COSU632189472',
          originPortName: '上海港',
          originPortCode: 'CNSHA',
          destinationPortName: '洛杉矶港',
          destinationPortCode: 'USLAX',
          containerSummary: '2×40HQ',
          grossWeightKg: 18500,
          volumeCbm: 68,
          statusName: '已配载',
          abnormalLevel: 'normal',
        } as OrderListItem,
      ],
      total: 1,
      success: true,
    });

    const { container } = render(
      <OrderListTemplate
        title="海运出口订单"
        subTitle="海运整箱与拼箱出口业务调度"
        queryOrders={mockQuery}
        onCreateOrder={vi.fn()}
      />,
    );

    expect(screen.getByText('海运出口订单')).toBeInTheDocument();
    expect(container.querySelector('.ant-breadcrumb')).not.toBeInTheDocument();
    expect(container.querySelector('.ant-tabs')).not.toBeInTheDocument();
    expect(screen.queryByText('全部订单')).not.toBeInTheDocument();
    expect(screen.queryByText('待订舱')).not.toBeInTheDocument();
    expect(screen.getByText('新增海运出口订单')).toBeInTheDocument();
    expect(screen.getByText('批量操作')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('ORD202608260001')).toBeInTheDocument();
      expect(screen.getByText('阿里巴巴国际站')).toBeInTheDocument();
      expect(screen.getByText('COSCO STAR / 024W')).toBeInTheDocument();
      expect(screen.getByText('COSU632189472')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /更多/ }));
    expect(screen.queryByText('流转状态')).not.toBeInTheDocument();
  });
});

describe('OrderListSearchFilter', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('提交复合单号类型和关键词', async () => {
    const onSearch = vi.fn();
    render(<OrderListSearchFilter onSearch={onSearch} onReset={vi.fn()} />);

    fireEvent.change(screen.getByPlaceholderText('输入单号'), {
      target: { value: 'SE202608280001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));

    await waitFor(() => {
      expect(onSearch).toHaveBeenCalledWith(
        expect.objectContaining({
          numberType: 'order',
          numberKeyword: 'SE202608280001',
        }),
      );
    });
  });

  it('提供客户业务号与 Booking No 筛选类型', () => {
    render(<OrderListSearchFilter onSearch={vi.fn()} onReset={vi.fn()} />);

    const numberTypeSelect = document.querySelector('#numberType');
    expect(numberTypeSelect).not.toBeNull();
    fireEvent.mouseDown(numberTypeSelect as HTMLElement);

    expect(screen.getByText('客户业务号')).toBeInTheDocument();
    expect(screen.getByText('Booking No')).toBeInTheDocument();
  });

  it('展开全量筛选并按服务端方式加载动态候选项', async () => {
    const loadPorts = vi.fn().mockResolvedValue([]);
    const loadPartners = vi.fn().mockResolvedValue([]);
    const loadCarriers = vi.fn().mockResolvedValue([]);
    const loadPersonnel = vi.fn().mockResolvedValue([]);
    render(
      <OrderListSearchFilter
        onSearch={vi.fn()}
        onReset={vi.fn()}
        options={{ loadPorts, loadPartners, loadCarriers, loadPersonnel }}
      />,
    );

    await waitFor(() => {
      expect(loadPorts).toHaveBeenCalled();
      expect(loadPartners).toHaveBeenCalled();
    });
    fireEvent.click(screen.getByRole('button', { name: /展开更多筛选/ }));

    expect(screen.getByText('创建时间')).toBeInTheDocument();
    expect(screen.getByText('ETA（预计到港时间）')).toBeInTheDocument();
    expect(screen.getByText('订单状态时间')).toBeInTheDocument();
    expect(screen.getByText('订单锁定时间')).toBeInTheDocument();
    expect(screen.getByText('船公司')).toBeInTheDocument();
    expect(screen.getByText('操作人员')).toBeInTheDocument();
    expect(screen.getByText('业务人员')).toBeInTheDocument();
    expect(screen.getByText('客服人员')).toBeInTheDocument();
    expect(screen.getByText('订单创建人员')).toBeInTheDocument();
    expect(screen.getByText('订单标签')).toBeInTheDocument();

    await waitFor(() => {
      expect(loadPorts).toHaveBeenCalled();
      expect(loadPartners).toHaveBeenCalled();
      expect(loadCarriers).toHaveBeenCalled();
      expect(loadPersonnel).toHaveBeenCalledTimes(1);
    });
  });
});
