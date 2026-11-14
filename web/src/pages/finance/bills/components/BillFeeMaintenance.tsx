import { useMutation, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Input,
  Modal,
  Popconfirm,
  Space,
  type TableColumnsType,
  Tooltip,
} from 'antd';
import React, { useState } from 'react';
import { ResponsiveTable as Table } from '@/components/ui';
import { FinanceBillStatus } from '@/enums.generated';
import { billFeeLockFactLabels } from '@/features/finance/bill-status';
import {
  settlementServiceListBillFeeCandidates,
  settlementServiceUpdateBillFees,
} from '@/services/roncin/settlementService';
import { formatAmount } from '@/utils/format';
import { useBillChangeNotice } from './use-bill-change-notice';

const CANDIDATE_PAGE_SIZE = 20;

export interface BillFeeMaintenanceProps {
  bill: API.FinanceBill;
  lineColumns: TableColumnsType<API.FinanceBillLine>;
  /** 增删成功：父级以响应替换详情快照并刷新列表。 */
  onUpdated: (bill: API.FinanceBill) => void;
  /** 显式刷新：父级重读账单详情。 */
  onRefresh: () => void;
}

/**
 * 账单详情「费用维护」：在账单行表格上提供移除与添加费用。
 * 候选来自服务端（同公司、同方向、同币种、同结算单位的未建账费用，关键字与分页由服务端处理）；
 * 存在发票、核销、对冲（含草稿对冲）时置灰并说明原因，最终以服务端锁内复核为准。
 * 提交携带当前详情版本作为 expectedVersion，外部变动只提醒、不自动覆盖。
 */
export default function BillFeeMaintenance({
  bill,
  lineColumns,
  onUpdated,
  onRefresh,
}: BillFeeMaintenanceProps) {
  const { message, modal } = App.useApp();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<string[]>([]);
  const editable =
    bill.status === FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED;
  const lockLabels = billFeeLockFactLabels(bill);
  const locked = lockLabels.length > 0;
  const activeLines = (bill.lines ?? []).filter((line) => line.active === true);
  const { changed } = useBillChangeNotice({
    billId: bill.id,
    version: bill.version,
    enabled: editable,
  });

  const candidates = useQuery({
    queryKey: [
      'finance',
      'bill-fee-candidates',
      { billId: bill.id, keyword, page },
    ],
    enabled: pickerOpen && Boolean(bill.id),
    meta: { errorMessage: '加载可添加费用失败' },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceListBillFeeCandidates(
        {
          billId: bill.id as string,
          keyword: keyword || undefined,
          page,
          pageSize: CANDIDATE_PAGE_SIZE,
        },
        { signal },
      );
      if (!response || response.success === false)
        throw new Error('加载可添加费用失败');
      return { items: response.data ?? [], total: Number(response.total ?? 0) };
    },
  });

  const mutation = useMutation({
    mutationFn: (body: API.UpdateBillFeesRequest) =>
      settlementServiceUpdateBillFees({ id: body.id }, body),
  });

  const submit = async (addFeeIds: string[], removeFeeIds: string[]) => {
    if (!bill.id || !bill.version) return false;
    try {
      const response = await mutation.mutateAsync({
        id: bill.id,
        expectedVersion: bill.version,
        addFeeIds,
        removeFeeIds,
      });
      if (response?.success !== true || response.data?.id !== bill.id)
        return false;
      message.success(
        addFeeIds.length ? '费用已加入账单' : '费用已从账单移除，可重新建账',
      );
      onUpdated(response.data);
      return true;
    } catch {
      // 统一请求错误处理已提示（如版本冲突、费用已锁定）；保留当前选择供用户刷新后重试。
      return false;
    }
  };

  const closePicker = () => {
    setPickerOpen(false);
    setSelected([]);
    setKeyword('');
    setPage(1);
  };

  const refresh = () => {
    if (!selected.length) {
      onRefresh();
      return;
    }
    modal.confirm({
      title: '刷新账单',
      content: '刷新会清空已勾选但尚未添加的费用，确认继续？',
      onOk: () => {
        closePicker();
        onRefresh();
      },
    });
  };

  const removeBlockedReason = locked
    ? `账单已有${lockLabels.join('、')}，不能移除费用`
    : activeLines.length <= 1
      ? '账单至少保留一条费用；如需释放全部费用，请取消账单'
      : undefined;

  const columns: TableColumnsType<API.FinanceBillLine> = editable
    ? [
        ...lineColumns,
        {
          title: '操作',
          key: 'actions',
          align: 'right',
          fixed: 'right',
          width: 80,
          render: (_, line) =>
            line.active === true && line.orderFeeId ? (
              <Tooltip title={removeBlockedReason}>
                <span>
                  <Popconfirm
                    title="确认从账单移除该费用？"
                    description="只解除账单关联，费用本身不变，可重新建账。"
                    disabled={Boolean(removeBlockedReason)}
                    onConfirm={() =>
                      submit([], [line.orderFeeId as string]).then(() => {})
                    }
                  >
                    <Button
                      type="link"
                      size="small"
                      danger
                      disabled={
                        Boolean(removeBlockedReason) || mutation.isPending
                      }
                    >
                      移除
                    </Button>
                  </Popconfirm>
                </span>
              </Tooltip>
            ) : null,
        },
      ]
    : lineColumns;

  const candidateColumns: TableColumnsType<API.FeeLedgerItem> = [
    { title: '订单编号', dataIndex: 'orderNo', width: 150 },
    { title: '费用代码', dataIndex: 'feeCode', width: 110 },
    { title: '费用名称', dataIndex: 'feeName', width: 140 },
    {
      title: '含税金额',
      key: 'totalAmount',
      align: 'right',
      render: (_, row) => `${formatAmount(row.totalAmount)} ${row.currency}`,
    },
    {
      title: '不可添加原因',
      key: 'blockedReasons',
      render: (_, row) => row.blockedReasons?.join('；') || '-',
    },
  ];

  return (
    <>
      {editable && changed && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 8 }}
          title="这个账单已经发生变动，请重新刷新。"
          description="增删费用会以当前显示的版本提交，刷新后再操作可避免冲突。"
          action={
            <Button size="small" type="primary" onClick={refresh}>
              刷新
            </Button>
          }
        />
      )}
      {editable && locked && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 8 }}
          title={`账单已有${lockLabels.join('、')}，不能增删费用`}
          description="如需调整费用，请先撤销对应的发票、核销或对冲记录。"
        />
      )}
      {editable && (
        <Space style={{ marginBottom: 8 }}>
          <Tooltip
            title={locked ? `账单已有${lockLabels.join('、')}` : undefined}
          >
            <Button
              disabled={locked || mutation.isPending}
              onClick={() => setPickerOpen(true)}
            >
              添加费用
            </Button>
          </Tooltip>
        </Space>
      )}
      <Table<API.FinanceBillLine>
        rowKey="id"
        size="small"
        bordered
        pagination={false}
        dataSource={bill.lines || []}
        columns={columns}
      />
      <Modal
        title={`添加费用到账单 ${bill.billNo || ''}`}
        open={pickerOpen}
        width={900}
        destroyOnHidden
        okText={`添加${selected.length ? ` ${selected.length} 笔` : ''}`}
        okButtonProps={{
          disabled: !selected.length,
          loading: mutation.isPending,
        }}
        onCancel={closePicker}
        onOk={async () => {
          if (await submit(selected, [])) closePicker();
        }}
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 8 }}
          title={`只列出与本账单同公司、同${bill.direction === 'PAYABLE' ? '应付' : '应收'}方向、同币种（${bill.currency ?? '-'}）、同结算单位（${bill.settlementPartyName ?? '-'}）的未建账费用`}
        />
        <Input.Search
          allowClear
          placeholder="输入费用代码/费用名称/订单号搜索"
          style={{ marginBottom: 8 }}
          onSearch={(value) => {
            setKeyword(value.trim());
            setPage(1);
          }}
        />
        <Table<API.FeeLedgerItem>
          rowKey="id"
          size="small"
          bordered
          loading={candidates.isFetching}
          dataSource={candidates.data?.items ?? []}
          columns={candidateColumns}
          rowSelection={{
            selectedRowKeys: selected,
            preserveSelectedRowKeys: true,
            getCheckboxProps: (row) => ({ disabled: row.canBill !== true }),
            onChange: (keys) => setSelected(keys as string[]),
          }}
          pagination={{
            current: page,
            pageSize: CANDIDATE_PAGE_SIZE,
            total: candidates.data?.total ?? 0,
            showSizeChanger: false,
            onChange: (next) => setPage(next),
          }}
        />
      </Modal>
    </>
  );
}
