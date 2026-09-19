export const resourceTabs = [
  { key: 'addresses', label: '地址管理', type: 1 },
  { key: 'remarks', label: '备注管理', type: 2 },
  { key: 'images', label: '图片管理', type: 3 },
  { key: 'tags', label: '标签管理', type: 4 },
  { key: 'consignees', label: '收货人管理', type: 6 },
  { key: 'shippers', label: '发货人管理', type: 5 },
  { key: 'notify-parties', label: '通知人管理', type: 7 },
] as const;

export type ResourceTab = (typeof resourceTabs)[number];

export const remarkTypes = [
  '订舱备注',
  '配舱备注',
  '运输委托备注',
  '订单备注',
  '提单备注',
  '客户备注',
  '供应商备注',
  '国外代理备注',
  '报价备注',
  '舱单备注',
  '装箱单备注',
  '操作备注',
  '提成备注',
  '仓储备注',
].map((label, index) => ({ label, value: index + 1 }));
export const addressTypes = [
  { label: '拆/装箱地址', value: 1 },
  { label: '提货地址', value: 2 },
  { label: '送货地址', value: 3 },
];
export const partyTypes = new Set([5, 6, 7]);
export const importHeaders = [
  '简称',
  '企业名称',
  '企业代码',
  '地址',
  '国家代码',
  '联系人',
  '电话',
  '邮箱',
  '税号',
  'AEO代码',
];
export const importConflictFieldLabels: Record<string, string> = {
  business_code: '企业代码',
  company_name: '企业名称',
};

export type EditorValues = {
  shortName: string;
  enabled: boolean;
  sortOrder?: number;
  partnerIds?: string[];
  contactName?: string;
  contactPhone?: string;
  countryCode?: string;
  provinceCode?: string;
  cityCode?: string;
  districtCode?: string;
  addressDetail?: string;
  addressRemark?: string;
  addressTypes?: number[];
  remarkType?: number;
  content?: string;
  companyName?: string;
  businessCode?: string;
  partyAddress?: string;
  email?: string;
  taxIdentifier?: string;
  aeoCode?: string;
  customDisplay?: boolean;
  displayContent?: string;
  partyRemark?: string;
  groupId?: string;
  assigneeIds?: string[];
};

export async function imageChecksum(file: Blob): Promise<string> {
  const digest = await crypto.subtle.digest(
    'SHA-256',
    await file.arrayBuffer(),
  );
  return btoa(String.fromCharCode(...new Uint8Array(digest)));
}

export function formatStorageSize(value?: string): string {
  const bytes = Number(value ?? 0);
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KiB`;
  if (bytes < 1024 * 1024 * 1024)
    return `${(bytes / 1024 / 1024).toFixed(2)} MiB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GiB`;
}

export function parseCSV(content: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let field = '';
  let quoted = false;
  for (let index = 0; index < content.length; index += 1) {
    const character = content[index];
    if (character === '"') {
      if (quoted && content[index + 1] === '"') {
        field += '"';
        index += 1;
      } else quoted = !quoted;
    } else if (character === ',' && !quoted) {
      row.push(field.trim());
      field = '';
    } else if ((character === '\n' || character === '\r') && !quoted) {
      if (character === '\r' && content[index + 1] === '\n') index += 1;
      row.push(field.trim());
      if (row.some(Boolean)) rows.push(row);
      row = [];
      field = '';
    } else field += character;
  }
  if (quoted) throw new Error('CSV 文件存在未闭合的双引号');
  row.push(field.trim());
  if (row.some(Boolean)) rows.push(row);
  return rows;
}

export function parseImportFile(
  content: string,
  resourceType: number,
): API.EnterpriseResourceInput[] {
  const rows = parseCSV(content.replace(/^\uFEFF/, ''));
  if (
    !rows.length ||
    importHeaders.some((header, index) => rows[0][index] !== header)
  )
    throw new Error(`CSV 表头必须为：${importHeaders.join(',')}`);
  return rows.slice(1).map((values) => {
    const [
      shortName,
      companyName,
      businessCode,
      address,
      countryCode = 'CN',
      contactName,
      contactPhone,
      email,
      taxIdentifier,
      aeoCode,
    ] = values;
    return {
      resourceType,
      shortName: shortName ?? '',
      enabled: true,
      sortOrder: 0,
      party: {
        companyName,
        businessCode,
        address,
        countryCode: countryCode || 'CN',
        contactName,
        contactPhone,
        email,
        taxIdentifier,
        aeoCode,
      },
    };
  });
}
