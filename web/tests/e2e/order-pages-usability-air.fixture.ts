import { expect, type Page, type TestInfo } from '@playwright/test';
import {
  OrderBusinessType,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '../../src/enums.generated';

export const airOrderIds = { ae: 'air-acceptance-ae', ai: 'air-acceptance-ai' };
const org = 'air-acceptance-org';
const customer = 'air-acceptance-customer';

function master(type: 'AE' | 'AI'): API.AirMasterWaybill {
  return {
    id: `master-${type.toLowerCase()}`,
    organizationId: org,
    businessType: type,
    version: '1',
    documentMode: 'HOUSE',
    memberCount: 1,
    airlineName: '合成航空',
    originAirportName: '上海浦东 (PVG)',
    destinationAirportName: '东京成田 (NRT)',
    content: {
      masterNo: type === 'AE' ? '78112345675' : '78112345682',
      airlineId: 'airline-1',
      originAirportId: 'airport-pvg',
      destinationAirportId: 'airport-nrt',
      flightNo: 'CA123',
      etd: '2026-11-18T01:00:00Z',
      eta: '2026-11-18T05:00:00Z',
      documentCutoffAt: '2026-11-17T12:00:00Z',
      customsCutoffAt: '2026-11-17T10:00:00Z',
      status: 'DRAFT',
      waybillContent: {
        shipperText: '合成发货人',
        consigneeText: '合成收货人',
        goodsDescription: '测试航空货物',
      },
    },
  };
}

function order(type: 'AE' | 'AI'): API.Order {
  const id = type === 'AE' ? airOrderIds.ae : airOrderIds.ai;
  const isImport = type === 'AI';
  const m = master(type);
  const c = m.content as API.AirMasterWaybillInput;
  return {
    id,
    orderNo: `${type}-ACCEPTANCE-001`,
    organizationId: org,
    organizationName: '合成验收公司',
    businessType: isImport
      ? OrderBusinessType.BUSINESS_TYPE_AI
      : OrderBusinessType.BUSINESS_TYPE_AE,
    version: '1',
    allowedActions: [],
    flowStatus: isImport
      ? OrderFlowStatus.ORDER_FLOW_STATUS_IMPORT_IN_PROGRESS
      : OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
    customerId: customer,
    customerName: '合成验收客户',
    customerReferenceNo: 'AIR-REF-001',
    internalReferenceNo: 'AIR-INTERNAL-001',
    totalPackages: 20,
    totalPackageUnit: 'CTNS',
    totalGrossWeightKg: 125,
    totalVolumeCbm: 1.2,
    goodsDescription: '测试航空货物',
    orderDate: '2026-11-16T00:00:00Z',
    createdAt: '2026-11-16T00:00:00Z',
    updatedAt: '2026-11-16T00:00:00Z',
    etd: c.etd,
    eta: c.eta,
    docCutoff: c.documentCutoffAt,
    customsCutoff: c.customsCutoffAt,
    originLocationId: c.originAirportId,
    destinationLocationId: c.destinationAirportId,
    originLocationName: '上海浦东 (PVG)',
    destinationLocationName: '东京成田 (NRT)',
    airExport: {
      master: m,
      documentMode: 'HOUSE',
      linkId: `link-${type}`,
      linkVersion: '1',
      active: true,
      house: {
        id: `house-${type}`,
        houseNo: `${type}HAWB001`,
        status: 'DRAFT',
        version: '1',
        content: {
          shipperText: '合成发货人',
          consigneeText: '合成收货人',
          goodsDescription: '测试航空货物',
        },
        contentHash: 'hash',
      },
      weightSummaries: [
        {
          source: 'PLANNED',
          confirmedWeightKg: '125',
          confirmationStale: false,
          confirmedAt: '2026-11-16T00:00:00Z',
        },
      ],
    },
  };
}

export async function installAirFixture(page: Page, type: 'AE' | 'AI') {
  const current = order(type);
  const masterData = master(type);
  const writes: string[] = [];
  const unexpected: string[] = [];
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  const permissions = [
    'read',
    'create',
    'update',
    'personnel.read',
    'cargo_item.read',
    'cargo_item.create',
    'attachment.read',
    'fee.read',
    'fee.update',
    'lock',
    'reassign',
  ].map((x) => `business.order.${type === 'AE' ? 'ae' : 'ai'}.${x}`);
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname;
    const json = (data: unknown, extra: Record<string, unknown> = {}) =>
      route.fulfill({ json: { success: true, data, ...extra } });
    if (req.method() !== 'GET' && path !== '/api/v1/air-weights/preview') {
      writes.push(`${req.method()} ${path}`);
      return route.fulfill({
        status: 405,
        json: { success: false, message: '只读验收禁止写入' },
      });
    }
    if (path === '/api/v1/auth/me')
      return json({
        id: 'air-acceptance-user',
        username: 'air-acceptance',
        displayName: '空运验收人员',
        currentOrganization: { id: org, name: '合成验收公司', kind: 2 },
        permissions,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'organization',
        })),
      });
    if (path === `/api/v1/orders/${current.id}` && req.method() === 'GET')
      return json(current);
    if (path === `/api/v1/air-export/master-waybills/${masterData.id}`)
      return json(masterData, {
        members: [],
        total: 1,
        totals: {
          memberCount: 1,
          complete: true,
          grossWeightKg: 125,
          volumeCbm: 1.2,
        },
      });
    if (path === '/api/v1/air-export/master-waybills')
      return json([masterData], { total: 1 });
    if (path === `/api/v1/orders/${current.id}/cargo-items`)
      return json(
        [
          {
            id: `cargo-${type}`,
            orderId: current.id,
            packageCount: 20,
            grossWeightKg: '125',
            volumeCbm: '1.2',
            version: '1',
          },
        ],
        { total: 1 },
      );
    if (path === '/api/v1/air-weights/preview') {
      const input = req.postDataJSON() as API.PreviewAirWeightRequest;
      return json({
        scope: 'ORDER',
        targetId: current.id,
        source: input.source,
        inputFingerprint: 'fingerprint',
        available: true,
        grossWeightKg: '125',
        volumeCbm: '1.2',
        volumetricWeightKg: '200',
        bubbleDifferenceKg: '75',
        sharedBubbleWeightKg: '75',
        suggestedChargeableWeightKg: '200',
        volumetricDivisor: '6000',
        bubbleShareNumerator: 1,
        bubbleShareDenominator: 1,
        currentConfirmation: {
          id: 'confirm',
          scope: 'ORDER',
          targetId: current.id,
          source: input.source,
          version: '1',
          confirmedWeightKg: '125',
          confirmedAt: '2026-11-16T00:00:00Z',
        },
        confirmationStale: false,
      });
    }
    if (path === `/api/v1/air-weights/history`) return json([], { total: 0 });
    if (path === `/api/v1/orders/${current.id}/import-handling`)
      return json({
        orderId: current.id,
        orderVersion: '1',
        steps: [1, 2, 3, 4, 5].map((kind) => ({
          kind,
          undertaken: kind === 1,
          referenceNo: '',
          notes: '',
          attachmentIds: [],
        })),
      });
    if (path === `/api/v1/orders/${current.id}/import-handling/events`)
      return json({ items: [], total: 0 });
    if (path === `/api/v1/orders/${current.id}/personnel`) return json([]);
    if (path === '/api/v1/order-personnel-options') return json([]);
    if (
      path === '/api/v1/master-data/options' ||
      path === '/api/v1/master-data/options/search'
    )
      return json([
        { id: 'general', code: 'GENERAL', name: '普货', enabled: true },
      ]);
    if (path === '/api/v1/master-data/airports')
      return json([
        {
          id: 'airport-pvg',
          nameZh: '上海浦东',
          iataCode: 'PVG',
          enabled: true,
        },
        {
          id: 'airport-nrt',
          nameZh: '东京成田',
          iataCode: 'NRT',
          enabled: true,
        },
      ]);
    if (path === '/api/v1/master-data/airlines')
      return json([
        { id: 'airline-1', nameZh: '合成航空', iataCode: 'CA', enabled: true },
      ]);
    if (path === '/api/v1/reference/currencies')
      return json([
        { code: 'USD', name: '美元', enabled: true },
        { code: 'CNY', name: '人民币', enabled: true },
      ]);
    if (path === '/api/v1/partners')
      return json([
        { id: customer, legalName: '合成验收客户', code: 'AIR-CUSTOMER' },
      ]);
    if (path === '/api/v1/order-reference-check')
      return json({ duplicate: false });
    if (path.includes('/lock-state'))
      return json({
        orderId: current.id,
        orderVersion: current.version,
        isLocked: false,
      });
    if (
      path.includes('/attachments') ||
      path.includes('/release-pods') ||
      path.includes('/abnormal-cases')
    )
      return json([], { total: 0 });
    // 未声明的请求必须暴露，不能用空成功响应掩盖接口缺失。
    unexpected.push(`${req.method()} ${path}`);
    return route.fulfill({
      status: 404,
      json: { success: false, message: '未声明的验收请求' },
    });
  });
  return {
    writes,
    unexpected,
    async report(testInfo: TestInfo) {
      await testInfo.attach('空运页面请求记录', {
        body: JSON.stringify({ writes, unexpected, pageErrors }, null, 2),
        contentType: 'application/json',
      });
      expect(writes).toEqual([]);
      expect(unexpected).toEqual([]);
      expect(pageErrors).toEqual([]);
    },
  };
}
