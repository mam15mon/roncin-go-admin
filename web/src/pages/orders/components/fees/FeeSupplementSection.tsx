import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Input,
  Modal,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { orderFeeStatusMeta } from '@/constants/statusMeta';
import { orderErrorReasons } from '@/errorReasons.generated';
import {
  orderFeeServiceApproveOrderFeeSupplement,
  orderFeeServiceCancelApprovedOrderFeeSupplement,
  orderFeeServiceCreateOrderFeeSupplement,
  orderFeeServiceListOrderFeeSupplementRequests,
  orderFeeServicePreviewOrderFeeSupplementApproval,
  orderFeeServiceRejectOrderFeeSupplement,
  orderFeeServiceWithdrawOrderFeeSupplement,
} from '@/services/roncin/orderFeeService';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { formatAmount, trimDecimal } from '@/utils/format';
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
  const listRequestSequenceRef = useRef(0);
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const canResubmitRef = useRef(canCreate && lockActive);
  canResubmitRef.current = canCreate && lockActive;
  const previousOrderIdRef = useRef(orderId);
  const [modalOpen, setModalOpen] = useState(false);
  const [modalOrderId, setModalOrderId] = useState<string>();
  const [resubmitRequest, setResubmitRequest] = useState<SupplementRequest>();
  const [reviewRequest, setReviewRequest] = useState<SupplementRequest>();
  const [reviewPreview, setReviewPreview] =
    useState<API.PreviewOrderFeeSupplementApprovalData>();
  const [reviewLoading, setReviewLoading] = useState(false);
  const [reviewSubmitting, setReviewSubmitting] = useState(false);
  const [reviewError, setReviewError] = useState('');
  const reviewSequenceRef = useRef(0);
  const idempotencyKeyRef = useRef(generateUUID());

  useEffect(() => {
    if (previousOrderIdRef.current === orderId) return;
    previousOrderIdRef.current = orderId;
    setModalOpen(false);
    setResubmitRequest(undefined);
    setModalOrderId(undefined);
    reviewSequenceRef.current += 1;
    setReviewRequest(undefined);
    setReviewPreview(undefined);
  }, [orderId]);

  const openReview = (record: SupplementRequest) => {
    if (!record.id || !record.version) return;
    const targetOrderId = orderId;
    const sequence = ++reviewSequenceRef.current;
    setReviewRequest(record);
    setReviewPreview(undefined);
    setReviewError('');
    setReviewLoading(true);
    orderFeeServicePreviewOrderFeeSupplementApproval({
      orderId: targetOrderId,
      id: record.id,
      // expectedVersion 必须位于 params 槽位（生成客户端第二参是请求配置对象），
      // 否则服务端收不到 expected_version 直接按参数无效拒绝。
      expectedVersion: record.version as string,
    })
      .then((response) => {
        if (
          sequence !== reviewSequenceRef.current ||
          targetOrderId !== activeOrderIdRef.current
        )
          return;
        if (!response.data) throw new Error('预览数据为空');
        setReviewPreview(response.data);
      })
      .catch((error: unknown) => {
        if (
          sequence !== reviewSequenceRef.current ||
          targetOrderId !== activeOrderIdRef.current
        )
          return;
        setReviewPreview(undefined);
        setReviewError(textOf(error) || '毛利预览加载失败，请刷新申请后重试');
      })
      .finally(() => {
        if (
          sequence === reviewSequenceRef.current &&
          targetOrderId === activeOrderIdRef.current
        )
          setReviewLoading(false);
      });
  };

  const closeReview = () => {
    reviewSequenceRef.current += 1;
    setReviewRequest(undefined);
    setReviewPreview(undefined);
    setReviewError('');
  };

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

  const handleApprove = async (record: SupplementRequest) => {
    const targetOrderId = orderId;
    const reviewSequence = reviewSequenceRef.current;
    if (
      !targetOrderId ||
      !record.id ||
      !record.version ||
      !reviewPreview ||
      reviewSubmitting
    )
      return;
    setReviewSubmitting(true);
    try {
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
          if (reviewSequence === reviewSequenceRef.current) closeReview();
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
          if (reviewSequence === reviewSequenceRef.current) {
            setReviewPreview(undefined);
            setReviewError('审批未完成，请关闭审核窗口并刷新申请后重试');
          }
        }
      });
    } finally {
      if (targetOrderId === activeOrderIdRef.current)
        setReviewSubmitting(false);
    }
  };

  const handleReject = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    const reviewSequence = reviewSequenceRef.current;
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
            if (reviewSequence === reviewSequenceRef.current) closeReview();
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

  const handleWithdrawAndResubmit = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (
      !targetOrderId ||
      !record.id ||
      !record.version ||
      !record.canWithdraw ||
      !canCreate ||
      !lockActive
    )
      return;
    modal.confirm({
      title: '撤回并重新提交补录申请？',
      content:
        '原申请撤回后将成为不可修改的历史记录。撤回成功才会打开预填表单；关闭表单不会恢复原申请，也不会自动提交新申请。',
      okText: '撤回并重提',
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
            if (!canResubmitRef.current) {
              message.warning('原申请已撤回，当前订单已不符合重提条件');
              reloadList();
              return;
            }
            idempotencyKeyRef.current = generateUUID();
            setResubmitRequest(record);
            setModalOrderId(targetOrderId);
            setModalOpen(true);
            message.success('原申请已撤回，请核对并提交新申请');
            reloadList();
          } catch (error: unknown) {
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            const { kind, text } = describeFeeSupplementError(
              reasonOf(error),
              textOf(error),
            );
            if (kind === 'transition') {
              modal.warning({
                title: '申请状态已变化',
                content: '该申请已被其他人员处理，列表已刷新。',
              });
            } else {
              message.error(text || '撤回失败，未打开重提表单');
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
            {record.requestedByName || '-'}
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
              {record.decidedByName || '-'}
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
            <a key="review" onClick={() => openReview(record)}>
              审核
            </a>,
          );
        }
        if (record.canWithdraw) {
          actions.push(
            <a key="withdraw" onClick={() => handleWithdraw(record)}>
              撤回申请
            </a>,
          );
          if (canCreate && lockActive) {
            actions.push(
              <a
                key="withdraw-resubmit"
                onClick={() => handleWithdrawAndResubmit(record)}
              >
                撤回并重提
              </a>,
            );
          }
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
              setResubmitRequest(undefined);
              setModalOrderId(orderId);
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
          const requestSequence = ++listRequestSequenceRef.current;
          const response = await orderFeeServiceListOrderFeeSupplementRequests({
            orderId: targetOrderId,
            page: params.current ?? 1,
            pageSize: params.pageSize ?? 10,
          });
          if (
            requestSequence !== listRequestSequenceRef.current ||
            targetOrderId !== activeOrderIdRef.current
          ) {
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
      <Modal
        title="审核补录费用"
        open={!!reviewRequest && reviewRequest.orderId === orderId}
        width={760}
        onCancel={closeReview}
        footer={
          <Space>
            <Button onClick={closeReview}>取消</Button>
            <Button
              danger
              disabled={reviewSubmitting}
              onClick={() => reviewRequest && handleReject(reviewRequest)}
            >
              驳回
            </Button>
            <Button
              type="primary"
              loading={reviewSubmitting}
              disabled={!reviewPreview || reviewLoading || !!reviewError}
              onClick={() => reviewRequest && void handleApprove(reviewRequest)}
            >
              确认通过
            </Button>
          </Space>
        }
      >
        {reviewRequest && (
          <>
            <Descriptions
              size="small"
              column={2}
              bordered
              items={[
                {
                  key: 'fee',
                  label: '费用项目',
                  children:
                    reviewRequest.feeName || reviewRequest.feeCode || '-',
                },
                {
                  key: 'party',
                  label: '结算单位',
                  children: reviewRequest.settlementPartyName || '-',
                },
                {
                  key: 'amount',
                  label: '申请金额',
                  children: `${trimDecimal(reviewRequest.quantity)} × ${trimDecimal(reviewRequest.unitPrice)} = ${trimDecimal(reviewRequest.totalAmount)} ${reviewRequest.currency}`,
                },
                {
                  key: 'date',
                  label: '发生日期',
                  children: reviewRequest.expenseDate || '-',
                },
                {
                  key: 'reason',
                  label: '补录原因',
                  children: reviewRequest.reason || '-',
                  span: 2,
                },
              ]}
            />
            <div style={{ marginTop: 16 }}>
              {reviewLoading && (
                <Spin description="正在复核毛利影响">
                  <div style={{ height: 120 }} />
                </Spin>
              )}
              {!!reviewError && (
                <Alert
                  type="error"
                  showIcon
                  title="无法预览毛利变化"
                  description={reviewError}
                />
              )}
              {reviewPreview && !reviewLoading && !reviewError && (
                <>
                  <Typography.Title level={5}>
                    预计毛利影响（{reviewPreview.baseCurrency}）
                  </Typography.Title>
                  <div
                    style={{
                      background: '#f5f7fa',
                      border: '1px solid #e5eaf0',
                      borderRadius: 8,
                      padding: '12px 16px',
                      marginBottom: 12,
                    }}
                  >
                    <Typography.Text type="secondary">
                      毛利率 当前 → 预计
                    </Typography.Text>
                    <Typography.Title
                      level={3}
                      style={{ margin: '4px 0 0', color: '#cf1322' }}
                    >
                      {`${reviewPreview.currentProfitRate == null ? '不可计算' : `${formatAmount(reviewPreview.currentProfitRate, 2)}%`} → ${reviewPreview.projectedProfitRate == null ? '不可计算' : `${formatAmount(reviewPreview.projectedProfitRate, 2)}%`}`}
                    </Typography.Title>
                  </div>
                  <Descriptions
                    size="small"
                    column={2}
                    bordered
                    items={[
                      {
                        key: 'cost',
                        label: '本次补录成本',
                        children: formatAmount(reviewPreview.supplementCost),
                        span: 2,
                      },
                      {
                        key: 'receivable',
                        label: '应收 当前 → 预计',
                        children: `${formatAmount(reviewPreview.currentReceivable)} → ${formatAmount(reviewPreview.projectedReceivable)}`,
                      },
                      {
                        key: 'payable',
                        label: '应付 当前 → 预计',
                        children: `${formatAmount(reviewPreview.currentPayable)} → ${formatAmount(reviewPreview.projectedPayable)}`,
                      },
                      {
                        key: 'profit',
                        label: '毛利 当前 → 预计',
                        children: `${formatAmount(reviewPreview.currentProfit)} → ${formatAmount(reviewPreview.projectedProfit)}`,
                        span: 2,
                      },
                      {
                        key: 'change',
                        label: '毛利变化',
                        children: (
                          <Typography.Text strong type="danger">
                            {formatAmount(reviewPreview.profitChange)}{' '}
                            {reviewPreview.baseCurrency}
                          </Typography.Text>
                        ),
                        span: 2,
                      },
                    ]}
                  />
                  <Typography.Text
                    type="secondary"
                    style={{ display: 'block', marginTop: 8 }}
                  >
                    以上为当前时点估算。确认通过时服务端会按最新费用、汇率和锁依据重新复核；通过后生成未建账应付费用，可能产生提成冲减建议。
                  </Typography.Text>
                </>
              )}
            </div>
          </>
        )}
      </Modal>
      <FeeSupplementModal
        key={resubmitRequest?.id ?? 'new'}
        orderId={orderId}
        open={modalOpen && modalOrderId === orderId}
        onOpenChange={(next) => {
          setModalOpen(next);
          if (!next) setResubmitRequest(undefined);
        }}
        initialRequest={resubmitRequest}
        feeSettings={feeSettings}
        settlementParties={settlementParties}
        currencies={currencies}
        billingUnits={billingUnits}
        onSubmit={handleCreate}
      />
    </>
  );
}
