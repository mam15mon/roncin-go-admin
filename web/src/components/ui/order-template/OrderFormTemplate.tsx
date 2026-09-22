import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProForm } from '@ant-design/pro-components';
import {
  App,
  Card,
  Form,
  type FormProps,
  Row,
  Skeleton,
  Space,
  Spin,
  Typography,
} from 'antd';
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
import {
  collectFormSectionErrors,
  FormAnchorNav,
  locateSectionError,
  scrollToFirstFormError,
} from '../form-navigator';
import { SectionCard } from '../page-shell/SectionCard';
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
  submitter,
  footer,
  tabKey,
  draftPathname,
  draftScope,
  actionsRef,
  enableCloseGuard = true,
  closeGuardMessage,
  onValuesChange,
  onReset,
  onRevealError,
  showAnchorNav = true,
}: OrderFormTemplateProps<T>) {
  const { message } = App.useApp();
  const [submitting, setSubmitting] = useState(false);
  const [sectionErrors, setSectionErrors] = useState<Record<string, number>>(
    {},
  );
  // 表单导航展开时内容区预留 164px 右侧空间，避免遮挡输入控件；
  // 窄屏（<1500px）默认折叠，保证输入区完整可用。
  const [navCollapsed, setNavCollapsed] = useState(
    () => typeof window !== 'undefined' && window.innerWidth < 1500,
  );
  const innerFormRef = useRef<ProFormInstance | undefined>(undefined);
  const resolvedFormRef = formRef ?? innerFormRef;

  const [form] = Form.useForm();
  const allSections = [...prependSections, ...sections, ...appendSections];
  const getVisibility = (values: Record<string, unknown>) =>
    allSections
      .map((section) =>
        !section.visible || section.visible(values) ? '1' : '0',
      )
      .join('');
  // 只订阅分节可见性，普通字段键入不触发模板整体重渲染。
  const watchedVisibility = Form.useWatch(getVisibility, {
    form,
    preserve: true,
  });
  const visibility = getVisibility(
    watchedVisibility === undefined
      ? (initialValues ?? {})
      : form.getFieldsValue(true),
  );
  const visibleSections = allSections.filter(
    (_, index) => visibility[index] === '1',
  );

  // 楼层锚点分节列表
  const anchorItems = visibleSections.map((section) => ({
    key: section.key,
    title: section.title,
  }));

  // 导航是否实际渲染：只读、加载中或分节不足时不渲染，也不预留右侧空间
  const anchorNavVisible =
    showAnchorNav && !loading && !readonly && anchorItems.length > 1;

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
        setSectionErrors({});
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
        setSectionErrors({});
      }
      return result;
    } finally {
      setSubmitting(false);
    }
  };

  // 校验失败处理：先让页面把隐藏区域（页签/折叠备注）中的首个错误字段
  // 变为可见，再自动平滑滚动居中并高亮首个错误项，同时统计各分节错误供导航器使用
  const handleFinishFailed = async (
    errorInfo: Parameters<NonNullable<FormProps<T>['onFinishFailed']>>[0],
  ) => {
    if (onRevealError) {
      try {
        await onRevealError(errorInfo);
      } catch {
        // 定位辅助失败不阻断默认错误处理
      }
      // 等待切换页签/展开折叠区引发的 React 渲染落地后再查 DOM。
      await new Promise<void>((resolve) =>
        requestAnimationFrame(() => resolve()),
      );
    }
    const res = scrollToFirstFormError({
      errorFields: errorInfo?.errorFields,
      notify: (msg) => message.warning(msg),
    });
    setSectionErrors(res.errorsBySection);

    // 延迟 100ms 兜底重算一次分节错误分布，防止依赖微任务时序导致 DOM 的 has-error 漏计
    window.setTimeout(() => {
      const delayedErrors = collectFormSectionErrors();
      if (Object.keys(delayedErrors).length > 0) {
        setSectionErrors(delayedErrors);
      }
    }, 100);
  };

  // 点击楼层中带错误的分节，精确定位至该分节内的错误字段
  const handleErrorClick = (sectionKey: string) => {
    void locateSectionError(sectionKey, 84);
  };

  const renderSection = (section: OrderFormTemplateSection) => (
    <SectionCard
      key={section.key}
      sectionKey={section.key}
      id={`section-${section.key}`}
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
          form={form}
          formRef={resolvedFormRef}
          autoComplete="off"
          readonly={readonly}
          grid
          layout="vertical"
          style={{
            paddingRight: navCollapsed || !anchorNavVisible ? 0 : 164,
            transition: 'padding-right 0.25s ease',
          }}
          initialValues={initialValues}
          onValuesChange={(changedValues, allValues) => {
            if (!internalDirty) {
              setInternalDirty(true);
            }
            if (!readonly && draftKey) {
              saveFormDraft(draftKey, allValues);
            }
            if (Object.keys(sectionErrors).length > 0) {
              window.setTimeout(() => {
                setSectionErrors(collectFormSectionErrors());
              }, 200);
            }
            onValuesChange?.(changedValues, allValues);
          }}
          onReset={() => {
            if (draftKey) {
              clearFormDraft(draftKey);
            }
            setInternalDirty(false);
            setSectionErrors({});
            onReset?.();
          }}
          onFinish={handleFinish}
          onFinishFailed={handleFinishFailed}
          submitter={
            readonly || submitter === false
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
          {visibleSections.map(renderSection)}

          {/* 4. 额外底部插槽 */}
          {footer}
        </ProForm>
      )}

      {/* 5. 悬浮楼层导航与分节错误指示器 */}
      {anchorNavVisible && (
        <FormAnchorNav
          items={anchorItems}
          sectionErrors={sectionErrors}
          defaultCollapsed={navCollapsed}
          onCollapsedChange={setNavCollapsed}
          onErrorClick={handleErrorClick}
        />
      )}
    </PageContainer>
  );
}

export default OrderFormTemplate;
