import type { ReactNode } from 'react';

export type OrderDetailKind =
  | 'sea-export'
  | 'sea-import'
  | 'air-export'
  | 'air-import'
  | 'rail'
  | 'truck'
  | 'customs';

/** 流程节点展示数据 */
export interface ProcessNodeItem {
  key: string;
  title: string;
  description?: string;
  status: 'wait' | 'process' | 'finish' | 'error';
  occurredAt?: string;
}

/** 操作记录日志项 */
export interface OperationLogItem {
  id?: string;
  action: string;
  operatorName?: string;
  operatorUserId?: string;
  occurredAt?: string;
  detail?: string;
}

/** 订单详情聚合数据接口 - 严格对齐 6 大业务模块 */
export interface OrderDetailData {
  // 基础标识
  id?: string;
  orderNo?: string;
  businessTypeTitle?: string;
  status?: string;
  isLocked?: boolean;
  canModify?: boolean;
  createdAt?: string;
  updatedAt?: string;

  // 模块 1：订单状态（流程节点展示）
  currentStepIndex?: number;
  processNodes?: ProcessNodeItem[];

  // 模块 2：业务信息
  customerName?: string;
  customerId?: string;
  customerReferenceNo?: string;
  internalReferenceNo?: string;
  shipmentTypeName?: string; // 集运/整箱/拼箱/散货
  shipmentModeName?: string; // 跨境/国内/加拼
  serviceTypeNames?: string[]; // 订舱/拖车/报关等
  cargoCategoryNames?: string[]; // 货物品类
  tradeTermName?: string; // 贸易条款 FOB/CIF/DDP
  carrierName?: string; // 船公司/航司
  shippingAgentName?: string; // 船代
  bookingAgentName?: string; // 订舱代理
  contractNo?: string; // 合约号
  cargoValueWithCurrency?: string; // 货物申报价值
  insurancePremiumWithCurrency?: string; // 保险保费

  // 模块 3：配舱信息
  masterBlNo?: string; // 主单号
  houseBlNo?: string; // 分单号
  containerSummary?: string; // 箱型箱量 如 40HQ*1, 20GP*2
  originName?: string; // 起运港
  originCode?: string;
  destinationName?: string; // 目的港
  destinationCode?: string;
  dischargeName?: string; // 卸货港
  transitName?: string; // 中转港
  vesselVoyage?: string; // 船名航次 / 航班号
  etd?: string; // 预计离港
  eta?: string; // 预计到港
  siCutoff?: string; // 截补料
  docCutoff?: string; // 截单
  customsCutoff?: string; // 截关
  vgmCutoff?: string; // 截VGM

  // 模块 4：提单信息
  shipperName?: string; // 发货人 (Shipper)
  consigneeName?: string; // 收货人 (Consignee)
  notifyPartyName?: string; // 通知人 (Notify Party)
  foreignAgentName?: string; // 国外代理
  shippingMarks?: string; // 唛头 (Marks & Numbers)
  goodsDescription?: string; // 品名/描述
  goodsEnglishDescription?: string; // 英文品名
  totalPackages?: number; // 件数
  packageUnit?: string; // 包装单位
  grossWeightKg?: number; // 毛重 (KGS)
  volumeCbm?: number; // 体积 (CBM)
  netWeightKg?: number; // 净重 (KGS)
  actualPackages?: number; // 实际件数
  actualGrossWeightKg?: number; // 实际毛重
  actualVolumeCbm?: number; // 实际体积
  paymentTermName?: string; // 付款方式 PP/CC
  transportTerms?: string; // 运输条款 如 CY-CY, CFS-CFS
  loadingTerms?: string; // 装货条款
  blTypeName?: string; // 提单形式 如 正本/电放/SeaWay

  // 模块 5：3 个备注
  bookingNotes?: string; // 订舱备注
  allocationNotes?: string; // 配舱备注
  operationNotes?: string; // 操作备注
  notes?: string; // 综合备注

  // 模块 6：内部信息与操作记录
  // 6.1 内部人员配置
  creatorName?: string; // 创建人
  operatorName?: string; // 操作人
  salesName?: string; // 业务人/销售
  customerServiceName?: string; // 客服人
  documentOperatorName?: string; // 单证人
  commercialOperatorName?: string; // 商务人
  associateNames?: string[]; // 关联人员
  belongOrganizationName?: string; // 所属组织/分公司
  personnel?: Array<{
    id?: string;
    roleName?: string;
    userId?: string;
    userName?: string;
  }>;

  // 6.2 操作记录日志
  operationLogs?: OperationLogItem[];

  // 扩展子明细表格
  shippingDocuments?: Array<{
    id?: string;
    masterNo?: string;
    houseNo?: string;
    masterDocumentType?: string;
    masterReleaseMethod?: string;
    releaseType?: string;
    status?: string;
    createdAt?: string;
  }>;

  containers?: Array<{
    id?: string;
    containerNo?: string;
    sealNo?: string;
    containerSpecName?: string;
    grossWeightKg?: number;
    volumeCbm?: number;
    note?: string;
  }>;

  cargoItems?: Array<{
    id?: string;
    cargoName?: string;
    packageCount?: number;
    grossWeightKg?: number;
    volumeCbm?: number;
    netWeightKg?: number;
    note?: string;
  }>;

  milestones?: Array<{
    id?: string;
    type?: string;
    occurredAt?: string;
    confirmedAt?: string;
    note?: string;
  }>;

  attachments?: Array<{
    id?: string;
    docType?: string;
    fileName?: string;
    fileSize?: string | number;
    mimeType?: string;
    createdAt?: string;
  }>;
}

export interface OrderDetailTemplateProps {
  /** 订单品类标识（如 'sea-export'） */
  orderKind?: OrderDetailKind;
  /** 业务类型标题，如 "海运出口订单" */
  title?: string;
  /** 订单聚合数据对象 */
  data: OrderDetailData;
  /** 加载中状态 */
  loading?: boolean;
  /** 返回列表回调 */
  onBack?: () => void;
  /** 刷新数据回调 */
  onRefresh?: () => void;
  /** 点击编辑订单回调（有权限且草稿/允许修改时） */
  onEdit?: () => void;
  /** 打开费用录入面板 */
  onOpenFees?: () => void;
  /** 打开里程碑履约面板 */
  onOpenMilestones?: () => void;
  /** 打开放货凭证 (POD) 面板 */
  onOpenReleasePod?: () => void;
  /** 打开异常标记与登记面板 */
  onOpenAbnormal?: () => void;
  /** 顶部右侧自定义操作按钮组插槽 */
  extraActions?: ReactNode[];
  /** 自定义扩展区块插槽 */
  children?: ReactNode;
}
