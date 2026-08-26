import type { ProColumns } from '@ant-design/pro-components';

export type OrderKind = 'sea-export' | 'sea-import' | 'air-export' | 'air-import' | 'rail' | 'truck' | 'customs';

/** 筛选面板字段参数定义 */
export interface OrderListFilterParams {
  // 时间与节点类
  createdAtRange?: [string, string];
  etaRange?: [string, string];
  etdRange?: [string, string];
  lockedAtRange?: [string, string];
  statusTimeRange?: [string, string];

  // 单号与业务实体类
  keyword?: string; // 订单/主单号/加拼主单号
  shippingLineId?: string; // 船公司/航司
  originLocationId?: string; // 起运港
  destinationLocationId?: string; // 目的港
  customerId?: string; // 委托单位
  consignee?: string; // 收货人简称
  shipper?: string; // 发货人简称

  // 人员与组织架构类
  operatorId?: string; // 操作人员
  operatorDeptId?: string;
  salesId?: string; // 业务人员
  salesDeptId?: string;
  customerServiceId?: string; // 客服人员
  customerServiceDeptId?: string;
  creatorId?: string; // 订单创建人员
  creatorDeptId?: string;

  // 状态与标记类
  stage?: string; // 进程（未退关、已完结等）
  shareStatus?: 'all' | 'shared' | 'unshared'; // 分享状态
  isLocked?: 'all' | 'locked' | 'unlocked'; // 是否锁定
  tags?: string[]; // 订单标签

  // 分页与排序
  page?: number;
  pageSize?: number;
  sorterField?: string;
  sorterOrder?: 'ascend' | 'descend';
}

/** 订单列表行完整业务模型 */
export interface OrderListItem {
  id: string;
  orderNo: string;
  orderKind?: OrderKind;
  businessType?: string | number;
  stage?: string; // 进程
  customerName?: string;
  customerId?: string;
  customerReferenceNo?: string;
  createdAt?: string;

  // 航程与船运
  bookingAgentName?: string;
  masterBlNo?: string;
  vesselVoyage?: string;
  originPortCode?: string;
  originPortName?: string;
  destinationPortCode?: string;
  destinationPortName?: string;
  finalDestination?: string;
  containerSummary?: string; // 如 "2×20GP, 1×40HQ"
  containerNos?: string[];

  // 干系人
  operatorName?: string;
  operatorBranch?: string;
  salesName?: string;
  salesBranch?: string;
  creatorName?: string;
  creatorOrg?: string;

  // 货物与计量
  totalPackages?: number;
  packageUnit?: string;
  grossWeightKg?: number;
  volumeCbm?: number;

  // 单证与商务
  paymentTerm?: string | number;
  tradeTerm?: string | number;
  contractNo?: string;
  consigneeName?: string;
  shipperName?: string;

  // 备注与控制
  spaceNotes?: string;
  bookingNotes?: string;
  operationNotes?: string;
  attachmentCount?: number;
  lockedAt?: string;
  isLocked?: boolean;

  // 状态与异常
  status?: string;
  statusCode?: number | string;
  statusName?: string;
  abnormalLevel?: 'normal' | 'low' | 'medium' | 'high';
  abnormalName?: string;

  [key: string]: any;
}

/** 状态切签项配置 */
export interface OrderStatusTabItem {
  key: string;
  label: string;
  count?: number;
  badgeColor?: string;
}

/** 批量操作菜单枚举/动作 */
export type BatchActionKey =
  | 'export-documents'
  | 'batch-collab'
  | 'cancel-collab'
  | 'share-orders'
  | 'finish-orders'
  | 'cancel-finish'
  | 'surrender'
  | 'cancel-surrender'
  | 'archive'
  | 'modify-lock-time'
  | 'lock-unlock'
  | 'interest-to-receivable'
  | 'batch-modify-fields'
  | 'manage-tags'
  | 'batch-import'
  | 'batch-delete';

/** OrderListTemplate 组件属性定义 */
export interface OrderListTemplateProps {
  /** 品类标识（如 'sea-export' | 'sea-import' | 'air-export' | 'air-import'） */
  orderKind: OrderKind;
  /** 页面/工作台主标题，如 "海运出口订单" */
  title?: string;
  /** 页面副标题 */
  subTitle?: string;
  /** 状态切签列表（如 全部、待订舱、已配载、在途、已放行、已完成、异常等） */
  statusTabs?: OrderStatusTabItem[];
  /** 当前激活的状态切签 key */
  activeStatusTab?: string;
  /** 切换状态切签回调 */
  onStatusTabChange?: (statusKey: string) => void;

  /** 自定义或扩展 ProTable 表格列 */
  customColumns?: ProColumns<OrderListItem>[];
  /** 扩展列：插入在货物与单证之间的品类专属列（如海运截关时间、空运起飞时间等） */
  extraColumns?: ProColumns<OrderListItem>[];

  /** 数据查询请求方法 */
  queryOrders: (params: OrderListFilterParams) => Promise<{
    data: OrderListItem[];
    total?: number;
    success?: boolean;
  }>;

  /** 主按钮：点击新建订单 */
  onCreateOrder?: () => void;
  /** 复制选中订单 */
  onCopyOrder?: (selectedRows: OrderListItem[]) => void;
  /** 批量导出单证 */
  onExportDocuments?: (selectedRows: OrderListItem[]) => void;
  /** 批量操作触发 */
  onBatchAction?: (actionKey: BatchActionKey, selectedRows: OrderListItem[]) => void;
  /** 点击查看订单详情 */
  onViewDetail?: (record: OrderListItem) => void;
  /** 点击编辑订单 */
  onEditOrder?: (record: OrderListItem) => void;
  /** 打开费用结算面板 */
  onOpenFees?: (record: OrderListItem) => void;
  /** 打开里程碑履约面板 */
  onOpenMilestones?: (record: OrderListItem) => void;
  /** 打开单据/提单维护面板 */
  onOpenDocuments?: (record: OrderListItem) => void;
  /** 打开集装箱管理面板 */
  onOpenContainers?: (record: OrderListItem) => void;
  /** 打开异常情况处理面板 */
  onOpenAbnormal?: (record: OrderListItem) => void;
  /** 状态流转操作 */
  onTransitionStatus?: (record: OrderListItem) => void;

  /** 基础主数据选项（用于筛选器下拉） */
  options?: {
    ports?: { label: string; value: string }[];
    airports?: { label: string; value: string }[];
    shippingLines?: { label: string; value: string }[];
    airlines?: { label: string; value: string }[];
    partners?: { label: string; value: string }[];
    users?: { label: string; value: string }[];
    departments?: { label: string; value: string }[];
  };

  /** 是否只读（无编辑/操作权限） */
  readonly?: boolean;
}
