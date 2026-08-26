import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { OrderDetailTemplate } from './OrderDetailTemplate';
import type { OrderDetailData } from './types';

const mockData: OrderDetailData = {
  id: 'order-123',
  orderNo: 'SE2026080001',
  businessTypeTitle: '海运出口订单',
  status: 'PROCESSING',
  canModify: true,
  createdAt: '2026-08-26T10:00:00Z',
  customerName: '上海测试进出口贸易有限公司',
  customerReferenceNo: 'CUST-REF-8888',
  internalReferenceNo: 'INT-REF-9999',
  tradeTermName: 'FOB',
  paymentTermName: 'FREIGHT PREPAID',
  bookingAgentName: '外运订舱代理',
  carrierName: '中远海运 COSRO',
  contractNo: 'CTR-2026-001',
  serviceTypeNames: ['订舱', '报关', '拖车'],
  originName: '上海港 / SHA (CNSHA)',
  destinationName: '洛杉矶港 / LAX (USLAX)',
  vesselVoyage: 'COSCO PRIDE / 024W',
  etd: '2026-09-01',
  eta: '2026-09-15',
  loadingTerms: 'CY-CY',
  totalPackages: 500,
  packageUnit: 'CTNS',
  grossWeightKg: 12500.5,
  volumeCbm: 45.2,
  shippingDocuments: [
    {
      id: 'doc-1',
      masterNo: 'COSU63001234',
      houseNo: 'RC2026080001H',
      masterDocumentType: '正本提单',
      masterReleaseMethod: '正本放单',
      releaseType: '正本放单',
      status: '正常',
    },
  ],
  containers: [
    {
      id: 'cntr-1',
      containerNo: 'TGHU1234567',
      sealNo: 'SL998877',
      containerSpecName: '40HQ 高柜',
      grossWeightKg: 12500.5,
      volumeCbm: 45.2,
      note: '无破损',
    },
  ],
  cargoItems: [
    {
      id: 'cargo-1',
      cargoName: '电子元器件与配件',
      packageCount: 500,
      grossWeightKg: 12500.5,
      volumeCbm: 45.2,
      netWeightKg: 11800,
      note: '防潮包装',
    },
  ],
  milestones: [
    {
      id: 'ms-1',
      type: '订舱确认',
      occurredAt: '2026-08-26T08:00:00Z',
      confirmedAt: '2026-08-26T08:30:00Z',
      note: '配舱回执已收到',
    },
  ],
  attachments: [
    {
      id: 'att-1',
      docType: 'SO',
      fileName: 'booking_confirmation.pdf',
      fileSize: 102400,
      mimeType: 'application/pdf',
      createdAt: '2026-08-26T09:00:00Z',
    },
  ],
  personnel: [
    {
      id: 'p-1',
      roleName: '主操作员',
      userId: 'user-operator-1',
    },
  ],
};

describe('OrderDetailTemplate 订单详情看板模板', () => {
  it('正常渲染订单编号、客户名称与核心指标卡', () => {
    render(<OrderDetailTemplate data={mockData} title="海运出口订单" />);

    expect(screen.getAllByText('SE2026080001').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('上海测试进出口贸易有限公司').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('CY-CY')).toBeInTheDocument();
  });

  it('正常渲染 6 大业务模块区块与流程节点', () => {
    render(<OrderDetailTemplate data={mockData} title="海运出口订单" />);

    // 模块 1：订单状态流程
    expect(screen.getByText('订单状态流程')).toBeInTheDocument();
    expect(screen.getByText('已订舱')).toBeInTheDocument();
    expect(screen.getByText('已配舱')).toBeInTheDocument();
    expect(screen.getByText('拖车已安排')).toBeInTheDocument();

    // 模块 2：业务信息
    expect(screen.getByText('业务信息')).toBeInTheDocument();
    expect(screen.getByText('上海测试进出口贸易有限公司')).toBeInTheDocument();

    // 模块 3：配舱信息
    expect(screen.getByText('配舱信息')).toBeInTheDocument();
    expect(screen.getByText('COSU63001234')).toBeInTheDocument();

    // 模块 4：提单信息
    expect(screen.getByText('提单信息')).toBeInTheDocument();

    // 模块 5：3 个备注
    expect(screen.getByText('业务与操作备注')).toBeInTheDocument();
    expect(screen.getByText('订舱备注')).toBeInTheDocument();
    expect(screen.getByText('配舱备注')).toBeInTheDocument();
    expect(screen.getByText('操作备注')).toBeInTheDocument();

    // 模块 6：内部信息与操作记录
    expect(screen.getByText('内部信息与操作记录')).toBeInTheDocument();
    expect(screen.getByText('内部人员分工配置')).toBeInTheDocument();
    expect(screen.getByText('操作记录与流转日志')).toBeInTheDocument();
  });

  it('点击返回与业务操作按钮触发回调', () => {
    const handleBack = vi.fn();
    const handleOpenFees = vi.fn();
    const handleOpenMilestones = vi.fn();
    const handleOpenAbnormal = vi.fn();

    render(
      <OrderDetailTemplate
        data={mockData}
        title="海运出口订单"
        onBack={handleBack}
        onOpenFees={handleOpenFees}
        onOpenMilestones={handleOpenMilestones}
        onOpenAbnormal={handleOpenAbnormal}
      />,
    );

    // 返回按钮
    const backBtn = screen.getByText('返回列表');
    fireEvent.click(backBtn);
    expect(handleBack).toHaveBeenCalled();

    // 费用核算
    const feeBtn = screen.getByText('费用核算');
    fireEvent.click(feeBtn);
    expect(handleOpenFees).toHaveBeenCalled();

    // 异常登记
    const abnormalBtn = screen.getByText('异常登记');
    fireEvent.click(abnormalBtn);
    expect(handleOpenAbnormal).toHaveBeenCalled();
  });
});
