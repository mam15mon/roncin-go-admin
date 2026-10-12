import { ModalForm, ProFormCheckbox } from '@ant-design/pro-components';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Alert, Descriptions, Spin, Table } from 'antd';
import React, { useRef } from 'react';
import { useInitialState } from '@/app/AppProvider';
import {
  settlementServiceMarkCommissionPaid,
  settlementServicePreviewCommissionPayment,
} from '@/services/roncin/settlementService';
import { formatAmount } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';

export default function CommissionPaymentModal({
  record,
  onClose,
  onPaid,
}: {
  record: API.FinanceCommission;
  onClose: () => void;
  onPaid: () => void;
}) {
  const { initialState } = useInitialState();
  const actionKey = useRef(generateUUID());
  const preview = useQuery({
    queryKey: [
      'finance',
      'commission-payment',
      initialState?.currentUser?.currentOrganization?.id,
      record.id,
      record.version,
    ],
    queryFn: async () => {
      if (!record.id || !record.version) throw new Error('提成身份或版本缺失');
      const response = await settlementServicePreviewCommissionPayment(
        { id: record.id as string },
        { id: record.id, expectedVersion: record.version },
      );
      if (!response?.data) throw new Error('发放预览为空');
      return response.data;
    },
    meta: { errorMessage: '无法预览提成实际发放金额' },
  });
  const mutation = useMutation({
    mutationFn: (body: API.MarkCommissionPaidRequest) =>
      settlementServiceMarkCommissionPaid({ id: record.id as string }, body),
  });
  return (
    <ModalForm
      title={`登记提成发放 ${record.commissionNo || ''}`}
      open
      modalProps={{ destroyOnHidden: true }}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      submitter={{
        searchConfig: {
          submitText: preview.data?.offsetSettlement
            ? '登记抵扣结清'
            : '登记实际发放',
        },
        submitButtonProps: {
          disabled: preview.isFetching || preview.isError || !preview.data,
        },
      }}
      onFinish={async () => {
        if (
          !record.id ||
          !preview.data?.version ||
          !preview.data.previewToken ||
          preview.data.actualAmount == null ||
          preview.isError
        )
          return false;
        try {
          const response = await mutation.mutateAsync({
            id: record.id,
            expectedVersion: preview.data.version,
            previewToken: preview.data.previewToken,
            actualAmount: preview.data.actualAmount,
            idempotencyKey: actionKey.current,
          });
          if (response?.success !== true || response.data?.id !== record.id)
            return false;
          onPaid();
          return true;
        } catch {
          return false;
        }
      }}
    >
      {preview.isPending && <Spin />}
      {preview.isError && (
        <Alert type="error" showIcon title="发放预览失败，请关闭后重新预览" />
      )}
      {!preview.isError && preview.data && (
        <>
          <Alert
            type="warning"
            showIcon
            title="请先核对线下实际付款。登记会同时执行下列抵扣调整，完成后不可重复扣回。"
          />
          <Descriptions
            column={1}
            items={[
              {
                key: 'original',
                label: '原提成',
                children: `${formatAmount(preview.data.originalAmount)} ${preview.data.baseCurrency}`,
              },
              {
                key: 'adjustment',
                label: '本次调整净额',
                children: formatAmount(preview.data.adjustmentAmount),
              },
              {
                key: 'actual',
                label: preview.data.offsetSettlement
                  ? '抵扣结清'
                  : '本次实际发放',
                children: `${formatAmount(preview.data.actualAmount)} ${preview.data.baseCurrency}`,
              },
            ]}
          />
          <Table<API.CommissionPaymentAdjustment>
            size="small"
            pagination={false}
            rowKey="id"
            dataSource={preview.data.adjustments ?? []}
            columns={[
              { title: '调整单号', dataIndex: 'adjustmentNo' },
              {
                title: '方向',
                dataIndex: 'direction',
                render: (value) => (value === 'INCREASE' ? '增提' : '冲减'),
              },
              {
                title: '金额',
                dataIndex: 'amount',
                render: (value) => formatAmount(value),
              },
            ]}
          />
          <ProFormCheckbox
            name="confirmed"
            rules={[
              {
                validator: (_, value) =>
                  value
                    ? Promise.resolve()
                    : Promise.reject(new Error('请确认本次实际金额')),
              },
            ]}
          >
            {preview.data.offsetSettlement
              ? '确认通过上述调整抵扣结清，没有发生转账'
              : '确认已按上述净额在线下完成实际发放'}
          </ProFormCheckbox>
        </>
      )}
    </ModalForm>
  );
}
