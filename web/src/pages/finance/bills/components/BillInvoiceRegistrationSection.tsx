import type { ActionType } from '@ant-design/pro-components';
import { Button } from 'antd';
import Decimal from 'decimal.js';
import { useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { ResponsiveProTable, SectionCard } from '@/components/ui';
import {
  FinanceBillInvoiceProgress,
  FinanceBillStatus,
} from '@/enums.generated';
import {
  InvoiceAllocationDetails,
  InvoiceRegistrationModal,
} from '@/features/finance/invoice-registration';
import { settlementServiceListInvoices } from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { formatAmount } from '@/utils/format';

export default function BillInvoiceRegistrationSection({
  bill,
  onRefresh,
  active,
}: {
  bill: API.FinanceBill;
  onRefresh?: () => void;
  active: boolean;
}) {
  const access = useAccess();
  const [open, setOpen] = useState(false);
  const actionRef = useRef<ActionType>(undefined);
  const canOperate =
    active && access.canOperateOrganization(bill.organizationId);
  const canRegister =
    canOperate &&
    access.canCreateFinanceInvoices &&
    bill.status === FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED &&
    bill.invoiceProgress !==
      FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED &&
    bill.invoiceProgress !==
      FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_FULL;
  return (
    <SectionCard
      title="已有发票登记"
      style={{ marginTop: 16 }}
      extra={
        canRegister && (
          <Button size="small" type="primary" onClick={() => setOpen(true)}>
            继续登记发票
          </Button>
        )
      }
    >
      {access.canReadFinanceInvoices && (
        <ResponsiveProTable<API.FinanceInvoice>
          actionRef={actionRef}
          rowKey="id"
          size="small"
          options={false}
          search={false}
          pagination={{ defaultPageSize: 10 }}
          expandable={{
            expandedRowRender: (invoice) => (
              <InvoiceAllocationDetails links={invoice.billLinks} />
            ),
            columnTitle: '分配追溯',
          }}
          columns={[
            { title: '记录编号', dataIndex: 'recordNo', width: 170 },
            { title: '发票号码', dataIndex: 'taxInvoiceNo', width: 150 },
            { title: '发票日期', dataIndex: 'invoiceDate', width: 110 },
            {
              title: '本账单分配',
              key: 'billAmount',
              width: 150,
              render: (_, invoice) => {
                const links =
                  invoice.billLinks?.filter(
                    (link) => link.billId === bill.id && link.active === true,
                  ) ?? [];
                return links.length
                  ? `${formatAmount(links.reduce((sum, link) => sum.plus(link.amount ?? '0'), new Decimal(0)).toString())} ${invoice.currency ?? ''}`
                  : '已释放';
              },
            },
            {
              title: '关联账单',
              key: 'bills',
              render: (_, invoice) =>
                [
                  ...new Set(invoice.billLinks?.map((link) => link.billNo)),
                ].join('、'),
            },
            {
              title: '有效情况',
              key: 'active',
              width: 130,
              render: (_, invoice) =>
                invoice.billLinks?.some(
                  (link) => link.billId === bill.id && link.active,
                )
                  ? '有效'
                  : '已释放，历史保留',
            },
          ]}
          request={async (params) => {
            if (!canOperate || !bill.id)
              return { data: [], total: 0, success: false };
            return toTableRequest(
              await settlementServiceListInvoices({
                organizationId: bill.organizationId,
                billId: bill.id,
                page: params.current,
                pageSize: params.pageSize,
              }),
            );
          }}
        />
      )}
      <InvoiceRegistrationModal
        open={open && canOperate}
        onOpenChange={setOpen}
        bills={[bill]}
        onSuccess={() => {
          void actionRef.current?.reload();
          onRefresh?.();
        }}
      />
    </SectionCard>
  );
}
