import { expect, type Page, type TestInfo } from '@playwright/test';
import {
  OrderBusinessType,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
  TradeDirection,
} from '../../src/enums.generated';
import {
  airOrderIds,
  installAirFixture,
} from './order-pages-usability-air.fixture';
import {
  installSeaStabilityFixture,
  stabilityOrderId,
} from './sea-operation-stability.fixture';

/** 扩展验收明确列出每个读取路径；未知请求失败，不交给基础夹具空成功兜底。 */
export async function installExtraFixture(
  page: Page,
  type: 'SI' | 'AE' | 'AI',
  direct = false,
) {
  const sea =
    type === 'SI' ? await installSeaStabilityFixture(page) : undefined;
  if (!sea) await installAirFixture(page, type === 'SI' ? 'AI' : type);
  const id = sea
    ? stabilityOrderId
    : type === 'AE'
      ? airOrderIds.ae
      : airOrderIds.ai;
  const unexpected: string[] = [];
  const writes: string[] = [];
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];
  const baselineWarnings: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  page.on('console', (entry) => {
    if (entry.type() !== 'error') return;
    const text = entry.text();
    if (
      text ===
      'Warning: Instance created by `useForm` is not connected to any Form element. Forget to pass `form` prop?'
    )
      baselineWarnings.push(text);
    else consoleErrors.push(text);
  });
  const orderPath = `/api/v1/orders/${id}`;
  const allowed = new Set([
    '/api/v1/auth/me',
    orderPath,
    `${orderPath}/personnel`,
    `${orderPath}/lock-state`,
    `${orderPath}/attachments`,
    `${orderPath}/release-pods`,
    `${orderPath}/abnormal-cases`,
    `${orderPath}/unlock-requests`,
    `${orderPath}/cargo-items`,
    `${orderPath}/import-handling`,
    `${orderPath}/import-handling/events`,
    `${orderPath}/sea-documents`,
    `${orderPath}/containers`,
    `${orderPath}/sea-shared-containers`,
    `${orderPath}/same-batch`,
    `${orderPath}/sea-order-change/events`,
    '/api/v1/order-personnel-options',
    '/api/v1/master-data/options',
    '/api/v1/master-data/options/search',
    '/api/v1/master-data/ports',
    '/api/v1/master-data/shipping-lines',
    '/api/v1/reference/currencies',
    '/api/v1/master-data/airports',
    '/api/v1/master-data/airlines',
    '/api/v1/partners',
    '/api/v1/partners/sea-stability-customer',
    '/api/v1/orders/sea-master-bill-candidate',
    '/api/v1/order-reference-check',
    '/api/v1/air-weights/preview',
    '/api/v1/air-weights/history',
    '/api/v1/air-export/master-waybills',
    `/api/v1/air-export/master-waybills/master-${type.toLowerCase()}`,
    `/api/v1/air-export/orders/${id}/house-versions`,
  ]);
  await page.route('**/api/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const send = (data: unknown, extra: Record<string, unknown> = {}) =>
      route.fulfill({ json: { success: true, data, ...extra } });
    if (request.method() === 'POST' && path === '/api/v1/air-weights/preview')
      return route.fallback();
    if (request.method() !== 'GET') {
      writes.push(`${request.method()} ${path}`);
      return route.fulfill({
        status: 405,
        json: { success: false, message: '此验收不允许业务写入' },
      });
    }
    if (!allowed.has(path)) {
      unexpected.push(`${request.method()} ${path}`);
      return route.fulfill({
        status: 404,
        json: { success: false, message: '扩展验收未声明此读取' },
      });
    }
    if (sea && path === '/api/v1/auth/me') {
      const permissions = [
        'read',
        'create',
        'update',
        'personnel.read',
        'container.read',
        'cargo_item.read',
        'attachment.read',
        'release_pod.read',
      ]
        .map((operation) => `business.order.si.${operation}`)
        .concat([
          'business.partner.read',
          'system.master_data.option.read',
          'system.master_data.port.read',
        ]);
      return send({
        id: 'stability-user',
        username: 'stability',
        displayName: '进口页面验收人员',
        currentOrganization: {
          id: 'sea-stability-org',
          name: '稳定性测试公司',
          kind: 2,
        },
        permissions,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'organization',
        })),
      });
    }
    if (sea && path === `${orderPath}/lock-state`)
      return send({
        orderId: id,
        orderVersion: '1',
        businessType: OrderBusinessType.BUSINESS_TYPE_SI,
        isLocked: false,
      });
    if (sea && path === orderPath)
      return send({
        ...sea.getOrder(),
        businessType: OrderBusinessType.BUSINESS_TYPE_SI,
        tradeDirection: TradeDirection.TRADE_DIRECTION_IMPORT,
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_IMPORT_IN_PROGRESS,
      });
    if (path === `${orderPath}/import-handling`)
      return send({
        orderId: id,
        orderVersion: '1',
        steps: [1, 2, 3, 4, 5].map((kind) => ({
          kind,
          undertaken: kind === 1,
        })),
      });
    if (path === `${orderPath}/import-handling/events`)
      return send({ items: [], total: 0 });
    if (path === `/api/v1/air-export/orders/${id}/house-versions`)
      return send([], { total: 0 });
    if (direct && path === orderPath) {
      const master: API.AirMasterWaybill = {
        id: `master-${type.toLowerCase()}`,
        organizationId: 'air-acceptance-org',
        businessType: type,
        version: '1',
        documentMode: 'DIRECT',
        airlineName: '合成直单航空',
        originAirportName: '上海浦东',
        destinationAirportName: '东京成田',
        content: {
          masterNo: '78112345675',
          airlineId: 'airline-1',
          originAirportId: 'airport-pvg',
          destinationAirportId: 'airport-nrt',
          flightNo: 'CA123',
          etd: '2026-11-18T01:00:00Z',
          eta: '2026-11-18T05:00:00Z',
          status: 'DRAFT',
          waybillContent: {
            shipperText: '直单发货人',
            consigneeText: '直单收货人',
            goodsDescription: '直单测试货物',
          },
        },
      };
      return send({
        id,
        orderNo: `${type}-DIRECT-001`,
        organizationId: 'air-acceptance-org',
        version: '1',
        businessType:
          type === 'AE'
            ? OrderBusinessType.BUSINESS_TYPE_AE
            : OrderBusinessType.BUSINESS_TYPE_AI,
        tradeDirection:
          type === 'AE'
            ? TradeDirection.TRADE_DIRECTION_EXPORT
            : TradeDirection.TRADE_DIRECTION_IMPORT,
        flowStatus:
          type === 'AE'
            ? OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED
            : OrderFlowStatus.ORDER_FLOW_STATUS_IMPORT_IN_PROGRESS,
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
        closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
        allowedActions: [],
        customerId: 'air-acceptance-customer',
        customerName: '合成验收客户',
        totalPackages: 20,
        totalPackageUnit: 'CTNS',
        totalGrossWeightKg: 125,
        totalVolumeCbm: 1.2,
        airExport: {
          documentMode: 'DIRECT',
          active: true,
          linkId: `link-${type}`,
          linkVersion: '1',
          master,
        },
      } satisfies API.Order);
    }
    return route.fallback();
  });
  return {
    id,
    async report(info: TestInfo) {
      await info.attach('扩展页面请求与错误记录', {
        body: JSON.stringify(
          { unexpected, writes, pageErrors, consoleErrors, baselineWarnings },
          null,
          2,
        ),
        contentType: 'application/json',
      });
      expect(unexpected).toEqual([]);
      expect(writes).toEqual([]);
      expect(pageErrors).toEqual([]);
      expect(consoleErrors).toEqual([]);
    },
  };
}
