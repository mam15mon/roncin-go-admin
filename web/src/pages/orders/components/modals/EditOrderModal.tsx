import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { App } from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { OrderShippingDocumentFields } from '../../order-plan-fields';
import { SeaContainerPlanFields } from '../../templates/sea-template';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';

export type EditOrderModalRef = {
  open: (order: API.Order) => void;
};

type EditOrderModalProps = {
  category: string;
  tradeDirectionOptions: { label: string; value: number }[];
  tradeTermOptions: { label: string; value: number }[];
  paymentTermOptions: { label: string; value: number }[];
  shipmentTypeOptions: { label: string; value: number }[];
  locationOptions: { label: string; value: string }[];
  searchLocations: (
    keyWords?: string,
  ) => Promise<{ label: string; value: string }[]>;
  serviceTypeOptions: { label: string; value: string }[];
  cargoCategoryOptions: { label: string; value: string }[];
  containerSpecOptions: { label: string; value: string }[];
  searchCustomers: (
    keyWords: string,
  ) => Promise<{ label: string; value: string }[]>;
  onSuccess: () => void;
};

type EditOrderFormValues = {
  customerId?: string;
  tradeDirection?: number;
  tradeTerm?: number;
  paymentTerm?: number;
  shipmentType?: number;
  serviceTypeIds?: string[];
  cargoCategoryIds?: string[];
  originLocationId?: string;
  destinationLocationId?: string;
  vesselVoyage?: string;
  etd?: string;
  eta?: string;
  goodsDescription?: string;
  totalPackages?: number;
  totalGrossWeightKg?: number;
  totalVolumeCbm?: number;
  totalPackageUnit?: string;
  notes?: string;
  shippingDocuments?: API.OrderShippingDocumentInput[];
  containerRequests?: API.OrderContainerRequestInput[];
};

const EditOrderModal = forwardRef<EditOrderModalRef, EditOrderModalProps>(
  function EditOrderModal(
    {
      category,
      tradeDirectionOptions,
      tradeTermOptions,
      paymentTermOptions,
      shipmentTypeOptions,
      locationOptions,
      searchLocations,
      serviceTypeOptions,
      cargoCategoryOptions,
      containerSpecOptions,
      searchCustomers,
      onSuccess,
    },
    ref,
  ) {
    const { message } = App.useApp();
    const formRef = useRef<ProFormInstance | undefined>(undefined);
    const [open, setOpen] = useState(false);
    const [record, setRecord] = useState<API.Order>();

    useImperativeHandle(ref, () => ({
      open: (order) => {
        setRecord(order);
        formRef.current?.setFieldsValue({
          customerId: order.customerId,
          tradeDirection: order.tradeDirection,
          tradeTerm: order.tradeTerm,
          paymentTerm: order.paymentTerm,
          shipmentType: order.shipmentType,
          serviceTypeIds: order.serviceTypeIds,
          cargoCategoryIds: order.cargoCategoryIds,
          originLocationId: order.originLocationId,
          destinationLocationId: order.destinationLocationId,
          vesselVoyage: order.vesselVoyage,
          etd: order.etd ? dayjs(order.etd) : undefined,
          eta: order.eta ? dayjs(order.eta) : undefined,
          goodsDescription: order.goodsDescription,
          totalPackages: order.totalPackages,
          totalGrossWeightKg: order.totalGrossWeightKg,
          totalVolumeCbm: order.totalVolumeCbm,
          totalPackageUnit: order.totalPackageUnit,
          notes: order.notes,
          shippingDocuments: order.shippingDocuments,
          containerRequests: order.containerRequests,
        });
        setOpen(true);
      },
    }));

    return (
      <ModalForm<EditOrderFormValues>
        title="编辑订单草稿"
        open={open}
        formRef={formRef}
        grid
        initialValues={
          record
            ? {
                customerId: record.customerId,
                tradeDirection: record.tradeDirection,
                tradeTerm: record.tradeTerm,
                paymentTerm: record.paymentTerm,
                shipmentType: record.shipmentType,
                serviceTypeIds: record.serviceTypeIds,
                cargoCategoryIds: record.cargoCategoryIds,
                originLocationId: record.originLocationId,
                destinationLocationId: record.destinationLocationId,
                vesselVoyage: record.vesselVoyage,
                etd: record.etd ? dayjs(record.etd) : undefined,
                eta: record.eta ? dayjs(record.eta) : undefined,
                goodsDescription: record.goodsDescription,
                totalPackages: record.totalPackages,
                totalGrossWeightKg: record.totalGrossWeightKg,
                totalVolumeCbm: record.totalVolumeCbm,
                totalPackageUnit: record.totalPackageUnit,
                notes: record.notes,
                shippingDocuments: record.shippingDocuments,
                containerRequests: record.containerRequests,
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 820,
          onCancel: () => setOpen(false),
        }}
        onOpenChange={setOpen}
        onFinish={async (values) => {
          if (!record?.id || !record?.version) return false;
          await orderServiceUpdateOrder(
            { id: record.id },
            {
              id: record.id,
              expectedVersion: record.version,
              customerId: values.customerId,
              tradeDirection: values.tradeDirection,
              tradeTerm: values.tradeTerm,
              paymentTerm: values.paymentTerm,
              shipmentType: values.shipmentType,
              serviceTypeIds: values.serviceTypeIds,
              cargoCategoryIds: values.cargoCategoryIds,
              originLocationId: values.originLocationId,
              destinationLocationId: values.destinationLocationId,
              vesselVoyage: values.vesselVoyage,
              etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
              eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
              goodsDescription: values.goodsDescription,
              totalPackages: values.totalPackages,
              totalGrossWeightKg: values.totalGrossWeightKg,
              totalVolumeCbm: values.totalVolumeCbm,
              totalPackageUnit: values.totalPackageUnit,
              notes: values.notes,
              shippingDocuments: values.shippingDocuments,
              containerRequests: values.containerRequests,
            },
          );
          message.success('更新订单成功');
          setOpen(false);
          onSuccess();
          return true;
        }}
      >
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="customerId"
          label="客户单位"
          rules={[{ required: true, message: '请选择客户' }]}
          placeholder="搜索客户"
          request={async ({ keyWords }) => searchCustomers(keyWords)}
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="tradeDirection"
          label="贸易方向"
          rules={[{ required: true, message: '请选择贸易方向' }]}
          options={tradeDirectionOptions}
          placeholder="请选择贸易方向"
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="tradeTerm"
          label="贸易条款"
          rules={[{ required: true, message: '请选择贸易条款' }]}
          options={tradeTermOptions}
          placeholder="请选择贸易条款"
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="paymentTerm"
          label="运费条款"
          rules={[{ required: true, message: '请选择付款条款' }]}
          options={paymentTermOptions}
          placeholder="请选择付款条款"
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="shipmentType"
          label="装载方式"
          options={shipmentTypeOptions}
          placeholder="请选择装载类型"
        />
        <OrderShippingDocumentFields
          transportMode={category === 'air' ? 'air' : 'sea'}
        />
        {category === 'sea' && (
          <SeaContainerPlanFields options={containerSpecOptions} />
        )}
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="originLocationId"
          label="起运港 / 地点"
          options={locationOptions}
          request={async ({ keyWords }) => searchLocations(keyWords)}
          fieldProps={{ filterOption: false }}
          placeholder="请选择起运地点"
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="destinationLocationId"
          label="目的港 / 地点"
          options={locationOptions}
          request={async ({ keyWords }) => searchLocations(keyWords)}
          fieldProps={{ filterOption: false }}
          placeholder="请选择目的地点"
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="vesselVoyage"
          label="船名航次 / 车次 / 航班号"
          placeholder="请输入船名航次/航班号"
        />
        <ProFormDatePicker
          colProps={{ span: 12 }}
          name="etd"
          label="预计离港时间 (ETD)"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDatePicker
          colProps={{ span: 12 }}
          name="eta"
          label="预计到达时间 (ETA)"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="serviceTypeIds"
          label="服务类型"
          mode="multiple"
          options={serviceTypeOptions}
          placeholder="请选择服务类型"
        />
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="cargoCategoryIds"
          label="货物类别"
          mode="multiple"
          options={cargoCategoryOptions}
          placeholder="请选择货物类别"
        />
        <ProFormDigit
          colProps={{ span: 12 }}
          name="totalPackages"
          label="总件数"
          min={0}
          placeholder="请输入件数"
        />
        <ProFormDigit
          colProps={{ span: 12 }}
          name="totalGrossWeightKg"
          label="委托总毛重 (KGS)"
          min={0}
          fieldProps={{ precision: 3 }}
          placeholder="请输入毛重"
        />
        <ProFormDigit
          colProps={{ span: 12 }}
          name="totalVolumeCbm"
          label="委托总体积 (CBM)"
          min={0}
          fieldProps={{ precision: 3 }}
          placeholder="请输入体积"
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="totalPackageUnit"
          label="包装单位"
          placeholder="例如: CTNS, PLTS"
        />
        <ProFormTextArea
          colProps={{ span: 24 }}
          name="goodsDescription"
          label="货物描述"
          placeholder="请输入货物描述"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
        <ProFormTextArea
          colProps={{ span: 24 }}
          name="notes"
          label="业务备注"
          placeholder="请输入备注说明"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
      </ModalForm>
    );
  },
);

export default EditOrderModal;
