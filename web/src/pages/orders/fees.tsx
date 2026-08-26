import {
  ArrowLeftOutlined,
  EditOutlined,
  FileDoneOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { SectionCard } from '@/components/ui';
import BillCreationWorkbench from '@/pages/finance/bills/components/BillCreationWorkbench';
import {
  feeCatalogServiceCreateFeeSetting,
  feeCatalogServiceListTaxableServices,
} from '@/services/roncin/feeCatalogService';
import {
  orderFeeServiceAddFee,
  orderFeeServiceConfirmFee,
  orderFeeServiceListFeeOptions,
  orderFeeServiceListFees,
  orderFeeServiceRemoveFee,
  orderFeeServiceReopenFee,
  orderFeeServiceResolveFeeExchangeRate,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import { parseOrderKind } from './common';
import {
  calculateExactFeeTotal,
  exchangeRatePattern,
  isPositiveExactDecimal,
  quantityOrPricePattern,
  trimExactDecimal,
} from './order-fee-decimal';

const { Text } = Typography;

const RECEIVABLE = 1;
const PAYABLE = 2;
const FEE_DRAFT = 1;
const FEE_CONFIRMED = 2;
const FEE_BILLED = 3;
const FEE_CANCELLED = 4;

const FEE_STATUS_CODES: Record<string, number> = {
  ORDER_FEE_STATUS_DRAFT: FEE_DRAFT,
  ORDER_FEE_STATUS_CONFIRMED: FEE_CONFIRMED,
  ORDER_FEE_STATUS_BILLED: FEE_BILLED,
  ORDER_FEE_STATUS_CANCELLED: FEE_CANCELLED,
};

const FEE_DIRECTION_CODES: Record<string, number> = {
  ORDER_FEE_DIRECTION_RECEIVABLE: RECEIVABLE,
  ORDER_FEE_DIRECTION_PAYABLE: PAYABLE,
};

function feeDirectionCode(direction: unknown): number {
  if (typeof direction === 'number') return direction;
  return FEE_DIRECTION_CODES[String(direction)] ?? 0;
}

function feeStatusCode(status: unknown): number {
  if (typeof status === 'number') return status;
  return FEE_STATUS_CODES[String(status)] ?? 0;
}

type FeeFormValues = {
  direction: number;
  feeSettingId: string;
  settlementPartyId: string;
  billingUnitId: string;
  quantity: string;
  unitPrice: string;
  currency: string;
  expenseDate: string | Dayjs;
  note?: string;
  exchangeRateOverride?: string;
};

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

type FeeRequestError = Error & {
  data?: { message?: string; reason?: string };
  response?: { data?: { message?: string; reason?: string } };
};

function positiveDecimalRule(pattern: RegExp, precisionMessage: string) {
  return async (_: unknown, value?: string) => {
    if (!value) throw new Error('请输入数值');
    if (!pattern.test(value)) throw new Error(precisionMessage);
    if (!isPositiveExactDecimal(value, pattern)) {
      throw new Error('数值必须大于 0');
    }
  };
}

export default function OrderFeesPage() {
  const params = useParams<{ kind: string; id: string }>();
  const access = useAccess();
  const { message, modal } = App.useApp();

  const kind = params.kind || 'sea-export';
  const orderId = params.id;
  const config = parseOrderKind(kind) || {
    kind: 'sea-export',
    title: '海运出口',
    businessType: 1,
    category: 'sea',
  };

  const receivableActionRef = useRef<ActionType | undefined>(undefined);
  const payableActionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance<FeeFormValues> | undefined>(undefined);
  const exchangeRateRequestRef = useRef(0);
  const createIdempotencyKeyRef = useRef(globalThis.crypto.randomUUID());

  const [loading, setLoading] = useState(true);
  const [order, setOrder] = useState<API.Order>();
  const [modalOpen, setModalOpen] = useState(false);
  const [modalDirection, setModalDirection] = useState<number>(RECEIVABLE);
  const [editingFee, setEditingFee] = useState<API.OrderFee>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [billWorkbenchFeeIds, setBillWorkbenchFeeIds] = useState<string[]>([]);
  const [selectedReceivableFeeIds, setSelectedReceivableFeeIds] = useState<
    React.Key[]
  >([]);
  const [selectedPayableFeeIds, setSelectedPayableFeeIds] = useState<
    React.Key[]
  >([]);

  const [currencies, setCurrencies] = useState<API.OrderFeeCurrencyOption[]>(
    [],
  );
  const [settlementParties, setSettlementParties] = useState<
    API.OrderFeeSettlementPartyOption[]
  >([]);
  const [feeSettings, setFeeSettings] = useState<API.OrderFeeSettingOption[]>(
    [],
  );
  const [billingUnits, setBillingUnits] = useState<
    API.OrderFeeBillingUnitOption[]
  >([]);
  const [_selectedFeeSetting, setSelectedFeeSetting] =
    useState<API.OrderFeeSettingOption>();

  const [totalPreview, setTotalPreview] = useState<string>();
  const [exchangeRatePreview, setExchangeRatePreview] = useState<string>();
  const [exchangeRateStatus, setExchangeRateStatus] =
    useState<ExchangeRateStatus>('idle');
  const [manualExchangeRate, setManualExchangeRate] = useState(false);

  // 汇总统计数据
  const [receivableSummary, setReceivableSummary] = useState<{
    totalAmount: number;
    count: number;
  }>({ totalAmount: 0, count: 0 });
  const [payableSummary, setPayableSummary] = useState<{
    totalAmount: number;
    count: number;
  }>({ totalAmount: 0, count: 0 });

  // 快捷新增费目状态
  const [quickAddFeeModalOpen, setQuickAddFeeModalOpen] = useState(false);
  const [quickAddFeeSaving, setQuickAddFeeSaving] = useState(false);
  const [taxableServices, setTaxableServices] = useState<API.TaxableService[]>([]);
  const [quickAddFeeForm] = Form.useForm();

  // 快捷新建结算单位状态
  const [quickAddPartnerModalOpen, setQuickAddPartnerModalOpen] = useState(false);
  const [quickAddPartnerSaving, setQuickAddPartnerSaving] = useState(false);
  const [quickAddPartnerForm] = Form.useForm();

  const handleOpenQuickAddFee = async () => {
    quickAddFeeForm.resetFields();
    quickAddFeeForm.setFieldsValue({
      defaultCurrency: 'CNY',
      billingUnitId: billingUnits[0]?.id || '',
      taxRate: '0',
    });
    setQuickAddFeeModalOpen(true);
    try {
      const res = await feeCatalogServiceListTaxableServices({
        skipErrorHandler: true,
      });
      setTaxableServices(res.data || []);
      if (res.data && res.data.length > 0) {
        quickAddFeeForm.setFieldValue('taxableServiceId', res.data[0].id);
      }
    } catch {
      // ignore
    }
  };

  const handleSaveQuickAddFee = async () => {
    const values = await quickAddFeeForm.validateFields();
    setQuickAddFeeSaving(true);
    try {
      const res = await feeCatalogServiceCreateFeeSetting(
        {
          feeCode: values.feeCode.trim().toUpperCase(),
          nameZh: values.nameZh.trim(),
          nameEn: values.nameEn?.trim() || undefined,
          defaultCurrency: values.defaultCurrency,
          billingUnitId: values.billingUnitId,
          taxRate: values.taxRate,
          taxableServiceId:
            values.taxableServiceId || taxableServices[0]?.id || '',
        },
        { skipErrorHandler: true },
      );
      const created = res.data;
      if (created) {
        const newOption: API.OrderFeeSettingOption = {
          id: created.id,
          feeCode: created.feeCode,
          nameZh: created.nameZh,
          nameEn: created.nameEn,
          defaultCurrency: created.defaultCurrency,
          defaultBillingUnitId: created.billingUnitId,
          defaultBillingUnitName: billingUnits.find(
            (b) => b.id === created.billingUnitId,
          )?.name,
          taxRate: created.taxRate,
        };
        setFeeSettings((prev) => [newOption, ...prev]);
        formRef.current?.setFieldValue('feeSettingId', created.id);
        setSelectedFeeSetting(newOption);
        if (created.billingUnitId) {
          formRef.current?.setFieldValue('billingUnitId', created.billingUnitId);
        }
        if (created.defaultCurrency) {
          formRef.current?.setFieldValue('currency', created.defaultCurrency);
        }
        handleValuesChange();
        message.success(`已成功新建费用科目【${created.nameZh}】并自动选用`);
        setQuickAddFeeModalOpen(false);
      }
    } catch (error: any) {
      message.error(error.message || '新建费用科目失败');
    } finally {
      setQuickAddFeeSaving(false);
    }
  };

  const handleOpenQuickAddPartner = () => {
    quickAddPartnerForm.resetFields();
    quickAddPartnerForm.setFieldsValue({
      roles: [modalDirection === RECEIVABLE ? 'CUSTOMER' : 'SUPPLIER'],
    });
    setQuickAddPartnerModalOpen(true);
  };

  const handleSaveQuickAddPartner = async () => {
    const values = await quickAddPartnerForm.validateFields();
    setQuickAddPartnerSaving(true);
    try {
      const res = await partnerServiceCreatePartner(
        {
          legalName: values.legalName.trim(),
          code: values.code
            ? values.code.trim().toUpperCase()
            : `P${Date.now().toString().slice(-6)}`,
          unifiedSocialCreditCode:
            values.unifiedSocialCreditCode?.trim().toUpperCase() || undefined,
          roles: (values.roles || ['SUPPLIER']).map((role: string) => ({
            type: role === 'CUSTOMER' ? 1 : 2,
            enabled: true,
          })),
        },
        { skipErrorHandler: true },
      );
      const created = res.data;
      if (created) {
        const newOption: API.OrderFeeSettlementPartyOption = {
          id: created.id,
          name: created.legalName,
          code: created.code,
        };
        setSettlementParties((prev) => [newOption, ...prev]);
        formRef.current?.setFieldValue('settlementPartyId', created.id);
        message.success(`已成功新建往来单位【${created.legalName}】并自动选用`);
        setQuickAddPartnerModalOpen(false);
      }
    } catch (error: any) {
      message.error(error.message || '新建往来单位失败');
    } finally {
      setQuickAddPartnerSaving(false);
    }
  };

  const loadData = async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const [orderRes, optionsRes] = await Promise.all([
        orderServiceGetOrder({ id: orderId }),
        orderFeeServiceListFeeOptions({ orderId }),
      ]);
      setOrder(orderRes.data);
      setCurrencies(optionsRes.currencies ?? []);
      setSettlementParties(optionsRes.settlementParties ?? []);
      setFeeSettings(optionsRes.feeSettings ?? []);
      setBillingUnits(optionsRes.billingUnits ?? []);
    } catch (error: any) {
      message.error(error.message || '加载费用信息失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadData();
  }, [orderId]);

  useEffect(() => {
    if (order?.orderNo && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: window.location.pathname,
            title: `${order.orderNo}_费用录入`,
          },
        }),
      );
    }
  }, [order?.orderNo]);

  const resolveExchangeRate = (
    currentOrderId: string,
    direction: number,
    currency: string,
    expenseDate: string,
  ) => {
    const requestSequence = ++exchangeRateRequestRef.current;
    setExchangeRateStatus('loading');
    setExchangeRatePreview(undefined);
    setManualExchangeRate(false);
    formRef.current?.setFieldValue('exchangeRateOverride', undefined);
    void orderFeeServiceResolveFeeExchangeRate(
      { orderId: currentOrderId, direction, currency, expenseDate },
      { skipErrorHandler: true },
    )
      .then((response) => {
        if (requestSequence !== exchangeRateRequestRef.current) return;
        if (!response.exchangeRate) {
          setExchangeRateStatus('error');
          message.error('汇率解析结果不完整');
          return;
        }
        setExchangeRatePreview(trimExactDecimal(response.exchangeRate));
        setExchangeRateStatus('resolved');
      })
      .catch((error: FeeRequestError) => {
        if (requestSequence !== exchangeRateRequestRef.current) return;
        const code = error.data?.reason || error.response?.data?.reason;
        if (code === 'FEE_EXCHANGE_RATE_MISSING') {
          setExchangeRateStatus('missing');
          setManualExchangeRate(true);
          return;
        }
        setExchangeRateStatus('error');
        message.error(error.message || '汇率解析失败');
      });
  };

  const handleValuesChange = () => {
    const values = formRef.current?.getFieldsValue();
    if (!values) return;
    const { quantity, unitPrice, currency, expenseDate, direction } = values;
    if (
      quantity &&
      unitPrice &&
      quantityOrPricePattern.test(quantity) &&
      quantityOrPricePattern.test(unitPrice)
    ) {
      setTotalPreview(calculateExactFeeTotal(quantity, unitPrice));
    } else {
      setTotalPreview(undefined);
    }
    if (orderId && direction && currency && expenseDate) {
      resolveExchangeRate(
        orderId,
        Number(direction),
        currency,
        dayjs(expenseDate).format('YYYY-MM-DD'),
      );
    }
  };

  const openFeeModal = (direction: number, fee?: API.OrderFee) => {
    if (!fee) createIdempotencyKeyRef.current = globalThis.crypto.randomUUID();
    setEditingFee(fee);
    setModalDirection(direction);
    setSelectedFeeSetting(undefined);
    setTotalPreview(undefined);
    setExchangeRatePreview(undefined);
    setExchangeRateStatus('idle');
    setManualExchangeRate(false);
    setModalOpen(true);

    if (fee) {
      const setting = feeSettings.find((item) => item.id === fee.feeSettingId);
      setSelectedFeeSetting(setting);
      setTotalPreview(calculateExactFeeTotal(fee.quantity, fee.unitPrice));
      setExchangeRatePreview(
        fee.exchangeRate ? trimExactDecimal(fee.exchangeRate) : undefined,
      );
      setExchangeRateStatus('resolved');
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          direction: fee.direction ?? direction,
          feeSettingId: fee.feeSettingId ?? '',
          settlementPartyId: fee.settlementPartyId ?? '',
          billingUnitId: fee.billingUnitId ?? '',
          quantity: fee.quantity ? trimExactDecimal(fee.quantity) : '',
          unitPrice: fee.unitPrice ? trimExactDecimal(fee.unitPrice) : '',
          currency: fee.currency ?? '',
          expenseDate: fee.expenseDate ? dayjs(fee.expenseDate) : dayjs(),
          note: fee.note ?? '',
        });
      }, 0);
    } else {
      const defaultParty =
        direction === RECEIVABLE
          ? order?.customerId
          : order?.bookingAgentId || order?.carrierId;
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          direction,
          settlementPartyId: defaultParty ?? '',
          currency: 'CNY',
          quantity: '1',
          expenseDate: dayjs(),
        });
        handleValuesChange();
      }, 0);
    }
  };

  const handleModalSubmit = async (values: FeeFormValues) => {
    if (!orderId) return false;
    const body = {
      direction: values.direction,
      feeSettingId: values.feeSettingId,
      settlementPartyId: values.settlementPartyId,
      billingUnitId: values.billingUnitId,
      quantity: values.quantity,
      unitPrice: values.unitPrice,
      currency: values.currency,
      expenseDate: dayjs(values.expenseDate).format('YYYY-MM-DD'),
      note: values.note,
      exchangeRateOverride: manualExchangeRate
        ? values.exchangeRateOverride
        : undefined,
      taxInclusive: true,
    };

    try {
      if (editingFee?.id) {
        if (!editingFee.version)
          throw new Error('费用版本信息缺失，请刷新后重试');
        await orderFeeServiceUpdateFee(
          { orderId, id: editingFee.id },
          {
            ...body,
            orderId,
            id: editingFee.id,
            expectedVersion: editingFee.version,
          },
        );
        message.success('费用更新成功');
      } else {
        await orderFeeServiceAddFee(
          { orderId },
          { ...body, orderId, idempotencyKey: createIdempotencyKeyRef.current },
        );
        createIdempotencyKeyRef.current = globalThis.crypto.randomUUID();
        message.success('费用录入成功');
      }
      setModalOpen(false);
      receivableActionRef.current?.reload();
      payableActionRef.current?.reload();
      return true;
    } catch (error: any) {
      message.error(error.message || '保存费用失败');
      return false;
    }
  };

  const reloadFeeTables = () => {
    receivableActionRef.current?.reload();
    payableActionRef.current?.reload();
  };

  const requestReason = (
    title: string,
    onSubmit: (reason: string) => Promise<void>,
  ) => {
    let reason = '';
    modal.confirm({
      title,
      content: (
        <Input.TextArea
          autoFocus
          maxLength={500}
          showCount
          placeholder="请输入操作原因（必填）"
          onChange={(event) => {
            reason = event.target.value.trim();
          }}
        />
      ),
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        if (!reason) {
          message.warning('请输入操作原因');
          throw new Error('操作原因不能为空');
        }
        await onSubmit(reason);
      },
    });
  };

  const handleCancelFee = (fee: API.OrderFee) => {
    const feeId = fee.id;
    const version = fee.version;
    if (!orderId || !feeId || !version) return;
    requestReason('确认作废该笔费用？', async (reason) => {
      await orderFeeServiceRemoveFee({
        orderId,
        id: feeId,
        expectedVersion: version,
        reason,
      });
      message.success('费用已作废并保留历史记录');
      reloadFeeTables();
    });
  };

  const handleConfirmFee = async (fee: API.OrderFee) => {
    if (!orderId || !fee.id || !fee.version) return;
    try {
      await orderFeeServiceConfirmFee(
        { orderId, id: fee.id },
        { orderId, id: fee.id, expectedVersion: fee.version },
      );
      message.success('费用已确认，可以进入账单');
      reloadFeeTables();
    } catch (error: any) {
      message.error(error.message || '确认费用失败');
    }
  };

  const handleReopenFee = (fee: API.OrderFee) => {
    const feeId = fee.id;
    const version = fee.version;
    if (!orderId || !feeId || !version) return;
    requestReason('撤回费用确认？', async (reason) => {
      await orderFeeServiceReopenFee(
        { orderId, id: feeId },
        { orderId, id: feeId, expectedVersion: version, reason },
      );
      message.success('费用已撤回为草稿');
      reloadFeeTables();
    });
  };

  const getTableColumns = (direction: number): ProColumns<API.OrderFee>[] => [
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => {
        if (feeStatusCode(record.status) === FEE_CONFIRMED)
          return <Tag color="green">已确认</Tag>;
        if (feeStatusCode(record.status) === FEE_BILLED)
          return <Tag color="blue">已进账单</Tag>;
        if (feeStatusCode(record.status) === FEE_CANCELLED)
          return <Tag>已作废</Tag>;
        return <Tag color="gold">草稿</Tag>;
      },
    },
    {
      title: '费用代码',
      dataIndex: 'feeCode',
      width: 120,
      copyable: true,
      render: (_, record) => record.feeCode || '-',
    },
    {
      title: '费用名称',
      dataIndex: 'feeName',
      width: 140,
      render: (_, record) => record.feeName || '-',
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 180,
      ellipsis: true,
      render: (_, record) => record.settlementPartyName || '-',
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 80,
      render: (_, record) => <Tag color="blue">{record.currency}</Tag>,
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      width: 100,
      align: 'right',
      render: (_, record) => trimExactDecimal(record.unitPrice),
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      width: 80,
      align: 'right',
      render: (_, record) => trimExactDecimal(record.quantity),
    },
    {
      title: '计费单位',
      dataIndex: 'billingUnit',
      width: 90,
      render: (_, record) => record.billingUnit || '-',
    },
    {
      title: '总金额',
      dataIndex: 'totalAmount',
      width: 130,
      align: 'right',
      render: (_, record) => (
        <span
          style={{
            fontWeight: 600,
            color: direction === RECEIVABLE ? '#1677ff' : '#fa8c16',
          }}
        >
          {trimExactDecimal(record.totalAmount)} {record.currency}
        </span>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 100,
      align: 'right',
      render: (_, record) => (
        <Space size={4}>
          <span>{trimExactDecimal(record.exchangeRate)}</span>
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
      width: 220,
      render: (_, record) => record.expenseDate || '-',
    },
    {
      title: '备注',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 110,
      fixed: 'right',
      render: (_, record) => [
        feeStatusCode(record.status) === FEE_DRAFT && (
          <Button
            key="edit"
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openFeeModal(direction, record)}
          >
            编辑
          </Button>
        ),
        feeStatusCode(record.status) === FEE_DRAFT && (
          <Popconfirm
            key="confirm"
            title="确认后该费用才能进入账单，确定继续？"
            onConfirm={() => handleConfirmFee(record)}
          >
            <Button type="link" size="small">
              确认
            </Button>
          </Popconfirm>
        ),
        feeStatusCode(record.status) === FEE_CONFIRMED && (
          <Button
            key="reopen"
            type="link"
            size="small"
            onClick={() => handleReopenFee(record)}
          >
            撤回
          </Button>
        ),
        (feeStatusCode(record.status) === FEE_DRAFT ||
          feeStatusCode(record.status) === FEE_CONFIRMED) && (
          <Button
            key="cancel"
            type="link"
            size="small"
            danger
            onClick={() => handleCancelFee(record)}
          >
            作废
          </Button>
        ),
      ],
    },
  ];

  if (loading) {
    return (
      <div
        style={{
          textAlign: 'center',
          padding: '120px 0',
          background: '#f5f7fa',
          minHeight: '100vh',
        }}
      >
        <Spin size="large" tip="正在加载费用工作台..." />
      </div>
    );
  }

  if (!order) {
    return (
      <div style={{ padding: 48, background: '#f5f7fa', minHeight: '100vh' }}>
        <Card
          bordered={false}
          style={{ borderRadius: 8, textAlign: 'center', padding: 32 }}
        >
          <Empty description="未找到对应的订单档案" />
          <Button
            type="primary"
            onClick={() => history.push(`/orders/${kind}`)}
            style={{ marginTop: 16 }}
          >
            返回订单列表
          </Button>
        </Card>
      </div>
    );
  }

  const profitCny = receivableSummary.totalAmount - payableSummary.totalAmount;
  const profitRate =
    receivableSummary.totalAmount > 0
      ? ((profitCny / receivableSummary.totalAmount) * 100).toFixed(1)
      : '0.0';

  return (
    <div
      style={{ padding: '0 0 40px', background: '#f5f7fa', minHeight: '100vh' }}
    >
      {/* 顶部面包屑与快捷返回 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '10px 24px',
          background: '#ffffff',
          borderBottom: '1px solid #e2e8f0',
          marginBottom: 16,
        }}
      >
        <Space size={8}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => history.push(`/orders/${kind}/${orderId}`)}
          >
            返回订单详情
          </Button>
          <span style={{ color: '#cbd5e1' }}>|</span>
          <span style={{ color: '#64748b' }}>{config.title}</span>
          <span>&gt;</span>
          <a
            style={{
              color: '#1677ff',
              fontWeight: 600,
              fontFamily: 'monospace',
            }}
            onClick={() => history.push(`/orders/${kind}/${orderId}`)}
          >
            {order.orderNo || order.id}
          </a>
          <span>&gt;</span>
          <span style={{ fontWeight: 600, color: '#0f172a' }}>费用录入</span>
          {order.canModify === false && order.status !== 'DRAFT' && (
            <Tag color="warning" icon={<LockOutlined />}>
              已锁单
            </Tag>
          )}
        </Space>

        <Space size={8}>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void loadData();
              receivableActionRef.current?.reload();
              payableActionRef.current?.reload();
            }}
          >
            刷新数据
          </Button>
          <Button
            type="primary"
            onClick={() => history.push(`/orders/${kind}/${orderId}`)}
          >
            回到订单详情
          </Button>
        </Space>
      </div>

      <div style={{ maxWidth: 1440, margin: '0 auto', padding: '0 24px' }}>
        {/* 1. 基础信息卡片 */}
        <SectionCard title="订单基础信息" style={{ marginBottom: 16 }}>
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, md: 3, lg: 4, xl: 4 }}
          >
            <Descriptions.Item label="订单编号">
              <a
                style={{
                  fontWeight: 600,
                  color: '#1677ff',
                  fontFamily: 'monospace',
                }}
                onClick={() => history.push(`/orders/${kind}/${orderId}`)}
              >
                {order.orderNo || order.id}
              </a>
            </Descriptions.Item>
            <Descriptions.Item label="委托单位">
              {order.customerId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="业务类型">
              {config.title}
            </Descriptions.Item>
            <Descriptions.Item label="贸易条款">
              {order.tradeTerm ? 'FOB / CIF' : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="主单号 (MBL)">
              {order.shippingDocuments?.[0]?.masterNo || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="船名航次">
              {order.vesselVoyage || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="起运港 (POL)">
              {order.originLocationId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="目的港 (POD)">
              {order.destinationLocationId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="承运人 (船司)">
              {order.carrierId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="订舱代理">
              {order.bookingAgentId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="ETD">
              {order.etd ? dayjs(order.etd).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="件重尺">
              {order.totalPackages || '-'} 件 /{' '}
              {order.totalGrossWeightKg || '-'} kg /{' '}
              {order.totalVolumeCbm || '-'} m³
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 2. 费用汇总统计指标卡 */}
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col xs={24} sm={8}>
            <Card
              bordered={false}
              style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
            >
              <Statistic
                title={<span style={{ color: '#64748b' }}>应收总计</span>}
                value={receivableSummary.totalAmount}
                precision={2}
                prefix="¥"
                valueStyle={{ color: '#1677ff', fontWeight: 700 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card
              bordered={false}
              style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
            >
              <Statistic
                title={<span style={{ color: '#64748b' }}>应付总计</span>}
                value={payableSummary.totalAmount}
                precision={2}
                prefix="¥"
                valueStyle={{ color: '#fa8c16', fontWeight: 700 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card
              bordered={false}
              style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
            >
              <Statistic
                title={
                  <Space>
                    <span style={{ color: '#64748b' }}>预计毛利</span>
                    <Tag color={profitCny >= 0 ? 'success' : 'error'}>
                      {profitRate}%
                    </Tag>
                  </Space>
                }
                value={profitCny}
                precision={2}
                prefix="¥"
                valueStyle={{
                  color: profitCny >= 0 ? '#52c41a' : '#ff4d4f',
                  fontWeight: 700,
                }}
              />
            </Card>
          </Col>
        </Row>

        {/* 3. 费用表格工作区 */}
        <Tabs
          type="card"
          defaultActiveKey="receivable"
          items={[
            {
              key: 'receivable',
              label: (
                <Space>
                  <span>应收费用</span>
                  <Tag color="blue">{receivableSummary.count}</Tag>
                </Space>
              ),
              children: (
                <ProTable<API.OrderFee>
                  actionRef={receivableActionRef}
                  rowKey="id"
                  search={false}
                  bordered
                  pagination={false}
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
                  toolBarRender={() =>
                    [
                      access.canCreateFinanceBills && (
                        <Button
                          key="bill"
                          icon={<FileDoneOutlined />}
                          disabled={selectedReceivableFeeIds.length === 0}
                          onClick={() => {
                            setBillWorkbenchFeeIds(
                              selectedReceivableFeeIds.map(String),
                            );
                            setBillWorkbenchOpen(true);
                          }}
                        >
                          生成账单（{selectedReceivableFeeIds.length}）
                        </Button>
                      ),
                      <Button
                        key="add"
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => openFeeModal(RECEIVABLE)}
                      >
                        + 新增应收费用
                      </Button>,
                    ].filter(Boolean)
                  }
                  request={async () => {
                    if (!orderId) return { data: [], success: true };
                    const res = await orderFeeServiceListFees({ orderId });
                    const rItems = (res.data ?? []).filter(
                      (f) => feeDirectionCode(f.direction) === RECEIVABLE,
                    );
                    const activeItems = rItems.filter(
                      (f) => feeStatusCode(f.status) !== FEE_CANCELLED,
                    );
                    const total = activeItems.reduce(
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
                  }}
                  columns={getTableColumns(RECEIVABLE)}
                />
              ),
            },
            {
              key: 'payable',
              label: (
                <Space>
                  <span>应付费用</span>
                  <Tag color="orange">{payableSummary.count}</Tag>
                </Space>
              ),
              children: (
                <ProTable<API.OrderFee>
                  actionRef={payableActionRef}
                  rowKey="id"
                  search={false}
                  bordered
                  pagination={false}
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
                  toolBarRender={() =>
                    [
                      access.canCreateFinanceBills && (
                        <Button
                          key="bill"
                          icon={<FileDoneOutlined />}
                          disabled={selectedPayableFeeIds.length === 0}
                          onClick={() => {
                            setBillWorkbenchFeeIds(
                              selectedPayableFeeIds.map(String),
                            );
                            setBillWorkbenchOpen(true);
                          }}
                        >
                          生成账单（{selectedPayableFeeIds.length}）
                        </Button>
                      ),
                      <Button
                        key="add"
                        type="primary"
                        icon={<PlusOutlined />}
                        style={{
                          backgroundColor: '#fa8c16',
                          borderColor: '#fa8c16',
                        }}
                        onClick={() => openFeeModal(PAYABLE)}
                      >
                        + 新增应付费用
                      </Button>,
                    ].filter(Boolean)
                  }
                  request={async () => {
                    if (!orderId) return { data: [], success: true };
                    const res = await orderFeeServiceListFees({ orderId });
                    const pItems = (res.data ?? []).filter(
                      (f) => feeDirectionCode(f.direction) === PAYABLE,
                    );
                    const activeItems = pItems.filter(
                      (f) => feeStatusCode(f.status) !== FEE_CANCELLED,
                    );
                    const total = activeItems.reduce(
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
                  }}
                  columns={getTableColumns(PAYABLE)}
                />
              ),
            },
          ]}
        />
      </div>

      <BillCreationWorkbench
        open={billWorkbenchOpen}
        initialFeeIds={billWorkbenchFeeIds}
        sourceLabel={`订单 ${order.orderNo || order.id}`}
        onClose={() => setBillWorkbenchOpen(false)}
        onCreated={() => {
          setSelectedReceivableFeeIds([]);
          setSelectedPayableFeeIds([]);
          receivableActionRef.current?.reload();
          payableActionRef.current?.reload();
        }}
      />

      {/* 4. 费用录入/编辑 ModalForm */}
      <ModalForm<FeeFormValues>
        title={
          editingFee
            ? `编辑${modalDirection === RECEIVABLE ? '应收' : '应付'}费用`
            : `新增${modalDirection === RECEIVABLE ? '应收' : '应付'}费用`
        }
        open={modalOpen}
        formRef={formRef}
        onOpenChange={setModalOpen}
        onFinish={handleModalSubmit}
        onValuesChange={handleValuesChange}
        width={680}
        modalProps={{ destroyOnClose: true }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <ProFormSelect
              name="feeSettingId"
              label="费用项目"
              rules={[{ required: true, message: '请选择费用项目' }]}
              options={feeSettings.map((item) => ({
                label: `${item.nameZh || item.nameEn || item.feeCode} (${item.feeCode})`,
                value: item.id ?? '',
              }))}
              fieldProps={{
                showSearch: true,
                dropdownRender: (menu) => (
                  <>
                    {menu}
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
                        void handleOpenQuickAddFee();
                      }}
                    >
                      <PlusOutlined /> 快捷新增费用科目
                    </div>
                  </>
                ),
                onChange: (val) => {
                  const setting = feeSettings.find((item) => item.id === val);
                  setSelectedFeeSetting(setting);
                  if (setting?.defaultBillingUnitId) {
                    formRef.current?.setFieldValue(
                      'billingUnitId',
                      setting.defaultBillingUnitId,
                    );
                  }
                  if (setting?.defaultCurrency) {
                    formRef.current?.setFieldValue(
                      'currency',
                      setting.defaultCurrency,
                    );
                  }
                  handleValuesChange();
                },
              }}
            />
          </Col>
          <Col span={12}>
            <ProFormSelect
              name="settlementPartyId"
              label="结算单位"
              rules={[{ required: true, message: '请选择结算单位' }]}
              options={settlementParties.map((item) => ({
                label: item.name ?? '',
                value: item.id ?? '',
              }))}
              fieldProps={{
                showSearch: true,
                dropdownRender: (menu) => (
                  <>
                    {menu}
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
                        handleOpenQuickAddPartner();
                      }}
                    >
                      <PlusOutlined /> 快捷新建往来单位
                    </div>
                  </>
                ),
              }}
            />
          </Col>
          <Col span={8}>
            <ProFormSelect
              name="currency"
              label="币种"
              rules={[{ required: true, message: '请选择币种' }]}
              options={currencies.map((c) => ({
                label: `${c.code} (${c.name})`,
                value: c.code ?? '',
              }))}
            />
          </Col>
          <Col span={8}>
            <ProFormText
              name="unitPrice"
              label="单价"
              rules={[
                { required: true, message: '请输入单价' },
                {
                  validator: positiveDecimalRule(
                    quantityOrPricePattern,
                    '单价格式不正确',
                  ),
                },
              ]}
              placeholder="0.00"
            />
          </Col>
          <Col span={8}>
            <ProFormText
              name="quantity"
              label="数量"
              rules={[
                { required: true, message: '请输入数量' },
                {
                  validator: positiveDecimalRule(
                    quantityOrPricePattern,
                    '数量格式不正确',
                  ),
                },
              ]}
              placeholder="1"
            />
          </Col>
          <Col span={12}>
            <ProFormSelect
              name="billingUnitId"
              label="计费单位"
              rules={[{ required: true, message: '请选择计费单位' }]}
              options={billingUnits.map((u) => ({
                label: `${u.name} (${u.code})`,
                value: u.id ?? '',
              }))}
            />
          </Col>
          <Col span={12}>
            <ProFormDatePicker
              name="expenseDate"
              label="发生日期"
              rules={[{ required: true, message: '请选择发生日期' }]}
              fieldProps={{ style: { width: '100%' } }}
            />
          </Col>

          {/* 汇率与金额计算预览 */}
          <Col span={24}>
            <Card
              size="small"
              style={{ backgroundColor: '#f8fafc', marginBottom: 16 }}
            >
              <Space
                split={<span style={{ color: '#cbd5e1' }}>|</span>}
                size={16}
              >
                <div>
                  <Text type="secondary">费用金额：</Text>
                  <Text
                    strong
                    style={{
                      fontSize: 16,
                      color:
                        modalDirection === RECEIVABLE ? '#1677ff' : '#fa8c16',
                    }}
                  >
                    {totalPreview
                      ? `${formRef.current?.getFieldValue('currency') || ''} ${totalPreview}`
                      : '-'}
                  </Text>
                </div>
                <div>
                  <Text type="secondary">生效汇率：</Text>
                  {exchangeRateStatus === 'loading' && <Spin size="small" />}
                  {exchangeRateStatus === 'resolved' && (
                    <Text strong style={{ color: '#52c41a' }}>
                      {exchangeRatePreview}
                    </Text>
                  )}
                  {exchangeRateStatus === 'missing' && (
                    <Space size={4}>
                      <Tag color="error">汇率未配置</Tag>
                      <Button
                        type="link"
                        size="small"
                        onClick={() => setManualExchangeRate(true)}
                      >
                        手动输入
                      </Button>
                    </Space>
                  )}
                </div>
              </Space>
            </Card>
          </Col>

          {manualExchangeRate && (
            <Col span={24}>
              <ProFormText
                name="exchangeRateOverride"
                label="手动指定汇率 (对 CNY)"
                rules={[
                  { required: true, message: '请输入手动汇率' },
                  {
                    validator: positiveDecimalRule(
                      exchangeRatePattern,
                      '汇率格式不正确',
                    ),
                  },
                ]}
                placeholder="例如 7.2345"
              />
            </Col>
          )}

          <Col span={24}>
            <ProFormTextArea
              name="note"
              label="备注说明"
              placeholder="请输入费用相关备注（可选）"
              fieldProps={{ rows: 2, maxLength: 500, showCount: true }}
            />
          </Col>
        </Row>
      </ModalForm>

      {/* 快捷新增费用科目 Modal */}
      <Modal
        title="快捷新增费用科目"
        open={quickAddFeeModalOpen}
        confirmLoading={quickAddFeeSaving}
        okText="保存并选用"
        cancelText="取消"
        onOk={() => void handleSaveQuickAddFee()}
        onCancel={() => setQuickAddFeeModalOpen(false)}
        destroyOnClose
        width={580}
      >
        <Form form={quickAddFeeForm} layout="vertical" preserve={false}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="feeCode"
                label="科目代码"
                rules={[
                  { required: true, whitespace: true, message: '请输入科目代码' },
                  { max: 30, message: '不能超过 30 字符' },
                ]}
              >
                <Input placeholder="例如：THC、OFRT、CUSTOMS" maxLength={30} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="nameZh"
                label="科目中文名称"
                rules={[
                  { required: true, whitespace: true, message: '请输入中文名称' },
                  { max: 100, message: '不能超过 100 字符' },
                ]}
              >
                <Input placeholder="例如：码头操作费、海运费" maxLength={100} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="nameEn" label="英文名称（选填）">
                <Input placeholder="例如：Terminal Handling Charge" maxLength={100} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="defaultCurrency"
                label="默认币种"
                rules={[{ required: true, message: '请选择币种' }]}
              >
                <Select
                  options={currencies.map((c) => ({
                    label: `${c.code} (${c.name})`,
                    value: c.code ?? '',
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="billingUnitId"
                label="默认计费单位"
                rules={[{ required: true, message: '请选择计费单位' }]}
              >
                <Select
                  options={billingUnits.map((u) => ({
                    label: u.name ?? '',
                    value: u.id ?? '',
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="taxRate"
                label="默认增值税税率"
                rules={[{ required: true, message: '请选择税率' }]}
              >
                <Select
                  options={[
                    { label: '0% (零税率/免税)', value: '0' },
                    { label: '6% (现代服务业/货运代理)', value: '0.06' },
                    { label: '9% (基础交通运输)', value: '0.09' },
                    { label: '13% (商品贸易/修箱)', value: '0.13' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item name="taxableServiceId" label="应税服务类别">
                <Select
                  placeholder="选择税目分类"
                  options={taxableServices.map((s) => ({
                    label: s.goodsCode
                      ? `${s.name} (${s.goodsCode})`
                      : s.name || '',
                    value: s.id ?? '',
                  }))}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      {/* 快捷新建往来单位 Modal */}
      <Modal
        title="快捷新建往来单位"
        open={quickAddPartnerModalOpen}
        confirmLoading={quickAddPartnerSaving}
        okText="保存并选用"
        cancelText="取消"
        onOk={() => void handleSaveQuickAddPartner()}
        onCancel={() => setQuickAddPartnerModalOpen(false)}
        destroyOnClose
        width={540}
      >
        <Form form={quickAddPartnerForm} layout="vertical" preserve={false}>
          <Row gutter={16}>
            <Col span={24}>
              <Form.Item
                name="legalName"
                label="单位全称"
                rules={[
                  { required: true, whitespace: true, message: '请输入单位全称' },
                  { max: 200, message: '不能超过 200 字符' },
                ]}
              >
                <Input placeholder="工商登记全称或客商名称" maxLength={200} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="code" label="客商代码（选填）">
                <Input placeholder="例如：COSCO、SITC（留空自动生成）" maxLength={50} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="unifiedSocialCreditCode" label="统一社会信用代码（选填）">
                <Input placeholder="18 位税号" maxLength={50} />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item
                name="roles"
                label="客商类型"
                rules={[{ required: true, message: '请选择至少一种类型' }]}
              >
                <Select
                  mode="multiple"
                  options={[
                    { label: '客户 (委托单位/收发通)', value: 'CUSTOMER' },
                    { label: '供应商 (船东/车队/报关行/码头)', value: 'SUPPLIER' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
}
