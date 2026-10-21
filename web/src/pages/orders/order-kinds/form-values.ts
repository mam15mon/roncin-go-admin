import type dayjs from 'dayjs';

export type CreateOrderFormValues = {
  customerId: string;
  customerReferenceNo?: string;
  bookingNo?: string;
  internalReferenceNo?: string;
  customerCode?: string;
  tradeTerm?: number;
  paymentTerm: number;
  shippingLineId?: string;
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
  /** 危险品包装类别（IATA DGR）；空运专用，0/UNSPECIFIED 视同未选。 */
  dangerousPackingGroup?: number;
  factoryName?: string;
  cargoReadyAt?: string | dayjs.Dayjs;
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
  seaMasterBillMasterNo?: string;
  seaMasterBillCandidateId?: string;
  seaMasterBillExpectedCandidateVersion?: number | string;
  seaMasterBillCandidateTeId?: string;
  seaMasterBillExpectedCandidateTeVersion?: number | string;
  seaMasterBillCorrectionReason?: string;
  seaMasterBill?: API.SeaMasterBillInput;
  /** 命中共享主单批次时由候选响应写入的分单号清单（含作废），仅用于失焦即时排重提示，不提交。 */
  seaMasterBillBatchHouseNos?: string[];
  operatorUserId?: string;
  salesUserId?: string;
  customerServiceUserId?: string;
  associateUserId?: string;
  documentUserId?: string;
  commercialUserId?: string;
  associate2UserId?: string;
  creatorUserId?: string;
  seaDocumentStructure?: number;
  seaMasterBillContent?: API.SeaBillContent;
  seaHouseBill?: API.SeaHouseBillInput;
  seaDocument?: API.SeaOrderDocumentInput;
  airMasterMode?: 'NEW' | 'EXISTING';
  airMasterId?: string;
  airMasterExpectedVersion?: string;
  airMaster?: API.AirMasterWaybillInput;
  airHouseNo?: string;
  airHouseStatus?: string;
  airHouseContent?: string;
  airHouseChangeReason?: string;
  airExport?: API.AirOrderInfo;
};

export type OrderDetailFormValues = Omit<
  CreateOrderFormValues,
  'containerRequests' | 'seaMasterBill' | 'shippingDocuments'
> & {
  containerRequests?: Array<
    API.OrderContainerRequest | API.OrderContainerRequestInput
  >;
  shippingDocuments?: Array<
    API.OrderShippingDocument | API.OrderShippingDocumentInput
  >;
  orderNo?: string;
  seaMasterBill?: API.SeaMasterBillInput | API.SeaMasterBillSummary;
  seaDocumentLinkVersion?: string;
  seaDocumentSummary?: API.SeaOrderDocumentSummary;
};
