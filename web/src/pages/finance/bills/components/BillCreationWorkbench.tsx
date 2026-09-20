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

export type { BillCreationMode } from './billWorkbenchHelpers';

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
  const { message } = App.useApp();
  const [form] = Form.useForm<WorkbenchFormValue>();
  const [current, setCurrent] = useState(0);
  const [selectedFeeIds, setSelectedFeeIds] = useState<React.Key[]>([]);
  const [splitByOrder, setSplitByOrder] = useState(true);
  const [splitByTaxRate, setSplitByTaxRate] = useState(false);
  const [preview, setPreview] = useState<API.PreviewBillBatchResponse>();
  const [organizationId, setOrganizationId] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [result, setResult] = useState<API.FinanceBillBatch>();
  const [loading, setLoading] = useState(false);
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [activeGroupKey, setActiveGroupKey] = useState<string>();
  const [sessionIdentity, setSessionIdentity] = useState('');
  const previewInitKeyRef = useRef<string | undefined>(undefined);
  const previewErrorKeyRef = useRef<string | undefined>(undefined);
  const previewRequestTokenRef = useRef(0);
  const previewFingerprintRef = useRef('');
  const previewTokenFingerprintRef = useRef('');
  const previewTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );
  const previewRef = useRef<API.PreviewBillBatchResponse | undefined>(
    undefined,
  );
  const sessionSequenceRef = useRef(0);
  const previewPendingRef = useRef<{
    key: string;
    promise: Promise<boolean>;
  } | null>(null);

  const initialFeeKey = useMemo(
    () => (initialFeeIds || []).filter(Boolean).join('|'),
    [initialFeeIds],
  );
  const fixedSelection = initialFeeKey.length > 0;

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_BILL_CREATE,
    })
      .then((response) => {
        if (!cancelled) setOrganizationOptions(response.data ?? []);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setOrganizationOptions([]);
          message.error(getErrorMessage(error, '加载可建账所属公司失败'));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [message, open]);

  const selectedIds = useMemo(
    () => selectedFeeIds.map(String).filter(Boolean),
    [selectedFeeIds],
  );

  const selectedIdsRef = useRef<string[]>([]);
  const organizationIdRef = useRef<string | undefined>(undefined);
  const splitByOrderRef = useRef(splitByOrder);
  const splitByTaxRateRef = useRef(splitByTaxRate);

  useEffect(() => {
    selectedIdsRef.current = selectedIds;
    organizationIdRef.current = organizationId;
    splitByOrderRef.current = splitByOrder;
    splitByTaxRateRef.current = splitByTaxRate;
    previewRef.current = preview;
  }, [organizationId, preview, selectedIds, splitByOrder, splitByTaxRate]);

  const loadPreview = useCallback(
    async (
      overrideIds?: string[],
      policyOverride?: API.BillGroupingPolicy,
      organizationIdOverride?: string,
    ) => {
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
        message.warning('请至少选择一笔已确认且未建立账单的费用');
        return false;
      }
      const policy = policyOverride ?? {
        mode: groupingMode,
        splitByOrder: splitByOrderRef.current,
        splitByTaxRate: splitByTaxRateRef.current,
      };
      const values = form.getFieldsValue(true) as WorkbenchFormValue;
      const currentGroups = previewRef.current?.data || [];
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
      const requestFingerprint = JSON.stringify({ sessionIdentity, request });
      previewFingerprintRef.current = requestFingerprint;
      const requestToken = ++previewRequestTokenRef.current;
      setLoading(true);
      try {
        const response = await settlementServicePreviewBillBatch(request, {
          ...longRequestOptions,
          skipErrorHandler: true,
        });
        if (
          requestToken !== previewRequestTokenRef.current ||
          requestFingerprint !== previewFingerprintRef.current ||
          requestedOrganizationId !== organizationIdRef.current
        ) {
          return false;
        }
        const groups = unwrapList(response);
        if (groups.length === 0) {
          throw new Error('服务端未返回拆单预览');
        }
        previewErrorKeyRef.current = undefined;
        setPreview(response);
        previewRef.current = response;
        previewTokenFingerprintRef.current = response.previewToken
          ? requestFingerprint
          : '';

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
                  : group.estimatedInvoiceCurrency ||
                    group.currency ||
                    undefined,
              estimatedInvoiceRate:
                existing.estimatedInvoiceRate !== undefined
                  ? existing.estimatedInvoiceRate
                  : group.estimatedInvoiceRate || undefined,
            };
          }
        }
        form.setFieldValue('groups', nextGroups);
        setActiveGroupKey((previous) =>
          previous && nextGroups[previous]
            ? previous
            : Object.keys(nextGroups)[0],
        );
        return true;
      } catch (rawError: unknown) {
        if (
          requestToken !== previewRequestTokenRef.current ||
          requestFingerprint !== previewFingerprintRef.current ||
          requestedOrganizationId !== organizationIdRef.current
        ) {
          return false;
        }
        const error = rawError as RequestError;
        const errorKey = `${requestedOrganizationId}:${ids.join('|')}:${requestReason(error) || requestMessage(error, '拆单预览失败')}`;
        if (previewErrorKeyRef.current !== errorKey) {
          previewErrorKeyRef.current = errorKey;
          message.error(requestMessage(error, '拆单预览失败'));
        }
        return false;
      } finally {
        if (requestToken === previewRequestTokenRef.current) {
          setLoading(false);
        }
      }
    },
    [form, groupingMode, message, sessionIdentity],
  );

  const loadPreviewRef = useRef(loadPreview);
  useEffect(() => {
    loadPreviewRef.current = loadPreview;
  }, [loadPreview]);

  // 初始化或当从业务页面进入时，自动快速预览并直达账单资料页
  useEffect(() => {
    if (!open) {
      previewRequestTokenRef.current += 1;
      previewFingerprintRef.current = '';
      previewTokenFingerprintRef.current = '';
      if (previewTimerRef.current) clearTimeout(previewTimerRef.current);
      previewInitKeyRef.current = undefined;
      previewErrorKeyRef.current = undefined;
      previewPendingRef.current = null;
      return;
    }
    const initialIds = initialFeeKey
      ? initialFeeKey.split('|').filter(Boolean)
      : [];
    const initKey = `${initialFeeKey}:${initialOrganizationId || ''}:${groupingMode}:${open ? 'open' : 'closed'}`;
    let cancelled = false;
    if (previewInitKeyRef.current === initKey) {
      const pending = previewPendingRef.current;
      if (pending?.key === initKey) {
        void pending.promise.then((ok) => {
          if (cancelled) return;
          setCurrent(ok ? 2 : 0);
        });
      }
      return () => {
        cancelled = true;
      };
    }
    previewInitKeyRef.current = initKey;
    previewRequestTokenRef.current += 1;
    previewFingerprintRef.current = '';
    previewTokenFingerprintRef.current = '';
    setSessionIdentity(`open-${++sessionSequenceRef.current}`);
    setSelectedFeeIds(initialIds);
    const nextOrganizationId =
      initialIds.length > 0 ? initialOrganizationId : undefined;
    organizationIdRef.current = nextOrganizationId;
    setOrganizationId(nextOrganizationId);
    setSplitByOrder(true);
    setSplitByTaxRate(false);
    setPreview(undefined);
    previewRef.current = undefined;
    setResult(undefined);
    setLoading(false);
    setConfirming(false);
    setIdempotencyKey(generateUUID());
    form.resetFields();

    if (initialIds.length > 0 && initialOrganizationId) {
      // 极速模式：从单票/多选费用带入时，直接拉取预览并切到账单资料页
      const previewPromise = loadPreviewRef.current(
        initialIds,
        {
          mode: groupingMode,
          splitByOrder: true,
          splitByTaxRate: false,
        },
        initialOrganizationId,
      );
      previewPendingRef.current = { key: initKey, promise: previewPromise };
      void previewPromise.then((ok) => {
        if (cancelled) return;
        if (ok) {
          setCurrent(2);
        } else {
          setCurrent(0);
        }
      });
    } else {
      previewPendingRef.current = null;
      setCurrent(0);
    }
    return () => {
      cancelled = true;
    };
  }, [open, initialFeeKey, initialOrganizationId, groupingMode]);

  const invalidatePreview = useCallback(() => {
    if (previewTimerRef.current) {
      clearTimeout(previewTimerRef.current);
      previewTimerRef.current = undefined;
    }
    previewRequestTokenRef.current += 1;
    previewFingerprintRef.current = '';
    previewTokenFingerprintRef.current = '';
    setLoading(false);
    setPreview((currentPreview) =>
      currentPreview
        ? { ...currentPreview, previewToken: undefined }
        : currentPreview,
    );
  }, []);

  const schedulePreview = useCallback(() => {
    if (previewTimerRef.current) clearTimeout(previewTimerRef.current);
    previewTimerRef.current = setTimeout(() => {
      if (organizationIdRef.current && selectedIdsRef.current.length > 0) {
        void loadPreview();
      }
    }, 350);
  }, [loadPreview]);

  const handleConfigurationChange = useCallback(() => {
    invalidatePreview();
    schedulePreview();
  }, [invalidatePreview, schedulePreview]);

  const handleOrganizationChange = (value: string | undefined) => {
    invalidatePreview();
    organizationIdRef.current = value;
    setOrganizationId(value);
    setSelectedFeeIds([]);
    setPreview(undefined);
    previewRef.current = undefined;
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
      setPreview(undefined);
      previewRef.current = undefined;
      invalidatePreview();
      setCurrent(0);
      return;
    }
    setSelectedFeeIds(nextIds);
    invalidatePreview();
    message.success('已从本次建单中移除该费用');
    await loadPreview(nextIds);
  };

  const next = async () => {
    if (current === 0) {
      if (!organizationId) {
        message.warning('请先选择可建账所属公司');
        return;
      }
      if (selectedIds.length === 0) {
        message.warning('请至少选择一笔已确认且未建立账单的费用');
        return;
      }
      setCurrent(1);
      return;
    }
    if (current === 1) {
      if (
        await loadPreview(undefined, {
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
      previewTokenFingerprintRef.current !== previewFingerprintRef.current ||
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
    setLoading(true);
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
        setPreview(undefined);
        setCurrent(1);
      } else if (requestReason(error) === 'FINANCE_BILL_FEE_INVALID') {
        message.error('所选费用必须为已确认状态且尚未进入其他账单');
      } else {
        message.error(requestMessage(error, '批量生成账单失败'));
      }
    } finally {
      setLoading(false);
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
          loadPreview={loadPreview}
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
