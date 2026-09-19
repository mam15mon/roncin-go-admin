import {
  CheckCircleOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Radio,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import Decimal from 'decimal.js';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { SectionCard, StickyFooterBar } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaOrderSplit,
  seaOrderChangeServiceGetSeaOrderSplitContext,
  seaOrderChangeServicePreviewSeaOrderSplit,
} from '@/services/roncin/seaOrderChangeService';
import { computeCanonicalSha256 } from '@/utils/hash';
import { searchShippingLineOptions } from '@/utils/options';
import OrderPageHeader from './components/OrderPageHeader';
import SplitAllocationSection from './components/split/SplitAllocationSection';
import SplitAttachmentsAndNotesSection from './components/split/SplitAttachmentsAndNotesSection';
import SplitBaselineSection from './components/split/SplitBaselineSection';
import SplitConservationSection from './components/split/SplitConservationSection';
import SplitFeesSection from './components/split/SplitFeesSection';
import SplitResultsSection from './components/split/SplitResultsSection';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from './templates/components/sea/SeaExternalConfirmationFields';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

const { Text } = Typography;
const { TextArea } = Input;

// 纯函数与类型已抽至 splitUtils；此处 re-export 维持测试与既有导入路径稳定。
export {
  buildSeaOrderSplitTargets,
  calculateFeeCurrencySummaries,
  type ResultConfig,
} from './splitUtils';

import {
  buildSeaOrderSplitTargets,
  calculateFeeCurrencySummaries,
  type FeeCurrencySummary,
  getErrorMessage,
  type ResultConfig,
} from './splitUtils';

export default function SeaOrderSplitPage() {
  const params = useParams<{ id: string }>();
  const orderId = params.id || '';
  const { message } = App.useApp();
  const access = useAccess();
  const canReassign = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'reassign',
  );
  const {
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
    refresh: refreshLockState,
  } = useOrderLockState(orderId);
  const lockWritePolicy = getOrderBusinessWritePolicy({
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
  });
  const lockWritePolicyRef = useRef(lockWritePolicy);
  lockWritePolicyRef.current = lockWritePolicy;

  const ensureSplitEditable = () => {
    const currentPolicy = lockWritePolicyRef.current;
    if (!currentPolicy.disabled) return true;
    message.warning(currentPolicy.reason || '订单当前不可拆票');
    return false;
  };

  const [loadingContext, setLoadingContext] = useState(false);
  const [splitContext, setSplitContext] =
    useState<API.SeaOrderSplitContextData | null>(null);

  // 拆票结果集：至少 1 个 ORIGINAL + 1 个 CREATED
  const [results, setResults] = useState<ResultConfig[]>([]);

  // 分配状态
  const [containerAssignments, setContainerAssignments] = useState<
    Record<string, string>
  >({}); // containerId -> resultKey
  const [cargoAllocations, setCargoAllocations] = useState<
    Record<
      string,
      Record<
        string,
        { packageCount: number; grossWeightKg: string; volumeCbm: string }
      >
    >
  >({}); // cargoItemId -> resultKey -> { packageCount, grossWeightKg, volumeCbm }
  const [sharedAllocations, setSharedAllocations] = useState<
    Record<
      string,
      Record<
        string,
        { packageCount: number; grossWeightKg: string; volumeCbm: string }
      >
    >
  >({}); // allocationId -> resultKey -> { packageCount, grossWeightKg, volumeCbm }
  const [feeAssignments, setFeeAssignments] = useState<Record<string, string>>(
    {},
  ); // feeId -> resultKey
  const [attAssignments, setAttAssignments] = useState<
    Record<string, string[]>
  >({}); // attId -> resultKeys[]
  const initialPreviewTriggeredRef = useRef(false);
  const [note, setNote] = useState<string>('');
  const [confirmationForm] = Form.useForm<SeaExternalConfirmationFormValues>();

  // 预览与校验结果
  const [previewing, setPreviewing] = useState(false);
  const [previewData, setPreviewData] =
    useState<API.SeaOrderSplitPreviewData | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 下拉选项
  const [carrierOptions, setCarrierOptions] = useState<DefaultOptionType[]>([]);

  const feeCurrencySummaries = useMemo(
    () =>
      calculateFeeCurrencySummaries(
        splitContext?.draftFees || [],
        feeAssignments,
        results.map((result) => result.key),
      ),
    [splitContext?.draftFees, feeAssignments, results],
  );

  // 加载拆票上下文
  const loadContext = async () => {
    if (!orderId) return;
    setLoadingContext(true);
    try {
      const resp = await seaOrderChangeServiceGetSeaOrderSplitContext({
        orderId,
      });
      if (resp?.data) {
        const ctx = resp.data;
        setSplitContext(ctx);
        if (ctx.currentMasterBill?.shippingLineId) {
          setCarrierOptions([
            {
              label:
                ctx.currentMasterBill.shippingLineName ||
                ctx.currentMasterBill.shippingLineId,
              value: ctx.currentMasterBill.shippingLineId,
            },
          ]);
        }

        const defaultHouseNo = ctx.currentHouseBill?.houseNo
          ? `${ctx.currentHouseBill.houseNo}-1`
          : 'HBL-1';

        // 初始化结果
        const initialResults: ResultConfig[] = [
          {
            key: 'res-origin',
            role: 'ORIGINAL',
            title: `原票 (${ctx.orderNo})`,
            targetType: 'CURRENT',
            internalReferenceNo: ctx.internalReferenceNo,
            bookingNotes: ctx.bookingNotes,
            allocationNotes: ctx.allocationNotes,
            operationNotes: ctx.operationNotes,
          },
          {
            key: 'res-new-1',
            role: 'CREATED',
            title: '拆出新票 1',
            targetType: 'CURRENT',
            internalReferenceNo: '',
            bookingNotes: ctx.bookingNotes,
            allocationNotes: ctx.allocationNotes || '',
            operationNotes: ctx.operationNotes,
            houseNo: defaultHouseNo,
            issuerSource: 'SELF_ORGANIZATION',
          },
        ];
        setResults(initialResults);

        // 初始化集装箱整箱归属：默认全在原票
        const initialCntrMap: Record<string, string> = {};
        ctx.containers?.forEach((c: API.SeaOrderSplitContainerItem) => {
          if (c.id) {
            initialCntrMap[c.id] = 'res-origin';
          }
        });
        setContainerAssignments(initialCntrMap);

        // 初始化货物件重尺分配：原票全量，新票 0
        const initialCargoAlloc: Record<
          string,
          Record<
            string,
            { packageCount: number; grossWeightKg: string; volumeCbm: string }
          >
        > = {};
        ctx.cargoItems?.forEach((ci: API.SeaOrderSplitCargoItem) => {
          if (ci.id) {
            initialCargoAlloc[ci.id] = {
              'res-origin': {
                packageCount: ci.packageCount || 0,
                grossWeightKg: String(ci.grossWeightKg || '0'),
                volumeCbm: String(ci.volumeCbm || '0'),
              },
              'res-new-1': {
                packageCount: 0,
                grossWeightKg: '0',
                volumeCbm: '0',
              },
            };
          }
        });
        setCargoAllocations(initialCargoAlloc);

        // 初始化共享箱分配：原票全量，新票 0
        const initialSharedAlloc: Record<
          string,
          Record<
            string,
            { packageCount: number; grossWeightKg: string; volumeCbm: string }
          >
        > = {};
        ctx.sharedContainerAllocations?.forEach(
          (sa: API.SeaOrderSplitSharedContainerAllocationItem) => {
            if (sa.allocationId) {
              initialSharedAlloc[sa.allocationId] = {
                'res-origin': {
                  packageCount: sa.packageCount || 0,
                  grossWeightKg: String(sa.grossWeightKg || '0'),
                  volumeCbm: String(sa.volumeCbm || '0'),
                },
                'res-new-1': {
                  packageCount: 0,
                  grossWeightKg: '0',
                  volumeCbm: '0',
                },
              };
            }
          },
        );
        setSharedAllocations(initialSharedAlloc);

        // 费用默认全部留原票
        const initialFeeMap: Record<string, string> = {};
        ctx.draftFees?.forEach((f: API.SeaOrderSplitDraftFeeItem) => {
          if (f.id) {
            initialFeeMap[f.id] = 'res-origin';
          }
        });
        setFeeAssignments(initialFeeMap);

        // 附件默认全部分配到原票
        const initialAttMap: Record<string, string[]> = {};
        ctx.attachments?.forEach((a: API.SeaOrderSplitAttachmentItem) => {
          if (a.id) {
            initialAttMap[a.id] = ['res-origin'];
          }
        });
        setAttAssignments(initialAttMap);
      }
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '加载拆票上下文失败'));
    } finally {
      setLoadingContext(false);
    }
  };

  useEffect(() => {
    loadContext();
  }, [orderId]);

  // 页签占位标题为中性「订单拆票」，加载成功后回填带单号的真实标题。
  useEffect(() => {
    if (splitContext?.orderNo && orderId && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: `/orders/sea-export/${orderId}/split`,
            title: `${splitContext.orderNo}_拆票`,
          },
        }),
      );
    }
  }, [splitContext?.orderNo, orderId]);

  // 构造母单目标
  const buildTargets = (
    currentResults = results,
  ): API.SeaOrderSplitTargetInput[] =>
    buildSeaOrderSplitTargets(currentResults);

  // 构造拆票结果明细
  const buildSplitResults = (
    currentResults = results,
    currentContainers = containerAssignments,
    currentCargoAllocs = cargoAllocations,
    currentSharedAllocs = sharedAllocations,
    currentFees = feeAssignments,
    currentAtts = attAssignments,
  ): API.SeaOrderSplitResultInput[] => {
    return currentResults.map((r) => {
      const feeIds = Object.entries(currentFees)
        .filter(([, resKey]) => resKey === r.key)
        .map(([fId]) => fId);

      const attIds = Object.entries(currentAtts)
        .filter(([, resKeys]) => resKeys.includes(r.key))
        .map(([aId]) => aId);

      const cIds = Object.entries(currentContainers)
        .filter(([, resKey]) => resKey === r.key)
        .map(([cId]) => cId);

      const cargoAllocs = (splitContext?.cargoItems || []).map((ci) => {
        const a = currentCargoAllocs[ci.id || '']?.[r.key] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        return {
          cargoItemId: ci.id || '',
          packageCount: Number(a.packageCount) || 0,
          grossWeightKg: String(a.grossWeightKg || '0'),
          volumeCbm: String(a.volumeCbm || '0'),
        };
      });

      const sharedAllocs = (splitContext?.sharedContainerAllocations || []).map(
        (sa) => {
          const a = currentSharedAllocs[sa.allocationId || '']?.[r.key] || {
            packageCount: 0,
            grossWeightKg: '0',
            volumeCbm: '0',
          };
          return {
            allocationId: sa.allocationId || '',
            packageCount: Number(a.packageCount) || 0,
            grossWeightKg: String(a.grossWeightKg || '0'),
            volumeCbm: String(a.volumeCbm || '0'),
          };
        },
      );

      let houseBill: API.SeaOrderSplitHouseBillInput | undefined;
      if (r.role === 'CREATED' && splitContext?.documentStructure === 'HOUSE') {
        houseBill = {
          houseNo: r.houseNo || '',
          issuerSource: r.issuerSource || 'SELF_ORGANIZATION',
          issuerPartnerId: r.issuerPartnerId,
          note: r.houseBillNote,
        };
      }

      return {
        clientResultKey: r.key,
        resultRole: r.role,
        clientTargetKey: r.key,
        draftFeeIds: feeIds,
        attachmentReferenceIds: attIds,
        containerIds: cIds,
        cargoAllocations: cargoAllocs,
        sharedContainerAllocations:
          sharedAllocs.length > 0 ? sharedAllocs : undefined,
        houseBill,
        internalReferenceNo: r.internalReferenceNo,
        bookingNotes: r.bookingNotes,
        allocationNotes: r.allocationNotes,
        operationNotes: r.operationNotes,
      };
    });
  };

  // 构造全量预期版本字典
  const buildExpectedVersions = (
    currentResults = results,
  ): API.SeaOrderSplitExpectedVersions | undefined => {
    if (!splitContext) return undefined;
    if (!splitContext.orderVersion || !splitContext.currentLinkVersion) {
      return undefined;
    }
    const cargoItemVersions: Record<string, string> = {};
    for (const ci of splitContext.cargoItems || []) {
      if (!ci.id || !ci.version) return undefined;
      cargoItemVersions[ci.id] = String(ci.version);
    }
    const containerVersions: Record<string, string> = {};
    for (const c of splitContext.containers || []) {
      if (!c.id || !c.version) return undefined;
      containerVersions[c.id] = String(c.version);
    }
    const feeVersions: Record<string, string> = {};
    for (const f of splitContext.draftFees || []) {
      if (!f.id || !f.version) return undefined;
      feeVersions[f.id] = String(f.version);
    }
    const sharedContainerVersions: Record<string, string> = {};
    for (const sa of splitContext.sharedContainerAllocations || []) {
      if (sa.sharedContainerId && sa.sharedContainerVersion) {
        sharedContainerVersions[sa.sharedContainerId] = String(
          sa.sharedContainerVersion,
        );
      }
    }
    const candidateMblVersions: Record<string, string> = {};
    const candidateTeVersions: Record<string, string> = {};
    for (const r of currentResults) {
      if (r.targetType === 'CANDIDATE') {
        if (
          !r.candidateId ||
          !r.candidateVersion ||
          !r.candidateTeId ||
          !r.candidateTeVersion
        ) {
          return undefined;
        }
        candidateMblVersions[r.candidateId] = String(r.candidateVersion);
        candidateTeVersions[r.candidateTeId] = String(r.candidateTeVersion);
      }
    }

    return {
      orderVersion: String(splitContext.orderVersion),
      linkVersion: String(splitContext.currentLinkVersion),
      currentHblVersion: splitContext.currentHouseBill?.version
        ? String(splitContext.currentHouseBill.version)
        : undefined,
      cargoItemVersions,
      containerVersions,
      feeVersions,
      sharedContainerVersions:
        Object.keys(sharedContainerVersions).length > 0
          ? sharedContainerVersions
          : undefined,
      candidateMblVersions,
      candidateTeVersions,
      attachmentReferenceFingerprint:
        splitContext.attachmentReferenceFingerprint,
    };
  };

  // 触发校验与预览
  const triggerPreview = async (
    currentResults = results,
    currentContainers = containerAssignments,
    currentCargoAllocs = cargoAllocations,
    currentSharedAllocs = sharedAllocations,
    currentFees = feeAssignments,
    currentAtts = attAssignments,
  ) => {
    if (
      lockWritePolicyRef.current.disabled ||
      !orderId ||
      !splitContext ||
      currentResults.length < 2
    )
      return;
    setPreviewing(true);
    setPreviewError(null);
    try {
      const targets = buildTargets(currentResults);
      const splitResults = buildSplitResults(
        currentResults,
        currentContainers,
        currentCargoAllocs,
        currentSharedAllocs,
        currentFees,
        currentAtts,
      );
      const expectedVersions = buildExpectedVersions(currentResults);
      if (!expectedVersions) {
        setPreviewError(
          '缺少完整版本控制信息或候选版本未获取，无法进行拆票校验',
        );
        setPreviewData(null);
        return;
      }

      const resp = await seaOrderChangeServicePreviewSeaOrderSplit(
        { orderId },
        {
          orderId,
          note: note ? note.trim() : undefined,
          targets,
          results: splitResults,
          expectedVersions,
        },
      );

      if (resp?.data) {
        setPreviewData(resp.data);
      }
    } catch (error: unknown) {
      setPreviewError(getErrorMessage(error, '拆票校验未通过'));
      setPreviewData(null);
    } finally {
      setPreviewing(false);
    }
  };

  // 依赖变化时防抖预览
  useEffect(() => {
    if (splitContext && results.length >= 2) {
      if (!initialPreviewTriggeredRef.current) {
        initialPreviewTriggeredRef.current = true;
        triggerPreview();
        return undefined;
      }
      const timer = setTimeout(() => {
        triggerPreview();
      }, 300);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [
    results,
    containerAssignments,
    cargoAllocations,
    sharedAllocations,
    feeAssignments,
    attAssignments,
    note,
    lockWritePolicy.disabled,
  ]);

  useEffect(() => {
    if (lockWritePolicy.disabled) {
      setPreviewData(null);
    }
  }, [lockWritePolicy.disabled]);

  // 添加新票
  const handleAddResult = () => {
    if (!ensureSplitEditable()) return;
    const nextIdx = results.filter((r) => r.role === 'CREATED').length + 1;
    const newKey = `res-new-${Date.now()}`;
    const defaultHouseNo = splitContext?.currentHouseBill?.houseNo
      ? `${splitContext.currentHouseBill.houseNo}-${nextIdx}`
      : `HBL-${nextIdx}`;
    const newRes: ResultConfig = {
      key: newKey,
      role: 'CREATED',
      title: `拆出新票 ${nextIdx}`,
      targetType: 'CURRENT',
      internalReferenceNo: '',
      bookingNotes: splitContext?.bookingNotes,
      allocationNotes: splitContext?.allocationNotes || '',
      operationNotes: splitContext?.operationNotes,
      houseNo: defaultHouseNo,
      issuerSource: 'SELF_ORGANIZATION',
    };
    const updated = [...results, newRes];
    setResults(updated);

    setCargoAllocations((prev) => {
      const next = { ...prev };
      for (const ciId of Object.keys(next)) {
        next[ciId] = {
          ...next[ciId],
          [newKey]: { packageCount: 0, grossWeightKg: '0', volumeCbm: '0' },
        };
      }
      return next;
    });

    setSharedAllocations((prev) => {
      const next = { ...prev };
      for (const saId of Object.keys(next)) {
        next[saId] = {
          ...next[saId],
          [newKey]: { packageCount: 0, grossWeightKg: '0', volumeCbm: '0' },
        };
      }
      return next;
    });
  };

  // 移除新票
  const handleRemoveResult = (key: string) => {
    if (!ensureSplitEditable()) return;
    if (results.filter((r) => r.role === 'CREATED').length <= 1) {
      message.warning('拆票必须至少保留一个拆出新票');
      return;
    }
    const updated = results.filter((r) => r.key !== key);
    const nextContainers = { ...containerAssignments };
    Object.keys(nextContainers).forEach((cId) => {
      if (nextContainers[cId] === key) nextContainers[cId] = 'res-origin';
    });
    const nextFees = { ...feeAssignments };
    Object.keys(nextFees).forEach((fId) => {
      if (nextFees[fId] === key) nextFees[fId] = 'res-origin';
    });
    setResults(updated);
    setContainerAssignments(nextContainers);
    setFeeAssignments(nextFees);

    setCargoAllocations((prev) => {
      const next = { ...prev };
      for (const ci of splitContext?.cargoItems || []) {
        if (!ci.id || !next[ci.id]) continue;
        const removed = next[ci.id][key] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const origin = next[ci.id]['res-origin'] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const pkg =
          (Number(origin.packageCount) || 0) +
          (Number(removed.packageCount) || 0);
        const wt = new Decimal(origin.grossWeightKg || '0')
          .add(new Decimal(removed.grossWeightKg || '0'))
          .toString();
        const vol = new Decimal(origin.volumeCbm || '0')
          .add(new Decimal(removed.volumeCbm || '0'))
          .toString();
        const { [key]: _, ...rest } = next[ci.id];
        next[ci.id] = {
          ...rest,
          'res-origin': {
            packageCount: pkg,
            grossWeightKg: wt,
            volumeCbm: vol,
          },
        };
      }
      return next;
    });

    setSharedAllocations((prev) => {
      const next = { ...prev };
      for (const sa of splitContext?.sharedContainerAllocations || []) {
        if (!sa.allocationId || !next[sa.allocationId]) continue;
        const removed = next[sa.allocationId][key] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const origin = next[sa.allocationId]['res-origin'] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const pkg =
          (Number(origin.packageCount) || 0) +
          (Number(removed.packageCount) || 0);
        const wt = new Decimal(origin.grossWeightKg || '0')
          .add(new Decimal(removed.grossWeightKg || '0'))
          .toString();
        const vol = new Decimal(origin.volumeCbm || '0')
          .add(new Decimal(removed.volumeCbm || '0'))
          .toString();
        const { [key]: _, ...rest } = next[sa.allocationId];
        next[sa.allocationId] = {
          ...rest,
          'res-origin': {
            packageCount: pkg,
            grossWeightKg: wt,
            volumeCbm: vol,
          },
        };
      }
      return next;
    });
  };

  // 执行拆票提交
  const handleExecuteSplit = async () => {
    if (!ensureSplitEditable()) return;
    if (!previewData?.isValid || !previewData?.conservationPassed) {
      message.error(
        previewError || '拆票数据未满足守恒校验或存在门禁错误，请检查！',
      );
      return;
    }
    // 任一结果换入其他母单（含新建）即产生内嵌改配，必须携带外部确认
    let confirmation: API.SeaExternalConfirmationInput | undefined;
    if (results.some((r) => r.targetType !== 'CURRENT')) {
      try {
        confirmation = buildSeaExternalConfirmation(
          await confirmationForm.validateFields(),
        );
      } catch {
        message.error('请完整填写承运方外部确认信息后再执行拆票');
        return;
      }
    }
    setSubmitting(true);
    try {
      const targets = buildTargets(results);
      const splitResults = buildSplitResults(
        results,
        containerAssignments,
        cargoAllocations,
        sharedAllocations,
        feeAssignments,
        attAssignments,
      );
      const expectedVersions = buildExpectedVersions(results);
      if (!expectedVersions) {
        message.error('未获取到有效拆票版本信息，请刷新重试！');
        return;
      }

      // 稳定指纹与幂等键：同参数输入生成稳定 SHA-256 哈希，参数变化生成新 key
      const payloadForHash = {
        orderId,
        targets,
        results: splitResults,
        note: note ? note.trim() : undefined,
        expectedVersions,
        confirmation,
      };
      const hash = computeCanonicalSha256(payloadForHash);
      const fingerprint = `split-fp:${hash}`;
      const idempotencyKey = `split:${orderId}:${hash}`;

      const resp = await seaOrderChangeServiceExecuteSeaOrderSplit(
        { orderId },
        {
          orderId,
          idempotencyKey,
          requestFingerprint: fingerprint,
          note: note ? note.trim() : undefined,
          targets,
          results: splitResults,
          expectedVersions,
          confirmation,
        },
      );

      const createdCount = resp?.data?.createdOrders?.length || 0;
      message.success(`拆票成功！原票已更新，成功生成 ${createdCount} 张新票`);
      history.push(`/orders/sea-export/${orderId}`);
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '执行拆票失败'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageContainer
      title={false}
      breadcrumbRender={false}
      header={{ title: false, style: { padding: 0 } }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      <Spin spinning={loadingContext}>
        {/* 顶部 OrderPageHeader */}
        <OrderPageHeader
          page="split"
          orderKind="sea-export"
          navigationTitle="海运出口"
          orderId={orderId}
          orderNo={splitContext?.orderNo}
          subTitle="支持整单部分拆票、HBL/箱货零误差守恒切分、草稿费用整行归属及多票并行派生"
          tags={<Tag color="blue">海运出口 (HOUSE)</Tag>}
          extra={
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
                void loadContext();
                void refreshLockState();
              }}
            >
              刷新数据
            </Button>
          }
        />

        <div style={{ paddingBottom: 80 }}>
          {lockWritePolicy.disabled && (
            <Alert
              type={
                lockWritePolicy.reason?.includes('已锁定') ? 'warning' : 'error'
              }
              showIcon
              message="拆票操作当前不可用"
              description={lockWritePolicy.reason}
              action={
                <Button size="small" onClick={() => void refreshLockState()}>
                  重试锁状态
                </Button>
              }
              style={{ marginBottom: 16 }}
            />
          )}
          <div
            aria-disabled={lockWritePolicy.disabled}
            style={
              lockWritePolicy.disabled
                ? { pointerEvents: 'none', opacity: 0.72 }
                : undefined
            }
          >
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              {previewError && (
                <Alert
                  type="error"
                  showIcon
                  message="拆票校验未通过"
                  description={previewError}
                />
              )}

              <SplitBaselineSection
                previewData={previewData}
                splitContext={splitContext}
              />

              <SplitResultsSection
                results={results}
                setResults={setResults}
                splitContext={splitContext}
                canReassign={canReassign}
                carrierOptions={carrierOptions}
                setCarrierOptions={setCarrierOptions}
                message={message}
                triggerPreview={triggerPreview}
                onAddResult={handleAddResult}
                onRemoveResult={handleRemoveResult}
              />

              <SplitAllocationSection
                splitContext={splitContext}
                results={results}
                containerAssignments={containerAssignments}
                setContainerAssignments={setContainerAssignments}
                cargoAllocations={cargoAllocations}
                setCargoAllocations={setCargoAllocations}
                sharedAllocations={sharedAllocations}
                setSharedAllocations={setSharedAllocations}
              />

              <SplitFeesSection
                splitContext={splitContext}
                results={results}
                feeAssignments={feeAssignments}
                setFeeAssignments={setFeeAssignments}
                feeCurrencySummaries={feeCurrencySummaries}
              />

              <SplitAttachmentsAndNotesSection
                splitContext={splitContext}
                results={results}
                attAssignments={attAssignments}
                setAttAssignments={setAttAssignments}
                note={note}
                setNote={setNote}
                orderId={orderId}
                confirmationForm={confirmationForm}
              />

              <SplitConservationSection
                previewData={previewData}
                previewing={previewing}
              />
            </Space>
          </div>
        </div>

        {/* 底部吸底操作栏 StickyFooterBar */}
        <StickyFooterBar
          info={
            <Space size="large">
              <Text strong>拆票规划：{results.length} 票</Text>
              <Text type="secondary">
                分配进度：{previewData?.allocated?.packageCount ?? 0} /{' '}
                {previewData?.baseline?.packageCount ?? 0} 件
              </Text>
              {previewData?.conservationPassed && previewData?.isValid ? (
                <Tag color="success">守恒验证通过</Tag>
              ) : (
                <Tag color="error">等待满足守恒条件</Tag>
              )}
            </Space>
          }
        >
          <Button onClick={() => history.push(`/orders/sea-export/${orderId}`)}>
            取消返回
          </Button>
          <Popconfirm
            title="确认提交拆票"
            description="拆票将原子锁定订单、分批移转货物、分单与费用并创建新订单。确定提交执行？"
            okText="确认执行"
            cancelText="取消"
            onConfirm={handleExecuteSplit}
            disabled={
              lockWritePolicy.disabled ||
              !previewData?.isValid ||
              !previewData?.conservationPassed
            }
          >
            <Button
              type="primary"
              loading={submitting}
              disabled={
                lockWritePolicy.disabled ||
                !previewData?.isValid ||
                !previewData?.conservationPassed
              }
            >
              确认执行拆票
            </Button>
          </Popconfirm>
        </StickyFooterBar>
      </Spin>
    </PageContainer>
  );
}
