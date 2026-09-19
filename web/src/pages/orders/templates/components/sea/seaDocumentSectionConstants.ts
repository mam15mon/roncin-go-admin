export const DEFAULT_TRANSPORT_TERMS = 'CY - CY';

export const SEA_TRANSPORT_TERM_OPTIONS: { label: string; value: string }[] = [
  // 核心整箱 / 拼箱条款（按业务常用度优先排列）
  { label: 'CY - CY', value: 'CY - CY' },
  { label: 'CFS - CFS', value: 'CFS - CFS' },
  { label: 'CY - CFS', value: 'CY - CFS' },
  { label: 'CFS - CY', value: 'CFS - CY' },
  { label: 'DOOR - DOOR', value: 'DOOR - DOOR' },
  { label: 'DOOR - CY', value: 'DOOR - CY' },
  { label: 'CY - DOOR', value: 'CY - DOOR' },
  { label: 'DOOR - CFS', value: 'DOOR - CFS' },
  { label: 'CFS - DOOR', value: 'CFS - DOOR' },

  // 码头 / 船边 / 装卸管辖条款
  { label: 'CY - FO', value: 'CY - FO' },
  { label: 'CY - LO', value: 'CY - LO' },
  { label: 'CY - HOOK', value: 'CY - HOOK' },
  { label: 'CY - TACKLE', value: 'CY - TACKLE' },
  { label: 'CY - RAMP', value: 'CY - RAMP' },
  { label: 'CY - SHIPS HOOK', value: 'CY - SHIPS HOOK' },
  { label: 'CY - LINER OUT', value: 'CY - LINER OUT' },
  { label: 'CY - FREE OUT', value: 'CY - FREE OUT' },
  { label: 'CFS - FO', value: 'CFS - FO' },
  { label: 'CFS / DDU', value: 'CFS / DDU' },
  { label: 'RAMP - RAMP', value: 'RAMP - RAMP' },
  { label: 'RAMP - CY', value: 'RAMP - CY' },
  { label: 'RAMP - CFS', value: 'RAMP - CFS' },
  { label: 'DOOR - RAMP', value: 'DOOR - RAMP' },
  { label: 'TACKLE - CY', value: 'TACKLE - CY' },
  { label: 'TACKLE - CFS', value: 'TACKLE - CFS' },
  { label: 'DR - LINER OUT', value: 'DR - LINER OUT' },
  { label: 'DR - FREE OUT', value: 'DR - FREE OUT' },
  { label: 'LINER IN - CY', value: 'LINER IN - CY' },
  { label: 'LINER IN - DR', value: 'LINER IN - DR' },
  { label: 'FREE IN - CY', value: 'FREE IN - CY' },
  { label: 'FREE IN - D', value: 'FREE IN - D' },
  { label: 'FEE IN - CY', value: 'FEE IN - CY' },
  { label: 'PIER - PIER', value: 'PIER - PIER' },

  // 空运 / 多式联运延伸条款
  { label: 'AIRPORT - AIRPORT', value: 'AIRPORT - AIRPORT' },
  { label: 'AIR PORT - DOOR', value: 'AIR PORT - DOOR' },
  { label: 'DOOR - AIR PORT', value: 'DOOR - AIR PORT' },
];

export const DEFAULT_FREIGHT_TERMS = 'FREIGHT PREPAID';

export const SEA_FREIGHT_TERM_OPTIONS: { label: string; value: string }[] = [
  { label: 'FREIGHT PREPAID', value: 'FREIGHT PREPAID' },
  { label: 'FREIGHT COLLECT', value: 'FREIGHT COLLECT' },
  {
    label: 'FREIGHT PAYABLE AT DESTINATION',
    value: 'FREIGHT PAYABLE AT DESTINATION',
  },
  { label: 'PAYABLE AT XXX', value: 'PAYABLE AT XXX' },
  { label: '预付', value: '预付' },
  { label: '到付', value: '到付' },
];

export const DEFAULT_BILL_FORM = 'ORIGINAL';

export const SEA_BILL_FORM_OPTIONS: { label: string; value: string }[] = [
  { label: 'ORIGINAL (正本)', value: 'ORIGINAL' },
  { label: 'SEAWAY BILL (海运单)', value: 'SEAWAY BILL' },
  { label: 'COPY (副本/电放件)', value: 'COPY' },
  { label: 'MEMORANDUM (备忘)', value: 'MEMORANDUM' },
];

export const DEFAULT_RELEASE_TYPE = '电放';

export const SEA_RELEASE_TYPE_OPTIONS: { label: string; value: string }[] = [
  { label: '电放 (Telex Release)', value: '电放' },
  { label: '正本 (Original)', value: '正本' },
  { label: '海运单 (Sea Waybill)', value: '海运单' },
  { label: '异地放单', value: '异地放单' },
];

export const SEA_DOCUMENT_CONTENT_FIELDS: (keyof API.SeaBillContent)[] = [
  'shipperText',
  'consigneeText',
  'notifyPartyText',
  'secondNotifyPartyText',
  'marksText',
  'goodsDescriptionText',
  'packageCount',
  'packageUnit',
  'grossWeightKg',
  'volumeCbm',
  'freightTerms',
  'transportTerms',
  'billForm',
  'releaseType',
  'clauses',
  'foreignAgentText',
];
