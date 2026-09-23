import { useQuery, useQueryClient } from '@tanstack/react-query';
import { App, Drawer, Form, Steps } from 'antd';
import dayjs from 'dayjs';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { DRAWER_SIZE } from '@/components/ui';
import {
  BillGroupingMode,
  FinanceOrganizationPurpose,
} from '@/enums.generated';
import {
  settlementServiceConfirmBillBatch,
  settlementServiceCreateBillBatch,
  settlementServiceListFinanceOrganizationOptions,
  settlementServicePreviewBillBatch,
} from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { longRequestOptions } from '@/utils/requestTimeout';
import { generateUUID } from '@/utils/uuid';
import BillCandidateSelectionStep from './BillCandidateSelectionStep';
import BillCreationResultTable from './BillCreationResultTable';
import BillGroupConfigurationStep from './BillGroupConfigurationStep';
import BillSplitStrategyCards from './BillSplitStrategyCards';
import BillWorkbenchFooter from './BillWorkbenchFooter';
import {
  type BillCreationMode,
  directionText,
  type GroupFormValue,
  isGroupComplete,
  makeGroupConfig,
  type RequestError,
  requestMessage,
  requestReason,
  type WorkbenchFormValue,
  type WorkbenchValidationError,
} from './billWorkbenchHelpers';

export type BillCreationWorkbenchProps = {
  open: boolean;
  initialFeeIds?: string[];
  initialOrganizationId?: string;
  initialOrganizationName?: string;
  sourceLabel?: string;
  mode?: BillCreationMode;
  onClose: () => void;
  onCreated?: (batch: API.FinanceBillBatch) => void;
};

/** 预览快照 queryKey 域前缀；会话序号与防抖后请求参数逐层进 key。 */
const PREVIEW_QUERY_KEY = ['finance-bills', 'bill-preview-batch'] as const;

/**
 * 拆单预览 queryFn：空叶子视为异常；错误文案与原实现 requestMessage 一致，
 * 经全局 cache onError 透出（无 meta 静默）。
 */
async function fetchPreviewBatch(
  request: API.PreviewBillBatchRequest,
): Promise<API.PreviewBillBatchResponse> {
  try {
    const response = await settlementServicePreviewBillBatch(request, {
      ...longRequestOptions,
      skipErrorHandler: true,
    });
    if (unwrapList(response).length === 0) {
      throw new Error('服务端未返回拆单预览');
    }
    return response;
  } catch (rawError: unknown) {
    throw new Error(requestMessage(rawError as RequestError, '拆单预览失败'));
  }
}

export default function BillCreationWorkbench({
  open,
  initialFeeIds = [],
  initialOrganizationId,
  initialOrganizationName,
  sourceLabel,
  mode = 'NORMAL',
  onClose,
  onCreated,
}: BillCreationWorkbenchProps) {
  // 对冲模式在服务端仍按方向分别生成原始账单叶子，并额外原子生成草稿对冲结算单。
  const groupingMode =
    mode === 'NETTING'
      ? BillGroupingMode.BILL_GROUPING_MODE_NETTING
      : BillGroupingMode.BILL_GROUPING_MODE_NORMAL;
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [form] = Form.useForm<WorkbenchFormValue>();
  const [current, setCurrent] = useState(0);
  const [selectedFeeIds, setSelectedFeeIds] = useState<React.Key[]>([]);
  const [splitByOrder, setSplitByOrder] = useState(true);
  const [splitByTaxRate, setSplitByTaxRate] = useState(false);
  const [organizationId, setOrganizationId] = useState<string>();
  const [result, setResult] = useState<API.FinanceBillBatch>();
  const [submitting, setSubmitting] = useState(false);
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [activeGroupKey, setActiveGroupKey] = useState<string>();
  const [sessionIdentity, setSessionIdentity] = useState('');
  // 防抖后进入 queryKey 的预览请求参数；undefined 表示当前没有待执行/进行中的预览。
  const [previewRequest, setPreviewRequest] =
    useState<API.PreviewBillBatchRequest>();
  // 快照令牌失效时间戳：晚于该时间成功落地的预览才允许携带 previewToken
  // 参与建单，等价原 fingerprint/previewTokenFingerprint 双 ref 的失效语义。
  const [snapshotInvalidatedAt, setSnapshotInvalidatedAt] = useState(0);

  const previewInitKeyRef = useRef<string | undefined>(undefined);
  const previewTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );
  const previewDataRef = useRef<API.PreviewBillBatchResponse | undefined>(
    undefined,
  );
  // 会话序号：同时驱动账户候选隔离身份（sessionIdentity）与预览 queryKey 分层。
  const sessionSequenceRef = useRef(0);
  const selectedIdsRef = useRef<string[]>([]);
  const organizationIdRef = useRef<string | undefined>(undefined);
  const splitByOrderRef = useRef(splitByOrder);
  const splitByTaxRateRef = useRef(splitByTaxRate);

  const initialFeeKey = useMemo(
    () => (initialFeeIds || []).filter(Boolean).join('|'),
    [initialFeeIds],
  );
  const fixedSelection = initialFeeKey.length > 0;

  // 可建账所属公司候选：抽屉打开即加载；错误文案由 queryFn 包装后经全局
  // cache onError 透出；失败时展示为空候选（等价原 catch 置空）。
  const organizationOptionsQuery = useQuery({
    queryKey: [
      'finance-bills',
      'organization-options',
      {
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_BILL_CREATE,
      },
    ],
    enabled: open,
    queryFn: async (): Promise<API.FinanceOrganizationOption[]> => {
      try {
        const response = await settlementServiceListFinanceOrganizationOptions({
          purpose:
            FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_BILL_CREATE,
        });
        return response.data ?? [];
      } catch (error: unknown) {
        throw error instanceof Error
          ? error
          : new Error(getErrorMessage(error, '加载可建账所属公司失败'));
      }
    },
  });
  const organizationOptions = organizationOptionsQuery.isError
    ? []
    : (organizationOptionsQuery.data ?? []);

  const selectedIds = useMemo(
    () => selectedFeeIds.map(String).filter(Boolean),
    [selectedFeeIds],
  );

  // 拆单预览：350ms 防抖后的参数进 queryKey。D1→D2、币种/汇率、固定来源
  // 切换等竞态一律由库按 queryKey 隔离（迟到响应只写自己的缓存键），
  // 不再使用任何请求令牌比对。会话序号进 key 并在快照清空时递增，
  // 用于切断换参期间 keepPreviousData 的跨会话旧数据桥接。
  const previewQuery = useQuery({
    queryKey: [
      ...PREVIEW_QUERY_KEY,
      sessionSequenceRef.current,
      previewRequest,
    ],
    enabled: previewRequest !== undefined,
    // 取数统一由 runPreview 的 fetchQuery（自带 staleTime=0，每次触发必发）
    // 与换参（key 变化）驱动；观察者侧关闭 mount/换参时的重查判定，
    // 避免 fetchQuery 快速落地后观察者才挂载/换参时再补一次同 key 请求，
    // 保证每次触发只发一次请求（等价原实现）。
    refetchOnMount: false,
    staleTime: Infinity,
    queryFn: async (): Promise<API.PreviewBillBatchResponse> => {
      if (!previewRequest) {
        // enabled 已保证参数存在；此处仅为类型收窄兜底。
        throw new Error('缺少拆单预览参数');
      }
      return fetchPreviewBatch(previewRequest);
    },
    // 换参重查期间保留上一份快照，避免第 3 步面板闪空；跨会话不桥接。
    placeholderData: (previousData, previousQuery) =>
      previousQuery &&
      (previousQuery.queryKey[2] as number | undefined) ===
        sessionSequenceRef.current
        ? previousData
        : undefined,
  });
  const preview = previewQuery.data;
  const loading = previewQuery.isFetching || submitting;

  useEffect(() => {
    selectedIdsRef.current = selectedIds;
    organizationIdRef.current = organizationId;
    splitByOrderRef.current = splitByOrder;
    splitByTaxRateRef.current = splitByTaxRate;
    previewDataRef.current = previewQuery.data;
  }, [
    organizationId,
    previewQuery.data,
    selectedIds,
    splitByOrder,
    splitByTaxRate,
  ]);

  // 仅当展示中的数据是「当前参数、当前会话、失效标记之后」的成功响应时，
  // 其 previewToken 才可用于创建；配置变更即刻失效，等待新预览落地。
  const snapshotTokenUsable =
    previewQuery.isSuccess &&
    !previewQuery.isPlaceholderData &&
    previewQuery.dataUpdatedAt > snapshotInvalidatedAt;

  const runPreview = useCallback(
    async (
      overrideIds?: string[],
      policyOverride?: API.BillGroupingPolicy,
      organizationIdOverride?: string,
    ): Promise<boolean> => {
      if (previewTimerRef.current) {
        clearTimeout(previewTimerRef.current);
        previewTimerRef.current = undefined;
      }
      const ids = overrideIds ?? selectedIdsRef.current;
      const requestedOrganizationId =
        organizationIdOverride ?? organizationIdRef.current;
      if (!requestedOrganizationId) {
        message.warning('请先选择可建账所属公司');
        return false;
      }
      if (ids.length === 0) {
        message.warning('请至少选择一笔未建账且未建立账单的费用');
        return false;
      }
      const policy = policyOverride ?? {
        mode: groupingMode,
        splitByOrder: splitByOrderRef.current,
        splitByTaxRate: splitByTaxRateRef.current,
      };
      const values = form.getFieldsValue(true) as WorkbenchFormValue;
      const currentGroups = previewDataRef.current?.data || [];
      const groupConfigs: API.BillBatchPreviewGroupConfigInput[] = [];
      for (const group of currentGroups) {
        if (group.groupKey) {
          groupConfigs.push(
            makeGroupConfig(group.groupKey, values.groups?.[group.groupKey]),
          );
        }
      }
      groupConfigs.sort((left, right) =>
        left.groupKey.localeCompare(right.groupKey),
      );
      const request = {
        feeIds: [...ids].sort(),
        groupingPolicy: policy,
        organizationId: requestedOrganizationId,
        groupConfigs,
      } satisfies API.PreviewBillBatchRequest;
      setPreviewRequest(request);
      try {
        // 与 useQuery 同 key：与观察者共享同一次网络往返；同参重复触发时
        // staleTime=0 保证仍会真实重查（等价原实现每次必发请求）。
        await queryClient.fetchQuery({
          queryKey: [...PREVIEW_QUERY_KEY, sessionSequenceRef.current, request],
          queryFn: () => fetchPreviewBatch(request),
        });
        return true;
      } catch {
        return false;
      }
    },
    [form, groupingMode, message, queryClient],
  );

  const runPreviewRef = useRef(runPreview);
  useEffect(() => {
    runPreviewRef.current = runPreview;
  }, [runPreview]);

  // 清空预览快照：递增会话序号（切断旧 key 的 placeholder 桥接与账户候选
  // 身份）、丢弃未决防抖并复位快照令牌。用于组织切换、全部移除、重建会话
  // 等需要彻底回到无快照状态的场景。
  const clearPreviewSnapshot = useCallback(() => {
    if (previewTimerRef.current) {
      clearTimeout(previewTimerRef.current);
      previewTimerRef.current = undefined;
    }
    previewDataRef.current = undefined;
    setSnapshotInvalidatedAt(Date.now());
    setSessionIdentity(`open-${++sessionSequenceRef.current}`);
    setPreviewRequest(undefined);
  }, []);

  // 配置变更后立刻让当前快照令牌失效（创建将被要求重新预览），
  // 数据本身保留展示，等待防抖后的新预览落地。
  const invalidatePreview = useCallback(() => {
    if (previewTimerRef.current) {
      clearTimeout(previewTimerRef.current);
      previewTimerRef.current = undefined;
    }
    setSnapshotInvalidatedAt(Date.now());
  }, []);

  const schedulePreview = useCallback(() => {
    if (previewTimerRef.current) clearTimeout(previewTimerRef.current);
    previewTimerRef.current = setTimeout(() => {
      if (organizationIdRef.current && selectedIdsRef.current.length > 0) {
        void runPreview();
      }
    }, 350);
  }, [runPreview]);

  const handleConfigurationChange = useCallback(() => {
    invalidatePreview();
    schedulePreview();
  }, [invalidatePreview, schedulePreview]);

  // 初始化或当从业务页面进入时，自动快速预览并直达账单资料页
  useEffect(() => {
    if (!open) {
      if (previewTimerRef.current) {
        clearTimeout(previewTimerRef.current);
        previewTimerRef.current = undefined;
      }
      previewInitKeyRef.current = undefined;
      return;
    }
    const initialIds = initialFeeKey
      ? initialFeeKey.split('|').filter(Boolean)
      : [];
    const initKey = `${initialFeeKey}:${initialOrganizationId || ''}:${groupingMode}`;
    if (previewInitKeyRef.current === initKey) return;
    previewInitKeyRef.current = initKey;
    clearPreviewSnapshot();
    setSelectedFeeIds(initialIds);
    const nextOrganizationId =
      initialIds.length > 0 ? initialOrganizationId : undefined;
    organizationIdRef.current = nextOrganizationId;
    setOrganizationId(nextOrganizationId);
    setSplitByOrder(true);
    setSplitByTaxRate(false);
    setResult(undefined);
    setSubmitting(false);
    setConfirming(false);
    setIdempotencyKey(generateUUID());
    form.resetFields();

    if (initialIds.length > 0 && initialOrganizationId) {
      // 极速模式：从单票/多选费用带入时，直接拉取预览并切到账单资料页
      void runPreviewRef
        .current(
          initialIds,
          {
            mode: groupingMode,
            splitByOrder: true,
            splitByTaxRate: false,
          },
          initialOrganizationId,
        )
        .then((ok) => {
          setCurrent(ok ? 2 : 0);
        });
    } else {
      setCurrent(0);
    }
  }, [
    open,
    initialFeeKey,
    initialOrganizationId,
    groupingMode,
    clearPreviewSnapshot,
    form,
  ]);

  const handleOrganizationChange = (value: string | undefined) => {
    organizationIdRef.current = value;
    setOrganizationId(value);
    setSelectedFeeIds([]);
    clearPreviewSnapshot();
    setResult(undefined);
    setCurrent(0);
  };

  // 从预览明细中即时剔除误选行
  const handleRemoveFee = async (feeId?: string) => {
    if (!feeId) return;
    const nextIds = selectedIds.filter((id) => id !== feeId);
    if (nextIds.length === 0) {
      message.info('已移除所有费用，请重新选择');
      setSelectedFeeIds([]);
      clearPreviewSnapshot();
      setCurrent(0);
      return;
    }
    setSelectedFeeIds(nextIds);
    invalidatePreview();
    message.success('已从本次建单中移除该费用');
    await runPreview(nextIds);
  };

  const next = async () => {
    if (current === 0) {
      if (!organizationId) {
        message.warning('请先选择可建账所属公司');
        return;
      }
      if (selectedIds.length === 0) {
        message.warning('请至少选择一笔未建账且未建立账单的费用');
        return;
      }
      setCurrent(1);
      return;
    }
    if (current === 1) {
      if (
        await runPreview(undefined, {
          mode: groupingMode,
          splitByOrder,
          splitByTaxRate,
        })
      ) {
        setCurrent(2);
      }
    }
  };

  const createBatch = async () => {
    if (!organizationId) {
      message.warning('请先选择可建账所属公司');
      return;
    }
    const currentValues = form.getFieldsValue(true) as WorkbenchFormValue;
    const firstInvalidGroup = preview?.data?.find(
      (group) =>
        !group.groupKey ||
        group.configurationComplete !== true ||
        !isGroupComplete(
          currentValues.groups?.[group.groupKey],
          group.currency,
        ),
    );
    if (firstInvalidGroup?.groupKey) {
      setActiveGroupKey(firstInvalidGroup.groupKey);
      message.warning('请先补齐首个标记叶子的日期和结算账户');
      return;
    }
    if (
      !preview?.previewToken ||
      !preview.data?.length ||
      !snapshotTokenUsable ||
      preview.data.some((group) => group.configurationComplete !== true)
    ) {
      message.warning('账单预览快照尚未完整或已失效，请补齐配置后重新预览');
      return;
    }
    let values: WorkbenchFormValue;
    try {
      values = await form.validateFields();
    } catch (errorInfo: unknown) {
      const { errorFields } = (errorInfo ?? {}) as WorkbenchValidationError;
      const firstErrorField = errorFields?.[0]?.name;
      if (
        Array.isArray(firstErrorField) &&
        firstErrorField[0] === 'groups' &&
        firstErrorField[1]
      ) {
        setActiveGroupKey(String(firstErrorField[1]));
      }
      message.warning('请为每张拟生成账单补齐必填资料和结算账户');
      return;
    }
    setSubmitting(true);
    try {
      const allFormValues = form.getFieldsValue(true) as WorkbenchFormValue;
      const response = await settlementServiceCreateBillBatch(
        {
          feeIds: selectedIds,
          groupingPolicy: {
            mode: groupingMode,
            splitByOrder,
            splitByTaxRate,
          },
          previewToken: preview.previewToken,
          idempotencyKey,
          organizationId,
          groups: preview.data.map((group) => {
            const groupKey = group.groupKey || '';
            const value =
              values?.groups?.[groupKey] || allFormValues.groups?.[groupKey];
            if (!value) {
              throw new Error(
                `账单组 ${group.settlementPartyName || groupKey} 缺少配置数据`,
              );
            }
            return {
              groupKey,
              statementTitle: value.statementTitle.trim(),
              billDate: value.billDate.format('YYYY-MM-DD'),
              paymentTermsDays: value.paymentTermsDays,
              note: value.note?.trim() || undefined,
              settlementAccountId: value.settlementAccountId || '',
              estimatedInvoiceCurrency:
                value.estimatedInvoiceCurrency || undefined,
              estimatedInvoiceRate:
                value.estimatedInvoiceRate?.trim() || undefined,
            };
          }),
        },
        { ...longRequestOptions, skipErrorHandler: true },
      );
      if (!response.data) throw new Error('服务端未返回建单结果');
      setResult(response.data);
      setCurrent(3);
      message.success(`批次 ${response.data.batchNo || ''} 已原子生成`);
      onCreated?.(response.data);
    } catch (rawError: unknown) {
      const error = rawError as RequestError;
      if (requestReason(error) === 'FINANCE_BILL_PREVIEW_STALE') {
        message.warning('费用已发生变化，请重新预览后再生成账单');
        clearPreviewSnapshot();
        setCurrent(1);
      } else if (requestReason(error) === 'FINANCE_BILL_FEE_INVALID') {
        message.error('所选费用必须为未建账状态且尚未进入其他账单');
      } else {
        message.error(requestMessage(error, '批量生成账单失败'));
      }
    } finally {
      setSubmitting(false);
    }
  };

  const confirmBatch = async () => {
    if (!result?.id || !result.bills?.length) return;
    setConfirming(true);
    try {
      const response = await settlementServiceConfirmBillBatch(
        { id: result.id },
        {
          id: result.id,
          bills: result.bills.map((bill) => ({
            billId: bill.id || '',
            expectedVersion: bill.version || '0',
          })),
        },
        longRequestOptions,
      );
      if (response.data) setResult(response.data);
      message.success('本批账单已全部确认，可以进入开票、收付款和核销流程');
      onCreated?.(response.data || result);
    } catch (error) {
      message.error(getErrorMessage(error, '批量确认账单失败'));
    } finally {
      setConfirming(false);
    }
  };

  // 预览快照落地后的草稿归位：新叶子补默认值、已编辑叶子按 groupKey 原样
  // 保留，已移除叶子的表单值随整体重写丢弃。仅对当前观察 key 的真实响应
  // 执行；迟到的旧参数响应只写自己的缓存键，不会触发本 effect。
  useEffect(() => {
    if (
      !previewRequest ||
      previewQuery.isPlaceholderData ||
      !previewQuery.isSuccess
    ) {
      return;
    }
    const response = previewQuery.data;
    if (!response) return;
    const groups = unwrapList(response);
    const previousGroups = form.getFieldValue('groups') || {};
    const nextGroups: Record<string, GroupFormValue> = {};
    for (const group of groups) {
      if (!group.groupKey) continue;
      const existing = previousGroups[group.groupKey] as
        | GroupFormValue
        | undefined;
      nextGroups[group.groupKey] = existing || {
        statementTitle: group.settlementPartyName || '',
        billDate: dayjs(),
        paymentTermsDays:
          group.defaultPaymentTermsDays !== undefined &&
          group.defaultPaymentTermsDays !== null
            ? group.defaultPaymentTermsDays
            : undefined,
        note: undefined,
        settlementAccountId: undefined,
        estimatedInvoiceCurrency:
          group.estimatedInvoiceCurrency || group.currency || undefined,
        estimatedInvoiceRate: group.estimatedInvoiceRate || undefined,
      };
      if (existing) {
        nextGroups[group.groupKey] = {
          ...existing,
          paymentTermsDays:
            existing.paymentTermsDays !== undefined
              ? existing.paymentTermsDays
              : group.defaultPaymentTermsDays !== undefined &&
                  group.defaultPaymentTermsDays !== null
                ? group.defaultPaymentTermsDays
                : undefined,
          estimatedInvoiceCurrency:
            existing.estimatedInvoiceCurrency !== undefined
              ? existing.estimatedInvoiceCurrency
              : group.estimatedInvoiceCurrency || group.currency || undefined,
          estimatedInvoiceRate:
            existing.estimatedInvoiceRate !== undefined
              ? existing.estimatedInvoiceRate
              : group.estimatedInvoiceRate || undefined,
        };
      }
    }
    form.setFieldValue('groups', nextGroups);
    setActiveGroupKey((previous) =>
      previous && nextGroups[previous] ? previous : Object.keys(nextGroups)[0],
    );
  }, [
    form,
    previewQuery.data,
    previewQuery.isPlaceholderData,
    previewQuery.isSuccess,
    previewRequest,
  ]);

  const formGroups = Form.useWatch('groups', form) as
    | Record<string, GroupFormValue>
    | undefined;
  const invalidGroupKeys = useMemo(
    () =>
      new Set(
        (preview?.data || [])
          .filter(
            (group) =>
              !group.groupKey ||
              group.configurationComplete !== true ||
              !isGroupComplete(formGroups?.[group.groupKey], group.currency),
          )
          .map((group) => group.groupKey || ''),
      ),
    [formGroups, preview?.data],
  );

  const footer = (
    <BillWorkbenchFooter
      current={current}
      loading={loading}
      mode={mode}
      groupCount={preview?.data?.length || 0}
      nettingPairCount={preview?.nettingPairs?.length || 0}
      onClose={onClose}
      onBack={() => setCurrent((value) => value - 1)}
      onNext={() => void next()}
      onCreate={() => void createBatch()}
    />
  );

  return (
    <Drawer
      title={mode === 'NETTING' ? '费用批量对冲建账' : '费用批量转账单'}
      open={open}
      size={DRAWER_SIZE.XL}
      destroyOnHidden
      mask={{ closable: false }}
      footer={footer}
      onClose={onClose}
    >
      <Steps
        current={current}
        size="small"
        style={{ marginBottom: 24 }}
        items={[
          { title: '选择费用' },
          { title: '拆单策略' },
          { title: '账单资料' },
          { title: '生成完成' },
        ]}
      />

      {current === 0 && (
        <BillCandidateSelectionStep
          fixedSelection={fixedSelection}
          sourceLabel={sourceLabel}
          selectedIds={selectedIds}
          selectedFeeIds={selectedFeeIds}
          onSelectedFeeIdsChange={setSelectedFeeIds}
          initialOrganizationId={initialOrganizationId}
          initialOrganizationName={initialOrganizationName}
          organizationId={organizationId}
          organizationOptions={organizationOptions}
          onOrganizationChange={handleOrganizationChange}
        />
      )}

      {current === 1 && (
        <BillSplitStrategyCards
          splitByOrder={splitByOrder}
          setSplitByOrder={(checked) => {
            setSplitByOrder(checked);
            invalidatePreview();
          }}
          splitByTaxRate={splitByTaxRate}
          setSplitByTaxRate={(checked) => {
            setSplitByTaxRate(checked);
            invalidatePreview();
          }}
          selectedCount={selectedIds.length}
        />
      )}

      {current === 2 && preview?.data && (
        <BillGroupConfigurationStep
          mode={mode}
          form={form}
          preview={preview}
          groupingMode={groupingMode}
          organizationId={organizationId}
          sessionIdentity={sessionIdentity}
          loading={loading}
          splitByOrder={splitByOrder}
          splitByTaxRate={splitByTaxRate}
          onSplitByOrderChange={setSplitByOrder}
          onSplitByTaxRateChange={setSplitByTaxRate}
          activeGroupKey={activeGroupKey}
          onActiveGroupKeyChange={setActiveGroupKey}
          invalidGroupKeys={invalidGroupKeys}
          invalidatePreview={invalidatePreview}
          loadPreview={runPreview}
          onRemoveFee={handleRemoveFee}
          onConfigurationChange={handleConfigurationChange}
        />
      )}

      {current === 3 && (
        <BillCreationResultTable
          result={result}
          confirming={confirming}
          onConfirmBatch={() => void confirmBatch()}
          directionText={directionText}
        />
      )}
    </Drawer>
  );
}
