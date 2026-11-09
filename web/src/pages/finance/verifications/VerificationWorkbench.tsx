import { ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Empty,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { type Dayjs } from 'dayjs';
import Decimal from 'decimal.js';
import { useEffect, useMemo, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { MODAL_SIZE, ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import {
  disableCreditExceededOptions,
  useCreditLimitIntervention,
} from '@/features/finance/credit-control';
import { getCurrencyOptions } from '@/features/master-data/currencies';
import { PartnerSelectOptionTags } from '@/features/partners';
import {
  settlementServiceCreateVerification,
  settlementServiceListFinanceSettlementPartyOptions,
  settlementServiceListVerificationCreationCandidates,
} from '@/services/roncin/settlementService';
import type { SelectOption } from '@/types/select-option';
import { isWriteResponse } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatAmount } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import {
  buildVerificationAllocations,
  isPositiveVerificationAmount,
  sumVerificationAmounts,
  type VerificationAllocationDraft,
} from './verification-allocation';

type Props = {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
};

type Scope = {
  direction: 'RECEIVABLE' | 'PAYABLE';
  settlementPartyId?: string;
  currency: string;
  verificationDate: Dayjs;
  note?: string;
};

/** 服务端状态域前缀：核销创建工作台查询的统一 key 前缀。 */
const VERIFICATION_WORKBENCH_QUERY_BASE = [
  'finance',
  'verification-workbench',
] as const;

export default function VerificationWorkbench({
  open,
  onClose,
  onCreated,
}: Props) {
  const { markDirty, resetDirty } = useUnsavedChangesGuard({ enabled: open });
  const { message } = App.useApp();
  const { initialState } = useInitialState();
  const access = useAccess();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const canOperate =
    Boolean(initialState?.currentUser?.id) &&
    access.canCreateFinanceVerifications &&
    access.canOperateOrganization(organizationId);
  const [scope, setScope] = useState<Scope>({
    direction: 'RECEIVABLE',
    currency: 'CNY',
    verificationDate: dayjs(),
  });
  // 结算单位关键字搜索态：输入即收敛进 queryKey，竞态由库按 key 收敛。
  const [partnerKeyword, setPartnerKeyword] = useState('');
  // 关键字过滤后仍保留在展示列表头的当前选中结算单位（保持 label 可见）。
  const [selectedPartyOption, setSelectedPartyOption] =
    useState<SelectOption>();
  const [selectedCashflowIds, setSelectedCashflowIds] = useState<React.Key[]>(
    [],
  );
  const [selectedBillIds, setSelectedBillIds] = useState<React.Key[]>([]);
  const [allocations, setAllocations] = useState<VerificationAllocationDraft[]>(
    [],
  );
  const [submitting, setSubmitting] = useState(false);
  // 直接干预模式下超额客户禁选（标签与禁用都由契约字段渲染）。
  const creditInterventionActive = useCreditLimitIntervention(
    open && canOperate,
  );

  // 币种候选只随当前公司工作台打开加载。
  const bootstrapQuery = useQuery({
    queryKey: [...VERIFICATION_WORKBENCH_QUERY_BASE, 'currencies'],
    enabled: open && canOperate,
    meta: { errorMessage: '加载核销创建候选失败' },
    queryFn: getCurrencyOptions,
  });
  const currencyOptions = bootstrapQuery.data ?? [];

  // 第二段：结算单位候选，依赖已选组织（enabled 门控）与搜索关键字。
  const partnerQuery = useQuery({
    queryKey: [
      ...VERIFICATION_WORKBENCH_QUERY_BASE,
      'settlement-party-options',
      {
        organizationId: organizationId,
        keyword: partnerKeyword || undefined,
      },
    ],
    enabled: open && canOperate && Boolean(organizationId),
    // 关键字搜索期间保留上一批候选，避免下拉闪空。
    placeholderData: (previous) => previous,
    meta: { errorMessage: '加载结算单位失败' },
    queryFn: async () => {
      if (!canOperate || !organizationId) return [] as SelectOption[];
      const response = await settlementServiceListFinanceSettlementPartyOptions(
        {
          purpose:
            FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE,
          organizationId: organizationId,
          keyword: partnerKeyword || undefined,
          page: 1,
          pageSize: 50,
        },
      );
      return (response.data ?? [])
        .filter((item) => item.id)
        .map((item) => ({
          value: item.id as string,
          label:
            item.name && item.code
              ? `${item.name} (${item.code})`
              : item.name || item.code || item.id || '',
          creditExceeded: Boolean(item.creditExceeded),
        }));
    },
  });

  // 第三段：待核销资金与账单候选，依赖组织 → 结算单位 → 币种全链就绪。
  const candidatesQuery = useQuery({
    queryKey: [
      ...VERIFICATION_WORKBENCH_QUERY_BASE,
      'candidates',
      {
        organizationId: organizationId,
        direction: scope.direction,
        settlementPartyId: scope.settlementPartyId,
        currency: scope.currency,
      },
    ],
    enabled: Boolean(
      open &&
        canOperate &&
        organizationId &&
        scope.settlementPartyId &&
        scope.currency,
    ),
    meta: { errorMessage: '加载待核销资金和账单失败' },
    queryFn: async () => {
      const response =
        await settlementServiceListVerificationCreationCandidates({
          organizationId: organizationId as string,
          direction: scope.direction,
          settlementPartyId: scope.settlementPartyId as string,
          currency: scope.currency,
        });
      return {
        cashflows: response.data?.cashflows ?? [],
        bills: response.data?.bills ?? [],
      };
    },
  });
  const cashflows = candidatesQuery.data?.cashflows ?? [];
  const bills = candidatesQuery.data?.bills ?? [];
  const candidateLoading = candidatesQuery.isFetching;

  // 作用域任一维度变化（含关闭弹窗）即清空勾选与分配，旧选择不跨作用域残留。
  useEffect(() => {
    setSelectedCashflowIds([]);
    setSelectedBillIds([]);
    setAllocations([]);
  }, [
    open,
    organizationId,
    scope.direction,
    scope.settlementPartyId,
    scope.currency,
  ]);

  // 关键字过滤后把当前选中项拼回展示列表头，保持 Select 回显 label 不退化为 ID。
  const partnerOptions = useMemo(() => {
    const options = partnerQuery.data ?? [];
    const merged =
      !selectedPartyOption ||
      options.some((option) => option.value === selectedPartyOption.value)
        ? options
        : [selectedPartyOption, ...options];
    return disableCreditExceededOptions(merged, creditInterventionActive);
  }, [partnerQuery.data, selectedPartyOption, creditInterventionActive]);

  const selectedCashflows = useMemo(
    () =>
      selectedCashflowIds
        .map((id) => cashflows.find((item) => item.id === id))
        .filter((item): item is API.FinanceCashflow => Boolean(item?.id)),
    [cashflows, selectedCashflowIds],
  );
  const selectedBills = useMemo(
    () =>
      selectedBillIds
        .map((id) => bills.find((item) => item.id === id))
        .filter((item): item is API.FinanceBill => Boolean(item?.id)),
    [bills, selectedBillIds],
  );

  const autoAllocate = () => {
    if (
      !canOperate ||
      selectedCashflows.some(
        (item) => item.organizationId !== organizationId,
      ) ||
      selectedBills.some((item) => item.organizationId !== organizationId)
    ) {
      message.warning('所选资金或账单不属于当前公司，不能核销');
      return;
    }
    const next = buildVerificationAllocations(
      selectedCashflows.map((item) => ({
        id: item.id as string,
        balance: item.unverifiedAmount || '0',
      })),
      selectedBills.map((item) => ({
        id: item.id as string,
        balance: item.unverifiedAmount || '0',
      })),
    );
    if (next.length) markDirty();
    setAllocations(next);
    if (next.length === 0) message.warning('请先选择资金流水和账单');
  };

  const cashflowById = useMemo(
    () => new Map(cashflows.map((item) => [item.id, item])),
    [cashflows],
  );
  const billById = useMemo(
    () => new Map(bills.map((item) => [item.id, item])),
    [bills],
  );
  const allocationAmount = sumVerificationAmounts(
    allocations.map((item) => item.amount),
  );
  const selectedCashAmount = sumVerificationAmounts(
    selectedCashflows.map((item) => item.unverifiedAmount || '0'),
  );
  const selectedBillAmount = sumVerificationAmounts(
    selectedBills.map((item) => item.unverifiedAmount || '0'),
  );
  const validationMessage = useMemo(() => {
    if (allocations.length === 0) return '请选择资金与账单并生成核销分配';
    const pairs = new Set<string>();
    const cashUsed = new Map<string, Decimal>();
    const billUsed = new Map<string, Decimal>();
    for (const allocation of allocations) {
      if (!isPositiveVerificationAmount(allocation.amount))
        return '核销金额必须大于 0';
      const pair = `${allocation.cashflowId}:${allocation.billId}`;
      if (pairs.has(pair)) return '同一资金与账单组合不能重复';
      pairs.add(pair);
      const amount = new Decimal(allocation.amount);
      cashUsed.set(
        allocation.cashflowId,
        (cashUsed.get(allocation.cashflowId) || new Decimal(0)).plus(amount),
      );
      billUsed.set(
        allocation.billId,
        (billUsed.get(allocation.billId) || new Decimal(0)).plus(amount),
      );
    }
    for (const [id, used] of cashUsed) {
      if (used.gt(cashflowById.get(id)?.unverifiedAmount || 0))
        return '核销金额超过资金未核销余额';
    }
    for (const [id, used] of billUsed) {
      if (used.gt(billById.get(id)?.unverifiedAmount || 0))
        return '核销金额超过账单未核销余额';
    }
    return undefined;
  }, [allocations, billById, cashflowById]);

  const submit = async (): Promise<false | undefined> => {
    if (
      !canOperate ||
      !organizationId ||
      !scope.settlementPartyId ||
      validationMessage
    ) {
      message.warning(validationMessage || '请在当前公司工作区选择结算单位');
      return;
    }
    if (
      allocations.some(
        (item) =>
          cashflowById.get(item.cashflowId)?.organizationId !==
            organizationId ||
          billById.get(item.billId)?.organizationId !== organizationId,
      )
    ) {
      message.warning('所选资金或账单不属于当前公司，不能核销');
      return;
    }
    setSubmitting(true);
    try {
      const writeResponse1 = await settlementServiceCreateVerification({
        allocations,
        verificationDate: scope.verificationDate.format('YYYY-MM-DD'),
        note: scope.note,
        idempotencyKey: generateUUID(),
      });
      if (!isWriteResponse(writeResponse1, true)) return false;
      resetDirty();
      message.success('核销成功，资金与账单余额已同步更新');
      onCreated();
      onClose();
    } catch (error) {
      message.error(getErrorMessage(error, '核销失败'));
    } finally {
      setSubmitting(false);
    }

    return undefined;
  };

  const cashflowColumns: ColumnsType<API.FinanceCashflow> = [
    { title: '流水号', dataIndex: 'flowNo', width: 150 },
    { title: '交易日', dataIndex: 'transactionDate', width: 105 },
    {
      title: '未核销余额',
      dataIndex: 'unverifiedAmount',
      align: 'right',
      render: (value) => `${formatAmount(value)} ${scope.currency}`,
    },
  ];
  const billColumns: ColumnsType<API.FinanceBill> = [
    { title: '账单号', dataIndex: 'billNo', width: 150 },
    { title: '账单日', dataIndex: 'billDate', width: 105 },
    {
      title: '未核销余额',
      dataIndex: 'unverifiedAmount',
      align: 'right',
      render: (value) => `${formatAmount(value)} ${scope.currency}`,
    },
  ];
  const allocationColumns: ColumnsType<VerificationAllocationDraft> = [
    {
      title: '资金流水',
      dataIndex: 'cashflowId',
      render: (id) => cashflowById.get(id)?.flowNo || id,
    },
    {
      title: '账单',
      dataIndex: 'billId',
      render: (id) => billById.get(id)?.billNo || id,
    },
    {
      title: '本次核销金额',
      dataIndex: 'amount',
      width: 210,
      render: (value, _record, index) => (
        <Input
          aria-label={`第 ${index + 1} 行核销金额`}
          value={value}
          suffix={scope.currency}
          onChange={(event) => {
            markDirty();
            setAllocations((current) =>
              current.map((item, itemIndex) =>
                itemIndex === index
                  ? { ...item, amount: event.target.value.trim() }
                  : item,
              ),
            );
          }}
        />
      ),
    },
  ];

  const cashflowColumnSettings = useColumnSettings<
    ColumnsType<API.FinanceCashflow>[number]
  >({
    tableKey: 'finance:verification-cashflow-candidates',
    columns: cashflowColumns,
  });
  const billColumnSettings = useColumnSettings<
    ColumnsType<API.FinanceBill>[number]
  >({
    tableKey: 'finance:verification-bill-candidates',
    columns: billColumns,
  });
  const allocationColumnSettings = useColumnSettings<
    ColumnsType<VerificationAllocationDraft>[number]
  >({
    tableKey: 'finance:verification-allocations',
    columns: allocationColumns,
  });

  return (
    <Modal
      title="资金与账单核销工作台"
      open={open}
      width={MODAL_SIZE.XL}
      destroyOnHidden
      mask={{ closable: false }}
      onCancel={onClose}
      footer={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button
            type="primary"
            loading={submitting}
            disabled={Boolean(validationMessage)}
            onClick={() => void submit()}
          >
            提交核销{' '}
            {allocationAmount.isPositive()
              ? formatAmount(allocationAmount.toFixed(2))
              : ''}
          </Button>
        </Space>
      }
    >
      <Card size="small" style={{ marginBottom: 12 }}>
        <Row gutter={12}>
          <Col span={4}>
            <Typography.Text>核销方向</Typography.Text>
            <Select
              aria-label="核销方向"
              value={scope.direction}
              style={{ width: '100%', marginTop: 6 }}
              options={[
                { value: 'RECEIVABLE', label: '应收核销' },
                { value: 'PAYABLE', label: '应付核销' },
              ]}
              onChange={(direction) => {
                setScope((value) => ({ ...value, direction }));
              }}
            />
          </Col>
          <Col span={12}>
            <Typography.Text>结算单位</Typography.Text>
            <Select
              showSearch={{
                filterOption: false,
                onSearch: (keyword) => setPartnerKeyword(keyword),
              }}
              aria-label="结算单位"
              value={scope.settlementPartyId}
              style={{ width: '100%', marginTop: 6 }}
              placeholder={
                canOperate ? '请选择结算单位' : '请在公司工作区办理核销'
              }
              disabled={!canOperate}
              options={partnerOptions}
              optionRender={(option) => (
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    width: '100%',
                  }}
                >
                  <span>{option.label}</span>
                  <PartnerSelectOptionTags
                    data={option.data as { creditExceeded?: boolean }}
                  />
                </div>
              )}
              onChange={(settlementPartyId) => {
                setSelectedPartyOption(
                  (partnerQuery.data ?? []).find(
                    (option) => option.value === settlementPartyId,
                  ),
                );
                setScope((value) => ({ ...value, settlementPartyId }));
              }}
            />
          </Col>
          <Col span={4}>
            <Typography.Text>币种</Typography.Text>
            <Select
              showSearch={{ optionFilterProp: ['label', 'value'] }}
              aria-label="核销币种"
              value={scope.currency}
              options={currencyOptions}
              style={{ width: '100%', marginTop: 6 }}
              onChange={(currency) => {
                setScope((value) => ({ ...value, currency }));
              }}
            />
          </Col>
          <Col span={4}>
            <Typography.Text>核销日期</Typography.Text>
            <DatePicker
              aria-label="核销日期"
              value={scope.verificationDate}
              style={{ width: '100%', marginTop: 6 }}
              onChange={(verificationDate) => {
                if (!verificationDate) return;
                markDirty();
                setScope((value) => ({ ...value, verificationDate }));
              }}
            />
          </Col>
        </Row>
        <Tag
          color={scope.direction === 'RECEIVABLE' ? 'blue' : 'orange'}
          style={{ marginTop: 12 }}
        >
          {candidateLoading
            ? '候选加载中'
            : '候选已按所属公司、方向、结算单位和币种隔离'}
        </Tag>
      </Card>

      {!canOperate ? (
        <Empty description="请在有核销创建权限的公司工作区办理" />
      ) : !scope.settlementPartyId ? (
        <Empty description="请先选择结算单位，再选择同方向、同币种的资金与账单" />
      ) : (
        <Row gutter={12}>
          <Col span={12}>
            <Card size="small" title={`待核销资金（${cashflows.length}）`}>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'flex-end',
                  marginBottom: 8,
                }}
              >
                {cashflowColumnSettings.entry}
              </div>
              <Table
                size="small"
                loading={candidateLoading}
                rowKey="id"
                columns={cashflowColumnSettings.columns}
                dataSource={cashflows}
                pagination={{ pageSize: 6, showSizeChanger: false }}
                rowSelection={{
                  selectedRowKeys: selectedCashflowIds,
                  preserveSelectedRowKeys: true,
                  onChange: setSelectedCashflowIds,
                  getCheckboxProps: (record) => ({
                    disabled:
                      !canOperate || record.organizationId !== organizationId,
                  }),
                }}
              />
            </Card>
          </Col>
          <Col span={12}>
            <Card size="small" title={`待核销账单（${bills.length}）`}>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'flex-end',
                  marginBottom: 8,
                }}
              >
                {billColumnSettings.entry}
              </div>
              <Table
                size="small"
                loading={candidateLoading}
                rowKey="id"
                columns={billColumnSettings.columns}
                dataSource={bills}
                pagination={{ pageSize: 6, showSizeChanger: false }}
                rowSelection={{
                  selectedRowKeys: selectedBillIds,
                  preserveSelectedRowKeys: true,
                  onChange: setSelectedBillIds,
                  getCheckboxProps: (record) => ({
                    disabled:
                      !canOperate || record.organizationId !== organizationId,
                  }),
                }}
              />
            </Card>
          </Col>
        </Row>
      )}

      <Card
        size="small"
        title="核销分配"
        style={{ marginTop: 12 }}
        extra={
          <Button icon={<ReloadOutlined />} onClick={autoAllocate}>
            按余额自动分配
          </Button>
        }
      >
        <Descriptions
          size="small"
          column={3}
          items={[
            {
              key: 'cash',
              label: '已选资金余额',
              children: `${formatAmount(selectedCashAmount.toFixed(2))} ${scope.currency}`,
            },
            {
              key: 'allocation',
              label: '本次分配',
              children: `${formatAmount(allocationAmount.toFixed(2))} ${scope.currency}`,
            },
            {
              key: 'bill',
              label: '已选账单余额',
              children: `${formatAmount(selectedBillAmount.toFixed(2))} ${scope.currency}`,
            },
          ]}
        />
        {validationMessage && allocations.length > 0 ? (
          <Alert
            type="error"
            showIcon
            title={validationMessage}
            style={{ marginBottom: 12 }}
          />
        ) : null}
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginBottom: 8,
          }}
        >
          {allocationColumnSettings.entry}
        </div>
        <Table
          size="small"
          rowKey={(item) => `${item.cashflowId}:${item.billId}`}
          columns={allocationColumnSettings.columns}
          dataSource={allocations}
          pagination={false}
          locale={{ emptyText: '选择资金和账单后，点击“按余额自动分配”' }}
        />
        <Input.TextArea
          aria-label="核销备注"
          value={scope.note}
          maxLength={500}
          placeholder="核销备注（选填）"
          style={{ marginTop: 12 }}
          onChange={(event) => {
            markDirty();
            setScope((value) => ({ ...value, note: event.target.value }));
          }}
        />
      </Card>
    </Modal>
  );
}
