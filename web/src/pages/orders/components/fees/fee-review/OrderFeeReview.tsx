import {
  ModalForm,
  ProFormCheckbox,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Drawer, Space, Spin, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  FeeReviewStatus,
  FeeReviewSummary,
  feeReviewCommentFieldProps,
  feeReviewCommentRule,
  useOrderFeeReview,
} from '@/features/orders/fee-review';
import {
  orderFeeServiceApproveOrderFeeReview,
  orderFeeServiceRejectOrderFeeReview,
  orderFeeServiceRevokeOrderFeeReview,
  orderFeeServiceSubmitOrderFeeReview,
  orderFeeServiceWithdrawOrderFeeReview,
} from '@/services/roncin/orderFeeService';
import { generateUUID } from '@/utils/uuid';
import { FeeReviewDetails } from './FeeReviewDetails';

type Action = 'submit' | 'withdraw' | 'approve' | 'reject' | 'revoke';
const titles: Record<Action, string> = {
  submit: '提交整票费用审核',
  withdraw: '撤回审核',
  approve: '批准整票费用',
  reject: '退回费用',
  revoke: '撤销批准',
};
const requests = {
  submit: orderFeeServiceSubmitOrderFeeReview,
  withdraw: orderFeeServiceWithdrawOrderFeeReview,
  approve: orderFeeServiceApproveOrderFeeReview,
  reject: orderFeeServiceRejectOrderFeeReview,
  revoke: orderFeeServiceRevokeOrderFeeReview,
};

export function OrderFeeReviewActions({
  data,
  onChanged,
}: {
  data: API.OrderFeeReviewData;
  onChanged: () => void;
}) {
  const { message } = App.useApp();
  const client = useQueryClient();
  const [action, setAction] = useState<Action>();
  const actionKey = useRef(generateUUID());
  const actionSnapshot = useRef(data);
  const mutation = useMutation({
    mutationFn: async ({
      action: target,
      values,
    }: {
      action: Action;
      values: { comment?: string; completenessConfirmed?: boolean };
    }) => {
      const snapshot = actionSnapshot.current;
      const response = await requests[target](
        { orderId: snapshot.orderId as string },
        {
          orderId: snapshot.orderId,
          expectedReviewVersion: snapshot.reviewVersion,
          expectedContentVersion: snapshot.contentVersion,
          submissionId: snapshot.submissionId,
          previewToken: snapshot.previewToken,
          idempotencyKey: actionKey.current,
          comment: values.comment?.trim(),
          completenessConfirmed: values.completenessConfirmed,
        },
      );
      return response?.success === true &&
        response.data?.orderId === snapshot.orderId
        ? response.data
        : undefined;
    },
    onSuccess: async (result) => {
      if (!result) return;
      await Promise.all([
        client.invalidateQueries({ queryKey: ['order-fee-review'] }),
        client.invalidateQueries({ queryKey: ['orders'] }),
        client.invalidateQueries({ queryKey: ['finance'] }),
      ]);
      message.success('费用审核状态已更新');
      onChanged();
    },
  });
  const open = (target: Action) => {
    actionSnapshot.current = data;
    actionKey.current = generateUUID();
    setAction(target);
  };
  const requiredComment =
    action === 'reject' ||
    action === 'revoke' ||
    (action === 'approve' &&
      Number(actionSnapshot.current.summary?.profit) < 0);
  return (
    <>
      <Space wrap>
        {data.canSubmit && (
          <Button type="primary" onClick={() => open('submit')}>
            提交整票费用审核
          </Button>
        )}
        {data.canWithdraw && (
          <Button onClick={() => open('withdraw')}>撤回审核</Button>
        )}
        {data.canApprove && (
          <>
            <Button type="primary" onClick={() => open('approve')}>
              批准整票费用
            </Button>
            <Button danger onClick={() => open('reject')}>
              退回
            </Button>
          </>
        )}
        {data.canRevoke && (
          <Button danger onClick={() => open('revoke')}>
            撤销批准
          </Button>
        )}
      </Space>
      {action && (
        <ModalForm<{ comment?: string; completenessConfirmed?: boolean }>
          key={action}
          title={titles[action]}
          open
          width={action === 'approve' || action === 'submit' ? '92vw' : 680}
          modalProps={{
            destroyOnHidden: true,
            styles: {
              body: { maxHeight: 'calc(100vh - 180px)', overflowY: 'auto' },
            },
          }}
          onOpenChange={(open) => {
            if (!open) setAction(undefined);
          }}
          onFinish={async (values) => {
            try {
              return Boolean(await mutation.mutateAsync({ action, values }));
            } catch {
              return false;
            }
          }}
        >
          <div style={{ display: 'grid', gap: 12, marginBottom: 12 }}>
            <Typography.Text type="secondary">
              {actionSnapshot.current.orderNo} ·
              整票应收、应付与利润一起核对，明细无需逐条审批。
            </Typography.Text>
            <FeeReviewSummary summary={actionSnapshot.current.summary} />
            {(action === 'approve' || action === 'submit') && (
              <FeeReviewDetails
                data={actionSnapshot.current}
                showHistory={false}
              />
            )}
          </div>
          {action === 'submit' && (
            <ProFormCheckbox
              name="completenessConfirmed"
              rules={[
                {
                  validator: (_, value) =>
                    value
                      ? Promise.resolve()
                      : Promise.reject(new Error('请确认整票费用录入完整')),
                },
              ]}
            >
              我已核对整票应收与应付，确认费用录入完整
            </ProFormCheckbox>
          )}
          {action === 'revoke' && (
            <Alert
              type="warning"
              showIcon
              title="撤销后整票费用必须重新提交审核，已有业务限制及提成保护仍然生效。"
            />
          )}
          {action === 'approve' &&
            Number(actionSnapshot.current.summary?.profit) < 0 && (
              <Alert
                type="error"
                showIcon
                title="该订单亏损，请在审批意见中说明亏损原因。"
              />
            )}
          <ProFormTextArea
            name="comment"
            label={
              action === 'reject' || action === 'revoke' ? '原因' : '审批意见'
            }
            rules={[
              feeReviewCommentRule,
              ...(requiredComment
                ? [
                    {
                      required: true,
                      whitespace: true,
                      message: '请填写有效意见或原因',
                    },
                  ]
                : []),
            ]}
            fieldProps={feeReviewCommentFieldProps}
          />
        </ModalForm>
      )}
    </>
  );
}

export function OrderFeeReviewSection({
  orderId,
  onChanged,
}: {
  orderId: string;
  onChanged: () => void;
}) {
  const review = useOrderFeeReview(orderId);
  const [open, setOpen] = useState(false);
  return (
    <>
      {review.isPending && <Spin />}
      {review.isError && (
        <Alert
          type="error"
          showIcon
          title="费用审核信息加载失败，暂不能执行费用操作"
          action={<Button onClick={() => void review.refetch()}>重试</Button>}
        />
      )}
      {!review.isError && review.data && (
        <Space
          orientation="vertical"
          style={{ width: '100%', marginBottom: 12 }}
        >
          <Space wrap>
            <FeeReviewStatus
              status={review.data.status}
              pendingSupplement={review.data.hasPendingSupplement}
            />
            <Button onClick={() => setOpen(true)}>
              查看整票费用包与审核记录
            </Button>
            <OrderFeeReviewActions data={review.data} onChanged={onChanged} />
          </Space>
          {(review.data.blockedReasons ?? []).length > 0 && (
            <Alert
              type="info"
              showIcon
              title="当前费用操作限制"
              description={review.data.blockedReasons?.join('；')}
            />
          )}
          {(Boolean(review.data.notificationFailedCount) ||
            Boolean(review.data.notificationUnboundCount)) && (
            <Alert
              type="warning"
              showIcon
              title={`钉钉提醒：${review.data.notificationFailedCount ?? 0}条投递失败，${review.data.notificationUnboundCount ?? 0}人未绑定；可在站内按权限处理。`}
            />
          )}
        </Space>
      )}
      <OrderFeeReviewDrawer
        orderId={open ? orderId : undefined}
        onClose={() => setOpen(false)}
        onChanged={onChanged}
      />
    </>
  );
}

export function OrderFeeReviewDrawer({
  orderId,
  onClose,
  onChanged,
}: {
  orderId?: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const review = useOrderFeeReview(orderId);
  return (
    <Drawer
      open={Boolean(orderId)}
      title={`整票费用包审核 ${review.data?.orderNo ?? ''}`}
      size="92vw"
      styles={{ body: { padding: 16 } }}
      footer={
        !review.isError && review.data ? (
          <OrderFeeReviewActions data={review.data} onChanged={onChanged} />
        ) : undefined
      }
      destroyOnHidden
      onClose={onClose}
    >
      {review.isPending && <Spin />}
      {review.isError && (
        <Alert
          type="error"
          showIcon
          title="审核详情加载失败，请重新加载"
          action={
            <Button onClick={() => void review.refetch()}>重新加载</Button>
          }
        />
      )}
      {!review.isError && review.data && (
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <FeeReviewStatus
            status={review.data.status}
            pendingSupplement={review.data.hasPendingSupplement}
          />
          <FeeReviewSummary summary={review.data.summary} />
          {!!review.data.blockedReasons?.length && (
            <Alert type="info" title={review.data.blockedReasons.join('；')} />
          )}
          <FeeReviewDetails data={review.data} />
        </Space>
      )}
    </Drawer>
  );
}
