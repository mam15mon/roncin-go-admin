import {
  AppstoreOutlined,
  CopyOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Form,
  Radio,
  Row,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  OrderBusinessType,
  OrderReleasePodStatus,
  SeaDocumentStructure,
  SeaDocumentType,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import { orderReleasePodServiceListReleasePods } from '@/services/roncin/orderReleasePodService';
import {
  seaCargoAllocationServiceApplySeaHouseBillAllocationSummary,
  seaCargoAllocationServiceApplySeaOrderCargoSummaryToMasterBill,
  seaCargoAllocationServiceGetSeaCargoAllocation,
} from '@/services/roncin/seaCargoAllocationService';
import {
  seaDocumentServiceAddSeaHouseBill,
  seaDocumentServiceCancelSeaOrderDirect,
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServiceMarkSeaOrderDirect,
  seaDocumentServiceRemoveSeaHouseBill,
  seaDocumentServiceUpdateSeaHouseBill,
  seaDocumentServiceUpdateSeaMasterBillContent,
} from '@/services/roncin/seaDocumentService';
import { searchPartnerOptions } from '@/utils/options';
import SeaCargoAllocationDrawer, {
  type SeaCargoAllocationDrawerRef,
} from '../../../components/drawers/SeaCargoAllocationDrawer';
import type { TemplateProps, TemplateSection } from '../../types';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';
import { RELEASE_PODS_CHANGED_EVENT } from '../../../release-pod-events';

const { Text } = Typography;

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
];

export function SeaBillContentFormFields({
  namePathPrefix,
  disabled = false,
}: {
  namePathPrefix: (string | number)[];
  disabled?: boolean;
}) {
  return (
    <Row gutter={[16, 0]}>
      <Col xs={24} lg={12}>
        <ProFormTextArea
          name={[...namePathPrefix, 'shipperText']}
          label="发货人 (Shipper)"
          placeholder="请输入发货人名称与地址"
          disabled={disabled}
          fieldProps={{ rows: 3 }}
        />
      </Col>
      <Col xs={24} lg={12}>
        <ProFormTextArea
          name={[...namePathPrefix, 'consigneeText']}
          label="收货人 (Consignee)"
          placeholder="请输入收货人名称与地址"
          disabled={disabled}
          fieldProps={{ rows: 3 }}
        />
      </Col>
      <Col xs={24} lg={12}>
        <ProFormTextArea
          name={[...namePathPrefix, 'notifyPartyText']}
          label="通知人 (Notify Party)"
          placeholder="请输入通知人名称与地址"
          disabled={disabled}
          fieldProps={{ rows: 3 }}
        />
      </Col>
      <Col xs={24} lg={12}>
        <ProFormTextArea
          name={[...namePathPrefix, 'secondNotifyPartyText']}
          label="第二通知人 (Second Notify Party)"
          placeholder="请输入第二通知人名称与地址"
          disabled={disabled}
          fieldProps={{ rows: 3 }}
        />
      </Col>
      <Col xs={24} lg={12}>
        <ProFormTextArea
          name={[...namePathPrefix, 'marksText']}
          label="唛头 (Marks & Numbers)"
          placeholder="请输入唛头信息"
          disabled={disabled}
          fieldProps={{ rows: 3 }}
        />
      </Col>
      <Col xs={24} lg={12}>
        <ProFormTextArea
          name={[...namePathPrefix, 'goodsDescriptionText']}
          label="品名/货描 (Description of Goods)"
          placeholder="请输入货物描述"
          disabled={disabled}
          fieldProps={{ rows: 3 }}
        />
      </Col>

      <Col xs={12} lg={6}>
        <ProFormDigit
          name={[...namePathPrefix, 'packageCount']}
          label="件数"
          placeholder="件数"
          disabled={disabled}
          min={0}
        />
      </Col>
      <Col xs={12} lg={6}>
        <ProFormText
          name={[...namePathPrefix, 'packageUnit']}
          label="包装单位"
          placeholder="例如 CTNS / PKGS"
          disabled={disabled}
        />
      </Col>
      <Col xs={12} lg={6}>
        <ProFormDigit
          name={[...namePathPrefix, 'grossWeightKg']}
          label="毛重 (KGS)"
          placeholder="毛重"
          disabled={disabled}
          min={0}
          fieldProps={{ precision: 3 }}
        />
      </Col>
      <Col xs={12} lg={6}>
        <ProFormDigit
          name={[...namePathPrefix, 'volumeCbm']}
          label="体积 (CBM)"
          placeholder="体积"
          disabled={disabled}
          min={0}
          fieldProps={{ precision: 3 }}
        />
      </Col>

      <Col xs={12} lg={6}>
        <ProFormText
          name={[...namePathPrefix, 'freightTerms']}
          label="运费条款"
          placeholder="例如 FREIGHT PREPAID"
          disabled={disabled}
        />
      </Col>
      <Col xs={12} lg={6}>
        <ProFormText
          name={[...namePathPrefix, 'transportTerms']}
          label="运输条款"
          placeholder="例如 CY-CY / FCL-FCL"
          disabled={disabled}
        />
      </Col>
      <Col xs={12} lg={6}>
        <ProFormText
          name={[...namePathPrefix, 'billForm']}
          label="提单形式"
          placeholder="例如 ORIGINAL / COPY"
          disabled={disabled}
        />
      </Col>
      <Col xs={12} lg={6}>
        <ProFormText
          name={[...namePathPrefix, 'releaseType']}
          label="放单方式"
          placeholder="例如 电放 / 正本"
          disabled={disabled}
        />
      </Col>
      <Col xs={24}>
        <ProFormTextArea
          name={[...namePathPrefix, 'clauses']}
          label="提单特别条款 (Clauses)"
          placeholder="请输入特别条款"
          disabled={disabled}
          fieldProps={{ rows: 2 }}
        />
      </Col>
    </Row>
  );
}

function HouseBillTabTitle({ index }: { index: number }) {
  const form = Form.useFormInstance();
  const houseNo = Form.useWatch(['seaHouseBills', index, 'houseNo'], form);

  return (
    <span>
      分单 {index + 1}{' '}
      {houseNo ? (
        <Text type="secondary" style={{ fontSize: 12 }}>
          ({houseNo})
        </Text>
      ) : (
        <Text type="secondary" style={{ fontSize: 12 }}>
          (未编号)
        </Text>
      )}
    </span>
  );
}

function isTerminalHouseBill(status?: number) {
  return (
    status === SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED ||
    status === SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_REPLACED
  );
}

function houseBillStatusPresentation(status?: number) {
  switch (status) {
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_RELEASED:
      return { color: 'success', text: '已签发' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_CONFIRMED:
      return { color: 'blue', text: '已确认' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED:
      return { color: 'error', text: '已作废' };
    case SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_REPLACED:
      return { color: 'warning', text: '已替代' };
    default:
      return { color: 'default', text: '草稿' };
  }
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
  const { message, modal } = App.useApp();
  const access = useAccess();

  const [activeTabKey, setActiveTabKey] = useState<string>('mbl');
  const [docStructure, setDocStructure] = useState<SeaDocumentStructure>(
    SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED,
  );
  const [linkVersion, setLinkVersion] = useState<string>('0');
  const [mblDetail, setMblDetail] = useState<API.SeaMasterBillDetail | null>(
    null,
  );
  const [houseBills, setHouseBills] = useState<API.SeaHouseBill[]>([]);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [releasePods, setReleasePods] = useState<API.OrderReleasePod[]>([]);
  const [releasePodsError, setReleasePodsError] = useState<string | null>(null);
  const [applyingHblId, setApplyingHblId] = useState<string | null>(null);
  const [applyingMbl, setApplyingMbl] = useState(false);
  const allocationDrawerRef = useRef<SeaCargoAllocationDrawerRef>(null);

  const orderId = Form.useWatch('id', form) || form.getFieldValue('id');
  const orderVersion =
    Form.useWatch('version', form) || form.getFieldValue('version');
  const mblMasterNo = Form.useWatch('seaMasterBillMasterNo', form);
  const canReadReleasePods = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'release_pod.read',
  );
  const canDeleteReleasePods = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'release_pod.delete',
  );

  const loadReleasePods = useCallback(async () => {
    if (!orderId || !isDetail || !canReadReleasePods) {
      setReleasePods([]);
      setReleasePodsError(null);
      return;
    }
    try {
      setReleasePodsError(null);
      const response = await orderReleasePodServiceListReleasePods({
        orderId: String(orderId),
      });
      setReleasePods(response.data ?? []);
    } catch (error: unknown) {
      setReleasePods([]);
      setReleasePodsError(
        error instanceof Error ? error.message : '放货记录加载失败',
      );
    }
  }, [orderId, isDetail, canReadReleasePods]);

  // 详情页加载单证聚合数据
  const loadOrderDocuments = useCallback(async () => {
    if (!orderId || !isDetail) return;
    try {
      setFetchError(null);
      const res = await seaDocumentServiceGetSeaOrderDocuments({
        orderId: String(orderId),
      });
      if (!res.data) {
        throw new Error('接口未返回海运单证数据');
      }

      const structure =
        (res.data.documentStructure as SeaDocumentStructure) ??
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED;
      setDocStructure(structure);
      setLinkVersion(String(res.data.linkVersion ?? '0'));
      setMblDetail(res.data.masterBill ?? null);
      const hbs = res.data.houseBills ?? [];
      setHouseBills(hbs);

      form.setFieldValue('seaDocumentStructure', structure);
      form.setFieldValue(
        'seaMasterBillContent',
        res.data.masterBill?.content || {},
      );
      form.setFieldValue(
        'seaHouseBills',
        hbs.map((hb) => ({
          id: hb.id,
          houseNo: hb.houseNo,
          issuerSource: hb.issuerSource,
          issuerPartnerId:
            hb.issuerSource ===
            SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
              ? hb.issuerPartnerId
              : undefined,
          issuerOrganizationId: hb.issuerOrganizationId,
          note: hb.note,
          content: hb.content || {},
          expectedVersion: hb.version,
          status: hb.status,
        })),
      );
    } catch (err: unknown) {
      const errMsg =
        err instanceof Error ? err.message : '获取海运单证信息失败';
      setDocStructure(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED);
      setLinkVersion('0');
      setMblDetail(null);
      setHouseBills([]);
      form.setFieldValue('seaMasterBillContent', {});
      form.setFieldValue('seaHouseBills', []);
      setFetchError(errMsg);
      message.error(errMsg);
    }
  }, [orderId, isDetail, form, message]);

  useEffect(() => {
    if (isDetail && orderId) {
      loadOrderDocuments();
      loadReleasePods();
    }
  }, [isDetail, orderId, loadOrderDocuments, loadReleasePods]);

  useEffect(() => {
    const handleChanged = (event: Event) => {
      const changedOrderId = (event as CustomEvent<{ orderId?: string }>).detail
        ?.orderId;
      if (changedOrderId === String(orderId)) {
        void loadReleasePods();
      }
    };
    window.addEventListener(RELEASE_PODS_CHANGED_EVENT, handleChanged);
    return () =>
      window.removeEventListener(RELEASE_PODS_CHANGED_EVENT, handleChanged);
  }, [orderId, loadReleasePods]);

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
                  {item.status === OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_RETURNED
                    ? '已回单'
                    : item.status === OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_SIGNED
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

  // 操作：标记直单
  const handleMarkDirect = async () => {
    if (houseBills.length > 0) {
      message.error('存在分单时不可标记为直单');
      return;
    }
    if (isDetail && orderId) {
      try {
        const res = await seaDocumentServiceMarkSeaOrderDirect(
          { orderId: String(orderId) },
          { orderId: String(orderId), expectedLinkVersion: linkVersion },
        );
        message.success('已标记为直单');
        if (res.data) {
          setDocStructure(res.data.documentStructure as SeaDocumentStructure);
          setLinkVersion(String(res.data.linkVersion ?? '0'));
        }
        await loadOrderDocuments();
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : '标记直单失败';
        message.error(msg);
      }
    } else {
      setDocStructure(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT);
      form.setFieldValue(
        'seaDocumentStructure',
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
      );
      message.success('已切换为直单模式');
    }
  };

  // 操作：取消直单
  const handleCancelDirect = async () => {
    if (isDetail && orderId) {
      try {
        const res = await seaDocumentServiceCancelSeaOrderDirect(
          { orderId: String(orderId) },
          { orderId: String(orderId), expectedLinkVersion: linkVersion },
        );
        message.success('已取消直单标记，回到未确定状态');
        if (res.data) {
          setDocStructure(res.data.documentStructure as SeaDocumentStructure);
          setLinkVersion(String(res.data.linkVersion ?? '0'));
        }
        await loadOrderDocuments();
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : '取消直单失败';
        message.error(msg);
      }
    } else {
      setDocStructure(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED);
      form.setFieldValue(
        'seaDocumentStructure',
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED,
      );
      message.success('已回到未确定状态');
    }
  };

  // 操作：添加分单
  const handleAddHouseBill = () => {
    if (docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT) {
      message.error('当前为直单，请先取消直单标记后再添加分单');
      return;
    }

    const newIndex = houseBills.length;
    const newHB: API.SeaHouseBill = {
      houseNo: '',
      issuerSource:
        SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_UNSPECIFIED,
      status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
      version: '1',
      content: {},
    };

    const nextHBs = [...houseBills, newHB];
    setHouseBills(nextHBs);
    setDocStructure(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE);
    form.setFieldValue(
      'seaDocumentStructure',
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
    );
    form.setFieldValue(['seaHouseBills', newIndex], {
      houseNo: '',
      issuerSource: undefined,
      status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
      version: '1',
      content: {},
    });
    setActiveTabKey(`hbl-${newIndex}`);
  };

  // 操作：复制上一张分单内容
  const handleCopyPreviousContent = (currentIndex: number) => {
    if (currentIndex <= 0) return;
    const prevFormValues = form.getFieldValue([
      'seaHouseBills',
      currentIndex - 1,
    ]) as API.SeaHouseBillInput | undefined;
    const prevContent =
      prevFormValues?.content || houseBills[currentIndex - 1]?.content;
    if (!prevContent) {
      message.warning('上一张分单尚无内容可复制');
      return;
    }

    const copiedContent: API.SeaBillContent = {};
    SEA_DOCUMENT_CONTENT_FIELDS.forEach((f) => {
      if (prevContent[f] !== undefined) {
        (copiedContent as Record<string, unknown>)[f] = prevContent[f];
      }
    });

    form.setFieldValue(
      ['seaHouseBills', currentIndex, 'content'],
      copiedContent,
    );
    message.success(`已复制分单 ${currentIndex} 的提单内容`);
  };

  // 操作：删除分单
  const handleRemoveHouseBill = async (index: number) => {
    const targetHB = houseBills[index];
    const isLast = houseBills.length === 1;
    const relatedPods = relatedReleasePods(
      SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
      targetHB.id,
    );
    const returnedPods = relatedPods.filter(
      (item) =>
        item.status ===
        OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_RETURNED,
    );

    if (returnedPods.length > 0) {
      modal.error({
        title: '该分单不能删除',
        content: `存在已回单记录：${returnedPods
          .map((item) => item.releaseNo || item.podNo || item.id)
          .join('、')}`,
      });
      return;
    }
    if (relatedPods.length > 0 && !canDeleteReleasePods) {
      message.error('该分单有关联放货记录，请联系具备放货记录删除权限的人员处理');
      return;
    }

    const performDelete = async () => {
      if (isDetail && orderId && targetHB.id) {
        if (!targetHB.version || Number(targetHB.version) <= 0) {
          message.error('分单版本缺失，请刷新后重试');
          return;
        }
        if (!Number.isSafeInteger(Number(linkVersion)) || Number(linkVersion) <= 0) {
          message.error('单证结构版本缺失，请刷新后重试');
          return;
        }
        try {
          await seaDocumentServiceRemoveSeaHouseBill({
            orderId: String(orderId),
            id: targetHB.id,
            expectedVersion: String(targetHB.version),
            expectedLinkVersion: linkVersion,
            returnToUndetermined: isLast,
            removeRelatedReleasePods: relatedPods.length > 0,
          });
          message.success('分单已删除');
          await loadOrderDocuments();
          await loadReleasePods();
          setActiveTabKey('mbl');
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : '删除分单失败';
          message.error(msg);
          await Promise.all([loadOrderDocuments(), loadReleasePods()]);
        }
      } else {
        const nextHBs = houseBills.filter((_, i) => i !== index);
        setHouseBills(nextHBs);
        // 重排表单值
        const curFormHBs = (form.getFieldValue('seaHouseBills') ||
          []) as API.SeaHouseBillInput[];
        const nextFormHBs = curFormHBs.filter((_, i) => i !== index);
        form.setFieldValue('seaHouseBills', nextFormHBs);
        if (nextHBs.length === 0) {
          setDocStructure(
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED,
          );
          form.setFieldValue(
            'seaDocumentStructure',
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED,
          );
          setActiveTabKey('mbl');
        } else {
          setActiveTabKey('hbl-0');
        }
        message.success('分单已移除');
      }
    };

    if (relatedPods.length > 0) {
      modal.confirm({
        title: '确认删除分单及关联放货记录',
        icon: <ExclamationCircleOutlined />,
        content: (
          <Space direction="vertical">
            <Text>以下待签收/已签收记录将与分单一起删除：</Text>
            {relatedPods.map((item) => (
              <Text key={item.id}>
                {item.releaseNo || '-'} / {item.podNo || '-'}
              </Text>
            ))}
            {isLast ? <Text>同时单证结构将回到未确定状态。</Text> : null}
          </Space>
        ),
        okText: '确认一并删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: performDelete,
      });
    } else if (isLast) {
      modal.confirm({
        title: '删除最后一张分单确认',
        icon: <ExclamationCircleOutlined />,
        content: '删除最后一张分单将使单证结构回到未确定状态，是否确认？',
        okText: '确认删除并回到未确定',
        okType: 'danger',
        cancelText: '取消',
        onOk: performDelete,
      });
    } else {
      modal.confirm({
        title: '确认删除分单',
        content: `确定要删除分单 ${targetHB.houseNo || index + 1} 吗？`,
        okText: '删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: performDelete,
      });
    }
  };

  // 详情模式：保存单独分单
  const handleSaveHouseBill = async (index: number) => {
    if (!orderId || !isDetail) return;
    const hbValues = form.getFieldValue(['seaHouseBills', index]) as
      | API.SeaHouseBillInput
      | undefined;
    if (!hbValues?.houseNo) {
      message.error('请输入分单号');
      return;
    }
    if (!hbValues?.issuerSource) {
      message.error('请选择签发主体');
      return;
    }
    if (
      hbValues.issuerSource ===
        SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER &&
      !hbValues.issuerPartnerId
    ) {
      message.error('其他主体签发必须选择合作伙伴');
      return;
    }

    const hbInput: API.SeaHouseBillInput = {
      id: hbValues.id,
      houseNo: hbValues.houseNo,
      issuerSource: hbValues.issuerSource,
      issuerPartnerId:
        hbValues.issuerSource ===
        SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
          ? hbValues.issuerPartnerId
          : undefined,
      note: hbValues.note?.trim() || undefined,
      content: hbValues.content,
      expectedVersion: hbValues.expectedVersion,
    };

    try {
      if (hbValues.id) {
        await seaDocumentServiceUpdateSeaHouseBill(
          { orderId: String(orderId), id: hbValues.id },
          {
            orderId: String(orderId),
            id: hbValues.id,
            expectedVersion: String(hbValues.expectedVersion || '1'),
            expectedLinkVersion: linkVersion,
            houseBill: hbInput,
          },
        );
        message.success('分单更新成功');
      } else {
        await seaDocumentServiceAddSeaHouseBill(
          { orderId: String(orderId) },
          {
            orderId: String(orderId),
            expectedLinkVersion: linkVersion,
            houseBill: hbInput,
          },
        );
        message.success('分单添加成功');
      }
      await loadOrderDocuments();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '保存分单失败';
      message.error(msg);
    }
  };

  // 详情模式：保存共享 MBL 内容
  const handleSaveMblContent = async () => {
    if (!orderId || !isDetail || !mblDetail?.id) return;
    const contentValues = form.getFieldValue(
      'seaMasterBillContent',
    ) as API.SeaBillContent;
    try {
      await seaDocumentServiceUpdateSeaMasterBillContent(
        { orderId: String(orderId) },
        {
          orderId: String(orderId),
          expectedMblVersion: String(mblDetail.version || '1'),
          content: contentValues,
        },
      );
      message.success('主单内容保存成功');
      await loadOrderDocuments();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '保存主单内容失败';
      message.error(msg);
    }
  };

  // 显式用分配汇总填入目标 HBL
  const handleApplyHblSummary = async (
    houseBillId: string,
    houseBillVersion: number,
  ) => {
    if (!orderId || !isDetail) return;
    setApplyingHblId(houseBillId);
    try {
      const allocRes = await seaCargoAllocationServiceGetSeaCargoAllocation({
        orderId: String(orderId),
      });
      const allocVersion = allocRes.data?.allocationVersion;
      if (!allocVersion) {
        throw new Error('箱货分配版本缺失，请刷新后重试');
      }
      await seaCargoAllocationServiceApplySeaHouseBillAllocationSummary(
        { orderId: String(orderId), houseBillId },
        {
          orderId: String(orderId),
          houseBillId,
          expectedAllocationVersion: allocVersion,
          expectedHouseBillVersion: String(houseBillVersion),
        },
      );
      message.success('已用分配汇总填入本张分单件重尺');
      await loadOrderDocuments();
    } catch (err: any) {
      message.error(err.message || '填入分单汇总失败');
    } finally {
      setApplyingHblId(null);
    }
  };

  // DIRECT 下显式用操作票货物汇总填入 MBL
  const handleApplyMblSummary = async () => {
    if (!orderId || !isDetail || !mblDetail) return;
    setApplyingMbl(true);
    if (!mblDetail.version) {
      message.error('主单版本缺失，请刷新后重试');
      return;
    }
    try {
      await seaCargoAllocationServiceApplySeaOrderCargoSummaryToMasterBill(
        { orderId: String(orderId) },
        {
          orderId: String(orderId),
          expectedMblVersion: String(mblDetail.version),
        },
      );
      message.success('已用操作票货物汇总填入主单件重尺');
      await loadOrderDocuments();
    } catch (err: any) {
      message.error(err.message || '填入主单汇总失败');
    } finally {
      setApplyingMbl(false);
    }
  };

  // 顶部结构 Tag
  const renderStructureTag = () => {
    switch (docStructure) {
      case SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT:
        return <Tag color="success">直单 (DIRECT)</Tag>;
      case SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE:
        return <Tag color="processing">分单 (HOUSE)</Tag>;
      default:
        return <Tag color="default">未确定 (UNDETERMINED)</Tag>;
    }
  };

  // Tab 项构造
  const tabItems = [
    {
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
          variant="borderless"
          style={{ background: '#fafafa', borderRadius: 4 }}
        >
          {isDetail && mblDetail ? (
            <Row gutter={[16, 8]} style={{ marginBottom: 16 }}>
              <Col xs={24} md={6}>
                <Text type="secondary">主单号：</Text>
                <Text strong>{mblDetail.masterNo}</Text>
              </Col>
              <Col xs={24} md={6}>
                <Text type="secondary">签发主体：</Text>
                <Text strong>{mblDetail.issuerPartnerName || '-'}</Text>
              </Col>
              <Col xs={24} md={6}>
                <Text type="secondary">共享订单数：</Text>
                <Tag color="blue">{mblDetail.memberCount ?? 1} 票</Tag>
              </Col>
              <Col xs={24} md={6}>
                <Text type="secondary">主单版本：</Text>
                <Tag>v{mblDetail.version}</Tag>
              </Col>
              <Col span={24} style={{ textAlign: 'right' }}>
                <SeaDocumentHistoryActions
                  orderId={String(orderId)}
                  orderVersion={String(orderVersion ?? '')}
                  documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL}
                  documentId={mblDetail.id ?? ''}
                  documentNo={mblDetail.masterNo ?? ''}
                  documentVersion={String(mblDetail.version ?? '')}
                  currentVersionId={mblDetail.currentVersionId}
                  documentStatus={mblDetail.status}
                  getAmendmentInput={() => ({
                    masterBillContent:
                      (form.getFieldValue(
                        'seaMasterBillContent',
                      ) as API.SeaBillContent) ?? {},
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

          {isDetail &&
          mblDetail &&
          mblDetail.status !== 'VOIDED' &&
          !disabled ? (
            <div style={{ textAlign: 'right', marginTop: 12 }}>
              <Space>
                {docStructure ===
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT ? (
                  <Button loading={applyingMbl} onClick={handleApplyMblSummary}>
                    用操作票货物汇总填入 MBL
                  </Button>
                ) : null}
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  onClick={handleSaveMblContent}
                >
                  保存主单内容
                </Button>
              </Space>
            </div>
          ) : null}
        </Card>
      ),
    },
    ...houseBills.map((hb, idx) => ({
      key: `hbl-${idx}`,
      label: <HouseBillTabTitle index={idx} />,
      children: (
        <Card
          size="small"
          variant="borderless"
          style={{ background: '#fafafa', borderRadius: 4 }}
        >
          {/* HBL 头部状态与操作 */}
          <Row
            justify="space-between"
            align="middle"
            style={{ marginBottom: 16 }}
          >
            <Col>
              <Space>
                <Text strong style={{ fontSize: 15 }}>
                  分单 #{idx + 1}
                </Text>
                {hb.status ? (
                  <Tag color={houseBillStatusPresentation(hb.status).color}>
                    {houseBillStatusPresentation(hb.status).text}
                  </Tag>
                ) : null}
                {hb.version ? <Tag>v{hb.version}</Tag> : null}
              </Space>
            </Col>
            <Col>
              <Space>
                {isDetail && hb.id ? (
                  <SeaDocumentHistoryActions
                    orderId={String(orderId)}
                    orderVersion={String(orderVersion ?? '')}
                    documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL}
                    documentId={hb.id}
                    documentNo={hb.houseNo ?? ''}
                    documentVersion={String(hb.version ?? '')}
                    currentVersionId={hb.currentVersionId}
                    currentHouseBill={hb}
                    getAmendmentInput={() => {
                      const input = form.getFieldValue([
                        'seaHouseBills',
                        idx,
                      ]) as API.SeaHouseBillInput;
                      return {
                        houseBill: {
                          ...input,
                          id: hb.id,
                          houseNo: input?.houseNo ?? hb.houseNo ?? '',
                          issuerSource:
                            input?.issuerSource ?? hb.issuerSource ?? 0,
                          content: input?.content ?? hb.content ?? {},
                          expectedVersion: hb.version,
                        },
                      };
                    }}
                    disabled={disabled}
                    onSuccess={async () => {
                      await onOrderDataChanged?.();
                      await loadOrderDocuments();
                    }}
                  />
                ) : null}
                {idx > 0 && !disabled && !isTerminalHouseBill(hb.status) ? (
                  <Button
                    size="small"
                    icon={<CopyOutlined />}
                    onClick={() => handleCopyPreviousContent(idx)}
                  >
                    复制上一张内容
                  </Button>
                ) : null}
                {!disabled && !isTerminalHouseBill(hb.status) ? (
                  <Button
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={() => handleRemoveHouseBill(idx)}
                  >
                    删除分单
                  </Button>
                ) : null}
              </Space>
            </Col>
          </Row>

          {/* HBL 基础信息（号、签发主体、备注） */}
          <Row gutter={[16, 0]}>
            <Col xs={24} md={8}>
              <ProFormText
                name={['seaHouseBills', idx, 'houseNo']}
                label="分单号 (HBL No.)"
                placeholder="请输入分单号"
                disabled={disabled || isTerminalHouseBill(hb.status)}
                rules={[{ required: true, message: '分单号不能为空' }]}
                fieldProps={{ maxLength: 128 }}
              />
            </Col>
            <Col xs={24} md={16}>
              <Form.Item label="签发主体" required style={{ marginBottom: 24 }}>
                <Form.Item
                  name={['seaHouseBills', idx, 'issuerSource']}
                  noStyle
                  rules={[{ required: true, message: '请选择签发主体' }]}
                >
                  <Radio.Group
                    disabled={disabled || isTerminalHouseBill(hb.status)}
                    onChange={() =>
                      form.setFieldValue(
                        ['seaHouseBills', idx, 'issuerPartnerId'],
                        undefined,
                      )
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
                  shouldUpdate={(prev, cur) =>
                    prev?.seaHouseBills?.[idx]?.issuerSource !==
                    cur?.seaHouseBills?.[idx]?.issuerSource
                  }
                >
                  {({ getFieldValue }) => {
                    const src = getFieldValue([
                      'seaHouseBills',
                      idx,
                      'issuerSource',
                    ]);
                    if (
                      src ===
                      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION
                    ) {
                      return (
                        <div style={{ marginTop: 6 }}>
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            💡 由所属公司或总部自动统一签发
                          </Text>
                        </div>
                      );
                    }
                    if (
                      src ===
                      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER
                    ) {
                      return (
                        <div style={{ marginTop: 6 }}>
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            💡 使用当前订单委托单位作为签发主体
                          </Text>
                        </div>
                      );
                    }
                    if (
                      src ===
                      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
                    ) {
                      return (
                        <div style={{ marginTop: 8 }}>
                          <ProFormSearchableSelect
                            name={['seaHouseBills', idx, 'issuerPartnerId']}
                            placeholder="请选择签发主体合作伙伴"
                            disabled={
                              disabled || isTerminalHouseBill(hb.status)
                            }
                            rules={[
                              { required: true, message: '请选择合作伙伴' },
                            ]}
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
                name={['seaHouseBills', idx, 'note']}
                label="分单备注"
                placeholder="请输入分单备注"
                disabled={disabled || isTerminalHouseBill(hb.status)}
                fieldProps={{ maxLength: 500 }}
              />
            </Col>
          </Row>

          {/* HBL 15 个提单内容字段 */}
          <div style={{ marginTop: 12 }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              提单正文内容
            </Text>
            <SeaBillContentFormFields
              namePathPrefix={['seaHouseBills', idx, 'content']}
              disabled={disabled || isTerminalHouseBill(hb.status)}
            />
          </div>
          {renderReleasePods(
            relatedReleasePods(
              SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
              hb.id,
            ),
          )}

          {isDetail && !disabled && !isTerminalHouseBill(hb.status) ? (
            <div style={{ textAlign: 'right', marginTop: 12 }}>
              <Space>
                {hb.id ? (
                  <Button
                    loading={applyingHblId === hb.id}
                    onClick={() => {
                      const targetHbId = hb.id;
                      if (targetHbId) {
                        handleApplyHblSummary(targetHbId, Number(hb.version));
                      }
                    }}
                  >
                    用分配汇总填入本张 HBL
                  </Button>
                ) : null}
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  onClick={() => handleSaveHouseBill(idx)}
                >
                  保存此分单
                </Button>
              </Space>
            </div>
          ) : null}
        </Card>
      ),
    })),
  ];

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

      {/* 单证结构状态与顶栏动作 */}
      <Card
        size="small"
        variant="outlined"
        style={{ marginBottom: 16, borderColor: '#f0f0f0' }}
      >
        <Row justify="space-between" align="middle">
          <Col>
            <Space size="middle">
              <Text strong>单证结构：</Text>
              {renderStructureTag()}
              {linkVersion && linkVersion !== '0' ? (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  (单证版本 v{linkVersion})
                </Text>
              ) : null}
            </Space>
          </Col>
          <Col>
            <Space>
              {docStructure ===
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNDETERMINED &&
              !disabled &&
              !fetchError ? (
                <>
                  <Button onClick={handleMarkDirect}>标记为直单</Button>
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={handleAddHouseBill}
                  >
                    添加首张分单
                  </Button>
                </>
              ) : null}

              {docStructure ===
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT &&
              !disabled &&
              !fetchError ? (
                <Button onClick={handleCancelDirect}>
                  取消直单标记 (回未确定)
                </Button>
              ) : null}

              {docStructure ===
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE &&
              !disabled &&
              !fetchError ? (
                <>
                  <Button
                    icon={<AppstoreOutlined />}
                    onClick={() => {
                      if (orderId && allocationDrawerRef.current) {
                        allocationDrawerRef.current.open({
                          id: String(orderId),
                          orderNo: form.getFieldValue('orderNo'),
                        });
                      }
                    }}
                  >
                    箱货分配
                  </Button>
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={handleAddHouseBill}
                  >
                    添加分单 (HBL)
                  </Button>
                </>
              ) : null}
            </Space>
          </Col>
        </Row>

        {docStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT ? (
          <Alert
            style={{ marginTop: 12 }}
            type="info"
            showIcon
            title="当前为直单（DIRECT）结构"
            description="主单直接签发给最终收货人，禁止添加分单。如需签发分单，请先点击右上角『取消直单标记』回到未确定状态。"
          />
        ) : null}
      </Card>

      {/* MBL / HBL Tabs */}
      <Tabs
        type="card"
        activeKey={activeTabKey}
        onChange={setActiveTabKey}
        items={tabItems}
      />

      <SeaCargoAllocationDrawer
        ref={allocationDrawerRef}
        canManage={
          !disabled &&
          access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'update')
        }
        onSuccess={loadOrderDocuments}
      />
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
