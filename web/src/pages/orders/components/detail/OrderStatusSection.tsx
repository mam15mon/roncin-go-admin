import { CheckOutlined, DownOutlined } from '@ant-design/icons';
import { Button, Col, Dropdown, Popover, Space, Tag } from 'antd';
import React from 'react';
import type { OrderFormTemplateSection } from '@/components/ui';
import { getOrderFlowStatusText } from '@/constants/statusMeta';
import {
  OrderBusinessType,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { formatDate } from '@/utils/format';

const STEPS = [
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    key: 'booked',
    status: 'BOOKED',
    label: '已订舱',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
    key: 'allocated',
    status: 'SPACE_ALLOCATED',
    label: '已配舱',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
    key: 'trucked',
    status: 'TRUCKING_ARRANGED',
    label: '拖车已安排',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
    key: 'si_cutoff',
    status: 'DOCUMENT_CUTOFF',
    label: '已截单',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED,
    key: 'customs',
    status: 'CUSTOMS_DECLARATION_ARRANGED',
    label: '报关已安排',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
    key: 'released',
    status: 'DOCUMENT_RELEASED',
    label: '已放单',
  },
] as const;

type StatusAction = {
  onClick: () => void;
  disabled?: boolean;
  disabledReason?: string;
};

export type OrderStatusSectionActions = {
  transitionFlow?: (targetStatus: number) => void;
  flowDisabled?: boolean;
  flowDisabledReason?: string;
  startTermination?: StatusAction;
  completeTermination?: StatusAction;
  cancelTermination?: StatusAction;
  close?: StatusAction;
  closureDisabledReason?: string;
  reopen?: StatusAction;
};

type ActionBadgeProps = {
  label: string;
  ariaLabel: string;
  action?: StatusAction;
  disabledReason?: string;
  showDisabledButton?: boolean;
  tone: 'neutral' | 'danger' | 'success';
  record?: API.OrderStatusRecord;
};

function StatusPopover({
  label,
  record,
  disabledReason,
  children,
}: {
  label: string;
  record?: API.OrderStatusRecord;
  disabledReason?: string;
  children: React.ReactElement;
}) {
  return (
    <Popover
      title={label}
      trigger={['hover', 'focus']}
      content={
        <div style={{ maxWidth: 320, overflowWrap: 'anywhere' }}>
          {record ? (
            <>
              <div>操作人：{record.operatorName || '未记录操作人'}</div>
              <div>操作时间：{formatDate(record.changedAt, 'datetime')}</div>
              <div style={{ whiteSpace: 'pre-wrap' }}>
                备注：{record.reason?.trim() || '无备注'}
              </div>
            </>
          ) : (
            <div>暂无操作记录</div>
          )}
          {disabledReason && (
            <div style={{ marginTop: 8, color: '#64748b' }}>
              {disabledReason}
            </div>
          )}
        </div>
      }
    >
      {children}
    </Popover>
  );
}

const BADGE_TONES = {
  neutral: {
    backgroundColor: '#f1f5f9',
    color: '#475569',
    dot: '⚪',
  },
  danger: {
    backgroundColor: '#fee2e2',
    color: '#ef4444',
    dot: '🔴',
  },
  success: {
    backgroundColor: '#dcfce7',
    color: '#16a34a',
    dot: '🟢',
  },
} as const;

function ActionBadge({
  label,
  ariaLabel,
  action,
  disabledReason,
  showDisabledButton = false,
  tone,
  record,
}: ActionBadgeProps) {
  const toneStyle = BADGE_TONES[tone];
  const badgeStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 6,
    height: 24,
    padding: '2px 10px',
    border: 0,
    borderRadius: 12,
    backgroundColor: toneStyle.backgroundColor,
    color: toneStyle.color,
    fontSize: 12,
    boxShadow: 'none',
  };

  if (!action) {
    if (showDisabledButton) {
      return (
        <StatusPopover
          label={label}
          record={record}
          disabledReason={disabledReason}
        >
          <span>
            <Button aria-label={ariaLabel} disabled style={badgeStyle}>
              <span style={{ fontSize: 10 }}>{toneStyle.dot}</span>
              <span>{label}</span>
            </Button>
          </span>
        </StatusPopover>
      );
    }
    return (
      <StatusPopover
        label={label}
        record={record}
        disabledReason={disabledReason}
      >
        <span style={{ ...badgeStyle, cursor: 'help' }}>
          <span style={{ fontSize: 10 }}>{toneStyle.dot}</span>
          <span>{label}</span>
        </span>
      </StatusPopover>
    );
  }

  return (
    <StatusPopover
      label={label}
      record={record}
      disabledReason={action.disabledReason}
    >
      <span>
        <Button
          aria-label={ariaLabel}
          disabled={action.disabled}
          onClick={action.onClick}
          style={badgeStyle}
        >
          <span style={{ fontSize: 10 }}>{toneStyle.dot}</span>
          <span>{label}</span>
        </Button>
      </span>
    </StatusPopover>
  );
}

function buildTerminationBadge(
  order: API.Order | undefined,
  actions: OrderStatusSectionActions,
) {
  const status =
    order?.terminationStatus ===
    OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING
      ? 'TERMINATING'
      : order?.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED
        ? 'TERMINATED'
        : 'ACTIVE';
  const record = order?.statusRecords?.find(
    (item) => item.dimension === 'TERMINATION' && item.status === status,
  );
  switch (order?.terminationStatus) {
    case OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING: {
      const menuItems = [
        actions.completeTermination
          ? {
              key: 'complete',
              label: '完成退关',
              disabled: actions.completeTermination.disabled,
              onClick: actions.completeTermination.onClick,
            }
          : undefined,
        actions.cancelTermination
          ? {
              key: 'cancel',
              label: '取消退关',
              disabled: actions.cancelTermination.disabled,
              onClick: actions.cancelTermination.onClick,
            }
          : undefined,
      ].filter((item): item is NonNullable<typeof item> => Boolean(item));

      if (menuItems.length === 0) {
        return (
          <ActionBadge
            label="退关中"
            ariaLabel="退关中"
            tone="danger"
            record={record}
          />
        );
      }

      return (
        <StatusPopover label="退关中" record={record}>
          <Dropdown menu={{ items: menuItems }} trigger={['click']}>
            <Button
              aria-label="选择退关操作"
              style={{
                height: 24,
                padding: '2px 10px',
                border: 0,
                borderRadius: 12,
                backgroundColor: '#fee2e2',
                color: '#ef4444',
                fontSize: 12,
                boxShadow: 'none',
              }}
            >
              <span style={{ fontSize: 10 }}>🔴</span>
              退关中
              <DownOutlined style={{ fontSize: 9 }} />
            </Button>
          </Dropdown>
        </StatusPopover>
      );
    }
    case OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED:
      return (
        <ActionBadge
          label="已退关"
          ariaLabel="恢复订单"
          action={actions.cancelTermination}
          tone="danger"
          record={record}
        />
      );
    default:
      return (
        <ActionBadge
          label="未退关"
          ariaLabel="发起退关"
          action={actions.startTermination}
          tone="neutral"
          record={record}
        />
      );
  }
}

function buildClosureBadge(
  order: API.Order | undefined,
  actions: OrderStatusSectionActions,
) {
  const status =
    order?.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED
      ? 'CLOSED'
      : 'OPEN';
  const record = order?.statusRecords?.find(
    (item) => item.dimension === 'CLOSURE' && item.status === status,
  );
  if (order?.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED) {
    return (
      <ActionBadge
        label="已完结"
        ariaLabel="反结案"
        action={actions.reopen}
        tone="success"
        record={record}
      />
    );
  }

  return (
    <ActionBadge
      label="未完结"
      ariaLabel="完结订单"
      action={actions.close}
      disabledReason={actions.closureDisabledReason ?? '当前订单暂不可完结'}
      showDisabledButton
      tone="neutral"
      record={record}
    />
  );
}

export function buildOrderStatusSection(
  order?: API.Order,
  actions: OrderStatusSectionActions = {},
): OrderFormTemplateSection {
  const allowedTargets = new Set(order?.allowedTargetFlowStatuses ?? []);
  const isAir = order?.businessType === OrderBusinessType.BUSINESS_TYPE_AE;
  const steps = STEPS.map((step) => ({
    ...step,
    label:
      isAir && step.key === 'released'
        ? '已签单'
        : isAir && step.key === 'trucked'
          ? '提货已安排'
          : step.label,
  }));

  return {
    key: 'orderStatusSection',
    title: '订单状态',
    extra: (
      <Space size={12} align="center" wrap>
        <StatusPopover
          label={`当前：${getOrderFlowStatusText(order?.flowStatus, order?.businessType)}`}
          record={order?.statusRecords?.find(
            (item) =>
              item.dimension === 'FLOW' &&
              item.status ===
                (steps.find((step) => step.value === order?.flowStatus)
                  ?.status ?? 'DRAFT'),
          )}
        >
          <Tag color="blue" style={{ cursor: 'help' }}>
            当前：
            {getOrderFlowStatusText(order?.flowStatus, order?.businessType)}
          </Tag>
        </StatusPopover>
        {buildTerminationBadge(order, actions)}
        {buildClosureBadge(order, actions)}
      </Space>
    ),
    content: (
      <Col span={24}>
        <div
          style={{
            padding: '20px 40px 12px',
            position: 'relative',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div
            style={{
              position: 'absolute',
              top: 30,
              left: 60,
              right: 60,
              height: 1,
              backgroundColor: '#cbd5e1',
              zIndex: 1,
            }}
          />

          {steps.map((step) => {
            const record = order?.statusRecords?.find(
              (item) =>
                item.dimension === 'FLOW' && item.status === step.status,
            );
            const isCurrent = order?.flowStatus === step.value;
            const isPassed =
              Boolean(record) && step.value < Number(order?.flowStatus ?? 0);
            const isAllowedTarget = allowedTargets.has(step.value);
            const disabled = actions.flowDisabled || !actions.transitionFlow;
            const node = (
              <span
                aria-current={isCurrent ? 'step' : undefined}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  cursor: isAllowedTarget ? undefined : 'help',
                }}
              >
                <span
                  aria-hidden="true"
                  style={{
                    width: 14,
                    height: 14,
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: '#ffffff',
                    fontSize: 9,
                    boxSizing: 'content-box',
                    backgroundColor: isCurrent
                      ? '#1677ff'
                      : isPassed
                        ? '#52c41a'
                        : '#e2e8f0',
                    border: '3px solid #ffffff',
                    boxShadow: isCurrent
                      ? '0 0 0 2px #91caff'
                      : '0 0 0 1px #cbd5e1',
                    marginBottom: 8,
                  }}
                >
                  {isPassed && <CheckOutlined />}
                </span>
                <span
                  style={{
                    fontSize: 12,
                    color: isCurrent
                      ? '#1677ff'
                      : isPassed
                        ? '#0f172a'
                        : '#64748b',
                    fontWeight: isCurrent ? 600 : 400,
                  }}
                >
                  {step.label}
                </span>
                <span
                  style={{
                    fontSize: 11,
                    marginTop: 2,
                    minHeight: 17,
                    color: isCurrent ? '#1677ff' : '#94a3b8',
                  }}
                >
                  {isCurrent
                    ? '当前'
                    : isPassed
                      ? '已完成'
                      : isAllowedTarget
                        ? '可流转'
                        : '未完成'}
                </span>
              </span>
            );

            return (
              <div
                key={step.key}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  position: 'relative',
                  zIndex: 2,
                }}
              >
                <StatusPopover
                  label={step.label}
                  record={record}
                  disabledReason={
                    isAllowedTarget && disabled
                      ? actions.flowDisabledReason
                      : undefined
                  }
                >
                  {isAllowedTarget ? (
                    <span>
                      <Button
                        aria-label={`流转到${step.label}`}
                        type="text"
                        disabled={disabled}
                        onClick={() => actions.transitionFlow?.(step.value)}
                        style={{
                          height: 'auto',
                          padding: '0 8px 4px',
                          background: '#ffffff',
                        }}
                      >
                        {node}
                      </Button>
                    </span>
                  ) : (
                    node
                  )}
                </StatusPopover>
              </div>
            );
          })}
        </div>
      </Col>
    ),
  };
}
