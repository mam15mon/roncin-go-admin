import { ProFormDependency, ProFormTextArea } from '@ant-design/pro-components';
import { App, Button } from 'antd';
import { useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { GuardedModalForm, ProFormSearchableSelect } from '@/components/ui';
import { defaultSelectFilterOption } from '@/components/ui/searchable-select/utils';
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
  canDelete,
  disabled,
  onCreated,
  ...options
}: FeeSupplementOptions & {
  orderId: string;
  fees: API.OrderFee[];
  canCreate: boolean;
  canUpdate: boolean;
  canDelete: boolean;
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
  const canApplyOperation = (operation: number) =>
    operation === 1
      ? canCreate
      : operation === 2
        ? canUpdate
        : operation === 3 && canDelete;
  const submit = async (
    selection: Session,
    values?: FeeSupplementFormValues,
  ) => {
    if (disabled || !canApplyOperation(selection.operation)) return false;
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
    message.success('费用补录已提交补审，批准后生效');
    setSession(undefined);
    onCreated();
    return true;
  };
  const hasApplyPermission = canCreate || canUpdate || canDelete;
  // 已打开的输入在失权时保留，只冻结本次操作，避免丢失尚未提交的原因和费用。
  if (!hasApplyPermission && !session && !selectionOpen) return null;
  const source = session?.source;
  return (
    <>
      <Button
        disabled={disabled || !hasApplyPermission}
        onClick={() => {
          setSelectionKey(generateUUID());
          setSelectionOpen(true);
        }}
      >
        申请费用补录
      </Button>
      <GuardedModalForm<Selection>
        key={`selection:${selectionKey}`}
        title="选择费用补录方式"
        open={selectionOpen}
        onOpenChange={setSelectionOpen}
        modalProps={{ destroyOnHidden: true }}
        initialValues={{ operation: canCreate ? 1 : canUpdate ? 2 : 3 }}
        onFinish={async (values) => {
          if (disabled) return false;
          if (
            (values.operation === 1 && !canCreate) ||
            (values.operation === 2 && !canUpdate) ||
            (values.operation === 3 && !canDelete)
          )
            return false;
          const selected = currentFees.find(
            (fee) => fee.id === values.sourceFeeId,
          );
          if (
            values.operation !== 1 &&
            (!selected?.id || !selected.version || selected.canCorrect !== true)
          )
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
        <ProFormSearchableSelect
          name="operation"
          label="补录方式"
          rules={[{ required: true }]}
          options={[
            ...(canCreate ? [{ label: '追加应收或应付', value: 1 }] : []),
            ...(canUpdate ? [{ label: '更正已有费用', value: 2 }] : []),
            ...(canDelete ? [{ label: '整笔作废', value: 3 }] : []),
          ]}
        />
        <ProFormDependency name={['operation']}>
          {({ operation }) => (
            <>
              {operation !== 1 && (
                <ProFormSearchableSelect
                  name="sourceFeeId"
                  label="原费用"
                  rules={[{ required: true }]}
                  fieldProps={{
                    showSearch: true,
                    filterOption: defaultSelectFilterOption,
                  }}
                  options={currentFees.map((fee) => ({
                    value: fee.id,
                    disabled: fee.canCorrect !== true,
                    label: `${fee.direction === 1 ? '应收' : '应付'} ${fee.feeName} / ${fee.settlementPartyName} / ${formatAmount(fee.totalAmount)} ${fee.currency}${fee.hasActiveBill ? '（须先回退至账单未建立）' : fee.hasPendingChange ? '（补审中）' : ''}`,
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
          key={`supplement:${session.key}`}
          {...options}
          orderId={orderId}
          open
          disabled={disabled || !canApplyOperation(session.operation)}
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
