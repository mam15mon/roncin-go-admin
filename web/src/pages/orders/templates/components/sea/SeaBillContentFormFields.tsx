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
import { useAccess } from '@/app/access';
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

import {
  DEFAULT_BILL_FORM,
  DEFAULT_FREIGHT_TERMS,
  DEFAULT_RELEASE_TYPE,
  DEFAULT_TRANSPORT_TERMS,
  SEA_BILL_FORM_OPTIONS,
  SEA_FREIGHT_TERM_OPTIONS,
  SEA_RELEASE_TYPE_OPTIONS,
  SEA_TRANSPORT_TERM_OPTIONS,
} from './seaDocumentSectionConstants';

type MeasurementCompareStatus =
  | 'match'
  | 'diff'
  | 'incomplete'
  | 'unitMismatch';

// compareMeasurement 计算实际相对委托的差额（保留 3 位小数与录入精度一致）；
// 仅当两侧件数单位均已录入且不同时判为单位不一致，缺单位时只比数值。
function compareMeasurement(
  consigned: unknown,
  actual: unknown,
  consignedUnit?: unknown,
  actualUnit?: unknown,
): { status: MeasurementCompareStatus; diff: number } {
  if (typeof consigned !== 'number' || typeof actual !== 'number') {
    return { status: 'incomplete', diff: 0 };
  }
  if (
    consignedUnit != null &&
    actualUnit != null &&
    String(consignedUnit) !== String(actualUnit)
  ) {
    return { status: 'unitMismatch', diff: actual - consigned };
  }
  const diff = Number((actual - consigned).toFixed(3));
  return { status: diff === 0 ? 'match' : 'diff', diff };
}

// formatMeasurementDiff 输出带符号差额；0 不会进入（match 已拦截），负数自带 - 号
function formatMeasurementDiff(value: number): string {
  return value > 0 ? `+${value}` : `${value}`;
}

export function SeaBillContentFormFields({
  namePathPrefix,
  disabled = false,
  showCargoMeasurements = true,
  createLayout = false,
}: {
  namePathPrefix: (string | number)[];
  disabled?: boolean;
  showCargoMeasurements?: boolean;
  createLayout?: boolean;
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

  // 件重尺对照差值：委托侧为订单级汇总字段，实际侧在提单内容前缀下；
  // 汇总行实时计算「实际 - 委托」差额，仅展示异常项，全部一致时收敛为一条绿色提示。
  const consignedPackages = Form.useWatch('totalPackages', form);
  const consignedPackageUnit = Form.useWatch('totalPackageUnit', form);
  const consignedGrossWeightKg = Form.useWatch('totalGrossWeightKg', form);
  const consignedVolumeCbm = Form.useWatch('totalVolumeCbm', form);
  const actualPackages = Form.useWatch(
    [...namePathPrefix, 'packageCount'],
    form,
  );
  const actualPackageUnit = Form.useWatch(
    [...namePathPrefix, 'packageUnit'],
    form,
  );
  const actualGrossWeightKg = Form.useWatch(
    [...namePathPrefix, 'grossWeightKg'],
    form,
  );
  const actualVolumeCbm = Form.useWatch([...namePathPrefix, 'volumeCbm'], form);
  const measurementCompares = [
    {
      label: '件数',
      unit: String(actualPackageUnit || consignedPackageUnit || ''),
      ...compareMeasurement(
        consignedPackages,
        actualPackages,
        consignedPackageUnit,
        actualPackageUnit,
      ),
    },
    {
      label: '毛重',
      unit: 'KGS',
      ...compareMeasurement(consignedGrossWeightKg, actualGrossWeightKg),
    },
    {
      label: '体积',
      unit: 'CBM',
      ...compareMeasurement(consignedVolumeCbm, actualVolumeCbm),
    },
  ];
  const flaggedMeasurements = measurementCompares.filter(
    (item) => item.status === 'diff' || item.status === 'unitMismatch',
  );
  const hasComparableMeasurement = measurementCompares.some(
    (item) => item.status !== 'incomplete',
  );
  const comparisonGridColumns = showCargoMeasurements
    ? '96px 1fr 1fr'
    : '96px 1fr';
  const comparisonHeadCellStyle = {
    padding: '6px 12px',
    fontSize: 12,
    fontWeight: 600,
    color: 'rgba(0, 0, 0, 0.65)',
  };
  const comparisonLabelCellStyle = {
    padding: '6px 12px',
    fontSize: 13,
    color: 'rgba(0, 0, 0, 0.85)',
    display: 'flex',
    alignItems: 'center',
  };
  const comparisonValueCellStyle = {
    padding: '6px 12px',
    display: 'flex',
    alignItems: 'center',
    minWidth: 0,
    borderLeft: '1px solid #f0f0f0',
  };

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
                flexWrap: createLayout ? 'wrap' : undefined,
                gap: createLayout ? 6 : undefined,
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
                  tabIndex={-1}
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
                flexWrap: createLayout ? 'wrap' : undefined,
                gap: createLayout ? 6 : undefined,
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
                flexWrap: createLayout ? 'wrap' : undefined,
                gap: createLayout ? 6 : undefined,
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
                flexWrap: createLayout ? 'wrap' : undefined,
                gap: createLayout ? 6 : undefined,
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
                  tabIndex={-1}
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
            flexWrap: createLayout ? 'wrap' : undefined,
            gap: createLayout ? 6 : undefined,
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
              tabIndex={-1}
              style={{ padding: 0 }}
            >
              从订单货物信息带入 (复制委托数据)
            </Button>
          )}
        </div>

        <FormRow cols={2}>
          <div style={{ marginBottom: 24 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                flexWrap: createLayout ? 'wrap' : undefined,
                gap: createLayout ? 6 : undefined,
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
                    form?.setFieldValue([...namePathPrefix, 'marksText'], 'N/M')
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
          <div style={{ marginBottom: 24 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                flexWrap: createLayout ? 'wrap' : undefined,
                gap: createLayout ? 6 : undefined,
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
                    !disabled ? () => handleToggleClause(!hasClause) : undefined
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
        </FormRow>
      </div>

      {/* 委托 vs 实际件重尺对照：标题行右侧放「带入委托件重尺」操作；
          对照数据以轻量表格呈现，左列客户委托、右列订舱/实际，底部汇总行自动比对差额 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexWrap: createLayout ? 'wrap' : undefined,
          gap: createLayout ? 6 : undefined,
          margin: '8px 0 8px 0',
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
            {showCargoMeasurements
              ? '委托 vs 实际件重尺对照'
              : 'HBL 实际件重尺'}
          </span>
        </div>
        {!disabled && (
          <Button
            type="link"
            size="small"
            icon={<SwapOutlined />}
            onClick={handleImportMeasurementsFromCargo}
            tabIndex={-1}
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
        {/* 件重尺对照表格：指标行头 + 左列客户委托 + 右列订舱/实际；
            字段名与既有契约一致（委托侧订单级、实际侧提单内容前缀下），
            HBL 专属模式（showCargoMeasurements=false）隐藏委托列与汇总行 */}
        <div
          style={{
            border: '1px solid #f0f0f0',
            borderRadius: 4,
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: comparisonGridColumns,
              background: '#fafafa',
            }}
          >
            <div style={comparisonHeadCellStyle}>指标</div>
            {showCargoMeasurements && (
              <div
                style={{
                  ...comparisonHeadCellStyle,
                  borderLeft: '1px solid #f0f0f0',
                }}
              >
                客户委托
              </div>
            )}
            <div
              style={{
                ...comparisonHeadCellStyle,
                borderLeft: '1px solid #f0f0f0',
              }}
            >
              订舱/实际
            </div>
          </div>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: comparisonGridColumns,
              borderTop: '1px solid #f0f0f0',
            }}
          >
            <div style={comparisonLabelCellStyle}>件数</div>
            {showCargoMeasurements && (
              <div style={comparisonValueCellStyle}>
                <PackageCountInput
                  countName="totalPackages"
                  unitName="totalPackageUnit"
                  countPlaceholder="0"
                  unitPlaceholder="请选择单位"
                  unitWidth={104}
                  style={{ width: '100%' }}
                  disabled={disabled}
                />
              </div>
            )}
            <div style={comparisonValueCellStyle}>
              <PackageCountInput
                countName={[...namePathPrefix, 'packageCount']}
                unitName={[...namePathPrefix, 'packageUnit']}
                countPlaceholder="0"
                unitPlaceholder="请选择单位"
                unitWidth={104}
                style={{ width: '100%' }}
                disabled={disabled}
              />
            </div>
          </div>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: comparisonGridColumns,
              borderTop: '1px solid #f0f0f0',
            }}
          >
            <div style={comparisonLabelCellStyle}>毛重（KGS）</div>
            {showCargoMeasurements && (
              <div style={comparisonValueCellStyle}>
                <ProFormDigit
                  name="totalGrossWeightKg"
                  placeholder="0"
                  disabled={disabled}
                  min={0}
                  noStyle
                  fieldProps={{ precision: 3, style: { width: '100%' } }}
                />
              </div>
            )}
            <div style={comparisonValueCellStyle}>
              <ProFormDigit
                name={[...namePathPrefix, 'grossWeightKg']}
                placeholder="0"
                disabled={disabled}
                min={0}
                noStyle
                fieldProps={{ precision: 3, style: { width: '100%' } }}
              />
            </div>
          </div>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: comparisonGridColumns,
              borderTop: '1px solid #f0f0f0',
            }}
          >
            <div style={comparisonLabelCellStyle}>体积（CBM）</div>
            {showCargoMeasurements && (
              <div style={comparisonValueCellStyle}>
                <ProFormDigit
                  name="totalVolumeCbm"
                  placeholder="0"
                  disabled={disabled}
                  min={0}
                  noStyle
                  fieldProps={{ precision: 3, style: { width: '100%' } }}
                />
              </div>
            )}
            <div style={comparisonValueCellStyle}>
              <ProFormDigit
                name={[...namePathPrefix, 'volumeCbm']}
                placeholder="0"
                disabled={disabled}
                min={0}
                noStyle
                fieldProps={{ precision: 3, style: { width: '100%' } }}
              />
            </div>
          </div>
          {showCargoMeasurements && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                flexWrap: 'wrap',
                gap: 12,
                padding: '6px 12px',
                background: '#fafafa',
                borderTop: '1px solid #f0f0f0',
                fontSize: 12,
              }}
            >
              <span style={{ fontWeight: 600, color: 'rgba(0, 0, 0, 0.65)' }}>
                汇总
              </span>
              {flaggedMeasurements.length > 0 ? (
                flaggedMeasurements.map((item) => (
                  <span
                    key={item.label}
                    style={{ color: '#fa8c16', fontWeight: 500 }}
                  >
                    {item.status === 'unitMismatch'
                      ? `${item.label}：委托与实际单位不一致`
                      : `${item.label}：实际${formatMeasurementDiff(item.diff)}${item.unit ? ` ${item.unit}` : ''}`}
                  </span>
                ))
              ) : hasComparableMeasurement ? (
                <span style={{ color: '#52c41a', fontWeight: 500 }}>
                  ✓ 委托与实际件重尺一致
                </span>
              ) : (
                <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>
                  录入委托与实际件重尺后自动比对差额
                </span>
              )}
            </div>
          )}
        </div>
        <FormRow cols={4}>
          <div
            style={{
              display: 'flex',
              alignItems: createLayout ? 'stretch' : 'center',
              flexDirection: createLayout ? 'column' : 'row',
              gap: createLayout ? 4 : undefined,
            }}
          >
            <span
              style={{
                width: createLayout ? 'auto' : 68,
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
          <div
            style={{
              display: 'flex',
              alignItems: createLayout ? 'stretch' : 'center',
              flexDirection: createLayout ? 'column' : 'row',
              gap: createLayout ? 4 : undefined,
            }}
          >
            <span
              style={{
                width: createLayout ? 'auto' : 68,
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
          <div
            style={{
              display: 'flex',
              alignItems: createLayout ? 'stretch' : 'center',
              flexDirection: createLayout ? 'column' : 'row',
              gap: createLayout ? 4 : undefined,
            }}
          >
            <span
              style={{
                width: createLayout ? 'auto' : 68,
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
          <div
            style={{
              display: 'flex',
              alignItems: createLayout ? 'stretch' : 'center',
              flexDirection: createLayout ? 'column' : 'row',
              gap: createLayout ? 4 : undefined,
            }}
          >
            <span
              style={{
                width: createLayout ? 'auto' : 68,
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
