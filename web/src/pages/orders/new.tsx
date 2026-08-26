import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { history, useAccess, useModel, useParams } from '@umijs/max';
import { App, Button, Result } from 'antd';
import dayjs from 'dayjs';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import {
  orderServiceCheckOrderReference,
  orderServiceCreateOrder,
  orderServiceListPersonnelOptions,
} from '@/services/roncin/orderService';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  loadStatusTemplatesByBusinessType,
  MASTER_DATA_KINDS,
  PARTNER_ROLES,
  parseOrderKind,
  searchPartnersByRole,
  seaServiceTypeNames,
} from './common';
import { recommendedServiceIDs, SEA_SHIPMENT_MODE } from './sea-order-policy';
import {
  getAirTemplateSections,
  getSeaTemplateSections,
  type SelectOption,
} from './templates';

type CreateOrderFormValues = {
  customerId: string;
  customerReferenceNo?: string;
  internalReferenceNo?: string;
  customerCode?: string;
  tradeTerm: number;
  paymentTerm: number;
  carrierId?: string;
  bookingAgentId?: string;
  foreignAgentId?: string;
  shippingAgentId?: string;
  contractNo?: string;
  cargoValue?: string;
  cargoCurrency?: string;
  insurancePremium?: string;
  insuranceCurrency?: string;
  unNumber?: string;
  hazardClass?: string;
  factoryName?: string;
  cargoReadyAt?: string | dayjs.Dayjs;
  loadingTerms?: string;
  declarationCutoffAt?: string | dayjs.Dayjs;
  receivedAt?: string | dayjs.Dayjs;
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
  totalGrossWeightKg?: number;
  totalVolumeCbm?: number;
  totalPackageUnit?: string;
  specialRequirements?: string;
  orderDate?: string | dayjs.Dayjs;
  notes?: string;
  bookingNotes?: string;
  allocationNotes?: string;
  operationNotes?: string;
  shippingDocuments?: API.OrderShippingDocumentInput[];
  containerRequests?: API.OrderContainerRequestInput[];
  operatorUserId?: string;
  operatorOrganizationId?: string;
  salesUserId?: string;
  salesOrganizationId?: string;
  customerServiceUserId?: string;
  customerServiceOrganizationId?: string;
  associateUserId?: string;
  associateOrganizationId?: string;
  documentUserId?: string;
  documentOrganizationId?: string;
  commercialUserId?: string;
  commercialOrganizationId?: string;
  associate2UserId?: string;
  associate2OrganizationId?: string;
  creatorUserId?: string;
  creatorOrganizationId?: string;
};

export default function NewOrderPage() {
  const params = useParams<{ kind: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useModel('@@initialState');

  const config = parseOrderKind(params.kind);

  const [loading, setLoading] = useState(true);
  const [statusTemplateOptions, setStatusTemplateOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [serviceTypeOptions, setServiceTypeOptions] = useState<SelectOption[]>(
    [],
  );
  const [cargoCategoryOptions, setCargoCategoryOptions] = useState<
    SelectOption[]
  >([]);
  const [locationOptions, setLocationOptions] = useState<SelectOption[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);
  const [containerSpecOptions, setContainerSpecOptions] = useState<
    SelectOption[]
  >([]);
  const [personnelOptions, setPersonnelOptions] = useState<
    API.OrderPersonnelOption[]
  >([]);

  useEffect(() => {
    if (!config) {
      setLoading(false);
      return;
    }

    setLoading(true);
    Promise.all([
      fetchOrderMasterData(),
      loadStatusTemplatesByBusinessType(config.businessType),
      config.category === 'sea'
        ? orderServiceListPersonnelOptions({
            businessType: config.businessType,
          })
        : Promise.resolve({ data: [] }),
    ])
      .then(([masterData, templates, personnelResponse]) => {
        const nextServiceTypeOptions =
          config.category === 'sea'
            ? seaServiceTypeNames.map((name) => {
                const option = masterData.serviceTypeOptions.find(
                  (item) => item.label === name,
                );
                if (!option) {
                  throw new Error(`缺少海运服务类型主数据：${name}`);
                }
                return option;
              })
            : masterData.serviceTypeOptions;

        setServiceTypeOptions(nextServiceTypeOptions);
        setCargoCategoryOptions(masterData.cargoCategoryOptions);
        setLocationOptions(
          config.category === 'sea'
            ? masterData.seaLocationOptions
            : masterData.airLocationOptions,
        );
        setCurrencyOptions(masterData.currencyOptions);
        setContainerSpecOptions(
          masterData.masterOptions
            .filter(
              (item) =>
                isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
                item.enabled !== false,
            )
            .map((item) => ({
              label: item.code
                ? `${item.name ?? item.code} (${item.code})`
                : (item.name ?? ''),
              value: item.id ?? '',
            }))
            .filter((item) => item.value !== ''),
        );
        setStatusTemplateOptions(templates);
        setPersonnelOptions(personnelResponse.data ?? []);
      })
      .catch((error: Error) => {
        message.error(error.message || '加载主数据或状态模板失败');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [config, message]);

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
    const defaultStatusTemplateId = statusTemplateOptions[0]?.value;
    if (typeof defaultStatusTemplateId !== 'string') {
      message.error('当前业务类型未配置默认状态流转模板');
      return false;
    }

    try {
      const personnelAssignments: API.OrderPersonnelAssignmentInput[] = [];
      const addPersonnel = (
        role: number,
        userId?: string,
        organizationId?: string,
      ) => {
        if (userId && organizationId) {
          personnelAssignments.push({ role, userId, organizationId });
        }
      };
      addPersonnel(2, values.operatorUserId, values.operatorOrganizationId);
      addPersonnel(3, values.salesUserId, values.salesOrganizationId);
      addPersonnel(
        4,
        values.customerServiceUserId,
        values.customerServiceOrganizationId,
      );
      addPersonnel(7, values.associateUserId, values.associateOrganizationId);
      addPersonnel(5, values.documentUserId, values.documentOrganizationId);
      addPersonnel(6, values.commercialUserId, values.commercialOrganizationId);
      addPersonnel(8, values.associate2UserId, values.associate2OrganizationId);

      const payload: API.CreateOrderRequest = {
        customerId: values.customerId,
        customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
        internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
        businessType: config.businessType,
        tradeDirection: config.tradeDirection,
        tradeTerm: Number(values.tradeTerm),
        paymentTerm: Number(values.paymentTerm),
        statusTemplateId: defaultStatusTemplateId,
        carrierId: values.carrierId || undefined,
        bookingAgentId: values.bookingAgentId || undefined,
        foreignAgentId: values.foreignAgentId || undefined,
        shippingAgentId: values.shippingAgentId || undefined,
        contractNo: values.contractNo?.trim() || undefined,
        cargoValue: values.cargoValue?.trim() || undefined,
        cargoCurrency: values.cargoCurrency || undefined,
        insurancePremium: values.insurancePremium?.trim() || undefined,
        insuranceCurrency: values.insuranceCurrency || undefined,
        unNumber: values.unNumber?.trim() || undefined,
        hazardClass: values.hazardClass?.trim() || undefined,
        factoryName: values.factoryName?.trim() || undefined,
        cargoReadyAt: values.cargoReadyAt
          ? dayjs(values.cargoReadyAt).toISOString()
          : undefined,
        loadingTerms: values.loadingTerms?.trim() || undefined,
        declarationCutoffAt: values.declarationCutoffAt
          ? dayjs(values.declarationCutoffAt).toISOString()
          : undefined,
        receivedAt: values.receivedAt
          ? dayjs(values.receivedAt).toISOString()
          : undefined,
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
        siCutoff: values.siCutoff
          ? dayjs(values.siCutoff).toISOString()
          : undefined,
        docCutoff: values.docCutoff
          ? dayjs(values.docCutoff).toISOString()
          : undefined,
        customsCutoff: values.customsCutoff
          ? dayjs(values.customsCutoff).toISOString()
          : undefined,
        vgmCutoff: values.vgmCutoff
          ? dayjs(values.vgmCutoff).toISOString()
          : undefined,
        goodsDescription: values.goodsDescription?.trim() || undefined,
        totalPackages:
          values.totalPackages !== undefined && values.totalPackages !== null
            ? Number(values.totalPackages)
            : undefined,
        totalGrossWeightKg:
          values.totalGrossWeightKg !== undefined &&
          values.totalGrossWeightKg !== null
            ? Number(values.totalGrossWeightKg)
            : undefined,
        totalVolumeCbm:
          values.totalVolumeCbm !== undefined && values.totalVolumeCbm !== null
            ? Number(values.totalVolumeCbm)
            : undefined,
        totalPackageUnit: values.totalPackageUnit?.trim() || undefined,
        specialRequirements: values.specialRequirements?.trim() || undefined,
        orderDate: values.orderDate
          ? dayjs(values.orderDate).toISOString()
          : undefined,
        notes: values.notes?.trim() || undefined,
        bookingNotes: values.bookingNotes?.trim() || undefined,
        allocationNotes: values.allocationNotes?.trim() || undefined,
        operationNotes: values.operationNotes?.trim() || undefined,
        personnelAssignments,
        shippingDocuments: values.shippingDocuments
          ?.map((doc) => ({
            ...doc,
            masterNo: doc.masterNo?.trim() || '',
            houseNo: doc.houseNo?.trim() || '',
          }))
          .filter((doc) => doc.masterNo || doc.houseNo),
        containerRequests: values.containerRequests,
      };

      await orderServiceCreateOrder(payload);
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
