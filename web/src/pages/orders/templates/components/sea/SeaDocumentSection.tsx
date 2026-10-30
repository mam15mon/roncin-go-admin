import { SwapOutlined } from '@ant-design/icons';
import { ProFormTextArea } from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Form,
  Modal,
  Row,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { createStyles } from 'antd-style';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import {
  OrderBusinessType,
  OrderReleasePodStatus,
  SeaDocumentStructure,
  SeaDocumentType,
  SeaHouseBillIssuerSource,
} from '@/enums.generated';
import { orderReleasePodServiceListReleasePods } from '@/services/roncin/orderReleasePodService';
import {
  seaDocumentServiceExecuteChangeSeaDocumentMode,
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServicePreviewChangeSeaDocumentMode,
} from '@/services/roncin/seaDocumentService';
import { isWriteResponse } from '@/utils/api';
import { generateUUID } from '@/utils/uuid';
import { RELEASE_PODS_CHANGED_EVENT } from '../../../release-pod-events';
import { seaBillContentFormValues } from '../../../sea-document-form';
import { onSeaFormErrorReveal } from '../../sea-form-error-reveal';
import type { TemplateProps, TemplateSection } from '../../types';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
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
  type ModeChangeFormValues,
  ModeChangePreviewResult,
} from './SeaCreateDocumentModeField';

export {
  HouseBillIdentityFields,
  normalizeSeaHouseNoForCompare,
} from './HouseBillIdentityFields';
export { SeaBillContentFormFields } from './SeaBillContentFormFields';
export { SeaCreateDocumentModeField } from './SeaCreateDocumentModeField';
// 迁出后的公共面保持原导入路径稳定。
export * from './seaDocumentSectionConstants';

const useDocumentTabsStyles = createStyles(({ css }) => ({
  tabs: css`
    &.ant-tabs {
      .ant-tabs-switch-enter-active,
      .ant-tabs-switch-leave-active {
        transition-duration: 180ms;
      }

      @media (prefers-reduced-motion: reduce) {
        .ant-tabs-switch-enter-active,
        .ant-tabs-switch-leave-active,
        .ant-tabs-ink-bar-animated {
          transition: none;
        }
      }
    }
  `,
}));

export function SeaDocumentSectionComponent({
  disabled = false,
  isDetail = false,
  organizationId,
  documents,
  onOrderDataChanged,
}: {
  disabled?: boolean;
  isDetail?: boolean;
  organizationId?: string;
  documents?: API.SeaOrderDocuments;
  onOrderDataChanged?: (
    document?: 'master' | 'house' | 'mode',
  ) => Promise<void> | void;
}) {
  const { styles } = useDocumentTabsStyles();
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
  const { markDirty: markModeDirty } = useUnsavedChangesGuard({
    enabled: modeModalOpen,
  });
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
  const canChangeMode =
    access.canOperateOrganization(organizationId) &&
    access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'update');

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
      if (!response) throw new Error('放货记录加载失败，请重试');
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

  const applyMasterBill = useCallback(
    (masterBill: API.SeaMasterBillDetail | null) => {
      setMblDetail(masterBill);
      form.setFieldValue(
        'seaMasterBillContent',
        seaBillContentFormValues(masterBill?.content),
      );
    },
    [form],
  );

  const applyHouseBill = useCallback(
    (currentHouseBill: API.SeaHouseBill | null) => {
      setHouseBill(currentHouseBill);
      form.setFieldValue(
        'seaHouseBill',
        currentHouseBill
          ? {
              id: currentHouseBill.id,
              houseNo: currentHouseBill.houseNo,
              status: currentHouseBill.status,
              issuerSource: currentHouseBill.issuerSource,
              issuerPartnerId:
                currentHouseBill.issuerSource ===
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
                  ? currentHouseBill.issuerPartnerId
                  : undefined,
              note: currentHouseBill.note,
              content: seaBillContentFormValues(currentHouseBill.content),
              expectedVersion: currentHouseBill.version,
            }
          : undefined,
      );
    },
    [form],
  );

  const loadOrderDocuments = useCallback(async () => {
    const requestedOrderId = orderId;
    const requestSequence = ++documentRequestSequenceRef.current;
    if (!requestedOrderId || !isDetail || documents) return;
    try {
      const response = await seaDocumentServiceGetSeaOrderDocuments({
        orderId: requestedOrderId,
      });
      if (
        requestedOrderId !== activeOrderIdRef.current ||
        requestSequence !== documentRequestSequenceRef.current
      )
        return;
      if (!response?.data) throw new Error('海运单证加载失败，请重试');
      const structure = response.data.documentStructure;
      if (!isDocumentStructure(structure))
        throw new Error('海运订单缺少明确的 HOUSE/DIRECT 提单模式');
      const currentHouseBill = response.data.houseBill ?? null;
      if (
        (structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE &&
          !currentHouseBill) ||
        (structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT &&
          currentHouseBill)
      ) {
        throw new Error('海运提单模式与当前 HBL 不一致');
      }

      setFetchError(null);
      setLoadedStructure(structure);
      setLinkVersion(String(response.data.linkVersion ?? '0'));
      form.setFieldValue('seaDocumentStructure', structure);
      form.setFieldValue(
        'seaMasterBillContentVersion',
        response.data.masterBill?.version,
      );
      form.setFieldValue('seaDocumentLinkVersion', response.data.linkVersion);
      applyMasterBill(response.data.masterBill ?? null);
      applyHouseBill(currentHouseBill);
    } catch (error: unknown) {
      if (
        requestedOrderId !== activeOrderIdRef.current ||
        requestSequence !== documentRequestSequenceRef.current
      )
        return;
      const errorMessage =
        error instanceof Error ? error.message : '获取海运单证信息失败';
      setFetchError(errorMessage);
      message.error(errorMessage);
    }
  }, [
    applyHouseBill,
    applyMasterBill,
    documents,
    form,
    isDetail,
    message,
    orderId,
  ]);

  useEffect(() => {
    if (!isDetail) return;
    if (documents) {
      setLoadedStructure(documents.documentStructure as SeaDocumentStructure);
      setLinkVersion(String(documents.linkVersion ?? '0'));
      setMblDetail(documents.masterBill ?? null);
      setHouseBill(documents.houseBill ?? null);
      void loadReleasePods();
      return;
    }
    setFetchError(null);
    setReleasePodsError(null);
    setLoadedStructure(undefined);
    setLinkVersion('0');
    setMblDetail(null);
    setHouseBill(null);
    form.setFieldValue('seaMasterBillContent', {});
    form.setFieldValue('seaHouseBill', undefined);
    setActiveTabKey('mbl');
    void loadOrderDocuments();
    void loadReleasePods();
  }, [documents, form, isDetail, loadOrderDocuments, loadReleasePods]);

  useEffect(() => {
    if (docStructure !== SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE)
      setActiveTabKey('mbl');
  }, [docStructure]);

  // 提交校验失败前置链路：按首个错误字段把对应提单页签切到前台，
  // 保证从未打开过的页签字段也能被滚动定位（页签 forceRender 常驻注册）。
  useEffect(
    () =>
      onSeaFormErrorReveal((fields) => {
        const [firstField] = fields;
        if (!firstField) return;
        if (
          firstField.name[0] === 'seaHouseBill' &&
          firstField.name[1] !== 'houseNo'
        ) {
          setActiveTabKey('hbl');
        } else if (firstField.name[0] === 'seaMasterBillContent') {
          setActiveTabKey('mbl');
        }
      }),
    [],
  );

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
    if (!isDetail || !canReadReleasePods) return null;
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
          <Space orientation="vertical" size={4}>
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

  const openModeChange = () => {
    if (!isDocumentStructure(docStructure)) {
      message.error('当前提单模式缺失，请刷新后重试');
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

  const executeModeChange = async (): Promise<false | undefined> => {
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
      const writeResponse3 =
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
      if (
        !isWriteResponse(writeResponse3) ||
        writeResponse3.data?.orderId !== orderId
      )
        return false;
      message.success(
        modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
          ? '已切换为有货代分单（HOUSE） 并建立当前 HBL'
          : '已切换为仅船公司主单（DIRECT），原 HBL 已形成作废历史',
      );
      setModeModalOpen(false);
      setModePreview(null);
      await onOrderDataChanged?.('mode');
      await Promise.all([loadOrderDocuments(), loadReleasePods()]);
    } catch (error: unknown) {
      if (error instanceof Error)
        message.error(error.message || '执行模式切换失败');
    } finally {
      setModeExecuting(false);
    }

    return undefined;
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
      <div>
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
                  await onOrderDataChanged?.('master');
                  await loadOrderDocuments();
                }}
              />
            </Col>
          </Row>
        ) : null}
        <SeaBillContentFormFields
          namePathPrefix={['seaMasterBillContent']}
          disabled={disabled || mblDetail?.status === 'VOIDED'}
          showCargoMeasurements={false}
          measurementsTitle="MBL 实际件重尺"
        />
        {renderReleasePods(
          relatedReleasePods(
            SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
            mblDetail?.id,
          ),
        )}
      </div>
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
      <div>
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
                    await onOrderDataChanged?.('house');
                    await loadOrderDocuments();
                  }}
                />
              ) : null}
            </Col>
          </Row>
        ) : null}
        <Alert
          type="info"
          showIcon
          title={
            isDetail
              ? '分单号、签发主体与分单正文将随整页保存一起提交。'
              : '分单号请在上方主单号旁填写；此处维护签发主体与分单内容。'
          }
          style={{ marginBottom: 16 }}
        />
        <HouseBillIdentityFields
          fieldKey="seaHouseBill"
          showHouseNo={false}
          disabled={disabled || isTerminalHouseBill(houseBill?.status)}
        />
        <div style={{ marginTop: 12 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>
            提单正文内容
          </Text>
          <SeaBillContentFormFields
            namePathPrefix={['seaHouseBill', 'content']}
            disabled={disabled || isTerminalHouseBill(houseBill?.status)}
            showCargoMeasurements={false}
          />
        </div>
        {renderReleasePods(
          relatedReleasePods(
            SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
            houseBill?.id,
          ),
        )}
      </div>
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

  const renderModeSwitchButton = () =>
    canChangeMode ? (
      <Button
        icon={<SwapOutlined />}
        disabled={disabled || !modeChangeReady}
        onClick={openModeChange}
      >
        {docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
          ? '切换为仅船公司主单（DIRECT）'
          : '切换为有货代分单（HOUSE）'}
      </Button>
    ) : null;

  return (
    <Col span={24}>
      {isDetail ? (
        <>
          <Form.Item hidden name="id">
            <input />
          </Form.Item>
          <Form.Item hidden name="version">
            <input />
          </Form.Item>
          <Form.Item hidden name="seaMasterBillContentVersion">
            <input />
          </Form.Item>
          <Form.Item hidden name="seaDocumentLinkVersion">
            <input />
          </Form.Item>
          <Form.Item hidden name={['seaHouseBill', 'id']}>
            <input />
          </Form.Item>
          <Form.Item hidden name={['seaHouseBill', 'expectedVersion']}>
            <input />
          </Form.Item>
        </>
      ) : null}
      {fetchError ? (
        <Alert
          type="error"
          showIcon
          title="获取单证信息失败"
          description={fetchError}
          style={{ marginBottom: 16 }}
        />
      ) : null}

      {!isDetail && !docStructure ? (
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          title="请先选择提单模式"
          description="HOUSE 必须随订单提交唯一 HBL；DIRECT 不提交 HBL。"
        />
      ) : null}

      <Tabs
        className={styles.tabs}
        animated={{ inkBar: true, tabPane: true }}
        type="card"
        activeKey={activeTabKey}
        onChange={setActiveTabKey}
        tabBarExtraContent={
          isDetail
            ? {
                right: (
                  <Space size="middle">
                    {linkVersion !== '0' ? (
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        单证版本 v{linkVersion}
                      </Text>
                    ) : null}
                    {renderModeSwitchButton()}
                  </Space>
                ),
              }
            : undefined
        }
        items={[
          { ...masterBillTab, forceRender: true },
          ...(docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
            ? [{ ...houseBillTab, forceRender: true }]
            : []),
        ]}
      />

      <Modal
        title={
          modeTarget === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
            ? '切换为有货代分单（HOUSE）'
            : '切换为仅船公司主单（DIRECT）'
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
            markModeDirty();
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
              fieldProps={{
                autoSize: { minRows: 3, maxRows: 6 },
                maxLength: 500,
                showCount: true,
              }}
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
    key: 'seaDocument',
    title: '提单信息',
    content: (
      <SeaDocumentSectionComponent
        disabled={props.readonly}
        isDetail={props.isDetail}
        organizationId={props.organizationId}
        documents={props.seaDocuments}
        onOrderDataChanged={props.onOrderDataChanged}
      />
    ),
  };
}
