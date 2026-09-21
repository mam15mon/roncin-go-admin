import { SaveOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { App, Button, Card, Result, Space } from 'antd';
import React, { useCallback, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { getFormDraftScope } from '@/components/layout/formDraft';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import { OrderReferenceType } from '@/enums.generated';
import { searchShippingLineOptions } from '@/features/master-data/shipping-lines';
import { history } from '@/router/history';
import {
  orderServiceCheckOrderReference,
  orderServiceCreateOrder,
} from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';
import { PARTNER_ROLES, searchPartnersByRole } from './common';
import OrderPageHeader from './components/OrderPageHeader';
import { getOrderKindDefinition } from './order-kinds/registry';
import type { CreateOrderFormValues } from './order-kinds/sea-export/form-adapter';
import { useOrderCreateOptions } from './use-order-create-options';

export default function NewOrderPage() {
  const params = useParams<{ kind: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [submitting, setSubmitting] = useState(false);
  // 创建幂等键：每次提交意图一个键；失败重试沿用同键，成功后重新生成，
  // 配合后端同键同意图重放返回原单，避免超时重试造成重复订单。
  const createIdempotencyKeyRef = useRef(generateUUID());
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useInitialState();
  const draftScope = getFormDraftScope(
    initialState?.currentUser?.id,
    initialState?.currentUser?.currentOrganization?.id,
  );

  const definition = getOrderKindDefinition(params.kind);
  // create 权限同时门控候选项请求（无权限不发请求）与页面 403 兜底。
  const canCreate = access.canOrder(definition?.businessType ?? '', 'create');

  const {
    loading,
    error,
    retry,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
  } = useOrderCreateOptions(definition, canCreate);

  const checkOrderReference = useCallback(
    async (referenceType: OrderReferenceType) => {
      const isCustomerReference =
        referenceType === OrderReferenceType.ORDER_REFERENCE_TYPE_CUSTOMER;
      const fieldName = isCustomerReference
        ? 'customerReferenceNo'
        : 'internalReferenceNo';
      const fieldLabel = isCustomerReference ? '客户业务编号' : '企业内部编号';
      const referenceNo = String(
        formRef.current?.getFieldValue(fieldName) ?? '',
      ).trim();
      if (!referenceNo) {
        message.warning(`请先输入${fieldLabel}`);
        return;
      }

      const customerId = String(
        formRef.current?.getFieldValue('customerId') ?? '',
      );
      if (isCustomerReference && !customerId) {
        message.warning('请先选择委托单位');
        return;
      }

      try {
        const response = await orderServiceCheckOrderReference({
          referenceType,
          referenceNo,
          customerId: isCustomerReference ? customerId : undefined,
        });
        if (response.data?.duplicate) {
          message.warning(
            `${fieldLabel}已用于订单 ${response.data.orderNo || response.data.orderId}`,
          );
          return;
        }
        message.success(`${fieldLabel}未发现重复`);
      } catch (error: unknown) {
        const err = error as Error;
        message.error(err.message || `${fieldLabel}查重失败`);
      }
    },
    [message],
  );

  const templateProps = useMemo(
    () => ({
      serviceTypeOptions,
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
      searchShippingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      setCustomerCode: (code?: string) =>
        formRef.current?.setFieldValue('customerCode', code ?? ''),
      checkCustomerReferenceNo: () =>
        checkOrderReference(OrderReferenceType.ORDER_REFERENCE_TYPE_CUSTOMER),
      checkInternalReferenceNo: () =>
        checkOrderReference(OrderReferenceType.ORDER_REFERENCE_TYPE_INTERNAL),
      personnelOptions,
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
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      searchLocations,
      currencyOptions,
      containerSpecOptions,
      checkOrderReference,
      personnelOptions,
      initialState,
    ],
  );

  const sections = useMemo(
    () => definition?.form.buildSections(templateProps) ?? [],
    [definition, templateProps],
  );

  if (!definition) {
    return (
      <PageContainer>
        <Result
          status="404"
          title="无效的订单业务类型"
          subTitle={`未知的业务类型路径 "${params.kind || ''}"，请选择有效业务入口。`}
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
    setSubmitting(true);
    try {
      await orderServiceCreateOrder({
        ...definition.form.buildCreatePayload(values),
        idempotencyKey: createIdempotencyKeyRef.current,
      });
      createIdempotencyKeyRef.current = generateUUID();
      message.success('创建订单成功');
      history.push(`/orders/${definition.kind}`);
      return true;
    } catch (error: unknown) {
      const err = error as Error;
      message.error(err.message || '创建订单失败');
      return false;
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <OrderFormTemplate<CreateOrderFormValues>
      tabKey={resolveTabKey(`/orders/${definition.kind}/new`)}
      draftPathname={`/orders/${definition.kind}/new`}
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
      initialValues={definition.form.buildCreateDefaults({
        creator: templateProps.creator,
        serviceTypeOptions,
        cargoCategoryOptions,
      })}
      onFinish={handleFinish}
      submitter={false}
    />
  );
}
