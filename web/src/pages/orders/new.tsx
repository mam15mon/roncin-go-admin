import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProForm } from '@ant-design/pro-components';
import { history, useParams } from '@umijs/max';
import { App, Button, Card, Result, Row, Space, Spin } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { orderServiceCreateOrder } from '@/services/roncin/orderService';
import {
  PARTNER_ROLES,
  fetchOrderMasterData,
  loadStatusTemplatesByBusinessType,
  parseOrderKind,
  searchPartnersByRole,
} from './common';
import {
  type SelectOption,
  getAirTemplateSections,
  getSeaTemplateSections,
} from './templates';

type CreateOrderFormValues = {
  customerId: string;
  tradeDirection: number;
  tradeTerm: number;
  paymentTerm: number;
  statusTemplateId: string;
  carrierId?: string;
  bookingAgentId?: string;
  shipmentType?: number;
  containerOwnership?: number;
  shipmentMode?: number;
  serviceTypeIds?: string[];
  cargoCategoryIds?: string[];
  originLocationId?: string;
  destinationLocationId?: string;
  dischargeLocationId?: string;
  transitLocationId?: string;
  vesselVoyage?: string;
  etd?: string | dayjs.Dayjs;
  eta?: string | dayjs.Dayjs;
  siCutoff?: string | dayjs.Dayjs;
  docCutoff?: string | dayjs.Dayjs;
  customsCutoff?: string | dayjs.Dayjs;
  vgmCutoff?: string | dayjs.Dayjs;
  goodsDescription?: string;
  totalPackages?: number;
  totalPackageUnit?: string;
  specialRequirements?: string;
  orderDate?: string | dayjs.Dayjs;
  notes?: string;
};

export default function NewOrderPage() {
  const params = useParams<{ kind: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();

  const config = parseOrderKind(params.kind);

  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [statusTemplateOptions, setStatusTemplateOptions] = useState<SelectOption[]>([]);
  const [serviceTypeOptions, setServiceTypeOptions] = useState<SelectOption[]>([]);
  const [cargoCategoryOptions, setCargoCategoryOptions] = useState<SelectOption[]>([]);
  const [locationOptions, setLocationOptions] = useState<SelectOption[]>([]);

  useEffect(() => {
    if (!config) {
      setLoading(false);
      return;
    }

    setLoading(true);
    Promise.all([
      fetchOrderMasterData(),
      loadStatusTemplatesByBusinessType(config.businessType),
    ])
      .then(([masterData, templates]) => {
        setServiceTypeOptions(masterData.serviceTypeOptions);
        setCargoCategoryOptions(masterData.cargoCategoryOptions);
        setLocationOptions(
          config.category === 'sea'
            ? masterData.seaLocationOptions
            : masterData.airLocationOptions,
        );
        setStatusTemplateOptions(templates);
        if (templates.length > 0 && formRef.current) {
          formRef.current.setFieldValue('statusTemplateId', templates[0].value);
        }
      })
      .catch((error: Error) => {
        message.error(error.message || '加载主数据或状态模板失败');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [config, message]);

  const templateProps = useMemo(
    () => ({
      statusTemplateOptions,
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      searchCustomers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CUSTOMER, keyword),
      searchCarriers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CARRIER, keyword),
      searchBookingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.BOOKING_AGENT, keyword),
    }),
    [
      statusTemplateOptions,
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
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

  const handleFinish = async (values: CreateOrderFormValues) => {
    if (!values.statusTemplateId) {
      message.error('请选择状态流转模板');
      return false;
    }

    try {
      setSubmitting(true);
      const payload: API.CreateOrderRequest = {
        customerId: values.customerId,
        businessType: config.businessType,
        tradeDirection: Number(values.tradeDirection),
        tradeTerm: Number(values.tradeTerm),
        paymentTerm: Number(values.paymentTerm),
        statusTemplateId: values.statusTemplateId,
        carrierId: values.carrierId || undefined,
        bookingAgentId: values.bookingAgentId || undefined,
        shipmentType:
          values.shipmentType !== undefined && values.shipmentType !== null
            ? Number(values.shipmentType)
            : undefined,
        containerOwnership:
          values.containerOwnership !== undefined &&
          values.containerOwnership !== null
            ? Number(values.containerOwnership)
            : undefined,
        shipmentMode:
          values.shipmentMode !== undefined && values.shipmentMode !== null
            ? Number(values.shipmentMode)
            : undefined,
        serviceTypeIds: values.serviceTypeIds,
        cargoCategoryIds: values.cargoCategoryIds,
        originLocationId: values.originLocationId || undefined,
        destinationLocationId: values.destinationLocationId || undefined,
        dischargeLocationId: values.dischargeLocationId || undefined,
        transitLocationId: values.transitLocationId || undefined,
        vesselVoyage: values.vesselVoyage?.trim() || undefined,
        etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
        eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
        siCutoff: values.siCutoff ? dayjs(values.siCutoff).toISOString() : undefined,
        docCutoff: values.docCutoff ? dayjs(values.docCutoff).toISOString() : undefined,
        customsCutoff: values.customsCutoff
          ? dayjs(values.customsCutoff).toISOString()
          : undefined,
        vgmCutoff: values.vgmCutoff ? dayjs(values.vgmCutoff).toISOString() : undefined,
        goodsDescription: values.goodsDescription?.trim() || undefined,
        totalPackages:
          values.totalPackages !== undefined && values.totalPackages !== null
            ? Number(values.totalPackages)
            : undefined,
        totalPackageUnit: values.totalPackageUnit?.trim() || undefined,
        specialRequirements: values.specialRequirements?.trim() || undefined,
        orderDate: values.orderDate
          ? dayjs(values.orderDate).format('YYYY-MM-DD')
          : undefined,
        notes: values.notes?.trim() || undefined,
      };

      await orderServiceCreateOrder(payload);
      message.success('创建订单成功');
      history.push(`/orders/${config.kind}`);
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
    <PageContainer
      header={{
        title: `新建${config.title.replace('订单', '')}订单`,
        subTitle: `录入${config.title}信息，套用${config.category === 'sea' ? '海运' : '空运'}专业业务模板`,
        onBack: () => history.push(`/orders/${config.kind}`),
      }}
    >
      {loading ? (
        <Card bordered={false} style={{ textAlign: 'center', padding: '60px 0' }}>
          <Spin size="large" tip="正在加载业务模板与主数据..." />
        </Card>
      ) : (
        <ProForm<CreateOrderFormValues>
          formRef={formRef}
          grid
          initialValues={{
            tradeDirection: config.tradeDirection,
          }}
          onFinish={handleFinish}
          submitter={{
            searchConfig: {
              submitText: '创建订单',
              resetText: '重置表单',
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
              title={section.title}
              bordered={false}
              style={{ marginBottom: 16 }}
            >
              <Row gutter={16}>{section.content}</Row>
            </Card>
          ))}
        </ProForm>
      )}
    </PageContainer>
  );
}
