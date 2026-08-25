import {
  ContainerOutlined,
  DollarOutlined,
  DownOutlined,
  EditOutlined,
  FileDoneOutlined,
  FileTextOutlined,
  FlagOutlined,
  InboxOutlined,
  OrderedListOutlined,
  PaperClipOutlined,
  PlusOutlined,
  ReloadOutlined,
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
  PageContainer,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { history, useAccess, useLocation } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Drawer,
  Dropdown,
  Popconfirm,
  Result,
  Space,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
  masterDataServiceListStatusTemplates,
} from '@/services/roncin/masterDataService';
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
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  MASTER_DATA_KINDS,
  businessTypeMap,
  businessTypeValueEnum,
  isMasterDataKind,
  orderPersonnelRoleOptions,
  orderPersonnelRoleValueEnum,
  parseOrderKind,
  paymentTermOptions,
  shipmentTypeOptions,
  shippingDocumentStatusValueEnum,
  tradeDirectionOptions,
  tradeDirectionValueEnum,
  tradeTermOptions,
} from './common';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import {
  formatHouseReleaseType,
  OrderContainerRequestFields,
  OrderShippingDocumentFields,
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
} from './order-plan-fields';

const { Text } = Typography;

type EditOrderFormValues = {
  customerId: string;
  tradeDirection: number;
  tradeTerm: number;
  paymentTerm: number;
  shipmentType?: number;
  serviceTypeIds?: string[];
  cargoCategoryIds?: string[];
  originLocationId?: string;
  destinationLocationId?: string;
  vesselVoyage?: string;
  etd?: string | dayjs.Dayjs;
  eta?: string | dayjs.Dayjs;
  goodsDescription?: string;
  totalPackages?: number;
  totalPackageUnit?: string;
  notes?: string;
  shippingDocuments?: API.OrderShippingDocumentInput[];
  containerRequests?: API.OrderContainerRequestInput[];
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

type PersonnelFormValues = {
  userId: string;
  organizationId: string;
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

export default function OrderListPage() {
  const location = useLocation();
  const config = parseOrderKind(location.pathname);

  const actionRef = useRef<ActionType | undefined>(undefined);
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
  const shippingDocumentFormRef = useRef<ProFormInstance | undefined>(
    undefined,
  );
  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);
  const { message } = App.useApp();
  const access = useAccess();

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
  const [editingMilestone, setEditingMilestone] =
    useState<API.OrderMilestone>();
  const [editingContainer, setEditingContainer] =
    useState<API.OrderContainer>();
  const [editingCargoItem, setEditingCargoItem] =
    useState<API.OrderCargoItem>();
  const [editingShippingDocument, setEditingShippingDocument] =
    useState<API.OrderShippingDocument>();
  const [targetStatusOptions, setTargetStatusOptions] = useState<
    { label: string; value: string }[]
  >([]);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});

  useEffect(() => {
    void Promise.all([
      masterDataServiceListOptions(),
      masterDataServiceListPorts({ page: 1, pageSize: 100 }),
      masterDataServiceListAirports({ page: 1, pageSize: 100 }),
    ])
      .then(([optionsResponse, portsResponse, airportsResponse]) => {
        setMasterOptions(optionsResponse.data ?? []);
        setPorts(portsResponse.data ?? []);
        setAirports(airportsResponse.data ?? []);
      })
      .catch((error: Error) =>
        message.error(error.message || '订单主数据加载失败'),
      );
  }, [message]);

  const containerSpecOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const containerSpecMap = Object.fromEntries(
    masterOptions
      .filter(
        (item) =>
          isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
          item.id,
      )
      .map((item) => [
        item.id as string,
        item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      ]),
  );

  const serviceTypeOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const cargoCategoryOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CARGO_CATEGORY) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const regionLocationOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.REGION) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const locationOptions = [
    ...regionLocationOptions,
    ...ports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
        value: item.id ?? '',
      })),
    ...airports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
        value: item.id ?? '',
      })),
  ];

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

  const openEdit = (record: API.Order) => {
    setEditingRecord(record);
    editFormRef.current?.setFieldsValue({
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
      totalPackageUnit: record.totalPackageUnit,
      notes: record.notes,
      shippingDocuments: record.shippingDocuments,
      containerRequests: record.containerRequests,
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

  const activeContainerOrderIdRef = useRef<string | undefined>(undefined);

  const openContainers = (record: API.Order) => {
    setContainerOrder(record);
    setContainerDocuments([]);
    setContainerDrawerOpen(true);
    const orderId = record.id as string;
    activeContainerOrderIdRef.current = orderId;
    orderShippingDocumentServiceListShippingDocuments({
      orderId,
    })
      .then((res) => {
        if (activeContainerOrderIdRef.current === orderId) {
          setContainerDocuments(res.data ?? []);
        }
      })
      .catch(() => {
        if (activeContainerOrderIdRef.current === orderId) {
          setContainerDocuments([]);
        }
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

  if (!config) {
    return (
      <PageContainer>
        <Result
          status="404"
          title="未知的业务类型"
          subTitle="请求的订单列表业务类型无效，请通过系统菜单导航进入。"
          extra={
            <Button
              type="primary"
              onClick={() => history.push('/orders/sea-export')}
            >
              前往海运出口订单
            </Button>
          }
        />
      </PageContainer>
    );
  }

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
      title: '分单签放方式',
      dataIndex: 'releaseType',
      width: 140,
      render: (_, record) =>
        record.releaseType ? (
          <Tag color="geekblue" bordered={false}>
            {formatHouseReleaseType(record.releaseType)}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: '单证状态',
      dataIndex: 'status',
      width: 120,
      valueType: 'select',
      valueEnum: shippingDocumentStatusValueEnum,
      render: (_, record) =>
        record.status !== undefined &&
        shippingDocumentStatusValueEnum[record.status] ? (
          <Tag
            color={
              record.status === 3
                ? 'success'
                : record.status === 2
                  ? 'processing'
                  : 'default'
            }
            bordered={false}
          >
            {shippingDocumentStatusValueEnum[record.status]?.text}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: '备注说明',
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
      width: 160,
      render: (_, record) => {
        if (!access.canOrder(config.businessType, 'shipping_document.update')) return null;
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => openEditShippingDocument(record)}
            >
              编辑
            </Button>
            {record.status === 1 && (
              <Popconfirm
                title="确认将该提单状态流转为【已确认】？"
                onConfirm={async () => {
                  if (!shippingDocumentOrder?.id || !record.id) return;
                  await orderShippingDocumentServiceTransitionShippingDocumentStatus(
                    {
                      orderId: shippingDocumentOrder.id,
                      id: record.id,
                    },
                    {
                      orderId: shippingDocumentOrder.id,
                      id: record.id,
                      expectedStatus: 1,
                      toStatus: 2,
                    },
                  );
                  message.success('提单已确认');
                  shippingDocumentActionRef.current?.reload();
                }}
              >
                <Button type="link" size="small">
                  确认
                </Button>
              </Popconfirm>
            )}
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
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>
          {record.containerNo || '-'}
        </Text>
      ),
    },
    {
      title: '集装箱规格',
      dataIndex: 'containerSpecId',
      width: 160,
      render: (_, record) =>
        record.containerSpecId
          ? containerSpecMap[record.containerSpecId] || record.containerSpecId
          : '-',
    },
    {
      title: '关联提单',
      dataIndex: 'shippingDocumentId',
      width: 180,
      ellipsis: true,
      render: (_, record) =>
        record.shippingDocumentId
          ? containerDocumentMap[record.shippingDocumentId] ||
            record.shippingDocumentId
          : <Text type="secondary">未关联</Text>,
    },
    {
      title: '铅封号',
      dataIndex: 'sealNo',
      copyable: true,
      ellipsis: true,
      render: (_, record) => record.sealNo || '-',
    },
    {
      title: '毛重(KG)',
      dataIndex: 'grossWeightKg',
      width: 120,
      render: (_, record) => record.grossWeightKg ?? '-',
    },
    {
      title: '体积(CBM)',
      dataIndex: 'volumeCbm',
      width: 120,
      render: (_, record) => record.volumeCbm ?? '-',
    },
    {
      title: '备注说明',
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
        if (!access.canOrder(config.businessType, 'container.update')) return null;
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
      title: '货物名称',
      dataIndex: 'cargoName',
      ellipsis: true,
      render: (_, record) => <Text strong>{record.cargoName || '-'}</Text>,
    },
    {
      title: '件数',
      dataIndex: 'packageCount',
      width: 100,
      render: (_, record) => record.packageCount ?? '-',
    },
    {
      title: '毛重(KG)',
      dataIndex: 'grossWeightKg',
      width: 120,
      render: (_, record) => record.grossWeightKg ?? '-',
    },
    {
      title: '体积(CBM)',
      dataIndex: 'volumeCbm',
      width: 120,
      render: (_, record) => record.volumeCbm ?? '-',
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
        if (!access.canOrder(config.businessType, 'cargo_item.update')) return null;
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
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.userId || '-'}
        </Text>
      ),
    },
    {
      title: '协作角色',
      dataIndex: 'role',
      valueType: 'select',
      valueEnum: orderPersonnelRoleValueEnum,
      render: (_, record) =>
        record.role !== undefined && orderPersonnelRoleValueEnum[record.role] ? (
          <Tag color="blue" bordered={false}>
            {orderPersonnelRoleValueEnum[record.role]?.text}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: '指派时间',
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
        if (!access.canOrder(config.businessType, 'personnel.remove')) return null;
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
      render: (type) => (
        <Tag color="geekblue" bordered={false}>
          {type}
        </Tag>
      ),
    },
    {
      title: '文件名',
      dataIndex: 'fileName',
      ellipsis: true,
      render: (name) => <Text strong>{name}</Text>,
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
      render: (size) => `${size} 字节`,
    },
    {
      title: '对象键',
      dataIndex: 'objectKey',
      copyable: true,
      ellipsis: true,
      render: (key) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {key}
        </Text>
      ),
    },
    {
      title: '校验和',
      dataIndex: 'checksum',
      copyable: true,
      ellipsis: true,
    },
    {
      title: '登记时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
    },
  ];

  const milestoneColumns: ProColumns<API.OrderMilestone>[] = [
    {
      title: '里程碑类型',
      dataIndex: 'type',
      width: 160,
      render: (_, record) => (
        <Tag color="blue" bordered={false}>
          {record.type || '-'}
        </Tag>
      ),
    },
    {
      title: '节点编码',
      dataIndex: 'templateNodeCode',
      width: 150,
      render: (_, record) =>
        record.templateNodeCode ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.templateNodeCode}
          </Text>
        ) : (
          '-'
        ),
    },
    {
      title: '节点名称',
      dataIndex: 'templateNodeLabel',
      width: 150,
      render: (_, record) => record.templateNodeLabel || '-',
    },
    {
      title: '发生时间',
      dataIndex: 'occurredAt',
      valueType: 'dateTime',
      width: 180,
      render: (_, record) =>
        record.occurredAt ? (
          dayjs(record.occurredAt).format('YYYY-MM-DD HH:mm:ss')
        ) : (
          <Text type="secondary">未完成</Text>
        ),
    },
    {
      title: '备注说明',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      render: (_, record) => {
        if (!access.canOrder(config.businessType, 'milestone.set')) return null;
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
      title: '订单编号',
      dataIndex: 'orderNo',
      copyable: true,
      search: false,
      width: 170,
      render: (_, record) => (
        <Text
          style={{
            fontFamily: 'monospace',
            fontWeight: 600,
            color: '#0f172a',
          }}
        >
          {record.orderNo || record.id}
        </Text>
      ),
    },
    {
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 160,
      search: false,
      render: (_, record) => (
        <Text>{record.organizationName || record.organizationId || '-'}</Text>
      ),
    },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      width: 130,
      search: false,
      valueType: 'select',
      valueEnum: businessTypeValueEnum,
      render: (_, record) => {
        const item = businessTypeMap.get(record.businessType ?? 0);
        return item ? (
          <Tag color={item.color} bordered={false}>
            {item.label}
          </Tag>
        ) : (
          '-'
        );
      },
    },
    {
      title: '客户单位',
      dataIndex: 'customerId',
      width: 200,
      ellipsis: true,
      valueType: 'select',
      fieldProps: {
        showSearch: true,
        placeholder: '搜索客户',
      },
      request: async ({ keyWords }) => searchCustomers(keyWords),
      render: (_, record) => (
        <Text strong>
          {customerMap[record.customerId ?? ''] || record.customerId || '-'}
        </Text>
      ),
    },
    {
      title: '当前状态',
      dataIndex: 'status',
      width: 120,
      valueType: 'text',
      render: (_, record) => {
        const isDraft = record.status === 'DRAFT';
        return (
          <Tag
            color={isDraft ? 'default' : 'processing'}
            bordered={false}
            style={{ fontWeight: 500 }}
          >
            {record.status || '-'}
          </Tag>
        );
      },
    },
    {
      title: '贸易方向',
      dataIndex: 'tradeDirection',
      width: 100,
      search: false,
      valueType: 'select',
      valueEnum: tradeDirectionValueEnum,
      render: (_, record) => (
        <Tag bordered={false}>
          {tradeDirectionValueEnum[record.tradeDirection ?? 0]?.text || '-'}
        </Tag>
      ),
    },
    {
      title: '预计离港 (ETD)',
      dataIndex: 'etd',
      width: 130,
      search: false,
      render: (_, record) =>
        record.etd ? (
          dayjs(record.etd).format('YYYY-MM-DD')
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '预计到港 (ETA)',
      dataIndex: 'eta',
      width: 130,
      search: false,
      render: (_, record) =>
        record.eta ? (
          dayjs(record.eta).format('YYYY-MM-DD')
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      key: 'option',
      width: 390,
      fixed: 'right',
      render: (_, record) => {
        if (!access.canOrder(config.businessType, 'read')) return null;

        const fulfillmentMenu = {
          items: [
            {
              key: 'milestone',
              label: '履约里程碑',
              icon: <FlagOutlined />,
              onClick: () => openMilestones(record),
            },
            {
              key: 'pod',
              label: '放货凭证 (POD)',
              icon: <FileDoneOutlined />,
              onClick: () => releasePodPanelRef.current?.open(record),
            },
            {
              key: 'abnormal',
              label: '异常标记与协同',
              icon: <WarningOutlined />,
              onClick: () => abnormalCasePanelRef.current?.open(record),
            },
          ],
        };

        const documentCargoMenu = {
          items: [
            {
              key: 'shippingDoc',
              label: '提单管理',
              icon: <FileTextOutlined />,
              onClick: () => openShippingDocuments(record),
            },
            {
              key: 'containers',
              label: '集装箱管理',
              icon: <ContainerOutlined />,
              onClick: () => openContainers(record),
            },
            {
              key: 'cargo',
              label: '货物明细',
              icon: <InboxOutlined />,
              onClick: () => openCargoItems(record),
            },
            {
              key: 'attachments',
              label: '附件档案',
              icon: <PaperClipOutlined />,
              onClick: () => openAttachments(record),
            },
            {
              key: 'personnel',
              label: '协作人员',
              icon: <TeamOutlined />,
              onClick: () => openPersonnel(record),
            },
          ],
        };

        return (
          <Space size={8}>
            {access.canOrder(config.businessType, 'update') && record.canModify && record.status === 'DRAFT' && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                style={{ padding: 0 }}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
            {access.canOrder(config.businessType, 'transition') && record.canModify && (
              <Button
                type="link"
                size="small"
                icon={<SwapOutlined />}
                style={{ padding: 0 }}
                onClick={() => openTransition(record)}
              >
                流转
              </Button>
            )}
            {access.canOrder(config.businessType, 'fee.read') && (
              <Button
                type="link"
                size="small"
                icon={<DollarOutlined />}
                style={{ padding: 0 }}
                onClick={() => orderFeePanelRef.current?.open(record)}
              >
                费用录入
              </Button>
            )}
            <Dropdown menu={fulfillmentMenu}>
              <Button type="link" size="small" style={{ padding: 0 }}>
                履约协同 <DownOutlined style={{ fontSize: 10 }} />
              </Button>
            </Dropdown>
            <Dropdown menu={documentCargoMenu}>
              <Button type="link" size="small" style={{ padding: 0 }}>
                单证箱货 <DownOutlined style={{ fontSize: 10 }} />
              </Button>
            </Dropdown>
          </Space>
        );
      },
    },
  ];

  return (
    <PageContainer title={config.title}>
      <ProTable<API.Order>
        headerTitle={
          <Space size={8}>
            <OrderedListOutlined style={{ color: '#1677ff' }} />
            <span>{config.title}列表</span>
          </Space>
        }
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        scroll={{ x: 1300 }}
        request={async (params) => {
          const response = await orderServiceListOrders({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            status: params.status,
            businessType: config.businessType,
            customerId: params.customerId,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? false,
            total: response.total ?? 0,
          };
        }}
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => actionRef.current?.reload()}
          >
            刷新
          </Button>,
          access.canOrder(config.businessType, 'create') && (
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => history.push(`/orders/${config.kind}/new`)}
            >
              新增订单
            </Button>
          ),
        ]}
      />

      <ModalForm<EditOrderFormValues>
        title="编辑订单草稿"
        open={editModalOpen}
        formRef={editFormRef}
        grid
        initialValues={
          editingRecord
            ? {
                customerId: editingRecord.customerId,
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
                shippingDocuments: editingRecord.shippingDocuments,
                containerRequests: editingRecord.containerRequests,
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 820,
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
              shippingDocuments: values.shippingDocuments,
              containerRequests: values.containerRequests,
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
          label="客户单位"
          rules={[{ required: true, message: '请选择客户' }]}
          fieldProps={{
            showSearch: true,
            placeholder: '搜索客户',
          }}
          request={async ({ keyWords }) => searchCustomers(keyWords)}
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
          label="运费条款"
          rules={[{ required: true, message: '请选择付款条款' }]}
          options={paymentTermOptions}
          placeholder="请选择付款条款"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="shipmentType"
          label="装载方式"
          options={shipmentTypeOptions}
          placeholder="请选择装载类型"
        />
        <OrderShippingDocumentFields
          transportMode={config.category === 'air' ? 'air' : 'sea'}
        />
        {config.category === 'sea' && (
          <OrderContainerRequestFields options={containerSpecOptions} />
        )}
        <ProFormSelect
          colProps={{ span: 12 }}
          name="originLocationId"
          label="起运港 / 地点"
          options={locationOptions}
          fieldProps={{ showSearch: true }}
          placeholder="请选择起运地点"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="destinationLocationId"
          label="目的港 / 地点"
          options={locationOptions}
          fieldProps={{ showSearch: true }}
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
          label="总件数"
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
          label="业务备注"
          placeholder="请输入备注说明"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
      </ModalForm>

      <ModalForm<TransitionFormValues>
        title="订单状态流转"
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
          label="目标流转状态"
          rules={[{ required: true, message: '请选择目标状态' }]}
          options={targetStatusOptions}
          placeholder="请选择目标状态"
        />
        <ProFormTextArea
          name="reason"
          label="流转原因说明"
          placeholder="请输入状态变更原因说明（可选）"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          milestoneOrder
            ? `订单履约里程碑 - ${milestoneOrder.orderNo || milestoneOrder.id}`
            : '订单履约里程碑'
        }
        open={milestoneDrawerOpen}
        onClose={() => {
          setMilestoneDrawerOpen(false);
          setMilestoneOrder(undefined);
        }}
        width={860}
        destroyOnHidden
      >
        {milestoneOrder?.id && (
          <ProTable<API.OrderMilestone>
            actionRef={milestoneActionRef}
            rowKey={(record) => record.id || record.type || ''}
            columns={milestoneColumns}
            bordered
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
              access.canOrder(config.businessType, 'milestone.set') && (
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
        <ProFormSwitch name="clearOccurredAt" label="清除发生时间" />
        <ProFormTextArea
          name="note"
          label="备注说明"
          placeholder="请输入备注"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          attachmentOrder
            ? `订单附件档案 - ${attachmentOrder.orderNo || attachmentOrder.id}`
            : '订单附件档案'
        }
        open={attachmentDrawerOpen}
        onClose={() => {
          setAttachmentDrawerOpen(false);
          setAttachmentOrder(undefined);
        }}
        width={920}
        destroyOnHidden
      >
        {attachmentOrder?.id && (
          <ProTable<API.OrderAttachment>
            actionRef={attachmentActionRef}
            rowKey={(record) => record.id || record.objectKey || ''}
            columns={attachmentColumns}
            bordered
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
              access.canOrder(config.businessType, 'attachment.register') && (
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
        title="登记订单附件"
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
          message="此处登记外部对象存储引用与元数据，不直接进行二进制文件上传。"
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
          label="文件大小 (字节)"
          min={1}
          fieldProps={{ precision: 0 }}
          placeholder="请输入文件大小"
          rules={[{ required: true, message: '请输入文件大小' }]}
        />
        <ProFormText
          name="objectKey"
          label="对象存储键 (Object Key)"
          placeholder="请输入对象存储键"
          rules={[{ required: true, message: '请输入对象键' }]}
        />
        <ProFormText
          name="checksum"
          label="校验和 (Checksum)"
          placeholder="请输入校验和 (可选)"
        />
      </ModalForm>

      <Drawer
        title={
          personnelOrder
            ? `订单协作团队 - ${personnelOrder.orderNo || personnelOrder.id}`
            : '订单协作团队'
        }
        open={personnelDrawerOpen}
        onClose={() => {
          setPersonnelDrawerOpen(false);
          setPersonnelOrder(undefined);
        }}
        width={820}
        destroyOnHidden
      >
        {personnelOrder?.id && (
          <ProTable<API.OrderPersonnel>
            actionRef={personnelActionRef}
            rowKey={(record) => record.id || `${record.userId}-${record.role}`}
            columns={personnelColumns}
            bordered
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
              access.canOrder(config.businessType, 'personnel.assign') && (
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openAssignPersonnel}
                >
                  分配协作人员
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
              organizationId: values.organizationId.trim(),
              role: Number(values.role),
            },
          );
          message.success('分配协作人员成功');
          setPersonnelModalOpen(false);
          personnelActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="userId"
          label="用户 UUID"
          placeholder="请输入组织内用户 UUID"
          rules={[{ required: true, message: '请输入用户 UUID' }]}
        />
        <ProFormText
          name="organizationId"
          label="所属公司 UUID"
          placeholder="请输入人员所属公司 UUID"
          rules={[{ required: true, message: '请输入人员所属公司 UUID' }]}
        />
        <ProFormSelect
          name="role"
          label="担当角色"
          rules={[{ required: true, message: '请选择角色' }]}
          options={orderPersonnelRoleOptions}
          placeholder="请选择角色"
        />
      </ModalForm>

      <Drawer
        title={
          containerOrder
            ? `订单集装箱列表 - ${containerOrder.orderNo || containerOrder.id}`
            : '订单集装箱列表'
        }
        open={containerDrawerOpen}
        onClose={() => {
          activeContainerOrderIdRef.current = undefined;
          setContainerDrawerOpen(false);
          setContainerOrder(undefined);
          setContainerDocuments([]);
        }}
        width={920}
        destroyOnHidden
      >
        {containerOrder?.id && (
          <ProTable<API.OrderContainer>
            actionRef={containerActionRef}
            rowKey="id"
            columns={containerColumns}
            bordered
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
              access.canOrder(config.businessType, 'container.create') && (
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
          placeholder="请输入箱号 (如 COSU1234567)"
          rules={[{ required: true, message: '请输入箱号' }]}
        />
        <ProFormSelect
          name="containerSpecId"
          label="集装箱规格"
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
          label="铅封号"
          placeholder="请输入铅封号 (可选)"
        />
        <ProFormDigit
          name="grossWeightKg"
          label="货物毛重 (KG)"
          min={0.001}
          placeholder="请输入毛重"
          rules={[{ required: true, message: '请输入毛重' }]}
        />
        <ProFormDigit
          name="volumeCbm"
          label="货物体积 (CBM)"
          min={0.001}
          placeholder="请输入体积"
          rules={[{ required: true, message: '请输入体积' }]}
        />
        <ProFormTextArea
          name="note"
          label="备注说明"
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
        width={920}
        destroyOnHidden
      >
        {cargoOrder?.id && (
          <ProTable<API.OrderCargoItem>
            actionRef={cargoActionRef}
            rowKey="id"
            columns={cargoColumns}
            bordered
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
              access.canOrder(config.businessType, 'cargo_item.create') && (
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
                  values.netWeightKg !== undefined &&
                  values.netWeightKg !== null
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
                  values.netWeightKg !== undefined &&
                  values.netWeightKg !== null
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
          label="货物名称"
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
          label="毛重 (KG)"
          min={0.001}
          placeholder="请输入毛重"
          rules={[{ required: true, message: '请输入毛重' }]}
        />
        <ProFormDigit
          name="volumeCbm"
          label="体积 (CBM)"
          min={0.001}
          placeholder="请输入体积"
          rules={[{ required: true, message: '请输入体积' }]}
        />
        <ProFormDigit
          name="netWeightKg"
          label="净重 (KG)"
          min={0.001}
          placeholder="请输入净重 (可选)"
        />
        <ProFormTextArea
          name="note"
          label="备注说明"
          placeholder="请输入备注 (可选)"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <Drawer
        title={
          shippingDocumentOrder
            ? `订单提单与放货 - ${shippingDocumentOrder.orderNo || shippingDocumentOrder.id}`
            : '订单提单与放货'
        }
        open={shippingDocumentDrawerOpen}
        onClose={() => {
          setShippingDocumentDrawerOpen(false);
          setShippingDocumentOrder(undefined);
        }}
        width={920}
        destroyOnHidden
      >
        {shippingDocumentOrder?.id && (
          <ProTable<API.OrderShippingDocument>
            actionRef={shippingDocumentActionRef}
            rowKey="id"
            columns={shippingDocumentColumns}
            bordered
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
              access.canOrder(config.businessType, 'shipping_document.create') && (
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
          label="主单号 (MBL)"
          placeholder="请输入主单号"
          rules={[{ required: true, message: '请输入主单号' }]}
        />
        <ProFormText
          name="houseNo"
          label="分单号 (HBL)"
          placeholder="请输入分单号"
          rules={[{ required: true, message: '请输入分单号' }]}
        />
        {config.category === 'sea' ? (
          <ProFormSelect
            name="releaseType"
            label="分单签放方式"
            options={SEA_HOUSE_RELEASE_TYPE_OPTIONS}
            placeholder="请选择分单签放方式"
          />
        ) : (
          <ProFormText
            name="releaseType"
            label="分单签放方式"
            placeholder="请输入分单签放方式"
          />
        )}
        <ProFormTextArea
          name="note"
          label="备注说明"
          placeholder="请输入备注 (可选)"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <ReleasePodPanel
        ref={releasePodPanelRef}
        canManage={access.canOrder(config.businessType, 'release_pod.create')}
      />
      <OrderFeePanel ref={orderFeePanelRef} />
      <AbnormalCasePanel
        ref={abnormalCasePanelRef}
        canManage={access.canOrder(config.businessType, 'abnormal_case.create')}
        masterOptions={masterOptions}
      />
    </PageContainer>
  );
}
