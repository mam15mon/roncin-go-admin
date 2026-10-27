import { SaveOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { App, Button, Card, Result, Space } from 'antd';
import React, { useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { getFormDraftScope } from '@/components/layout/formDraft';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import { SeaDocumentStructure } from '@/enums.generated';
import { searchShippingLineOptions } from '@/features/master-data/shipping-lines';
import { history } from '@/router/history';
import {
  orderServiceCreateOrder,
  orderServiceGetOrder,
} from '@/services/roncin/orderService';
import { isWriteResponse } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { generateUUID } from '@/utils/uuid';
import {
  PARTNER_ROLES,
  searchPartnersByRole,
  searchShippingAgentOptions,
} from './common';
import OrderPageHeader from './components/OrderPageHeader';
import type { CreateOrderFormValues } from './order-kinds/form-values';
import { getOrderKindDefinition } from './order-kinds/registry';
import {
  getSeaHouseOrderCreateSource,
  resolveSeaHouseOrderCreateRequest,
} from './sea-house-order-context';
import { revealSeaFormErrors } from './templates';
import type { SeaHouseOrderCreateSource } from './templates/types';
import { useOrderCreateOptions } from './use-order-create-options';

export default function NewOrderPage() {
  const params = useParams<{ kind: string }>();
  const { search } = useLocation();
  const kind = params.kind ?? '';
  const sourceRequest = resolveSeaHouseOrderCreateRequest(kind, search);
  return (
    <NewOrderContent
      key={`${kind}:${sourceRequest.error ? 'invalid' : 'create'}:${sourceRequest.draftPathname}`}
      kind={kind}
      sourceRequest={sourceRequest}
    />
  );
}

function NewOrderContent({
  kind,
  sourceRequest,
}: {
  kind: string;
  sourceRequest: ReturnType<typeof resolveSeaHouseOrderCreateRequest>;
}) {
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [submitting, setSubmitting] = useState(false);
  // 创建幂等键：每次提交意图一个键；失败重试沿用同键，成功后重新生成，
  // 配合后端同键同意图重放返回原单，避免超时重试造成重复订单。
  const createIdempotencyKeyRef = useRef(generateUUID());
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const draftScope = getFormDraftScope(
    initialState?.currentUser?.id,
    initialState?.currentUser?.currentOrganization?.id,
  );

  // 实例卸载即失效，覆盖 A→B→A 以及 StrictMode 的 setup/cleanup 往返。
  const instanceGenerationRef = useRef(0);
  useLayoutEffect(() => {
    instanceGenerationRef.current += 1;
    return () => {
      instanceGenerationRef.current += 1;
    };
  }, []);

  const definition = getOrderKindDefinition(kind);
  // create 权限同时门控候选项请求（无权限不发请求）与页面 403 兜底。
  const canCreate = access.canOrder(definition?.businessType ?? '', 'create');
  const canReadSource = access.canOrder(definition?.businessType ?? '', 'read');
  const { sourceOrderId } = sourceRequest;
  const canUseSource = Boolean(
    sourceOrderId &&
      canCreate &&
      canReadSource &&
      organizationId &&
      access.canOperateOrganization(organizationId),
  );
  const sourceQuery = useQuery({
    queryKey: [
      'orders',
      'sea-house-create-source',
      { organizationId, sourceOrderId },
    ],
    enabled: Boolean(sourceOrderId && !sourceRequest.error && canUseSource),
    refetchOnMount: 'always',
    queryFn: async () => {
      if (!sourceOrderId || !organizationId) {
        throw new Error('缺少来源订单或当前公司');
      }
      const response = await orderServiceGetOrder(
        { id: sourceOrderId },
        { skipErrorHandler: true },
      );
      if (response?.success === false)
        throw new Error('来源订单读取失败，请重试。');
      return getSeaHouseOrderCreateSource(
        response?.data,
        sourceOrderId,
        organizationId,
      );
    },
    // 错误由页面 Result 展示，避免与查询全局通知重复。
    meta: { silent: true },
  });
  // 每个创建实例先完成本次来源读取，再固定初始化上下文；缓存旧值和后续
  // 背景刷新均不能替换当前表单正在使用的来源身份或覆盖已恢复草稿。
  const [source, setSource] = useState<SeaHouseOrderCreateSource>();
  useLayoutEffect(() => {
    if (sourceQuery.isFetchedAfterMount && sourceQuery.isSuccess) {
      setSource((initialized) => initialized ?? sourceQuery.data);
    }
  }, [
    sourceQuery.data,
    sourceQuery.isFetchedAfterMount,
    sourceQuery.isSuccess,
  ]);

  const {
    loading,
    error,
    retry,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
  } = useOrderCreateOptions(
    definition,
    canCreate && !sourceRequest.error && (!sourceOrderId || canUseSource),
  );

  const templateProps = useMemo(
    () => ({
      cargoCategoryOptions,
      locationOptions,
      searchLocations,
      currencyOptions,
      containerSpecOptions,
      searchCustomers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CUSTOMER, keyword),
      searchShippingLines: searchShippingLineOptions,
      searchBookingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      searchForeignAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.FOREIGN_AGENT, keyword),
      searchShippingAgents: searchShippingAgentOptions,
      setCustomerCode: (code?: string) =>
        formRef.current?.setFieldValue('customerCode', code ?? ''),
      personnelOptions,
      organizationId,
      seaHouseOrderCreateSource: source,
      creator: initialState?.currentUser?.id
        ? {
            userId: initialState.currentUser.id,
            displayName:
              initialState.currentUser.displayName ||
              initialState.currentUser.username ||
              initialState.currentUser.id,
          }
        : undefined,
    }),
    [
      cargoCategoryOptions,
      locationOptions,
      searchLocations,
      currencyOptions,
      containerSpecOptions,
      personnelOptions,
      initialState,
      organizationId,
      source,
    ],
  );

  const sections = useMemo(
    () => definition?.form.buildSections(templateProps) ?? [],
    [definition, templateProps],
  );
  const initialValues = useMemo(
    () => ({
      ...definition?.form.buildCreateDefaults({
        creator: templateProps.creator,
        cargoCategoryOptions,
      }),
      ...(source
        ? {
            shippingLineId: source.shippingLineId,
            seaMasterBillMasterNo: source.masterNo,
            seaDocumentStructure:
              SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
          }
        : {}),
    }),
    [
      definition,
      cargoCategoryOptions,
      templateProps.creator?.userId,
      templateProps.creator?.displayName,
      source,
    ],
  );

  if (!definition) {
    return (
      <PageContainer>
        <Result
          status="404"
          title="无效的订单业务类型"
          subTitle={`未知的业务类型路径 "${kind}"，请选择有效业务入口。`}
          extra={
            <Button
              type="primary"
              onClick={() => history.push('/orders/sea-export')}
            >
              返回海运出口订单
            </Button>
          }
        />
      </PageContainer>
    );
  }

  if (!canCreate) {
    return <Result status="403" title="无权新建此类订单" />;
  }

  if (sourceRequest.error || (sourceOrderId && !canUseSource)) {
    return (
      <PageContainer>
        <Result
          status={sourceRequest.error ? 'warning' : '403'}
          title={
            sourceRequest.error
              ? '无法从此订单新增分单'
              : '无权读取来源订单或新建分单'
          }
          subTitle={sourceRequest.error}
        />
      </PageContainer>
    );
  }

  if (sourceOrderId && !source) {
    return (
      <PageContainer>
        <Result
          status={sourceQuery.isError ? 'warning' : 'info'}
          title={
            sourceQuery.isError ? '来源订单加载失败' : '正在读取来源主单...'
          }
          subTitle={
            sourceQuery.isError
              ? getErrorMessage(sourceQuery.error, '无法读取来源订单，请重试。')
              : undefined
          }
          extra={
            sourceQuery.isError ? (
              <Button type="primary" onClick={() => void sourceQuery.refetch()}>
                重新加载来源订单
              </Button>
            ) : undefined
          }
        />
      </PageContainer>
    );
  }

  if (error && !loading) {
    return (
      <PageContainer
        title={false}
        breadcrumbRender={false}
        header={{
          title: false,
          breadcrumb: undefined,
          style: { padding: 0 },
        }}
        style={{ marginTop: 0 }}
      >
        <OrderPageHeader
          page="create"
          orderKind={definition.kind}
          navigationTitle={definition.navigationTitle}
          subTitle="填写业务委托与配舱信息"
        />
        <Card
          variant="borderless"
          style={{
            marginTop: 12,
            borderRadius: 8,
            border: '1px solid #f0f0f0',
            backgroundColor: '#ffffff',
          }}
        >
          <Result
            status="warning"
            title="主数据加载失败"
            subTitle={
              error.message ||
              '无法获取创建订单所需的主数据，请检查网络或重试。'
            }
            extra={
              <Button type="primary" onClick={retry}>
                重新加载
              </Button>
            }
          />
        </Card>
      </PageContainer>
    );
  }

  const handleFinish = async (values: CreateOrderFormValues) => {
    const generation = instanceGenerationRef.current;
    const isCurrentInstance = () =>
      generation === instanceGenerationRef.current;
    setSubmitting(true);
    try {
      const response = await orderServiceCreateOrder(
        {
          ...definition.form.buildCreatePayload(values),
          idempotencyKey: createIdempotencyKeyRef.current,
        },
        // 当前实例统一反馈写入错误，失效实例的迟到失败不触发全局通知。
        { skipErrorHandler: true },
      );
      if (!isCurrentInstance() || !isWriteResponse(response, true))
        return false;
      // 创建结果必须以服务端返回的订单 ID 为准；空响应或缺失 ID 时
      // 不提示成功、不跳转、不轮换幂等键，保留表单供同键重试确认，
      // 避免响应丢失时重复建单。
      const createdId = response?.data?.id;
      if (!createdId) {
        message.error(
          '服务端未返回订单编号，创建结果待确认；表单已保留，请再次点击创建订单确认（不会重复建单）',
        );
        return false;
      }
      createIdempotencyKeyRef.current = generateUUID();
      message.success('创建订单成功');
      history.push(`/orders/${definition.kind}/${createdId}`);
      return true;
    } catch (error: unknown) {
      if (!isCurrentInstance()) return false;
      message.error(getErrorMessage(error, '创建订单失败'));
      return false;
    } finally {
      if (isCurrentInstance()) setSubmitting(false);
    }
  };

  return (
    <OrderFormTemplate<CreateOrderFormValues>
      tabKey={resolveTabKey(`/orders/${definition.kind}/new`)}
      draftPathname={sourceRequest.draftPathname}
      draftScope={draftScope}
      loading={loading}
      loadingTip="正在加载业务模板与主数据..."
      formRef={formRef}
      header={
        <OrderPageHeader
          page="create"
          orderKind={definition.kind}
          navigationTitle={definition.navigationTitle}
          subTitle="填写业务委托与配舱信息"
          actions={
            loading ? undefined : (
              <Space size={8}>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  loading={submitting}
                  onClick={() => formRef.current?.submit()}
                >
                  创建订单
                </Button>
                <Button
                  disabled={submitting}
                  onClick={() => history.push(`/orders/${definition.kind}`)}
                >
                  取消
                </Button>
              </Space>
            )
          }
        />
      }
      sections={sections}
      initialValues={initialValues}
      onFinish={handleFinish}
      onRevealError={({ errorFields }) => revealSeaFormErrors(errorFields)}
      submitter={false}
    />
  );
}
