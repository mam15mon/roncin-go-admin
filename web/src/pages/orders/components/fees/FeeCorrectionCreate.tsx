import {
  ProFormDependency,
  ProFormSelect,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App, Button } from 'antd';
import dayjs from 'dayjs';
import { useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { GuardedModalForm } from '@/components/ui';
import { orderFeeServiceCreateOrderFeeCorrection } from '@/services/roncin/orderFeeService';
import { isWriteResponse } from '@/utils/api';
import { normalizeDecimalInput } from '@/utils/decimal';
import { formatAmount } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import FeeSupplementModal, {
  type FeeSupplementFormValues,
  type FeeSupplementOptions,
} from './FeeSupplementModal';

type Selection = { operation: number; sourceFeeId?: string; reason?: string };
type Session = Selection & { key: string; source?: API.OrderFee };

/** 更正复用现有费用表单；未批准时不允许改写原费用。 */
export default function FeeCorrectionCreate({
  orderId,
  fees,
  canCreate,
  canUpdate,
  disabled,
  onCreated,
  ...options
}: FeeSupplementOptions & {
  orderId: string;
  fees: API.OrderFee[];
  canCreate: boolean;
  canUpdate: boolean;
  disabled: boolean;
  onCreated: () => void;
}) {
  const { message } = App.useApp();
  const [session, setSession] = useState<Session>();
  const [selectionOpen, setSelectionOpen] = useState(false);
  const [selectionKey, setSelectionKey] = useState(generateUUID);
  const { markDirty, resetDirty } = useUnsavedChangesGuard({
    enabled: selectionOpen || Boolean(session),
    identity: orderId,
  });
  const currentFees = fees.filter(
    (fee) => !fee.supersededByRequestId && fee.economicEffect !== 'DECREASE',
  );
  const submit = async (
    selection: Session,
    values?: FeeSupplementFormValues,
  ) => {
    const source = selection.source;
    const response = await orderFeeServiceCreateOrderFeeCorrection(
      { orderId },
      {
        orderId,
        operation: selection.operation,
        sourceFeeId: source?.id,
        sourceFeeExpectedVersion: source?.version,
        idempotencyKey: selection.key,
        reason: values?.reason || selection.reason,
        targetFee: values
          ? {
              direction: values.direction,
              feeSettingId: values.feeSettingId,
              settlementPartyId: values.settlementPartyId,
              billingUnitId: values.billingUnitId,
              quantity: normalizeDecimalInput(String(values.quantity)),
              unitPrice: normalizeDecimalInput(String(values.unitPrice)),
              currency: values.currency,
              expenseDate: dayjs(values.expenseDate).format('YYYY-MM-DD'),
              note: values.note,
              exchangeRateOverride: values.exchangeRateOverride,
              taxInclusive: source ? source.taxInclusive === true : true,
              feeNameOverride:
                source?.feeSettingId === values.feeSettingId
                  ? source.feeName
                  : undefined,
              taxRateOverride:
                source?.feeSettingId === values.feeSettingId
                  ? source.taxRate
                  : undefined,
            }
          : undefined,
      },
    );
    if (!isWriteResponse(response, true)) return false;
    if (!response?.success) return false;
    resetDirty();
    message.success('费用更正已提交审批');
    setSession(undefined);
    onCreated();
    return true;
  };
  if (!canCreate && !canUpdate) return null;
  const source = session?.source;
  return (
    <>
      <Button
        disabled={disabled}
        onClick={() => {
          setSelectionKey(generateUUID());
          setSelectionOpen(true);
        }}
      >
        申请费用更正
      </Button>
      <GuardedModalForm<Selection>
        key={selectionKey}
        title="选择费用更正方式"
        open={selectionOpen}
        onOpenChange={setSelectionOpen}
        modalProps={{ destroyOnHidden: true }}
        initialValues={{ operation: canCreate ? 1 : 2 }}
        onFinish={async (values) => {
          const selected = currentFees.find(
            (fee) => fee.id === values.sourceFeeId,
          );
          if (values.operation !== 1 && (!selected?.id || !selected.version))
            return false;
          const next = {
            ...values,
            key: selectionKey,
            source: values.operation === 1 ? undefined : selected,
          };
          if (values.operation === 3) return submit(next);
          markDirty();
          setSession(next);
          return true;
        }}
      >
        <ProFormSelect
          name="operation"
          label="更正方式"
          rules={[{ required: true }]}
          options={[
            ...(canCreate ? [{ label: '追加应收或应付', value: 1 }] : []),
            ...(canUpdate
              ? [
                  { label: '更正已有费用', value: 2 },
                  { label: '整笔作废', value: 3 },
                ]
              : []),
          ]}
        />
        <ProFormDependency name={['operation']}>
          {({ operation }) => (
            <>
              {operation !== 1 && (
                <ProFormSelect
                  name="sourceFeeId"
                  label="原费用"
                  rules={[{ required: true }]}
                  fieldProps={{ showSearch: true, optionFilterProp: 'label' }}
                  options={currentFees.map((fee) => ({
                    value: fee.id,
                    label: `${fee.direction === 1 ? '应收' : '应付'} ${fee.feeName} / ${fee.settlementPartyName} / ${formatAmount(fee.totalAmount)} ${fee.currency}`,
                  }))}
                />
              )}
              {operation === 3 && (
                <ProFormTextArea
                  name="reason"
                  label="作废原因"
                  rules={[{ required: true, whitespace: true, max: 500 }]}
                  fieldProps={{ maxLength: 500, showCount: true }}
                />
              )}
            </>
          )}
        </ProFormDependency>
      </GuardedModalForm>
      {session && (
        <FeeSupplementModal
          key={session.key}
          {...options}
          orderId={orderId}
          open
          correction
          initialDirection={source?.direction || 1}
          onOpenChange={(open) => {
            if (!open) setSession(undefined);
          }}
          initialRequest={
            source
              ? {
                  id: source.id,
                  feeSettingId: source.feeSettingId,
                  settlementPartyId: source.settlementPartyId,
                  billingUnitId: source.billingUnitId,
                  quantity: source.quantity,
                  unitPrice: source.unitPrice,
                  currency: source.currency,
                  expenseDate: source.expenseDate,
                  note: source.note,
                  exchangeRate: source.exchangeRate,
                  exchangeRateSource: source.exchangeRateSource,
                }
              : undefined
          }
          onSubmit={(values) => submit(session, values)}
        />
      )}
    </>
  );
}
