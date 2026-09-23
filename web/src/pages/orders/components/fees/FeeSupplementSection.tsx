import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { App, Button, Input, Space, Tag, Tooltip, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef, useState } from 'react';
import { orderFeeStatusMeta } from '@/constants/statusMeta';
import { orderErrorReasons } from '@/errorReasons.generated';
import {
  orderFeeServiceApproveOrderFeeSupplement,
  orderFeeServiceCancelApprovedOrderFeeSupplement,
  orderFeeServiceCreateOrderFeeSupplement,
  orderFeeServiceListOrderFeeSupplementRequests,
  orderFeeServiceRejectOrderFeeSupplement,
  orderFeeServiceWithdrawOrderFeeSupplement,
} from '@/services/roncin/orderFeeService';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { generateUUID } from '@/utils/uuid';
import FeeSupplementModal, {
  type FeeSupplementFormValues,
  type FeeSupplementOptions,
} from './FeeSupplementModal';
import { feeStatusCode, PAYABLE } from './feeConstants';

type SupplementRequest = API.OrderFeeSupplementRequestData;

type SupplementError = Error & {
  data?: { message?: string; reason?: string };
  response?: { data?: { message?: string; reason?: string } };
};

/** 补录申请状态文案：PENDING 统一称「待审批」，终态只读展示。 */
export const supplementStatusMeta: Record<
  string,
  { text: string; color: string }
> = {
  PENDING: { text: '待审批', color: 'processing' },
  APPROVED: { text: '已通过', color: 'success' },
  REJECTED: { text: '已驳回', color: 'error' },
  WITHDRAWN: { text: '已撤回', color: 'default' },
};

export function supplementStatusText(status?: string): string {
  return (status && supplementStatusMeta[status]?.text) || status || '-';
}

/** 锁依据只做文案翻译，不推导业务含义。 */
export function supplementLockBasisText(lockBasis?: string): string {
  if (lockBasis === 'BUSINESS') return '业务锁';
  if (lockBasis === 'FINANCIAL') return '财务锁';
  if (lockBasis === 'BOTH') return '业务锁+财务锁';
  return lockBasis || '-';
}

export type SupplementErrorKind =
  | 'transition'
  | 'lock-basis-changed'
  | 'not-applicable'
  | 'approver-unavailable'
  | 'generic';

/**
 * 服务端稳定错误码 → 用户可理解提示。LOCK_BASIS_CHANGED 的服务端 message
 * 已携带 next_action 引导（重新发起补录或改走普通新增），原样透出。
 */
export function describeFeeSupplementError(
  reason: string,
  serverMessage: string,
): { kind: SupplementErrorKind; text: string } {
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_TRANSITION) {
    return { kind: 'transition', text: '申请状态已变化，列表已刷新' };
  }
  if (reason === orderErrorReasons.LOCK_BASIS_CHANGED) {
    return {
      kind: 'lock-basis-changed',
      text: serverMessage || '申请提交时的锁依据已全部失效，请刷新后重新申请',
    };
  }
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_NOT_APPLICABLE) {
    return {
      kind: 'not-applicable',
      text: '订单当前没有业务锁或财务锁，请直接使用「录入费用」普通入口',
    };
  }
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_APPROVER_UNAVAILABLE) {
    return {
      kind: 'approver-unavailable',
      text: '当前没有任何具备直接解锁资格的审批人，请先配置审批资格后再提交补录申请',
    };
  }
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT) {
    return {
      kind: 'generic',
      text: serverMessage || '同一补录申请的内容已变化，请刷新后重新发起',
    };
  }
  return { kind: 'generic', text: serverMessage };
}

type FeeSupplementSectionProps = FeeSupplementOptions & {
  orderId: string;
  /** 是否具备 fee.create 能力（来自前端 access 对权限清单的统一判断）。 */
  canCreate: boolean;
  /** 订单处于业务锁或财务锁下才开放补录申请入口。 */
  lockActive: boolean;
  /** 审批通过生成未建账费用后，调用方刷新费用表格。 */
  onFeeTablesReload: () => void;
};

export default function FeeSupplementSection({
  orderId,
  canCreate,
  lockActive,
  feeSettings,
  settlementParties,
  currencies,
  billingUnits,
  onFeeTablesReload,
}: FeeSupplementSectionProps) {
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const requestSequenceRef = useRef(0);
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const [modalOpen, setModalOpen] = useState(false);
  const idempotencyKeyRef = useRef(generateUUID());

  const reloadList = () => actionRef.current?.reload();

  /** 迟到响应守卫：订单切换后旧订单的异步回写全部丢弃。 */
  const isStaleResponse = (requestSequence: number, targetOrderId: string) =>
    requestSequence !== requestSequenceRef.current ||
    targetOrderId !== activeOrderIdRef.current;

  const errorOf = (error: unknown): SupplementError => error as SupplementError;
  const reasonOf = (error: unknown) =>
    errorOf(error).data?.reason ?? errorOf(error).response?.data?.reason ?? '';
  const textOf = (error: unknown) =>
    errorOf(error).data?.message ??
    errorOf(error).response?.data?.message ??
    '';

  const runGuarded = <T,>(action: (requestSequence: number) => Promise<T>) => {
    const requestSequence = ++requestSequenceRef.current;
    return action(requestSequence);
  };

  const handleCreate = async (values: FeeSupplementFormValues) => {
    const targetOrderId = orderId;
    if (!targetOrderId) return false;
    return runGuarded(async (requestSequence) => {
      const idempotencyKey = idempotencyKeyRef.current;
      try {
        const response = await orderFeeServiceCreateOrderFeeSupplement(
          { orderId: targetOrderId },
          {
            orderId: targetOrderId,
            // 补录固定应付方向；应收由服务端二次拒绝。
            direction: PAYABLE,
            feeSettingId: values.feeSettingId,
            settlementPartyId: values.settlementPartyId,
            billingUnitId: values.billingUnitId,
            quantity: values.quantity,
            unitPrice: values.unitPrice,
            currency: values.currency,
            expenseDate: dayjs(values.expenseDate).format('YYYY-MM-DD'),
            note: values.note?.trim() || undefined,
            exchangeRateOverride:
              values.exchangeRateOverride?.trim() || undefined,
            reason: values.reason.trim(),
            idempotencyKey,
          },
        );
        if (isStaleResponse(requestSequence, targetOrderId)) return true;
        idempotencyKeyRef.current = generateUUID();
        message.success('补录申请已提交，等待审批');
        if (response.data?.approverAvailable !== true) {
          modal.warning({
            title: '当前暂无可用审批人',
            content:
              '具备该订单直接解锁资格的人员当前均无法审批；申请会保持待审批状态，您可以在下方撤回该申请。',
          });
        }
        setModalOpen(false);
        reloadList();
        return true;
      } catch (error: unknown) {
        if (isStaleResponse(requestSequence, targetOrderId)) return false;
        const { kind, text } = describeFeeSupplementError(
          reasonOf(error),
          textOf(error),
        );
        // 幂等冲突后更换幂等键，允许用户修改内容重新发起。
        idempotencyKeyRef.current = generateUUID();
        if (kind === 'not-applicable' || kind === 'approver-unavailable') {
          modal.warning({ title: '无法提交补录申请', content: text });
        } else {
          message.error(text || '提交补录申请失败');
        }
        return false;
      }
    });
  };

  const handleApprove = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (!targetOrderId || !record.id || !record.version) return;
    modal.confirm({
      title: `通过补录申请并生成应付费用？`,
      content:
        '通过后将按申请快照原样生成一条未建账应付费用，可能同时生成提成冲减建议；审批不会修改订单锁定状态。',
      okText: '通过',
      onOk: async () => {
        await runGuarded(async (requestSequence) => {
          try {
            await orderFeeServiceApproveOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            message.success('补录申请已通过，费用已生成');
            reloadList();
            onFeeTablesReload();
          } catch (error: unknown) {
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            const { kind, text } = describeFeeSupplementError(
              reasonOf(error),
              textOf(error),
            );
            if (kind === 'transition') {
              message.warning(text);
            } else if (kind === 'lock-basis-changed') {
              modal.warning({ title: '锁依据已变化，无法审批', content: text });
            } else {
              message.error(text || '审批失败');
            }
            reloadList();
          }
        });
      },
    });
  };

  const handleReject = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (!targetOrderId || !record.id || !record.version) return;
    confirmWithReason(
      { modal, message },
      `驳回补录申请？`,
      async (reason) => {
        await runGuarded(async (requestSequence) => {
          try {
            await orderFeeServiceRejectOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
                reason,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            message.success('补录申请已驳回');
            reloadList();
          } catch (error: unknown) {
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            const { kind, text } = describeFeeSupplementError(
              reasonOf(error),
              textOf(error),
            );
            if (kind === 'transition') {
              message.warning(text);
            } else {
              message.error(text || '驳回失败');
            }
            reloadList();
          }
        });
      },
      {
        placeholder: '请输入驳回原因（必填）',
        requiredMessage: '请输入驳回原因',
      },
    );
  };

  const handleWithdraw = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (!targetOrderId || !record.id || !record.version) return;
    modal.confirm({
      title: '撤回补录申请？',
      content:
        '撤回后申请进入终态，不再显示审批动作；撤回与审批并发时只有先提交的一方成功。',
      okText: '撤回',
      okButtonProps: { danger: true },
      onOk: async () => {
        await runGuarded(async (requestSequence) => {
          try {
            await orderFeeServiceWithdrawOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            message.success('补录申请已撤回');
            reloadList();
          } catch (error: unknown) {
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            const { kind, text } = describeFeeSupplementError(
              reasonOf(error),
              textOf(error),
            );
            // 撤回与审批竞争失败属于稳定冲突：提示用户刷新，不静默重试。
            if (kind === 'transition') {
              modal.warning({
                title: '申请状态已变化',
                content:
                  '该申请已被其他人员处理（可能已审批或已撤回），列表已刷新。',
              });
            } else {
              message.error(text || '撤回失败');
            }
            reloadList();
          }
        });
      },
    });
  };

  const handleCancelFee = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (!targetOrderId || !record.id || !record.feeId) return;
    let cancelReason = '';
    modal.confirm({
      title: '作废该补录生成的费用？',
      content: (
        <div>
          <p>
            作废后费用转为「已作废」，仍为待处理的关联冲减建议会同步取消；
            APPROVED 申请历史保持不变。
          </p>
          <p style={{ color: '#faad14' }}>
            已建账需先按现有链路取消账单；关联冲减已确认或已扣回后不能直接作废。
          </p>
          <Input.TextArea
            autoFocus
            maxLength={500}
            showCount
            placeholder="请输入作废原因（必填）"
            onChange={(event) => {
              cancelReason = event.target.value.trim();
            }}
          />
        </div>
      ),
      okText: '确认作废',
      okButtonProps: { danger: true },
      onOk: (_close) => {
        if (!cancelReason) {
          message.warning('请输入作废原因');
          return;
        }
        return runGuarded(async (requestSequence) => {
          try {
            await orderFeeServiceCancelApprovedOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
                reason: cancelReason,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            message.success('补录费用已作废，关联的待处理冲减建议已同步取消');
            reloadList();
            onFeeTablesReload();
          } catch (error: unknown) {
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            const { kind, text } = describeFeeSupplementError(
              reasonOf(error),
              textOf(error),
            );
            if (kind === 'transition') {
              message.warning(text);
            } else {
              // 已建账、存在更晚补录或冲减已确认时展示服务端阻断原因，
              // 不引导用户使用普通删除。
              message.error(text || '作废补录费用失败');
            }
            reloadList();
          }
        });
      },
    });
  };

  const columns: ProColumns<SupplementRequest>[] = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => {
        const meta = supplementStatusMeta[record.status || ''];
        return (
          <Tag color={meta?.color || 'default'}>
            {meta?.text || record.status || '-'}
          </Tag>
        );
      },
    },
    {
      title: '锁依据',
      dataIndex: 'lockBasis',
      width: 110,
      renderText: (value: string) => supplementLockBasisText(value),
    },
    {
      title: '补录费用（应付）',
      dataIndex: 'feeName',
      width: 230,
      render: (_, record) => (
        <Space orientation="vertical" size={0}>
          <span>{record.feeName || record.feeCode || '-'}</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {`${record.quantity || '-'} × ${record.unitPrice || '-'} = ${record.totalAmount || '-'} ${record.currency || ''} · 发生日期 ${record.expenseDate || '-'}`}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: '补录原因',
      dataIndex: 'reason',
      width: 200,
      ellipsis: true,
      renderText: (value: string) => value || '-',
    },
    {
      title: '发起人 / 时间',
      dataIndex: 'requestedAt',
      width: 180,
      render: (_, record) => (
        <Space orientation="vertical" size={0}>
          <Typography.Text style={{ fontSize: 12 }} ellipsis>
            {record.requestedBy || '-'}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {record.requestedAt
              ? dayjs(record.requestedAt).format('YYYY-MM-DD HH:mm')
              : '-'}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: '审批 / 决定',
      dataIndex: 'decidedAt',
      width: 180,
      render: (_, record) => {
        if (record.status === 'PENDING') {
          return record.approverAvailable !== true ? (
            <Tooltip title="当前没有具备实时资格的审批人；发起人仍可撤回，新获得资格的人员可接手处理。">
              <Tag color="warning">暂无可用审批人</Tag>
            </Tooltip>
          ) : (
            <Typography.Text type="secondary">等待审批</Typography.Text>
          );
        }
        return (
          <Space orientation="vertical" size={0}>
            <Typography.Text style={{ fontSize: 12 }} ellipsis>
              {record.decidedBy || '-'}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.decidedAt
                ? dayjs(record.decidedAt).format('YYYY-MM-DD HH:mm')
                : '-'}
              {record.decisionReason ? ` · ${record.decisionReason}` : ''}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: '生成费用',
      dataIndex: 'feeStatus',
      width: 100,
      render: (_, record) => {
        if (record.status !== 'APPROVED') return '-';
        if (!record.feeStatus) return '-';
        const meta = orderFeeStatusMeta[feeStatusCode(record.feeStatus)];
        return <Tag color={meta?.color}>{meta?.text || record.feeStatus}</Tag>;
      },
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, record) => {
        const actions: React.ReactNode[] = [];
        // 审批/驳回/撤回/作废全部只消费后端能力投影，前端不复制第二套资格规则。
        if (record.canApprove) {
          actions.push(
            <a key="approve" onClick={() => handleApprove(record)}>
              通过
            </a>,
            <a key="reject" onClick={() => handleReject(record)}>
              驳回
            </a>,
          );
        }
        if (record.canWithdraw) {
          actions.push(
            <a key="withdraw" onClick={() => handleWithdraw(record)}>
              撤回申请
            </a>,
          );
        }
        if (record.status === 'APPROVED') {
          if (record.canCancel) {
            actions.push(
              <a key="cancel-fee" onClick={() => handleCancelFee(record)}>
                作废补录费用
              </a>,
            );
          } else if (record.cancelBlockedReason) {
            actions.push(
              <Tooltip key="cancel-blocked" title={record.cancelBlockedReason}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  不可作废
                </Typography.Text>
              </Tooltip>,
            );
          }
        }
        return actions;
      },
    },
  ];

  return (
    <>
      {canCreate && lockActive && (
        <Space style={{ marginBottom: 12 }}>
          <Button
            type="primary"
            onClick={() => {
              idempotencyKeyRef.current = generateUUID();
              setModalOpen(true);
            }}
          >
            补录费用
          </Button>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            锁定订单无法使用普通录入时，可申请补录应付成本，经具备直接解锁资格的人员审批后入账。
          </Typography.Text>
        </Space>
      )}
      <ProTable<SupplementRequest>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        size="small"
        search={false}
        options={false}
        params={{ orderId }}
        pagination={{ pageSize: 10, hideOnSinglePage: true }}
        request={async (params) => {
          const targetOrderId =
            (params as { orderId?: string }).orderId || orderId;
          const requestSequence = ++requestSequenceRef.current;
          const response = await orderFeeServiceListOrderFeeSupplementRequests({
            orderId: targetOrderId,
            page: params.current ?? 1,
            pageSize: params.pageSize ?? 10,
          });
          if (isStaleResponse(requestSequence, targetOrderId)) {
            return { data: [], success: true };
          }
          const data = response.data?.items ?? [];
          return {
            data,
            success: response.success ?? true,
            total: response.data?.total ?? data.length,
          };
        }}
        locale={{ emptyText: '暂无补录申请' }}
      />
      <FeeSupplementModal
        orderId={orderId}
        open={modalOpen}
        onOpenChange={setModalOpen}
        feeSettings={feeSettings}
        settlementParties={settlementParties}
        currencies={currencies}
        billingUnits={billingUnits}
        onSubmit={handleCreate}
      />
    </>
  );
}
