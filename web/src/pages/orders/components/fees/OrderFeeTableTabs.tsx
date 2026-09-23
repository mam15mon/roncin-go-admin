import {
  ClockCircleOutlined,
  DeleteOutlined,
  DownOutlined,
  EditOutlined,
  FileDoneOutlined,
  PlusOutlined,
  SettingOutlined,
  SwapOutlined,
  TagsOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  EditableProTable,
  ModalForm,
  ProFormDatePicker,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Dropdown,
  type MenuProps,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
import {
  defaultSelectFilterOption,
  ProFormSearchableSelect,
  SectionCard,
  scrollToFirstTableError,
} from '@/components/ui';
import {
  normalizeOrderFeeStatus,
  orderFeeStatusMeta,
  statusTag,
} from '@/constants/statusMeta';
import { financeErrorReasons } from '@/errorReasons.generated';
import { history } from '@/router/history';
import {
  orderFeeServiceAddFee,
  orderFeeServiceBatchAssignOrderFeeTags,
  orderFeeServiceBatchRemoveOrderFeeTags,
  orderFeeServiceBulkRemoveOrderFees,
  orderFeeServiceBulkUpdateOrderFees,
  orderFeeServiceListFees,
  orderFeeServiceListOrderFeeTagOptions,
  orderFeeServiceResolveFeeExchangeRate,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { unwrapList } from '@/utils/api';
import {
  calculateExactFeeTotal,
  normalizeDecimalInput,
  quantityOrPricePattern,
} from '@/utils/decimal';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatDate, trimDecimal } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import FeeColumnSettingsModal from './FeeColumnSettingsModal';
import type { FeeBillTrackingView } from './feeBillTracking';
import {
  clearFeeColumnPreference,
  defaultFeeColumnPreference,
  type FeeColumnPreference,
  type FeeColumnPreferenceScope,
  isDefaultFeeColumnPreference,
  loadFeeColumnPreference,
  resolveFeeColumnPreference,
  saveFeeColumnPreference,
} from './feeColumnPreference';
import {
  FEE_BILLED,
  FEE_CANCELLED,
  FEE_UNBILLED,
  feeDirectionCode,
  feeStatusCode,
  PAYABLE,
  RECEIVABLE,
} from './feeConstants';
import {
  buildOptionalFeeColumns,
  orderColumnsByPreference,
} from './orderFeeOptionalColumns';

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
  inheritedLastWeek?: boolean;
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
};

type FeeRequestError = Error & {
  data?: { reason?: string };
  response?: { data?: { reason?: string } };
};

interface OrderFeeTableTabsProps {
  orderId: string;
  receivableActionRef: React.RefObject<ActionType | undefined>;
  payableActionRef: React.RefObject<ActionType | undefined>;
  receivableSummary: { totalAmount: number; count: number };
  payableSummary: { totalAmount: number; count: number };
  selectedReceivableFeeIds: React.Key[];
  setSelectedReceivableFeeIds: (keys: React.Key[]) => void;
  selectedPayableFeeIds: React.Key[];
  setSelectedPayableFeeIds: (keys: React.Key[]) => void;
  setAllReceivableItems: (items: API.OrderFee[]) => void;
  setAllPayableItems: (items: API.OrderFee[]) => void;
  setReceivableSummary: (summary: {
    totalAmount: number;
    count: number;
  }) => void;
  setPayableSummary: (summary: { totalAmount: number; count: number }) => void;
  canCreateFinanceBills: boolean;
  feeWritesDisabled: boolean;
  onOpenBillWorkbench: (feeIds: string[]) => void;
  onOpenFeeModal?: (direction: number, fee?: API.OrderFee) => void;
  getTableColumns?: (direction: number) => ProColumns<API.OrderFee>[];

  feeSettings?: API.OrderFeeSettingOption[];
  settlementParties?: { id?: string; name?: string; code?: string }[];
  currencies?: API.Currency[];
  billingUnits?: API.BillingUnit[];
  order?: API.Order;
  customerName?: string;
  onOpenQuickAddFee?: () => void;
  onOpenQuickAddPartner?: () => void;
  onCancelFee?: (fee: API.OrderFee) => void;
  /** 列偏好隔离范围（用户 + 当前组织）；缺省时设置仅当前页面会话内生效。 */
  columnSettingScope?: FeeColumnPreferenceScope;
  /** 关联账单投影；仅当页面具备财务费用读取权限时提供，两表共用一次查询。 */
  feeBillTracking?: FeeBillTrackingView;
  /** 行内保存成功后的通知（页面据此刷新关联账单投影）。 */
  onFeeSaved?: () => void;
}

export default function OrderFeeTableTabs({
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
  feeWritesDisabled,
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
  onCancelFee,
  columnSettingScope,
  feeBillTracking,
  onFeeSaved,
}: OrderFeeTableTabsProps) {
  const { message } = App.useApp();
  const currentOrderIdRef = useRef(orderId);
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

  // 列设置状态：应收/应付共享同一份偏好；编辑进行中禁用入口，避免丢失未保存行。
  const [columnSettingsOpen, setColumnSettingsOpen] = useState(false);
  const [feeColumnPref, setFeeColumnPref] = useState<FeeColumnPreference>(() =>
    defaultFeeColumnPreference(false),
  );
  const columnEditing =
    receivableEditableKeys.length > 0 || payableEditableKeys.length > 0;
  // 账单号与关联账单财务进度列仅在页面提供关联投影（财务读取权限）时可用。
  const financeAvailable = Boolean(feeBillTracking);

  useEffect(() => {
    setFeeColumnPref(
      resolveFeeColumnPreference(
        columnSettingScope ? loadFeeColumnPreference(columnSettingScope) : null,
        financeAvailable,
      ),
    );
  }, [
    columnSettingScope?.userId,
    columnSettingScope?.organizationId,
    financeAvailable,
  ]);

  const handleColumnSettingsConfirm = (next: FeeColumnPreference) => {
    setColumnSettingsOpen(false);
    setFeeColumnPref(next);
    const scope = columnSettingScope ?? {};
    const persisted = isDefaultFeeColumnPreference(next, financeAvailable)
      ? clearFeeColumnPreference(scope)
      : saveFeeColumnPreference(scope, next);
    if (!persisted) {
      message.warning('列设置未能保存到本地浏览器，本次设置仅当前页面生效');
    }
  };

  const renderColumnToolbar = () => (
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
      <Tooltip
        title={
          columnEditing
            ? '请先保存或取消正在编辑的费用行'
            : '设置费用表格列（应收/应付共用）'
        }
      >
        <span>
          <Button
            icon={<SettingOutlined />}
            disabled={columnEditing}
            onClick={() => setColumnSettingsOpen(true)}
          >
            列设置
          </Button>
        </span>
      </Tooltip>
    </Space>
  );

  // ===== 批量维护（阶段二）：应收/应付两表各自一套入口，作用于该表勾选行 =====

  // 标签弹窗上下文：记录触发表格方向与入口模式（添加/移除标签）。
  const [tagModalContext, setTagModalContext] = useState<
    { direction: number; mode: 'assign' | 'remove' } | undefined
  >(undefined);

  /** 按方向取该表最近一次成功加载的有效费用行（版本号快照来源）。 */
  const getDirectionItems = (direction: number): API.OrderFee[] => {
    const result =
      direction === RECEIVABLE
        ? latestReceivableResultRef.current
        : latestPayableResultRef.current;
    return result?.orderId === orderId ? result.items : [];
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

  /** 已建账行不参与改结算单位/改时间/删除/再次建账；显式报错，禁止静默过滤。 */
  const rejectBilledRows = (rows: API.OrderFee[], actionText: string) => {
    const billedRows = rows.filter(
      (row) => feeStatusCode(row.status) !== FEE_UNBILLED,
    );
    if (billedRows.length === 0) return false;
    const names = billedRows
      .map((row) => row.feeName || row.feeCode || row.id || '')
      .filter(Boolean);
    const brief =
      names.slice(0, 3).join('、') + (names.length > 3 ? ' 等' : '');
    message.warning(
      `选中费用中有 ${billedRows.length} 笔已建账，不能${actionText}（${brief}），请取消勾选后重试`,
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
    const rows = getSelectedRows(direction);
    if (rows.length === 0) {
      message.warning('未找到勾选的费用，请刷新后重试');
      return;
    }
    if (rejectBilledRows(rows, '生成账单')) return;
    onOpenBillWorkbench(rows.map((row) => String(row.id)));
  };

  type BulkActionType = 'settlementParty' | 'expenseDate' | 'remove';
  const [bulkActionContext, setBulkActionContext] = useState<
    { direction: number; type: BulkActionType } | undefined
  >(undefined);

  const renderBulkToolbar = (direction: number) => {
    const isReceivable = direction === RECEIVABLE;
    const selectedIds = isReceivable
      ? selectedReceivableFeeIds
      : selectedPayableFeeIds;
    const bulkDisabled = feeWritesDisabled || selectedIds.length === 0;

    const items: MenuProps['items'] = [
      {
        key: 'settlementParty',
        icon: <SwapOutlined />,
        label: '批量改结算单位',
        onClick: () =>
          setBulkActionContext({ direction, type: 'settlementParty' }),
      },
      {
        key: 'expenseDate',
        icon: <ClockCircleOutlined />,
        label: '批量改费用时间',
        onClick: () => setBulkActionContext({ direction, type: 'expenseDate' }),
      },
      {
        key: 'addTags',
        icon: <TagsOutlined />,
        label: '添加标签',
        onClick: () => setTagModalContext({ direction, mode: 'assign' }),
      },
      {
        key: 'removeTags',
        icon: <TagsOutlined />,
        label: '删除标签',
        onClick: () => setTagModalContext({ direction, mode: 'remove' }),
      },
      {
        type: 'divider',
      },
      {
        key: 'remove',
        icon: <DeleteOutlined />,
        danger: true,
        label: '批量删除',
        onClick: () => setBulkActionContext({ direction, type: 'remove' }),
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
  // 币种/发生日期/单价/数量/费用项目变化即刷新，保存后以后端落库值为准。
  const [rowPreviews, setRowPreviews] = useState<
    Record<string, RowEditPreview>
  >({});
  const rateRequestSeqRef = useRef<Record<string, number>>({});

  currentOrderIdRef.current = orderId;

  useEffect(() => {
    mountedRef.current = true;
    setReceivableEditableKeys([]);
    setPayableEditableKeys([]);
    setRowPreviews({});
    rateRequestSeqRef.current = {};
    return () => {
      mountedRef.current = false;
      receivableRequestSequenceRef.current += 1;
      payableRequestSequenceRef.current += 1;
    };
  }, []);

  useEffect(() => {
    setRowPreviews((prev) => {
      const activeKeys = new Set(
        [...receivableEditableKeys, ...payableEditableKeys].map(String),
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
  }, [receivableEditableKeys, payableEditableKeys]);

  useEffect(() => {
    setRowPreviews({});
    rateRequestSeqRef.current = {};
  }, [orderId]);

  // 默认计费单位：优先查找“票”（PIAO / 包含票），否则选用首项
  const defaultBillingUnit =
    billingUnits?.find(
      (u) => u.code === 'PIAO' || u.name === '票' || u.name?.includes('票'),
    ) || billingUnits?.[0];

  // 币种或发生日期变化时立即解析参考汇率，行内预览；保存后以后端落库值为准。
  const resolveRowRatePreview = (
    rowKey: React.Key | undefined,
    direction: number,
    currency?: string,
    expenseDate?: string | dayjs.Dayjs | null,
  ) => {
    const key = String(rowKey ?? '');
    const requestSequence = (rateRequestSeqRef.current[key] ?? 0) + 1;
    rateRequestSeqRef.current[key] = requestSequence;
    const patchRate = (rate: ExchangeRatePreview | undefined) => {
      setRowPreviews((prev) => {
        const current = prev[key];
        if (!current) return rate ? { ...prev, [key]: { rate } } : prev;
        const next = { ...current, rate };
        if (
          !next.rate &&
          !next.amount &&
          next.feeCode === undefined &&
          next.taxRate === undefined &&
          next.unitPrice === undefined
        ) {
          const { [key]: _removed, ...rest } = prev;
          return rest;
        }
        return { ...prev, [key]: next };
      });
    };
    if (!currency || !expenseDate) {
      patchRate(undefined);
      return;
    }
    patchRate({ status: 'loading' });
    orderFeeServiceResolveFeeExchangeRate(
      {
        orderId,
        direction,
        currency,
        expenseDate: dayjs(expenseDate).format('YYYY-MM-DD'),
      },
      { skipErrorHandler: true },
    )
      .then((response) => {
        if (rateRequestSeqRef.current[key] !== requestSequence) return;
        if (response.exchangeRate) {
          patchRate({
            status: 'resolved',
            rate: trimDecimal(response.exchangeRate),
            inheritedLastWeek:
              response.exchangeRateSource === 'INHERITED_LAST_WEEK',
          });
        } else {
          patchRate({ status: 'error' });
          message.error('汇率解析结果不完整');
        }
      })
      .catch((error: FeeRequestError) => {
        if (rateRequestSeqRef.current[key] !== requestSequence) return;
        const reason = error.data?.reason ?? error.response?.data?.reason;
        if (reason === financeErrorReasons.FEE_EXCHANGE_RATE_MISSING) {
          patchRate({ status: 'missing' });
          return;
        }
        patchRate({ status: 'error' });
        message.error(error.message || '汇率解析失败');
      });
  };

  // 单价/数量/币种变化后重算总金额预览，并同步行内单价供不含税单价列
  // 实时折算；输入不完整时清除该行对应预览。
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
    setRowPreviews((prev) => {
      const current = prev[key];
      if (!total && !hasUnitPrice) {
        if (!current?.amount && current?.unitPrice === undefined) return prev;
        const next = { ...current };
        delete next.amount;
        delete next.unitPrice;
        if (
          !next.rate &&
          next.feeCode === undefined &&
          next.taxRate === undefined &&
          next.unitPrice === undefined
        ) {
          const { [key]: _removed, ...rest } = prev;
          return rest;
        }
        return { ...prev, [key]: next };
      }
      const next: RowEditPreview = { ...current };
      if (hasUnitPrice) {
        next.unitPrice = unitPriceInput;
      } else {
        delete next.unitPrice;
      }
      if (total) {
        next.amount = { total, currency: String(currency || 'CNY') };
      } else {
        delete next.amount;
      }
      return { ...prev, [key]: next };
    });
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

  const handleAddReceivable = () => {
    if (feeWritesDisabled) return;
    const newId = `new_${Date.now()}`;
    const defaultPartyId = order?.customerId;
    const defaultPartyName =
      customerName ||
      settlementParties?.find((p) => p.id === defaultPartyId)?.name;

    receivableActionRef.current?.addEditRecord?.(
      {
        id: newId,
        direction: RECEIVABLE,
        currency: 'CNY',
        quantity: '1',
        billingUnitId: defaultBillingUnit?.id,
        billingUnit: defaultBillingUnit?.name,
        settlementPartyId: defaultPartyId,
        settlementPartyName: defaultPartyName,
        expenseDate: dayjs().format('YYYY-MM-DD HH:mm'),
        status: FEE_UNBILLED,
      },
      { position: 'top' },
    );
    resolveRowRatePreview(newId, RECEIVABLE, 'CNY', dayjs());
  };

  const handleAddPayable = () => {
    if (feeWritesDisabled) return;
    const newId = `new_${Date.now()}`;
    const defaultPartyId = order?.bookingAgentId;
    const defaultPartyName = settlementParties?.find(
      (p) => p.id === defaultPartyId,
    )?.name;

    payableActionRef.current?.addEditRecord?.(
      {
        id: newId,
        direction: PAYABLE,
        currency: 'CNY',
        quantity: '1',
        billingUnitId: defaultBillingUnit?.id,
        billingUnit: defaultBillingUnit?.name,
        settlementPartyId: defaultPartyId,
        settlementPartyName: defaultPartyName,
        expenseDate: dayjs().format('YYYY-MM-DD HH:mm'),
        status: FEE_UNBILLED,
      },
      { position: 'top' },
    );
    resolveRowRatePreview(newId, PAYABLE, 'CNY', dayjs());
  };

  const handleSaveFee = async (
    key: React.Key | React.Key[],
    row: API.OrderFee,
    originRow: API.OrderFee,
    newLine?: boolean,
  ) => {
    if (!orderId) return false;
    const singleKey = Array.isArray(key) ? key[0] : key;
    const isNew =
      newLine || String(singleKey).startsWith('new_') || !originRow?.version;
    const direction = row.direction ?? RECEIVABLE;

    if (!row.feeSettingId) {
      message.error('请选择费用项目');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'feeSettingId'],
            errors: ['请选择费用项目'],
          },
        ],
      });
      return false;
    }
    if (!row.settlementPartyId) {
      message.error('请选择结算单位');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'settlementPartyId'],
            errors: ['请选择结算单位'],
          },
        ],
      });
      return false;
    }
    if (!row.currency) {
      message.error('请选择币种');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'currency'],
            errors: ['请选择币种'],
          },
        ],
      });
      return false;
    }
    if (
      !row.unitPrice ||
      !quantityOrPricePattern.test(
        normalizeDecimalInput(String(row.unitPrice)),
      ) ||
      Number(row.unitPrice) <= 0
    ) {
      message.error('单价必须为大于 0 的有效数值');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'unitPrice'],
            errors: ['单价必须为大于 0 的有效数值'],
          },
        ],
      });
      return false;
    }
    if (
      !row.quantity ||
      !quantityOrPricePattern.test(
        normalizeDecimalInput(String(row.quantity)),
      ) ||
      Number(row.quantity) <= 0
    ) {
      message.error('数量必须为大于 0 的有效数值');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'quantity'],
            errors: ['数量必须为大于 0 的有效数值'],
          },
        ],
      });
      return false;
    }
    if (!row.billingUnitId) {
      message.error('请选择计费单位');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'billingUnitId'],
            errors: ['请选择计费单位'],
          },
        ],
      });
      return false;
    }
    if (!row.expenseDate) {
      message.error('请选择发生日期');
      scrollToFirstTableError({
        rowKey: singleKey,
        errorFields: [
          {
            name: [String(singleKey), 'expenseDate'],
            errors: ['请选择发生日期'],
          },
        ],
      });
      return false;
    }

    const body = {
      direction,
      feeSettingId: row.feeSettingId,
      settlementPartyId: row.settlementPartyId,
      billingUnitId: row.billingUnitId,
      // 提交前去掉小数尾部零，与后端固定位数十进制校验口径对齐。
      quantity: normalizeDecimalInput(String(row.quantity)),
      unitPrice: normalizeDecimalInput(String(row.unitPrice)),
      currency: row.currency,
      expenseDate: dayjs(row.expenseDate).format('YYYY-MM-DD HH:mm'),
      note: row.note?.trim() || undefined,
      taxInclusive: true,
    };

    try {
      if (isNew) {
        await orderFeeServiceAddFee(
          { orderId },
          { ...body, orderId, idempotencyKey: generateUUID() },
        );
        message.success('费用录入成功');
      } else {
        if (!originRow.version) {
          message.error('费用版本信息缺失，请刷新后重试');
          return false;
        }
        await orderFeeServiceUpdateFee(
          { orderId, id: String(singleKey) },
          {
            ...body,
            orderId,
            id: String(singleKey),
            expectedVersion: originRow.version,
          },
        );
        message.success('费用更新成功');
      }
      receivableActionRef.current?.reload();
      payableActionRef.current?.reload();
      onFeeSaved?.();
      return true;
    } catch (error) {
      message.error(
        getErrorMessage(error, isNew ? '录入费用失败' : '更新费用失败'),
      );
      return false;
    }
  };

  // 渲染行内编辑操作列：为保存操作接入错误捕获与平滑导航
  const renderEditableActions = (
    rowKey: React.Key,
    defaultDom: { save: React.ReactNode; cancel: React.ReactNode },
  ) => [
    <span
      key={`save-action-${String(rowKey)}`}
      onClickCapture={() => {
        // 等待 Ant Design ProTable 表单校验执行完毕并完成 DOM 错误渲染
        window.setTimeout(() => {
          scrollToFirstTableError({
            rowKey,
            notify: (msg) => message.warning(msg),
          });
        }, 60);
      }}
    >
      {defaultDom.save}
    </span>,
    defaultDom.cancel,
  ];

  const buildColumns = (direction: number): ProColumns<API.OrderFee>[] => {
    if (getTableColumns) {
      return getTableColumns(direction);
    }
    const isReceivable = direction === RECEIVABLE;

    return [
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        editable: false,
        render: (_, record) =>
          statusTag(orderFeeStatusMeta, normalizeOrderFeeStatus(record.status)),
      },
      {
        title: '费用代码',
        dataIndex: 'feeCode',
        width: 100,
        editable: false,
        // 行内编辑预览依赖外部状态，单元格需随组件重渲染刷新。
        shouldCellUpdate: () => true,
        render: (_, record) =>
          rowPreviews[String(record.id ?? '')]?.feeCode ??
          record.feeCode ??
          '-',
      },
      {
        title: '费用名称',
        dataIndex: 'feeSettingId',
        width: 180,
        formItemProps: {
          rules: [{ required: true, message: '请选择费用项目' }],
        },
        render: (_, record) => record.feeName || '-',
        valueType: 'select',
        fieldProps: (form, { rowKey }) => ({
          showSearch: true,
          placeholder: '请选择费用项目',
          options: (feeSettings || []).map((item) => ({
            label: `${item.nameZh || item.nameEn || item.feeCode} (${item.feeCode})`,
            value: item.id ?? '',
            nameZh: item.nameZh,
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
                  form?.getFieldValue([rowKey, 'expenseDate']),
                );
                refreshAmountPreview(rowKey, form);
              }
              if (option.defaultBillingUnitId) {
                const bu = billingUnits?.find(
                  (u) => u.id === option.defaultBillingUnitId,
                );
                form?.setFieldValue(
                  [rowKey, 'billingUnitId'],
                  option.defaultBillingUnitId,
                );
                if (bu) {
                  form?.setFieldValue([rowKey, 'billingUnit'], bu.name);
                }
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
                    onOpenQuickAddPartner();
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
            resolveRowRatePreview(
              rowKey,
              direction,
              val,
              form?.getFieldValue([rowKey, 'expenseDate']),
            );
            refreshAmountPreview(rowKey, form);
          },
        }),
      },
      {
        title: '单价',
        dataIndex: 'unitPrice',
        width: 110,
        align: 'right',
        formItemProps: {
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
        width: 90,
        align: 'right',
        formItemProps: {
          rules: [
            { required: true, message: '请输入数量' },
            {
              validator: positiveDecimalRule(
                quantityOrPricePattern,
                '数量必须大于 0',
              ),
            },
          ],
        },
        render: (_, record) => trimDecimal(record.quantity),
        fieldProps: (form, { rowKey }) => ({
          placeholder: '1',
          allowClear: false,
          style: { textAlign: 'right' },
          styles: { input: { textAlign: 'right' } },
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
          if (preview) {
            return (
              <span
                style={{
                  fontWeight: 600,
                  color: isReceivable ? '#1677ff' : '#fa8c16',
                }}
              >
                {`${preview.total} ${preview.currency}`}
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
                ? `${trimDecimal(displayVal)} ${record.currency || 'CNY'}`
                : '-'}
            </span>
          );
        },
      },
      {
        title: '汇率',
        dataIndex: 'exchangeRate',
        width: 120,
        align: 'right',
        editable: false,
        shouldCellUpdate: () => true,
        render: (_, record) => {
          const preview = rowPreviews[String(record.id ?? '')]?.rate;
          if (preview) {
            if (preview.status === 'loading') {
              return <span style={{ color: '#8c8c8c' }}>获取中…</span>;
            }
            if (preview.status === 'missing') {
              return <span style={{ color: '#cf1322' }}>汇率缺失</span>;
            }
            if (preview.status === 'error') {
              return <span style={{ color: '#cf1322' }}>汇率解析失败</span>;
            }
            return (
              <Space size={4}>
                <span>{preview.rate}</span>
                <Tag color="processing">预览</Tag>
                {preview.inheritedLastWeek && <Tag color="gold">沿用上周</Tag>}
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
        title: '发生日期',
        dataIndex: 'expenseDate',
        width: 150,
        valueType: 'date',
        formItemProps: {
          rules: [{ required: true, message: '请选择发生日期' }],
        },
        render: (_, record) =>
          record.expenseDate
            ? formatDate(
                record.expenseDate,
                record.expenseDate.includes(' ') ? 'minute' : 'date',
              )
            : '-',
        fieldProps: (form, { rowKey }) => ({
          style: { width: '100%' },
          format: 'YYYY-MM-DD HH:mm',
          showTime: { format: 'HH:mm' },
          onChange: (date: dayjs.Dayjs | null) => {
            resolveRowRatePreview(
              rowKey,
              direction,
              form?.getFieldValue([rowKey, 'currency']),
              date,
            );
          },
        }),
      },
      {
        title: '备注',
        dataIndex: 'note',
        width: 140,
        ellipsis: true,
        render: (_, record) => record.note || '-',
        fieldProps: {
          placeholder: '备注（可选）',
        },
      },
      ...buildOptionalFeeColumns(feeBillTracking, rowPreviews),
      {
        title: '操作',
        valueType: 'option',
        width: 120,
        fixed: 'right',
        render: (_, record, __, action) => {
          if (feeWritesDisabled) return [];
          return [
            (feeStatusCode(record.status) === FEE_UNBILLED ||
              feeStatusCode(record.status) === FEE_BILLED) && (
              <Button
                key="edit"
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => {
                  if (record.id) {
                    action?.startEditable(record.id);
                  }
                }}
              >
                编辑
              </Button>
            ),
            feeStatusCode(record.status) === FEE_UNBILLED && onCancelFee && (
              <Button
                key="cancel"
                type="link"
                size="small"
                danger
                onClick={() => onCancelFee(record)}
              >
                删除
              </Button>
            ),
          ].filter(Boolean);
        },
      },
    ];
  };

  // 标签弹窗打开时解析目标行快照（含已建账行；标签维护不受账单占用限制）。
  const tagModalRows = tagModalContext
    ? getSelectedRows(tagModalContext.direction)
    : [];

  return (
    <>
      <SectionCard
        title={
          <Space size={8} align="center">
            <span>应收费用</span>
            <Tag color="blue">{receivableSummary.count}</Tag>
          </Space>
        }
        extra={
          <Space size={8} wrap>
            {renderColumnToolbar()}
            {renderBulkToolbar(RECEIVABLE)}
            {canCreateFinanceBills && (
              <Button
                key="bill"
                icon={<FileDoneOutlined />}
                disabled={selectedReceivableFeeIds.length === 0}
                onClick={() => handleOpenBill(RECEIVABLE)}
              >
                生成账单（{selectedReceivableFeeIds.length}）
              </Button>
            )}
            <Button
              key="add"
              type="primary"
              icon={<PlusOutlined />}
              disabled={feeWritesDisabled}
              onClick={handleAddReceivable}
            >
              新增应收费用
            </Button>
          </Space>
        }
      >
        <EditableProTable<API.OrderFee>
          key={`receivable:${orderId}`}
          actionRef={receivableActionRef}
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
          editable={{
            type: 'single',
            editableKeys: receivableEditableKeys,
            onChange: setReceivableEditableKeys,
            onSave: (key, row, originRow, newLine) =>
              handleSaveFee(key, row, originRow, Boolean(newLine)),
            actionRender: (row, _config, defaultDom) =>
              renderEditableActions(row.id ?? '', defaultDom),
          }}
          rowSelection={{
            selectedRowKeys: selectedReceivableFeeIds,
            onChange: setSelectedReceivableFeeIds,
            getCheckboxProps: (record) => ({
              // 所有已保存有效费用（含已建账）均可勾选；行内新增或
              // 缺少版本的未保存行不参与选择，避免提交不存在的事实。
              disabled:
                !record.id ||
                String(record.id).startsWith('new_') ||
                !record.version,
            }),
          }}
          tableAlertRender={({ selectedRowKeys }) =>
            `已选择 ${selectedRowKeys.length} 笔有效应收费用`
          }
          request={async () => {
            if (!orderId) return { data: [], success: true };
            const requestedOrderId = orderId;
            const requestSequence = ++receivableRequestSequenceRef.current;
            try {
              const res = await orderFeeServiceListFees({ orderId });
              // 录入表只保留有效费用：已作废（历史软删除）行不进入表格、
              // 最近请求结果与父级集合，笔数与金额消费同一有效集合。
              const rItems = unwrapList(res).filter(
                (f) =>
                  feeDirectionCode(f.direction) === RECEIVABLE &&
                  feeStatusCode(f.status) !== FEE_CANCELLED,
              );
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                receivableRequestSequenceRef.current === requestSequence;
              if (!isCurrentRequest) {
                const currentResult = latestReceivableResultRef.current;
                return {
                  data:
                    currentResult?.orderId === currentOrderIdRef.current
                      ? currentResult.items
                      : [],
                  success: true,
                };
              }
              latestReceivableResultRef.current = {
                orderId: requestedOrderId,
                items: rItems,
              };
              setAllReceivableItems(rItems);
              const total = rItems.reduce(
                (acc, cur) =>
                  acc +
                  (cur.baseCurrencyAmount ? Number(cur.baseCurrencyAmount) : 0),
                0,
              );
              setReceivableSummary({
                totalAmount: total,
                count: rItems.length,
              });
              return { data: rItems, success: true };
            } catch {
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                receivableRequestSequenceRef.current === requestSequence;
              if (isCurrentRequest) {
                return { data: [], success: false };
              }
              const currentResult = latestReceivableResultRef.current;
              return {
                data:
                  currentResult?.orderId === currentOrderIdRef.current
                    ? currentResult.items
                    : [],
                success: true,
              };
            }
          }}
          columns={
            getTableColumns
              ? buildColumns(RECEIVABLE)
              : orderColumnsByPreference(
                  buildColumns(RECEIVABLE),
                  feeColumnPref,
                )
          }
        />
      </SectionCard>

      <SectionCard
        title={
          <Space size={8} align="center">
            <span>应付费用</span>
            <Tag color="blue">{payableSummary.count}</Tag>
          </Space>
        }
        extra={
          <Space size={8} wrap>
            {renderColumnToolbar()}
            {renderBulkToolbar(PAYABLE)}
            {canCreateFinanceBills && (
              <Button
                key="bill"
                icon={<FileDoneOutlined />}
                disabled={selectedPayableFeeIds.length === 0}
                onClick={() => handleOpenBill(PAYABLE)}
              >
                生成账单（{selectedPayableFeeIds.length}）
              </Button>
            )}
            <Button
              key="add"
              type="primary"
              icon={<PlusOutlined />}
              disabled={feeWritesDisabled}
              onClick={handleAddPayable}
            >
              新增应付费用
            </Button>
          </Space>
        }
      >
        <EditableProTable<API.OrderFee>
          key={`payable:${orderId}`}
          actionRef={payableActionRef}
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
          editable={{
            type: 'single',
            editableKeys: payableEditableKeys,
            onChange: setPayableEditableKeys,
            onSave: (key, row, originRow, newLine) =>
              handleSaveFee(key, row, originRow, Boolean(newLine)),
            actionRender: (row, _config, defaultDom) =>
              renderEditableActions(row.id ?? '', defaultDom),
          }}
          rowSelection={{
            selectedRowKeys: selectedPayableFeeIds,
            onChange: setSelectedPayableFeeIds,
            getCheckboxProps: (record) => ({
              // 所有已保存有效费用（含已建账）均可勾选；行内新增或
              // 缺少版本的未保存行不参与选择，避免提交不存在的事实。
              disabled:
                !record.id ||
                String(record.id).startsWith('new_') ||
                !record.version,
            }),
          }}
          tableAlertRender={({ selectedRowKeys }) =>
            `已选择 ${selectedRowKeys.length} 笔有效应付费用`
          }
          request={async () => {
            if (!orderId) return { data: [], success: true };
            const requestedOrderId = orderId;
            const requestSequence = ++payableRequestSequenceRef.current;
            try {
              const res = await orderFeeServiceListFees({ orderId });
              // 录入表只保留有效费用：已作废（历史软删除）行不进入表格、
              // 最近请求结果与父级集合，笔数与金额消费同一有效集合。
              const pItems = unwrapList(res).filter(
                (f) =>
                  feeDirectionCode(f.direction) === PAYABLE &&
                  feeStatusCode(f.status) !== FEE_CANCELLED,
              );
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                payableRequestSequenceRef.current === requestSequence;
              if (!isCurrentRequest) {
                const currentResult = latestPayableResultRef.current;
                return {
                  data:
                    currentResult?.orderId === currentOrderIdRef.current
                      ? currentResult.items
                      : [],
                  success: true,
                };
              }
              latestPayableResultRef.current = {
                orderId: requestedOrderId,
                items: pItems,
              };
              setAllPayableItems(pItems);
              const total = pItems.reduce(
                (acc, cur) =>
                  acc +
                  (cur.baseCurrencyAmount ? Number(cur.baseCurrencyAmount) : 0),
                0,
              );
              setPayableSummary({
                totalAmount: total,
                count: pItems.length,
              });
              return { data: pItems, success: true };
            } catch {
              const isCurrentRequest =
                mountedRef.current &&
                currentOrderIdRef.current === requestedOrderId &&
                payableRequestSequenceRef.current === requestSequence;
              if (isCurrentRequest) {
                return { data: [], success: false };
              }
              const currentResult = latestPayableResultRef.current;
              return {
                data:
                  currentResult?.orderId === currentOrderIdRef.current
                    ? currentResult.items
                    : [],
                success: true,
              };
            }
          }}
          columns={
            getTableColumns
              ? buildColumns(PAYABLE)
              : orderColumnsByPreference(buildColumns(PAYABLE), feeColumnPref)
          }
        />
      </SectionCard>

      <FeeColumnSettingsModal
        open={columnSettingsOpen}
        financeAvailable={financeAvailable}
        value={feeColumnPref}
        onCancel={() => setColumnSettingsOpen(false)}
        onConfirm={handleColumnSettingsConfirm}
      />

      {/* 批量标签维护：两个表共用一个弹窗，打开时按入口锁定添加/移除模式。 */}
      <BusinessTagModal
        open={Boolean(tagModalContext)}
        defaultMode={tagModalContext?.mode}
        targetCount={tagModalRows.length}
        existingTags={tagModalRows.flatMap((row) => row.tags ?? [])}
        canQuickCreate={false}
        loadOptions={(params) =>
          orderFeeServiceListOrderFeeTagOptions({ orderId, ...params })
        }
        onSubmit={async (mode, tagIds) => {
          const direction = tagModalContext?.direction ?? RECEIVABLE;
          const rows = getSelectedRows(direction);
          const feeIds = rows.map((row) => String(row.id));
          if (feeIds.length === 0 || tagIds.length === 0) return;
          try {
            if (mode === 'assign') {
              await orderFeeServiceBatchAssignOrderFeeTags(
                { orderId },
                { orderId, feeIds, tagIds },
              );
              message.success(`已为 ${feeIds.length} 笔费用添加标签`);
            } else {
              await orderFeeServiceBatchRemoveOrderFeeTags(
                { orderId },
                { orderId, feeIds, tagIds },
              );
              message.success(`已从 ${feeIds.length} 笔费用移除标签`);
            }
            handleBulkDone(direction);
          } catch (error) {
            message.error(getErrorMessage(error, '维护费用标签失败'));
          }
        }}
        onCancel={() => setTagModalContext(undefined)}
      />

      {/* 批量修改结算单位 */}
      <ModalForm<{ settlementPartyId: string }>
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
          const rows = getSelectedRows(direction);
          if (rows.length === 0) {
            message.warning('请先勾选要修改的费用');
            return false;
          }
          if (rejectBilledRows(rows, '批量修改结算单位')) return false;
          try {
            const response = await orderFeeServiceBulkUpdateOrderFees(
              { orderId },
              {
                orderId,
                targets: buildBulkTargets(rows),
                settlementPartyId: values.settlementPartyId,
              },
            );
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
        <ProFormSearchableSelect
          name="settlementPartyId"
          label="结算单位"
          rules={[{ required: true, message: '请选择结算单位' }]}
          placeholder="请选择新的结算单位"
          options={(settlementParties || []).map((p) => ({
            label: p.name ?? '',
            value: p.id ?? '',
            code: p.code,
          }))}
        />
      </ModalForm>

      {/* 批量修改费用时间 */}
      <ModalForm<{ expenseDate: string | Dayjs }>
        open={bulkActionContext?.type === 'expenseDate'}
        onOpenChange={(open) => !open && setBulkActionContext(undefined)}
        title={`批量修改费用时间（已选 ${
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
          const rows = getSelectedRows(direction);
          if (rows.length === 0) {
            message.warning('请先勾选要修改的费用');
            return false;
          }
          if (rejectBilledRows(rows, '批量修改费用时间')) return false;
          try {
            const response = await orderFeeServiceBulkUpdateOrderFees(
              { orderId },
              {
                orderId,
                targets: buildBulkTargets(rows),
                expenseDate: dayjs(values.expenseDate).format(
                  'YYYY-MM-DD HH:mm',
                ),
              },
            );
            message.success(
              `已修改 ${response.updatedCount ?? rows.length} 笔费用的费用时间`,
            );
            handleBulkDone(direction);
            setBulkActionContext(undefined);
            return true;
          } catch (error) {
            message.error(getErrorMessage(error, '批量修改费用时间失败'));
            return false;
          }
        }}
      >
        <ProFormDatePicker
          name="expenseDate"
          label="费用发生时间"
          rules={[{ required: true, message: '请选择费用发生时间' }]}
          fieldProps={{
            style: { width: '100%' },
            placeholder: '请选择费用发生时间',
            format: 'YYYY-MM-DD HH:mm',
            showTime: { format: 'HH:mm' },
          }}
        />
      </ModalForm>

      {/* 批量删除费用 */}
      <ModalForm<{ reason?: string }>
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
          const rows = getSelectedRows(direction);
          if (rows.length === 0) {
            message.warning('请先勾选要删除的费用');
            return false;
          }
          if (rejectBilledRows(rows, '批量删除')) return false;
          try {
            const response = await orderFeeServiceBulkRemoveOrderFees(
              { orderId },
              {
                orderId,
                targets: buildBulkTargets(rows),
                reason: values.reason?.trim() || undefined,
              },
            );
            message.success(
              `已删除 ${response.removedCount ?? rows.length} 笔费用`,
            );
            handleBulkDone(direction);
            setBulkActionContext(undefined);
            return true;
          } catch (error) {
            message.error(getErrorMessage(error, '批量删除费用失败'));
            return false;
          }
        }}
      >
        <Alert
          type="warning"
          showIcon
          message={`将删除 ${
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
      </ModalForm>
    </>
  );
}
