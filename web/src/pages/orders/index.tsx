import {
  ContainerOutlined,
  EditOutlined,
  FileTextOutlined,
  FlagOutlined,
  InboxOutlined,
  PaperClipOutlined,
  PlusOutlined,
  SwapOutlined,
  TeamOutlined,
  WarningOutlined,
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
import { Alert, App, Button, Drawer, Popconfirm, Space, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import {
  masterDataServiceListOptions,
  masterDataServiceListStatusTemplates,
} from '@/services/roncin/masterDataService';
import {
  orderAbnormalCaseServiceListAbnormalCases,
  orderAbnormalCaseServiceMarkAbnormalCase,
  orderAbnormalCaseServiceRemoveAbnormalCase,
  orderAbnormalCaseServiceResolveAbnormalCase,
} from '@/services/roncin/orderAbnormalCaseService';
import {
  orderAttachmentServiceListAttachments,
  orderAttachmentServiceRegisterAttachment,
} from '@/services/roncin/orderAttachmentService';
import {
  orderCargoItemServiceAddCargoItem,
  orderCargoItemServiceListCargoItems,
  orderCargoItemServiceRemoveCargoItem,
  orderCargoItemServiceUpdateCargoItem,
} from '@/services/roncin/orderCargoItemService';
import {
  orderContainerServiceAddContainer,
  orderContainerServiceListContainers,
  orderContainerServiceRemoveContainer,
  orderContainerServiceUpdateContainer,
} from '@/services/roncin/orderContainerService';
import {
  orderMilestoneServiceListMilestones,
  orderMilestoneServiceSetMilestone,
} from '@/services/roncin/orderMilestoneService';
import {
  orderPersonnelServiceAssignPersonnel,
  orderPersonnelServiceListPersonnel,
  orderPersonnelServiceRemovePersonnel,
} from '@/services/roncin/orderPersonnelService';
import {
  orderServiceCreateOrder,
  orderServiceListOrders,
  orderServiceTransitionOrderStatus,
  orderServiceUpdateOrder,
} from '@/services/roncin/orderService';
import {
  orderShippingDocumentServiceAddShippingDocument,
  orderShippingDocumentServiceListShippingDocuments,
  orderShippingDocumentServiceRemoveShippingDocument,
  orderShippingDocumentServiceTransitionShippingDocumentStatus,
  orderShippingDocumentServiceUpdateShippingDocument,
} from '@/services/roncin/orderShippingDocumentService';
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
  CONTAINER_SPEC: 7,
  SERVICE_TYPE: 8,
  CARGO_CATEGORY: 9,
  ABNORMAL_CASE: 10,
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

const orderPersonnelRoleOptions = [
  { label: 'CREATOR', value: 1 },
  { label: 'OPERATOR', value: 2 },
  { label: 'SALES', value: 3 },
  { label: 'CUSTOMER_SERVICE', value: 4 },
  { label: 'DOCUMENT', value: 5 },
  { label: 'COMMERCIAL', value: 6 },
  { label: 'ASSOCIATE', value: 7 },
  { label: 'ASSOCIATE2', value: 8 },
];

const orderPersonnelRoleValueEnum = Object.fromEntries(
  orderPersonnelRoleOptions.map((opt) => [opt.value, { text: opt.label }]),
);

type PersonnelFormValues = {
  userId: string;
  role: number;
};

type ContainerFormValues = {
  containerNo: string;
  containerSpecId: string;
  shippingDocumentId?: string;
  sealNo?: string;
  grossWeightKg: number;
  volumeCbm: number;
  note?: string;
};

type CargoItemFormValues = {
  cargoName: string;
  packageCount: number;
  grossWeightKg: number;
  volumeCbm: number;
  netWeightKg?: number;
  note?: string;
};

type ShippingDocumentFormValues = {
  masterNo: string;
  houseNo: string;
  releaseType?: string;
  note?: string;
};

const shippingDocumentStatusValueEnum: Record<
  number,
  { text: string; status: 'Default' | 'Processing' | 'Success' }
> = {
  1: { text: '草稿', status: 'Default' },
  2: { text: '已确认', status: 'Processing' },
  3: { text: '已放货', status: 'Success' },
};

type AbnormalCaseFormValues = {
  abnormalCaseId: string;
};

const abnormalCaseStatusValueEnum: Record<
  number,
  { text: string; status: 'Error' | 'Success' }
> = {
  1: { text: '进行中', status: 'Error' },
  2: { text: '已解决', status: 'Success' },
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
  const personnelActionRef = useRef<ActionType | undefined>(undefined);
  const personnelFormRef = useRef<ProFormInstance | undefined>(undefined);
  const containerActionRef = useRef<ActionType | undefined>(undefined);
  const containerFormRef = useRef<ProFormInstance | undefined>(undefined);
  const cargoActionRef = useRef<ActionType | undefined>(undefined);
  const cargoFormRef = useRef<ProFormInstance | undefined>(undefined);
  const shippingDocumentActionRef = useRef<ActionType | undefined>(undefined);
  const shippingDocumentFormRef = useRef<ProFormInstance | undefined>(undefined);
  const abnormalCaseActionRef = useRef<ActionType | undefined>(undefined);
  const abnormalCaseFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [transitionModalOpen, setTransitionModalOpen] = useState(false);
  const [milestoneDrawerOpen, setMilestoneDrawerOpen] = useState(false);
  const [milestoneModalOpen, setMilestoneModalOpen] = useState(false);
  const [attachmentDrawerOpen, setAttachmentDrawerOpen] = useState(false);
  const [attachmentModalOpen, setAttachmentModalOpen] = useState(false);
  const [personnelDrawerOpen, setPersonnelDrawerOpen] = useState(false);
  const [personnelModalOpen, setPersonnelModalOpen] = useState(false);
  const [containerDrawerOpen, setContainerDrawerOpen] = useState(false);
  const [containerModalOpen, setContainerModalOpen] = useState(false);
  const [cargoDrawerOpen, setCargoDrawerOpen] = useState(false);
  const [cargoModalOpen, setCargoModalOpen] = useState(false);
  const [shippingDocumentDrawerOpen, setShippingDocumentDrawerOpen] =
    useState(false);
  const [shippingDocumentModalOpen, setShippingDocumentModalOpen] =
    useState(false);
  const [abnormalCaseDrawerOpen, setAbnormalCaseDrawerOpen] = useState(false);
  const [abnormalCaseModalOpen, setAbnormalCaseModalOpen] = useState(false);

  const [editingRecord, setEditingRecord] = useState<API.Order>();
  const [transitionRecord, setTransitionRecord] = useState<API.Order>();
  const [milestoneOrder, setMilestoneOrder] = useState<API.Order>();
  const [attachmentOrder, setAttachmentOrder] = useState<API.Order>();
  const [personnelOrder, setPersonnelOrder] = useState<API.Order>();
  const [containerOrder, setContainerOrder] = useState<API.Order>();
  const [containerDocuments, setContainerDocuments] = useState<
    API.OrderShippingDocument[]
  >([]);
  const [cargoOrder, setCargoOrder] = useState<API.Order>();
  const [shippingDocumentOrder, setShippingDocumentOrder] =
    useState<API.Order>();
  const [abnormalCaseOrder, setAbnormalCaseOrder] = useState<API.Order>();
  const [editingMilestone, setEditingMilestone] = useState<API.OrderMilestone>();
  const [editingContainer, setEditingContainer] = useState<API.OrderContainer>();
  const [editingCargoItem, setEditingCargoItem] = useState<API.OrderCargoItem>();
  const [editingShippingDocument, setEditingShippingDocument] =
    useState<API.OrderShippingDocument>();
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

  const containerSpecOptions = masterOptions
    .filter(
      (item) =>
        item.kind === MASTER_DATA_KINDS.CONTAINER_SPEC && item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const containerSpecMap = Object.fromEntries(
    masterOptions
      .filter(
        (item) => item.kind === MASTER_DATA_KINDS.CONTAINER_SPEC && item.id,
      )
      .map((item) => [
        item.id as string,
        item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      ]),
  );

  const abnormalCaseOptions = masterOptions
    .filter(
      (item) =>
        item.kind === MASTER_DATA_KINDS.ABNORMAL_CASE && item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const abnormalCaseMap = Object.fromEntries(
    masterOptions
      .filter(
        (item) => item.kind === MASTER_DATA_KINDS.ABNORMAL_CASE && item.id,
      )
      .map((item) => [
        item.id as string,
        item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      ]),
  );

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

  const openPersonnel = (record: API.Order) => {
    setPersonnelOrder(record);
    setPersonnelDrawerOpen(true);
  };

  const openAssignPersonnel = () => {
    personnelFormRef.current?.resetFields();
    setPersonnelModalOpen(true);
  };

  const openContainers = (record: API.Order) => {
    setContainerOrder(record);
    setContainerDrawerOpen(true);
    orderShippingDocumentServiceListShippingDocuments({
      orderId: record.id as string,
    }).then((res) => {
      setContainerDocuments(res.data ?? []);
    });
  };

  const containerDocumentOptions = containerDocuments.map((doc) => ({
    label: `${doc.masterNo} / ${doc.houseNo}`,
    value: doc.id ?? '',
  }));

  const containerDocumentMap = Object.fromEntries(
    containerDocuments
      .filter((doc) => doc.id)
      .map((doc) => [doc.id as string, `${doc.masterNo} / ${doc.houseNo}`]),
  );

  const openCreateContainer = () => {
    setEditingContainer(undefined);
    containerFormRef.current?.resetFields();
    setContainerModalOpen(true);
  };

  const openEditContainer = (record: API.OrderContainer) => {
    setEditingContainer(record);
    containerFormRef.current?.setFieldsValue({
      containerNo: record.containerNo,
      containerSpecId: record.containerSpecId,
      shippingDocumentId: record.shippingDocumentId,
      sealNo: record.sealNo,
      grossWeightKg: record.grossWeightKg,
      volumeCbm: record.volumeCbm,
      note: record.note,
    });
    setContainerModalOpen(true);
  };

  const openCargoItems = (record: API.Order) => {
    setCargoOrder(record);
    setCargoDrawerOpen(true);
  };

  const openCreateCargoItem = () => {
    setEditingCargoItem(undefined);
    cargoFormRef.current?.resetFields();
    setCargoModalOpen(true);
  };

  const openEditCargoItem = (record: API.OrderCargoItem) => {
    setEditingCargoItem(record);
    cargoFormRef.current?.setFieldsValue({
      cargoName: record.cargoName,
      packageCount: record.packageCount,
      grossWeightKg: record.grossWeightKg,
      volumeCbm: record.volumeCbm,
      netWeightKg: record.netWeightKg,
      note: record.note,
    });
    setCargoModalOpen(true);
  };

  const openShippingDocuments = (record: API.Order) => {
    setShippingDocumentOrder(record);
    setShippingDocumentDrawerOpen(true);
  };

  const openCreateShippingDocument = () => {
    setEditingShippingDocument(undefined);
    shippingDocumentFormRef.current?.resetFields();
    setShippingDocumentModalOpen(true);
  };

  const openEditShippingDocument = (record: API.OrderShippingDocument) => {
    setEditingShippingDocument(record);
    shippingDocumentFormRef.current?.setFieldsValue({
      masterNo: record.masterNo,
      houseNo: record.houseNo,
      releaseType: record.releaseType,
      note: record.note,
    });
    setShippingDocumentModalOpen(true);
  };

  const openAbnormalCases = (record: API.Order) => {
    setAbnormalCaseOrder(record);
    setAbnormalCaseDrawerOpen(true);
  };

  const openMarkAbnormalCase = () => {
    abnormalCaseFormRef.current?.resetFields();
    setAbnormalCaseModalOpen(true);
  };

  const abnormalCaseColumns: ProColumns<API.OrderAbnormalCase>[] = [
    {
      title: '异常类型',
      dataIndex: 'abnormalCaseId',
      ellipsis: true,
      render: (_, record) =>
        (record.abnormalCaseId && abnormalCaseMap[record.abnormalCaseId]) || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      valueType: 'select',
      valueEnum: abnormalCaseStatusValueEnum,
      render: (_, record) => {
        if (record.status === 1) {
          return <Tag color="error">进行中</Tag>;
        }
        if (record.status === 2) {
          return <Tag color="success">已解决</Tag>;
        }
        return '-';
      },
    },
    {
      title: '标记时间',
      dataIndex: 'markedAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.markedAt
          ? dayjs(record.markedAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '标记人',
      dataIndex: 'markedBy',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.markedBy || '-',
    },
    {
      title: '解决时间',
      dataIndex: 'resolvedAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.resolvedAt
          ? dayjs(record.resolvedAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '解决人',
      dataIndex: 'resolvedBy',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.resolvedBy || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      search: false,
      width: 140,
      render: (_, record) => {
        if (!access.canManageOrders) return null;
        return (
          <Space size="small">
            {record.status === 1 && (
              <Popconfirm
                title="确定解决该异常？"
                onConfirm={async () => {
                  if (!abnormalCaseOrder?.id || !record.id) return;
                  await orderAbnormalCaseServiceResolveAbnormalCase(
                    {
                      orderId: abnormalCaseOrder.id,
                      id: record.id,
                    },
                    {
                      orderId: abnormalCaseOrder.id,
                      id: record.id,
                    },
                  );
                  message.success('解决异常成功');
                  abnormalCaseActionRef.current?.reload();
                }}
              >
                <Button type="link" size="small">
                  解决
                </Button>
              </Popconfirm>
            )}
            <Popconfirm
              title="确定移除该异常？"
              onConfirm={async () => {
                if (!abnormalCaseOrder?.id || !record.id) return;
                await orderAbnormalCaseServiceRemoveAbnormalCase({
                  orderId: abnormalCaseOrder.id,
                  id: record.id,
                });
                message.success('移除异常成功');
                abnormalCaseActionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small">
                移除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const shippingDocumentColumns: ProColumns<API.OrderShippingDocument>[] = [
    {
      title: '主单号',
      dataIndex: 'masterNo',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.masterNo || '-',
    },
    {
      title: '分单号',
      dataIndex: 'houseNo',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.houseNo || '-',
    },
    {
      title: '放货类型',
      dataIndex: 'releaseType',
      ellipsis: true,
      render: (_, record) => record.releaseType || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      valueType: 'select',
      valueEnum: shippingDocumentStatusValueEnum,
      render: (_, record) => {
        if (record.status === 1) {
          return <Tag color="default">草稿</Tag>;
        }
        if (record.status === 2) {
          return <Tag color="processing">已确认</Tag>;
        }
        if (record.status === 3) {
          return <Tag color="success">已放货</Tag>;
        }
        return '-';
      },
    },
    {
      title: '备注',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.createdAt
          ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      search: false,
      width: 180,
      render: (_, record) => {
        if (!access.canManageOrders) return null;
        if (record.status !== 1 && record.status !== 2) return null;
        const currentText = record.status === 1 ? '草稿' : '已确认';
        const nextText = record.status === 1 ? '已确认' : '已放货';
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => openEditShippingDocument(record)}
            >
              编辑
            </Button>
            <Popconfirm
              title={`确定将提单状态从「${currentText}」流转为「${nextText}」？`}
              onConfirm={async () => {
                if (!shippingDocumentOrder?.id || !record.id || !record.status)
                  return;
                const toStatus = record.status === 1 ? 2 : 3;
                await orderShippingDocumentServiceTransitionShippingDocumentStatus(
                  {
                    orderId: shippingDocumentOrder.id,
                    id: record.id,
                  },
                  {
                    orderId: shippingDocumentOrder.id,
                    id: record.id,
                    expectedStatus: record.status,
                    toStatus,
                  },
                );
                message.success('流转提单状态成功');
                shippingDocumentActionRef.current?.reload();
              }}
            >
              <Button type="link" size="small">
                流转
              </Button>
            </Popconfirm>
            <Popconfirm
              title="确定移除该提单？"
              onConfirm={async () => {
                if (!shippingDocumentOrder?.id || !record.id) return;
                await orderShippingDocumentServiceRemoveShippingDocument({
                  orderId: shippingDocumentOrder.id,
                  id: record.id,
                });
                message.success('移除提单成功');
                shippingDocumentActionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small">
                删除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const containerColumns: ProColumns<API.OrderContainer>[] = [
    {
      title: '箱号',
      dataIndex: 'containerNo',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.containerNo || '-',
    },
    {
      title: '箱型',
      dataIndex: 'containerSpecId',
      ellipsis: true,
      render: (_, record) =>
        (record.containerSpecId && containerSpecMap[record.containerSpecId]) ||
        '-',
    },
    {
      title: '关联提单',
      dataIndex: 'shippingDocumentId',
      ellipsis: true,
      render: (_, record) =>
        (record.shippingDocumentId &&
          containerDocumentMap[record.shippingDocumentId]) ||
        '-',
    },
    {
      title: '封号',
      dataIndex: 'sealNo',
      ellipsis: true,
      render: (_, record) => record.sealNo || '-',
    },
    {
      title: '毛重(KG)',
      dataIndex: 'grossWeightKg',
      width: 120,
      render: (_, record) =>
        record.grossWeightKg !== undefined && record.grossWeightKg !== null
          ? record.grossWeightKg
          : '-',
    },
    {
      title: '体积(CBM)',
      dataIndex: 'volumeCbm',
      width: 120,
      render: (_, record) =>
        record.volumeCbm !== undefined && record.volumeCbm !== null
          ? record.volumeCbm
          : '-',
    },
    {
      title: '备注',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.createdAt
          ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 120,
      render: (_, record) => {
        if (!access.canManageOrders) return null;
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => openEditContainer(record)}
            >
              编辑
            </Button>
            <Popconfirm
              title="确定移除该集装箱？"
              onConfirm={async () => {
                if (!containerOrder?.id || !record.id) return;
                await orderContainerServiceRemoveContainer({
                  orderId: containerOrder.id,
                  id: record.id,
                });
                message.success('移除集装箱成功');
                containerActionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small">
                删除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const cargoColumns: ProColumns<API.OrderCargoItem>[] = [
    {
      title: '货名',
      dataIndex: 'cargoName',
      ellipsis: true,
      render: (_, record) => record.cargoName || '-',
    },
    {
      title: '件数',
      dataIndex: 'packageCount',
      width: 100,
      render: (_, record) =>
        record.packageCount !== undefined && record.packageCount !== null
          ? record.packageCount
          : '-',
    },
    {
      title: '毛重(KG)',
      dataIndex: 'grossWeightKg',
      width: 120,
      render: (_, record) =>
        record.grossWeightKg !== undefined && record.grossWeightKg !== null
          ? record.grossWeightKg
          : '-',
    },
    {
      title: '体积(CBM)',
      dataIndex: 'volumeCbm',
      width: 120,
      render: (_, record) =>
        record.volumeCbm !== undefined && record.volumeCbm !== null
          ? record.volumeCbm
          : '-',
    },
    {
      title: '净重(KG)',
      dataIndex: 'netWeightKg',
      width: 120,
      render: (_, record) =>
        record.netWeightKg !== undefined && record.netWeightKg !== null
          ? record.netWeightKg
          : '-',
    },
    {
      title: '备注',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.createdAt
          ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 120,
      render: (_, record) => {
        if (!access.canManageOrders) return null;
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => openEditCargoItem(record)}
            >
              编辑
            </Button>
            <Popconfirm
              title="确定移除该货物明细？"
              onConfirm={async () => {
                if (!cargoOrder?.id || !record.id) return;
                await orderCargoItemServiceRemoveCargoItem({
                  orderId: cargoOrder.id,
                  id: record.id,
                });
                message.success('移除货物明细成功');
                cargoActionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small">
                删除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const personnelColumns: ProColumns<API.OrderPersonnel>[] = [
    {
      title: '用户 ID',
      dataIndex: 'userId',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.userId || '-',
    },
    {
      title: '角色',
      dataIndex: 'role',
      valueType: 'select',
      valueEnum: orderPersonnelRoleValueEnum,
      render: (_, record) =>
        (record.role !== undefined &&
          orderPersonnelRoleValueEnum[record.role]?.text) ||
        '-',
    },
    {
      title: '分配时间',
      dataIndex: 'assignedAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.assignedAt
          ? dayjs(record.assignedAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      render: (_, record) => {
        if (!access.canManageOrders) return null;
        return (
          <Popconfirm
            title="确定移除该协作人员？"
            onConfirm={async () => {
              if (!personnelOrder?.id || !record.id) return;
              await orderPersonnelServiceRemovePersonnel({
                orderId: personnelOrder.id,
                id: record.id,
              });
              message.success('移除协作人员成功');
              personnelActionRef.current?.reload();
            }}
          >
            <Button type="link" danger size="small">
              删除
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

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
      width: 600,
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
            <Button
              type="link"
              size="small"
              icon={<TeamOutlined />}
              onClick={() => openPersonnel(record)}
            >
              人员
            </Button>
            <Button
              type="link"
              size="small"
              icon={<ContainerOutlined />}
              onClick={() => openContainers(record)}
            >
              集装箱
            </Button>
            <Button
              type="link"
              size="small"
              icon={<InboxOutlined />}
              onClick={() => openCargoItems(record)}
            >
              货物
            </Button>
            <Button
              type="link"
              size="small"
              icon={<FileTextOutlined />}
              onClick={() => openShippingDocuments(record)}
            >
              提单
            </Button>
            <Button
              type="link"
              size="small"
              icon={<WarningOutlined />}
              onClick={() => openAbnormalCases(record)}
            >
              异常
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

      <Drawer
        title={
          personnelOrder
            ? `订单协作人员 - ${personnelOrder.orderNo || personnelOrder.id}`
            : '订单协作人员'
        }
        open={personnelDrawerOpen}
        onClose={() => {
          setPersonnelDrawerOpen(false);
          setPersonnelOrder(undefined);
        }}
        width={800}
        destroyOnHidden
      >
        {personnelOrder?.id && (
          <ProTable<API.OrderPersonnel>
            actionRef={personnelActionRef}
            rowKey={(record) => record.id || `${record.userId}-${record.role}`}
            columns={personnelColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderPersonnelServiceListPersonnel({
                orderId: personnelOrder.id as string,
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
                  onClick={openAssignPersonnel}
                >
                  分配人员
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<PersonnelFormValues>
        title="分配协作人员"
        open={personnelModalOpen}
        formRef={personnelFormRef}
        modalProps={{
          destroyOnHidden: true,
          width: 520,
          onCancel: () => setPersonnelModalOpen(false),
        }}
        onOpenChange={setPersonnelModalOpen}
        onFinish={async (values) => {
          if (!personnelOrder?.id) return false;
          await orderPersonnelServiceAssignPersonnel(
            { orderId: personnelOrder.id },
            {
              orderId: personnelOrder.id,
              userId: values.userId.trim(),
              role: Number(values.role),
            },
          );
          message.success('分配协作人员成功');
          setPersonnelModalOpen(false);
          personnelActionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          type="info"
          showIcon
          message="当前录入的是组织内用户 UUID，后续可另建可分配用户目录。"
          style={{ marginBottom: 16 }}
        />
        <ProFormText
          name="userId"
          label="用户 UUID"
          placeholder="请输入组织内用户 UUID"
          rules={[{ required: true, message: '请输入用户 UUID' }]}
        />
        <ProFormSelect
          name="role"
          label="角色"
          rules={[{ required: true, message: '请选择角色' }]}
          options={orderPersonnelRoleOptions}
          placeholder="请选择角色"
        />
      </ModalForm>

      <Drawer
        title={
          containerOrder
            ? `订单集装箱 - ${containerOrder.orderNo || containerOrder.id}`
            : '订单集装箱'
        }
        open={containerDrawerOpen}
        onClose={() => {
          setContainerDrawerOpen(false);
          setContainerOrder(undefined);
        }}
        width={900}
        destroyOnHidden
      >
        {containerOrder?.id && (
          <ProTable<API.OrderContainer>
            actionRef={containerActionRef}
            rowKey="id"
            columns={containerColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderContainerServiceListContainers({
                orderId: containerOrder.id as string,
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
                  onClick={openCreateContainer}
                >
                  添加集装箱
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<ContainerFormValues>
        title={editingContainer ? '编辑集装箱' : '添加集装箱'}
        open={containerModalOpen}
        formRef={containerFormRef}
        initialValues={
          editingContainer
            ? {
                containerNo: editingContainer.containerNo,
                containerSpecId: editingContainer.containerSpecId,
                shippingDocumentId: editingContainer.shippingDocumentId,
                sealNo: editingContainer.sealNo,
                grossWeightKg: editingContainer.grossWeightKg,
                volumeCbm: editingContainer.volumeCbm,
                note: editingContainer.note,
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 560,
          onCancel: () => setContainerModalOpen(false),
        }}
        onOpenChange={setContainerModalOpen}
        onFinish={async (values) => {
          if (!containerOrder?.id) return false;
          if (editingContainer?.id) {
            await orderContainerServiceUpdateContainer(
              {
                orderId: containerOrder.id,
                id: editingContainer.id,
              },
              {
                id: editingContainer.id,
                orderId: containerOrder.id,
                containerNo: values.containerNo.trim(),
                containerSpecId: values.containerSpecId,
                shippingDocumentId: values.shippingDocumentId || undefined,
                sealNo: values.sealNo?.trim() || undefined,
                grossWeightKg: Number(values.grossWeightKg),
                volumeCbm: Number(values.volumeCbm),
                note: values.note?.trim() || undefined,
              },
            );
            message.success('更新集装箱成功');
          } else {
            await orderContainerServiceAddContainer(
              {
                orderId: containerOrder.id,
              },
              {
                orderId: containerOrder.id,
                containerNo: values.containerNo.trim(),
                containerSpecId: values.containerSpecId,
                shippingDocumentId: values.shippingDocumentId || undefined,
                sealNo: values.sealNo?.trim() || undefined,
                grossWeightKg: Number(values.grossWeightKg),
                volumeCbm: Number(values.volumeCbm),
                note: values.note?.trim() || undefined,
              },
            );
            message.success('添加集装箱成功');
          }
          setContainerModalOpen(false);
          containerActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="containerNo"
          label="箱号"
          placeholder="请输入箱号"
          rules={[{ required: true, message: '请输入箱号' }]}
        />
        <ProFormSelect
          name="containerSpecId"
          label="箱型"
          rules={[{ required: true, message: '请选择箱型' }]}
          options={containerSpecOptions}
          placeholder="请选择箱型"
        />
        <ProFormSelect
          name="shippingDocumentId"
          label="关联提单"
          options={containerDocumentOptions}
          placeholder="请选择关联提单 (可选)"
        />
        <ProFormText
          name="sealNo"
          label="封号"
          placeholder="请输入封号 (可选)"
        />
        <ProFormDigit
          name="grossWeightKg"
          label="毛重(KG)"
          min={0.001}
          placeholder="请输入毛重"
          rules={[{ required: true, message: '请输入毛重' }]}
        />
        <ProFormDigit
          name="volumeCbm"
          label="体积(CBM)"
          min={0.001}
          placeholder="请输入体积"
          rules={[{ required: true, message: '请输入体积' }]}
        />
        <ProFormTextArea
          name="note"
          label="备注"
          placeholder="请输入备注 (可选)"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          cargoOrder
            ? `订单货物明细 - ${cargoOrder.orderNo || cargoOrder.id}`
            : '订单货物明细'
        }
        open={cargoDrawerOpen}
        onClose={() => {
          setCargoDrawerOpen(false);
          setCargoOrder(undefined);
        }}
        width={900}
        destroyOnHidden
      >
        {cargoOrder?.id && (
          <ProTable<API.OrderCargoItem>
            actionRef={cargoActionRef}
            rowKey="id"
            columns={cargoColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderCargoItemServiceListCargoItems({
                orderId: cargoOrder.id as string,
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
                  onClick={openCreateCargoItem}
                >
                  添加货物明细
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<CargoItemFormValues>
        title={editingCargoItem ? '编辑货物明细' : '添加货物明细'}
        open={cargoModalOpen}
        formRef={cargoFormRef}
        initialValues={
          editingCargoItem
            ? {
                cargoName: editingCargoItem.cargoName,
                packageCount: editingCargoItem.packageCount,
                grossWeightKg: editingCargoItem.grossWeightKg,
                volumeCbm: editingCargoItem.volumeCbm,
                netWeightKg: editingCargoItem.netWeightKg,
                note: editingCargoItem.note,
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 560,
          onCancel: () => setCargoModalOpen(false),
        }}
        onOpenChange={setCargoModalOpen}
        onFinish={async (values) => {
          if (!cargoOrder?.id) return false;
          if (editingCargoItem?.id) {
            await orderCargoItemServiceUpdateCargoItem(
              {
                orderId: cargoOrder.id,
                id: editingCargoItem.id,
              },
              {
                id: editingCargoItem.id,
                orderId: cargoOrder.id,
                cargoName: values.cargoName.trim(),
                packageCount: Number(values.packageCount),
                grossWeightKg: Number(values.grossWeightKg),
                volumeCbm: Number(values.volumeCbm),
                netWeightKg:
                  values.netWeightKg !== undefined && values.netWeightKg !== null
                    ? Number(values.netWeightKg)
                    : undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('更新货物明细成功');
          } else {
            await orderCargoItemServiceAddCargoItem(
              {
                orderId: cargoOrder.id,
              },
              {
                orderId: cargoOrder.id,
                cargoName: values.cargoName.trim(),
                packageCount: Number(values.packageCount),
                grossWeightKg: Number(values.grossWeightKg),
                volumeCbm: Number(values.volumeCbm),
                netWeightKg:
                  values.netWeightKg !== undefined && values.netWeightKg !== null
                    ? Number(values.netWeightKg)
                    : undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('添加货物明细成功');
          }
          setCargoModalOpen(false);
          cargoActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="cargoName"
          label="货名"
          placeholder="请输入货名"
          rules={[{ required: true, message: '请输入货名' }]}
        />
        <ProFormDigit
          name="packageCount"
          label="件数"
          min={1}
          fieldProps={{ precision: 0 }}
          placeholder="请输入件数"
          rules={[{ required: true, message: '请输入件数' }]}
        />
        <ProFormDigit
          name="grossWeightKg"
          label="毛重(KG)"
          min={0.001}
          placeholder="请输入毛重"
          rules={[{ required: true, message: '请输入毛重' }]}
        />
        <ProFormDigit
          name="volumeCbm"
          label="体积(CBM)"
          min={0.001}
          placeholder="请输入体积"
          rules={[{ required: true, message: '请输入体积' }]}
        />
        <ProFormDigit
          name="netWeightKg"
          label="净重(KG)"
          min={0.001}
          placeholder="请输入净重 (可选)"
        />
        <ProFormTextArea
          name="note"
          label="备注"
          placeholder="请输入备注 (可选)"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          shippingDocumentOrder
            ? `订单提单 - ${shippingDocumentOrder.orderNo || shippingDocumentOrder.id}`
            : '订单提单'
        }
        open={shippingDocumentDrawerOpen}
        onClose={() => {
          setShippingDocumentDrawerOpen(false);
          setShippingDocumentOrder(undefined);
        }}
        width={900}
        destroyOnHidden
      >
        {shippingDocumentOrder?.id && (
          <ProTable<API.OrderShippingDocument>
            actionRef={shippingDocumentActionRef}
            rowKey="id"
            columns={shippingDocumentColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response =
                await orderShippingDocumentServiceListShippingDocuments({
                  orderId: shippingDocumentOrder.id as string,
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
                  onClick={openCreateShippingDocument}
                >
                  添加提单
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<ShippingDocumentFormValues>
        title={editingShippingDocument ? '编辑提单' : '添加提单'}
        open={shippingDocumentModalOpen}
        formRef={shippingDocumentFormRef}
        initialValues={
          editingShippingDocument
            ? {
                masterNo: editingShippingDocument.masterNo,
                houseNo: editingShippingDocument.houseNo,
                releaseType: editingShippingDocument.releaseType,
                note: editingShippingDocument.note,
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 560,
          onCancel: () => setShippingDocumentModalOpen(false),
        }}
        onOpenChange={setShippingDocumentModalOpen}
        onFinish={async (values) => {
          if (!shippingDocumentOrder?.id) return false;
          if (editingShippingDocument?.id) {
            await orderShippingDocumentServiceUpdateShippingDocument(
              {
                orderId: shippingDocumentOrder.id,
                id: editingShippingDocument.id,
              },
              {
                orderId: shippingDocumentOrder.id,
                id: editingShippingDocument.id,
                masterNo: values.masterNo.trim(),
                houseNo: values.houseNo.trim(),
                releaseType: values.releaseType?.trim() || undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('更新提单成功');
          } else {
            await orderShippingDocumentServiceAddShippingDocument(
              {
                orderId: shippingDocumentOrder.id,
              },
              {
                orderId: shippingDocumentOrder.id,
                masterNo: values.masterNo.trim(),
                houseNo: values.houseNo.trim(),
                releaseType: values.releaseType?.trim() || undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('添加提单成功');
          }
          setShippingDocumentModalOpen(false);
          shippingDocumentActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="masterNo"
          label="主单号"
          placeholder="请输入主单号"
          rules={[{ required: true, message: '请输入主单号' }]}
        />
        <ProFormText
          name="houseNo"
          label="分单号"
          placeholder="请输入分单号"
          rules={[{ required: true, message: '请输入分单号' }]}
        />
        <ProFormText
          name="releaseType"
          label="放货类型"
          placeholder="请输入放货类型 (可选)"
        />
        <ProFormTextArea
          name="note"
          label="备注"
          placeholder="请输入备注 (可选)"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          abnormalCaseOrder
            ? `订单异常 - ${abnormalCaseOrder.orderNo || abnormalCaseOrder.id}`
            : '订单异常'
        }
        open={abnormalCaseDrawerOpen}
        onClose={() => {
          setAbnormalCaseDrawerOpen(false);
          setAbnormalCaseOrder(undefined);
        }}
        width={900}
        destroyOnHidden
      >
        {abnormalCaseOrder?.id && (
          <ProTable<API.OrderAbnormalCase>
            actionRef={abnormalCaseActionRef}
            rowKey="id"
            columns={abnormalCaseColumns}
            search={false}
            pagination={false}
            request={async () => {
              const response =
                await orderAbnormalCaseServiceListAbnormalCases({
                  orderId: abnormalCaseOrder.id as string,
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
                  onClick={openMarkAbnormalCase}
                >
                  标记异常
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<AbnormalCaseFormValues>
        title="标记异常"
        open={abnormalCaseModalOpen}
        formRef={abnormalCaseFormRef}
        modalProps={{
          destroyOnHidden: true,
          width: 520,
          onCancel: () => setAbnormalCaseModalOpen(false),
        }}
        onOpenChange={setAbnormalCaseModalOpen}
        onFinish={async (values) => {
          if (!abnormalCaseOrder?.id) return false;
          await orderAbnormalCaseServiceMarkAbnormalCase(
            {
              orderId: abnormalCaseOrder.id,
            },
            {
              orderId: abnormalCaseOrder.id,
              abnormalCaseId: values.abnormalCaseId,
            },
          );
          message.success('标记异常成功');
          setAbnormalCaseModalOpen(false);
          abnormalCaseActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="abnormalCaseId"
          label="异常类型"
          rules={[{ required: true, message: '请选择异常类型' }]}
          options={abnormalCaseOptions}
          placeholder="请选择异常类型"
        />
      </ModalForm>
    </>
  );
}
