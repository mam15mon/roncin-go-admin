import { FileDoneOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Space, Tabs, Tag } from 'antd';
import React from 'react';
import {
  FEE_CANCELLED,
  FEE_CONFIRMED,
  PAYABLE,
  RECEIVABLE,
  feeDirectionCode,
  feeStatusCode,
} from './feeConstants';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { unwrapList } from '@/utils/api';

interface OrderFeeTableTabsProps {
  orderId: string;
  receivableActionRef: React.RefObject<ActionType | undefined>;
  payableActionRef: React.RefObject<ActionType | undefined>;
  receivableSummary: { totalAmount: number; count: number };
  payableSummary: { totalAmount: number; count: number };
  selectedReceivableFeeIds: React.Key[];
  setSelectedReceivableFeeIds: (keys: React.Key[]) => void;
  selectedPayableFeeIds: React.Key[];
  setSelectedPayableFeeIds: (keys: React.Key[]) => void;
  setAllReceivableItems: (items: API.OrderFee[]) => void;
  setAllPayableItems: (items: API.OrderFee[]) => void;
  setReceivableSummary: (summary: { totalAmount: number; count: number }) => void;
  setPayableSummary: (summary: { totalAmount: number; count: number }) => void;
  canCreateFinanceBills: boolean;
  financeLocked: boolean;
  onOpenBillWorkbench: (feeIds: string[]) => void;
  onOpenFeeModal: (direction: number) => void;
  getTableColumns: (direction: number) => ProColumns<API.OrderFee>[];
}

export default function OrderFeeTableTabs({
  orderId,
  receivableActionRef,
  payableActionRef,
  receivableSummary,
  payableSummary,
  selectedReceivableFeeIds,
  setSelectedReceivableFeeIds,
  selectedPayableFeeIds,
  setSelectedPayableFeeIds,
  setAllReceivableItems,
  setAllPayableItems,
  setReceivableSummary,
  setPayableSummary,
  canCreateFinanceBills,
  financeLocked,
  onOpenBillWorkbench,
  onOpenFeeModal,
  getTableColumns,
}: OrderFeeTableTabsProps) {
  return (
    <Tabs
      type="card"
      defaultActiveKey="receivable"
      items={[
        {
          key: 'receivable',
          label: (
            <Space>
              <span>应收费用</span>
              <Tag color="blue">{receivableSummary.count}</Tag>
            </Space>
          ),
          children: (
            <ProTable<API.OrderFee>
              actionRef={receivableActionRef}
              rowKey="id"
              search={false}
              bordered
              pagination={false}
              rowSelection={{
                selectedRowKeys: selectedReceivableFeeIds,
                onChange: setSelectedReceivableFeeIds,
                getCheckboxProps: (record) => ({
                  disabled: feeStatusCode(record.status) !== FEE_CONFIRMED,
                }),
              }}
              tableAlertRender={({ selectedRowKeys }) =>
                `已选择 ${selectedRowKeys.length} 笔已确认应收费用`
              }
              toolBarRender={() =>
                [
                  canCreateFinanceBills && (
                    <Button
                      key="bill"
                      icon={<FileDoneOutlined />}
                      disabled={selectedReceivableFeeIds.length === 0}
                      onClick={() =>
                        onOpenBillWorkbench(
                          selectedReceivableFeeIds.map(String),
                        )
                      }
                    >
                      生成账单（{selectedReceivableFeeIds.length}）
                    </Button>
                  ),
                  <Button
                    key="add"
                    type="primary"
                    icon={<PlusOutlined />}
                    disabled={financeLocked}
                    onClick={() => onOpenFeeModal(RECEIVABLE)}
                  >
                    + 新增应收费用
                  </Button>,
                ].filter(Boolean)
              }
              request={async () => {
                if (!orderId) return { data: [], success: true };
                const res = await orderFeeServiceListFees({ orderId });
                const rItems = unwrapList(res).filter(
                  (f) => feeDirectionCode(f.direction) === RECEIVABLE,
                );
                setAllReceivableItems(rItems);
                const activeItems = rItems.filter(
                  (f) => feeStatusCode(f.status) !== FEE_CANCELLED,
                );
                const total = activeItems.reduce(
                  (acc, cur) =>
                    acc +
                    (cur.baseCurrencyAmount
                      ? Number(cur.baseCurrencyAmount)
                      : 0),
                  0,
                );
                setReceivableSummary({
                  totalAmount: total,
                  count: rItems.length,
                });
                return { data: rItems, success: true };
              }}
              columns={getTableColumns(RECEIVABLE)}
            />
          ),
        },
        {
          key: 'payable',
          label: `应付费用 (${payableSummary.count})`,
          children: (
            <ProTable<API.OrderFee>
              actionRef={payableActionRef}
              rowKey="id"
              search={false}
              bordered
              size="small"
              pagination={false}
              rowSelection={{
                selectedRowKeys: selectedPayableFeeIds,
                onChange: setSelectedPayableFeeIds,
                getCheckboxProps: (record) => ({
                  disabled: feeStatusCode(record.status) !== FEE_CONFIRMED,
                }),
              }}
              tableAlertRender={({ selectedRowKeys }) =>
                `已选择 ${selectedRowKeys.length} 笔已确认应付费用`
              }
              toolBarRender={() =>
                [
                  canCreateFinanceBills && (
                    <Button
                      key="bill"
                      icon={<FileDoneOutlined />}
                      disabled={selectedPayableFeeIds.length === 0}
                      onClick={() =>
                        onOpenBillWorkbench(selectedPayableFeeIds.map(String))
                      }
                    >
                      生成账单（{selectedPayableFeeIds.length}）
                    </Button>
                  ),
                  <Button
                    key="add"
                    type="primary"
                    icon={<PlusOutlined />}
                    disabled={financeLocked}
                    style={{
                      backgroundColor: '#fa8c16',
                      borderColor: '#fa8c16',
                    }}
                    onClick={() => onOpenFeeModal(PAYABLE)}
                  >
                    + 新增应付费用
                  </Button>,
                ].filter(Boolean)
              }
              request={async () => {
                if (!orderId) return { data: [], success: true };
                const res = await orderFeeServiceListFees({ orderId });
                const pItems = unwrapList(res).filter(
                  (f) => feeDirectionCode(f.direction) === PAYABLE,
                );
                setAllPayableItems(pItems);
                const activeItems = pItems.filter(
                  (f) => feeStatusCode(f.status) !== FEE_CANCELLED,
                );
                const total = activeItems.reduce(
                  (acc, cur) =>
                    acc +
                    (cur.baseCurrencyAmount
                      ? Number(cur.baseCurrencyAmount)
                      : 0),
                  0,
                );
                setPayableSummary({
                  totalAmount: total,
                  count: pItems.length,
                });
                return { data: pItems, success: true };
              }}
              columns={getTableColumns(PAYABLE)}
            />
          ),
        },
      ]}
    />
  );
}
