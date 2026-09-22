import { DownOutlined } from '@ant-design/icons';
import { Button, Col, Dropdown, Space, Tag, Tooltip } from 'antd';
import React from 'react';
import type { OrderFormTemplateSection } from '@/components/ui';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';

const STEPS = [
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    key: 'booked',
    label: '已订舱',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
    key: 'allocated',
    label: '已配舱',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
    key: 'trucked',
    label: '拖车已安排',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
    key: 'si_cutoff',
    label: '已截单',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED,
    key: 'customs',
    label: '报关已安排',
  },
  {
    value: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
    key: 'released',
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
};

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
        <Tooltip title={disabledReason}>
          <span>
            <Button aria-label={ariaLabel} disabled style={badgeStyle}>
              <span style={{ fontSize: 10 }}>{toneStyle.dot}</span>
              <span>{label}</span>
            </Button>
          </span>
        </Tooltip>
      );
    }
    return (
      <Tooltip title={disabledReason}>
        <span style={{ ...badgeStyle, cursor: 'default' }}>
          <span style={{ fontSize: 10 }}>{toneStyle.dot}</span>
          <span>{label}</span>
        </span>
      </Tooltip>
    );
  }

  return (
    <Tooltip title={action.disabledReason}>
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
    </Tooltip>
  );
}

function buildTerminationBadge(
  order: API.Order | undefined,
  actions: OrderStatusSectionActions,
) {
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
        return <ActionBadge label="退关中" ariaLabel="退关中" tone="danger" />;
      }

      return (
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
      );
    }
    case OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED:
      return (
        <ActionBadge
          label="已退关"
          ariaLabel="恢复订单"
          action={actions.cancelTermination}
          tone="danger"
        />
      );
    default:
      return (
        <ActionBadge
          label="未退关"
          ariaLabel="发起退关"
          action={actions.startTermination}
          tone="neutral"
        />
      );
  }
}

function buildClosureBadge(
  order: API.Order | undefined,
  actions: OrderStatusSectionActions,
) {
  if (order?.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED) {
    return (
      <ActionBadge
        label="已完结"
        ariaLabel="反结案"
        action={actions.reopen}
        tone="success"
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
    />
  );
}

export function buildOrderStatusSection(
  order?: API.Order,
  actions: OrderStatusSectionActions = {},
): OrderFormTemplateSection {
  const allowedTargets = new Set(order?.allowedTargetFlowStatuses ?? []);

  return {
    key: 'orderStatusSection',
    title: '订单状态',
    extra: (
      <Space size={12} align="center" wrap>
        <Tag color="blue">海运出口固定流程</Tag>
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
              top: 27,
              left: 60,
              right: 60,
              height: 1,
              backgroundColor: '#cbd5e1',
              zIndex: 1,
            }}
          />

          {STEPS.map((step) => {
            const isPassed = Number(order?.flowStatus ?? 0) >= step.value;
            const isAllowedTarget = allowedTargets.has(step.value);
            const disabled = actions.flowDisabled || !actions.transitionFlow;
            const node = (
              <span
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                }}
              >
                <span
                  aria-hidden="true"
                  style={{
                    width: 14,
                    height: 14,
                    borderRadius: '50%',
                    backgroundColor: isPassed
                      ? '#1677ff'
                      : isAllowedTarget
                        ? '#4096ff'
                        : '#94a3b8',
                    border: '3px solid #ffffff',
                    boxShadow: isAllowedTarget
                      ? '0 0 0 2px #91caff'
                      : '0 0 0 1px #cbd5e1',
                    marginBottom: 8,
                  }}
                />
                <span
                  style={{
                    fontSize: 12,
                    color: isPassed
                      ? '#0f172a'
                      : isAllowedTarget
                        ? '#1677ff'
                        : '#64748b',
                    fontWeight: isPassed || isAllowedTarget ? 500 : 400,
                  }}
                >
                  {step.label}
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
                {isAllowedTarget ? (
                  <Tooltip
                    title={disabled ? actions.flowDisabledReason : undefined}
                  >
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
                  </Tooltip>
                ) : (
                  node
                )}
              </div>
            );
          })}
        </div>
      </Col>
    ),
  };
}
