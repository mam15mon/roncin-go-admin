import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { history, useAccess, useModel, useParams } from '@umijs/max';
import { App, Button, Result } from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useMemo, useRef } from 'react';
import { PageHeaderShell } from '@/components/ui';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import {
  orderServiceCheckOrderReference,
  orderServiceCreateOrder,
} from '@/services/roncin/orderService';
import {
  PARTNER_ROLES,
  parseOrderKind,
  searchPartnersByRole,
} from './common';
import {
  type CreateOrderFormValues,
  buildCreateOrderPayload,
} from './order-create-payload';
import { recommendedServiceIDs, SEA_SHIPMENT_MODE } from './sea-order-policy';
import { getAirTemplateSections, getSeaTemplateSections } from './templates';
import { useOrderCreateOptions } from './use-order-create-options';

export default function NewOrderPage() {
  const params = useParams<{ kind: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useModel('@@initialState');

  const config = parseOrderKind(params.kind);

  const {
    loading,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
  } = useOrderCreateOptions(config);

  const checkOrderReference = useCallback(
    async (referenceType: 1 | 2) => {
      const isCustomerReference = referenceType === 1;
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
      currencyOptions,
      containerSpecOptions,
      searchCustomers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CUSTOMER, keyword),
      searchCarriers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CARRIER, keyword),
      searchBookingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.BOOKING_AGENT, keyword),
      searchForeignAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.FOREIGN_AGENT, keyword),
      searchShippingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      setCustomerCode: (code?: string) =>
        formRef.current?.setFieldValue('customerCode', code ?? ''),
      checkCustomerReferenceNo: () => checkOrderReference(1),
      checkInternalReferenceNo: () => checkOrderReference(2),
      personnelOptions,
      creator:
        initialState?.currentUser?.id &&
        initialState.currentUser.currentOrganization?.id
          ? {
              userId: initialState.currentUser.id,
              displayName:
                initialState.currentUser.displayName ||
                initialState.currentUser.username ||
                initialState.currentUser.id,
              organizationId: initialState.currentUser.currentOrganization.id,
              organizationName:
                initialState.currentUser.currentOrganization.name ||
                initialState.currentUser.currentOrganization.id,
            }
          : undefined,
    }),
    [
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      currencyOptions,
      containerSpecOptions,
      checkOrderReference,
      personnelOptions,
      initialState,
    ],
  );

  const sections = useMemo(() => {
    if (!config) return [];
    return config.category === 'sea'
      ? getSeaTemplateSections(templateProps)
      : getAirTemplateSections(templateProps);
  }, [config, templateProps]);

  if (!config) {
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

  if (!access.canOrder(config.businessType, 'create')) {
    return <Result status="403" title="无权新建此类订单" />;
  }

  const handleFinish = async (values: CreateOrderFormValues) => {
    try {
      await orderServiceCreateOrder(buildCreateOrderPayload(values, config));
      message.success('创建订单成功');
      history.push(`/orders/${config.kind}`);
      return true;
    } catch (error: unknown) {
      const err = error as Error;
      message.error(err.message || '创建订单失败');
      return false;
    }
  };

  const defaultCargoCategoryId = cargoCategoryOptions.find(
    (item) => item.label === '普货',
  )?.value;

  return (
    <OrderFormTemplate<CreateOrderFormValues>
      loading={loading}
      loadingTip="正在加载业务模板与主数据..."
      formRef={formRef}
      header={
        <PageHeaderShell
          title={<span style={{ fontSize: 16, fontWeight: 600 }}>新建{config.title}</span>}
          subTitle="填写业务委托与配舱信息"
          breadcrumbs={[
            { label: '订单管理' },
            { label: config.title, onClick: () => history.push(`/orders/${config.kind}`) },
            { label: '新建订单' },
          ]}
          onBack={() => history.push(`/orders/${config.kind}`)}
        />
      }
      sections={sections}
      initialValues={{
        orderDate: dayjs(),
        ...(config.category === 'sea'
          ? {
              shipmentMode: 1,
              shipmentType: 1,
              serviceTypeIds: recommendedServiceIDs(
                serviceTypeOptions,
                SEA_SHIPMENT_MODE.TRADITIONAL_FORWARDING,
              ),
              cargoCategoryIds:
                typeof defaultCargoCategoryId === 'string'
                  ? [defaultCargoCategoryId]
                  : undefined,
              creatorUserId: initialState?.currentUser?.id,
              creatorOrganizationId:
                initialState?.currentUser?.currentOrganization?.id,
            }
          : {}),
      }}
      onFinish={handleFinish}
      submitText="创建订单"
      resetText="重置表单"
    />
  );
}
