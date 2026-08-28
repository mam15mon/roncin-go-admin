import { Col, Space, Tag } from 'antd';
import React from 'react';
import type { OrderFormTemplateSection } from '@/components/ui';

const STEPS = [
  { value: 2, key: 'booked', label: '已订舱' },
  { value: 3, key: 'allocated', label: '已配舱' },
  { value: 4, key: 'trucked', label: '拖车已安排' },
  { value: 5, key: 'si_cutoff', label: '已截单' },
  { value: 6, key: 'customs', label: '报关已安排' },
  { value: 7, key: 'released', label: '已放单' },
];

export function buildOrderStatusSection(
  order?: API.Order,
): OrderFormTemplateSection {
  const isUnreturned = order?.terminationStatus !== 3;
  const isUncompleted = order?.closureStatus !== 2;

  return {
    key: 'orderStatusSection',
    title: '订单状态',
    extra: (
      <Space size={12} align="center">
        <Tag color="blue">海运出口固定流程</Tag>
        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '2px 10px',
            borderRadius: 12,
            backgroundColor: isUnreturned ? '#f1f5f9' : '#fee2e2',
            color: isUnreturned ? '#475569' : '#ef4444',
            fontSize: 12,
            userSelect: 'none',
          }}
        >
          <span style={{ fontSize: 10 }}>{isUnreturned ? '⚪' : '🔴'}</span>
          <span>{isUnreturned ? '未退关' : '已退关'}</span>
        </div>
        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '2px 10px',
            borderRadius: 12,
            backgroundColor: isUncompleted ? '#f1f5f9' : '#dcfce7',
            color: isUncompleted ? '#475569' : '#16a34a',
            fontSize: 12,
            userSelect: 'none',
          }}
        >
          <span style={{ fontSize: 10 }}>{isUncompleted ? '⚪' : '🟢'}</span>
          <span>{isUncompleted ? '未完结' : '已完结'}</span>
        </div>
      </Space>
    ),
    content: (
      <Col span={24}>
        {/* 海管家极简横向流程节点 */}
        <div
          style={{
            padding: '20px 40px 12px',
            position: 'relative',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          {/* 背景贯穿灰色连接线 */}
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

          {STEPS.map((st) => {
            const isPassed = Number(order?.flowStatus ?? 0) >= st.value;
            return (
              <div
                key={st.key}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  position: 'relative',
                  zIndex: 2,
                }}
              >
                <div
                  style={{
                    width: 14,
                    height: 14,
                    borderRadius: '50%',
                    backgroundColor: isPassed ? '#1677ff' : '#94a3b8',
                    border: '3px solid #ffffff',
                    boxShadow: '0 0 0 1px #cbd5e1',
                    marginBottom: 8,
                  }}
                />
                <span
                  style={{
                    fontSize: 12,
                    color: isPassed ? '#0f172a' : '#64748b',
                    fontWeight: isPassed ? 500 : 400,
                  }}
                >
                  {st.label}
                </span>
              </div>
            );
          })}
        </div>
      </Col>
    ),
  };
}
