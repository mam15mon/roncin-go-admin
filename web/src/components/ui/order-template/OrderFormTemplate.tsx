import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProForm } from '@ant-design/pro-components';
import { Card, Row, Skeleton, Space, Spin, Typography } from 'antd';
import React, {
  useImperativeHandle,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import {
  clearFormDraft,
  getFormDraft,
  getFormDraftKey,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { useTabCloseGuard } from '@/components/layout/tabCloseGuard';
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
  tabKey,
  draftPathname,
  draftScope,
  actionsRef,
  enableCloseGuard = true,
  closeGuardMessage,
  onValuesChange,
  onReset,
}: OrderFormTemplateProps<T>) {
  const [submitting, setSubmitting] = useState(false);
  const innerFormRef = useRef<ProFormInstance | undefined>(undefined);
  const resolvedFormRef = formRef ?? innerFormRef;

  // 完整草稿身份（tabKey + draftPathname + draftScope）只能由调用方显式提供；
  // 任一缺失时不生成草稿键，也不读写持久草稿。
  const draftKey =
    tabKey && draftPathname && draftScope
      ? getFormDraftKey(tabKey, draftPathname, draftScope)
      : undefined;

  const [internalDirty, setInternalDirty] = useState(false);

  useTabCloseGuard({
    tabKey,
    isDirty: internalDirty,
    message: closeGuardMessage,
    enabled: !readonly && enableCloseGuard,
  });

  // 草稿恢复：draftKey 在模板挂载期内恒定，组织与单据身份变化分别由 OrganizationWorkspace 和调用方 key 负责卸载重挂载。
  // 一个模板身份只在首次同时满足「非 loading、非 readonly、身份完整」时恢复一次；
  // 初始 loading/readonly 时等待首次合法时机，后续锁状态往返不再覆盖内存中的表单值。
  const draftRestoreAttemptedRef = useRef(false);
  useLayoutEffect(() => {
    if (loading || readonly || !draftKey) return;
    if (draftRestoreAttemptedRef.current) return;
    draftRestoreAttemptedRef.current = true;
    const draft = getFormDraft<Partial<T>>(draftKey);
    if (draft && typeof draft === 'object' && Object.keys(draft).length > 0) {
      resolvedFormRef.current?.setFieldsValue(draft);
      setInternalDirty(true);
    }
  }, [draftKey, loading, readonly]);

  useImperativeHandle(
    actionsRef,
    () => ({
      resetTo: (values?: Partial<T>) => {
        if (draftKey) {
          clearFormDraft(draftKey);
        }
        // 先清 Form store 再回填最新快照，移除服务端新值中已不存在的旧字段。
        resolvedFormRef.current?.resetFields();
        if (values) {
          resolvedFormRef.current?.setFieldsValue(values);
        }
        setInternalDirty(false);
      },
    }),
    [draftKey, resolvedFormRef],
  );

  const handleFinish = async (values: T) => {
    if (!onFinish) return true;
    setSubmitting(true);
    try {
      const result = await onFinish(values);
      if (result !== false) {
        if (draftKey) {
          clearFormDraft(draftKey);
        }
        setInternalDirty(false);
      }
      return result;
    } finally {
      setSubmitting(false);
    }
  };

  const renderSection = (section: OrderFormTemplateSection) => (
    <SectionCard key={section.key} title={section.title} extra={section.extra}>
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
        <div className="roncin-order-form-skeleton" style={{ marginTop: 12 }}>
          {loadingTip && (
            <Card
              variant="borderless"
              style={{
                marginBottom: 12,
                borderRadius: 8,
                border: '1px solid #f0f0f0',
                backgroundColor: '#ffffff',
              }}
              styles={{ body: { padding: '10px 16px' } }}
            >
              <Space size="small" align="center">
                <Spin size="small" />
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {loadingTip}
                </Text>
              </Space>
            </Card>
          )}
          <SectionCard title="业务基本信息">
            <Skeleton active paragraph={{ rows: 3 }} />
          </SectionCard>
          <SectionCard title="运输与订舱信息">
            <Skeleton active paragraph={{ rows: 4 }} />
          </SectionCard>
        </div>
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
          onValuesChange={(changedValues, allValues) => {
            if (!internalDirty) {
              setInternalDirty(true);
            }
            if (!readonly && draftKey) {
              saveFormDraft(draftKey, allValues);
            }
            onValuesChange?.(changedValues, allValues);
          }}
          onReset={() => {
            if (draftKey) {
              clearFormDraft(draftKey);
            }
            setInternalDirty(false);
            onReset?.();
          }}
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
