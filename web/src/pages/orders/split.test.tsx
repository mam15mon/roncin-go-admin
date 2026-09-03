import React from 'react';
import { App } from 'antd';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import SeaOrderSplitPage, { calculateFeeCurrencySummaries } from './split';
import * as changeService from '@/services/roncin/seaOrderChangeService';
import { computeCanonicalSha256 } from '@/utils/hash';

// Mock umi hooks
vi.mock('@umijs/max', () => ({
  useParams: () => ({ id: 'test-order-123' }),
  useAccess: () => ({ canOrder: () => true }),
  history: {
    push: vi.fn(),
  },
}));

describe('SeaOrderSplitPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('渲染拆票工作台并展示守恒差额与分单分配表格', async () => {
    const mockContext: API.SeaOrderSplitContextData = {
      orderId: 'test-order-123',
      orderNo: 'SE20260903001',
      orderVersion: '1',
      currentLinkVersion: '1',
      cargoAllocationVersion: '1',
      documentStructure: 'HOUSE',
      flowStatus: 'BOOKED',
      bookingNotes: '测试订舱备注',
      houseBills: [
        {
          id: 'hb-1',
          houseNo: 'HBL001',
          status: 'DRAFT',
          version: '1',
        },
        {
          id: 'hb-2',
          houseNo: 'HBL002',
          status: 'DRAFT',
          version: '1',
        },
      ],
      containers: [
        {
          id: 'cntr-1',
          containerNo: 'MSCU1234567',
          containerSpecName: '40GP',
          packageCount: 100,
          grossWeightKg: '2000.000',
          volumeCbm: '15.000000',
        },
      ],
      allocations: [
        {
          id: 'alloc-1',
          cargoItemId: 'cargo-1',
          houseBillId: 'hb-1',
          containerId: 'cntr-1',
          packageCount: 60,
          grossWeightKg: '1200.000',
          volumeCbm: '9.000000',
        },
        {
          id: 'alloc-2',
          cargoItemId: 'cargo-1',
          houseBillId: 'hb-2',
          containerId: 'cntr-1',
          packageCount: 40,
          grossWeightKg: '800.000',
          volumeCbm: '6.000000',
        },
      ],
      draftFees: [
        {
          id: 'fee-1',
          feeCode: 'OFT',
          feeName: '海运费',
          direction: 'RECEIVABLE',
          settlementPartyName: '客户A',
          currency: 'USD',
          totalAmount: '1200.00',
        },
      ],
      attachments: [
        {
          id: 'att-1',
          fileName: '订舱单.pdf',
          docType: 'BOOKING_NOTE',
          fileSize: '1024',
        },
      ],
    };

    const mockPreview: API.SeaOrderSplitPreviewData = {
      isValid: true,
      conservationPassed: true,
      baseline: {
        packageCount: 100,
        grossWeightKg: '2000.000',
        volumeCbm: '15.000000',
        houseBillCount: 2,
        containerCount: 1,
        feeCount: 1,
      },
      allocated: {
        packageCount: 100,
        grossWeightKg: '2000.000',
        volumeCbm: '15.000000',
        houseBillCount: 2,
        containerCount: 1,
        feeCount: 1,
      },
      remaining: {
        packageCount: 0,
        grossWeightKg: '0.000',
        volumeCbm: '0.000000',
        houseBillCount: 0,
        containerCount: 0,
        feeCount: 0,
      },
      results: [
        {
          clientResultKey: 'res-origin',
          resultRole: 'ORIGINAL',
          packageCount: 60,
          grossWeightKg: '1200.000',
          volumeCbm: '9.000000',
          houseBillCount: 1,
          feeCount: 1,
        },
        {
          clientResultKey: 'res-new-1',
          resultRole: 'CREATED',
          packageCount: 40,
          grossWeightKg: '800.000',
          volumeCbm: '6.000000',
          houseBillCount: 1,
          feeCount: 0,
        },
      ],
    };

    vi.spyOn(changeService, 'seaOrderChangeServiceGetSeaOrderSplitContext').mockResolvedValue({
      data: mockContext,
    } as Awaited<ReturnType<typeof changeService.seaOrderChangeServiceGetSeaOrderSplitContext>>);

    vi.spyOn(changeService, 'seaOrderChangeServicePreviewSeaOrderSplit').mockResolvedValue({
      data: mockPreview,
    } as Awaited<ReturnType<typeof changeService.seaOrderChangeServicePreviewSeaOrderSplit>>);

    render(
      <App>
        <SeaOrderSplitPage />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText('海运出口拆票工作台')).toBeInTheDocument();
      expect(screen.getByText('SE20260903001')).toBeInTheDocument();
      expect(screen.getByText('HBL001')).toBeInTheDocument();
      expect(screen.getByText('HBL002')).toBeInTheDocument();
      expect(screen.getByText('海运费')).toBeInTheDocument();
      expect(screen.getByText('订舱单.pdf')).toBeInTheDocument();
      expect(screen.getByText('确认执行拆票')).toBeInTheDocument();
    });
  });

  it('守恒未满足时展示未满足标签并禁用提交按钮', async () => {
    const mockContext: API.SeaOrderSplitContextData = {
      orderId: 'test-order-123',
      orderNo: 'SE20260903001',
      orderVersion: '1',
      currentLinkVersion: '1',
      cargoAllocationVersion: '1',
      documentStructure: 'HOUSE',
      flowStatus: 'BOOKED',
      houseBills: [],
      containers: [],
      allocations: [],
      draftFees: [],
      attachments: [],
    };

    const mockFailedPreview: API.SeaOrderSplitPreviewData = {
      isValid: false,
      conservationPassed: false,
      validationErrors: [
        {
          reason: 'SEA_ORDER_SPLIT_CONSERVATION_FAILED',
          message: '件数守恒校验未通过：仍有 40 件未分配',
        },
      ],
      baseline: {
        packageCount: 100,
        grossWeightKg: '2000.000',
        volumeCbm: '15.000000',
      },
      allocated: {
        packageCount: 60,
        grossWeightKg: '1200.000',
        volumeCbm: '9.000000',
      },
      remaining: {
        packageCount: 40,
        grossWeightKg: '800.000',
        volumeCbm: '6.000000',
      },
    };

    vi.spyOn(changeService, 'seaOrderChangeServiceGetSeaOrderSplitContext').mockResolvedValue({
      data: mockContext,
    } as Awaited<ReturnType<typeof changeService.seaOrderChangeServiceGetSeaOrderSplitContext>>);

    vi.spyOn(changeService, 'seaOrderChangeServicePreviewSeaOrderSplit').mockResolvedValue({
      data: mockFailedPreview,
    } as Awaited<ReturnType<typeof changeService.seaOrderChangeServicePreviewSeaOrderSplit>>);

    render(
      <App>
        <SeaOrderSplitPage />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText('件数守恒校验未通过：仍有 40 件未分配')).toBeInTheDocument();
      expect(screen.getByText('等待满足守恒条件')).toBeInTheDocument();
      const submitBtn = screen.getByRole('button', { name: '确认执行拆票' });
      expect(submitBtn).toBeDisabled();
    });
  });

  it('SHA-256 哈希稳定性与等长不同参数碰撞防御', () => {
    const payload1 = {
      orderId: '123',
      targets: [{ masterNo: 'ABC' }],
      note: 'foo',
    };
    const payload2 = {
      orderId: '123',
      targets: [{ masterNo: 'XYZ' }],
      note: 'bar',
    };
    // 两个 payload 字符串长度完全相同
    expect(JSON.stringify(payload1).length).toBe(JSON.stringify(payload2).length);

    const hash1 = computeCanonicalSha256(payload1);
    const hash2 = computeCanonicalSha256(payload2);
    expect(hash1).not.toBe(hash2);

    // 相同 payload 重复计算哈希绝对稳定
    const hash1Repeat = computeCanonicalSha256(payload1);
    expect(hash1).toBe(hash1Repeat);
  });

  it('使用十进制精度汇总各币种费用，不把 0.1 + 0.2 计算成浮点误差', () => {
    const summaries = calculateFeeCurrencySummaries(
      [
        {
          id: 'fee-1',
          direction: 'RECEIVABLE',
          currency: 'USD',
          totalAmount: '0.1',
        },
        {
          id: 'fee-2',
          direction: 'RECEIVABLE',
          currency: 'USD',
          totalAmount: '0.2',
        },
      ],
      { 'fee-1': 'res-origin', 'fee-2': 'res-new-1' },
      ['res-origin', 'res-new-1'],
    );

    expect(summaries).toHaveLength(1);
    expect(summaries[0].baseline.toString()).toBe('0.3');
    expect(summaries[0].assignedByResult['res-origin'].toString()).toBe('0.1');
    expect(summaries[0].assignedByResult['res-new-1'].toString()).toBe('0.2');
    expect(summaries[0].remaining.isZero()).toBe(true);
  });

  it('缺少关键版本信息时阻断拆票预览与提交', async () => {
    const mockContextWithoutVersion: API.SeaOrderSplitContextData = {
      orderId: 'test-order-123',
      orderNo: 'SE20260903001',
      orderVersion: '', // 缺少版本
      currentLinkVersion: '1',
      cargoAllocationVersion: '1',
      documentStructure: 'HOUSE',
      flowStatus: 'BOOKED',
      houseBills: [],
      containers: [],
      allocations: [],
      draftFees: [],
      attachments: [],
    };

    vi.spyOn(changeService, 'seaOrderChangeServiceGetSeaOrderSplitContext').mockResolvedValue({
      data: mockContextWithoutVersion,
    } as Awaited<ReturnType<typeof changeService.seaOrderChangeServiceGetSeaOrderSplitContext>>);

    render(
      <App>
        <SeaOrderSplitPage />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText(/缺少完整版本控制信息/)).toBeInTheDocument();
      const submitBtn = screen.getByRole('button', { name: '确认执行拆票' });
      expect(submitBtn).toBeDisabled();
    });
  });
});
