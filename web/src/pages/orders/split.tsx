import { ReloadOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Form,
  Result,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import Decimal from 'decimal.js';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { useAccess } from '@/app/access';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { AsyncPopconfirm, StickyFooterBar } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import { usePageForeground } from '@/hooks/usePageForeground';
import { history } from '@/router/history';
import {
  seaOrderChangeServiceExecuteSeaOrderSplit,
  seaOrderChangeServiceGetSeaOrderSplitContext,
  seaOrderChangeServicePreviewSeaOrderSplit,
} from '@/services/roncin/seaOrderChangeService';
import { isWriteResponse } from '@/utils/api';
import { computeCanonicalSha256 } from '@/utils/hash';
import CargoItemDrawer, {
  type CargoItemDrawerRef,
} from './components/drawers/CargoItemDrawer';
import OrderPageHeader from './components/OrderPageHeader';
import SplitAllocationSection from './components/split/SplitAllocationSection';
import SplitAttachmentsAndNotesSection from './components/split/SplitAttachmentsAndNotesSection';
import SplitBaselineSection from './components/split/SplitBaselineSection';
import SplitConservationSection from './components/split/SplitConservationSection';
import SplitFeesSection from './components/split/SplitFeesSection';
import SplitResultsSection from './components/split/SplitResultsSection';
import {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from './templates/components/sea/SeaExternalConfirmationFields';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

const { Text } = Typography;

/** 拆票域前缀：跨组件失效按该前缀 invalidate。 */
const SPLIT_QUERY_PREFIX = 'order-split';

/**
 * 拆票预览输入：ready 表示可发起预览请求（完整请求体进 queryKey）；
 * missing-versions 表示缺少版本控制信息，只本地报错不发请求。
 */
type SplitPreviewInput =
  | { kind: 'ready'; request: API.PreviewSeaOrderSplitRequest }
  | { kind: 'missing-versions' };

// 纯函数与类型已抽至 splitUtils；此处 re-export 维持测试与既有导入路径稳定。
export {
  buildSeaOrderSplitTargets,
  calculateFeeCurrencySummaries,
  type ResultConfig,
} from './splitUtils';

import {
  buildSeaOrderSplitTargets,
  calculateFeeCurrencySummaries,
  getErrorMessage,
  type ResultConfig,
} from './splitUtils';

export default function SeaOrderSplitPage() {
  const params = useParams<{ id: string }>();
  const orderId = params.id || '';
  const { message, modal } = App.useApp();
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const cargoItemDrawerRef = useRef<CargoItemDrawerRef>(null);
  const access = useAccess();
  const canSplit = access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'split');
  const canReassign = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'reassign',
  );
  const canReadCargo = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'cargo_item.read',
  );
  const canCreateCargo = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'cargo_item.create',
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
    canOperate: canSplit,
  });
  const lockWritePolicyRef = useRef(lockWritePolicy);
  lockWritePolicyRef.current = lockWritePolicy;

  const ensureSplitEditable = () => {
    const currentPolicy = lockWritePolicyRef.current;
    if (!currentPolicy.disabled) return true;
    message.warning(currentPolicy.reason || '订单当前不可拆票');
    return false;
  };

  const [submitting, setSubmitting] = useState(false);
  const [refreshingOrderId, setRefreshingOrderId] = useState<string>();
  const resumeCheckRef = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resumeCheckRef.current());

  // 拆票上下文：React Query 管理，orderId 变化自然重查；
  // 无拆票能力时不发起请求（与旧 loadContext 门禁一致）。
  const splitContextQuery = useQuery({
    queryKey: [SPLIT_QUERY_PREFIX, 'context', { orderId }],
    enabled: Boolean(orderId) && canSplit,
    // 独立箱货写入不会推进订单版本，检查上下文自身才能发现分配基线变化。
    refetchInterval: foreground && !submitting ? 15_000 : false,
    refetchIntervalInBackground: false,
    queryFn: async () => {
      if (!orderId) {
        // enabled 已保证；此处仅为类型收窄兜底。
        throw new Error('缺少订单标识');
      }
      try {
        const resp = await seaOrderChangeServiceGetSeaOrderSplitContext({
          orderId,
        });
        if (!resp?.data) throw new Error('接口未返回拆票上下文，请重试');
        return resp.data;
      } catch (error) {
        // 非 Error 抛出统一包装，保证全局 onError 文案与旧 catch 一致。
        throw error instanceof Error
          ? error
          : new Error(getErrorMessage(error, '加载拆票上下文失败'));
      }
    },
  });
  resumeCheckRef.current = () => {
    if (canSplit && orderId && !submitting)
      void splitContextQuery.refetch({ cancelRefetch: false });
  };
  // 已采纳上下文是本次分配的编辑基线；后台重查只更新查询缓存，不能替换该基线。
  const [splitBaseline, setSplitBaseline] = useState<{
    orderId: string;
    data: API.SeaOrderSplitContextData;
  } | null>(null);
  const splitContext =
    splitBaseline?.orderId === orderId ? splitBaseline.data : null;
  useEffect(() => {
    if (
      !splitContext &&
      splitContextQuery.isSuccess &&
      !splitContextQuery.isFetching &&
      splitContextQuery.data
    ) {
      setSplitBaseline({ orderId, data: splitContextQuery.data });
    }
  }, [
    orderId,
    splitContext,
    splitContextQuery.data,
    splitContextQuery.isSuccess,
    splitContextQuery.isFetching,
  ]);
  const loadingContext =
    (splitContextQuery.isFetching && !splitContext) ||
    refreshingOrderId === orderId;

  // 拆票结果集：至少 1 个 ORIGINAL + 1 个 CREATED
  const { markDirty: markUnsaved, resetDirty: resetUnsaved } =
    useUnsavedChangesGuard({
      identity: orderId,
    });
  const splitEditedRef = useRef(false);
  const markDirty = useCallback(() => {
    splitEditedRef.current = true;
    markUnsaved();
  }, [markUnsaved]);
  const resetDirty = useCallback(() => {
    splitEditedRef.current = false;
    resetUnsaved();
  }, [resetUnsaved]);
  const [changedCargoOrderId, setChangedCargoOrderId] = useState<string>();
  const cargoChanged = changedCargoOrderId === orderId;
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
  const confirmationActiveRef = useRef(false);
  confirmationActiveRef.current = results.some(
    (result) => result.targetType !== 'CURRENT',
  );

  // 防抖后的预览输入：进 queryKey 驱动预览查询（沿用「防抖关键词进 queryKey」模式）
  const [debouncedPreviewInput, setDebouncedPreviewInput] =
    useState<SplitPreviewInput | null>(null);
  const baselineNeedsRefresh =
    cargoChanged ||
    Boolean(
      splitContext &&
        splitContextQuery.data &&
        splitContextQuery.data !== splitContext,
    );

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

  // 只在首次采纳或用户确认刷新后重建分配，不跟随查询缓存自动覆盖输入。
  useEffect(() => {
    if (!splitContext) return;
    const ctx = splitContext;
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
    setNote('');
    if (confirmationActiveRef.current) confirmationForm.resetFields();
    setChangedCargoOrderId(undefined);
    setDebouncedPreviewInput(null);
    initialPreviewTriggeredRef.current = false;
    resetDirty();
  }, [splitBaseline, splitContext, confirmationForm, resetDirty]);

  const refreshSplitBaseline = async () => {
    const requestedOrderId = orderId;
    setRefreshingOrderId(requestedOrderId);
    try {
      const [contextResult] = await Promise.all([
        splitContextQuery.refetch(),
        refreshLockState(),
      ]);
      if (activeOrderIdRef.current !== requestedOrderId) return;
      if (contextResult.isSuccess && contextResult.data) {
        setSplitBaseline({
          orderId: requestedOrderId,
          data: contextResult.data,
        });
        // 相同内容的响应会被 React Query 复用，仍需解除货物写入后的过期标记。
        setChangedCargoOrderId(undefined);
      }
    } finally {
      setRefreshingOrderId((current) =>
        current === requestedOrderId ? undefined : current,
      );
    }
  };

  const handleRefresh = () => {
    if (!splitEditedRef.current) {
      void refreshSplitBaseline();
      return;
    }
    modal.confirm({
      title: '刷新拆票数据？',
      content:
        '刷新后将重新分配货物、箱、费用和附件，并清除当前未保存的拆票编辑。',
      okText: '确认刷新',
      cancelText: '保留编辑',
      onOk: refreshSplitBaseline,
    });
  };

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

  // 预览请求输入：由当前分配状态派生；null 表示上下文未就绪或结果不足
  const previewInput = useMemo<SplitPreviewInput | null>(() => {
    if (!splitContext || results.length < 2 || baselineNeedsRefresh)
      return null;
    const expectedVersions = buildExpectedVersions(results);
    if (!expectedVersions) return { kind: 'missing-versions' };
    return {
      kind: 'ready',
      request: {
        orderId,
        note: note ? note.trim() : undefined,
        targets: buildTargets(results),
        results: buildSplitResults(
          results,
          containerAssignments,
          cargoAllocations,
          sharedAllocations,
          feeAssignments,
          attAssignments,
        ),
        expectedVersions,
      },
    };
  }, [
    splitContext,
    results,
    orderId,
    note,
    containerAssignments,
    cargoAllocations,
    sharedAllocations,
    feeAssignments,
    attAssignments,
    baselineNeedsRefresh,
  ]);

  // 供结果分节在「匹配候选母单」后立即发起预览：直接写入防抖值绕过等待窗口
  const triggerPreview = (currentResults: ResultConfig[] = results) => {
    if (
      lockWritePolicyRef.current.disabled ||
      baselineNeedsRefresh ||
      !orderId ||
      !splitContext ||
      currentResults.length < 2
    ) {
      return;
    }
    const expectedVersions = buildExpectedVersions(currentResults);
    if (!expectedVersions) {
      setDebouncedPreviewInput({ kind: 'missing-versions' });
      return;
    }
    setDebouncedPreviewInput({
      kind: 'ready',
      request: {
        orderId,
        note: note ? note.trim() : undefined,
        targets: buildTargets(currentResults),
        results: buildSplitResults(
          currentResults,
          containerAssignments,
          cargoAllocations,
          sharedAllocations,
          feeAssignments,
          attAssignments,
        ),
        expectedVersions,
      },
    });
  };

  // 防抖收敛：预览输入变化后 300ms 才更新 queryKey；上下文就绪后的首次立即触发。
  // 预览竞态由 queryKey 隔离保证，不再手写竞态令牌。
  useEffect(() => {
    if (!previewInput) {
      setDebouncedPreviewInput(null);
      return undefined;
    }
    if (!initialPreviewTriggeredRef.current) {
      initialPreviewTriggeredRef.current = true;
      setDebouncedPreviewInput(previewInput);
      return undefined;
    }
    const timer = setTimeout(() => {
      setDebouncedPreviewInput(previewInput);
    }, 300);
    return () => clearTimeout(timer);
  }, [previewInput]);

  // 预览查询：换参请求期间保留上一次结果（keepPreviousData），失败即清空；
  // 锁定/无版本信息时经 enabled 门禁不发请求。
  const previewQuery = useQuery({
    queryKey: [
      SPLIT_QUERY_PREFIX,
      'preview',
      { orderId, input: debouncedPreviewInput },
    ],
    enabled:
      Boolean(orderId) &&
      !lockWritePolicy.disabled &&
      !baselineNeedsRefresh &&
      debouncedPreviewInput?.kind === 'ready',
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const input = debouncedPreviewInput;
      if (input?.kind !== 'ready') {
        // enabled 已保证；此处仅为类型收窄兜底。
        throw new Error('缺少拆票预览参数');
      }
      const resp = await seaOrderChangeServicePreviewSeaOrderSplit(
        { orderId },
        input.request,
      );
      return resp?.data ?? null;
    },
    meta: { silent: true },
  });

  // 旧 previewData / previewError / previewing 三个 state 的派生等价映射
  const previewData =
    !lockWritePolicy.disabled &&
    !baselineNeedsRefresh &&
    debouncedPreviewInput?.kind === 'ready' &&
    !previewQuery.error
      ? (previewQuery.data ?? null)
      : null;
  const previewError =
    debouncedPreviewInput?.kind === 'missing-versions'
      ? '缺少完整版本控制信息或候选版本未获取，无法进行拆票校验'
      : previewQuery.error
        ? getErrorMessage(previewQuery.error, '拆票校验未通过')
        : null;
  const previewing = previewQuery.isFetching;
  const previewMatchesInput =
    previewInput?.kind === 'ready' &&
    debouncedPreviewInput?.kind === 'ready' &&
    computeCanonicalSha256(previewInput.request) ===
      computeCanonicalSha256(debouncedPreviewInput.request);
  const canExecuteSplit =
    !lockWritePolicy.disabled &&
    !loadingContext &&
    !splitContextQuery.error &&
    !baselineNeedsRefresh &&
    Boolean(splitContext?.cargoItems?.length) &&
    previewMatchesInput &&
    !previewQuery.isFetching &&
    !previewQuery.isPlaceholderData &&
    Boolean(previewData?.isValid && previewData.conservationPassed);

  // 添加新票
  const handleAddResult = () => {
    if (!ensureSplitEditable()) return;
    markDirty();
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
    markDirty();
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
  const handleExecuteSplit = async (): Promise<false | undefined> => {
    if (!ensureSplitEditable()) return false;
    if (!canExecuteSplit) {
      message.error(
        previewError || '拆票数据未满足守恒校验或存在门禁错误，请检查！',
      );
      return false;
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
        return false;
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
        return false;
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
      if (
        !isWriteResponse(resp) ||
        !resp.data?.splitEventId ||
        resp.data.originalOrder?.orderId !== orderId
      )
        return false;

      const createdCount = resp?.data?.createdOrders?.length || 0;
      resetDirty();
      message.success(`拆票成功！原票已更新，成功生成 ${createdCount} 张新票`);
      history.push(`/orders/sea-export/${orderId}`);
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '执行拆票失败'));
      return false;
    } finally {
      setSubmitting(false);
    }

    return undefined;
  };

  if (!canSplit) {
    return <Result status="403" title="请进入获授权的分公司工作台办理拆票" />;
  }

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
          subTitle="支持整单部分拆票、HBL/箱货零误差守恒切分、未建账费用整行归属及多票并行派生"
          tags={<Tag color="blue">海运出口 (HOUSE)</Tag>}
          extra={
            <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
              刷新数据
            </Button>
          }
        />

        <div style={{ paddingBottom: 80 }}>
          {splitContextQuery.error && (
            <Alert
              type="error"
              showIcon
              title="拆票数据加载失败"
              description={getErrorMessage(
                splitContextQuery.error,
                '请重新加载拆票数据',
              )}
              style={{ marginBottom: 16 }}
            />
          )}
          {baselineNeedsRefresh && (
            <Alert
              type="warning"
              showIcon
              title="拆票数据需要刷新"
              description={
                cargoChanged
                  ? '实际货物明细已更新，请刷新拆票数据后重新确认分配。当前编辑暂时保留，刷新前不能执行拆票。'
                  : '这个订单已经发生变动，请重新刷新。当前拆票编辑暂时保留，刷新后请重新确认分配。'
              }
              action={<Button onClick={handleRefresh}>刷新拆票数据</Button>}
              style={{ marginBottom: 16 }}
            />
          )}
          {lockWritePolicy.disabled && (
            <Alert
              type={
                lockWritePolicy.reason?.includes('已锁定') ? 'warning' : 'error'
              }
              showIcon
              title="拆票操作当前不可用"
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
            <Space
              orientation="vertical"
              size="middle"
              style={{ width: '100%' }}
            >
              {previewError && (
                <Alert
                  type="error"
                  showIcon
                  title="拆票校验未通过"
                  description={previewError}
                />
              )}

              <SplitBaselineSection
                previewData={previewData}
                splitContext={splitContext}
              />

              {splitContext && (
                <Alert
                  type={splitContext.cargoItems?.length ? 'info' : 'warning'}
                  showIcon
                  title={
                    splitContext.cargoItems?.length
                      ? '拆票按实际货物明细分配'
                      : '尚未登记可分配的实际货物明细'
                  }
                  description={
                    <>
                      拆票以实际货物明细的件数、毛重和体积为守恒依据；委托件重尺和实际箱记录不会自动生成货物明细。请先登记实际货物，再刷新拆票数据并分配到原票和新票。
                      {(!canReadCargo || !canCreateCargo) && (
                        <div>
                          当前没有实际货物录入权限，请联系有权限的操作人员补录。
                        </div>
                      )}
                    </>
                  }
                  action={
                    <Button
                      disabled={
                        !canReadCargo ||
                        !canCreateCargo ||
                        lockWritePolicy.disabled
                      }
                      onClick={() => {
                        if (
                          !canReadCargo ||
                          !canCreateCargo ||
                          !ensureSplitEditable()
                        )
                          return;
                        cargoItemDrawerRef.current?.open({
                          id: orderId,
                          orderNo: splitContext.orderNo,
                          businessType: OrderBusinessType.BUSINESS_TYPE_SE,
                        });
                      }}
                    >
                      录入实际货物
                    </Button>
                  }
                />
              )}

              <SplitResultsSection
                results={results}
                setResults={(value) => {
                  markDirty();
                  setResults(value);
                }}
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
                setContainerAssignments={(value) => {
                  markDirty();
                  setContainerAssignments(value);
                }}
                cargoAllocations={cargoAllocations}
                setCargoAllocations={(value) => {
                  markDirty();
                  setCargoAllocations(value);
                }}
                sharedAllocations={sharedAllocations}
                setSharedAllocations={(value) => {
                  markDirty();
                  setSharedAllocations(value);
                }}
              />

              <SplitFeesSection
                splitContext={splitContext}
                results={results}
                feeAssignments={feeAssignments}
                setFeeAssignments={(value) => {
                  markDirty();
                  setFeeAssignments(value);
                }}
                feeCurrencySummaries={feeCurrencySummaries}
              />

              <SplitAttachmentsAndNotesSection
                splitContext={splitContext}
                results={results}
                attAssignments={attAssignments}
                setAttAssignments={(value) => {
                  markDirty();
                  setAttAssignments(value);
                }}
                note={note}
                setNote={(value) => {
                  markDirty();
                  setNote(value);
                }}
                orderId={orderId}
                confirmationForm={confirmationForm}
                onUserEdit={markDirty}
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
              {canExecuteSplit ? (
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
          <AsyncPopconfirm
            title="确认提交拆票"
            description="拆票将原子锁定订单、分批移转货物、分单与费用并创建新订单。确定提交执行？"
            okText="确认执行"
            cancelText="取消"
            onConfirm={handleExecuteSplit}
            disabled={!canExecuteSplit}
          >
            <Button
              type="primary"
              loading={submitting}
              disabled={!canExecuteSplit}
            >
              确认执行拆票
            </Button>
          </AsyncPopconfirm>
        </StickyFooterBar>
      </Spin>
      <CargoItemDrawer
        key={orderId}
        ref={cargoItemDrawerRef}
        canCreate={!lockWritePolicy.disabled && canCreateCargo}
        canUpdate={
          !lockWritePolicy.disabled &&
          access.canOrder(
            OrderBusinessType.BUSINESS_TYPE_SE,
            'cargo_item.update',
          )
        }
        canRemove={
          !lockWritePolicy.disabled &&
          access.canOrder(
            OrderBusinessType.BUSINESS_TYPE_SE,
            'cargo_item.delete',
          )
        }
        onChanged={(order) => {
          if (order.id === activeOrderIdRef.current)
            setChangedCargoOrderId(order.id);
        }}
      />
    </PageContainer>
  );
}
