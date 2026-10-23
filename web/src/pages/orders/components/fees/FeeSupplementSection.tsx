import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProFormTextArea } from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Input,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import {
  GuardedModalForm,
  ResponsiveProTable as ProTable,
} from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { orderErrorReasons } from '@/errorReasons.generated';
import {
  feeReviewCommentFieldProps,
  feeReviewCommentRule,
} from '@/features/orders/fee-review';
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
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import FeeSupplementModal, {
  type FeeSupplementFormValues,
  type FeeSupplementOptions,
} from './FeeSupplementModal';
import { PAYABLE } from './feeConstants';

type SupplementRequest = API.OrderFeeSupplementRequestData;

// 请求层已呈现失败；拒绝确认 Promise 以保留弹窗和已输入的原因。
const consumedResponseError = new Error('请求未成功完成');

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

/** 审核依据只做文案翻译，不推导业务含义。 */
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
  | 'cancel-blocked'
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
      text: serverMessage || '申请依据的费用批准版本已变化，请刷新后重新申请',
    };
  }
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_NOT_APPLICABLE) {
    return {
      kind: 'not-applicable',
      text: '订单费用尚未批准，请先完成整票费用录入并提交审核',
    };
  }
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_APPROVER_UNAVAILABLE) {
    return {
      kind: 'approver-unavailable',
      text: '当前没有任何具备费用审核权限的审批人，请先配置审批资格后再提交补录申请',
    };
  }
  if (reason === orderErrorReasons.FEE_SUPPLEMENT_CANCEL_BLOCKED) {
    return {
      kind: 'cancel-blocked',
      // 服务端 message 携带具体阻断原因（含重复撤销的明确拒绝），缺失时兜底。
      text: serverMessage || '当前不满足撤销条件，请刷新申请后查看阻断原因',
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
  /** 整票费用当前具有批准基础，撤回待审补录后可重提。 */
  feeReviewApproved: boolean;
  /** 待审期间禁止另建申请，但保留撤回重提入口。 */
  hasPendingSupplement: boolean;
  /** 审批通过生成未建账费用后，调用方刷新费用表格。 */
  onFeeTablesReload: () => void;
  /** 通知外层是否有申请记录，用于分节的默认展开。 */
  onRecordsLoaded?: (hasRecords: boolean) => void;
};

export default function FeeSupplementSection({
  orderId,
  canCreate,
  feeReviewApproved,
  hasPendingSupplement,
  feeSettings,
  settlementParties,
  currencies,
  billingUnits,
  onFeeTablesReload,
  onRecordsLoaded,
}: FeeSupplementSectionProps) {
  const [cancelReasonOpen, setCancelReasonOpen] = useState(false);
  const { markDirty: markCancelReasonDirty } = useUnsavedChangesGuard({
    enabled: cancelReasonOpen,
    identity: orderId,
  });
  const getModalOwnerSignal = useModalOwnerSignal();
  const { message, modal } = App.useApp();
  const { initialState } = useInitialState();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const requestSequenceRef = useRef(0);
  const listRequestSequenceRef = useRef(0);
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const canResubmitRef = useRef(canCreate && feeReviewApproved);
  canResubmitRef.current = canCreate && feeReviewApproved;
  const previousOrderIdRef = useRef(orderId);
  const [modalOpen, setModalOpen] = useState(false);
  const [modalOrderId, setModalOrderId] = useState<string>();
  const [resubmitRequest, setResubmitRequest] = useState<SupplementRequest>();
  const [reviewRequest, setReviewRequest] = useState<SupplementRequest>();
  const [reviewError, setReviewError] = useState('');
  const reviewQuery = useQuery({
    queryKey: [
      'order-fee-supplement-preview',
      initialState?.currentUser?.currentOrganization?.id,
      orderId,
      reviewRequest?.id,
      reviewRequest?.version,
    ],
    enabled: Boolean(reviewRequest?.id && reviewRequest.orderId === orderId),
    queryFn: async () => {
      const response = await orderFeeServicePreviewOrderFeeSupplementApproval({
        orderId,
        id: reviewRequest?.id as string,
        expectedVersion: reviewRequest?.version as string,
      });
      if (!response?.data) throw new Error('补录预览数据为空');
      return response.data;
    },
    meta: { errorMessage: '补录费用及提成影响加载失败' },
  });
  const reviewPreview =
    !reviewQuery.isError && !reviewError ? reviewQuery.data : undefined;
  const reviewLoading = reviewQuery.isFetching;
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
  }, [orderId]);

  const openReview = (record: SupplementRequest) => {
    if (!record.id || !record.version) return;
    ++reviewSequenceRef.current;
    setReviewError('');
    setReviewRequest(record);
  };

  const closeReview = () => {
    reviewSequenceRef.current += 1;
    setReviewRequest(undefined);
    setReviewError('');
  };

  const reloadList = () => actionRef.current?.reload();
  const refreshCapabilities = () => {
    void queryClient.invalidateQueries({ queryKey: ['order-fee-review'] });
    void queryClient.invalidateQueries({ queryKey: ['order-detail'] });
    void queryClient.invalidateQueries({ queryKey: ['orders'] });
    void queryClient.invalidateQueries({ queryKey: ['finance'] });
  };

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
        if (response?.success !== true || !response.data?.id) return false;
        refreshCapabilities();
        idempotencyKeyRef.current = generateUUID();
        message.success('补录申请已提交，等待审批');
        if (response.data?.approverAvailable !== true) {
          modal.warning({
            title: '当前暂无可用审批人',
            content:
              '具备该订单费用审核资格的人员当前均无法审批；申请会保持待审批状态，您可以在下方撤回该申请。',
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
        // 只有明确的内容冲突才更换身份；网络失败重试沿用原幂等键。
        if (
          reasonOf(error) ===
          orderErrorReasons.FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT
        ) {
          idempotencyKeyRef.current = generateUUID();
        }
        if (kind === 'not-applicable' || kind === 'approver-unavailable') {
          modal.warning({ title: '无法提交补录申请', content: text });
        } else {
          message.error(text || '提交补录申请失败');
        }
        return false;
      }
    });
  };

  const handleApprove = async (record: SupplementRequest, comment?: string) => {
    const targetOrderId = orderId;
    const reviewSequence = reviewSequenceRef.current;
    if (
      !targetOrderId ||
      !record.id ||
      !record.version ||
      !reviewPreview?.previewToken ||
      reviewLoading ||
      reviewQuery.isError
    )
      return false;
    await runGuarded(async (requestSequence) => {
      try {
        const response = await orderFeeServiceApproveOrderFeeSupplement(
          { orderId: targetOrderId, id: record.id as string },
          {
            orderId: targetOrderId,
            id: record.id as string,
            expectedVersion: record.version as string,
            previewToken: reviewPreview.previewToken as string,
            comment: comment?.trim(),
          },
        );
        if (isStaleResponse(requestSequence, targetOrderId)) return;
        if (response?.success !== true || response.data?.id !== record.id)
          return;
        refreshCapabilities();
        message.success('补录已批准，费用与提成净额调整已同步保存');
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
          modal.warning({
            title: '审核依据已变化，请刷新后重新审核',
            content: text,
          });
        } else {
          message.error(text || '审批失败');
        }
        reloadList();
        if (reviewSequence === reviewSequenceRef.current) {
          setReviewError('审批未完成，请关闭审核窗口并刷新申请后重试');
        }
      }
    });
    return false;
  };

  const handleReject = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    const reviewSequence = reviewSequenceRef.current;
    if (!targetOrderId || !record.id || !record.version) return;
    void confirmWithReason(
      { modal, message },
      `驳回补录申请？`,
      async (reason) => {
        await runGuarded(async (requestSequence) => {
          try {
            const response = await orderFeeServiceRejectOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
                reason,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            if (response?.success !== true || response.data?.id !== record.id)
              throw consumedResponseError;
            refreshCapabilities();
            message.success('补录申请已驳回');
            if (reviewSequence === reviewSequenceRef.current) closeReview();
            reloadList();
          } catch (error: unknown) {
            if (error === consumedResponseError) throw error;
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
            throw error;
          }
        });
      },
      {
        signal: getModalOwnerSignal(),
        placeholder: '请输入驳回原因（必填）',
        requiredMessage: '请输入驳回原因',
      },
    ).then(
      () => {},
      () => {},
    );
  };

  const handleWithdraw = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (!targetOrderId || !record.id || !record.version) return;
    const confirmation = modal.confirm({
      title: '撤回补录申请？',
      content:
        '撤回后申请进入终态，不再显示审批动作；撤回与审批并发时只有先提交的一方成功。',
      okText: '撤回',
      okButtonProps: { danger: true },
      onOk: async () => {
        await runGuarded(async (requestSequence) => {
          try {
            const response = await orderFeeServiceWithdrawOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            if (response?.success !== true || response.data?.id !== record.id)
              throw consumedResponseError;
            refreshCapabilities();
            message.success('补录申请已撤回');
            reloadList();
          } catch (error: unknown) {
            if (error === consumedResponseError) throw error;
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
            throw error;
          }
        });
      },
    });
    void confirmation.then(
      () => {},
      () => {},
    );
  };

  const handleWithdrawAndResubmit = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (
      !targetOrderId ||
      !record.id ||
      !record.version ||
      !record.canWithdraw ||
      !canCreate ||
      !feeReviewApproved
    )
      return;
    const confirmation = modal.confirm({
      title: '撤回并重新提交补录申请？',
      content:
        '原申请撤回后将成为不可修改的历史记录。撤回成功才会打开预填表单；关闭表单不会恢复原申请，也不会自动提交新申请。',
      okText: '撤回并重提',
      onOk: async () => {
        await runGuarded(async (requestSequence) => {
          try {
            const response = await orderFeeServiceWithdrawOrderFeeSupplement(
              { orderId: targetOrderId, id: record.id as string },
              {
                orderId: targetOrderId,
                id: record.id as string,
                expectedVersion: record.version as string,
              },
            );
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            if (response?.success !== true || response.data?.id !== record.id)
              throw consumedResponseError;
            refreshCapabilities();
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
            if (error === consumedResponseError) throw error;
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
            throw error;
          }
        });
      },
    });
    void confirmation.then(
      () => {},
      () => {},
    );
  };

  const handleCancelFee = (record: SupplementRequest) => {
    const targetOrderId = orderId;
    if (!targetOrderId || !record.id || !record.feeId) return;
    let cancelReason = '';
    setCancelReasonOpen(true);
    const ownerSignal = getModalOwnerSignal();
    const confirmation = modal.confirm({
      title: '撤销补录费用？',
      afterClose: () => {
        setCancelReasonOpen(false);
        ownerSignal.removeEventListener('abort', destroyCancelReason);
      },
      content: (
        <div>
          <p>
            撤销后生成的费用将被删除，审批历史保留；存在已确认提成冲减时不能撤销。
          </p>
          <p style={{ color: '#faad14' }}>
            已建账需先按现有链路取消账单；关联冲减已确认或已扣回后不能直接撤销。
          </p>
          <Input.TextArea
            autoFocus
            maxLength={500}
            showCount
            placeholder="请输入撤销原因（必填）"
            onChange={(event) => {
              markCancelReasonDirty();
              cancelReason = event.target.value.trim();
            }}
          />
        </div>
      ),
      okText: '确认撤销',
      okButtonProps: { danger: true },
      onOk: (_close) => {
        if (ownerSignal.aborted) return Promise.reject();
        if (!cancelReason) {
          message.warning('请输入撤销原因');
          return;
        }
        return runGuarded(async (requestSequence) => {
          try {
            const response =
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
            if (response?.success !== true || response.data?.id !== record.id)
              throw consumedResponseError;
            refreshCapabilities();
            message.success('补录费用已撤销删除，整票费用批准状态已刷新');
            // 响应只返回刷新后的申请数据（申请保持 APPROVED、fee_id 为空），
            // 列表与费用/汇总一律经重查刷新，不读取已删除的费用实体。
            reloadList();
            onFeeTablesReload();
          } catch (error: unknown) {
            if (error === consumedResponseError) throw error;
            if (isStaleResponse(requestSequence, targetOrderId)) return;
            const { kind, text } = describeFeeSupplementError(
              reasonOf(error),
              textOf(error),
            );
            if (kind === 'transition') {
              message.warning(text);
            } else {
              // 已建账、存在更晚补录、冲减已确认或重复撤销时展示服务端阻断原因，
              // 不引导用户使用普通删除。
              message.error(text || '撤销补录费用失败');
            }
            reloadList();
            throw error;
          }
        });
      },
    });
    function destroyCancelReason() {
      confirmation.destroy();
    }
    ownerSignal.addEventListener('abort', destroyCancelReason, { once: true });
    void confirmation.then(
      () => {},
      () => {},
    );
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
      title: '审核依据',
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
            {`${record.quantity || '-'} × ${record.unitPrice || '-'} = ${formatAmount(record.totalAmount)} ${record.currency || ''} · 发生日期 ${formatDate(record.expenseDate, 'date')}`}
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
      dataIndex: 'feeId',
      width: 110,
      render: (_, record) => {
        if (record.status !== 'APPROVED') return '-';
        // fee_id 为空表示生成费用已专用撤销删除；申请保持 APPROVED，不显示空白。
        if (!record.feeId) {
          return <Tag color="default">生成费用已删除</Tag>;
        }
        return <Tag color="blue">已生成</Tag>;
      },
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, record) => {
        const actions: React.ReactNode[] = [];
        // 审批/驳回/撤回/撤销全部只消费后端能力投影，前端不复制第二套资格规则。
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
          if (canCreate && feeReviewApproved) {
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
                撤销补录费用
              </a>,
            );
          } else if (record.cancelBlockedReason) {
            actions.push(
              <Tooltip key="cancel-blocked" title={record.cancelBlockedReason}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  不可撤销
                </Typography.Text>
              </Tooltip>,
            );
          }
        }
        return actions;
      },
    },
  ];

  const columnSettings = useColumnSettings<ProColumns<SupplementRequest>>({
    tableKey: 'orders:fee-supplements',
    columns,
  });

  return (
    <>
      {canCreate && feeReviewApproved && !hasPendingSupplement && (
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
            费用批准后，可申请新增真实应付成本；审批同时核对整票利润和提成影响。
          </Typography.Text>
        </Space>
      )}
      <ProTable<SupplementRequest>
        actionRef={actionRef}
        rowKey="id"
        columns={columnSettings.columns}
        size="small"
        search={false}
        options={{ reload: true, density: true, setting: false }}
        toolBarRender={() => [columnSettings.entry]}
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
        onLoad={(records) => onRecordsLoaded?.(records.length > 0)}
        locale={{ emptyText: '暂无补录申请' }}
      />
      <GuardedModalForm<{ comment?: string }>
        title="审核补录费用与提成影响"
        open={!!reviewRequest && reviewRequest.orderId === orderId}
        width={760}
        modalProps={{ destroyOnHidden: true, onCancel: closeReview }}
        onOpenChange={(open) => {
          if (!open) closeReview();
        }}
        onFinish={async (values) => {
          if (!reviewRequest) return false;
          await handleApprove(reviewRequest, values.comment);
          return false;
        }}
        submitter={{
          searchConfig: { submitText: '批准并确认提成影响' },
          submitButtonProps: {
            disabled:
              !reviewPreview ||
              reviewLoading ||
              reviewQuery.isError ||
              !!reviewError,
          },
          render: (_, dom) => (
            <Space>
              <Button
                danger
                onClick={() => reviewRequest && handleReject(reviewRequest)}
              >
                退回
              </Button>
              {dom}
            </Space>
          ),
        }}
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
                  children: `${trimDecimal(reviewRequest.quantity)} × ${trimDecimal(reviewRequest.unitPrice)} = ${formatAmount(reviewRequest.totalAmount)} ${reviewRequest.currency}`,
                },
                {
                  key: 'date',
                  label: '发生日期',
                  children: formatDate(reviewRequest.expenseDate, 'date'),
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
              {(!!reviewError || reviewQuery.isError) && (
                <Alert
                  type="error"
                  showIcon
                  title="无法预览毛利变化"
                  description={reviewError || '请关闭窗口后重新加载补录预览'}
                />
              )}
              {reviewPreview && !reviewLoading && !reviewError && (
                <>
                  <Alert
                    type="warning"
                    showIcon
                    title="批准后同时确认下列提成净额调整；保留已经发放或扣回的记录，财务按差额发放、补付或扣回。"
                  />
                  <Table<API.FeeSupplementCommissionImpact>
                    size="small"
                    pagination={false}
                    rowKey="commissionId"
                    dataSource={reviewPreview.commissionImpacts ?? []}
                    columns={[
                      { title: '员工', dataIndex: 'employeeName' },
                      { title: '原提成单', dataIndex: 'commissionNo' },
                      { title: '币种', dataIndex: 'baseCurrency' },
                      {
                        title: '有效提成 当前 → 批准后',
                        render: (_, item) =>
                          `${formatAmount(item.amountBefore)} → ${formatAmount(item.amountAfter)}`,
                      },
                      {
                        title: '本次系统调整',
                        render: (_, item) =>
                          `${item.adjustmentDirection === 'INCREASE' ? '+' : item.adjustmentDirection === 'DECREASE' ? '-' : ''}${formatAmount(item.adjustmentAmount ?? '0')}`,
                      },
                      {
                        title: '超过余额未冲减部分',
                        dataIndex: 'excessAmount',
                        render: (value) => formatAmount(value),
                      },
                      { title: '原提成状态', dataIndex: 'commissionStatus' },
                    ]}
                  />
                  <Typography.Title level={5}>
                    整票预计毛利影响（{reviewPreview.baseCurrency}）
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
                      {`${reviewPreview.currentProfitRate == null ? '不可计算' : `${trimDecimal(reviewPreview.currentProfitRate)}%`} → ${reviewPreview.projectedProfitRate == null ? '不可计算' : `${trimDecimal(reviewPreview.projectedProfitRate)}%`}`}
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
                    以上为当前时点估算。确认通过时服务端会按当前整票费用与汇率复核；批准后生成未建账应付费用并刷新整票费用批准，适用时直接确认提成净额调整。
                  </Typography.Text>
                </>
              )}
            </div>
          </>
        )}
        <ProFormTextArea
          name="comment"
          label={
            Number(reviewPreview?.projectedProfit) < 0
              ? '审批意见（亏损原因，必填）'
              : '审批意见'
          }
          fieldProps={feeReviewCommentFieldProps}
          rules={[
            feeReviewCommentRule,
            ...(Number(reviewPreview?.projectedProfit) < 0
              ? [
                  {
                    required: true,
                    whitespace: true,
                    message: '请说明补录后亏损原因',
                  },
                ]
              : []),
          ]}
        />
      </GuardedModalForm>
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
