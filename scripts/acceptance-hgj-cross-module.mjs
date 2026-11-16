import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { invoiceRegistrationForBills, requireSuccessfulBillBatch } from './acceptance-finance-fixtures.mjs';
import { createHgjClient, masterNumber, orderKinds, requiredEnvironment } from './acceptance-hgj-client.mjs';

// 仅供已获授权的一次性验收库；编排器显式注入，禁止默认回落开发数据。
assert.equal(requiredEnvironment('RONCIN_HGJ_DISPOSABLE'), '1', '必须由隔离库编排启动');
process.umask(0o077);
const artifactDir = requiredEnvironment('RONCIN_ACCEPTANCE_ARTIFACT_DIR');
fs.mkdirSync(artifactDir, { recursive: true, mode: 0o700 });
const repository = fs.realpathSync(process.cwd());
assert(!fs.realpathSync(artifactDir).startsWith(`${repository}${path.sep}`), '证据目录必须在仓库外');
const client = await createHgjClient(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'), requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
const { request, raw, post, put } = client;
const stamp = Date.now();
const today = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
const plusDays = (date, count) => new Date(Date.parse(`${date}T12:00:00Z`) + count * 86400000).toISOString().slice(0, 10);
const businessDate = today;
const transactionDate = plusDays(today, 1);
const me = (await request('/api/v1/auth/me')).data;
const systemId = me.currentOrganization.id;
const companyA = (await post('/api/v1/admin/organizations', { code: `HGJ_A_${stamp}`, name: `四类闭环验收公司A${stamp}`, nameEn: 'HGJ ISOLATED COMPANY A', kind: 2, baseCurrency: 'CNY' })).data;
const companyB = (await post('/api/v1/admin/organizations', { code: `HGJ_B_${stamp}`, name: `四类闭环验收公司B${stamp}`, nameEn: 'HGJ ISOLATED COMPANY B', kind: 2, baseCurrency: 'CNY' })).data;
const shippingLine = (await post('/api/v1/master-data/shipping-lines', { scacCode: 'HGJA', nameZh: '四类验收船司', nameEn: 'HGJ ISOLATED CARRIER', countryCode: 'CN' })).data;
const airline = (await post('/api/v1/master-data/airlines', { iataCode: 'QX', awbPrefix: '990', nameZh: '四类验收航司', nameEn: 'HGJ ISOLATED AIRLINE', countryCode: 'CN' })).data;
const airports = [];
for (const [iataCode, nameZh] of [['XAA', '验收起运机场'], ['XAB', '验收目的机场']]) airports.push((await post('/api/v1/master-data/airports', { iataCode, nameZh, nameEn: `Isolated ${iataCode}`, cityNameZh: '验收城市', countryCode: 'CN' })).data);
const abnormalType = (await post('/api/v1/master-data/items', { kind: 10, code: `HGJ_ABN_${stamp}`, name: '四类验收待处理异常', source: '验收', sortOrder: 1 })).data;
const restricted = [];
for (const kind of orderKinds) {
  const username = `hgj_${kind.code.toLowerCase()}_${stamp}`;
  const password = `Hgj-${crypto.randomUUID()}`;
  const user = (await post('/api/v1/admin/users', { username, displayName: `仅${kind.code}验收人员`, password, roleIds: [] })).data;
  restricted.push({ ...kind, user, username, password });
}
await post('/api/v1/auth/switch-organization', { organizationId: companyA.id });
const permissions = (await request('/api/v1/admin/permissions')).data;
for (const identity of restricted) {
  const requested = [`business.order.${identity.code.toLowerCase()}.read`, `business.order.${identity.code.toLowerCase()}.fee.read`, `business.order.${identity.code.toLowerCase()}.abnormal_case.read`];
  assert(requested.every((key) => permissions.some((item) => item.key === key)), `${identity.code}权限键存在`);
  const role = (await post('/api/v1/admin/roles', { code: `hgj_${identity.code.toLowerCase()}_${stamp}`, name: `仅${identity.code}验收读取`, permissionKeys: requested })).data;
  await post('/api/v1/auth/switch-organization', { organizationId: systemId });
  await post(`/api/v1/admin/users/${identity.user.id}/memberships:batchCreate`, { userId: identity.user.id, items: [{ organizationId: companyA.id, roleIds: [role.id] }] });
  await post('/api/v1/auth/switch-organization', { organizationId: companyA.id });
}
const partner = (await post('/api/v1/partners', {
  code: `HGJ_PARTNER_${stamp}`, legalName: `四类验收客商${stamp}`, unifiedSocialCreditCode: `91310000HGJ${String(stamp).slice(-7)}`,
  roles: [{ type: 1, enabled: true }, { type: 2, enabled: true }],
  profile: { nameEn: 'HGJ ISOLATED PARTNER', countryCode: 'CN', businessTypes: [1, 2, 3, 4] },
  assignments: [3, 4].map((role) => ({ role, userId: me.id, organizationId: companyA.id })),
})).data;
const accounts = {};
for (const [direction, usage] of [['RECEIVABLE', 1], ['PAYABLE', 2]]) accounts[direction] = (await post(`/api/v1/partners/${partner.id}/accounts`, { partnerId: partner.id, account: {
  name: `${direction}验收账户`, accountHolder: partner.legalName, currency: 'CNY', bankName: '验收银行', accountNo: `HGJ${stamp}${usage}`, usage,
  isDefaultReceivable: usage === 1, isDefaultPayable: usage === 2, enabled: true,
} })).data;
const profile = (await post(`/api/v1/partners/${partner.id}/invoice-profiles`, { partnerId: partner.id, invoiceTitle: partner.legalName,
  taxpayerIdentificationNo: partner.unifiedSocialCreditCode, defaultInvoiceType: 'DIGITAL_NORMAL', isDefault: true })).data;
const taxable = (await post('/api/v1/finance/taxable-services', { name: `四类验收零税服务${stamp}`, defaultTaxRate: '0' })).data;
const units = (await request('/api/v1/finance/billing-units?page=1&pageSize=200')).data;
const unit = units.find((item) => item.code === 'KG');
assert(unit?.id, '正式迁移应初始化KG单位');
const feeSetting = (await post('/api/v1/finance/fee-settings', { feeCode: `HGJ_FEE_${stamp}`, nameZh: '四类验收经营费用', defaultCurrency: 'CNY', billingUnitId: unit.id, taxableServiceId: taxable.id, taxRate: '0' })).data;
await post('/api/v1/finance/commission-rules', { organizationId: companyA.id, rule: { name: `四类验收方案${stamp}`, personnelRole: 'SALES', employeeIds: [me.id], calculationBasis: 'REALIZED_PROFIT', ratePercent: '10', effectiveFrom: today, effectiveTo: transactionDate, enabled: true } });
let serial = 0;
function orderInput(kind, reference, extra = {}) {
  const index = serial++;
  const etd = kind.direction === 1 ? `${businessDate}T00:30:00+08:00` : `${plusDays(businessDate, -1)}T00:30:00+08:00`;
  const eta = kind.direction === 2 ? `${businessDate}T00:30:00+08:00` : `${plusDays(businessDate, 1)}T00:30:00+08:00`;
  return {
    customerId: partner.id, businessType: kind.type, tradeDirection: kind.direction, paymentTerm: 2, tradeTerm: 3,
    shipmentMode: kind.type % 2 ? 1 : 2, customerReferenceNo: reference,
    goodsDescription: `四类完整导出中文,"引号"\n第二行 ${kind.code}`, internalReferenceNo: '=SUM(1,2)',
    totalPackages: 2, totalGrossWeightKg: 12.345, totalVolumeCbm: 1.234,
    cargoValue: '15362.5', cargoCurrency: 'CNY', insurancePremium: '125.67', insuranceCurrency: 'CNY', contractNo: `HGJ-CONTRACT-${kind.code}`, factoryName: `四类验收工厂-${kind.code}`,
    orderDate: `${today}T00:30:00+08:00`, receivedAt: `${today}T08:15:00+08:00`, cargoReadyAt: `${today}T09:45:00+08:00`, bookingNotes: `${kind.code}订舱独立备注`, allocationNotes: `${kind.code}配舱独立备注`, operationNotes: `${kind.code}操作独立备注`, notes: `${kind.code}普通独立备注`, etd, eta,
    personnelAssignments: [{ role: 2, userId: restricted.find((item) => item.code === kind.code).user.id }, ...[3, 4].map((role) => ({ role, userId: me.id }))],
    ...(kind.air ? { airExport: { documentMode: 'DIRECT', master: { masterNo: masterNumber(stamp + index), airlineId: airline.id,
      originAirportId: airports[0].id, destinationAirportId: airports[1].id, flightNo: 'QX101', etd, eta } } }
      : { shipmentType: 2, shippingLineId: shippingLine.id, seaMasterBill: { masterNo: `HGJ${kind.code}${stamp}${index}` }, seaDocument: { documentStructure: 2 } }),
    ...extra,
  };
}
const facts = [];
for (const kind of orderKinds) {
  console.log(`[HGJ] ${kind.code}真实业务、费用与财务链准备`);
  let order = (await post('/api/v1/orders', orderInput(kind, `HGJ-LIFECYCLE-${kind.code}-${stamp}`))).data;
  const reread = (await request(`/api/v1/orders/${order.id}`)).data;
  assert.equal(reread.paymentTerm, 2); assert.equal(reread.factoryName, `四类验收工厂-${kind.code}`); assert.equal(reread.contractNo, `HGJ-CONTRACT-${kind.code}`);
  assert.equal(Number(reread.cargoValue), 15362.5); assert.equal(Number(reread.insurancePremium), 125.67);
  assert.equal(reread.operationNotes, `${kind.code}操作独立备注`); assert.equal(reread.bookingNotes, `${kind.code}订舱独立备注`);
  assert.equal(reread.allocationNotes, `${kind.code}配舱独立备注`); assert.equal(reread.notes, `${kind.code}普通独立备注`);
  assert.notEqual(reread.receivedAt, reread.cargoReadyAt);
  let dangerousFields;
  if (kind.air) {
    // 每次使用刚成功提交并重读的精确版本；省略与清空各走真实HTTP，不靠货类推断。
    order = reread;
    const versions = [order.version];
    const updateDangerousAndRead = async (patch) => {
      const previousVersion = order.version;
      const updated = (await put(`/api/v1/orders/${order.id}`, { id: order.id, expectedVersion: previousVersion, ...patch })).data;
      assert(updated?.version && BigInt(updated.version) > BigInt(previousVersion), `${kind.code}成功更新必须返回新版本`);
      const reloaded = (await request(`/api/v1/orders/${order.id}`)).data;
      assert.equal(reloaded.version, updated.version, `${kind.code}重读版本与本次写入精确一致`);
      versions.push(reloaded.version);
      order = reloaded;
      return reloaded;
    };
    const declarationCutoffAt = `${today}T09:30:00+08:00`;
    const assertDangerousPreserved = (value) => {
      assert.equal(value.unNumber, '3481'); assert.equal(value.hazardClass, '9'); assert.equal(value.dangerousPackingGroup, 2);
      assert.equal(Date.parse(value.declarationCutoffAt), Date.parse(declarationCutoffAt));
    };
    assertDangerousPreserved(await updateDangerousAndRead({ unNumber: '3481', hazardClass: '9', declarationCutoffAt, dangerousPackingGroup: 2 }));
    const preserved = await updateDangerousAndRead({ specialRequirements: `${kind.code}仅更新普通文本，危险品字段省略` });
    assertDangerousPreserved(preserved);
    assert.equal(preserved.specialRequirements, `${kind.code}仅更新普通文本，危险品字段省略`);
    const cleared = await updateDangerousAndRead({ unNumber: '', hazardClass: '', declarationCutoffAt: '', dangerousPackingGroup: 0 });
    assert.equal(cleared.unNumber ?? '', ''); assert.equal(cleared.hazardClass ?? '', ''); assert.equal(cleared.declarationCutoffAt ?? '', '');
    assert.equal(cleared.dangerousPackingGroup, undefined, `${kind.code}PG显式零应清空optional字段`);
    dangerousFields = { set: true, omittedPreserved: true, explicitEmptyCleared: true, versions };
    console.log(`[HGJ] ${kind.code}危险品真实HTTP设置/省略保留/显式空及PG0清空通过`);
  }
  const abnormal = (await post(`/api/v1/orders/${order.id}/abnormal-cases`, { orderId: order.id, abnormalCaseId: abnormalType.id })).data;
  const fees = [];
  for (const [direction, amount] of [[1, '100'], [2, '40']]) fees.push((await post(`/api/v1/orders/${order.id}/fees`, { orderId: order.id, direction, settlementPartyId: partner.id,
    quantity: '1', unitPrice: amount, currency: 'CNY', feeSettingId: feeSetting.id, billingUnitId: unit.id, taxInclusive: true, idempotencyKey: `hgj-fee-${kind.code}-${direction}-${stamp}` })).data);
  assert(fees.every((fee) => fee.exchangeRateDate === businessDate), `${kind.code}费用取出口ETD/进口ETA中国自然日`);
  const todo = await request('/api/v1/workbench/my-unbilled-fees?page=1&pageSize=200');
  assert(fees.every((fee) => todo.data?.some((item) => item.feeId === fee.id && item.businessType === kind.code)), `${kind.code}工作台未建账待办必须存在`);
  const abnormalTodo = await request('/api/v1/workbench/my-open-abnormal-cases?page=1&pageSize=200');
  assert(abnormalTodo.data?.some((item) => item.orderId === order.id && item.businessType === kind.code), `${kind.code}异常待办存在`);
  await post(`/api/v1/orders/${order.id}/abnormal-cases/${abnormal.id}/resolve`, { orderId: order.id, id: abnormal.id });
  if (kind.direction === 2) {
    const handling = (await request(`/api/v1/orders/${order.id}/import-handling`)).data;
    await put(`/api/v1/orders/${order.id}/import-handling`, { expectedVersion: handling.orderVersion, steps: handling.steps.map((step) => ({ ...step, undertaken: step.kind === 2 || step.kind === 5,
      ...(step.kind === 2 || step.kind === 5 ? { occurredAt: `${today}T09:00:00+08:00` } : {}) })) });
  }
  order = (await request(`/api/v1/orders/${order.id}`)).data;
  const staleVersion = order.version;
  // 海运普通资料更新沿现行完整表单合同带回既有主单身份，不更换主单。
  const draftContext = kind.air ? {} : { seaMasterBill: { masterNo: order.seaMasterBill.masterNo } };
  order = (await put(`/api/v1/orders/${order.id}`, { id: order.id, expectedVersion: staleVersion, ...draftContext, specialRequirements: `${kind.code}真实旧正版本冲突夹具` })).data;
  assert(BigInt(order.version) > BigInt(staleVersion), `${kind.code}版本冲突前置成功更新`);
  const stale = await raw(`/api/v1/orders/${order.id}`, { method: 'PUT', body: JSON.stringify({ id: order.id, expectedVersion: staleVersion, ...draftContext, notes: '不应写入' }) });
  assert.equal(stale.response.status, 409, `${kind.code}过期版本拒绝`);
  const afterConflict = (await request(`/api/v1/orders/${order.id}`)).data;
  assert.equal(afterConflict.notes, `${kind.code}普通独立备注`, `${kind.code}冲突请求不得写入`);
  assert.equal(afterConflict.version, order.version, `${kind.code}冲突请求不得增加版本`);
  order = (await post(`/api/v1/orders/${order.id}/closure`, { id: order.id, expectedVersion: order.version, targetStatus: 2, reason: '四类真实闭环验收完结' })).data;
  const lockState = (await request(`/api/v1/orders/${order.id}/lock-state`)).data;
  assert.equal(lockState.orderVersion, order.version, `${kind.code}审核基线与刚完成业务版本一致`);
  assert(lockState.canLock, `${kind.code}审核可用：${JSON.stringify(lockState.lockBlockedReasons)}`);
  const lock = await post(`/api/v1/orders/${order.id}/lock`, { orderId: order.id, expectedOrderVersion: lockState.orderVersion,
    expectedFeeContentVersion: lockState.feeContentVersion, feeContentFingerprint: lockState.feeContentFingerprint,
    ...(kind.air ? { airDocumentFingerprint: lockState.airDocumentFingerprint } : {}), idempotencyKey: `hgj-lock-${kind.code}-${stamp}` });
  assert.equal(lock.data?.state?.isLocked, true, `${kind.code}人工审核锁成功`);
  const billFacts = [];
  for (let i = 0; i < fees.length; i++) {
    const fee = fees[i]; const direction = i === 0 ? 'RECEIVABLE' : 'PAYABLE';
    const common = { organizationId: companyA.id, feeIds: [fee.id], groupingPolicy: { mode: 1, splitByOrder: true } };
    const initial = await post('/api/v1/finance/bill-batches/preview', common);
    const preview = await post('/api/v1/finance/bill-batches/preview', { ...common, groupConfigs: initial.data.map((group) => ({ groupKey: group.groupKey, billDate: today, settlementAccountId: accounts[direction].id })) });
    const batch = requireSuccessfulBillBatch(await post('/api/v1/finance/bill-batches', { organizationId: companyA.id, groupingPolicy: common.groupingPolicy, idempotencyKey: `hgj-bill-${kind.code}-${direction}-${stamp}`,
      groups: preview.data.map((group) => ({ groupKey: group.groupKey, groupToken: group.groupToken, feeIds: [fee.id], billDate: today, paymentTermsDays: 30, settlementAccountId: accounts[direction].id, statementTitle: `${kind.code}四类财务验收` })) }), 1);
    let bill = batch.bills[0];
    const agingBefore = await request(`/api/v1/finance/reports/aging?organizationId=${companyA.id}&direction=${direction}&keyword=${bill.billNo}&page=1&pageSize=200`);
    assert.equal(Number(agingBefore.total), 1, `${kind.code}${direction}未清账龄真实存在`);
    const cashflow = (await post('/api/v1/finance/cashflows', { organizationId: companyA.id, direction, settlementPartyId: partner.id, currency: 'CNY', amount: bill.totalAmount,
      transactionDate, ourAccount: `HGJ-${kind.code}-${stamp}`, paymentMethod: 'BANK_TRANSFER', idempotencyKey: `hgj-cash-${kind.code}-${direction}-${stamp}` })).data;
    const confirmed = (await post(`/api/v1/finance/cashflows/${cashflow.id}/confirm`, { id: cashflow.id, expectedVersion: cashflow.version })).data;
    // 先核销再发票/免票，证明没有新增先开票的强制前置。
    const verification = (await post('/api/v1/finance/verifications', { allocations: [{ cashflowId: confirmed.id, billId: bill.id, amount: bill.totalAmount }], verificationDate: transactionDate,
      idempotencyKey: `hgj-verify-${kind.code}-${direction}-${stamp}`, note: '先真实收付核销，后补发票适用事实' })).data;
    bill = (await request(`/api/v1/finance/bills/${bill.id}`)).data;
    assert.equal(Number(bill.unverifiedAmount), 0);
    let invoiceId;
    if (direction === 'RECEIVABLE') {
      const registered = await post('/api/v1/finance/invoices/register', { idempotencyKey: `hgj-invoice-${kind.code}-${stamp}`, records: [invoiceRegistrationForBills([bill], { invoiceProfileId: profile.id, taxInvoiceNo: `HGJ-TAX-${kind.code}-${stamp}`, invoiceDate: transactionDate })] });
      invoiceId = registered.data[0].id;
    } else await post(`/api/v1/finance/bills/${bill.id}/invoice-exemption`, { id: bill.id, expectedVersion: bill.version, reason: '隔离验收：本笔应付按适用免票事实结清' });
    bill = (await request(`/api/v1/finance/bills/${bill.id}`)).data;
    assert.equal(bill.completed, true, `${kind.code}${direction}账单核销与发票事实完成`);
    const agingAfter = await request(`/api/v1/finance/reports/aging?organizationId=${companyA.id}&direction=${direction}&keyword=${bill.billNo}&page=1&pageSize=200`);
    assert.equal(Number(agingAfter.total ?? 0), 0, `${kind.code}${direction}结清后不计当前账龄`);
    billFacts.push({ direction, billId: bill.id, billNo: bill.billNo, amount: bill.totalAmount, invoiceId, invoiceExempt: direction === 'PAYABLE', cashflowId: confirmed.id,
      transactionDate: confirmed.transactionDate, verificationId: verification.id, completed: bill.completed });
  }
  const commissionInput = { verificationId: billFacts[0].verificationId, employeeId: me.id, personnelRole: 'SALES', organizationId: companyA.id };
  const preview = (await post('/api/v1/finance/commissions/preview', commissionInput)).data;
  assert.equal(Number(preview.commissionAmount), 6, `${kind.code}100-40零税已实现利润按10%提成=6`);
  const commission = (await post('/api/v1/finance/commissions', { ...commissionInput, idempotencyKey: `hgj-commission-${kind.code}-${stamp}` })).data;
  const confirmedCommission = (await post(`/api/v1/finance/commissions/${commission.id}/confirm`, { id: commission.id, expectedVersion: commission.version })).data;
  const payment = (await post(`/api/v1/finance/commissions/${commission.id}/payment-preview`, { id: commission.id, expectedVersion: confirmedCommission.version })).data;
  const paid = (await post(`/api/v1/finance/commissions/${commission.id}/paid`, { id: commission.id, expectedVersion: confirmedCommission.version, previewToken: payment.previewToken,
    actualAmount: payment.actualAmount, idempotencyKey: `hgj-paid-${kind.code}-${stamp}` })).data;
  assert.equal(paid.status, 3, `${kind.code}提成实际标记支付事实`);
  const profitReport = await request(`/api/v1/finance/reports/profits?organizationId=${companyA.id}&keyword=${order.orderNo}&businessType=${kind.code}&dateFrom=${businessDate}&dateTo=${businessDate}&page=1&pageSize=200`);
  const profit = profitReport.data.find((item) => item.orderId === order.id);
  assert(profit && Number(profit.profitAmount) === 60 && Number(profit.netProfitAmount) === 60, `${kind.code}经营日期/含税未税利润60`);
  const cashflowReport = await request(`/api/v1/finance/reports/cashflows?organizationId=${companyA.id}&dateFrom=${transactionDate}&dateTo=${transactionDate}&ourAccount=HGJ-${kind.code}-${stamp}&page=1&pageSize=200`);
  assert.equal(cashflowReport.data?.length, 2, `${kind.code}实际交易日两笔收付`);
  assert(cashflowReport.data.every((item) => item.transactionDate === transactionDate), '实际现金流不冒充业务日');
  assert.deepEqual(cashflowReport.data.map((item) => item.id).sort(), billFacts.map((item) => item.cashflowId).sort(), `${kind.code}报表必须是本票两笔真实资金流水`);
  for (const direction of ['RECEIVABLE', 'PAYABLE']) {
    const ledger = await request(`/api/v1/finance/fees?organizationId=${companyA.id}&keyword=${encodeURIComponent(kind.air ? order.airExport.master.content.masterNo : order.seaMasterBill.masterNo)}&direction=${direction}&page=1&pageSize=200`);
    assert(ledger.data.some((item) => item.orderId === order.id), `${kind.code}${direction}台账真实主单搜索`);
  }
  facts.push({ ...kind, orderId: order.id, orderNo: order.orderNo, order, businessDate, transactionDate, dangerousFields, bills: billFacts, commissionId: paid.id, commissionAmount: paid.commissionAmount, commissionStatus: paid.status });
  console.log(`[HGJ] ${kind.code}应收/应付、收付核销、登记/免票、提成及报表链通过`);
}

// 四类各205匹配结果，均由专用单次导出读取；不以客户端循环分页拼接。
const csvFixtures = [];
for (const kind of orderKinds) {
  const orders = [];
  for (let index = 0; index < 205; index++) orders.push((await post('/api/v1/orders', orderInput(kind, `HGJCSV-${kind.code}-${String(index).padStart(3, '0')}`))).data);
  const query = new URLSearchParams({ businessType: String(kind.type), numberType: '4', numberKeyword: `HGJCSV-${kind.code}`, sortBy: 'orderNo', sortOrder: 'ascend' });
  const exported = await request(`/api/v1/orders/export?${query}`);
  assert.equal(exported.total, 205, `${kind.code}完整导出超过200条`);
  assert.equal(exported.data.length, 205);
  assert(exported.generatedAt && !Number.isNaN(Date.parse(exported.generatedAt)), '导出生成时间');
  assert(exported.data.every((item) => item.businessType === kind.type && item.organizationId === companyA.id), `${kind.code}方向与公司范围`);
  assert.deepEqual(exported.data.map((item) => item.orderNo), [...exported.data.map((item) => item.orderNo)].sort(), `${kind.code}完整稳定升序`);
  csvFixtures.push({ ...kind, keyword: `HGJCSV-${kind.code}`, expected: 205, firstNo: exported.data[0].orderNo, lastNo: exported.data.at(-1).orderNo,
    orderedNumbers: exported.data.map((item) => item.orderNo),
    firstTimes: { createdAt: exported.data[0].createdAt, etd: exported.data[0].etd, eta: exported.data[0].eta, orderDate: exported.data[0].orderDate } });
  console.log(`[HGJ] ${kind.code}专用导出205条与排序PASS`);
}
const restrictedFacts = [];
for (const identity of restricted) {
  const scoped = await createHgjClient(identity.username, identity.password);
  await scoped.post('/api/v1/auth/switch-organization', { organizationId: companyA.id });
  const identityMe = (await scoped.request('/api/v1/auth/me')).data;
  assert(identityMe.permissionCapabilities.every((item) => !item.key.startsWith('business.order.') || item.key.startsWith(`business.order.${identity.code.toLowerCase()}.`)), '仅一种真实类型权限');
  for (const kind of orderKinds) {
    const target = facts.find((item) => item.code === kind.code);
    const detail = await scoped.raw(`/api/v1/orders/${target.orderId}`);
    const exported = await scoped.raw(`/api/v1/orders/export?businessType=${kind.type}&numberType=4&numberKeyword=HGJCSV-${kind.code}`);
    if (kind.code === identity.code) {
      assert.equal(detail.response.status, 200); assert.equal(exported.response.status, 200);
      assert.equal(exported.body.data.length, 205);
      assert(exported.body.data.every((item) => item.containerNos === undefined && item.attachmentCount === undefined), '无箱/附件权限字段应裁去');
      const write = await scoped.raw(`/api/v1/orders/${target.orderId}`, { method: 'PUT', body: JSON.stringify({ id: target.orderId, expectedVersion: detail.body.data.version, notes: '不应成功' }) });
      assert.equal(write.response.status, 403, '只读身份不得写订单');
    } else { assert.equal(detail.response.status, 403); assert.equal(exported.response.status, 403); }
  }
  const todo = await scoped.request('/api/v1/workbench/my-recent-orders?page=1&pageSize=200');
  assert(todo.data?.every((item) => item.businessType === identity.code), '工作台本人类型交集');
  restrictedFacts.push({ code: identity.code, readOwn: true, foreignTypesForbidden: true, writeForbidden: true, authorizedFieldsOnly: true });
}

await post('/api/v1/auth/switch-organization', { organizationId: companyB.id });
const foreignPartner = (await post('/api/v1/partners', { legalName: '独立公司B客商', roles: [{ type: 1, enabled: true }],
  profile: { countryCode: 'CN', businessTypes: [2] }, assignments: [3, 4].map((role) => ({ role, userId: me.id, organizationId: companyB.id })) })).data;
const foreignOrder = (await post('/api/v1/orders', { ...orderInput(orderKinds[1], `HGJCSV-SI-FOREIGN`), customerId: foreignPartner.id,
  personnelAssignments: [2, 3, 4].map((role) => ({ role, userId: me.id })) })).data;
await post('/api/v1/auth/switch-organization', { organizationId: companyA.id });
assert([403, 404].includes((await raw(`/api/v1/orders/${foreignOrder.id}`)).response.status), '跨公司订单不可读');
const ownExport = await request('/api/v1/orders/export?businessType=2&numberType=4&numberKeyword=HGJCSV-SI');
assert.equal(ownExport.total, 205); assert(!ownExport.data.some((item) => item.id === foreignOrder.id), '导出不串公司');
assert.equal((await raw(`/api/v1/finance/reports/profits?organizationId=${companyB.id}&page=1&pageSize=200`)).response.status, 403, '财务报表不能覆盖当前公司');
await post('/api/v1/auth/switch-organization', { organizationId: systemId });
const notificationTypes = (await request('/api/v1/admin/notification-tests/types')).data;
assert.equal(notificationTypes.canSelectOrder ?? false, false, '系统测试页不具经营订单读取资格');
const notificationPreview = (await request('/api/v1/admin/notification-tests/preview?notificationType=FEE_REVIEW_PENDING')).data;
assert(new URL(notificationPreview.actionUrl).pathname === '/welcome', '系统测试页仅首页示例动作');
assert([400, 403, 404].includes((await raw(`/api/v1/admin/notification-tests/preview?notificationType=FEE_REVIEW_PENDING&orderId=${facts[0].orderId}`)).response.status), '系统预览不能读取真实业务单');
await post('/api/v1/auth/switch-organization', { organizationId: companyA.id });
const privateIdentities = restricted.map(({ code, username, password }) => ({ code, username, password }));
fs.writeFileSync(path.join(artifactDir, 'hgj-private-identities.json'), JSON.stringify(privateIdentities), { mode: 0o600 });
const manifest = { companyA, companyB, facts, csvFixtures, restricted: restrictedFacts, businessDate, transactionDate,
  notification: { exampleOnly: true, canSelectOrder: false, realOrderRejected: true, noSendCalled: true } };
fs.writeFileSync(path.join(artifactDir, 'hgj-cross-module-facts.json'), JSON.stringify(manifest, null, 2), { mode: 0o600 });
console.log('[HGJ] 四类财务/CSV/权限/系统通知示例HTTP实证全部通过；未调用通知发送接口');
