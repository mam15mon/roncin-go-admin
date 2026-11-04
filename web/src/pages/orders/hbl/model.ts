import dayjs, { type Dayjs } from 'dayjs';
import {
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';

export type HblBundle = {
  order: API.Order;
  documents: API.SeaOrderDocuments;
};

export type HblFormValues = {
  source: {
    houseId?: string;
    houseVersion?: string;
    linkVersion?: string;
  };
  content: API.SeaBillContent;
  printDetails: Omit<
    API.SeaHouseBillPrintDetails,
    'shippedOnBoardDate' | 'issueDate'
  > & {
    shippedOnBoardDate?: string | Dayjs;
    issueDate?: string | Dayjs;
  };
};

/** 初次制单可从已保存订单提供印字，已维护的印字不会随订单刷新改变。 */
export function buildHblFormValues(bundle: HblBundle): HblFormValues {
  const { order, documents } = bundle;
  const details = documents.houseBill?.printDetails ?? {
    bookingNoText: order.bookingNo,
    vesselVoyageText: order.vesselVoyage,
    portOfLoadingText: order.originLocationName,
    portOfDischargeText: order.dischargeLocationName,
    placeOfDeliveryText: order.destinationLocationName,
  };
  return {
    source: {
      houseId: documents.houseBill?.id,
      houseVersion: documents.houseBill?.version,
      linkVersion: documents.linkVersion,
    },
    content: { ...documents.houseBill?.content },
    printDetails: {
      ...details,
      shippedOnBoardDate: details.shippedOnBoardDate
        ? dayjs(details.shippedOnBoardDate)
        : undefined,
      issueDate: details.issueDate ? dayjs(details.issueDate) : undefined,
    },
  };
}

/** 表单日期与空值只在请求边界规范化，不能借机改写打印正文。 */
export function hblFormPayload(values: HblFormValues): {
  content: API.SeaBillContent;
  printDetails: API.SeaHouseBillPrintDetails;
} {
  const date = (value?: string | Dayjs) =>
    value
      ? dayjs.isDayjs(value)
        ? value.format('YYYY-MM-DD')
        : value
      : undefined;
  return {
    content: { ...values.content },
    printDetails: {
      ...values.printDetails,
      shippedOnBoardDate: date(values.printDetails?.shippedOnBoardDate),
      issueDate: date(values.printDetails?.issueDate),
    },
  };
}

export function hblValuesFingerprint(values: HblFormValues): string {
  const normalize = (value: unknown): unknown => {
    if (value === '' || value == null) return undefined;
    if (typeof value !== 'object') return value;
    return Object.fromEntries(
      Object.entries(value)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, item]) => [key, normalize(item)])
        .filter(([, item]) => item !== undefined),
    );
  };
  return JSON.stringify(
    normalize({ ...hblFormPayload(values), source: values.source }),
  );
}

export function hblBundleFingerprint(bundle: HblBundle): string {
  return JSON.stringify([
    bundle.order.id,
    bundle.order.version,
    bundle.order.seaMasterBill?.masterBillId,
    bundle.order.seaMasterBill?.transportExecutionId,
    bundle.order.seaMasterBill?.transportExecutionVersion,
    bundle.documents.documentStructure,
    bundle.documents.linkVersion,
    bundle.documents.masterBill?.id,
    bundle.documents.masterBill?.version,
    bundle.documents.houseBill?.id,
    bundle.documents.houseBill?.version,
  ]);
}

export function hblUnavailableReason(bundle: HblBundle): string | undefined {
  const house = bundle.documents.houseBill;
  if (!house?.id) return '当前订单没有可制作的 HBL，请先在订单资料中维护分单。';
  if (house.status === SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED)
    return '当前 HBL 已作废，不能制作。';
  if (
    house.issuerSource !==
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION ||
    house.printTemplate !== 'RONCIN_BEIJING'
  )
    return '当前 HBL 暂无适用模板，本期仅支持融迅北京公司自签 HBL。';
  return undefined;
}

export function formatHblContainers(
  orderId: string,
  exclusive: API.OrderContainer[],
  shared: API.SeaSharedContainer[],
  specs: API.MasterDataItem[],
): string {
  const names = new Map(specs.map((spec) => [spec.id, spec.code || spec.name]));
  const rows = exclusive.map((row) => ({
    no: row.containerNo,
    seal: row.sealNo,
    spec: names.get(row.containerSpecId),
  }));
  for (const row of shared) {
    if (!row.allocations?.some((allocation) => allocation.orderId === orderId))
      continue;
    rows.push({
      no: row.containerNo,
      seal: row.sealNo,
      spec: row.containerSpecName || names.get(row.containerSpecId),
    });
  }
  if (rows.some((row) => !row.no || !row.spec))
    throw new Error('实际箱号或箱型资料不完整，请先在订单装箱资料中补齐。');
  return rows
    .map((row) =>
      [row.no, row.seal ? `SEAL ${row.seal}` : undefined, row.spec]
        .filter(Boolean)
        .join(' / '),
    )
    .join('\n');
}
