import {
  EditOutlined,
  FlagOutlined,
  PaperClipOutlined,
  PlusOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Alert, App, Button, Drawer, Space, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import {
  masterDataServiceListOptions,
  masterDataServiceListStatusTemplates,
} from '@/services/roncin/masterDataService';
import {
  orderAttachmentServiceListAttachments,
  orderAttachmentServiceRegisterAttachment,
} from '@/services/roncin/orderAttachmentService';
import {
  orderMilestoneServiceListMilestones,
  orderMilestoneServiceSetMilestone,
} from '@/services/roncin/orderMilestoneService';
import {
  orderServiceCreateOrder,
  orderServiceListOrders,
  orderServiceTransitionOrderStatus,
  orderServiceUpdateOrder,
} from '@/services/roncin/orderService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

const businessTypeOptions = [
  { label: '海运出口', value: 1 },
  { label: '海运进口', value: 2 },
  { label: '空运出口', value: 3 },
  { label: '空运进口', value: 4 },
  { label: '陆运', value: 5 },
  { label: '铁路', value: 6 },
];

const businessTypeValueEnum = Object.fromEntries(
  businessTypeOptions.map((opt) => [opt.value, { text: opt.label }]),
);

const tradeDirectionOptions = [
  { label: '出口', value: 1 },
  { label: '进口', value: 2 },
];

const tradeDirectionValueEnum = Object.fromEntries(
  tradeDirectionOptions.map((opt) => [opt.value, { text: opt.label }]),
);

const tradeTermOptions = [
  { label: 'EXW', value: 1 },
  { label: 'FCA', value: 2 },
  { label: 'FOB', value: 3 },
  { label: 'CFR', value: 4 },
  { label: 'CIF', value: 5 },
  { label: 'CPT', value: 6 },
  { label: 'CIP', value: 7 },
  { label: 'DAP', value: 8 },
  { label: 'DPU', value: 9 },
  { label: 'DDU', value: 10 },
  { label: 'DDP', value: 11 },
  { label: 'LDP', value: 12 },
];

const paymentTermOptions = [
  { label: '预付 (PP)', value: 1 },
  { label: '到付 (CC)', value: 2 },
];

const shipmentTypeOptions = [
  { label: '整箱 (FCL)', value: 1 },
  { label: '拼箱 (LCL)', value: 2 },
  { label: '散杂货 (Break Bulk)', value: 3 },
];

const MASTER_DATA_KINDS = {
  REGION: 3,
  PORT: 4,
  AIRPORT: 5,
  SERVICE_TYPE: 8,
  CARGO_CATEGORY: 9,
} as const;

type OrderFormValues = {
  customerId: string;
  businessType: number;
  tradeDirection: number;
  tradeTerm: number;
  paymentTerm: number;
  statusTemplateId?: string;
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
  totalPackageUnit?: string;
  notes?: string;
};

type TransitionFormValues = {
  targetStatus: string;
  reason?: string;
};

type MilestoneFormValues = {
  type: string;
  occurredAt?: string;
  clearOccurredAt?: boolean;
  note?: string;
};

type AttachmentFormValues = {
  docType: string;
  idempotencyKey: string;
  fileName: string;
  mimeType: string;
  fileSize: string | number;
  objectKey: string;
  checksum?: string;
};

export default function Orders() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const createFormRef = useRef<ProFormInstance | undefined>(undefined);
  const editFormRef = useRef<ProFormInstance | undefined>(undefined);
  const transitionFormRef = useRef<ProFormInstance | undefined>(undefined);
  const milestoneActionRef = useRef<ActionType | undefined>(undefined);
  const milestoneFormRef = useRef<ProFormInstance | undefined>(undefined);
  const attachmentActionRef = useRef<ActionType | undefined>(undefined);
  const attachmentFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [transitionModalOpen, setTransitionModalOpen] = useState(false);
  const [milestoneDrawerOpen, setMilestoneDrawerOpen] = useState(false);
  const [milestoneModalOpen, setMilestoneModalOpen] = useState(false);
  const [attachmentDrawerOpen, setAttachmentDrawerOpen] = useState(false);
  const [attachmentModalOpen, setAttachmentModalOpen] = useState(false);

  const [editingRecord, setEditingRecord] = useState<API.Order>();
  const [transitionRecord, setTransitionRecord] = useState<API.Order>();
  const [milestoneOrder, setMilestoneOrder] = useState<API.Order>();
  const [attachmentOrder, setAttachmentOrder] = useState<API.Order>();
  const [editingMilestone, setEditingMilestone] = useState<API.OrderMilestone>();
  const [targetStatusOptions, setTargetStatusOptions] = useState<
    { label: string; value: string }[]
  >([]);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});

  useEffect(() => {
    masterDataServiceListOptions().then((res) => {
      setMasterOptions(res.data ?? []);
    });
  }, []);

  const serviceTypeOptions = masterOptions
    .filter(
      (item) =>
        item.kind === MASTER_DATA_KINDS.SERVICE_TYPE && item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const cargoCategoryOptions = masterOptions
    .filter(
      (item) =>
        item.kind === MASTER_DATA_KINDS.CARGO_CATEGORY && item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const locationOptions = masterOptions
    .filter(
      (item) =>
        item.kind !== undefined &&
        [
          MASTER_DATA_KINDS.REGION,
          MASTER_DATA_KINDS.PORT,
          MASTER_DATA_KINDS.AIRPORT,
        ].includes(item.kind as 3 | 4 | 5) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const searchCustomers = async (keyword?: string) => {
    const res = await partnerServiceListPartners({
      role: 1,
      enabled: true,
      keyword,
    });
    const partners = res.data ?? [];
    setCustomerMap((prev) => {
      const next = { ...prev };
      for (const p of partners) {
        if (p.id) {
          next[p.id] = p.legalName
            ? `${p.legalName} (${p.code})`
            : p.code || p.id;
        }
      }
      return next;
    });
    return partners.map((p) => ({
      label: p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id,
      value: p.id ?? '',
    }));
  };

  const loadStatusTemplates = async (businessType?: number) => {
    if (!businessType) {
      return [];
    }
    const res = await masterDataServiceListStatusTemplates({
      businessType,
      published: true,
    });
    return (res.data ?? [])
      .filter(
        (tpl) =>
          tpl.enabled !== false &&
          (tpl.items ?? []).some((item) => item.code === 'DRAFT'),
      )
      .map((tpl) => ({
        label: `${tpl.name} (v${tpl.version})`,
        value: tpl.id ?? '',
      }));
  };

  const openCreate = () => {
    createFormRef.current?.resetFields();
    setCreateModalOpen(true);
  };

  const openEdit = (record: API.Order) => {
    setEditingRecord(record);
    editFormRef.current?.setFieldsValue({
      customerId: record.customerId,
      businessType: record.businessType,
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
      totalPackageUnit: record.totalPackageUnit,
      notes: record.notes,
    });
    setEditModalOpen(true);
  };

  const openTransition = async (record: API.Order) => {
    setTransitionRecord(record);
    transitionFormRef.current?.setFieldsValue({
      currentStatus: record.status,
      targetStatus: undefined,
      reason: undefined,
    });
    const res = await masterDataServiceListStatusTemplates({
      businessType: record.businessType,
      published: true,
    });
    const template = (res.data ?? []).find(
      (tpl) => tpl.id === record.statusTemplateId,
    );
    const validTargets = (template?.items ?? [])
      .filter(
        (item) =>
          item.enabled !== false &&
          item.code !== record.status &&
          item.code !== 'DRAFT',
      )
      .map((item) => ({
        label: item.code ? `${item.label} (${item.code})` : (item.label ?? ''),
        value: item.code ?? '',
      }));
    setTargetStatusOptions(validTargets);
    setTransitionModalOpen(true);
  };

  const openMilestones = (record: API.Order) => {
    setMilestoneOrder(record);
    setMilestoneDrawerOpen(true);
  };

  const openCreateMilestone = () => {
    setEditingMilestone(undefined);
    milestoneFormRef.current?.resetFields();
    setMilestoneModalOpen(true);
  };

  const openEditMilestone = (record: API.OrderMilestone) => {
    setEditingMilestone(record);
    milestoneFormRef.current?.setFieldsValue({
      type: record.type,
      occurredAt: record.occurredAt ? dayjs(record.occurredAt) : undefined,
      clearOccurredAt: false,
      note: record.note,
    });
    setMilestoneModalOpen(true);
  };

  const openAttachments = (record: API.Order) => {
    setAttachmentOrder(record);
    setAttachmentDrawerOpen(true);
  };

  const openRegisterAttachment = () => {
    attachmentFormRef.current?.resetFields();
    setAttachmentModalOpen(true);
  };

  const attachmentColumns: ProColumns<API.OrderAttachment>[] = [
    {
      title: '文档类型',
      dataIndex: 'docType',
      width: 140,
      ellipsis: true,
    },
    {
      title: '文件名',
      dataIndex: 'fileName',
      ellipsis: true,
    },
    {
      title: 'MIME 类型',
      dataIndex: 'mimeType',
      width: 140,
      ellipsis: true,
    },
    {
      title: '文件大小',
      dataIndex: 'fileSize',
      width: 120,
    },
    {
      title: '对象键',
      dataIndex: 'objectKey',
      copyable: true,
      ellipsis: true,
    },
    {
      title: '校验和',
      dataIndex: 'checksum',
      copyable: true,
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
    },
  ];

  const milestoneColumns: ProColumns<API.OrderMilestone>[] = [
    {
      title: '类型',
      dataIndex: 'type',
      width: 140,
      render: (_, record) => record.type || '-',
    },
    {
      title: '节点编码',
      dataIndex: 'templateNodeCode',
      width: 140,
      render: (_, record) => record.templateNodeCode || '-',
    },
    {
      title: '节点名称',
      dataIndex: 'templateNodeLabel',
      width: 140,
      render: (_, record) => record.templateNodeLabel || '-',
    },
    {
      title: '发生时间',
      dataIndex: 'occurredAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.occurredAt
          ? dayjs(record.occurredAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '备注',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      render: (_, record) => {
        if (!access.canManageOrders) return null;
        return (
          <Button
            type="link"
            size="small"
            onClick={() => openEditMilestone(record)}
          >
            编辑
          </Button>
        );
      },
    },
  ];

  const columns: ProColumns<API.Order>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      valueType: 'text',
      fieldProps: {
        placeholder: '搜索订单号/备注/描述',
      },
    },
    {
      title: '订单号',
      dataIndex: 'orderNo',
      copyable: true,
      search: false,
      render: (_, record) => record.orderNo || '-',
    },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      valueType: 'select',
      valueEnum: businessTypeValueEnum,
      render: (_, record) =>
        businessTypeValueEnum[record.businessType ?? 0]?.text || '-',
    },
    {
      title: '客户',
      dataIndex: 'customerId',
      valueType: 'select',
      fieldProps: {
        showSearch: true,
        placeholder: '搜索客户',
      },
      request: async ({ keyWords }) => searchCustomers(keyWords),
      render: (_, record) =>
        customerMap[record.customerId ?? ''] || record.customerId || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      valueType: 'text',
      render: (_, record) => (
        <Tag color={record.status === 'DRAFT' ? 'default' : 'blue'}>
          {record.status || '-'}
        </Tag>
      ),
    },
    {
      title: '贸易方向',
      dataIndex: 'tradeDirection',
      search: false,
      valueType: 'select',
      valueEnum: tradeDirectionValueEnum,
      render: (_, record) =>
        tradeDirectionValueEnum[record.tradeDirection ?? 0]?.text || '-',
    },
    {
      title: 'ETD',
      dataIndex: 'etd',
      search: false,
      render: (_, record) =>
        record.etd ? dayjs(record.etd).format('YYYY-MM-DD') : '-',
    },
    {
      title: 'ETA',
      dataIndex: 'eta',
      search: false,
      render: (_, record) =>
        record.eta ? dayjs(record.eta).format('YYYY-MM-DD') : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      key: 'option',
      width: 280,
      render: (_, record) => {
        if (!access.canReadOrders && !access.canManageOrders) return null;
        return (
          <Space size="small">
            {access.canManageOrders && record.status === 'DRAFT' && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
            {access.canManageOrders && (
              <Button
                type="link"
                size="small"
                icon={<SwapOutlined />}
                onClick={() => openTransition(record)}
              >
                状态流转
              </Button>
            )}
            <Button
              type="link"
              size="small"
              icon={<FlagOutlined />}
              onClick={() => openMilestones(record)}
            >
              里程碑
            </Button>
            <Button
              type="link"
              size="small"
              icon={<PaperClipOutlined />}
              onClick={() => openAttachments(record)}
            >
              附件
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <>
      <ProTable<API.Order>
        headerTitle="订单列表"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        request={async (params) => {
          const response = await orderServiceListOrders({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            status: params.status,
            businessType: params.businessType,
            customerId: params.customerId,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? false,
            total: response.total ?? 0,
          };
        }}
        toolBarRender={() => [
          access.canManageOrders && (
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={openCreate}
            >
              新增订单
            </Button>
          ),
        ]}
      />

      <ModalForm<OrderFormValues>
        title="新增订单"
        open={createModalOpen}
        formRef={createFormRef}
        grid
        modalProps={{
          destroyOnHidden: true,
          width: 800,
          onCancel: () => setCreateModalOpen(false),
        }}
        onOpenChange={setCreateModalOpen}
        onFinish={async (values) => {
          if (!values.statusTemplateId) return false;
          await orderServiceCreateOrder({
            customerId: values.customerId,
            businessType: values.businessType,
            tradeDirection: values.tradeDirection,
            tradeTerm: values.tradeTerm,
            paymentTerm: values.paymentTerm,
            statusTemplateId: values.statusTemplateId,
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
            totalPackageUnit: values.totalPackageUnit,
            notes: values.notes,
          });
          message.success('创建订单成功');
          setCreateModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          colProps={{ span: 12 }}
          name="customerId"
          label="客户"
          rules={[{ required: true, message: '请选择客户' }]}
          fieldProps={{
            showSearch: true,
            placeholder: '搜索客户',
          }}
          request={async ({ keyWords }) => searchCustomers(keyWords)}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="businessType"
          label="业务类型"
          rules={[{ required: true, message: '请选择业务类型' }]}
          options={businessTypeOptions}
          placeholder="请选择业务类型"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="statusTemplateId"
          label="状态模板"
          rules={[{ required: true, message: '请选择状态模板' }]}
          dependencies={['businessType']}
          request={async ({ businessType }) =>
            loadStatusTemplates(businessType ? Number(businessType) : undefined)
          }
          placeholder="请选择状态模板"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="tradeDirection"
          label="贸易方向"
          rules={[{ required: true, message: '请选择贸易方向' }]}
          options={tradeDirectionOptions}
          placeholder="请选择贸易方向"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="tradeTerm"
          label="贸易条款"
          rules={[{ required: true, message: '请选择贸易条款' }]}
          options={tradeTermOptions}
          placeholder="请选择贸易条款"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="paymentTerm"
          label="付款条款"
          rules={[{ required: true, message: '请选择付款条款' }]}
          options={paymentTermOptions}
          placeholder="请选择付款条款"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="shipmentType"
          label="装载类型"
          options={shipmentTypeOptions}
          placeholder="请选择装载类型"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="originLocationId"
          label="起运地点"
          options={locationOptions}
          fieldProps={{ showSearch: true }}
          placeholder="请选择起运地点"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="destinationLocationId"
          label="目的地点"
          options={locationOptions}
          fieldProps={{ showSearch: true }}
          placeholder="请选择目的地点"
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="vesselVoyage"
          label="船名航次"
          placeholder="请输入船名航次"
        />
        <ProFormDatePicker
          colProps={{ span: 12 }}
          name="etd"
          label="ETD"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDatePicker
          colProps={{ span: 12 }}
          name="eta"
          label="ETA"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="serviceTypeIds"
          label="服务类型"
          mode="multiple"
          options={serviceTypeOptions}
          placeholder="请选择服务类型"
        />
        <ProFormSelect
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
          label="件数"
          min={0}
          placeholder="请输入件数"
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
          label="备注"
          placeholder="请输入备注"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
      </ModalForm>

      <ModalForm<OrderFormValues>
        title="编辑订单草稿"
        open={editModalOpen}
        formRef={editFormRef}
        grid
        initialValues={
          editingRecord
            ? {
                customerId: editingRecord.customerId,
                businessType: editingRecord.businessType,
                tradeDirection: editingRecord.tradeDirection,
                tradeTerm: editingRecord.tradeTerm,
                paymentTerm: editingRecord.paymentTerm,
                shipmentType: editingRecord.shipmentType,
                serviceTypeIds: editingRecord.serviceTypeIds,
                cargoCategoryIds: editingRecord.cargoCategoryIds,
                originLocationId: editingRecord.originLocationId,
                destinationLocationId: editingRecord.destinationLocationId,
                vesselVoyage: editingRecord.vesselVoyage,
                etd: editingRecord.etd ? dayjs(editingRecord.etd) : undefined,
                eta: editingRecord.eta ? dayjs(editingRecord.eta) : undefined,
                goodsDescription: editingRecord.goodsDescription,
                totalPackages: editingRecord.totalPackages,
                totalPackageUnit: editingRecord.totalPackageUnit,
                notes: editingRecord.notes,
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 800,
          onCancel: () => setEditModalOpen(false),
        }}
        onOpenChange={setEditModalOpen}
        onFinish={async (values) => {
          if (!editingRecord?.id || !editingRecord?.status) return false;
          await orderServiceUpdateOrder(
            { id: editingRecord.id },
            {
              id: editingRecord.id,
              expectedStatus: editingRecord.status,
              customerId: values.customerId,
              businessType: values.businessType,
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
              totalPackageUnit: values.totalPackageUnit,
              notes: values.notes,
            },
          );
          message.success('更新订单成功');
          setEditModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          colProps={{ span: 12 }}
          name="customerId"
          label="客户"
          rules={[{ required: true, message: '请选择客户' }]}
          fieldProps={{
            showSearch: true,
            placeholder: '搜索客户',
          }}
          request={async ({ keyWords }) => searchCustomers(keyWords)}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="businessType"
          label="业务类型"
          rules={[{ required: true, message: '请选择业务类型' }]}
          options={businessTypeOptions}
          placeholder="请选择业务类型"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="tradeDirection"
          label="贸易方向"
          rules={[{ required: true, message: '请选择贸易方向' }]}
          options={tradeDirectionOptions}
          placeholder="请选择贸易方向"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="tradeTerm"
          label="贸易条款"
          rules={[{ required: true, message: '请选择贸易条款' }]}
          options={tradeTermOptions}
          placeholder="请选择贸易条款"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="paymentTerm"
          label="付款条款"
          rules={[{ required: true, message: '请选择付款条款' }]}
          options={paymentTermOptions}
          placeholder="请选择付款条款"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="shipmentType"
          label="装载类型"
          options={shipmentTypeOptions}
          placeholder="请选择装载类型"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="originLocationId"
          label="起运地点"
          options={locationOptions}
          fieldProps={{ showSearch: true }}
          placeholder="请选择起运地点"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="destinationLocationId"
          label="目的地点"
          options={locationOptions}
          fieldProps={{ showSearch: true }}
          placeholder="请选择目的地点"
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="vesselVoyage"
          label="船名航次"
          placeholder="请输入船名航次"
        />
        <ProFormDatePicker
          colProps={{ span: 12 }}
          name="etd"
          label="ETD"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDatePicker
          colProps={{ span: 12 }}
          name="eta"
          label="ETA"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="serviceTypeIds"
          label="服务类型"
          mode="multiple"
          options={serviceTypeOptions}
          placeholder="请选择服务类型"
        />
        <ProFormSelect
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
          label="件数"
          min={0}
          placeholder="请输入件数"
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
          label="备注"
          placeholder="请输入备注"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
      </ModalForm>

      <ModalForm<TransitionFormValues>
        title="状态流转"
        open={transitionModalOpen}
        formRef={transitionFormRef}
        initialValues={
          transitionRecord
            ? { currentStatus: transitionRecord.status }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 520,
          onCancel: () => setTransitionModalOpen(false),
        }}
        onOpenChange={setTransitionModalOpen}
        onFinish={async (values) => {
          if (!transitionRecord?.id || !transitionRecord?.status) return false;
          await orderServiceTransitionOrderStatus(
            { id: transitionRecord.id },
            {
              id: transitionRecord.id,
              expectedStatus: transitionRecord.status,
              targetStatus: values.targetStatus,
              reason: values.reason,
            },
          );
          message.success('状态流转成功');
          setTransitionModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="currentStatus"
          label="当前状态"
          readonly
          initialValue={transitionRecord?.status}
        />
        <ProFormSelect
          name="targetStatus"
          label="目标状态"
          rules={[{ required: true, message: '请选择目标状态' }]}
          options={targetStatusOptions}
          placeholder="请选择目标状态"
        />
        <ProFormTextArea
          name="reason"
          label="流转原因"
          placeholder="请输入流转原因"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          milestoneOrder
            ? `订单里程碑 - ${milestoneOrder.orderNo || milestoneOrder.id}`
            : '订单里程碑'
        }
        open={milestoneDrawerOpen}
        onClose={() => {
          setMilestoneDrawerOpen(false);
          setMilestoneOrder(undefined);
        }}
        width={800}
        destroyOnHidden
      >
        {milestoneOrder?.id && (
          <ProTable<API.OrderMilestone>
            actionRef={milestoneActionRef}
            rowKey={(record) => record.id || record.type || ''}
            columns={milestoneColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderMilestoneServiceListMilestones({
                orderId: milestoneOrder.id as string,
              });
              return {
                data: response.data ?? [],
                success: response.success ?? true,
              };
            }}
            toolBarRender={() => [
              access.canManageOrders && (
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openCreateMilestone}
                >
                  设置里程碑
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<MilestoneFormValues>
        title={editingMilestone ? '编辑里程碑' : '设置里程碑'}
        open={milestoneModalOpen}
        formRef={milestoneFormRef}
        modalProps={{
          destroyOnHidden: true,
          width: 520,
          onCancel: () => setMilestoneModalOpen(false),
        }}
        onOpenChange={setMilestoneModalOpen}
        onFinish={async (values) => {
          if (!milestoneOrder?.id || !milestoneOrder?.status) return false;
          const milestoneType = (
            editingMilestone?.type ||
            values.type ||
            ''
          ).trim();
          if (!milestoneType) return false;

          await orderMilestoneServiceSetMilestone(
            {
              orderId: milestoneOrder.id,
              type: milestoneType,
            },
            {
              orderId: milestoneOrder.id,
              type: milestoneType,
              expectedOrderStatus: milestoneOrder.status,
              occurredAt: values.clearOccurredAt
                ? undefined
                : values.occurredAt
                  ? dayjs(values.occurredAt).toISOString()
                  : undefined,
              clearOccurredAt: Boolean(values.clearOccurredAt),
              note: values.note,
            },
          );
          message.success('里程碑设置成功');
          setMilestoneModalOpen(false);
          milestoneActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="type"
          label="里程碑类型"
          placeholder="请输入里程碑类型 (如 BOOKING_CONFIRMED)"
          disabled={Boolean(editingMilestone?.type)}
          rules={[{ required: true, message: '请输入里程碑类型' }]}
        />
        <ProFormDateTimePicker
          name="occurredAt"
          label="发生时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormSwitch
          name="clearOccurredAt"
          label="清除发生时间"
        />
        <ProFormTextArea
          name="note"
          label="备注"
          placeholder="请输入备注"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          attachmentOrder
            ? `订单附件 - ${attachmentOrder.orderNo || attachmentOrder.id}`
            : '订单附件'
        }
        open={attachmentDrawerOpen}
        onClose={() => {
          setAttachmentDrawerOpen(false);
          setAttachmentOrder(undefined);
        }}
        width={900}
        destroyOnHidden
      >
        {attachmentOrder?.id && (
          <ProTable<API.OrderAttachment>
            actionRef={attachmentActionRef}
            rowKey={(record) => record.id || record.objectKey || ''}
            columns={attachmentColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderAttachmentServiceListAttachments({
                orderId: attachmentOrder.id as string,
              });
              return {
                data: response.data ?? [],
                success: response.success ?? true,
              };
            }}
            toolBarRender={() => [
              access.canManageOrders && (
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openRegisterAttachment}
                >
                  登记附件
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<AttachmentFormValues>
        title="登记附件"
        open={attachmentModalOpen}
        formRef={attachmentFormRef}
        modalProps={{
          destroyOnHidden: true,
          width: 560,
          onCancel: () => setAttachmentModalOpen(false),
        }}
        onOpenChange={setAttachmentModalOpen}
        onFinish={async (values) => {
          if (!attachmentOrder?.id) return false;
          await orderAttachmentServiceRegisterAttachment(
            { orderId: attachmentOrder.id },
            {
              orderId: attachmentOrder.id,
              docType: values.docType.trim(),
              idempotencyKey: values.idempotencyKey.trim(),
              fileName: values.fileName.trim(),
              mimeType: values.mimeType.trim(),
              fileSize: String(values.fileSize),
              objectKey: values.objectKey.trim(),
              checksum: values.checksum?.trim() || undefined,
            },
          );
          message.success('登记附件成功');
          setAttachmentModalOpen(false);
          attachmentActionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          type="info"
          showIcon
          message="此处仅登记外部对象存储引用，不上传文件内容。"
          style={{ marginBottom: 16 }}
        />
        <ProFormText
          name="docType"
          label="文档类型"
          placeholder="请输入文档类型 (如 BL, INVOICE, PACKING_LIST)"
          rules={[{ required: true, message: '请输入文档类型' }]}
        />
        <ProFormText
          name="idempotencyKey"
          label="幂等键"
          placeholder="请输入幂等键"
          rules={[{ required: true, message: '请输入幂等键' }]}
        />
        <ProFormText
          name="fileName"
          label="文件名"
          placeholder="请输入文件名"
          rules={[{ required: true, message: '请输入文件名' }]}
        />
        <ProFormText
          name="mimeType"
          label="MIME 类型"
          placeholder="请输入 MIME 类型 (如 application/pdf)"
          rules={[{ required: true, message: '请输入 MIME 类型' }]}
        />
        <ProFormDigit
          name="fileSize"
          label="文件大小"
          min={1}
          fieldProps={{ precision: 0 }}
          placeholder="请输入文件大小 (字节)"
          rules={[{ required: true, message: '请输入文件大小' }]}
        />
        <ProFormText
          name="objectKey"
          label="对象键"
          placeholder="请输入对象键"
          rules={[{ required: true, message: '请输入对象键' }]}
        />
        <ProFormText
          name="checksum"
          label="校验和"
          placeholder="请输入校验和 (可选)"
        />
      </ModalForm>
    </>
  );
}
