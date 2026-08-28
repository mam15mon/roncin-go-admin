import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProForm } from '@ant-design/pro-components';
import { Card, Row, Space, Spin, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { SectionCard } from '../page-shell/SectionCard';
import './OrderFormTemplate.less';
import type { OrderFormTemplateProps, OrderFormTemplateSection } from './types';

const { Text } = Typography;

/**
 * OrderFormTemplate 订单通用业务模板。
 *
 * 统一承载订单「新建/编辑」与「详情查看」的 UI 骨架：
 * - 页面头部（Header / PageHeaderShell）
 * - 加载占位
 * - 前置区块（如订单状态流程）
 * - 核心 5 大业务区块（业务信息、配舱信息、提单信息、3个备注、内部信息）
 * - 后置区块（如操作记录日志）
 * - 底部提交栏 / 操作栏
 *
 * 通过 readonly={true/false} 与 sections 数组驱动，保证新建与详情 100% 视觉排版统一！
 */
export function OrderFormTemplate<T>({
  loading = false,
  loadingTip,
  readonly = false,
  header,
  formRef,
  prependSections = [],
  sections,
  appendSections = [],
  initialValues,
  onFinish,
  submitText = '提交',
  resetText = '重置',
  footer,
}: OrderFormTemplateProps<T>) {
  const [submitting, setSubmitting] = useState(false);
  const innerFormRef = useRef<ProFormInstance | undefined>(undefined);
  const resolvedFormRef = formRef ?? innerFormRef;

  const handleFinish = async (values: T) => {
    if (!onFinish) return true;
    setSubmitting(true);
    try {
      return await onFinish(values);
    } finally {
      setSubmitting(false);
    }
  };

  const renderSection = (section: OrderFormTemplateSection) => (
    <SectionCard
      key={section.key}
      title={section.title}
      extra={section.extra}
    >
      <Row gutter={16}>{section.content}</Row>
    </SectionCard>
  );

  return (
    <PageContainer
      title={false}
      breadcrumbRender={false}
      header={{
        title: false,
        breadcrumb: undefined,
        style: header ? { padding: 0 } : undefined,
      }}
      style={{ marginTop: header ? 0 : -6 }}
    >
      {header}

      {loading ? (
        <Card
          variant="borderless"
          style={{ textAlign: 'center', padding: '60px 0', marginTop: 12 }}
        >
          <Space vertical size="middle">
            <Spin size="large" />
            {loadingTip && <Text type="secondary">{loadingTip}</Text>}
          </Space>
        </Card>
      ) : (
        <ProForm<T>
          className="roncin-order-form"
          formRef={resolvedFormRef}
          autoComplete="off"
          readonly={readonly}
          grid
          layout="horizontal"
          labelAlign="right"
          labelCol={{ flex: '96px' }}
          labelWrap={false}
          wrapperCol={{ flex: 'auto' }}
          initialValues={initialValues}
          onFinish={handleFinish}
          submitter={
            readonly
              ? false
              : {
                  searchConfig: {
                    submitText,
                    resetText,
                  },
                  submitButtonProps: {
                    loading: submitting,
                    size: 'large',
                    style: { minWidth: 120 },
                  },
                  resetButtonProps: {
                    size: 'large',
                    style: { minWidth: 100 },
                  },
                  render: (_, dom) => (
                    <div
                      style={{
                        textAlign: 'center',
                        marginTop: 24,
                        padding: '16px 0 32px',
                      }}
                    >
                      <Space size="middle">{dom}</Space>
                    </div>
                  ),
                }
          }
        >
          {/* 1. 前置自定义区块（如：订单状态流程） */}
          {prependSections.map(renderSection)}

          {/* 2. 核心 5 大业务区块 */}
          {sections.map(renderSection)}

          {/* 3. 后置自定义区块（如：操作记录日志） */}
          {appendSections.map(renderSection)}

          {/* 4. 额外底部插槽 */}
          {footer}
        </ProForm>
      )}
    </PageContainer>
  );
}

export default OrderFormTemplate;
