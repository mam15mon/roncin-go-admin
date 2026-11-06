import { expect, type Page, type Route, type TestInfo } from '@playwright/test';
import {
  ContainerOwnership,
  OrderAllowedAction,
  OrderBusinessType,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderPersonnelRole,
  OrderTerminationStatus,
  PaymentTerm,
  SeaDocumentAction,
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
  ShipmentMode,
  ShipmentType,
  TradeTerm,
} from '../../src/enums.generated';

export const stabilityOrderId = 'sea-stability-order';
export const stabilityOrderPath = `/orders/sea-export/${stabilityOrderId}`;
const orderAPI = `/api/v1/orders/${stabilityOrderId}`;
const organizationId = 'sea-stability-org';
const customerId = 'sea-stability-customer';

/** 完整详情字段与真实组件使用合成业务响应，所有 API 都在浏览器内截获。 */
export async function installSeaStabilityFixture(page: Page) {
  const writes: API.UpdateOrderRequest[] = [];
  const reads: string[] = [];
  const unexpectedRequests: string[] = [];
  const pageErrors: string[] = [];
  const renderErrors: string[] = [];
  const consoleErrors: string[] = [];
  let heldReadKind: 'order' | 'documents' | undefined;
  let releaseRead: (() => void) | undefined;
  let failDocumentsAfterWrite = false;
  let failedDocuments = 0;
  const master: API.SeaMasterBillSummary = {
    masterBillId: 'sea-stability-master',
    masterNo: 'STABILITYMBL001',
    shippingLineId: 'sea-stability-line',
    shippingLineName: '稳定性测试船公司',
    transportExecutionId: 'sea-stability-te',
    transportExecutionVersion: '1',
    vesselName: 'STABILITY VESSEL',
    voyageNo: 'V001',
    originLocationId: 'stability-sha',
    originLocationName: '上海港',
    dischargeLocationId: 'stability-sin',
    dischargeLocationName: '新加坡港',
    etd: '2026-11-10T00:00:00+08:00',
    eta: '2026-11-12T00:00:00+08:00',
    status: 'DRAFT',
    version: '1',
    memberCount: 1,
  };
  const documents: API.SeaOrderDocuments = {
    orderId: stabilityOrderId,
    documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
    linkVersion: '1',
    masterBill: {
      id: master.masterBillId,
      masterNo: master.masterNo,
      shippingLineId: master.shippingLineId,
      shippingLineName: master.shippingLineName,
      status: 'DRAFT',
      version: '1',
      memberCount: 1,
      currentVersionId: 'sea-stability-master-version',
      content: { freightTerms: '预付', transportTerms: 'CY-CY' },
    },
    houseBill: {
      id: 'sea-stability-house',
      orderId: stabilityOrderId,
      organizationId,
      masterBillId: master.masterBillId,
      houseNo: 'STABILITYHBL001',
      issuerSource:
        SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
      issuerOrganizationId: organizationId,
      issuerOrganizationName: '稳定性测试公司',
      issuerOrganizationNameEn: 'STABILITY TEST COMPANY LTD',
      status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
      version: '1',
      currentVersionId: 'sea-stability-house-version',
      content: {
        freightTerms: '预付',
        transportTerms: 'CY-CY',
        shipperText: '合成测试发货人',
      },
    },
    allowedActions: [
      SeaDocumentAction.SEA_DOCUMENT_ACTION_UPDATE_HOUSE_BILL,
      SeaDocumentAction.SEA_DOCUMENT_ACTION_CHANGE_MODE,
    ],
  };
  let order: API.Order = {
    id: stabilityOrderId,
    orderNo: 'SE-STABILITY-001',
    organizationId,
    organizationName: '稳定性测试公司',
    businessType: OrderBusinessType.BUSINESS_TYPE_SE,
    version: '1',
    allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
    flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
    customerId,
    customerName: '稳定性测试客户',
    customerReferenceNo: 'STABILITY-REF001',
    shippingLineId: master.shippingLineId,
    shippingLineName: master.shippingLineName,
    shipmentType: ShipmentType.SHIPMENT_TYPE_FCL,
    containerOwnership: ContainerOwnership.CONTAINER_OWNERSHIP_COC,
    shipmentMode: ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
    tradeTerm: TradeTerm.TRADE_TERM_FOB,
    paymentTerm: PaymentTerm.PAYMENT_TERM_PREPAID,
    originLocationId: master.originLocationId,
    originLocationName: master.originLocationName,
    dischargeLocationId: master.dischargeLocationId,
    dischargeLocationName: master.dischargeLocationName,
    destinationLocationId: master.dischargeLocationId,
    destinationLocationName: master.dischargeLocationName,
    etd: master.etd,
    eta: master.eta,
    vesselVoyage: 'STABILITY VESSEL / V001',
    totalPackages: 100,
    totalGrossWeightKg: 2000,
    totalVolumeCbm: 15,
    totalPackageUnit: 'CTNS',
    goodsDescription: '初始合成货物描述',
    operationNotes: '初始合成操作备注',
    orderDate: '2026-11-06T08:00:00+08:00',
    createdAt: '2026-11-06T08:00:00+08:00',
    updatedAt: '2026-11-06T08:00:00+08:00',
    seaMasterBill: master,
    seaDocumentSummary: {
      documentStructure: documents.documentStructure,
      linkVersion: documents.linkVersion,
      houseNo: documents.houseBill?.houseNo,
    },
  };
  const personnel: API.OrderPersonnel[] = [
    {
      userId: 'stability-operator',
      role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
    },
    {
      userId: 'stability-sales',
      role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES,
    },
    {
      userId: 'stability-service',
      role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE,
    },
  ];
  const customer = {
    id: customerId,
    legalName: '稳定性测试客户',
    code: 'STABILITY-CUSTOMER',
    assignments: personnel.map((person) => ({
      role: person.role,
      userId: person.userId,
    })),
  };
  page.on('pageerror', (error) =>
    pageErrors.push(error.stack ?? error.message),
  );
  page.on('console', (message) => {
    if (message.type() !== 'error') return;
    const text = message.text();
    consoleErrors.push(text);
    // 单独保留已知控件警告；它们没有异常栈，不能据此认定路由错误页根因。
    const knownWarning =
      /Instance created by `useForm` is not connected|React does not recognize the.*showCount|A component is changing (?:a controlled|an uncontrolled) input/.test(
        text,
      );
    if (!knownWarning && !text.startsWith('Failed to load resource:'))
      renderErrors.push(text);
  });
  await page.route('**/api/**', async (route: Route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const send = (data: unknown, extra = {}) =>
      route.fulfill({ json: { success: true, data, ...extra } });
    const kind =
      path === orderAPI && request.method() === 'GET'
        ? 'order'
        : path === `${orderAPI}/sea-documents`
          ? 'documents'
          : undefined;
    if (kind) {
      reads.push(kind);
      if (heldReadKind === kind) {
        heldReadKind = undefined;
        await new Promise<void>((resolve) => {
          releaseRead = resolve;
        });
        releaseRead = undefined;
      }
    }
    if (path === '/api/v1/auth/me') {
      const permissions = [
        ...[
          'read',
          'create',
          'update',
          'split',
          'reassign',
          'personnel.read',
          'container.read',
          'cargo_item.read',
          'cargo_item.create',
          'attachment.read',
          'release_pod.read',
        ].map((operation) => `business.order.se.${operation}`),
        'business.partner.read',
        'system.master_data.option.read',
        'system.master_data.port.read',
      ];
      return send({
        id: 'stability-user',
        username: 'stability',
        displayName: '稳定性测试人员',
        currentOrganization: {
          id: organizationId,
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
    if (path === orderAPI && request.method() === 'GET') return send(order);
    if (path === orderAPI && request.method() === 'PUT') {
      const body = request.postDataJSON() as API.UpdateOrderRequest;
      writes.push(body);
      order = {
        ...order,
        goodsDescription: body.goodsDescription,
        operationNotes: body.operationNotes,
        version: String(Number(order.version) + 1),
      };
      return send(order);
    }
    if (path === `${orderAPI}/sea-documents`) {
      if (
        failDocumentsAfterWrite &&
        writes.length > 0 &&
        failedDocuments === 0
      ) {
        failedDocuments += 1;
        return route.fulfill({
          status: 503,
          json: { code: 503, message: '模拟保存后单证读取失败' },
        });
      }
      return send(documents);
    }
    if (path === `${orderAPI}/lock-state`)
      return send({
        orderId: stabilityOrderId,
        businessType: order.businessType,
        orderVersion: order.version,
        isLocked: false,
      });
    if (path === `${orderAPI}/sea-order-change/actions`)
      return send({ canSplit: true, canReassign: true });
    if (path === `${orderAPI}/personnel`) return send(personnel);
    if (path === `${orderAPI}/sea-order-change/split-context`)
      return send({
        orderId: stabilityOrderId,
        orderNo: order.orderNo,
        orderVersion: order.version,
        currentLinkVersion: '1',
        documentStructure: 'HOUSE',
        flowStatus: 'BOOKED',
        currentMasterBill: { id: master.masterBillId, ...master },
        currentHouseBill: {
          id: documents.houseBill?.id,
          houseNo: documents.houseBill?.houseNo,
          version: '1',
          status: 'DRAFT',
        },
        cargoItems: [
          {
            id: 'stability-cargo',
            cargoName: '合成实际货物',
            packageCount: 100,
            grossWeightKg: '2000',
            volumeCbm: '15',
            version: '1',
          },
        ],
      });
    if (path === `${orderAPI}/sea-order-change/split/preview`)
      return send({
        isValid: false,
        conservationPassed: true,
        baseline: { packageCount: 100, grossWeightKg: '2000', volumeCbm: '15' },
        allocated: {
          packageCount: 100,
          grossWeightKg: '2000',
          volumeCbm: '15',
        },
        validationErrors: [
          { reason: 'CREATED_ORDER_EMPTY', message: '新票必须分配非零货物' },
        ],
      });
    if (
      [
        `${orderAPI}/containers`,
        `${orderAPI}/sea-shared-containers`,
        `${orderAPI}/same-batch`,
        `${orderAPI}/sea-order-change/events`,
        `${orderAPI}/release-pods`,
        `${orderAPI}/attachments`,
        `${orderAPI}/unlock-requests`,
      ].includes(path)
    )
      return send([]);
    if (path === '/api/v1/master-data/options/search') return send([]);
    if (path === '/api/v1/master-data/options')
      return send([
        {
          id: 'stability-general',
          name: '普货',
          code: 'GENERAL',
          kind: 9,
          enabled: true,
        },
        {
          id: 'stability-box',
          name: '40HQ',
          code: '40HQ',
          kind: 7,
          enabled: true,
        },
      ]);
    if (path === '/api/v1/reference/currencies')
      return send([
        { code: 'USD', name: '美元', enabled: true },
        { code: 'CNY', name: '人民币', enabled: true },
      ]);
    if (path === '/api/v1/master-data/ports')
      return send([
        {
          id: 'stability-sha',
          nameZh: '上海港',
          nameEn: 'SHANGHAI',
          unLocode: 'CNSHA',
          enabled: true,
        },
        {
          id: 'stability-sin',
          nameZh: '新加坡港',
          nameEn: 'SINGAPORE',
          unLocode: 'SGSIN',
          enabled: true,
        },
      ]);
    if (path === '/api/v1/master-data/shipping-lines')
      return send([
        {
          id: master.shippingLineId,
          nameZh: master.shippingLineName,
          nameEn: 'STABILITY LINE',
          scacCode: 'STAB',
          enabled: true,
        },
      ]);
    if (path === '/api/v1/partners') return send([customer]);
    if (path === `/api/v1/partners/${customerId}`) return send(customer);
    if (path === '/api/v1/order-personnel-options')
      return send(
        personnel.map((person, index) => ({
          userId: person.userId,
          displayName: ['测试操作员', '测试业务员', '测试客服'][index],
        })),
      );
    if (path === '/api/v1/orders/sea-master-bill-candidate') return send(null);
    if (path === '/api/v1/order-reference-check')
      return send({ duplicate: false });
    unexpectedRequests.push(`${request.method()} ${path}`);
    return route.fulfill({
      status: 404,
      json: { message: '浏览器回归夹具未声明此请求' },
    });
  });
  return {
    writes,
    reads,
    pageErrors,
    renderErrors,
    consoleErrors,
    unexpectedRequests,
    holdNext(kind: 'order' | 'documents') {
      heldReadKind = kind;
    },
    async waitHeld() {
      await expect.poll(() => Boolean(releaseRead)).toBe(true);
    },
    release() {
      releaseRead?.();
    },
    failNextSavedDocuments() {
      failDocumentsAfterWrite = true;
    },
    getOrder() {
      return order;
    },
    async report(testInfo: TestInfo) {
      await testInfo.attach('页面稳定性浏览器记录', {
        body: JSON.stringify(
          {
            writes,
            reads,
            pageErrors,
            renderErrors,
            consoleErrors,
            unexpectedRequests,
          },
          null,
          2,
        ),
        contentType: 'application/json',
      });
      expect(pageErrors).toEqual([]);
      expect(renderErrors).toEqual([]);
      expect(unexpectedRequests).toEqual([]);
    },
  };
}
