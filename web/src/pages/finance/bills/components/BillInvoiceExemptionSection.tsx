import { Alert, App, Button, Form, Input, Space, Tag, Tooltip } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import {
  GuardedModalForm,
  SectionCard,
  ResponsiveTable as Table,
} from '@/components/ui';
import { FinanceBillFeeLockFact, FinanceBillStatus } from '@/enums.generated';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import {
  settlementServiceRevokeBillInvoiceExemption,
  settlementServiceSetBillInvoiceExemption,
} from '@/services/roncin/settlementService';
import { isWriteResponse } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatDate } from '@/utils/format';
import { useBillChangeNotice } from './use-bill-change-notice';

export type BillInvoiceExemptionActions = {
  onUpdated: (bill: API.FinanceBill) => void;
  onRefresh: () => void;
};

type Props = {
  bill: API.FinanceBill;
  /** 父页面按独立免票权限与当前公司授权，读取历史不要求此权限。 */
  actions?: BillInvoiceExemptionActions;
};

type Exemption = NonNullable<API.FinanceBill['invoiceExemptions']>[number];

export default function BillInvoiceExemptionSection({ bill, actions }: Props) {
  const records = bill.invoiceExemptions ?? [];
  const active = records.some((record) => !record.revokedAt);
  return (
    <SectionCard
      title="无需发票记录"
      style={{ marginTop: 16 }}
      extra={
        actions && (
          <ExemptionActions key={bill.id} bill={bill} actions={actions} />
        )
      }
    >
      <Space style={{ marginBottom: 8 }}>
        <Tag color={active ? 'gold' : 'default'}>
          {active ? '无需发票' : '按实际发票登记进度处理'}
        </Tag>
        {active && <span>标记有效期间不能增删账单费用。</span>}
      </Space>
      <Table<Exemption>
        rowKey="id"
        size="small"
        bordered
        pagination={false}
        dataSource={records}
        columns={[
          {
            title: '状态',
            key: 'status',
            width: 80,
            render: (_, record) =>
              record.revokedAt ? (
                <Tag>已撤销</Tag>
              ) : (
                <Tag color="gold">有效</Tag>
              ),
          },
          { title: '免票原因', dataIndex: 'reason', width: 220 },
          {
            title: '标记人',
            dataIndex: 'markedByName',
            width: 100,
            render: (value) => value || '未命名人员',
          },
          {
            title: '标记时间',
            dataIndex: 'createdAt',
            width: 170,
            render: (value) => formatDate(value, 'datetime'),
          },
          {
            title: '撤销人',
            key: 'revokedByName',
            width: 100,
            render: (_, record) =>
              record.revokedAt ? record.revokedByName || '未命名人员' : '-',
          },
          {
            title: '撤销时间',
            dataIndex: 'revokedAt',
            width: 170,
            render: (value) => formatDate(value, 'datetime'),
          },
          {
            title: '撤销原因',
            dataIndex: 'revocationReason',
            width: 220,
            render: (value) => value || '-',
          },
        ]}
      />
    </SectionCard>
  );
}

type Operation = {
  billId: string;
  organizationId: string;
  expectedVersion: string;
  mode: 'mark' | 'revoke';
};

/** 打开时固化提交版本，探测及父页刷新不会悄悄替换用户输入的保存基线。 */
function ExemptionActions({
  bill,
  actions,
}: {
  bill: API.FinanceBill;
  actions: BillInvoiceExemptionActions;
}) {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<{ reason: string }>();
  const [operation, setOperation] = useState<Operation>();
  const guard = useAsyncGuard();
  const refreshConfirmRef = useRef<
    ReturnType<typeof modal.confirm> | undefined
  >(undefined);
  const latestRef = useRef({ bill, actions });
  latestRef.current = { bill, actions };
  const active = (bill.invoiceExemptions ?? []).some(
    (record) => !record.revokedAt,
  );
  const hasInvoice = (bill.feeLockFacts ?? []).includes(
    FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE,
  );
  const editable = Boolean(
    bill.id &&
      bill.organizationId &&
      bill.version &&
      bill.status === FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
  );
  const { changed } = useBillChangeNotice({
    billId: operation?.billId,
    version: operation?.expectedVersion,
    enabled: Boolean(operation),
  });
  useEffect(
    () => () => {
      guard.invalidate();
      refreshConfirmRef.current?.destroy();
      refreshConfirmRef.current = undefined;
    },
    [guard, operation],
  );

  const close = () => {
    guard.invalidate();
    setOperation(undefined);
    form.resetFields();
  };
  const title = operation?.mode === 'revoke' ? '撤销无需发票' : '标记无需发票';
  const refresh = () => {
    const apply = () => {
      close();
      latestRef.current.actions.onRefresh();
    };
    if (!form.isFieldsTouched()) {
      apply();
      return;
    }
    refreshConfirmRef.current = modal.confirm({
      title: '刷新账单',
      content: '刷新会清除尚未保存的原因，确认继续？',
      onOk: apply,
    });
  };

  return (
    <>
      <Tooltip
        title={
          !active && hasInvoice ? '已有有效发票，不能标记无需发票' : undefined
        }
      >
        <Button
          size="small"
          danger={active}
          disabled={!editable || (!active && hasInvoice)}
          onClick={() => {
            if (!editable || (!active && hasInvoice)) return;
            form.resetFields();
            setOperation({
              billId: bill.id as string,
              organizationId: bill.organizationId as string,
              expectedVersion: bill.version as string,
              mode: active ? 'revoke' : 'mark',
            });
          }}
        >
          {active ? '撤销无需发票' : '标记无需发票'}
        </Button>
      </Tooltip>
      <GuardedModalForm<{ reason: string }>
        title={title}
        open={Boolean(operation)}
        form={form}
        guardIdentity={operation}
        width={560}
        modalProps={{ destroyOnHidden: true, onCancel: close }}
        onOpenChange={(open) => {
          if (!open) close();
        }}
        submitter={{ searchConfig: { submitText: '保存' } }}
        onFinish={async ({ reason }) => {
          if (!operation) return false;
          const current = latestRef.current.bill;
          if (
            current.id !== operation.billId ||
            current.organizationId !== operation.organizationId ||
            current.status !== FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED
          )
            return false;
          let saved = false;
          try {
            await guard.run(
              ({ signal }) => {
                const submit =
                  operation.mode === 'revoke'
                    ? settlementServiceRevokeBillInvoiceExemption
                    : settlementServiceSetBillInvoiceExemption;
                return submit(
                  { id: operation.billId },
                  {
                    id: operation.billId,
                    expectedVersion: operation.expectedVersion,
                    reason: reason.trim(),
                  },
                  { signal, skipErrorHandler: true },
                );
              },
              (response) => {
                if (!response || response.success === false) return;
                if (
                  !isWriteResponse(response, true) ||
                  response.data?.id !== operation.billId ||
                  !response.data.version
                )
                  throw new Error('无需发票操作未返回有效账单，请刷新后核对');
                latestRef.current.actions.onUpdated(response.data);
                message.success(
                  operation.mode === 'revoke'
                    ? '无需发票标记已撤销'
                    : '已标记无需发票',
                );
                saved = true;
              },
            );
            return saved;
          } catch (error) {
            message.error(getErrorMessage(error, `${title}失败`));
            return false;
          }
        }}
      >
        <Alert
          type="info"
          showIcon
          title={
            operation?.mode === 'revoke'
              ? '撤销保留原记录；存在其他有效发票、核销或对冲时，费用仍受保护。'
              : '无需发票作为独立财务决定保存，标记后禁止增删费用，结清后才能完成账单。'
          }
          style={{ marginBottom: 12 }}
        />
        {changed && (
          <Alert
            type="warning"
            showIcon
            title="这个账单已经发生变动，请重新刷新。"
            action={
              <Button size="small" onClick={refresh}>
                刷新
              </Button>
            }
            style={{ marginBottom: 12 }}
          />
        )}
        <Form.Item
          name="reason"
          label={operation?.mode === 'revoke' ? '撤销原因' : '免票原因'}
          rules={[
            { required: true, whitespace: true, message: '请填写原因' },
            { max: 500, message: '原因不能超过 500 字' },
          ]}
        >
          <Input.TextArea rows={3} maxLength={500} showCount />
        </Form.Item>
      </GuardedModalForm>
    </>
  );
}
