import { FileDoneOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag } from 'antd';
import React, { useEffect, useRef } from 'react';
import { SectionCard } from '@/components/ui';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { unwrapList } from '@/utils/api';
import {
  FEE_CANCELLED,
  FEE_CONFIRMED,
  feeDirectionCode,
  feeStatusCode,
  PAYABLE,
  RECEIVABLE,
} from './feeConstants';

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
  setReceivableSummary: (summary: {
    totalAmount: number;
    count: number;
  }) => void;
  setPayableSummary: (summary: { totalAmount: number; count: number }) => void;
  canCreateFinanceBills: boolean;
  feeWritesDisabled: boolean;
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
  feeWritesDisabled,
  onOpenBillWorkbench,
  onOpenFeeModal,
  getTableColumns,
}: OrderFeeTableTabsProps) {
  const currentOrderIdRef = useRef(orderId);
  const mountedRef = useRef(true);
  const receivableRequestSequenceRef = useRef(0);
  const payableRequestSequenceRef = useRef(0);
  const latestReceivableResultRef = useRef<
    | {
        orderId: string;
        items: API.OrderFee[];
      }
    | undefined
  >(undefined);
  const latestPayableResultRef = useRef<
    | {
        orderId: string;
        items: API.OrderFee[];
      }
    | undefined
  >(undefined);

  currentOrderIdRef.current = orderId;

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      receivableRequestSequenceRef.current += 1;
      payableRequestSequenceRef.current += 1;
    };
  }, []);

  return (
    <>
      <SectionCard
        title={
          <Space size={8} align="center">
            <span>应收费用</span>
            <Tag color="blue">{receivableSummary.count}</Tag>
          </Space>
        }
        extra={
          <Space size={8}>
            {canCreateFinanceBills && (
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
            )}
            <Button
              key="add"
              type="primary"
              icon={<PlusOutlined />}
              disabled={feeWritesDisabled}
              onClick={() => onOpenFeeModal(RECEIVABLE)}
            >
              新增应收费用
            </Button>
          </Space>
        }
      >
        <ProTable<API.OrderFee>
          key={`receivable:${orderId}`}
          actionRef={receivableActionRef}
          rowKey="id"
          search={false}
          params={{ orderId }}
          bordered
          size="small"
          cardProps={false}
          toolBarRender={false}
          pagination={false}
          scroll={{ x: 'max-content' }}
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
          request={async () => {
            if (!orderId) return { data: [], success: true };
            const requestedOrderId = orderId;
            const requestSequence = ++receivableRequestSequenceRef.current;
            try {
              const res = await orderFeeServiceListFees({ orderId });
              const rItems = unwrapList(res).filter(
                (f) => feeDirectionCode(f.direction) === RECEIVABLE,
              );
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                receivableRequestSequenceRef.current === requestSequence;
              if (!isCurrentRequest) {
                const currentResult = latestReceivableResultRef.current;
                return {
                  data:
                    currentResult?.orderId === currentOrderIdRef.current
                      ? currentResult.items
                      : [],
                  success: true,
                };
              }
              latestReceivableResultRef.current = {
                orderId: requestedOrderId,
                items: rItems,
              };
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
            } catch {
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                receivableRequestSequenceRef.current === requestSequence;
              if (isCurrentRequest) {
                return { data: [], success: false };
              }
              const currentResult = latestReceivableResultRef.current;
              return {
                data:
                  currentResult?.orderId === currentOrderIdRef.current
                    ? currentResult.items
                    : [],
                success: true,
              };
            }
          }}
          columns={getTableColumns(RECEIVABLE)}
        />
      </SectionCard>

      <SectionCard
        title={
          <Space size={8} align="center">
            <span>应付费用</span>
            <Tag color="blue">{payableSummary.count}</Tag>
          </Space>
        }
        extra={
          <Space size={8}>
            {canCreateFinanceBills && (
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
            )}
            <Button
              key="add"
              type="primary"
              icon={<PlusOutlined />}
              disabled={feeWritesDisabled}
              onClick={() => onOpenFeeModal(PAYABLE)}
            >
              新增应付费用
            </Button>
          </Space>
        }
      >
        <ProTable<API.OrderFee>
          key={`payable:${orderId}`}
          actionRef={payableActionRef}
          rowKey="id"
          search={false}
          params={{ orderId }}
          bordered
          size="small"
          cardProps={false}
          toolBarRender={false}
          pagination={false}
          scroll={{ x: 'max-content' }}
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
          request={async () => {
            if (!orderId) return { data: [], success: true };
            const requestedOrderId = orderId;
            const requestSequence = ++payableRequestSequenceRef.current;
            try {
              const res = await orderFeeServiceListFees({ orderId });
              const pItems = unwrapList(res).filter(
                (f) => feeDirectionCode(f.direction) === PAYABLE,
              );
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                payableRequestSequenceRef.current === requestSequence;
              if (!isCurrentRequest) {
                const currentResult = latestPayableResultRef.current;
                return {
                  data:
                    currentResult?.orderId === currentOrderIdRef.current
                      ? currentResult.items
                      : [],
                  success: true,
                };
              }
              latestPayableResultRef.current = {
                orderId: requestedOrderId,
                items: pItems,
              };
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
            } catch {
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                payableRequestSequenceRef.current === requestSequence;
              if (isCurrentRequest) {
                return { data: [], success: false };
              }
              const currentResult = latestPayableResultRef.current;
              return {
                data:
                  currentResult?.orderId === currentOrderIdRef.current
                    ? currentResult.items
                    : [],
                success: true,
              };
            }
          }}
          columns={getTableColumns(PAYABLE)}
        />
      </SectionCard>
    </>
  );
}
