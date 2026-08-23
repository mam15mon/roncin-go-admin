import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProForm } from '@ant-design/pro-components';
import { Card, Row, Space, Spin, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import type { OrderFormTemplateProps } from './types';

const { Text } = Typography;

/**
 * OrderFormTemplate 订单整页表单模板。
 *
 * 统一承载订单新建/编辑页的 UI 骨架：页头、加载占位、按区块分组的
 * 卡片表单和底部提交栏。业务差异通过 sections 区块数组注入，修改本
 * 组件即可让全部业务类型（海运/空运、出口/进口）的订单表单同步生效。
 */
export function OrderFormTemplate<T>({
  title,
  subTitle,
  onBack,
  loading = false,
  loadingTip,
  formRef,
  sections,
  initialValues,
  onFinish,
  submitText = '提交',
  resetText = '重置',
}: OrderFormTemplateProps<T>) {
  const [submitting, setSubmitting] = useState(false);
  const innerFormRef = useRef<ProFormInstance | undefined>(undefined);
  const resolvedFormRef = formRef ?? innerFormRef;

  const handleFinish = async (values: T) => {
    setSubmitting(true);
    try {
      return await onFinish(values);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageContainer
      header={{
        title,
        subTitle,
        onBack,
      }}
    >
      {loading ? (
        <Card bordered={false} style={{ textAlign: 'center', padding: '60px 0' }}>
          <Space direction="vertical" size="middle">
            <Spin size="large" />
            {loadingTip && <Text type="secondary">{loadingTip}</Text>}
          </Space>
        </Card>
      ) : (
        <ProForm<T>
          formRef={resolvedFormRef}
          autoComplete="off"
          grid
          layout="horizontal"
          labelAlign="right"
          labelCol={{ flex: '96px' }}
          labelWrap={false}
          wrapperCol={{ flex: 'auto' }}
          initialValues={initialValues}
          onFinish={handleFinish}
          submitter={{
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
          }}
        >
          {sections.map((section) => (
            <Card
              key={section.key}
              size="small"
              title={
                <Space size={8}>
                  <span
                    style={{
                      width: 3,
                      height: 16,
                      borderRadius: 2,
                      background: '#1677ff',
                    }}
                  />
                  <Text strong>{section.title}</Text>
                </Space>
              }
              bordered={false}
              styles={{ body: { padding: 16 } }}
              style={{ width: '100%', marginBottom: 12 }}
            >
              <Row gutter={16}>{section.content}</Row>
            </Card>
          ))}
        </ProForm>
      )}
    </PageContainer>
  );
}
