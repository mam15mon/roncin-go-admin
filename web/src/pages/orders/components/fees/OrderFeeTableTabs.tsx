import {
  EditOutlined,
  FileDoneOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { EditableProTable } from '@ant-design/pro-components';
import { App, Button, Popconfirm, Space, Tag, Tooltip } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { SectionCard } from '@/components/ui';
import {
  normalizeOrderFeeStatus,
  orderFeeStatusMeta,
  statusTag,
} from '@/constants/statusMeta';
import { history } from '@/router/history';
import {
  orderFeeServiceAddFee,
  orderFeeServiceListFees,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { unwrapList } from '@/utils/api';
import { quantityOrPricePattern } from '@/utils/decimal';
import { getErrorMessage } from '@/utils/errorMessage';
import { trimDecimal } from '@/utils/format';
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
  FEE_CONFIRMED,
  FEE_DRAFT,
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
    const trimmed = String(value).trim();
    if (!pattern.test(trimmed) || Number(trimmed) <= 0) {
      return Promise.reject(new Error(messageText));
    }
    return Promise.resolve();
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
  onConfirmFee?: (fee: API.OrderFee) => void;
  onReopenFee?: (fee: API.OrderFee) => void;
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
  onConfirmFee,
  onReopenFee,
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

  currentOrderIdRef.current = orderId;

  useEffect(() => {
    mountedRef.current = true;
    setReceivableEditableKeys([]);
    setPayableEditableKeys([]);
    return () => {
      mountedRef.current = false;
      receivableRequestSequenceRef.current += 1;
      payableRequestSequenceRef.current += 1;
    };
  }, []);

  // 默认计费单位：优先查找“票”（PIAO / 包含票），否则选用首项
  const defaultBillingUnit =
    billingUnits?.find(
      (u) => u.code === 'PIAO' || u.name === '票' || u.name?.includes('票'),
    ) || billingUnits?.[0];

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
        expenseDate: dayjs().format('YYYY-MM-DD'),
        status: 1, // FEE_DRAFT
      },
      { position: 'top' },
    );
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
        expenseDate: dayjs().format('YYYY-MM-DD'),
        status: 1, // FEE_DRAFT
      },
      { position: 'top' },
    );
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
      return false;
    }
    if (!row.settlementPartyId) {
      message.error('请选择结算单位');
      return false;
    }
    if (!row.currency) {
      message.error('请选择币种');
      return false;
    }
    if (
      !row.unitPrice ||
      !quantityOrPricePattern.test(String(row.unitPrice).trim()) ||
      Number(row.unitPrice) <= 0
    ) {
      message.error('单价必须为大于 0 的有效数值');
      return false;
    }
    if (
      !row.quantity ||
      !quantityOrPricePattern.test(String(row.quantity).trim()) ||
      Number(row.quantity) <= 0
    ) {
      message.error('数量必须为大于 0 的有效数值');
      return false;
    }
    if (!row.billingUnitId) {
      message.error('请选择计费单位');
      return false;
    }
    if (!row.expenseDate) {
      message.error('请选择发生日期');
      return false;
    }

    const body = {
      direction,
      feeSettingId: row.feeSettingId,
      settlementPartyId: row.settlementPartyId,
      billingUnitId: row.billingUnitId,
      quantity: String(row.quantity).trim(),
      unitPrice: String(row.unitPrice).trim(),
      currency: row.currency,
      expenseDate: dayjs(row.expenseDate).format('YYYY-MM-DD'),
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
        render: (_, record) => record.feeCode || '-',
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
            defaultCurrency: item.defaultCurrency,
            defaultBillingUnitId: item.defaultBillingUnitId,
          })),
          filterOption: (
            input: string,
            option?: { label?: string; feeCode?: string },
          ) =>
            Boolean(
              option?.label?.toLowerCase().includes(input.toLowerCase()) ||
                option?.feeCode?.toLowerCase().includes(input.toLowerCase()),
            ),
          onChange: (
            _: string,
            option?: {
              nameZh?: string;
              feeCode?: string;
              defaultCurrency?: string;
              defaultBillingUnitId?: string;
            },
          ) => {
            if (option) {
              form?.setFieldValue([rowKey, 'feeName'], option.nameZh);
              form?.setFieldValue([rowKey, 'feeCode'], option.feeCode);
              if (option.defaultCurrency) {
                form?.setFieldValue(
                  [rowKey, 'currency'],
                  option.defaultCurrency,
                );
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
          filterOption: (
            input: string,
            option?: { label?: string; code?: string },
          ) =>
            Boolean(
              option?.label?.toLowerCase().includes(input.toLowerCase()) ||
                option?.code?.toLowerCase().includes(input.toLowerCase()),
            ),
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
        fieldProps: {
          options: (currencies || []).map((c) => ({
            label: c.code,
            value: c.code ?? '',
          })),
        },
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
        fieldProps: {
          placeholder: '0.00',
          style: { textAlign: 'right' },
        },
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
        fieldProps: {
          placeholder: '1',
          style: { textAlign: 'right' },
        },
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
          placeholder: '计费单位',
          options: (billingUnits || []).map((u) => ({
            label: u.name,
            value: u.id ?? '',
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
        render: (_, record) => {
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
        width: 100,
        align: 'right',
        editable: false,
        render: (_, record) => (
          <Space size={4}>
            <span>{trimDecimal(record.exchangeRate)}</span>
            {record.exchangeRateSource === 'MANUAL' && (
              <Tag color="gold">手工</Tag>
            )}
            {record.exchangeRateSource === 'SYSTEM' && (
              <Tag color="blue">系统</Tag>
            )}
          </Space>
        ),
      },
      {
        title: '发生日期',
        dataIndex: 'expenseDate',
        width: 130,
        valueType: 'date',
        formItemProps: {
          rules: [{ required: true, message: '请选择发生日期' }],
        },
        render: (_, record) => record.expenseDate || '-',
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
      ...buildOptionalFeeColumns(feeBillTracking),
      {
        title: '操作',
        valueType: 'option',
        width: 120,
        fixed: 'right',
        render: (_, record, __, action) => {
          if (feeWritesDisabled) return [];
          return [
            (feeStatusCode(record.status) === FEE_DRAFT ||
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
            feeStatusCode(record.status) === FEE_DRAFT && onConfirmFee && (
              <Popconfirm
                key="confirm"
                title="确认后该费用才能进入账单，确定继续？"
                onConfirm={() => onConfirmFee(record)}
              >
                <Button type="link" size="small">
                  确认
                </Button>
              </Popconfirm>
            ),
            feeStatusCode(record.status) === FEE_CONFIRMED && onReopenFee && (
              <Button
                key="reopen"
                type="link"
                size="small"
                onClick={() => onReopenFee(record)}
              >
                撤回
              </Button>
            ),
            (feeStatusCode(record.status) === FEE_DRAFT ||
              feeStatusCode(record.status) === FEE_CONFIRMED) &&
              onCancelFee && (
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
          <Space size={8}>
            {renderColumnToolbar()}
            {canCreateFinanceBills && (
              <Button
                key="bill"
                icon={<FileDoneOutlined />}
                disabled={selectedReceivableFeeIds.length === 0}
                onClick={() =>
                  onOpenBillWorkbench(selectedReceivableFeeIds.map(String))
                }
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
            actionRender: (_row, _config, defaultDom) => [
              defaultDom.save,
              defaultDom.cancel,
            ],
          }}
          rowSelection={{
            selectedRowKeys: selectedReceivableFeeIds,
            onChange: setSelectedReceivableFeeIds,
            getCheckboxProps: (record) => ({
              disabled: feeStatusCode(record.status) !== FEE_CONFIRMED,
            }),
          }}
          tableAlertRender={({ selectedRowKeys }) =>
            `已选择 ${selectedRowKeys.length} 笔已确认应收费用`
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
                  (cur.baseCurrencyAmount
                    ? Number(cur.baseCurrencyAmount)
                    : 0),
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
          <Space size={8}>
            {renderColumnToolbar()}
            {canCreateFinanceBills && (
              <Button
                key="bill"
                icon={<FileDoneOutlined />}
                disabled={selectedPayableFeeIds.length === 0}
                onClick={() =>
                  onOpenBillWorkbench(selectedPayableFeeIds.map(String))
                }
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
            actionRender: (_row, _config, defaultDom) => [
              defaultDom.save,
              defaultDom.cancel,
            ],
          }}
          rowSelection={{
            selectedRowKeys: selectedPayableFeeIds,
            onChange: setSelectedPayableFeeIds,
            getCheckboxProps: (record) => ({
              disabled: feeStatusCode(record.status) !== FEE_CONFIRMED,
            }),
          }}
          tableAlertRender={({ selectedRowKeys }) =>
            `已选择 ${selectedRowKeys.length} 笔已确认应付费用`
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
                  (cur.baseCurrencyAmount
                    ? Number(cur.baseCurrencyAmount)
                    : 0),
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
    </>
  );
}
