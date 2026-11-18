import {
  DeleteOutlined,
  DownOutlined,
  FileDoneOutlined,
  LockOutlined,
  PlusOutlined,
  SaveOutlined,
  SwapOutlined,
  UndoOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ProFormTextArea } from '@ant-design/pro-components';
import { useQueries } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Dropdown,
  Form,
  type MenuProps,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import React, {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  captureFormDraftSession,
  clearFormDraft,
  discardFormDraft,
  getFormDraft,
  getFormDraftKey,
  getFormDraftScope,
  hasFormDraft,
  prepareFormDraftClear,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import {
  ColumnSettingsEntry,
  type ColumnSettingsField,
  type ColumnSettingsValue,
  defaultSelectFilterOption,
  ResponsiveEditableProTable as EditableProTable,
  GuardedModalForm,
  SectionCard,
  scrollToFirstTableError,
} from '@/components/ui';
import { OrderBusinessType, PartnerRoleType } from '@/enums.generated';
import { PartnerQuickCreateSelect } from '@/features/partners';
import { describeRateMissing } from '@/requestErrorConfig';
import { history } from '@/router/history';
import {
  orderFeeServiceBulkRemoveOrderFees,
  orderFeeServiceBulkUpdateOrderFees,
  orderFeeServiceListFees,
  orderFeeServiceSaveOrderFees,
} from '@/services/roncin/orderFeeService';
import { isWriteResponse, unwrapList } from '@/utils/api';
import {
  calculateExactFeeTotal,
  normalizeDecimalInput,
  quantityOrPricePattern,
} from '@/utils/decimal';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import {
  type FeeExchangeRateError,
  type FeeExchangeRateInput,
  fetchFeeExchangeRate,
} from '../../fee-exchange-rate';
import FeeQuantityAssist from './FeeQuantityAssist';
import type { FeeBillTrackingView } from './feeBillTracking';
import {
  clearFeeColumnPreference,
  defaultFeeColumnPreference,
  effectiveFeeColumnDefs,
  type FeeColumnPreference,
  type FeeColumnPreferenceScope,
  isDefaultFeeColumnPreference,
  loadFeeColumnPreference,
  resolveFeeColumnPreference,
  saveFeeColumnPreference,
} from './feeColumnPreference';
import { feeDirectionCode, PAYABLE, RECEIVABLE } from './feeConstants';
import { feeQuantityRuleError } from './feeQuantityRule';
import { FeeQuantitySourceFields, FeeQuantityValue } from './feeQuantitySource';
import {
  buildSaveRow,
  EXCHANGE_RATE_OVERRIDE_MESSAGE,
  type FeeDraftRow,
  type FeeSaveRowError,
  feeRowChanged,
  isNewFeeRowKey,
  NEW_FEE_ROW_PREFIX,
  parseFeeSaveError,
  SaveRequestKeyHolder,
  taxRateInputPattern,
  validateFeeRow,
  validExchangeRateOverride,
} from './feeSave';
import { type FeeSideSummary, summarizeFees } from './feeSummary';
import {
  buildOptionalFeeColumns,
  orderColumnsByPreference,
} from './orderFeeOptionalColumns';

/**
 * 数量、单价的校验错误直接显示在输入框下方；ProTable 行内默认放进点击才弹出的气泡，
 * 业务人员只看到红框不知原因。
 */
const inlineFieldError = { errorType: 'inline' } as Record<string, unknown>;

/** 输入框展示用：去掉后端 decimal 的小数尾零（1.0000 → 1），空值原样保留。 */
const trimDecimalInput = (value?: string) =>
  value ? trimDecimal(value) : value;

/** 已保存费用进入行内表单前，数量与单价按去尾零文本展示；比较与提交仍按数值。 */
const toFeeFormRow = (fee: API.OrderFee): API.OrderFee => ({
  ...fee,
  quantity: trimDecimalInput(fee.quantity),
  unitPrice: trimDecimalInput(fee.unitPrice),
});

/** 费用行按当前可见列顺序切换输入字段，跳过清空及带入等辅助按钮。 */
const handleFeeRowTab = (event: React.KeyboardEvent<HTMLElement>) => {
  if (event.key !== 'Tab' || event.altKey || event.ctrlKey || event.metaKey)
    return;
  const target = event.target;
  if (
    !(
      target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement
    )
  )
    return;
  const fields = Array.from(
    event.currentTarget.querySelectorAll<
      HTMLInputElement | HTMLTextAreaElement
    >('input:not([type="hidden"]):not([type="checkbox"]), textarea'),
  ).filter(
    (field) =>
      !field.disabled &&
      field.tabIndex >= 0 &&
      field.getClientRects().length > 0,
  );
  const index = fields.indexOf(target);
  if (index < 0) return;
  const next = fields[index + (event.shiftKey ? -1 : 1)];
  if (!next) return;
  event.preventDefault();
  event.stopPropagation();
  next.focus();
};

const positiveDecimalRule =
  (pattern: RegExp, messageText: string) => (_: unknown, value?: string) => {
    if (!value) return Promise.resolve();
    // 先去掉小数尾部零再匹配，避免 "211.04500" 这类合法输入被固定位数正则误拒。
    const normalized = normalizeDecimalInput(String(value));
    if (!pattern.test(normalized) || Number(normalized) <= 0) {
      return Promise.reject(new Error(messageText));
    }
    return Promise.resolve();
  };

/** 行内编辑时汇率列的实时预览状态。 */
type ExchangeRatePreview = {
  status: 'loading' | 'resolved' | 'missing' | 'error';
  rate?: string;
};

/** 行内编辑时总金额列的实时预览。 */
type AmountPreview = {
  total: string;
  currency: string;
};

/** 单行编辑态的实时预览集合：汇率、总金额、费用代码、税率与单价，保存后以后端落库值为准。 */
type RowEditPreview = {
  rate?: ExchangeRatePreview;
  amount?: AmountPreview;
  feeCode?: string;
  /** 所选费用项目默认税率（API 口径：百分数值字符串，如 "6.00" 表示 6%）。 */
  taxRate?: string;
  /** 行内输入的单价（供不含税单价列实时折算；编辑始终提交含税口径）。 */
  unitPrice?: string;
  /** 行内手工汇率（格式有效时），优先于系统汇率参与折本币预览。 */
  rateOverride?: string;
  /** 已存手工汇率改回系统汇率，折本币按 rate 中的系统汇率预览。 */
  rateReset?: boolean;
};

export type OrderFeeDraftActions = {
  discard: () => void;
  hasDraft: () => boolean;
};

interface OrderFeeTableTabsProps {
  draftActionsRef?: React.RefObject<OrderFeeDraftActions | undefined>;
  orderId: string;
  receivableActionRef: React.RefObject<ActionType | undefined>;
  payableActionRef: React.RefObject<ActionType | undefined>;
  receivableSummary?: FeeSideSummary;
  payableSummary?: FeeSideSummary;
  selectedReceivableFeeIds: React.Key[];
  setSelectedReceivableFeeIds: (keys: React.Key[]) => void;
  selectedPayableFeeIds: React.Key[];
  setSelectedPayableFeeIds: (keys: React.Key[]) => void;
  setAllReceivableItems: (items: API.OrderFee[]) => void;
  setAllPayableItems: (items: API.OrderFee[]) => void;
  setReceivableSummary: (summary: FeeSideSummary) => void;
  setPayableSummary: (summary: FeeSideSummary) => void;
  canCreateFinanceBills: boolean;
  canCreateFees: boolean;
  /** 整表保存接口要求费用修改权限；仅有新增权限的用户不能保存。 */
  canUpdateFees: boolean;
  canDeleteFees: boolean;
  /** 订单费用税率更改权限（access.canEditFeeTaxRate）；无权限时税率列只读。 */
  canEditFeeTaxRate?: boolean;
  /** 订单费用汇率覆盖权限（access.canOverrideFeeExchangeRate）；无权限时汇率列只读。 */
  canOverrideFeeExchangeRate?: boolean;
  /** 外部「刷新」确认丢弃草稿后递增，两表整体重建并重新加载。 */
  reloadToken?: number;
  /** 任一侧存在未保存修改时通知页面（刷新前确认用）。 */
  onDraftChange?: (hasDraft: boolean) => void;
  feeWritesDisabled: boolean;
  /**
   * 行内输入与新增入口是否按审核状态锁定。父级在审核状态后台重读期间沿用最近确认
   * 的状态，避免保存后整表收起成只读文本再展开造成跳动；未传入时与
   * feeWritesDisabled 一致。提交与财务动作始终由 feeWritesDisabled 判定。
   */
  feeRowsLocked?: boolean;
  onOpenBillWorkbench: (feeIds: string[]) => void;
  getTableColumns?: (direction: number) => ProColumns<API.OrderFee>[];

  feeSettings?: API.OrderFeeSettingOption[];
  settlementParties?: { id?: string; name?: string; code?: string }[];
  currencies?: API.Currency[];
  billingUnits?: API.BillingUnit[];
  order?: API.Order;
  customerName?: string;
  onOpenQuickAddFee?: () => void;
  /** 快捷新建往来单位；direction 为发起入口所在侧，决定默认往来角色。 */
  onOpenQuickAddPartner?: (direction: number) => void;
  /** 列偏好隔离范围（用户 + 当前组织）；缺省时设置仅当前页面会话内生效。 */
  columnSettingScope?: FeeColumnPreferenceScope;
  /** 关联账单投影；仅当页面具备财务费用读取权限时提供，两表共用一次查询。 */
  feeBillTracking?: FeeBillTrackingView;
  /** 费用写入成功后的通知（页面据此刷新关联账单投影）。 */
  onFeeSaved?: (feeContentVersion?: string) => void;
  /** 任一侧整表保存或批量删除进行中；父级据此暂停外部变动比较，避免把本人写入当成外部变动。 */
  onSavingChange?: (saving: boolean) => void;
}

export default function OrderFeeTableTabs({
  draftActionsRef,
  orderId,
  receivableActionRef,
  payableActionRef,
  receivableSummary,
  payableSummary,
  selectedReceivableFeeIds,
  setSelectedReceivableFeeIds,
  selectedPayableFeeIds,
  setSelectedPayableFeeIds,
  setAllReceivableItems,
  setAllPayableItems,
  setReceivableSummary,
  setPayableSummary,
  canCreateFinanceBills,
  canCreateFees,
  canUpdateFees,
  canDeleteFees,
  canEditFeeTaxRate = false,
  canOverrideFeeExchangeRate = false,
  reloadToken = 0,
  onDraftChange,
  feeWritesDisabled,
  feeRowsLocked = feeWritesDisabled,
  onOpenBillWorkbench,
  getTableColumns,
  feeSettings,
  settlementParties,
  currencies,
  billingUnits,
  order,
  customerName,
  onOpenQuickAddFee,
  onOpenQuickAddPartner,
  columnSettingScope,
  feeBillTracking,
  onFeeSaved,
  onSavingChange,
}: OrderFeeTableTabsProps) {
  const { message } = App.useApp();
  const location = history.location;
  const draftScope = getFormDraftScope(
    columnSettingScope?.userId,
    columnSettingScope?.organizationId,
  );
  const draftKeys = useMemo<Record<number, string>>(
    () => ({
      [RECEIVABLE]: getFormDraftKey(
        resolveTabKey(location.pathname),
        `${location.pathname}${location.search}:receivable`,
        draftScope,
      ),
      [PAYABLE]: getFormDraftKey(
        resolveTabKey(location.pathname),
        `${location.pathname}${location.search}:payable`,
        draftScope,
      ),
    }),
    [location.pathname, location.search, draftScope],
  );
  const draftSessions = useMemo<Record<number, object | undefined>>(
    () => ({
      [RECEIVABLE]: captureFormDraftSession(draftKeys[RECEIVABLE]),
      [PAYABLE]: captureFormDraftSession(draftKeys[PAYABLE]),
    }),
    [draftKeys],
  );
  useLayoutEffect(() => {
    if (!draftActionsRef) return;
    const actions = {
      hasDraft: () =>
        [RECEIVABLE, PAYABLE].some((direction) =>
          hasFormDraft(draftKeys[direction]),
        ),
      discard: () => {
        for (const direction of [RECEIVABLE, PAYABLE])
          discardFormDraft(draftKeys[direction], draftSessions[direction]);
      },
    };
    draftActionsRef.current = actions;
    return () => {
      if (draftActionsRef.current === actions)
        draftActionsRef.current = undefined;
    };
  }, [draftActionsRef, draftKeys, draftSessions]);
  type FeeSideDraft = {
    baseline: API.OrderFee[];
    drafts: [string, API.OrderFee][];
    values: Record<string, API.OrderFee>;
    request?: { fingerprint: string; key: string };
  };
  const restoredDrafts = useMemo<Record<number, FeeSideDraft | null>>(
    () => ({
      [RECEIVABLE]: getFormDraft<FeeSideDraft>(draftKeys[RECEIVABLE]),
      [PAYABLE]: getFormDraft<FeeSideDraft>(draftKeys[PAYABLE]),
    }),
    [draftKeys],
  );
  const loadedDraftSides = useRef(new Set<number>());
  const restoredFormSides = useRef(new Set<number>());
  const persistenceRef = useMemo<{ current: (direction: number) => void }>(
    () => ({ current: () => undefined }),
    [draftKeys],
  );
  const currentOrderIdRef = useRef(orderId);
  const localReloadRef = useRef(new Set<number>());
  const mountedRef = useRef(true);
  const receivableRequestSequenceRef = useRef(0);
  const payableRequestSequenceRef = useRef(0);
  const latestReceivableResultRef = useRef<
    | {
        orderId: string;
        items: API.OrderFee[];
      }
    | undefined
  >(undefined);
  const latestPayableResultRef = useRef<
    | {
        orderId: string;
        items: API.OrderFee[];
      }
    | undefined
  >(undefined);

  const [receivableEditableKeys, setReceivableEditableKeys] = useState<
    React.Key[]
  >([]);
  const [payableEditableKeys, setPayableEditableKeys] = useState<React.Key[]>(
    [],
  );
  // 两侧均为多行常开编辑：所有可编辑行始终处于输入态，各侧独立保存；
  // 草稿（输入、新增行、行错误、请求键）按方向隔离，一侧保存或刷新不影响另一侧。
  const rowWritesLocked = feeWritesDisabled || !canUpdateFees;
  const rowInputsLocked = feeRowsLocked || !canUpdateFees;
  const receivableEditable = rowInputsLocked ? [] : receivableEditableKeys;
  const payableEditable = rowInputsLocked ? [] : payableEditableKeys;
  const [dirty, setDirty] = useState({ receivable: false, payable: false });
  const [saving, setSaving] = useState({ receivable: false, payable: false });
  const savingRef = useRef({ receivable: false, payable: false });
  const anySaving = saving.receivable || saving.payable;
  const onSavingChangeRef = useRef(onSavingChange);
  onSavingChangeRef.current = onSavingChange;
  useEffect(() => {
    onSavingChangeRef.current?.(anySaving);
  }, [anySaving]);
  const [rowErrors, setRowErrors] = useState<{
    receivable: FeeSaveRowError[];
    payable: FeeSaveRowError[];
    rateDetail?: string;
  }>({ receivable: [], payable: [] });
  const receivableFormRef = useRef<ProFormInstance | undefined>(undefined);
  const payableFormRef = useRef<ProFormInstance | undefined>(undefined);
  const draftRowsRef = useRef<Record<number, Map<string, API.OrderFee>>>({
    [RECEIVABLE]: new Map(),
    [PAYABLE]: new Map(),
  });
  const requestKeyHoldersRef = useRef<Record<number, SaveRequestKeyHolder>>({
    [RECEIVABLE]: new SaveRequestKeyHolder(generateUUID),
    [PAYABLE]: new SaveRequestKeyHolder(generateUUID),
  });
  const receivableGuard = useUnsavedChangesGuard({
    enabled: Boolean(orderId),
    identity: orderId,
    draftKey: draftKeys[RECEIVABLE],
  });
  const payableGuard = useUnsavedChangesGuard({
    enabled: Boolean(orderId),
    identity: orderId,
    draftKey: draftKeys[PAYABLE],
  });
  const sideName = (direction: number) =>
    direction === RECEIVABLE ? 'receivable' : 'payable';
  const formRefOf = (direction: number) =>
    direction === RECEIVABLE ? receivableFormRef : payableFormRef;
  const markDirty = (direction: number) => {
    if (direction === RECEIVABLE) receivableGuard.markDirty();
    else payableGuard.markDirty();
    setDirty((prev) =>
      prev[sideName(direction)]
        ? prev
        : { ...prev, [sideName(direction)]: true },
    );
  };
  const clearDirty = (direction: number) => {
    clearFormDraft(draftKeys[direction], draftSessions[direction]);
    if (direction === RECEIVABLE) receivableGuard.resetDirty();
    else payableGuard.resetDirty();
    setDirty((prev) =>
      prev[sideName(direction)]
        ? { ...prev, [sideName(direction)]: false }
        : prev,
    );
  };
  const hasDraft = dirty.receivable || dirty.payable;
  const onDraftChangeRef = useRef(onDraftChange);
  onDraftChangeRef.current = onDraftChange;
  useEffect(() => {
    onDraftChangeRef.current?.(hasDraft);
  }, [hasDraft]);

  // 列设置状态：应收/应付共享同一份偏好；编辑进行中禁用入口，避免丢失未保存行。
  const columnEditing = hasDraft;
  // 账单号与关联账单财务进度列仅在页面提供关联投影（财务读取权限）时可用。
  const financeAvailable = Boolean(feeBillTracking);

  const columnPreferenceIdentity = JSON.stringify([
    columnSettingScope?.userId ?? '',
    columnSettingScope?.organizationId ?? '',
    financeAvailable,
  ]);
  const loadedColumnPreference = useMemo(
    () =>
      resolveFeeColumnPreference(
        columnSettingScope ? loadFeeColumnPreference(columnSettingScope) : null,
        financeAvailable,
      ),
    [
      columnSettingScope?.userId,
      columnSettingScope?.organizationId,
      financeAvailable,
    ],
  );
  const [columnPreferenceState, setColumnPreferenceState] = useState<{
    identity: string;
    preference: FeeColumnPreference;
  } | null>(null);
  // 切换用户、组织或财务权限时，本次渲染立即使用对应配置，不闪现旧作用域。
  const feeColumnPref =
    columnPreferenceState?.identity === columnPreferenceIdentity
      ? columnPreferenceState.preference
      : loadedColumnPreference;

  /** 统一列设置变更：应收/应付共享；恢复默认等价偏好时清空本地存储。 */
  const handleColumnSettingsChange = (next: ColumnSettingsValue) => {
    // 字段清单来自 feeColumnPreference 域定义，运行时 key 必然属于该联合类型。
    const preference = resolveFeeColumnPreference(
      {
        order: next.order as FeeColumnPreference['order'],
        hidden: next.hidden as FeeColumnPreference['hidden'],
        freeze: next.freeze,
      },
      financeAvailable,
    );
    setColumnPreferenceState({
      identity: columnPreferenceIdentity,
      preference,
    });
    const scope = columnSettingScope ?? {};
    const persisted = isDefaultFeeColumnPreference(preference, financeAvailable)
      ? clearFeeColumnPreference(scope)
      : saveFeeColumnPreference(scope, preference);
    if (!persisted) {
      // 存储失败：表格已应用本次设置，提示未持久化。
      message.error('列设置保存到本地浏览器失败，本次设置仅当前页面生效');
    }
  };

  /** 订单费用列设置元数据：含权限裁剪与必显标记，供统一弹窗消费。 */
  const feeColumnFields: ColumnSettingsField[] = effectiveFeeColumnDefs(
    financeAvailable,
  ).map(({ key, title, lockVisible }) => ({ key, title, lockVisible }));

  const renderColumnToolbar = () => {
    return (
      <Space size={8}>
        {feeBillTracking?.state === 'error' && (
          <Space size={4}>
            <Tag color="error">账单关联加载失败</Tag>
            <Button
              type="link"
              size="small"
              style={{ padding: 0 }}
              onClick={feeBillTracking.onRetry}
            >
              重试
            </Button>
          </Space>
        )}
        {feeBillTracking && (
          <Button
            type="link"
            size="small"
            style={{ padding: 0 }}
            onClick={() => history.push(`/finance/fees/detail/${orderId}`)}
          >
            财务详情
          </Button>
        )}
        <ColumnSettingsEntry
          fields={feeColumnFields}
          value={feeColumnPref}
          onChange={handleColumnSettingsChange}
          onReset={() =>
            handleColumnSettingsChange(
              defaultFeeColumnPreference(financeAvailable),
            )
          }
          disabled={columnEditing || Boolean(getTableColumns)}
          disabledReason={
            columnEditing
              ? '请先保存正在编辑的费用行'
              : '外部自定义列模式下不支持列设置'
          }
        />
      </Space>
    );
  };

  // ===== 批量维护（阶段二）：应收/应付两表各自一套入口，作用于该表勾选行 =====

  /** 按方向取该表最近一次成功加载的有效费用行（版本号快照来源）。 */
  const getDirectionItems = (direction: number): API.OrderFee[] => {
    const result =
      direction === RECEIVABLE
        ? latestReceivableResultRef.current
        : latestPayableResultRef.current;
    return result?.orderId === orderId ? result.items : [];
  };

  /** 与保存使用相同的字段比较，改回原值后恢复无改动状态。 */
  const refreshDirty = (direction: number) => {
    const form = formRefOf(direction).current;
    const changed =
      draftRowsRef.current[direction].size > 0 ||
      getDirectionItems(direction).some(
        (row) =>
          row.id &&
          row.canEdit === true &&
          feeRowChanged({ ...row, ...form?.getFieldValue([row.id]) }, row),
      );
    if (changed) {
      markDirty(direction);
      persistenceRef.current(direction);
    } else clearDirty(direction);
  };

  /** 由选中 ID 解析可执行批量操作的目标行：仅已保存且带版本的行。 */
  const getSelectedRows = (direction: number): API.OrderFee[] => {
    const selectedIds = new Set(
      (direction === RECEIVABLE
        ? selectedReceivableFeeIds
        : selectedPayableFeeIds
      ).map(String),
    );
    return getDirectionItems(direction).filter(
      (row) => row.id && selectedIds.has(String(row.id)) && row.version,
    );
  };

  /** 构造批量契约目标：费用 ID + 勾选时的版本快照，按 ID 排序保持固定顺序。 */
  const buildBulkTargets = (rows: API.OrderFee[]): API.BulkOrderFeeTarget[] =>
    [...rows]
      .sort((a, b) => String(a.id).localeCompare(String(b.id)))
      .map((row) => ({
        feeId: String(row.id),
        expectedVersion: String(row.version),
      }));

  /** 批量写入与建账均消费服务端逐费能力；混合选择显式报错，禁止静默过滤。 */
  const rejectUnavailableRows = (
    rows: API.OrderFee[],
    actionText: string,
    capability: 'canEdit' | 'canDelete' | 'canBill' = 'canEdit',
  ) => {
    const billedRows = rows.filter((row) => row[capability] !== true);
    if (billedRows.length === 0) return false;
    const names = billedRows
      .map((row) => row.feeName || row.feeCode || row.id || '')
      .filter(Boolean);
    const brief =
      names.slice(0, 3).join('、') + (names.length > 3 ? ' 等' : '');
    message.warning(
      `${billedRows.every((row) => row.hasActiveBill) ? `选中费用中有 ${billedRows.length} 笔已建账` : `选中费用中有 ${billedRows.length} 笔当前不可操作`}，不能${actionText}（${brief}），${billedRows.flatMap((row) => row.blockedReasons ?? []).join('；') || '请取消勾选后重试'}`,
    );
    return true;
  };

  /** 批量操作成功后：清空该表选择，刷新两表与关联账单/汇总链路；失败不调用以保留核对上下文。 */
  const handleBulkDone = (direction: number) => {
    if (direction === RECEIVABLE) {
      setSelectedReceivableFeeIds([]);
    } else {
      setSelectedPayableFeeIds([]);
    }
    receivableActionRef.current?.reload();
    payableActionRef.current?.reload();
    onFeeSaved?.();
  };

  /** 建账入口显式资格校验：仅未建账费用可生成账单，混合勾选时明确拒绝。 */
  const handleOpenBill = (direction: number) => {
    if (savingRef.current[sideName(direction)]) return;
    const rows = getSelectedRows(direction);
    const selectedIds =
      direction === RECEIVABLE
        ? selectedReceivableFeeIds
        : selectedPayableFeeIds;
    if (rows.length !== selectedIds.length) {
      message.warning('请先保存新增费用，再执行财务操作');
      return;
    }
    if (rows.length === 0) {
      message.warning('未找到勾选的费用，请刷新后重试');
      return;
    }
    if (rejectUnavailableRows(rows, '生成账单', 'canBill')) return;
    onOpenBillWorkbench(rows.map((row) => String(row.id)));
  };

  type BulkActionType = 'settlementParty' | 'remove';
  const [bulkActionContext, setBulkActionContext] = useState<
    { direction: number; type: BulkActionType } | undefined
  >(undefined);

  const renderBulkToolbar = (direction: number) => {
    const isReceivable = direction === RECEIVABLE;
    const selectedIds = isReceivable
      ? selectedReceivableFeeIds
      : selectedPayableFeeIds;
    const bulkDisabled =
      feeWritesDisabled ||
      saving[sideName(direction)] ||
      selectedIds.length === 0;

    const items: MenuProps['items'] = [
      {
        key: 'settlementParty',
        icon: <SwapOutlined />,
        label: '批量改结算单位',
        onClick: () =>
          setBulkActionContext({ direction, type: 'settlementParty' }),
      },
    ];

    return (
      <Dropdown menu={{ items }} disabled={bulkDisabled} trigger={['click']}>
        <Button disabled={bulkDisabled}>
          <Space size={4}>
            <span>批量操作</span>
            {selectedIds.length > 0 && <span>({selectedIds.length})</span>}
            <DownOutlined style={{ fontSize: 10 }} />
          </Space>
        </Button>
      </Dropdown>
    );
  };

  // 行内编辑的实时预览：按行 key 记录汇率解析进度、总金额与费用代码，
  // 币种/单价/数量/费用项目变化即刷新，保存后以后端落库值为准。
  const [draftRowPreviews, setRowPreviews] = useState<
    Record<string, RowEditPreview>
  >({});
  const rateScope = `${orderId}:${reloadToken}`;
  const [rateRequests, setRateRequests] = useState<{
    scope: string;
    rows: Record<string, FeeExchangeRateInput>;
  }>({ scope: rateScope, rows: {} });
  const editableRows = new Set(
    [...receivableEditable, ...payableEditable].map(String),
  );
  const requestedRows =
    rateRequests.scope === rateScope
      ? Object.entries(rateRequests.rows).filter(([key]) =>
          editableRows.has(key),
        )
      : [];
  // 相同订单、方向和币种共用一个查询，避免多行同时编辑时重复取同一参考汇率。
  const rateInputs = [
    ...new Map(
      requestedRows.map(([, input]) => [
        `${input.direction}:${input.currency}`,
        input,
      ]),
    ).values(),
  ];
  const rateQueries = useQueries({
    queries: rateInputs.map((input) => ({
      queryKey: ['orders', 'row-fee-exchange-rate', { ...input, reloadToken }],
      queryFn: async ({
        signal,
      }: {
        signal: AbortSignal;
      }): Promise<ExchangeRatePreview> => {
        try {
          return {
            status: 'resolved',
            rate: await fetchFeeExchangeRate(input, signal),
          };
        } catch (error) {
          if (
            (error as FeeExchangeRateError).reason ===
            'FEE_EXCHANGE_RATE_MISSING'
          )
            return { status: 'missing' };
          throw error;
        }
      },
    })),
  });
  const rowPreviews: Record<string, RowEditPreview> = { ...draftRowPreviews };
  for (const [key, input] of requestedRows) {
    const index = rateInputs.findIndex(
      (item) =>
        item.direction === input.direction && item.currency === input.currency,
    );
    const query = rateQueries[index];
    const rate: ExchangeRatePreview = query.isError
      ? { status: 'error' }
      : (query.data ?? { status: 'loading' });
    rowPreviews[key] = { ...rowPreviews[key], rate };
  }

  currentOrderIdRef.current = orderId;

  useEffect(() => {
    mountedRef.current = true;
    setReceivableEditableKeys([]);
    setPayableEditableKeys([]);
    setRowPreviews({});
    setRateRequests({ scope: rateScope, rows: {} });
    return () => {
      mountedRef.current = false;
      receivableRequestSequenceRef.current += 1;
      payableRequestSequenceRef.current += 1;
    };
  }, []);

  useEffect(() => {
    setRowPreviews((prev) => {
      const activeKeys = new Set(
        [...receivableEditable, ...payableEditable].map(String),
      );
      const next: Record<string, RowEditPreview> = {};
      let changed = false;
      for (const [key, preview] of Object.entries(prev)) {
        if (activeKeys.has(key)) {
          next[key] = preview;
        } else {
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [receivableEditable, payableEditable]);

  const lastResetIdentity = useRef({ orderId, reloadToken });
  // 切换订单或外部确认刷新后：丢弃全部草稿状态，表格随 key 重建并重新加载。
  useEffect(() => {
    if (
      lastResetIdentity.current.orderId === orderId &&
      lastResetIdentity.current.reloadToken === reloadToken
    )
      return;
    lastResetIdentity.current = { orderId, reloadToken };
    localReloadRef.current.clear();
    setRowPreviews({});
    setRateRequests({ scope: rateScope, rows: {} });
    setRowErrors({ receivable: [], payable: [] });
    setDirty({ receivable: false, payable: false });
    for (const holder of Object.values(requestKeyHoldersRef.current)) {
      holder.clear();
    }
    for (const rows of Object.values(draftRowsRef.current)) rows.clear();
    receivableGuard.resetDirty();
    payableGuard.resetDirty();
    for (const direction of [RECEIVABLE, PAYABLE])
      clearFormDraft(draftKeys[direction], draftSessions[direction]);
  }, [orderId, reloadToken]);

  persistenceRef.current = (direction) => {
    const isDirty = direction === RECEIVABLE ? dirty.receivable : dirty.payable;
    if (
      (!isDirty && draftRowsRef.current[direction].size === 0) ||
      !formRefOf(direction).current
    )
      return;
    saveFormDraft(
      draftKeys[direction],
      {
        baseline: getDirectionItems(direction),
        drafts: [...draftRowsRef.current[direction]],
        values: formRefOf(direction).current?.getFieldsValue(true) ?? {},
        request: requestKeyHoldersRef.current[direction].snapshot(),
      } satisfies FeeSideDraft,
      draftSessions[direction],
    );
  };
  useLayoutEffect(() => {
    for (const direction of [RECEIVABLE, PAYABLE]) {
      const draft = restoredDrafts[direction];
      const form = formRefOf(direction).current;
      if (
        draft &&
        loadedDraftSides.current.has(direction) &&
        form &&
        !restoredFormSides.current.has(direction)
      ) {
        restoredFormSides.current.add(direction);
        form.setFieldsValue(draft.values);
      }
      persistenceRef.current(direction);
    }
  });
  useLayoutEffect(
    () => () => {
      persistenceRef.current(RECEIVABLE);
      persistenceRef.current(PAYABLE);
    },
    [draftKeys],
  );

  // 默认计费单位：优先查找“票”（PIAO / 包含票），否则选用首项
  const defaultBillingUnit =
    billingUnits?.find(
      (u) => u.code === 'PIAO' || u.name === '票' || u.name?.includes('票'),
    ) || billingUnits?.[0];

  // 事件只更新查询参数，汇率数据和请求状态由当前行所订阅的 Query 管理。
  const resolveRowRatePreview = (
    rowKey: React.Key | undefined,
    direction: number,
    currency?: string,
  ) => {
    const key = String(rowKey ?? '');
    setRateRequests((previous) => {
      const rows = previous.scope === rateScope ? previous.rows : {};
      if (!currency) {
        if (!rows[key]) return previous;
        const { [key]: _removed, ...remaining } = rows;
        return { scope: rateScope, rows: remaining };
      }
      if (
        rows[key]?.currency === currency &&
        rows[key]?.direction === direction
      )
        return previous;
      return {
        scope: rateScope,
        rows: { ...rows, [key]: { orderId, direction, currency } },
      };
    });
  };

  /**
   * 计费单位或带入数量变化后重跑数量字段规则；校验失败只展示错误，不抛出。
   * ProTable 列回调给出的 rowKey 可能是 ['id'] 形式的路径，validateFields 按段
   * 精确匹配，须先展平，否则静默匹配不到字段。
   */
  const revalidateQuantity = (
    form:
      | { validateFields: (names: React.Key[][]) => Promise<unknown> }
      | undefined,
    rowKey: React.Key | React.Key[] | undefined,
  ) => {
    if (!form || rowKey === undefined) return;
    void form
      .validateFields([[...[rowKey].flat(), 'quantity']])
      .catch(() => undefined);
  };

  // 单价/数量/币种变化后重算总金额预览，并同步行内单价供不含税单价列
  // 实时折算；输入不完整时保留空预览，避免显示保存前的值。
  const refreshAmountPreview = (
    rowKey: React.Key | undefined,
    form?: {
      getFieldValue: (name: [React.Key | undefined, string]) => unknown;
    },
  ) => {
    const key = String(rowKey ?? '');
    const quantity = form?.getFieldValue([rowKey, 'quantity']);
    const unitPrice = form?.getFieldValue([rowKey, 'unitPrice']);
    const currency = form?.getFieldValue([rowKey, 'currency']);
    const total = calculateExactFeeTotal(
      String(quantity ?? ''),
      String(unitPrice ?? ''),
    );
    const unitPriceInput = unitPrice
      ? normalizeDecimalInput(String(unitPrice))
      : '';
    const hasUnitPrice =
      unitPriceInput !== '' && Number.isFinite(Number(unitPriceInput));
    const feeSettingId = form?.getFieldValue([rowKey, 'feeSettingId']);
    const catalogTaxRate = feeSettings?.find(
      (item) => item.id === feeSettingId,
    )?.taxRate;
    // 税率列可编辑时，行内输入的有效税率优先于目录默认税率参与折算预览。
    const taxInput = String(
      form?.getFieldValue([rowKey, 'taxRate']) ?? '',
    ).trim();
    const taxRate =
      taxInput !== '' &&
      taxRateInputPattern.test(normalizeDecimalInput(taxInput))
        ? normalizeDecimalInput(taxInput)
        : catalogTaxRate;
    setRowPreviews((prev) => ({
      ...prev,
      [key]: {
        ...prev[key],
        // 空字符串明确表示当前输入已清空，计算列不能复用保存前的值。
        unitPrice: hasUnitPrice ? unitPriceInput : '',
        taxRate: taxRate ?? '',
        amount: total
          ? { total, currency: String(currency || 'CNY') }
          : undefined,
      },
    }));
  };

  // 选择费用项目后同步预览费用代码与默认税率列。
  const setRowFeeOptionPreview = (
    rowKey: React.Key | undefined,
    option?: { feeCode?: string; taxRate?: string },
  ) => {
    const key = String(rowKey ?? '');
    setRowPreviews((prev) => {
      const current = prev[key];
      const next: RowEditPreview = { ...current };
      let changed = false;
      if (option?.feeCode !== undefined) {
        next.feeCode = option.feeCode;
        changed = true;
      }
      if (option?.taxRate !== undefined) {
        next.taxRate = option.taxRate;
        changed = true;
      }
      if (!changed) return prev;
      return { ...prev, [key]: next };
    });
  };

  // 行内手工汇率：格式有效时进入预览参与折本币金额，清空或无效时回到系统/当前汇率。
  const setRowRateOverride = (rowKey: React.Key | undefined, raw?: string) => {
    const key = String(rowKey ?? '');
    const text = normalizeDecimalInput(String(raw ?? '').trim());
    const rateOverride =
      text !== '' && validExchangeRateOverride(text) ? text : undefined;
    setRowPreviews((prev) => {
      const current = prev[key];
      if (current?.rateOverride === rateOverride) return prev;
      return { ...prev, [key]: { ...current, rateOverride } };
    });
  };

  /**
   * 已存手工汇率行的「改回系统汇率」开关：状态写入行表单值 useSystemExchangeRate 随保存
   * 提交；开启时清空手工输入并预览系统汇率，关闭时撤销预览回到当前手工汇率。
   */
  const setRowRateReset = (
    direction: number,
    form: {
      setFieldValue: (name: [string, string], value: unknown) => void;
    },
    key: string,
    enabled: boolean,
    currency?: string,
  ) => {
    form.setFieldValue([key, 'useSystemExchangeRate'], enabled || undefined);
    if (enabled) {
      form.setFieldValue([key, 'exchangeRateOverride'], undefined);
      setRowRateOverride(key, undefined);
    }
    setRowPreviews((prev) => {
      const current = prev[key];
      if (Boolean(current?.rateReset) === enabled) return prev;
      return {
        ...prev,
        [key]: { ...current, rateReset: enabled || undefined },
      };
    });
    resolveRowRatePreview(key, direction, enabled ? currency : undefined);
    refreshDirty(direction);
  };

  /** 在本侧表格顶部新增一行；新增行以 dataSource 模式加入，允许同时存在多行。 */
  const handleAddRow = (direction: number) => {
    if (rowInputsLocked || !canCreateFees) return;
    const isReceivable = direction === RECEIVABLE;
    const newId = `${NEW_FEE_ROW_PREFIX}${generateUUID()}`;
    const defaultPartyId = isReceivable
      ? order?.customerId
      : order?.bookingAgentId;
    const defaultPartyName = isReceivable
      ? customerName ||
        settlementParties?.find((p) => p.id === defaultPartyId)?.name
      : settlementParties?.find((p) => p.id === defaultPartyId)?.name;
    const record: API.OrderFee = {
      id: newId,
      direction,
      currency: 'CNY',
      quantity: '1',
      billingUnitId: defaultBillingUnit?.id,
      billingUnit: defaultBillingUnit?.name,
      settlementPartyId: defaultPartyId,
      settlementPartyName: defaultPartyName,
    };
    const actionRef = isReceivable ? receivableActionRef : payableActionRef;
    const added = actionRef.current?.addEditRecord?.(record, {
      position: 'top',
      newRecordType: 'dataSource',
    });
    if (added === false) return;
    draftRowsRef.current[direction].set(newId, record);
    markDirty(direction);
    resolveRowRatePreview(newId, direction, 'CNY');
  };

  /** 费用字段标签：行错误提示用。 */
  const feeFieldLabels: Record<string, string> = {
    feeSettingId: '费用项目',
    settlementPartyId: '结算单位',
    currency: '币种',
    billingUnitId: '计费单位',
    quantity: '数量',
    unitPrice: '单价',
    taxRate: '税率',
    note: '备注',
    exchangeRateOverride: '汇率',
    version: '版本',
  };

  const rowLabelOf = (direction: number, clientRowId: string) => {
    const baseline = getDirectionItems(direction).find(
      (item) => item.id === clientRowId,
    );
    if (baseline) return baseline.feeName || baseline.feeCode || '费用';
    const settingId = formRefOf(direction).current?.getFieldValue([
      clientRowId,
      'feeSettingId',
    ]);
    const setting = feeSettings?.find((item) => item.id === settingId);
    return setting?.nameZh || setting?.feeCode || '新增费用';
  };

  /** 把行级错误标到对应单元格；无对应列的字段只在错误汇总中展示。 */
  const applyRowErrorsToForm = (
    direction: number,
    errors: FeeSaveRowError[],
  ) => {
    const form = formRefOf(direction).current;
    if (!form) return;
    const fields = errors
      .filter(
        (item) => item.field in feeFieldLabels && item.field !== 'version',
      )
      .map((item) => ({
        name: [item.clientRowId, item.field],
        errors: [item.message],
      }));
    if (fields.length > 0) form.setFields(fields);
  };

  /** 本侧整表保存：只提交新增行与已改动行，未改动行不提交，失败整体保留输入。 */
  const saveSide = async (direction: number) => {
    const side = sideName(direction);
    if (!orderId) return;
    const form = formRefOf(direction).current;
    if (!form) return;
    if (rowWritesLocked) {
      message.warning('费用当前不可修改，请刷新后重试');
      return;
    }
    const requestedOrderId = orderId;
    const baselineById = new Map(
      getDirectionItems(direction).map((item) => [String(item.id), item]),
    );
    const editingKeys = (
      direction === RECEIVABLE ? receivableEditable : payableEditable
    ).map(String);

    // 候选行：本侧所有编辑行，值取表单当前输入并以基线/新增默认值补齐未渲染字段。
    const candidates = editingKeys.flatMap((key) => {
      const isNew = isNewFeeRowKey(key);
      const baseline = isNew ? undefined : baselineById.get(key);
      if (!isNew && !baseline) return [];
      const stored = (form.getFieldValue([key]) ?? {}) as FeeDraftRow;
      const row: FeeDraftRow = {
        ...(baseline ?? draftRowsRef.current[direction].get(key) ?? {}),
        ...stored,
      };
      if (!isNew && baseline && !feeRowChanged(row, baseline)) return [];
      return [{ key, isNew, baseline, row }];
    });
    if (candidates.length === 0) {
      message.info('本侧费用没有需要保存的修改');
      return;
    }
    if (!canCreateFees && candidates.some((item) => item.isNew)) {
      message.warning('无费用录入权限，不能保存新增行');
      return;
    }

    // 校验：先跑单元格规则，再跑提交前业务校验；任一失败不发请求。
    const fieldNames = Object.keys(feeFieldLabels).filter(
      (name) =>
        name !== 'version' &&
        (name !== 'exchangeRateOverride' || canOverrideFeeExchangeRate),
    );
    try {
      await form.validateFields(
        candidates.flatMap((item) =>
          fieldNames.map((name) => [item.key, name]),
        ),
      );
    } catch (error) {
      const first = (error as { errorFields?: { name: (string | number)[] }[] })
        ?.errorFields?.[0];
      message.warning('请先修正标红的费用行');
      scrollToFirstTableError({
        rowKey: first ? String(first.name[0]) : undefined,
      });
      return;
    }
    for (const item of candidates) {
      const issue = validateFeeRow(
        item.row,
        item.baseline,
        billingUnits ?? [],
        canEditFeeTaxRate,
      );
      if (issue) {
        form.setFields([
          { name: [item.key, issue.field], errors: [issue.message] },
        ]);
        message.error(issue.message);
        scrollToFirstTableError({
          rowKey: item.key,
          errorFields: [
            { name: [item.key, issue.field], errors: [issue.message] },
          ],
        });
        return;
      }
    }

    const rows = candidates.map((item) =>
      buildSaveRow({
        clientRowId: item.key,
        row: item.row,
        baseline: item.baseline,
        catalogTaxRate: feeSettings?.find(
          (setting) => setting.id === item.row.feeSettingId,
        )?.taxRate,
      }),
    );
    const requestKey = requestKeyHoldersRef.current[direction].keyFor(rows);
    persistenceRef.current(direction);
    const clearSubmittedDraft = prepareFormDraftClear(
      draftKeys[direction],
      draftSessions[direction],
    );
    setRowErrors((prev) => ({ ...prev, [side]: [], rateDetail: undefined }));
    try {
      const response = await orderFeeServiceSaveOrderFees(
        { orderId },
        { orderId, direction, requestKey, rows },
        { skipErrorHandler: true },
      );
      if (response && response.success !== false) clearSubmittedDraft();
      if (!mountedRef.current || currentOrderIdRef.current !== requestedOrderId)
        return;
      if (!response || response.success === false) {
        message.error('未能确认保存结果，原样重试不会重复新增');
        return;
      }
      requestKeyHoldersRef.current[direction].clear();
      for (const item of candidates) {
        if (item.isNew) draftRowsRef.current[direction].delete(item.key);
      }
      clearDirty(direction);
      message.success(
        response.replayed
          ? '这批费用此前已保存，已同步最新数据'
          : `已保存 ${rows.length} 笔${direction === RECEIVABLE ? '应收' : '应付'}费用`,
      );
      // 只刷新本侧：另一侧的草稿与基线完全保留。
      try {
        await (direction === RECEIVABLE
          ? receivableActionRef
          : payableActionRef
        ).current?.reload();
      } catch {
        message.warning('费用已保存，列表刷新失败，请手动刷新');
      }
      onFeeSaved?.(response.feeContentVersion);
    } catch (error) {
      if (!mountedRef.current || currentOrderIdRef.current !== requestedOrderId)
        return;
      const parsed = parseFeeSaveError(error);
      const rateDetail = describeRateMissing({
        reason: parsed.reason,
        metadata: parsed.metadata,
      });
      if (parsed.rowErrors.length > 0) {
        setRowErrors((prev) => ({
          ...prev,
          [side]: parsed.rowErrors,
          rateDetail,
        }));
        applyRowErrorsToForm(direction, parsed.rowErrors);
      } else if (rateDetail) {
        setRowErrors((prev) => ({ ...prev, rateDetail }));
      }
      message.error(
        rateDetail ? `${parsed.message}（${rateDetail}）` : parsed.message,
      );
    }
  };

  // 校验及提交期间冻结本侧输入，避免成功刷新覆盖请求发出后的新编辑；另一侧仍可编辑。
  const handleSaveSide = async (direction: number) => {
    const side = sideName(direction);
    if (savingRef.current[side]) return;
    savingRef.current[side] = true;
    setSaving((prev) => ({ ...prev, [side]: true }));
    try {
      await saveSide(direction);
    } finally {
      savingRef.current[side] = false;
      if (mountedRef.current) setSaving((prev) => ({ ...prev, [side]: false }));
    }
  };

  const buildColumns = (direction: number): ProColumns<API.OrderFee>[] => {
    if (getTableColumns) {
      return getTableColumns(direction);
    }
    const isReceivable = direction === RECEIVABLE;

    return [
      {
        title: '费用代码',
        dataIndex: 'feeCode',
        width: 100,
        editable: false,
        // 行内编辑预览依赖外部状态，单元格需随组件重渲染刷新。
        shouldCellUpdate: () => true,
        render: (_, record) => {
          const code =
            rowPreviews[String(record.id ?? '')]?.feeCode ??
            record.feeCode ??
            '-';
          const rowKey = String(record.id ?? '');
          const editingKeys = isReceivable
            ? receivableEditable
            : payableEditable;
          if (!rowKey || editingKeys.map(String).includes(rowKey)) return code;
          // 只读行：悬停说明不可编辑原因，优先使用服务端返回的阻断原因。
          const reason =
            record.blockedReasons?.join('；') ||
            (!canUpdateFees ? '没有费用编辑权限' : '当前费用不可编辑');
          return (
            <Tooltip title={reason}>
              <Space size={4} data-testid={`fee-readonly-${rowKey}`}>
                <LockOutlined style={{ color: '#8c8c8c' }} />
                <span>{code}</span>
              </Space>
            </Tooltip>
          );
        },
      },
      {
        title: '费用名称',
        dataIndex: 'feeSettingId',
        width: 200,
        formItemProps: {
          rules: [{ required: true, message: '请选择费用项目' }],
        },
        render: (_, record) => record.feeName || '-',
        valueType: 'select',
        fieldProps: (form, { rowKey }) => ({
          showSearch: true,
          autoFocus: true,
          placeholder: '请选择费用项目',
          options: (feeSettings || []).map((item) => ({
            label: `${item.nameZh || item.nameEn || item.feeCode} (${item.feeCode})`,
            value: item.id ?? '',
            nameZh: item.nameZh,
            nameEn: item.nameEn,
            aliasName: item.aliasName,
            feeCode: item.feeCode,
            taxRate: item.taxRate,
            defaultCurrency: item.defaultCurrency,
            defaultBillingUnitId: item.defaultBillingUnitId,
          })),
          filterOption: defaultSelectFilterOption,
          onChange: (
            _: string,
            option?: {
              nameZh?: string;
              feeCode?: string;
              taxRate?: string;
              defaultCurrency?: string;
              defaultBillingUnitId?: string;
            },
          ) => {
            if (option) {
              form?.setFieldValue([rowKey, 'feeName'], option.nameZh);
              form?.setFieldValue([rowKey, 'feeCode'], option.feeCode);
              setRowFeeOptionPreview(rowKey, option);
              if (option.defaultCurrency) {
                form?.setFieldValue(
                  [rowKey, 'currency'],
                  option.defaultCurrency,
                );
                resolveRowRatePreview(
                  rowKey,
                  direction,
                  option.defaultCurrency,
                );
              }
              refreshAmountPreview(rowKey, form);
              if (option.defaultBillingUnitId) {
                const bu = billingUnits?.find(
                  (u) => u.id === option.defaultBillingUnitId,
                );
                form?.setFieldValue([rowKey, 'quantitySource'], {
                  mode: 'MANUAL',
                });
                form?.setFieldValue(
                  [rowKey, 'billingUnitId'],
                  option.defaultBillingUnitId,
                );
                if (bu) {
                  form?.setFieldValue([rowKey, 'billingUnit'], bu.name);
                }
                revalidateQuantity(form, rowKey);
              }
            }
          },
          popupRender: (menu: React.ReactNode) => (
            <>
              {menu}
              {onOpenQuickAddFee && (
                <div
                  style={{
                    padding: '6px 12px',
                    cursor: 'pointer',
                    color: '#1677ff',
                    fontSize: 12,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    background: '#f6faff',
                    borderTop: '1px solid #f0f0f0',
                  }}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onOpenQuickAddFee();
                  }}
                >
                  <PlusOutlined /> 快捷新增费用科目
                </div>
              )}
            </>
          ),
        }),
      },
      {
        title: '结算单位',
        dataIndex: 'settlementPartyId',
        width: 200,
        ellipsis: true,
        formItemProps: {
          rules: [{ required: true, message: '请选择结算单位' }],
        },
        render: (_, record) => record.settlementPartyName || '-',
        valueType: 'select',
        fieldProps: (form, { rowKey }) => ({
          showSearch: true,
          placeholder: '请选择结算单位',
          options: (settlementParties || []).map((p) => ({
            label: p.name,
            value: p.id ?? '',
            code: p.code,
          })),
          filterOption: defaultSelectFilterOption,
          onChange: (_: string, option?: { label?: string }) => {
            form?.setFieldValue([rowKey, 'settlementPartyName'], option?.label);
          },
          popupRender: (menu: React.ReactNode) => (
            <>
              {menu}
              {onOpenQuickAddPartner && (
                <div
                  style={{
                    padding: '6px 12px',
                    cursor: 'pointer',
                    color: '#1677ff',
                    fontSize: 12,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    background: '#f6faff',
                    borderTop: '1px solid #f0f0f0',
                  }}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onOpenQuickAddPartner(direction);
                  }}
                >
                  <PlusOutlined /> 快捷新增往来单位
                </div>
              )}
            </>
          ),
        }),
      },
      {
        title: '币种',
        dataIndex: 'currency',
        width: 90,
        formItemProps: {
          rules: [{ required: true, message: '请选择币种' }],
        },
        render: (_, record) => <Tag color="blue">{record.currency}</Tag>,
        valueType: 'select',
        fieldProps: (form, { rowKey }) => ({
          showSearch: true,
          filterOption: defaultSelectFilterOption,
          placeholder: '请选择币种',
          options: (currencies || []).map((c) => ({
            label: c.code,
            value: c.code ?? '',
            code: c.code,
            name: c.name,
          })),
          onChange: (val: string) => {
            // 手工汇率针对原币种填写，换币种后清空，改按新币种系统汇率。
            // 换币种后服务端本就重新解析系统汇率，「改回系统汇率」同时失去意义。
            form.setFieldValue([rowKey, 'exchangeRateOverride'], undefined);
            form.setFieldValue([rowKey, 'useSystemExchangeRate'], undefined);
            setRowRateOverride(rowKey, undefined);
            setRowPreviews((prev) =>
              prev[String(rowKey)]?.rateReset
                ? {
                    ...prev,
                    [String(rowKey)]: {
                      ...prev[String(rowKey)],
                      rateReset: undefined,
                    },
                  }
                : prev,
            );
            resolveRowRatePreview(rowKey, direction, val);
            refreshAmountPreview(rowKey, form);
          },
        }),
      },
      {
        title: '单价',
        dataIndex: 'unitPrice',
        width: 120,
        align: 'right',
        formItemProps: {
          ...inlineFieldError,
          rules: [
            { required: true, message: '请输入单价' },
            {
              validator: positiveDecimalRule(
                quantityOrPricePattern,
                '单价必须大于 0',
              ),
            },
          ],
        },
        render: (_, record) => trimDecimal(record.unitPrice),
        fieldProps: (form, { rowKey }) => ({
          placeholder: '0.00',
          allowClear: false,
          onFocus: (event: React.FocusEvent<HTMLInputElement>) => {
            event.currentTarget.select();
          },
          style: { textAlign: 'right' },
          styles: { input: { textAlign: 'right' } },
          onChange: () => {
            refreshAmountPreview(rowKey, form);
          },
        }),
      },
      {
        title: '数量',
        dataIndex: 'quantity',
        width: 190,
        align: 'right',
        formItemProps: (form, { rowKey }) => ({
          ...inlineFieldError,
          // 整数计费单位（箱、票等）不接受小数点：键入的 "." 直接不出现；
          // 粘贴的小数在小数点处截断（1.5 → 1），避免去点后放大成 15。
          normalize: (value?: string) =>
            value &&
            billingUnits?.find(
              (unit) =>
                unit.id === form.getFieldValue([rowKey, 'billingUnitId']),
            )?.quantityMustBeInteger
              ? value.split('.')[0].replace(/\D/g, '')
              : value,
          rules: [
            { required: true, message: '请输入数量' },
            {
              validator: positiveDecimalRule(
                quantityOrPricePattern,
                '数量必须大于 0',
              ),
            },
            {
              // 整数单位（箱、票等）随表单规则校验，避免手写错误被字段校验覆盖。
              validator: (_: unknown, value?: string) => {
                const error = feeQuantityRuleError(
                  value,
                  form.getFieldValue([rowKey, 'billingUnitId']),
                  billingUnits ?? [],
                  [
                    ...(latestReceivableResultRef.current?.items ?? []),
                    ...(latestPayableResultRef.current?.items ?? []),
                  ].find((item) => item.id === String(rowKey)),
                );
                return error
                  ? Promise.reject(new Error(error))
                  : Promise.resolve();
              },
            },
          ],
          // 无来源说明时不渲染 extra，避免空说明区撑高单元格使输入框偏上。
          extra: form.getFieldValue([
            rowKey,
            'quantitySource',
            'confirmationId',
          ]) ? (
            <Form.Item noStyle shouldUpdate>
              {() => (
                <FeeQuantitySourceFields
                  prefix={[rowKey || '']}
                  actionTabIndex={-1}
                  disabled={
                    feeWritesDisabled ||
                    [
                      ...(latestReceivableResultRef.current?.items ?? []),
                      ...(latestPayableResultRef.current?.items ?? []),
                    ].some(
                      (fee) =>
                        fee.id === String(rowKey) && fee.canEdit !== true,
                    )
                  }
                  onSourceChange={() => markDirty(direction)}
                />
              )}
            </Form.Item>
          ) : undefined,
        }),
        render: (_, record) => <FeeQuantityValue fee={record} />,
        fieldProps: (form, { rowKey }) => ({
          placeholder: '1',
          allowClear: false,
          onFocus: (event: React.FocusEvent<HTMLInputElement>) => {
            event.currentTarget.select();
          },
          style: { textAlign: 'right' },
          styles: { input: { textAlign: 'right' } },
          suffix: (
            <Form.Item noStyle shouldUpdate>
              {() => (
                <FeeQuantityAssist
                  tabIndex={-1}
                  allowChargeable={
                    order?.businessType ===
                      OrderBusinessType.BUSINESS_TYPE_AE ||
                    order?.businessType === OrderBusinessType.BUSINESS_TYPE_AI
                  }
                  orderId={orderId}
                  billingUnit={billingUnits?.find(
                    (unit) =>
                      unit.id === form.getFieldValue([rowKey, 'billingUnitId']),
                  )}
                  disabled={
                    feeWritesDisabled ||
                    [
                      ...(latestReceivableResultRef.current?.items ?? []),
                      ...(latestPayableResultRef.current?.items ?? []),
                    ].some(
                      (fee) =>
                        fee.id === String(rowKey) && fee.canEdit !== true,
                    )
                  }
                  onConfirm={(quantity, selection) => {
                    markDirty(direction);
                    form.setFieldValue(
                      [rowKey, 'quantity'],
                      trimDecimalInput(quantity),
                    );
                    form.setFieldValue(
                      [rowKey, 'quantitySource'],
                      selection || { mode: 'MANUAL' },
                    );
                    refreshAmountPreview(rowKey, form);
                    revalidateQuantity(form, rowKey);
                  }}
                />
              )}
            </Form.Item>
          ),

          onChange: () => {
            refreshAmountPreview(rowKey, form);
          },
        }),
      },
      {
        title: '计费单位',
        dataIndex: 'billingUnitId',
        width: 100,
        formItemProps: {
          rules: [{ required: true, message: '请选择计费单位' }],
        },
        render: (_, record) => record.billingUnit || '-',
        valueType: 'select',
        fieldProps: (form, { rowKey }) => ({
          showSearch: true,
          filterOption: defaultSelectFilterOption,
          placeholder: '计费单位',
          options: (billingUnits || []).map((u) => ({
            label: u.name,
            value: u.id ?? '',
            code: u.code,
          })),
          onChange: (_: string, option?: { label?: string }) => {
            form?.setFieldValue([rowKey, 'billingUnit'], option?.label);
            form?.setFieldValue([rowKey, 'quantitySource'], { mode: 'MANUAL' });
            revalidateQuantity(form, rowKey);
          },
        }),
      },
      {
        title: '总金额',
        dataIndex: 'totalAmount',
        width: 120,
        align: 'right',
        editable: false,
        shouldCellUpdate: () => true,
        render: (_, record) => {
          const preview = rowPreviews[String(record.id ?? '')]?.amount;
          if (
            rowPreviews[String(record.id ?? '')]?.unitPrice !== undefined &&
            !preview
          ) {
            return '-';
          }
          if (preview) {
            return (
              <span
                style={{
                  fontWeight: 600,
                  color: isReceivable ? '#1677ff' : '#fa8c16',
                }}
              >
                {`${formatAmount(preview.total)} ${preview.currency}`}
              </span>
            );
          }
          const calculated =
            record.unitPrice &&
            record.quantity &&
            !Number.isNaN(Number(record.unitPrice)) &&
            !Number.isNaN(Number(record.quantity))
              ? (Number(record.unitPrice) * Number(record.quantity)).toFixed(2)
              : undefined;
          const displayVal = record.totalAmount || calculated;
          return (
            <span
              style={{
                fontWeight: 600,
                color: isReceivable ? '#1677ff' : '#fa8c16',
              }}
            >
              {displayVal
                ? `${formatAmount(displayVal)} ${record.currency || 'CNY'}`
                : '-'}
            </span>
          );
        },
      },
      {
        title: '汇率',
        // 有权限时行内输入写入独立字段 exchangeRateOverride（列偏好经别名仍记为 exchangeRate），
        // 默认为空表示不覆盖；可编辑范围与税率列一致（新增行或未建账可编辑行）。
        dataIndex: canOverrideFeeExchangeRate
          ? 'exchangeRateOverride'
          : 'exchangeRate',
        // 宽度不随权限变化，有无改汇率权限的用户看到相同列宽与表格布局。
        width: 130,
        align: 'right',
        editable: canOverrideFeeExchangeRate
          ? (_, record) =>
              isNewFeeRowKey(record.id) ||
              (record.canEdit === true && record.hasActiveBill !== true)
          : false,
        formItemProps: canOverrideFeeExchangeRate
          ? {
              rules: [
                {
                  validator: (_: unknown, value?: string) => {
                    const text = normalizeDecimalInput(
                      String(value ?? '').trim(),
                    );
                    return text === '' || validExchangeRateOverride(text)
                      ? Promise.resolve()
                      : Promise.reject(
                          new Error(EXCHANGE_RATE_OVERRIDE_MESSAGE),
                        );
                  },
                },
              ],
            }
          : undefined,
        fieldProps: canOverrideFeeExchangeRate
          ? (form, config) => {
              // 可编辑单元格只传入 rowKey（行字段路径），按行 key 回查基线或新增草稿。
              const key = String([config.rowKey].flat()[0] ?? '');
              const entity =
                getDirectionItems(direction).find(
                  (item) => String(item.id) === key,
                ) ?? draftRowsRef.current[direction].get(key);
              const currency =
                form.getFieldValue([key, 'currency']) ?? entity?.currency;
              const baseCurrency = entity?.baseCurrency || 'CNY';
              const preview = rowPreviews[key]?.rate;
              const resetting = form.getFieldValue([
                key,
                'useSystemExchangeRate',
              ]);
              // 仅已存手工汇率且未换币种的行可改回系统汇率。
              const canReset =
                !isNewFeeRowKey(key) &&
                entity?.exchangeRateSource === 'MANUAL' &&
                currency === entity.currency &&
                currency !== baseCurrency;
              const placeholder = preview
                ? preview.status === 'resolved'
                  ? `${preview.rate ?? ''} 系统`
                  : preview.status === 'loading'
                    ? '获取中…'
                    : preview.status === 'missing'
                      ? '请先维护汇率'
                      : '汇率解析失败'
                : entity?.exchangeRate
                  ? `${trimDecimal(entity.exchangeRate)} ${
                      entity.exchangeRateSource === 'MANUAL' ? '手工' : '系统'
                    }`
                  : '系统汇率';
              return {
                // 本币汇率固定为 1，不提供手工覆盖。
                'aria-label': '手工汇率',
                disabled: currency === baseCurrency,
                placeholder: currency === baseCurrency ? '1' : placeholder,
                allowClear: false,
                style: { textAlign: 'right' },
                styles: { input: { textAlign: 'right' } },
                onFocus: (event: React.FocusEvent<HTMLInputElement>) => {
                  event.currentTarget.select();
                },
                onChange: (event: React.ChangeEvent<HTMLInputElement>) => {
                  // 填写手工汇率即放弃「改回系统汇率」。
                  if (resetting && event.target.value.trim() !== '') {
                    setRowRateReset(direction, form, key, false);
                  }
                  setRowRateOverride(key, event.target.value);
                },
                suffix: canReset ? (
                  <Tooltip
                    title={resetting ? '撤销改回系统汇率' : '改回系统汇率'}
                  >
                    <UndoOutlined
                      role="button"
                      aria-label={
                        resetting ? '撤销改回系统汇率' : '改回系统汇率'
                      }
                      aria-pressed={Boolean(resetting)}
                      style={{
                        color: resetting ? '#1677ff' : '#8c8c8c',
                        cursor: 'pointer',
                      }}
                      onClick={() =>
                        setRowRateReset(
                          direction,
                          form,
                          key,
                          !resetting,
                          currency,
                        )
                      }
                    />
                  </Tooltip>
                ) : undefined,
              };
            }
          : undefined,
        shouldCellUpdate: () => true,
        render: (_, record) => {
          const preview = rowPreviews[String(record.id ?? '')]?.rate;
          if (preview) {
            if (preview.status === 'loading') {
              return <span style={{ color: '#8c8c8c' }}>获取中…</span>;
            }
            if (preview.status === 'missing') {
              return <span style={{ color: '#cf1322' }}>请先维护汇率</span>;
            }
            if (preview.status === 'error') {
              return <span style={{ color: '#cf1322' }}>汇率解析失败</span>;
            }
            return (
              <Space size={4}>
                <span>{preview.rate}</span>
                <Tag color="processing">预览</Tag>
              </Space>
            );
          }
          return (
            <Space size={4}>
              <span>{trimDecimal(record.exchangeRate)}</span>
              {record.exchangeRateSource === 'MANUAL' && (
                <Tag color="gold">手工</Tag>
              )}
              {record.exchangeRateSource === 'SYSTEM' && (
                <Tag color="blue">系统</Tag>
              )}
            </Space>
          );
        },
      },
      {
        title: '录入日期',
        dataIndex: 'createdAt',
        width: 150,
        editable: false,
        render: (_, record) =>
          record.createdAt ? formatDate(record.createdAt, 'minute') : '-',
      },
      {
        title: '备注',
        dataIndex: 'note',
        width: 160,
        ellipsis: true,
        render: (_, record) => record.note || '-',
        fieldProps: {
          placeholder: '备注（可选）',
        },
      },
      ...buildOptionalFeeColumns(feeBillTracking, rowPreviews, {
        taxRateEditable: canEditFeeTaxRate,
        taxRateFieldProps: (form, { rowKey }) => ({
          placeholder: '默认取目录税率',
          allowClear: false,
          onFocus: (event: React.FocusEvent<HTMLInputElement>) => {
            event.currentTarget.select();
          },
          style: { textAlign: 'right' },
          styles: { input: { textAlign: 'right' } },
          onChange: () => refreshAmountPreview(rowKey, form),
        }),
      }),
    ];
  };

  const sideRefsOf = (direction: number) =>
    direction === RECEIVABLE
      ? {
          sequence: receivableRequestSequenceRef,
          latest: latestReceivableResultRef,
          setAll: setAllReceivableItems,
          setSummary: setReceivableSummary,
          setKeys: setReceivableEditableKeys,
        }
      : {
          sequence: payableRequestSequenceRef,
          latest: latestPayableResultRef,
          setAll: setAllPayableItems,
          setSummary: setPayableSummary,
          setKeys: setPayableEditableKeys,
        };

  /**
   * 重新加载后同步表单：服务端版本已变化的行以服务端值覆盖本地输入（基线随之更新，
   * 保存只会基于最新版本）；版本未变的行保留未保存输入。仅在本侧重载时执行，
   * 另一侧表单不受影响。
   */
  const syncRefreshedRows = (
    direction: number,
    previous: API.OrderFee[],
    next: API.OrderFee[],
  ) => {
    const form = formRefOf(direction).current;
    if (!form) return;
    const previousVersions = new Map(
      previous.map((item) => [item.id, item.version]),
    );
    const refreshed = next.filter(
      (item) =>
        item.id &&
        previousVersions.has(item.id) &&
        previousVersions.get(item.id) !== item.version,
    );
    if (refreshed.length === 0) return;
    form.setFieldsValue(
      Object.fromEntries(
        refreshed.map((item) => [
          item.id,
          {
            feeSettingId: item.feeSettingId,
            feeName: item.feeName,
            feeCode: item.feeCode,
            settlementPartyId: item.settlementPartyId,
            settlementPartyName: item.settlementPartyName,
            currency: item.currency,
            billingUnitId: item.billingUnitId,
            billingUnit: item.billingUnit,
            quantity: trimDecimalInput(item.quantity),
            unitPrice: trimDecimalInput(item.unitPrice),
            note: item.note ?? '',
            taxRate: item.taxRate ?? '',
            exchangeRateOverride: undefined,
            useSystemExchangeRate: undefined,
            quantitySource: item.quantitySource,
          },
        ]),
      ),
    );
    const refreshedIds = new Set(refreshed.map((item) => String(item.id)));
    setRateRequests((previous) => ({
      ...previous,
      rows: Object.fromEntries(
        Object.entries(previous.rows).filter(([key]) => !refreshedIds.has(key)),
      ),
    }));
    setRowPreviews((prev) => {
      const kept = Object.entries(prev).filter(
        ([key]) => !refreshedIds.has(key),
      );
      return kept.length === Object.keys(prev).length
        ? prev
        : Object.fromEntries(kept);
    });
  };

  /** 本侧列表请求：返回服务端有效费用并保留本侧未保存的新增行。 */
  const loadSide = async (direction: number) => {
    if (!orderId) return { data: [], success: true };
    const refs = sideRefsOf(direction);
    const requestedOrderId = orderId;
    const requestSequence = ++refs.sequence.current;
    const isCurrentRequest = () =>
      mountedRef.current &&
      currentOrderIdRef.current === requestedOrderId &&
      refs.sequence.current === requestSequence;
    const currentRows = () => {
      const result = refs.latest.current;
      return result?.orderId === currentOrderIdRef.current ? result.items : [];
    };
    const withDrafts = (items: API.OrderFee[]) => [
      ...draftRowsRef.current[direction].values(),
      ...items.map(toFeeFormRow),
    ];
    // 删除后仅重绘该侧已采纳快照，不读远端，保留其他费用的版本和输入。
    if (localReloadRef.current.delete(direction)) {
      await Promise.resolve();
      if (!isCurrentRequest())
        return { data: withDrafts(currentRows()), success: true };
      const items = currentRows();
      refs.setAll(items);
      refs.setSummary(summarizeFees(items));
      refs.setKeys([
        ...draftRowsRef.current[direction].keys(),
        ...items
          .filter((fee) => fee.id && fee.canEdit === true)
          .map((fee) => String(fee.id)),
      ]);
      return { data: withDrafts(items), success: true };
    }
    const restored = restoredDrafts[direction];
    if (restored && !loadedDraftSides.current.has(direction)) {
      loadedDraftSides.current.add(direction);
      draftRowsRef.current[direction] = new Map(restored.drafts);
      requestKeyHoldersRef.current[direction].restore(restored.request);
      refs.latest.current = { orderId, items: restored.baseline };
      refs.setAll(restored.baseline);
      refs.setSummary(summarizeFees(restored.baseline));
      refs.setKeys([
        ...restored.drafts.map(([key]) => key),
        ...restored.baseline
          .filter((row) => row.id && row.canEdit)
          .map((row) => String(row.id)),
      ]);
      markDirty(direction);
      return { data: withDrafts(restored.baseline), success: true };
    }
    try {
      const res = await orderFeeServiceListFees({ orderId });
      // 录入表只保留有效费用（费用无独立状态，服务端仅返回现存费用行）、
      // 最近请求结果与父级集合，笔数与金额消费同一有效集合。
      const items = unwrapList(res).filter(
        (fee) => feeDirectionCode(fee.direction) === direction,
      );
      if (!isCurrentRequest()) {
        return { data: withDrafts(currentRows()), success: true };
      }
      const previous = currentRows();
      refs.latest.current = { orderId: requestedOrderId, items };
      syncRefreshedRows(direction, previous, items);
      refs.setAll(items);
      refs.setSummary(summarizeFees(items));
      // 可编辑行常开编辑；新增行（草稿）保持编辑态。
      refs.setKeys([
        ...draftRowsRef.current[direction].keys(),
        ...items
          .filter((fee) => fee.id && fee.canEdit === true)
          .map((fee) => fee.id as string),
      ]);
      return { data: withDrafts(items), success: true };
    } catch {
      if (isCurrentRequest()) return { data: [], success: false };
      return { data: withDrafts(currentRows()), success: true };
    }
  };

  const renderRowErrors = (direction: number) => {
    const errors = rowErrors[sideName(direction)];
    if (errors.length === 0) return null;
    return (
      <Alert
        type="error"
        showIcon
        style={{ marginBottom: 8 }}
        title="本次保存失败，所有输入已保留"
        description={
          <div>
            {errors.map((item) => (
              <div key={`${item.clientRowId}:${item.field}`}>
                {rowLabelOf(direction, item.clientRowId)}
                {item.field && feeFieldLabels[item.field]
                  ? `（${feeFieldLabels[item.field]}）`
                  : ''}
                ：{item.message}
              </div>
            ))}
            {rowErrors.rateDetail ? <div>{rowErrors.rateDetail}</div> : null}
          </div>
        }
      />
    );
  };

  const renderFeeSection = (direction: number) => {
    const isReceivable = direction === RECEIVABLE;
    const side = sideName(direction);
    const summary = isReceivable ? receivableSummary : payableSummary;
    const selectedIds = isReceivable
      ? selectedReceivableFeeIds
      : selectedPayableFeeIds;
    const setSelectedIds = isReceivable
      ? setSelectedReceivableFeeIds
      : setSelectedPayableFeeIds;
    const actionRef = isReceivable ? receivableActionRef : payableActionRef;
    const editableKeys = isReceivable ? receivableEditable : payableEditable;
    const setKeys = isReceivable
      ? setReceivableEditableKeys
      : setPayableEditableKeys;
    const label = isReceivable ? '应收' : '应付';
    return (
      <SectionCard
        title={
          <Space size={8} align="center">
            <span>{`${label}费用`}</span>
            <Tag color="blue">{summary?.count ?? 0}</Tag>
          </Space>
        }
        extra={
          <Space size={8} wrap>
            {renderColumnToolbar()}
            {renderBulkToolbar(direction)}
            <Button
              danger
              icon={<DeleteOutlined aria-hidden />}
              disabled={
                feeWritesDisabled ||
                (!canDeleteFees &&
                  selectedIds.some(
                    (id) => !draftRowsRef.current[direction].has(String(id)),
                  )) ||
                selectedIds.length === 0 ||
                saving[side]
              }
              onClick={() =>
                setBulkActionContext({ direction, type: 'remove' })
              }
            >
              删除
            </Button>
            {canCreateFinanceBills && (
              <Button
                key="bill"
                icon={<FileDoneOutlined />}
                disabled={selectedIds.length === 0}
                onClick={() => handleOpenBill(direction)}
              >
                生成账单（{selectedIds.length}）
              </Button>
            )}
            <Tooltip
              title={
                canCreateFees && !canUpdateFees
                  ? '整表保存需要费用修改权限，当前账号不能新增或保存费用'
                  : undefined
              }
            >
              <Button
                key="add"
                icon={<PlusOutlined />}
                disabled={rowInputsLocked || !canCreateFees || saving[side]}
                onClick={() => handleAddRow(direction)}
              >
                新增{label}费用
              </Button>
            </Tooltip>
            <Button
              key="save"
              type="primary"
              icon={<SaveOutlined />}
              loading={saving[side]}
              disabled={rowWritesLocked || !dirty[side]}
              onClick={() => void handleSaveSide(direction)}
            >
              保存{label}费用
            </Button>
          </Space>
        }
      >
        {renderRowErrors(direction)}
        <EditableProTable<API.OrderFee>
          key={`${side}:${orderId}:${reloadToken}`}
          actionRef={actionRef}
          rowKey="id"
          search={false}
          params={{ orderId }}
          bordered
          size="small"
          cardProps={false}
          toolBarRender={false}
          pagination={false}
          scroll={{ x: 'max-content' }}
          recordCreatorProps={false}
          onRow={() => ({ onKeyDownCapture: handleFeeRowTab })}
          editable={{
            type: 'multiple',
            editableKeys,
            onChange: setKeys,
            formProps: {
              formRef: formRefOf(direction),
              disabled: saving[side],
              // EditableProTable.onValuesChange 内部防抖 64ms；用字段事件同步保护紧接着的组织切换。
              onFieldsChange: (fields) => {
                if (fields.some((field) => field.touched))
                  refreshDirty(direction);
              },
            },
            actionRender: () => [],
          }}
          rowSelection={{
            selectedRowKeys: selectedIds,
            onChange: setSelectedIds,
            getCheckboxProps: (record) => ({
              // 新增草稿也可勾选以从顶部移除；财务写入仍只接受真实费用版本。
              disabled:
                saving[side] ||
                !record.id ||
                (!draftRowsRef.current[direction].has(String(record.id)) &&
                  !record.version),
            }),
          }}
          tableAlertRender={false}
          request={() => loadSide(direction)}
          columns={
            getTableColumns
              ? buildColumns(direction)
              : orderColumnsByPreference(buildColumns(direction), feeColumnPref)
          }
        />
      </SectionCard>
    );
  };

  return (
    <>
      {renderFeeSection(RECEIVABLE)}
      {renderFeeSection(PAYABLE)}

      {/* 批量修改结算单位 */}
      <GuardedModalForm<{ settlementPartyId: string }>
        open={bulkActionContext?.type === 'settlementParty'}
        onOpenChange={(open) => !open && setBulkActionContext(undefined)}
        title={`批量修改结算单位（已选 ${
          (bulkActionContext?.direction === RECEIVABLE
            ? selectedReceivableFeeIds
            : selectedPayableFeeIds
          ).length
        } 笔${bulkActionContext?.direction === RECEIVABLE ? '应收' : '应付'}费用）`}
        width={480}
        modalProps={{ destroyOnHidden: true }}
        submitter={{
          searchConfig: { submitText: '确定修改', resetText: '取消' },
        }}
        onFinish={async (values) => {
          if (!bulkActionContext) return false;
          const { direction } = bulkActionContext;
          if (savingRef.current[sideName(direction)] || feeWritesDisabled)
            return false;
          const rows = getSelectedRows(direction);
          const selectedIds =
            direction === RECEIVABLE
              ? selectedReceivableFeeIds
              : selectedPayableFeeIds;
          if (rows.length !== selectedIds.length) {
            message.warning('请先保存新增费用，再执行财务操作');
            return false;
          }
          if (rows.length === 0) {
            message.warning('请先勾选要修改的费用');
            return false;
          }
          if (rejectUnavailableRows(rows, '批量修改结算单位')) return false;
          try {
            const response = await orderFeeServiceBulkUpdateOrderFees(
              { orderId },
              {
                orderId,
                targets: buildBulkTargets(rows),
                settlementPartyId: values.settlementPartyId,
              },
            );
            if (!isWriteResponse(response)) return false;
            message.success(
              `已修改 ${response.updatedCount ?? rows.length} 笔费用的结算单位`,
            );
            handleBulkDone(direction);
            setBulkActionContext(undefined);
            return true;
          } catch (error) {
            message.error(getErrorMessage(error, '批量修改结算单位失败'));
            return false;
          }
        }}
      >
        <Form.Item
          name="settlementPartyId"
          label="结算单位"
          rules={[{ required: true, message: '请选择结算单位' }]}
        >
          <PartnerQuickCreateSelect
            sessionIdentity={`${orderId}:bulk-settlement:${bulkActionContext?.direction}`}
            active={bulkActionContext?.type === 'settlementParty'}
            disabled={feeWritesDisabled}
            defaultRole={
              bulkActionContext?.direction === RECEIVABLE
                ? PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER
                : PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER
            }
            placeholder="请选择新的结算单位"
            options={(settlementParties || []).map((p) => ({
              label: p.name ?? '',
              value: p.id ?? '',
              code: p.code,
            }))}
          />
        </Form.Item>
      </GuardedModalForm>

      {/* 批量删除费用 */}
      <GuardedModalForm<{ reason?: string }>
        open={bulkActionContext?.type === 'remove'}
        onOpenChange={(open) => !open && setBulkActionContext(undefined)}
        title={`批量删除费用（已选 ${
          (bulkActionContext?.direction === RECEIVABLE
            ? selectedReceivableFeeIds
            : selectedPayableFeeIds
          ).length
        } 笔${bulkActionContext?.direction === RECEIVABLE ? '应收' : '应付'}费用）`}
        width={520}
        modalProps={{ destroyOnHidden: true }}
        submitter={{
          searchConfig: { submitText: '删除', resetText: '取消' },
          submitButtonProps: { danger: true },
        }}
        onFinish={async (values) => {
          if (!bulkActionContext) return false;
          const { direction } = bulkActionContext;
          const side = sideName(direction);
          if (savingRef.current[side] || feeWritesDisabled) return false;
          const selectedIds = (
            direction === RECEIVABLE
              ? selectedReceivableFeeIds
              : selectedPayableFeeIds
          ).map(String);
          const localIds = selectedIds.filter((id) =>
            draftRowsRef.current[direction].has(id),
          );
          const rows = getSelectedRows(direction);
          if (selectedIds.length === 0) {
            message.warning('请先勾选要删除的费用');
            return false;
          }
          if (localIds.length + rows.length !== selectedIds.length) {
            message.warning('勾选的费用已变化，请重新勾选后重试');
            return false;
          }
          if (rows.length > 0 && !canDeleteFees) return false;
          if (rejectUnavailableRows(rows, '批量删除', 'canDelete'))
            return false;
          const requestedOrderId = orderId;
          savingRef.current[side] = true;
          setSaving((prev) => ({ ...prev, [side]: true }));
          try {
            let removedCount = 0;
            if (rows.length > 0) {
              const response = await orderFeeServiceBulkRemoveOrderFees(
                { orderId },
                {
                  orderId,
                  targets: buildBulkTargets(rows),
                  reason: values.reason?.trim() || undefined,
                },
              );
              if (!isWriteResponse(response)) return false;
              removedCount = response.removedCount ?? rows.length;
            }
            if (currentOrderIdRef.current !== requestedOrderId) return true;
            // 远端删除成功后才移除本地草稿，失败时全部输入和选择保留。
            for (const id of localIds)
              draftRowsRef.current[direction].delete(id);
            const deletedIds = new Set(selectedIds);
            setRowErrors((prev) => ({
              ...prev,
              [side]: prev[side].filter(
                (error) => !deletedIds.has(error.clientRowId),
              ),
            }));
            const remainingChanged = getDirectionItems(direction).some(
              (row) =>
                row.id &&
                !deletedIds.has(row.id) &&
                row.canEdit === true &&
                feeRowChanged(
                  {
                    ...row,
                    ...formRefOf(direction).current?.getFieldValue([row.id]),
                  },
                  row,
                ),
            );
            if (draftRowsRef.current[direction].size === 0 && !remainingChanged)
              clearDirty(direction);
            (direction === RECEIVABLE
              ? setSelectedReceivableFeeIds
              : setSelectedPayableFeeIds)([]);
            message.success(`已删除 ${removedCount + localIds.length} 笔费用`);
            setBulkActionContext(undefined);
            // 删除只采纳明确被删除的行，其他行继续使用原版本与草稿基线。
            const refs = sideRefsOf(direction);
            refs.latest.current = {
              orderId,
              items: getDirectionItems(direction).filter(
                (row) => !deletedIds.has(String(row.id)),
              ),
            };
            localReloadRef.current.add(direction);
            await (direction === RECEIVABLE
              ? receivableActionRef
              : payableActionRef
            ).current?.reload();
            if (rows.length > 0) onFeeSaved?.();
            return true;
          } catch (error) {
            if (currentOrderIdRef.current === requestedOrderId)
              message.error(getErrorMessage(error, '批量删除费用失败'));
            return false;
          } finally {
            savingRef.current[side] = false;
            if (mountedRef.current)
              setSaving((prev) => ({ ...prev, [side]: false }));
          }
        }}
      >
        <Alert
          type="warning"
          showIcon
          title={`将删除 ${
            (bulkActionContext?.direction === RECEIVABLE
              ? selectedReceivableFeeIds
              : selectedPayableFeeIds
            ).length
          } 笔${bulkActionContext?.direction === RECEIVABLE ? '应收' : '应付'}费用`}
          description="删除为物理删除且整批原子执行，任一笔失败则全部回滚；已进入账单的费用需先取消对应账单。"
        />
        <ProFormTextArea
          name="reason"
          label="删除原因（选填）"
          placeholder="请输入删除原因，仅写入审计明细（最长 500 字）"
          fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
        />
      </GuardedModalForm>
    </>
  );
}
