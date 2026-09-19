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
  HouseBillIdentityFields,
  houseBillStatusPresentation,
  isTerminalHouseBill,
} from './HouseBillIdentityFields';
import { SeaBillContentFormFields } from './SeaBillContentFormFields';
import {
  buildHouseBillInput,
  isDocumentStructure,
  ModeChangeFormValues,
  ModeChangePreviewResult,
  SeaCreateDocumentModeField,
} from './SeaCreateDocumentModeField';
import {
  DEFAULT_BILL_FORM,
  DEFAULT_FREIGHT_TERMS,
  DEFAULT_RELEASE_TYPE,
  DEFAULT_TRANSPORT_TERMS,
  SEA_BILL_FORM_OPTIONS,
  SEA_DOCUMENT_CONTENT_FIELDS,
  SEA_FREIGHT_TERM_OPTIONS,
  SEA_RELEASE_TYPE_OPTIONS,
  SEA_TRANSPORT_TERM_OPTIONS,
} from './seaDocumentSectionConstants';

export {
  HouseBillIdentityFields,
  normalizeSeaHouseNoForCompare,
} from './HouseBillIdentityFields';
export { SeaBillContentFormFields } from './SeaBillContentFormFields';
export { SeaCreateDocumentModeField } from './SeaCreateDocumentModeField';
// 迁出后的公共面保持原导入路径稳定。
export * from './seaDocumentSectionConstants';

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
  // Form.Item initialValue 场景下 useWatch 可能滞后于 store，回退读取当前值。
  const storedStructure = form.getFieldValue('seaDocumentStructure') as
    | number
    | undefined;
  const watchedHouseNo = Form.useWatch(['seaHouseBill', 'houseNo'], form) as
    | string
    | undefined;
  const mblMasterNo = Form.useWatch('seaMasterBillMasterNo', form);
  const docStructure = isDocumentStructure(watchedStructure)
    ? watchedStructure
    : isDocumentStructure(storedStructure)
      ? storedStructure
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
          <SeaCreateDocumentModeField
            disabled={disabled}
            onModeChange={(nextMode) => {
              setLoadedStructure(nextMode);
              setActiveTabKey(
                nextMode === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
                  ? 'hbl'
                  : 'mbl',
              );
            }}
          />
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
