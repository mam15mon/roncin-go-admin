import {
  DownloadOutlined,
  SaveOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import {
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Modal,
  Radio,
  Row,
  Segmented,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { createStyles } from 'antd-style';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  FormRow,
  PackageCountInput,
  ProFormSearchableSelect,
} from '@/components/ui';

const useVerticalFormStyles = createStyles(({ css }) => ({
  verticalFields: css`
    .ant-form-item {
      width: 100% !important;
    }
    .ant-form-item-row {
      flex-direction: column !important;
    }
    .ant-form-item-label,
    .ant-form-item-label.ant-form-item-label-right,
    .ant-form-item-label.ant-form-item-label-left {
      display: flex !important;
      text-align: left !important;
      justify-content: flex-start !important;
      align-items: center !important;
      flex: 0 0 auto !important;
      width: 100% !important;
      max-width: 100% !important;
      padding-bottom: 4px !important;
    }
    .ant-form-item-label > label {
      height: auto !important;
      font-size: 13px !important;
      font-weight: 500 !important;
      color: rgba(0, 0, 0, 0.88) !important;
      justify-content: flex-start !important;
      text-align: left !important;
      margin-left: 0 !important;
    }
    .ant-form-item-label > label::after {
      display: none !important;
    }
    .ant-form-item-control {
      flex: 1 1 100% !important;
      max-width: 100% !important;
      width: 100% !important;
    }
    .ant-form-item-control-input {
      width: 100% !important;
    }
    .ant-form-item-control-input-content {
      width: 100% !important;
    }
    .ant-input,
    .ant-input-number,
    .ant-picker {
      width: 100% !important;
    }
  `,
}));

import {
  OrderBusinessType,
  OrderReleasePodStatus,
  SeaDocumentStructure,
  SeaDocumentType,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import { orderReleasePodServiceListReleasePods } from '@/services/roncin/orderReleasePodService';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  seaDocumentServiceExecuteChangeSeaDocumentMode,
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServicePreviewChangeSeaDocumentMode,
  seaDocumentServiceUpdateSeaHouseBill,
  seaDocumentServiceUpdateSeaMasterBillContent,
} from '@/services/roncin/seaDocumentService';
import { searchPartnerOptions } from '@/utils/options';
import { generateUUID } from '@/utils/uuid';
import { RELEASE_PODS_CHANGED_EVENT } from '../../../release-pod-events';
import type { TemplateProps, TemplateSection } from '../../types';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from './SeaExternalConfirmationFields';

const { Text } = Typography;

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

export function SeaBillContentFormFields({
  namePathPrefix,
  disabled = false,
}: {
  namePathPrefix: (string | number)[];
  disabled?: boolean;
}) {
  const { styles } = useVerticalFormStyles();
  const form = Form.useFormInstance();
  const { message } = App.useApp();
  const [notifyTab, setNotifyTab] = useState<'notify' | 'secondNotify'>(
    'notify',
  );
  const [importingShipper, setImportingShipper] = useState(false);
  const [importingAgent, setImportingAgent] = useState(false);

  const secondNotifyValue = Form.useWatch(
    [...namePathPrefix, 'secondNotifyPartyText'],
    form,
  );
  const hasSecondNotify = Boolean(
    secondNotifyValue && String(secondNotifyValue).trim(),
  );

  const handleImportShipperFromCustomer = async () => {
    if (!form) return;
    const customerId = form.getFieldValue('customerId');
    if (!customerId) {
      message.warning(
        '当前订单尚未选择委托客户，请先在「业务信息」中选择委托客户',
      );
      return;
    }
    try {
      setImportingShipper(true);
      const res = await partnerServiceGetPartner({ id: customerId });
      const partner = res.data;
      if (!partner) {
        message.warning('未获取到该委托客户的档案信息');
        return;
      }
      const lines: string[] = [];
      const name = partner.profile?.nameEn || partner.legalName;
      if (name) lines.push(name);
      const address =
        partner.profile?.addressEn ||
        partner.registeredAddress ||
        partner.profile?.addressDetail;
      if (address) lines.push(address);
      if (partner.contacts && partner.contacts.length > 0) {
        const c = partner.contacts[0];
        const contactParts = [c.name, c.phone, c.email].filter(Boolean);
        if (contactParts.length > 0) {
          lines.push(`TEL/CONTACT: ${contactParts.join(' ')}`);
        }
      }
      const text = lines.join('\n');
      if (!text) {
        message.warning('该委托客户未维护英文名称或地址信息');
        return;
      }
      form.setFieldValue([...namePathPrefix, 'shipperText'], text);
      message.success('已从委托客户带入发货人抬头信息');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '获取委托客户信息失败';
      message.error(msg);
    } finally {
      setImportingShipper(false);
    }
  };

  const handleImportForeignAgent = async () => {
    if (!form) return;
    const foreignAgentId = form.getFieldValue('foreignAgentId');
    if (!foreignAgentId) {
      message.warning(
        '当前订单尚未选择国外代理，请先在「基础信息」中选择国外代理',
      );
      return;
    }
    try {
      setImportingAgent(true);
      const res = await partnerServiceGetPartner({ id: foreignAgentId });
      const partner = res.data;
      if (!partner) {
        message.warning('未获取到该国外代理的档案信息');
        return;
      }
      const lines: string[] = [];
      const name = partner.profile?.nameEn || partner.legalName;
      if (name) lines.push(name);
      const address =
        partner.profile?.addressEn ||
        partner.registeredAddress ||
        partner.profile?.addressDetail;
      if (address) lines.push(address);
      if (partner.contacts && partner.contacts.length > 0) {
        const c = partner.contacts[0];
        const contactParts = [c.name, c.phone, c.email].filter(Boolean);
        if (contactParts.length > 0) {
          lines.push(`TEL/CONTACT: ${contactParts.join(' ')}`);
        }
      }
      const text = lines.join('\n');
      if (!text) {
        message.warning('该国外代理未维护英文名称或地址信息');
        return;
      }
      form.setFieldValue([...namePathPrefix, 'foreignAgentText'], text);
      message.success('已从订单国外代理带入抬头信息');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '获取国外代理信息失败';
      message.error(msg);
    } finally {
      setImportingAgent(false);
    }
  };

  const handleImportFromCargoInfo = () => {
    if (!form) return;
    const goodsDescription = form.getFieldValue('goodsDescription');
    const totalPackages = form.getFieldValue('totalPackages');
    const totalPackageUnit = form.getFieldValue('totalPackageUnit');
    const totalGrossWeightKg = form.getFieldValue('totalGrossWeightKg');
    const totalVolumeCbm = form.getFieldValue('totalVolumeCbm');

    if (
      !goodsDescription &&
      totalPackages === undefined &&
      !totalPackageUnit &&
      totalGrossWeightKg === undefined &&
      totalVolumeCbm === undefined
    ) {
      message.warning('订单货物信息尚未录入件重尺或品名');
      return;
    }

    const updates: Record<string, unknown> = {};
    if (goodsDescription) updates.goodsDescriptionText = goodsDescription;
    if (totalPackages !== undefined) updates.packageCount = totalPackages;
    if (totalPackageUnit) updates.packageUnit = totalPackageUnit;
    if (totalGrossWeightKg !== undefined)
      updates.grossWeightKg = totalGrossWeightKg;
    if (totalVolumeCbm !== undefined) updates.volumeCbm = totalVolumeCbm;

    const currentContent = (form.getFieldValue(namePathPrefix) ?? {}) as Record<
      string,
      unknown
    >;
    form.setFieldValue(namePathPrefix, {
      ...currentContent,
      ...updates,
    });
    message.success('已从订单货物信息带入品名、件数、单位及毛重体积');
  };

  const CLAUSE_TEXT = 'SHIPPER LOAD,COUNT AND SEAL';
  const goodsDescriptionValue = Form.useWatch(
    [...namePathPrefix, 'goodsDescriptionText'],
    form,
  );
  const hasClause = Boolean(
    goodsDescriptionValue &&
      String(goodsDescriptionValue).includes(CLAUSE_TEXT),
  );

  const handleToggleClause = (targetChecked?: boolean) => {
    if (!form) return;
    const current =
      (form.getFieldValue([
        ...namePathPrefix,
        'goodsDescriptionText',
      ]) as string) || '';
    const isPresent = current.includes(CLAUSE_TEXT);
    const shouldAdd = targetChecked !== undefined ? targetChecked : !isPresent;

    if (shouldAdd && !isPresent) {
      const trimmed = current.trim();
      const next = trimmed ? `${trimmed}\n${CLAUSE_TEXT}` : CLAUSE_TEXT;
      form.setFieldValue([...namePathPrefix, 'goodsDescriptionText'], next);
    } else if (!shouldAdd && isPresent) {
      const next = current
        .replace(CLAUSE_TEXT, '')
        .replace(/\n\s*\n/g, '\n')
        .trim();
      form.setFieldValue([...namePathPrefix, 'goodsDescriptionText'], next);
    }
  };

  const handleImportMeasurementsFromCargo = () => {
    if (!form) return;
    const totalPackages = form.getFieldValue('totalPackages');
    const totalPackageUnit = form.getFieldValue('totalPackageUnit');
    const totalGrossWeightKg = form.getFieldValue('totalGrossWeightKg');
    const totalVolumeCbm = form.getFieldValue('totalVolumeCbm');

    if (
      totalPackages === undefined &&
      !totalPackageUnit &&
      totalGrossWeightKg === undefined &&
      totalVolumeCbm === undefined
    ) {
      message.warning('订单尚未录入委托件重尺数据');
      return;
    }

    const updates: Record<string, unknown> = {};
    if (totalPackages !== undefined) updates.packageCount = totalPackages;
    if (totalPackageUnit) updates.packageUnit = totalPackageUnit;
    if (totalGrossWeightKg !== undefined)
      updates.grossWeightKg = totalGrossWeightKg;
    if (totalVolumeCbm !== undefined) updates.volumeCbm = totalVolumeCbm;

    const currentContent = (form.getFieldValue(namePathPrefix) ?? {}) as Record<
      string,
      unknown
    >;
    form.setFieldValue(namePathPrefix, {
      ...currentContent,
      ...updates,
    });
    message.success('已将委托件重尺同步至提单实际数据');
  };

  return (
    <div>
      <div className={styles.verticalFields}>
        {/* 发货人/收货人 + 通知人/外国代理：FormRow row-2 对半，两行共享列边界 */}
        <FormRow cols={2}>
          <div style={{ marginBottom: 24 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 6,
                minHeight: 24,
              }}
            >
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 500,
                  color: 'rgba(0, 0, 0, 0.88)',
                }}
              >
                发货人 (Shipper)
              </span>
              {!disabled && (
                <Button
                  type="link"
                  size="small"
                  icon={<DownloadOutlined />}
                  loading={importingShipper}
                  onClick={handleImportShipperFromCustomer}
                  style={{
                    padding: 0,
                    height: 'auto',
                    fontSize: 12,
                    fontWeight: 'normal',
                  }}
                >
                  从委托客户带入
                </Button>
              )}
            </div>
            <ProFormTextArea
              name={[...namePathPrefix, 'shipperText']}
              placeholder="请输入发货人英文名称与详细地址"
              disabled={disabled}
              fieldProps={{ rows: 3 }}
              noStyle
            />
          </div>
          <div style={{ marginBottom: 24 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 6,
                minHeight: 24,
              }}
            >
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 500,
                  color: 'rgba(0, 0, 0, 0.88)',
                }}
              >
                收货人 (Consignee)
              </span>
              {!disabled && (
                <Space size={4}>
                  <Tag
                    style={{
                      cursor: 'pointer',
                      userSelect: 'none',
                      margin: 0,
                      fontSize: 11,
                      padding: '0 4px',
                    }}
                    onClick={() =>
                      form?.setFieldValue(
                        [...namePathPrefix, 'consigneeText'],
                        'TO ORDER',
                      )
                    }
                  >
                    + TO ORDER
                  </Tag>
                  <Tag
                    style={{
                      cursor: 'pointer',
                      userSelect: 'none',
                      margin: 0,
                      fontSize: 11,
                      padding: '0 4px',
                    }}
                    onClick={() =>
                      form?.setFieldValue(
                        [...namePathPrefix, 'consigneeText'],
                        'TO ORDER OF SHIPPER',
                      )
                    }
                  >
                    + TO ORDER OF SHIPPER
                  </Tag>
                </Space>
              )}
            </div>
            <ProFormTextArea
              name={[...namePathPrefix, 'consigneeText']}
              placeholder="请输入收货人名称与地址 (TO ORDER 或具体收货人)"
              disabled={disabled}
              fieldProps={{ rows: 3 }}
              noStyle
            />
          </div>

          {/* 通知人与第二通知人 Tab 切换 */}
          <div style={{ marginBottom: 24 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 6,
                minHeight: 24,
              }}
            >
              <Segmented
                size="small"
                value={notifyTab}
                onChange={(val) =>
                  setNotifyTab(val as 'notify' | 'secondNotify')
                }
                options={[
                  {
                    value: 'notify',
                    label: '通知人 (Notify Party)',
                  },
                  {
                    value: 'secondNotify',
                    label: (
                      <Space size={4}>
                        <span>第二通知人 (Second Notify Party)</span>
                        {hasSecondNotify ? (
                          <Tag
                            color="blue"
                            variant="filled"
                            style={{
                              margin: 0,
                              fontSize: 10,
                              lineHeight: '16px',
                              padding: '0 4px',
                            }}
                          >
                            已填写
                          </Tag>
                        ) : null}
                      </Space>
                    ),
                  },
                ]}
              />
              {!disabled && notifyTab === 'notify' && (
                <Tag
                  style={{
                    cursor: 'pointer',
                    userSelect: 'none',
                    margin: 0,
                    fontSize: 11,
                    padding: '0 4px',
                  }}
                  onClick={() =>
                    form?.setFieldValue(
                      [...namePathPrefix, 'notifyPartyText'],
                      'SAME AS CONSIGNEE',
                    )
                  }
                >
                  + SAME AS CONSIGNEE
                </Tag>
              )}
            </div>
            {notifyTab === 'notify' ? (
              <ProFormTextArea
                name={[...namePathPrefix, 'notifyPartyText']}
                placeholder="请输入通知人名称与详细地址 (例如：SAME AS CONSIGNEE)"
                disabled={disabled}
                fieldProps={{ rows: 3 }}
                noStyle
              />
            ) : (
              <ProFormTextArea
                name={[...namePathPrefix, 'secondNotifyPartyText']}
                placeholder="请输入第二通知人名称与详细地址 (选填，多数提单无需填写)"
                disabled={disabled}
                fieldProps={{ rows: 3 }}
                noStyle
              />
            )}
          </div>

          {/* 外国代理 (Foreign Agent) */}
          <div style={{ marginBottom: 24 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 6,
                minHeight: 24,
              }}
            >
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 500,
                  color: 'rgba(0, 0, 0, 0.88)',
                }}
              >
                外国代理 (Foreign Agent)
              </span>
              {!disabled && (
                <Button
                  type="link"
                  size="small"
                  icon={<DownloadOutlined />}
                  loading={importingAgent}
                  onClick={handleImportForeignAgent}
                  style={{
                    padding: 0,
                    height: 'auto',
                    fontSize: 12,
                    fontWeight: 'normal',
                  }}
                >
                  从订单国外代理带入
                </Button>
              )}
            </div>
            <ProFormTextArea
              name={[...namePathPrefix, 'foreignAgentText']}
              placeholder="请输入目的港/国外代理名称、地址与联系方式"
              disabled={disabled}
              fieldProps={{ rows: 3 }}
              noStyle
            />
          </div>
        </FormRow>

        {/* 唛头与货物信息区块 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            margin: '8px 0 12px 0',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <div
              style={{
                width: 3,
                height: 14,
                backgroundColor: '#1677ff',
                borderRadius: 2,
                marginRight: 8,
              }}
            />
            <span style={{ fontSize: 13, fontWeight: 600, color: '#1f2329' }}>
              实际与提单货物描述 (Actual B/L Marks & Cargo)
            </span>
            <Tag color="cyan" style={{ marginLeft: 8 }}>
              提单实际数据
            </Tag>
          </div>
          {!disabled && (
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              onClick={handleImportFromCargoInfo}
              style={{ padding: 0 }}
            >
              从订单货物信息带入 (复制委托数据)
            </Button>
          )}
        </div>

        <Row gutter={[16, 0]}>
          <Col xs={24} lg={12}>
            <div style={{ marginBottom: 24 }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 6,
                  minHeight: 24,
                }}
              >
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 500,
                    color: 'rgba(0, 0, 0, 0.88)',
                  }}
                >
                  唛头 (Marks & Numbers)
                </span>
                {!disabled && (
                  <Tag
                    style={{
                      cursor: 'pointer',
                      userSelect: 'none',
                      margin: 0,
                      fontSize: 11,
                      padding: '0 4px',
                    }}
                    onClick={() =>
                      form?.setFieldValue(
                        [...namePathPrefix, 'marksText'],
                        'N/M',
                      )
                    }
                  >
                    + N/M
                  </Tag>
                )}
              </div>
              <ProFormTextArea
                name={[...namePathPrefix, 'marksText']}
                placeholder="请输入唛头信息 (例如：N/M)"
                disabled={disabled}
                fieldProps={{ rows: 3 }}
                noStyle
              />
            </div>
          </Col>
          <Col xs={24} lg={12}>
            <div style={{ marginBottom: 24 }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 6,
                  minHeight: 24,
                }}
              >
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 500,
                    color: 'rgba(0, 0, 0, 0.88)',
                  }}
                >
                  英文品名 / 提单货描
                </span>
                <Space size={6} align="center">
                  <Checkbox
                    checked={hasClause}
                    disabled={disabled}
                    onChange={(e) => handleToggleClause(e.target.checked)}
                    style={{ fontSize: 12, userSelect: 'none' }}
                  >
                    免责条款
                  </Checkbox>
                  <Tag
                    color={hasClause ? 'blue' : 'default'}
                    style={{
                      cursor: disabled ? 'not-allowed' : 'pointer',
                      userSelect: 'none',
                      margin: 0,
                      fontSize: 11,
                      padding: '0 4px',
                      fontFamily: 'monospace',
                    }}
                    onClick={
                      !disabled
                        ? () => handleToggleClause(!hasClause)
                        : undefined
                    }
                  >
                    SHIPPER LOAD,COUNT AND SEAL
                  </Tag>
                </Space>
              </div>
              <ProFormTextArea
                name={[...namePathPrefix, 'goodsDescriptionText']}
                placeholder="请输入提单打印品名与货物描述"
                disabled={disabled}
                fieldProps={{ rows: 3 }}
                noStyle
              />
            </div>
          </Col>
        </Row>
      </div>

      {/* 委托 vs 实际件重尺对照：标题行右侧放「带入委托件重尺」操作；
          字段按 row-4 四分排三行（委托 / 实际 / 条款），跨行共享列边界 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          margin: '8px 0 12px 0',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 3,
              height: 14,
              backgroundColor: '#1677ff',
              borderRadius: 2,
              marginRight: 8,
            }}
          />
          <span style={{ fontSize: 13, fontWeight: 600, color: '#1f2329' }}>
            委托 vs 实际件重尺对照
          </span>
        </div>
        {!disabled && (
          <Button
            type="link"
            size="small"
            icon={<SwapOutlined />}
            onClick={handleImportMeasurementsFromCargo}
            style={{
              fontSize: 12,
              padding: '0 4px',
              height: 'auto',
              fontWeight: 'normal',
              color: '#1677ff',
            }}
          >
            带入委托件重尺 ↓
          </Button>
        )}
      </div>
      <div style={{ display: 'grid', gap: 12 }}>
        <FormRow cols={4}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              委托总件数
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <PackageCountInput
                countName="totalPackages"
                unitName="totalPackageUnit"
                countPlaceholder="0"
                unitPlaceholder="请选择单位"
                unitWidth={104}
                style={{ width: 210 }}
                disabled={disabled}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              委托总毛重
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormDigit
                name="totalGrossWeightKg"
                placeholder="0"
                disabled={disabled}
                min={0}
                noStyle
                fieldProps={{
                  precision: 3,
                  addonAfter: 'KGS',
                  style: { width: '100%' },
                }}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              委托总体积
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormDigit
                name="totalVolumeCbm"
                placeholder="0"
                disabled={disabled}
                min={0}
                noStyle
                fieldProps={{
                  precision: 3,
                  addonAfter: 'CBM',
                  style: { width: '100%' },
                }}
              />
            </div>
          </div>
        </FormRow>
        <FormRow cols={4}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: '#1677ff',
                fontWeight: 500,
                flexShrink: 0,
              }}
            >
              实际总件数
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <PackageCountInput
                countName={[...namePathPrefix, 'packageCount']}
                unitName={[...namePathPrefix, 'packageUnit']}
                countPlaceholder="0"
                unitPlaceholder="请选择单位"
                unitWidth={104}
                style={{ width: 210 }}
                disabled={disabled}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: '#1677ff',
                fontWeight: 500,
                flexShrink: 0,
              }}
            >
              实际总毛重
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormDigit
                name={[...namePathPrefix, 'grossWeightKg']}
                placeholder="0"
                disabled={disabled}
                min={0}
                noStyle
                fieldProps={{
                  precision: 3,
                  addonAfter: 'KGS',
                  style: { width: '100%' },
                }}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: '#1677ff',
                fontWeight: 500,
                flexShrink: 0,
              }}
            >
              实际总体积
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormDigit
                name={[...namePathPrefix, 'volumeCbm']}
                placeholder="0"
                disabled={disabled}
                min={0}
                noStyle
                fieldProps={{
                  precision: 3,
                  addonAfter: 'CBM',
                  style: { width: '100%' },
                }}
              />
            </div>
          </div>
        </FormRow>
        <FormRow cols={4}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              付款方式
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormSelect
                name={[...namePathPrefix, 'freightTerms']}
                placeholder="请选择付款方式"
                initialValue={DEFAULT_FREIGHT_TERMS}
                disabled={disabled}
                options={SEA_FREIGHT_TERM_OPTIONS}
                noStyle
                fieldProps={{
                  style: { width: '100%' },
                  showSearch: true,
                  allowClear: true,
                  onChange: (val) => {
                    if (form) {
                      const str = String(val ?? '');
                      if (str.includes('COLLECT') || str.includes('到付')) {
                        form.setFieldValue('paymentTerm', 2);
                      } else {
                        form.setFieldValue('paymentTerm', 1);
                      }
                    }
                  },
                  filterOption: (input, option) => {
                    const normInput = input.trim().toLowerCase();
                    const normLabel = String(option?.label ?? '').toLowerCase();
                    const normValue = String(option?.value ?? '').toLowerCase();
                    return (
                      normLabel.includes(normInput) ||
                      normValue.includes(normInput)
                    );
                  },
                }}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              运输条款
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormSelect
                name={[...namePathPrefix, 'transportTerms']}
                placeholder="请选择运输条款"
                initialValue={DEFAULT_TRANSPORT_TERMS}
                disabled={disabled}
                options={SEA_TRANSPORT_TERM_OPTIONS}
                noStyle
                fieldProps={{
                  style: { width: '100%' },
                  showSearch: true,
                  allowClear: true,
                  filterOption: (input, option) => {
                    const normInput = input
                      .replace(/[\s\-_/]/g, '')
                      .toLowerCase();
                    const normLabel = String(option?.label ?? '')
                      .replace(/[\s\-_/]/g, '')
                      .toLowerCase();
                    const normValue = String(option?.value ?? '')
                      .replace(/[\s\-_/]/g, '')
                      .toLowerCase();
                    return (
                      normLabel.includes(normInput) ||
                      normValue.includes(normInput)
                    );
                  },
                }}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              提单形式
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormSelect
                name={[...namePathPrefix, 'billForm']}
                placeholder="请选择提单形式"
                initialValue={DEFAULT_BILL_FORM}
                disabled={disabled}
                options={SEA_BILL_FORM_OPTIONS}
                noStyle
                fieldProps={{
                  style: { width: '100%' },
                  allowClear: true,
                  showSearch: true,
                }}
              />
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{
                width: 68,
                fontSize: 13,
                color: 'rgba(0, 0, 0, 0.85)',
                flexShrink: 0,
              }}
            >
              放单方式
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <ProFormSelect
                name={[...namePathPrefix, 'releaseType']}
                placeholder="请选择放单方式"
                initialValue={DEFAULT_RELEASE_TYPE}
                disabled={disabled}
                options={SEA_RELEASE_TYPE_OPTIONS}
                noStyle
                fieldProps={{
                  style: { width: '100%' },
                  allowClear: true,
                  showSearch: true,
                }}
              />
            </div>
          </div>
        </FormRow>
      </div>

      {/* 提单特别条款：row-2 对半取左格，右格空置 */}
      <div style={{ marginTop: 12 }}>
        <FormRow cols={2}>
          <ProFormTextArea
            name={[...namePathPrefix, 'clauses']}
            label="提单特别条款 (Clauses)"
            placeholder="选填，请输入提单特别条款"
            disabled={disabled}
            fieldProps={{ maxLength: 1000, showCount: true, rows: 2 }}
          />
        </FormRow>
      </div>
    </div>
  );
}

type HouseBillFormKey = 'seaHouseBill' | 'newHouseBill';

function HouseBillIdentityFields({
  fieldKey,
  disabled = false,
}: {
  fieldKey: HouseBillFormKey;
  disabled?: boolean;
}) {
  const form = Form.useFormInstance();
  return (
    <Row gutter={[16, 0]}>
      <Col xs={24} md={8}>
        <ProFormText
          name={[fieldKey, 'houseNo']}
          label="分单号 (HBL No.)"
          placeholder="请输入分单号"
          disabled={disabled}
          layout="vertical"
          rules={[
            { required: true, whitespace: true, message: '分单号不能为空' },
          ]}
          fieldProps={{ maxLength: 128 }}
        />
      </Col>
      <Col xs={24} md={16}>
        <Form.Item
          label="签发主体"
          required
          style={{ marginBottom: 24 }}
          layout="vertical"
        >
          <Form.Item
            name={[fieldKey, 'issuerSource']}
            noStyle
            rules={[{ required: true, message: '请选择签发主体' }]}
          >
            <Radio.Group
              disabled={disabled}
              onChange={() =>
                form.setFieldValue([fieldKey, 'issuerPartnerId'], undefined)
              }
            >
              <Radio
                value={
                  SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION
                }
              >
                本公司
              </Radio>
              <Radio
                value={
                  SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER
                }
              >
                委托单位
              </Radio>
              <Radio
                value={
                  SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
                }
              >
                其他主体
              </Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(previous, current) =>
              previous?.[fieldKey]?.issuerSource !==
              current?.[fieldKey]?.issuerSource
            }
          >
            {({ getFieldValue }) => {
              const issuerSource = getFieldValue([fieldKey, 'issuerSource']);
              if (
                issuerSource ===
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION
              ) {
                return (
                  <div style={{ marginTop: 6 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      由所属公司或总部统一签发
                    </Text>
                  </div>
                );
              }
              if (
                issuerSource ===
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER
              ) {
                return (
                  <div style={{ marginTop: 6 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      使用当前订单委托单位作为签发主体
                    </Text>
                  </div>
                );
              }
              if (
                issuerSource ===
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
              ) {
                return (
                  <div style={{ marginTop: 8 }}>
                    <ProFormSearchableSelect
                      name={[fieldKey, 'issuerPartnerId']}
                      placeholder="请选择签发主体合作伙伴"
                      disabled={disabled}
                      rules={[{ required: true, message: '请选择合作伙伴' }]}
                      request={async ({ keyWords }) =>
                        searchPartnerOptions(keyWords)
                      }
                      fieldProps={{ filterOption: false }}
                    />
                  </div>
                );
              }
              return null;
            }}
          </Form.Item>
        </Form.Item>
      </Col>
      <Col xs={24}>
        <ProFormText
          name={[fieldKey, 'note']}
          label="分单备注"
          placeholder="请输入分单备注"
          disabled={disabled}
          layout="vertical"
          fieldProps={{ maxLength: 500 }}
        />
      </Col>
    </Row>
  );
}

function isTerminalHouseBill(status?: number) {
  return status === SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED;
}

function houseBillStatusPresentation(status?: number) {
  switch (status) {
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_RELEASED:
      return { color: 'success', text: '已签发' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_CONFIRMED:
      return { color: 'blue', text: '已确认' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED:
      return { color: 'error', text: '已作废' };
    default:
      return { color: 'default', text: '草稿' };
  }
}

type ModeChangeFormValues = SeaExternalConfirmationFormValues & {
  reason?: string;
  newHouseBill?: Partial<API.SeaHouseBillInput>;
};

function buildHouseBillInput(
  values?: Partial<API.SeaHouseBillInput>,
): API.SeaHouseBillInput | undefined {
  if (!values) return undefined;
  return {
    id: values.id,
    houseNo: values.houseNo?.trim() ?? '',
    issuerSource:
      values.issuerSource ??
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_UNSPECIFIED,
    issuerPartnerId:
      values.issuerSource ===
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
        ? values.issuerPartnerId
        : undefined,
    note: values.note?.trim() || undefined,
    content: values.content,
    expectedVersion: values.expectedVersion,
  };
}

function isDocumentStructure(
  value: number | undefined,
): value is SeaDocumentStructure {
  return (
    value === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE ||
    value === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
  );
}

function ModeChangePreviewResult({
  preview,
}: {
  preview: API.SeaDocumentModeChangePreview;
}) {
  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <Alert
        showIcon
        type={preview.executable ? 'success' : 'error'}
        title={preview.executable ? '预览通过，可以执行' : '当前变更不可执行'}
        description="离港时间、财务和放货事实只作为影响提示；执行不会自动改写这些下游事实。"
      />
      <Table<API.SeaDocumentFieldDifference>
        size="small"
        rowKey={(row) => `${row.field ?? ''}-${row.label ?? ''}`}
        pagination={false}
        dataSource={preview.differences ?? []}
        columns={[
          { title: '字段', dataIndex: 'label', width: 140 },
          { title: '变更前', dataIndex: 'beforeValue' },
          { title: '变更后', dataIndex: 'afterValue' },
        ]}
      />
      {(preview.impacts?.length ?? 0) > 0 ? (
        <Table<API.SeaDocumentDownstreamImpact>
          size="small"
          rowKey={(row) =>
            `${row.factType ?? ''}-${row.referenceId ?? ''}-${row.referenceNo ?? ''}`
          }
          pagination={false}
          dataSource={preview.impacts ?? []}
          columns={[
            { title: '事实类型', dataIndex: 'factType', width: 130 },
            { title: '编号', dataIndex: 'referenceNo', width: 150 },
            { title: '影响', dataIndex: 'message' },
            {
              title: '结论',
              dataIndex: 'blocksExecution',
              width: 80,
              render: (blocked: boolean) => (
                <Tag color={blocked ? 'error' : 'success'}>
                  {blocked ? '阻断' : '提示'}
                </Tag>
              ),
            },
          ]}
        />
      ) : null}
    </Space>
  );
}

export function SeaDocumentSectionComponent({
  disabled = false,
  isDetail = false,
  onOrderDataChanged,
}: {
  disabled?: boolean;
  isDetail?: boolean;
  onOrderDataChanged?: () => Promise<void> | void;
}) {
  const form = Form.useFormInstance();
  const { message } = App.useApp();
  const access = useAccess();
  const [modeForm] = Form.useForm<ModeChangeFormValues>();
  const [activeTabKey, setActiveTabKey] = useState('mbl');
  const [loadedStructure, setLoadedStructure] =
    useState<SeaDocumentStructure>();
  const [linkVersion, setLinkVersion] = useState('0');
  const [mblDetail, setMblDetail] = useState<API.SeaMasterBillDetail | null>(
    null,
  );
  const [houseBill, setHouseBill] = useState<API.SeaHouseBill | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [releasePods, setReleasePods] = useState<API.OrderReleasePod[]>([]);
  const [releasePodsError, setReleasePodsError] = useState<string | null>(null);
  const [modeModalOpen, setModeModalOpen] = useState(false);
  const [modeTarget, setModeTarget] = useState<SeaDocumentStructure>();
  const [modePreview, setModePreview] =
    useState<API.SeaDocumentModeChangePreview | null>(null);
  const [modePreviewing, setModePreviewing] = useState(false);
  const [modeExecuting, setModeExecuting] = useState(false);
  const [modeIdempotencyKey, setModeIdempotencyKey] = useState('');
  const documentRequestSequenceRef = useRef(0);
  const releasePodRequestSequenceRef = useRef(0);
  const activeOrderIdRef = useRef('');

  const orderIdValue = Form.useWatch('id', form) ?? form.getFieldValue('id');
  const orderId = orderIdValue ? String(orderIdValue) : '';
  const orderVersion =
    Form.useWatch('version', form) ?? form.getFieldValue('version');
  const watchedStructure = Form.useWatch('seaDocumentStructure', form) as
    | number
    | undefined;
  const watchedHouseNo = Form.useWatch(['seaHouseBill', 'houseNo'], form) as
    | string
    | undefined;
  const mblMasterNo = Form.useWatch('seaMasterBillMasterNo', form);
  const docStructure = isDocumentStructure(watchedStructure)
    ? watchedStructure
    : loadedStructure;
  activeOrderIdRef.current = orderId;

  const canReadReleasePods = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'release_pod.read',
  );
  const canChangeMode = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'update',
  );

  const loadReleasePods = useCallback(async () => {
    const requestedOrderId = orderId;
    const requestSequence = ++releasePodRequestSequenceRef.current;
    if (!requestedOrderId || !isDetail || !canReadReleasePods) {
      setReleasePods([]);
      setReleasePodsError(null);
      return;
    }
    try {
      const response = await orderReleasePodServiceListReleasePods({
        orderId: requestedOrderId,
      });
      if (
        requestedOrderId !== activeOrderIdRef.current ||
        requestSequence !== releasePodRequestSequenceRef.current
      )
        return;
      setReleasePodsError(null);
      setReleasePods(response.data ?? []);
    } catch (error: unknown) {
      if (
        requestedOrderId !== activeOrderIdRef.current ||
        requestSequence !== releasePodRequestSequenceRef.current
      )
        return;
      setReleasePods([]);
      setReleasePodsError(
        error instanceof Error ? error.message : '放货记录加载失败',
      );
    }
  }, [canReadReleasePods, isDetail, orderId]);

  const loadOrderDocuments = useCallback(async () => {
    const requestedOrderId = orderId;
    const requestSequence = ++documentRequestSequenceRef.current;
    if (!requestedOrderId || !isDetail) return;
    try {
      const response = await seaDocumentServiceGetSeaOrderDocuments({
        orderId: requestedOrderId,
      });
      if (
        requestedOrderId !== activeOrderIdRef.current ||
        requestSequence !== documentRequestSequenceRef.current
      )
        return;
      if (!response.data) throw new Error('接口未返回海运单证数据');
      const structure = response.data.documentStructure;
      if (!isDocumentStructure(structure))
        throw new Error('海运订单缺少明确的 HOUSE/DIRECT 单证模式');
      const currentHouseBill = response.data.houseBill ?? null;
      if (
        (structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE &&
          !currentHouseBill) ||
        (structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT &&
          currentHouseBill)
      ) {
        throw new Error('海运单证模式与当前 HBL 不一致');
      }

      setFetchError(null);
      setLoadedStructure(structure);
      setLinkVersion(String(response.data.linkVersion ?? '0'));
      setMblDetail(response.data.masterBill ?? null);
      setHouseBill(currentHouseBill);
      form.setFieldValue('seaDocumentStructure', structure);
      const mblContent = response.data.masterBill?.content ?? {};
      const mblTransportTerms =
        mblContent.transportTerms && mblContent.transportTerms.trim() !== ''
          ? mblContent.transportTerms
          : DEFAULT_TRANSPORT_TERMS;
      const mblFreightTerms =
        mblContent.freightTerms && mblContent.freightTerms.trim() !== ''
          ? mblContent.freightTerms
          : DEFAULT_FREIGHT_TERMS;
      form.setFieldValue('seaMasterBillContent', {
        ...mblContent,
        transportTerms: mblTransportTerms,
        freightTerms: mblFreightTerms,
      });
      const hblContent = currentHouseBill?.content ?? {};
      const hblTransportTerms =
        hblContent.transportTerms && hblContent.transportTerms.trim() !== ''
          ? hblContent.transportTerms
          : DEFAULT_TRANSPORT_TERMS;
      const hblFreightTerms =
        hblContent.freightTerms && hblContent.freightTerms.trim() !== ''
          ? hblContent.freightTerms
          : DEFAULT_FREIGHT_TERMS;
      form.setFieldValue(
        'seaHouseBill',
        currentHouseBill
          ? {
              id: currentHouseBill.id,
              houseNo: currentHouseBill.houseNo,
              issuerSource: currentHouseBill.issuerSource,
              issuerPartnerId:
                currentHouseBill.issuerSource ===
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
                  ? currentHouseBill.issuerPartnerId
                  : undefined,
              note: currentHouseBill.note,
              content: {
                ...hblContent,
                transportTerms: hblTransportTerms,
                freightTerms: hblFreightTerms,
              },
              expectedVersion: currentHouseBill.version,
            }
          : undefined,
      );
    } catch (error: unknown) {
      if (
        requestedOrderId !== activeOrderIdRef.current ||
        requestSequence !== documentRequestSequenceRef.current
      )
        return;
      const errorMessage =
        error instanceof Error ? error.message : '获取海运单证信息失败';
      setLoadedStructure(undefined);
      setLinkVersion('0');
      setMblDetail(null);
      setHouseBill(null);
      form.setFieldValue('seaMasterBillContent', {});
      form.setFieldValue('seaHouseBill', undefined);
      setFetchError(errorMessage);
      message.error(errorMessage);
    }
  }, [form, isDetail, message, orderId]);

  useEffect(() => {
    if (!isDetail) return;
    setFetchError(null);
    setReleasePodsError(null);
    setLoadedStructure(undefined);
    setMblDetail(null);
    setHouseBill(null);
    setActiveTabKey('mbl');
    void loadOrderDocuments();
    void loadReleasePods();
  }, [isDetail, loadOrderDocuments, loadReleasePods]);

  useEffect(() => {
    if (docStructure !== SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE)
      setActiveTabKey('mbl');
  }, [docStructure]);

  useEffect(() => {
    const handleChanged = (event: Event) => {
      const changedOrderId = (event as CustomEvent<{ orderId?: string }>).detail
        ?.orderId;
      if (changedOrderId === orderId) void loadReleasePods();
    };
    window.addEventListener(RELEASE_PODS_CHANGED_EVENT, handleChanged);
    return () =>
      window.removeEventListener(RELEASE_PODS_CHANGED_EVENT, handleChanged);
  }, [loadReleasePods, orderId]);

  const relatedReleasePods = (documentType: number, documentId?: string) =>
    documentId
      ? releasePods.filter(
          (item) =>
            item.seaDocumentType === documentType &&
            item.seaDocumentId === documentId,
        )
      : [];

  const renderReleasePods = (items: API.OrderReleasePod[]) => {
    if (!canReadReleasePods) return null;
    if (releasePodsError) {
      return (
        <Alert
          type="error"
          showIcon
          title="关联放货记录加载失败"
          description={releasePodsError}
          style={{ marginTop: 12 }}
        />
      );
    }
    return (
      <Card size="small" title="关联放货记录" style={{ marginTop: 12 }}>
        {items.length === 0 ? (
          <Text type="secondary">暂无关联放货记录</Text>
        ) : (
          <Space direction="vertical" size={4}>
            {items.map((item) => (
              <Space key={item.id} wrap>
                <Text>放货编号：{item.releaseNo || '-'}</Text>
                <Text>回单编号：{item.podNo || '-'}</Text>
                <Tag>
                  {item.status ===
                  OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_RETURNED
                    ? '已回单'
                    : item.status ===
                        OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_SIGNED
                      ? '已签收'
                      : '待签收'}
                </Tag>
              </Space>
            ))}
          </Space>
        )}
      </Card>
    );
  };

  const changeCreateMode = (nextMode: SeaDocumentStructure) => {
    setLoadedStructure(nextMode);
    form.setFieldValue('seaDocumentStructure', nextMode);

    const goodsDescription = form.getFieldValue('goodsDescription');
    const totalPackages = form.getFieldValue('totalPackages');
    const totalPackageUnit = form.getFieldValue('totalPackageUnit');
    const totalGrossWeightKg = form.getFieldValue('totalGrossWeightKg');
    const totalVolumeCbm = form.getFieldValue('totalVolumeCbm');

    const cargoDefaults = {
      ...(goodsDescription ? { goodsDescriptionText: goodsDescription } : {}),
      ...(totalPackages !== undefined ? { packageCount: totalPackages } : {}),
      ...(totalPackageUnit ? { packageUnit: totalPackageUnit } : {}),
      ...(totalGrossWeightKg !== undefined
        ? { grossWeightKg: totalGrossWeightKg }
        : {}),
      ...(totalVolumeCbm !== undefined ? { volumeCbm: totalVolumeCbm } : {}),
    };

    const currentMbl = (form.getFieldValue('seaMasterBillContent') ??
      {}) as Record<string, unknown>;
    form.setFieldValue('seaMasterBillContent', {
      transportTerms: DEFAULT_TRANSPORT_TERMS,
      freightTerms: DEFAULT_FREIGHT_TERMS,
      ...cargoDefaults,
      ...currentMbl,
    });

    if (nextMode === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE) {
      form.setFieldValue('seaHouseBill', {
        content: {
          transportTerms: DEFAULT_TRANSPORT_TERMS,
          freightTerms: DEFAULT_FREIGHT_TERMS,
          ...cargoDefaults,
        },
      });
      setActiveTabKey('hbl');
    } else {
      form.setFieldValue('seaHouseBill', undefined);
      setActiveTabKey('mbl');
    }
  };

  const handleSaveHouseBill = async () => {
    if (!orderId || !isDetail || !houseBill?.id) return;
    try {
      await form.validateFields([
        ['seaHouseBill', 'houseNo'],
        ['seaHouseBill', 'issuerSource'],
        ['seaHouseBill', 'issuerPartnerId'],
      ]);
      const input = buildHouseBillInput(
        form.getFieldValue('seaHouseBill') as
          | Partial<API.SeaHouseBillInput>
          | undefined,
      );
      if (!input) return;
      await seaDocumentServiceUpdateSeaHouseBill(
        { orderId, id: houseBill.id },
        {
          orderId,
          id: houseBill.id,
          expectedVersion: String(houseBill.version ?? ''),
          expectedLinkVersion: linkVersion,
          houseBill: input,
        },
      );
      message.success('分单更新成功');
      await loadOrderDocuments();
    } catch (error: unknown) {
      if (error instanceof Error)
        message.error(error.message || '保存分单失败');
    }
  };

  const handleSaveMblContent = async () => {
    if (!orderId || !isDetail || !mblDetail?.id) return;
    try {
      const content = (form.getFieldValue('seaMasterBillContent') ??
        {}) as API.SeaBillContent;
      await seaDocumentServiceUpdateSeaMasterBillContent(
        { orderId },
        {
          orderId,
          expectedMblVersion: String(mblDetail.version ?? ''),
          content,
        },
      );
      message.success('主单内容保存成功');
      await loadOrderDocuments();
    } catch (error: unknown) {
      message.error(
        error instanceof Error ? error.message : '保存主单内容失败',
      );
    }
  };

  const openModeChange = () => {
    if (!isDocumentStructure(docStructure)) {
      message.error('当前单证模式缺失，请刷新后重试');
      return;
    }
    const targetMode =
      docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
        ? SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
        : SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
    setModeTarget(targetMode);
    setModePreview(null);
    setModeIdempotencyKey(`sea-document-mode:${generateUUID()}`);
    modeForm.resetFields();
    modeForm.setFieldsValue({ confirmedAt: dayjs() });
    setModeModalOpen(true);
  };

  const buildModeChangeHouseBill = (values: ModeChangeFormValues) =>
    modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
      ? buildHouseBillInput(values.newHouseBill)
      : undefined;

  const previewModeChange = async () => {
    if (!orderId || !modeTarget) return;
    setModePreviewing(true);
    try {
      const values = await modeForm.validateFields();
      const response = await seaDocumentServicePreviewChangeSeaDocumentMode(
        { orderId },
        {
          orderId,
          targetMode: modeTarget,
          newHouseBill: buildModeChangeHouseBill(values),
          reason: values.reason?.trim() ?? '',
        },
      );
      if (!response.data) throw new Error('接口未返回模式切换预览');
      setModePreview(response.data);
    } catch (error: unknown) {
      if (error instanceof Error)
        message.error(error.message || '模式切换预览失败');
    } finally {
      setModePreviewing(false);
    }
  };

  const executeModeChange = async () => {
    if (!orderId || !modeTarget || !modePreview?.executable) return;
    setModeExecuting(true);
    try {
      const values = await modeForm.validateFields();
      if (!orderVersion || linkVersion === '0')
        throw new Error('订单或单证版本缺失，请刷新后重试');
      if (
        docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE &&
        (!houseBill?.version || !houseBill.currentVersionId)
      ) {
        throw new Error('当前分单版本缺失，请刷新后重试');
      }
      await seaDocumentServiceExecuteChangeSeaDocumentMode(
        { orderId },
        {
          orderId,
          expectedOrderVersion: String(orderVersion),
          expectedLinkVersion: linkVersion,
          expectedHouseBillVersion:
            docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
              ? String(houseBill?.version ?? '')
              : undefined,
          expectedCurrentVersionId:
            docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
              ? houseBill?.currentVersionId
              : undefined,
          targetMode: modeTarget,
          newHouseBill: buildModeChangeHouseBill(values),
          reason: values.reason?.trim() ?? '',
          confirmation: buildSeaExternalConfirmation(values),
          idempotencyKey: modeIdempotencyKey,
        },
      );
      message.success(
        modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
          ? '已切换为 HOUSE 并建立当前 HBL'
          : '已切换为 DIRECT，原 HBL 已形成作废历史',
      );
      setModeModalOpen(false);
      setModePreview(null);
      await onOrderDataChanged?.();
      await Promise.all([loadOrderDocuments(), loadReleasePods()]);
    } catch (error: unknown) {
      if (error instanceof Error)
        message.error(error.message || '执行模式切换失败');
    } finally {
      setModeExecuting(false);
    }
  };

  const renderStructureTag = () => {
    if (docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT) {
      return <Tag color="success">直单 (DIRECT)</Tag>;
    }
    if (docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE) {
      return <Tag color="processing">分单 (HOUSE)</Tag>;
    }
    return <Tag>请选择单证模式</Tag>;
  };

  const masterBillTab = {
    key: 'mbl',
    label: (
      <span>
        主单 (MBL){' '}
        {mblMasterNo ? (
          <Text type="secondary" style={{ fontSize: 12 }}>
            ({mblMasterNo})
          </Text>
        ) : null}
      </span>
    ),
    children: (
      <Card
        size="small"
        variant="outlined"
        style={{
          background: '#ffffff',
          borderRadius: 6,
          borderColor: '#f0f0f0',
        }}
      >
        {isDetail && mblDetail ? (
          <Row gutter={[16, 8]} style={{ marginBottom: 16 }}>
            <Col xs={24} md={8}>
              <Text type="secondary">主单号：</Text>
              <Text strong>{mblDetail.masterNo}</Text>
            </Col>
            <Col xs={24} md={8}>
              <Text type="secondary">共享订单数：</Text>
              <Tag color="blue">{mblDetail.memberCount ?? 1} 票</Tag>
            </Col>
            <Col xs={24} md={8}>
              <Text type="secondary">主单版本：</Text>
              <Tag>v{mblDetail.version}</Tag>
            </Col>
            <Col span={24} style={{ textAlign: 'right' }}>
              <SeaDocumentHistoryActions
                orderId={orderId}
                orderVersion={String(orderVersion ?? '')}
                documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL}
                documentId={mblDetail.id ?? ''}
                documentNo={mblDetail.masterNo ?? ''}
                documentVersion={String(mblDetail.version ?? '')}
                currentVersionId={mblDetail.currentVersionId}
                documentStatus={mblDetail.status}
                getAmendmentInput={() => ({
                  masterBillContent: (form.getFieldValue(
                    'seaMasterBillContent',
                  ) ?? {}) as API.SeaBillContent,
                })}
                disabled={disabled}
                onSuccess={async () => {
                  await onOrderDataChanged?.();
                  await loadOrderDocuments();
                }}
              />
            </Col>
          </Row>
        ) : null}
        <SeaBillContentFormFields
          namePathPrefix={['seaMasterBillContent']}
          disabled={disabled || mblDetail?.status === 'VOIDED'}
        />
        {renderReleasePods(
          relatedReleasePods(
            SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
            mblDetail?.id,
          ),
        )}
        {isDetail && mblDetail && mblDetail.status !== 'VOIDED' && !disabled ? (
          <div style={{ textAlign: 'right', marginTop: 12 }}>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={handleSaveMblContent}
            >
              保存主单内容
            </Button>
          </div>
        ) : null}
      </Card>
    ),
  };

  const houseBillTab = {
    key: 'hbl',
    label: (
      <span>
        分单 (HBL){' '}
        <Text type="secondary" style={{ fontSize: 12 }}>
          ({watchedHouseNo || houseBill?.houseNo || '未编号'})
        </Text>
      </span>
    ),
    children: (
      <Card
        size="small"
        variant="outlined"
        style={{
          background: '#ffffff',
          borderRadius: 6,
          borderColor: '#f0f0f0',
        }}
      >
        {isDetail && houseBill ? (
          <Row
            justify="space-between"
            align="middle"
            style={{ marginBottom: 16 }}
          >
            <Col>
              <Space>
                <Text strong style={{ fontSize: 15 }}>
                  当前分单
                </Text>
                <Tag
                  color={houseBillStatusPresentation(houseBill.status).color}
                >
                  {houseBillStatusPresentation(houseBill.status).text}
                </Tag>
                {houseBill.version ? <Tag>v{houseBill.version}</Tag> : null}
              </Space>
            </Col>
            <Col>
              {houseBill.id ? (
                <SeaDocumentHistoryActions
                  orderId={orderId}
                  orderVersion={String(orderVersion ?? '')}
                  documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL}
                  documentId={houseBill.id}
                  documentNo={houseBill.houseNo ?? ''}
                  documentVersion={String(houseBill.version ?? '')}
                  currentVersionId={houseBill.currentVersionId}
                  currentHouseBill={houseBill}
                  getAmendmentInput={() => ({
                    houseBill: buildHouseBillInput(
                      form.getFieldValue('seaHouseBill') as
                        | Partial<API.SeaHouseBillInput>
                        | undefined,
                    ),
                  })}
                  disabled={disabled}
                  onSuccess={async () => {
                    await onOrderDataChanged?.();
                    await loadOrderDocuments();
                  }}
                />
              ) : null}
            </Col>
          </Row>
        ) : null}
        <HouseBillIdentityFields
          fieldKey="seaHouseBill"
          disabled={disabled || isTerminalHouseBill(houseBill?.status)}
        />
        <div style={{ marginTop: 12 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>
            提单正文内容
          </Text>
          <SeaBillContentFormFields
            namePathPrefix={['seaHouseBill', 'content']}
            disabled={disabled || isTerminalHouseBill(houseBill?.status)}
          />
        </div>
        {renderReleasePods(
          relatedReleasePods(
            SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
            houseBill?.id,
          ),
        )}
        {isDetail &&
        houseBill &&
        !disabled &&
        !isTerminalHouseBill(houseBill.status) ? (
          <div style={{ textAlign: 'right', marginTop: 12 }}>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={handleSaveHouseBill}
            >
              保存分单
            </Button>
          </div>
        ) : null}
      </Card>
    ),
  };

  const modeChangeReady = Boolean(
    isDetail &&
      orderId &&
      orderVersion &&
      linkVersion !== '0' &&
      isDocumentStructure(docStructure) &&
      !fetchError,
  );

  return (
    <Col span={24}>
      {fetchError ? (
        <Alert
          type="error"
          showIcon
          title="获取单证信息失败"
          description={fetchError}
          style={{ marginBottom: 16 }}
        />
      ) : null}

      <Card
        size="small"
        variant="outlined"
        style={{ marginBottom: 16, borderColor: '#f0f0f0' }}
      >
        {isDetail ? (
          <Row justify="space-between" align="middle" gutter={[12, 12]}>
            <Col>
              <Space size="middle">
                <Text strong>单证模式：</Text>
                {renderStructureTag()}
                {linkVersion !== '0' ? (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    (单证版本 v{linkVersion})
                  </Text>
                ) : null}
              </Space>
            </Col>
            <Col>
              {canChangeMode ? (
                <Button
                  icon={<SwapOutlined />}
                  disabled={!modeChangeReady}
                  onClick={openModeChange}
                >
                  {docStructure ===
                  SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
                    ? '切换为 DIRECT'
                    : '切换为 HOUSE'}
                </Button>
              ) : null}
            </Col>
          </Row>
        ) : (
          <Form.Item
            name="seaDocumentStructure"
            label="单证模式"
            rules={[{ required: true, message: '请选择 HOUSE 或 DIRECT' }]}
            style={{ marginBottom: 0 }}
          >
            <Radio.Group
              disabled={disabled}
              onChange={(event) =>
                changeCreateMode(event.target.value as SeaDocumentStructure)
              }
            >
              <Radio.Button
                value={SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE}
              >
                HOUSE（签发 HBL）
              </Radio.Button>
              <Radio.Button
                value={SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT}
              >
                DIRECT（直接交付 MBL）
              </Radio.Button>
            </Radio.Group>
          </Form.Item>
        )}

        {docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT ? (
          <Alert
            style={{ marginTop: 12 }}
            type="info"
            showIcon
            title="当前为直单（DIRECT）"
            description="本订单不签发 HBL，直接向客户交付船公司或船代提供的 MBL。"
          />
        ) : null}
        {!isDetail && !docStructure ? (
          <Alert
            style={{ marginTop: 12 }}
            type="warning"
            showIcon
            title="请先选择单证模式"
            description="HOUSE 必须随订单提交唯一 HBL；DIRECT 不提交 HBL。"
          />
        ) : null}
      </Card>

      <Tabs
        type="card"
        activeKey={activeTabKey}
        onChange={setActiveTabKey}
        items={[
          masterBillTab,
          ...(docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
            ? [houseBillTab]
            : []),
        ]}
      />

      <Modal
        title={
          modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
            ? '切换为 HOUSE'
            : '切换为 DIRECT'
        }
        width={860}
        open={modeModalOpen}
        destroyOnHidden
        mask={{ closable: false }}
        onCancel={() => {
          if (modeExecuting) return;
          setModeModalOpen(false);
          setModePreview(null);
        }}
        footer={[
          <Button
            key="cancel"
            disabled={modeExecuting}
            onClick={() => setModeModalOpen(false)}
          >
            取消
          </Button>,
          modePreview ? (
            <Button
              key="preview-again"
              disabled={modeExecuting}
              loading={modePreviewing}
              onClick={previewModeChange}
            >
              重新预览
            </Button>
          ) : null,
          modePreview ? (
            <Button
              key="execute"
              type="primary"
              disabled={!modePreview.executable}
              loading={modeExecuting}
              onClick={executeModeChange}
            >
              确认执行
            </Button>
          ) : (
            <Button
              key="preview"
              type="primary"
              loading={modePreviewing}
              onClick={previewModeChange}
            >
              预览切换影响
            </Button>
          ),
        ]}
      >
        <Alert
          type="warning"
          showIcon
          title="模式切换以外部确认结果为准"
          description={
            modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
              ? '执行后会建立一张新的当前 HBL；请先录入已取得的分单号和签发主体。'
              : '执行后当前 HBL 会作废并保留历史；已有放货、账单等事实不会被删除或搬移。'
          }
          style={{ marginBottom: 16 }}
        />
        <Form<ModeChangeFormValues>
          form={modeForm}
          layout="vertical"
          preserve={false}
          onValuesChange={(changedValues) => {
            if (Object.keys(changedValues).length === 0) return;
            setModePreview(null);
            setModeIdempotencyKey(`sea-document-mode:${generateUUID()}`);
          }}
        >
          {modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE ? (
            <HouseBillIdentityFields fieldKey="newHouseBill" />
          ) : null}
          <Form.Item
            name="reason"
            label="切换原因"
            rules={[
              { required: true, whitespace: true, message: '请输入切换原因' },
            ]}
          >
            <ProFormTextArea
              noStyle
              placeholder="说明客户请求及本次 HOUSE/DIRECT 切换原因"
              fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
            />
          </Form.Item>
          <div style={{ marginTop: 12, marginBottom: 12 }}>
            <Text strong>外部确认留痕</Text>
            <br />
            <Text type="secondary">
              请先完整填写确认信息再预览；确认信息变化后必须重新预览。
            </Text>
          </div>
          <SeaExternalConfirmationFields orderId={orderId} />
          {modePreview ? (
            <ModeChangePreviewResult preview={modePreview} />
          ) : null}
        </Form>
      </Modal>
    </Col>
  );
}

export function buildSeaDocumentSection(props: TemplateProps): TemplateSection {
  return {
    key: 'sea-document',
    title: '提单信息',
    content: (
      <SeaDocumentSectionComponent
        disabled={props.readonly}
        isDetail={props.isDetail}
        onOrderDataChanged={props.onOrderDataChanged}
      />
    ),
  };
}
