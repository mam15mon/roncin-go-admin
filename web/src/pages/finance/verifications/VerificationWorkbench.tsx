import { ReloadOutlined } from '@ant-design/icons';
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
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { type Dayjs } from 'dayjs';
import Decimal from 'decimal.js';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import PartnerSelectOptionTags from '@/components/PartnerSelectOptionTags';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import { useCreditLimitIntervention } from '@/hooks/useCreditLimitIntervention';
import {
  settlementServiceCreateVerification,
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceListFinanceSettlementPartyOptions,
  settlementServiceListVerificationCreationCandidates,
} from '@/services/roncin/settlementService';
import {
  disableCreditExceededOptions,
  getCurrencyOptions,
  type SelectOption,
} from '@/utils/options';
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
  organizationId?: string;
  direction: 'RECEIVABLE' | 'PAYABLE';
  settlementPartyId?: string;
  currency: string;
  verificationDate: Dayjs;
  note?: string;
};

export default function VerificationWorkbench({
  open,
  onClose,
  onCreated,
}: Props) {
  const { message } = App.useApp();
  const [scope, setScope] = useState<Scope>({
    direction: 'RECEIVABLE',
    currency: 'CNY',
    verificationDate: dayjs(),
  });
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [partnerOptions, setPartnerOptions] = useState<SelectOption[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);
  const [cashflows, setCashflows] = useState<API.FinanceCashflow[]>([]);
  const [bills, setBills] = useState<API.FinanceBill[]>([]);
  const [selectedCashflowIds, setSelectedCashflowIds] = useState<React.Key[]>(
    [],
  );
  const [selectedBillIds, setSelectedBillIds] = useState<React.Key[]>([]);
  const [allocations, setAllocations] = useState<VerificationAllocationDraft[]>(
    [],
  );
  const [candidateLoading, setCandidateLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const partnerSearchSequence = useRef(0);
  // 直接干预模式下超额客户禁选（标签与禁用都由契约字段渲染）。
  const creditInterventionActive = useCreditLimitIntervention();

  const resetSelections = () => {
    setSelectedCashflowIds([]);
    setSelectedBillIds([]);
    setAllocations([]);
  };

  const clearCandidates = () => {
    setCashflows([]);
    setBills([]);
    resetSelections();
  };

  const loadPartnerOptions = useCallback(
    async (keyword?: string, selectedId?: string) => {
      if (!scope.organizationId) {
        setPartnerOptions([]);
        return;
      }
      const sequence = ++partnerSearchSequence.current;
      try {
        const response =
          await settlementServiceListFinanceSettlementPartyOptions({
            purpose:
              FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE,
            organizationId: scope.organizationId,
            keyword,
            page: 1,
            pageSize: 50,
          });
        const options = disableCreditExceededOptions(
          (response.data ?? [])
            .filter((item) => item.id)
            .map((item) => ({
              value: item.id as string,
              label:
                item.name && item.code
                  ? `${item.name} (${item.code})`
                  : item.name || item.code || item.id || '',
              creditExceeded: Boolean(item.creditExceeded),
            })),
          creditInterventionActive,
        );
        if (sequence !== partnerSearchSequence.current) return;
        setPartnerOptions((current) => {
          const selected = current.find(
            (option) => option.value === selectedId,
          );
          return selected
            ? [
                selected,
                ...options.filter((option) => option.value !== selected.value),
              ]
            : options;
        });
      } catch {
        if (sequence === partnerSearchSequence.current) {
          message.error('加载结算单位失败');
        }
      }
    },
    [creditInterventionActive, message, scope.organizationId],
  );

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void Promise.all([
      settlementServiceListFinanceOrganizationOptions({
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE,
      }),
      getCurrencyOptions(),
    ])
      .then(([organizationResponse, currencies]) => {
        if (cancelled) return;
        setOrganizationOptions(organizationResponse.data ?? []);
        setCurrencyOptions(currencies);
      })
      .catch(() => {
        if (!cancelled) message.error('加载核销创建候选失败');
      });
    return () => {
      cancelled = true;
    };
  }, [message, open]);

  useEffect(() => {
    if (!open || !scope.organizationId) {
      setPartnerOptions([]);
      clearCandidates();
      return;
    }
    void loadPartnerOptions(undefined, scope.settlementPartyId);
  }, [loadPartnerOptions, open, scope.organizationId, scope.settlementPartyId]);

  useEffect(() => {
    if (
      !open ||
      !scope.organizationId ||
      !scope.settlementPartyId ||
      !scope.currency
    ) {
      clearCandidates();
      return;
    }
    let cancelled = false;
    setCandidateLoading(true);
    resetSelections();
    void settlementServiceListVerificationCreationCandidates({
      organizationId: scope.organizationId,
      direction: scope.direction,
      settlementPartyId: scope.settlementPartyId,
      currency: scope.currency,
    })
      .then((response) => {
        if (cancelled) return;
        setCashflows(response.data?.cashflows ?? []);
        setBills(response.data?.bills ?? []);
      })
      .catch(() => {
        if (!cancelled) message.error('加载待核销资金和账单失败');
      })
      .finally(() => {
        if (!cancelled) setCandidateLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    message,
    open,
    scope.currency,
    scope.direction,
    scope.organizationId,
    scope.settlementPartyId,
  ]);

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

  const submit = async () => {
    if (
      !scope.organizationId ||
      !scope.settlementPartyId ||
      validationMessage
    ) {
      message.warning(validationMessage || '请先选择所属公司和结算单位');
      return;
    }
    setSubmitting(true);
    try {
      await settlementServiceCreateVerification({
        allocations,
        verificationDate: scope.verificationDate.format('YYYY-MM-DD'),
        note: scope.note,
        idempotencyKey: generateUUID(),
      });
      message.success('核销成功，资金与账单余额已同步更新');
      onCreated();
      onClose();
    } catch (error: any) {
      message.error(error.message || '核销失败');
    } finally {
      setSubmitting(false);
    }
  };

  const cashflowColumns: ColumnsType<API.FinanceCashflow> = [
    { title: '流水号', dataIndex: 'flowNo', width: 150 },
    { title: '交易日', dataIndex: 'transactionDate', width: 105 },
    {
      title: '未核销余额',
      dataIndex: 'unverifiedAmount',
      align: 'right',
      render: (value) => `${value} ${scope.currency}`,
    },
  ];
  const billColumns: ColumnsType<API.FinanceBill> = [
    { title: '账单号', dataIndex: 'billNo', width: 150 },
    { title: '账单日', dataIndex: 'billDate', width: 105 },
    {
      title: '未核销余额',
      dataIndex: 'unverifiedAmount',
      align: 'right',
      render: (value) => `${value} ${scope.currency}`,
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
          onChange={(event) =>
            setAllocations((current) =>
              current.map((item, itemIndex) =>
                itemIndex === index
                  ? { ...item, amount: event.target.value.trim() }
                  : item,
              ),
            )
          }
        />
      ),
    },
  ];

  return (
    <Modal
      title="资金与账单核销工作台"
      open={open}
      width={1240}
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
            {allocationAmount.isPositive() ? allocationAmount.toFixed(2) : ''}
          </Button>
        </Space>
      }
    >
      <Card size="small" style={{ marginBottom: 12 }}>
        <Row gutter={12}>
          <Col span={5}>
            <Typography.Text>所属公司</Typography.Text>
            <Select
              aria-label="所属公司"
              value={scope.organizationId}
              style={{ width: '100%', marginTop: 6 }}
              placeholder="请选择有核销创建权限的公司"
              options={organizationOptions.map((item) => ({
                value: item.id,
                label: item.name || item.code || item.id,
              }))}
              onChange={(organizationId) => {
                partnerSearchSequence.current += 1;
                setPartnerOptions([]);
                clearCandidates();
                setScope((value) => ({
                  ...value,
                  organizationId,
                  settlementPartyId: undefined,
                }));
              }}
            />
          </Col>
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
                clearCandidates();
                setScope((value) => ({ ...value, direction }));
              }}
            />
          </Col>
          <Col span={7}>
            <Typography.Text>结算单位</Typography.Text>
            <Select
              showSearch={{
                filterOption: false,
                onSearch: (keyword) =>
                  void loadPartnerOptions(keyword, scope.settlementPartyId),
              }}
              aria-label="结算单位"
              value={scope.settlementPartyId}
              style={{ width: '100%', marginTop: 6 }}
              placeholder={
                scope.organizationId ? '请选择结算单位' : '请先选择所属公司'
              }
              disabled={!scope.organizationId}
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
                clearCandidates();
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
                clearCandidates();
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
              onChange={(verificationDate) =>
                verificationDate &&
                setScope((value) => ({ ...value, verificationDate }))
              }
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

      {!scope.organizationId ? (
        <Empty description="请先选择有核销创建权限的所属公司" />
      ) : !scope.settlementPartyId ? (
        <Empty description="请先选择结算单位，再选择同方向、同币种的资金与账单" />
      ) : (
        <Row gutter={12}>
          <Col span={12}>
            <Card size="small" title={`待核销资金（${cashflows.length}）`}>
              <Table
                size="small"
                loading={candidateLoading}
                rowKey="id"
                columns={cashflowColumns}
                dataSource={cashflows}
                pagination={{ pageSize: 6, showSizeChanger: false }}
                rowSelection={{
                  selectedRowKeys: selectedCashflowIds,
                  preserveSelectedRowKeys: true,
                  onChange: setSelectedCashflowIds,
                }}
              />
            </Card>
          </Col>
          <Col span={12}>
            <Card size="small" title={`待核销账单（${bills.length}）`}>
              <Table
                size="small"
                loading={candidateLoading}
                rowKey="id"
                columns={billColumns}
                dataSource={bills}
                pagination={{ pageSize: 6, showSizeChanger: false }}
                rowSelection={{
                  selectedRowKeys: selectedBillIds,
                  preserveSelectedRowKeys: true,
                  onChange: setSelectedBillIds,
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
              children: `${selectedCashAmount.toFixed(8)} ${scope.currency}`,
            },
            {
              key: 'allocation',
              label: '本次分配',
              children: `${allocationAmount.toFixed(8)} ${scope.currency}`,
            },
            {
              key: 'bill',
              label: '已选账单余额',
              children: `${selectedBillAmount.toFixed(8)} ${scope.currency}`,
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
        <Table
          size="small"
          rowKey={(item) => `${item.cashflowId}:${item.billId}`}
          columns={allocationColumns}
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
          onChange={(event) =>
            setScope((value) => ({ ...value, note: event.target.value }))
          }
        />
      </Card>
    </Modal>
  );
}
