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

    expect(screen.getByText('SE2026080001')).toBeInTheDocument();
    expect(screen.getByText('上海测试进出口贸易有限公司')).toBeInTheDocument();
    expect(screen.getByText(/500 CTNS/)).toBeInTheDocument();
    expect(screen.getAllByText('12500.5').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('45.2').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('1 柜')).toBeInTheDocument();
  });

  it('正常渲染各业务 SectionCard 区块与子表格数据', () => {
    render(<OrderDetailTemplate data={mockData} title="海运出口订单" />);

    expect(screen.getByText('基础委托与商务条款')).toBeInTheDocument();
    expect(screen.getByText('航程路线与节点截关时间')).toBeInTheDocument();
    expect(screen.getByText('提单与单证档案')).toBeInTheDocument();
    expect(screen.getByText('集装箱装载与货物明细')).toBeInTheDocument();
    expect(screen.getByText('业务履约里程碑轨迹')).toBeInTheDocument();
    expect(screen.getByText('附件档案明细')).toBeInTheDocument();
    expect(screen.getByText('干系人员与内部备注')).toBeInTheDocument();

    // 子表格内容
    expect(screen.getByText('COSU63001234')).toBeInTheDocument();
    expect(screen.getByText('TGHU1234567')).toBeInTheDocument();
    expect(screen.getByText('电子元器件与配件')).toBeInTheDocument();
    expect(screen.getByText('booking_confirmation.pdf')).toBeInTheDocument();
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
