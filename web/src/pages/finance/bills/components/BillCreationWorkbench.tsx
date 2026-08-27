import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  DeleteOutlined,
  FileDoneOutlined,
  PlusOutlined,
  ReloadOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { history, useAccess } from '@umijs/max';
import {
  Alert,
  App,
  AutoComplete,
  Button,
  Card,
  Checkbox,
  Col,
  DatePicker,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Result,
  Row,
  Select,
  Space,
  Steps,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  settlementServiceConfirmBillBatch,
  settlementServiceCreateBillBatch,
  settlementServiceListFeeLedger,
  settlementServicePreviewBillBatch,
} from '@/services/roncin/settlementService';
import {
  partnerServiceCreatePartnerInvoiceProfile,
  partnerServiceListPartnerInvoiceProfiles,
} from '@/services/roncin/partnerService';

const { Text, Title } = Typography;

type GroupFormValue = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
};

type WorkbenchFormValue = {
  groups: GroupFormValue[];
};

type QuickAddInvoiceProfileFormValue = {
  invoiceTitle: string;
  taxpayerIdentificationNo: string;
  defaultInvoiceType: 'NORMAL' | 'SPECIAL';
  bankName?: string;
  bankAccount?: string;
  registeredAddress?: string;
  registeredPhone?: string;
  isDefault?: boolean;
};

type RequestError = Error & {
  data?: { reason?: string; message?: string };
  response?: { data?: { reason?: string; message?: string } };
};

export type BillCreationWorkbenchProps = {
  open: boolean;
  initialFeeIds?: string[];
  sourceLabel?: string;
  onClose: () => void;
  onCreated?: (batch: API.FinanceBillBatch) => void;
};

function directionText(value?: string) {
  return value === 'RECEIVABLE' ? '应收' : '应付';
}

function requestReason(error: RequestError) {
  return error.data?.reason || error.response?.data?.reason;
}

function requestMessage(error: RequestError, fallback: string) {
  const msg =
    error.response?.data?.message ||
    error.data?.message ||
    (error.message && !error.message.toLowerCase().includes('status code')
      ? error.message
      : '');
  if (msg) return msg;
  const reason = requestReason(error);
  if (reason === 'FINANCE_BILL_FEE_INVALID') {
    return '所选费用必须为已确认状态且尚未进入其他账单';
  }
  if (reason === 'FINANCE_BILL_PREVIEW_STALE') {
    return '费用已发生变化，请重新预览后再生成账单';
  }
  return fallback;
}

export default function BillCreationWorkbench({
  open,
  initialFeeIds = [],
  sourceLabel,
  onClose,
  onCreated,
}: BillCreationWorkbenchProps) {
  const { message } = App.useApp();
  const access = useAccess();
  const [form] = Form.useForm<WorkbenchFormValue>();
  const [quickAddForm] = Form.useForm<QuickAddInvoiceProfileFormValue>();
  const [current, setCurrent] = useState(0);
  const [selectedFeeIds, setSelectedFeeIds] = useState<React.Key[]>([]);
  const [splitByOrder, setSplitByOrder] = useState(false);
  const [splitByTaxRate, setSplitByTaxRate] = useState(false);
  const [preview, setPreview] = useState<API.PreviewBillBatchResponse>();
  const [invoiceProfilesMap, setInvoiceProfilesMap] = useState<
    Record<string, API.PartnerInvoiceProfile[]>
  >({});
  const [result, setResult] = useState<API.FinanceBillBatch>();
  const [loading, setLoading] = useState(false);
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [confirming, setConfirming] = useState(false);
  const previewInitKeyRef = useRef<string | undefined>(undefined);
  const previewErrorKeyRef = useRef<string | undefined>(undefined);
  const previewPendingRef = useRef<{
    key: string;
    promise: Promise<boolean>;
  } | null>(null);

  // 就地快捷新增客商开票抬头状态
  const [quickAddOpen, setQuickAddOpen] = useState(false);
  const [quickAddGroupIndex, setQuickAddGroupIndex] = useState(0);
  const [quickAddPartnerId, setQuickAddPartnerId] = useState('');
  const [quickAddPartnerName, setQuickAddPartnerName] = useState('');
  const [quickAddSaving, setQuickAddSaving] = useState(false);

  const initialFeeKey = useMemo(
    () => (initialFeeIds || []).filter(Boolean).join('|'),
    [initialFeeIds],
  );
  const fixedSelection = initialFeeKey.length > 0;

  const selectedIds = useMemo(
    () => selectedFeeIds.map(String).filter(Boolean),
    [selectedFeeIds],
  );

  const selectedIdsRef = useRef<string[]>([]);
  selectedIdsRef.current = selectedIds;
  const splitByOrderRef = useRef(splitByOrder);
  splitByOrderRef.current = splitByOrder;
  const splitByTaxRateRef = useRef(splitByTaxRate);
  splitByTaxRateRef.current = splitByTaxRate;

  const loadPreview = useCallback(
    async (
      overrideIds?: string[],
      policyOverride?: { splitByOrder: boolean; splitByTaxRate: boolean },
    ) => {
      const ids = overrideIds ?? selectedIdsRef.current;
      if (ids.length === 0) {
        message.warning('请至少选择一笔已确认且未建立账单的费用');
        return false;
      }
      setLoading(true);
      try {
        const policy = policyOverride ?? {
          splitByOrder: splitByOrderRef.current,
          splitByTaxRate: splitByTaxRateRef.current,
        };
        const response = await settlementServicePreviewBillBatch(
          {
            feeIds: ids,
            groupingPolicy: policy,
          },
          { skipErrorHandler: true },
        );
        const groups = response.data || [];
        if (!response.previewToken || groups.length === 0) {
          throw new Error('服务端未返回有效的拆单预览');
        }
        previewErrorKeyRef.current = undefined;
        setPreview(response);

        // 异步批量查询各结算单位维护的全部开票抬头资料
        const uniquePartyIds = Array.from(
          new Set(
            groups
              .map((g) => g.settlementPartyId)
              .filter((id): id is string => Boolean(id)),
          ),
        );
        const profilesMap: Record<string, API.PartnerInvoiceProfile[]> = {};
        await Promise.all(
          uniquePartyIds.map(async (partyId) => {
            try {
              const res = await partnerServiceListPartnerInvoiceProfiles(
                { partnerId: partyId },
                { skipErrorHandler: true },
              );
              profilesMap[partyId] = res.data || [];
            } catch {
              profilesMap[partyId] = [];
            }
          }),
        );
        setInvoiceProfilesMap(profilesMap);

        form.setFieldsValue({
          groups: groups.map((group) => {
            const profiles = profilesMap[group.settlementPartyId || ''] || [];
            const defaultProfile =
              profiles.find((p) => p.isDefault && p.enabled !== false) ||
              profiles.find((p) => p.enabled !== false);
            return {
              statementTitle:
                defaultProfile?.invoiceTitle || group.settlementPartyName || '',
              billDate: dayjs(),
              paymentTermsDays: undefined,
              note: undefined,
            };
          }),
        });
        return true;
      } catch (rawError: unknown) {
        const error = rawError as RequestError;
        const errorKey = `${ids.join('|')}:${requestReason(error) || requestMessage(error, '拆单预览失败')}`;
        if (previewErrorKeyRef.current !== errorKey) {
          previewErrorKeyRef.current = errorKey;
          message.error(requestMessage(error, '拆单预览失败'));
        }
        return false;
      } finally {
        setLoading(false);
      }
    },
    [form, message],
  );

  // 初始化或当从业务页面进入时，自动快速预览并直达账单资料页
  useEffect(() => {
    if (!open) {
      previewInitKeyRef.current = undefined;
      previewErrorKeyRef.current = undefined;
      previewPendingRef.current = null;
      return;
    }
    const initialIds = initialFeeKey ? initialFeeKey.split('|').filter(Boolean) : [];
    const initKey = `${initialFeeKey}:${open ? 'open' : 'closed'}`;
    let cancelled = false;
    const pending = previewPendingRef.current;
    if (previewInitKeyRef.current === initKey && pending?.key === initKey) {
      void pending.promise.then((ok) => {
        if (cancelled) return;
        setCurrent(ok ? 2 : 0);
      });
      return () => {
        cancelled = true;
      };
    }
    previewInitKeyRef.current = initKey;
    setSelectedFeeIds(initialIds);
    setSplitByOrder(false);
    setSplitByTaxRate(false);
    setPreview(undefined);
    setResult(undefined);
    setLoading(false);
    setConfirming(false);
    setIdempotencyKey(globalThis.crypto.randomUUID());
    form.resetFields();

    if (initialIds.length > 0) {
      // 极速模式：从单票/多选费用带入时，直接拉取预览并切到账单资料页
      const previewPromise = loadPreview(initialIds, {
        splitByOrder: false,
        splitByTaxRate: false,
      });
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
  }, [open, initialFeeKey, form, loadPreview]);

  // 从预览明细中即时剔除误选行
  const handleRemoveFee = async (feeId?: string) => {
    if (!feeId) return;
    const nextIds = selectedIds.filter((id) => id !== feeId);
    if (nextIds.length === 0) {
      message.info('已移除所有费用，请重新选择');
      setSelectedFeeIds([]);
      setPreview(undefined);
      setCurrent(0);
      return;
    }
    setSelectedFeeIds(nextIds);
    message.success('已从本次建单中移除该费用');
    await loadPreview(nextIds);
  };

  // 打开快捷新增客商抬头弹窗
  const handleOpenQuickAddProfile = (
    groupIndex: number,
    partnerId?: string,
    partnerName?: string,
  ) => {
    if (!partnerId) {
      message.warning('当前账单缺少结算单位关联，无法维护抬头');
      return;
    }
    setQuickAddGroupIndex(groupIndex);
    setQuickAddPartnerId(partnerId);
    setQuickAddPartnerName(partnerName || '');
    quickAddForm.resetFields();
    quickAddForm.setFieldsValue({
      invoiceTitle: partnerName || '',
      taxpayerIdentificationNo: '',
      defaultInvoiceType: 'NORMAL',
      isDefault: true,
    });
    setQuickAddOpen(true);
  };

  // 提交保存快捷开票抬头
  const handleSaveQuickAddProfile = async () => {
    const values = await quickAddForm.validateFields();
    setQuickAddSaving(true);
    try {
      const res = await partnerServiceCreatePartnerInvoiceProfile(
        { partnerId: quickAddPartnerId },
        {
          partnerId: quickAddPartnerId,
          invoiceTitle: values.invoiceTitle.trim(),
          taxpayerIdentificationNo: values.taxpayerIdentificationNo
            .trim()
            .toUpperCase(),
          bankName: values.bankName?.trim() || undefined,
          bankAccount: values.bankAccount?.trim() || undefined,
          registeredAddress: values.registeredAddress?.trim() || undefined,
          registeredPhone: values.registeredPhone?.trim() || undefined,
          defaultInvoiceType: values.defaultInvoiceType,
          isDefault: values.isDefault ?? false,
        },
        { skipErrorHandler: true },
      );
      const created = res.data;
      if (!created) throw new Error('服务端未返回新增抬头数据');

      // 更新客商开票资料缓存
      setInvoiceProfilesMap((prev) => {
        const existing = prev[quickAddPartnerId] || [];
        return {
          ...prev,
          [quickAddPartnerId]: [
            created,
            ...existing.map((profile) =>
              created.isDefault ? { ...profile, isDefault: false } : profile,
            ),
          ],
        };
      });

      // 自动回显并选中到当前账单的 statementTitle 表单字段
      const currentGroups = form.getFieldValue('groups') || [];
      if (currentGroups[quickAddGroupIndex]) {
        currentGroups[quickAddGroupIndex] = {
          ...currentGroups[quickAddGroupIndex],
          statementTitle: created.invoiceTitle || '',
        };
        form.setFieldsValue({ groups: [...currentGroups] });
      }

      message.success(`已为【${quickAddPartnerName}】新增开票抬头并自动选中`);
      setQuickAddOpen(false);
    } catch (rawError: unknown) {
      const error = rawError as RequestError;
      message.error(requestMessage(error, '新增开票抬头失败'));
    } finally {
      setQuickAddSaving(false);
    }
  };

  const baseFeeColumns: ProColumns<API.FeeLedgerItem>[] = [
    {
      title: '订单编号',
      dataIndex: 'orderNo',
      width: 140,
      copyable: true,
      search: false,
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 70,
      valueType: 'select',
      search: false,
      valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
      render: (_, row) => (
        <Tag color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
          {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 85,
      search: false,
      render: (_, row) => {
        if (row.status === 'CONFIRMED') {
          return <Tag color="blue">已确认</Tag>;
        }
        if (row.status === 'BILLED') {
          return <Tag color="green">已开账</Tag>;
        }
        if (row.status === 'CANCELLED') {
          return <Tag color="red">已作废</Tag>;
        }
        return <Tag>草稿</Tag>;
      },
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 180,
      ellipsis: true,
      search: false,
    },
    { title: '费用名称', dataIndex: 'feeName', width: 130, search: false },
    {
      title: '税率',
      dataIndex: 'taxRate',
      width: 75,
      align: 'right',
      search: false,
      renderText: (value) => (value == null ? '-' : `${Number(value)}%`),
    },
    {
      title: '金额',
      dataIndex: 'totalAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, row) => (
        <Text strong>
          {row.totalAmount} {row.currency}
        </Text>
      ),
    },
    {
      title: '费用日期',
      dataIndex: 'expenseDate',
      width: 110,
      search: false,
    },
  ];

  const selectionFeeColumns: ProColumns<API.FeeLedgerItem>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '订单号、费用或结算单位' },
    },
    {
      title: '方向',
      dataIndex: 'direction',
      hideInTable: true,
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
    },
    ...baseFeeColumns,
  ];

  const getPreviewFeeColumns = (
    canRemove = true,
  ): ProColumns<API.FeeLedgerItem>[] => [
    ...baseFeeColumns.filter((col) => col.dataIndex !== 'direction'),
    ...(canRemove
      ? [
          {
            title: '操作',
            width: 75,
            align: 'center' as const,
            search: false,
            render: (_: unknown, row: API.FeeLedgerItem) => (
              <Popconfirm
                title="确认将该笔费用移出本次建单？"
                onConfirm={() => void handleRemoveFee(row.id)}
                okText="移出"
                cancelText="取消"
              >
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  style={{ padding: 0 }}
                >
                  移出
                </Button>
              </Popconfirm>
            ),
          },
        ]
      : []),
  ];

  const next = async () => {
    if (current === 0) {
      if (selectedIds.length === 0) {
        message.warning('请至少选择一笔已确认且未建立账单的费用');
        return;
      }
      setCurrent(1);
      return;
    }
    if (current === 1) {
      if (await loadPreview()) setCurrent(2);
    }
  };

  const createBatch = async () => {
    if (!preview?.previewToken || !preview.data?.length) {
      message.warning('账单预览快照已失效或为空，请重新预览');
      return;
    }
    const values = await form.validateFields();
    setLoading(true);
    try {
      const response = await settlementServiceCreateBillBatch(
        {
          feeIds: selectedIds,
          groupingPolicy: { splitByOrder, splitByTaxRate },
          previewToken: preview.previewToken,
          idempotencyKey,
          groups: preview.data.map((group, index) => {
            const value = values.groups[index];
            return {
              groupKey: group.groupKey || '',
              statementTitle: value.statementTitle.trim(),
              billDate: value.billDate.format('YYYY-MM-DD'),
              paymentTermsDays: value.paymentTermsDays,
              note: value.note?.trim() || undefined,
            };
          }),
        },
        { skipErrorHandler: true },
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
      );
      if (response.data) setResult(response.data);
      message.success('本批账单已全部确认，可以进入开票、收付款和核销流程');
      onCreated?.(response.data || result);
    } catch (error: any) {
      message.error(error.message || '批量确认账单失败');
    } finally {
      setConfirming(false);
    }
  };

  const footer = (
    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
      <Button onClick={onClose}>{current === 3 ? '关闭' : '取消'}</Button>
      {current < 3 && (
        <Space>
          {current > 0 && (
            <Button
              icon={<ArrowLeftOutlined />}
              disabled={loading}
              onClick={() => setCurrent((value) => value - 1)}
            >
              上一步
            </Button>
          )}
          {current < 2 && (
            <Button
              type="primary"
              loading={loading}
              onClick={() => void next()}
            >
              下一步
            </Button>
          )}
          {current === 2 && (
            <Button
              type="primary"
              icon={<FileDoneOutlined />}
              loading={loading}
              onClick={() => void createBatch()}
            >
              原子生成 {preview?.data?.length || 0} 张账单
            </Button>
          )}
        </Space>
      )}
    </div>
  );

  return (
    <>
      <Drawer
        title="费用批量转账单"
        open={open}
        width="min(1280px, 96vw)"
      destroyOnHidden
      maskClosable={false}
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

      {current === 0 &&
        (fixedSelection ? (
          <Card>
            <Alert
              type="info"
              showIcon
              message={`已从${sourceLabel || '业务页面'}带入 ${selectedIds.length} 笔已确认费用`}
              description="费用状态、结算维度和金额快照将在预览及最终建单事务中由服务端再次校验。"
            />
          </Card>
        ) : (
          <ProTable<API.FeeLedgerItem>
            rowKey="id"
            headerTitle="选择待结算费用"
            columns={selectionFeeColumns}
            size="small"
            bordered
            options={false}
            pagination={{ defaultPageSize: 15, showSizeChanger: true }}
            rowSelection={{
              selectedRowKeys: selectedFeeIds,
              preserveSelectedRowKeys: true,
              onChange: setSelectedFeeIds,
              getCheckboxProps: (record) => {
                const isSelectable =
                  record.status === 'CONFIRMED' && !record.billNo;
                return {
                  disabled: !isSelectable,
                  title: !isSelectable
                    ? record.billNo
                      ? `已进入账单 ${record.billNo}`
                      : '只有已确认且未入账单的费用方可创建账单'
                    : undefined,
                };
              },
            }}
            tableAlertRender={({ selectedRowKeys }) => (
              <Text>已选择 {selectedRowKeys.length} 笔已确认费用</Text>
            )}
            request={async (params) => {
              const response = await settlementServiceListFeeLedger({
                page: params.current,
                pageSize: params.pageSize,
                keyword: params.keyword,
                direction: params.direction,
                status: 'CONFIRMED',
              });
              return {
                data: response.data || [],
                total: Number(response.total || 0),
                success: response.success ?? true,
              };
            }}
          />
        ))}

      {current === 1 && (
        <Card>
          <Title level={5}>拆单维度</Title>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 20 }}
            message="收付方向、结算单位、原币和本币始终是强制拆单维度"
            description="不同强制维度的费用绝不会进入同一张账单。下面两个开关只控制额外拆分，不会放宽服务端的财务边界。"
          />
          <Row gutter={[24, 16]}>
            <Col xs={24} lg={12}>
              <Card size="small" title="按订单拆分">
                <Space direction="vertical">
                  <Switch
                    checked={splitByOrder}
                    checkedChildren="已启用"
                    unCheckedChildren="未启用"
                    onChange={setSplitByOrder}
                  />
                  <Text type="secondary">
                    启用后每个业务订单独立成账；关闭时允许同一结算单位的多票订单汇总对账。
                  </Text>
                </Space>
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card size="small" title="按税率拆分">
                <Space direction="vertical">
                  <Switch
                    checked={splitByTaxRate}
                    checkedChildren="已启用"
                    unCheckedChildren="未启用"
                    onChange={setSplitByTaxRate}
                  />
                  <Text type="secondary">
                    启用后不同税率独立成账；关闭时税率仍会逐费用行固化，后续开票可按行处理。
                  </Text>
                </Space>
              </Card>
            </Col>
          </Row>
          <Descriptions size="small" column={2} style={{ marginTop: 20 }}>
            <Descriptions.Item label="已选费用">
              {selectedIds.length} 笔
            </Descriptions.Item>
            <Descriptions.Item label="预览机制">
              服务端实时拆分并签发快照令牌
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {current === 2 && preview?.data && (
        <Form form={form} layout="vertical">
          <Card
            size="small"
            style={{ marginBottom: 16, background: '#fafafa', border: '1px solid #f0f0f0' }}
          >
            <Row justify="space-between" align="middle" gutter={[16, 8]}>
              <Col xs={24} md={16}>
                <Space size="large" wrap>
                  <Space>
                    <SettingOutlined style={{ color: '#1677ff' }} />
                    <Text strong>拆单策略微调：</Text>
                  </Space>
                  <Space>
                    <Text type="secondary">按订单拆分：</Text>
                    <Switch
                      size="small"
                      checked={splitByOrder}
                      onChange={async (checked) => {
                        setSplitByOrder(checked);
                        await loadPreview(undefined, {
                          splitByOrder: checked,
                          splitByTaxRate,
                        });
                      }}
                    />
                  </Space>
                  <Space>
                    <Text type="secondary">按税率拆分：</Text>
                    <Switch
                      size="small"
                      checked={splitByTaxRate}
                      onChange={async (checked) => {
                        setSplitByTaxRate(checked);
                        await loadPreview(undefined, {
                          splitByOrder,
                          splitByTaxRate: checked,
                        });
                      }}
                    />
                  </Space>
                </Space>
              </Col>
              <Col xs={24} md={8} style={{ textAlign: 'right' }}>
                <Space>
                  <Tag color="blue">
                    共 {preview.data.length} 张拟生成账单
                  </Tag>
                  <Button
                    size="small"
                    icon={<ReloadOutlined />}
                    loading={loading}
                    onClick={() => void loadPreview()}
                  >
                    刷新快照
                  </Button>
                </Space>
              </Col>
            </Row>
          </Card>

          {preview.data.map((group, index) => (
            <Card
              key={group.groupKey}
              size="small"
              style={{
                marginBottom: 16,
                border: '1px solid #e8e8e8',
                borderRadius: 6,
              }}
              title={
                <Space wrap>
                  <Tag
                    color={
                      group.direction === 'RECEIVABLE' ? 'green' : 'volcano'
                    }
                  >
                    {directionText(group.direction)}
                  </Tag>
                  <span style={{ fontWeight: 600 }}>{group.settlementPartyName}</span>
                  <Text type="secondary">
                    {group.orderNo ? `订单 ${group.orderNo}` : '多订单汇总'}
                  </Text>
                  {group.taxRate != null && (
                    <Tag>{Number(group.taxRate)}% 税率</Tag>
                  )}
                  <Tag color="geekblue">{group.fees?.length || 0} 笔费用</Tag>
                </Space>
              }
              extra={
                <Text strong style={{ color: '#1677ff', fontSize: 14 }}>
                  {group.totalAmount} {group.currency}
                </Text>
              }
            >
              <Row gutter={16} style={{ marginBottom: 8 }}>
                <Col xs={24} md={8}>
                  <Form.Item
                    name={['groups', index, 'statementTitle']}
                    label={
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                          width: '100%',
                        }}
                      >
                        <span>对账抬头</span>
                        <Tooltip title="为该结算单位新增开票抬头并自动选中">
                          <Button
                            type="link"
                            size="small"
                            icon={<PlusOutlined />}
                            style={{
                              padding: 0,
                              height: 'auto',
                              fontSize: 12,
                              fontWeight: 'normal',
                            }}
                            onClick={() =>
                              handleOpenQuickAddProfile(
                                index,
                                group.settlementPartyId,
                                group.settlementPartyName,
                              )
                            }
                          >
                            新增抬头
                          </Button>
                        </Tooltip>
                      </div>
                    }
                    rules={[
                      {
                        required: true,
                        whitespace: true,
                        message: '请输入对账抬头',
                      },
                      { max: 200, message: '对账抬头不能超过 200 字' },
                    ]}
                  >
                    <AutoComplete
                      options={(() => {
                        const profiles = (
                          invoiceProfilesMap[group.settlementPartyId || ''] || []
                        ).filter((p) => p.enabled !== false);
                        const list = profiles.map((p) => ({
                          value: p.invoiceTitle || '',
                          label: (
                            <div
                              style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                                alignItems: 'center',
                              }}
                            >
                              <span style={{ fontWeight: 500 }}>
                                {p.invoiceTitle}
                              </span>
                              <Space size="small">
                                {p.isDefault && (
                                  <Tag
                                    color="blue"
                                    style={{ margin: 0, fontSize: 11 }}
                                  >
                                    默认
                                  </Tag>
                                )}
                                {p.taxpayerIdentificationNo && (
                                  <Text
                                    type="secondary"
                                    style={{ fontSize: 11 }}
                                  >
                                    税号: {p.taxpayerIdentificationNo}
                                  </Text>
                                )}
                              </Space>
                            </div>
                          ),
                        }));
                        if (
                          !list.some(
                            (opt) => opt.value === group.settlementPartyName,
                          )
                        ) {
                          list.unshift({
                            value: group.settlementPartyName || '',
                            label: (
                              <span>
                                {group.settlementPartyName}（结算单位全称）
                              </span>
                            ),
                          });
                        }
                        return list;
                      })()}
                      dropdownRender={(menu) => (
                        <>
                          {menu}
                          <Divider style={{ margin: '4px 0' }} />
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
                            }}
                            onMouseDown={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              handleOpenQuickAddProfile(
                                index,
                                group.settlementPartyId,
                                group.settlementPartyName,
                              );
                            }}
                          >
                            <PlusOutlined /> 为【{group.settlementPartyName}】新增开票抬头
                          </div>
                        </>
                      )}
                      placeholder="输入或下拉选择对账抬头"
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={5}>
                  <Form.Item
                    name={['groups', index, 'billDate']}
                    label="账单日期"
                    rules={[{ required: true, message: '请选择账单日期' }]}
                  >
                    <DatePicker allowClear={false} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col xs={24} md={4}>
                  <Form.Item
                    name={['groups', index, 'paymentTermsDays']}
                    label="账期（天）"
                  >
                    <InputNumber
                      min={0}
                      max={3650}
                      precision={0}
                      placeholder="天数"
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={7}>
                  <Form.Item
                    name={['groups', index, 'note']}
                    label="备注"
                    rules={[{ max: 500, message: '备注不能超过 500 字' }]}
                  >
                    <Input maxLength={500} placeholder="选填，账单备注" />
                  </Form.Item>
                </Col>
              </Row>
              <ProTable<API.FeeLedgerItem>
                rowKey="id"
                size="small"
                bordered
                search={false}
                options={false}
                toolBarRender={false}
                pagination={false}
                columns={getPreviewFeeColumns(true)}
                dataSource={group.fees || []}
                scroll={{ x: 880 }}
              />
            </Card>
          ))}
        </Form>
      )}

      {current === 3 &&
        (result ? (
          <>
            <Result
              status="success"
              icon={<CheckCircleOutlined />}
              title={`批次 ${result.batchNo || ''} 生成成功`}
              subTitle={`${result.feeCount || 0} 笔费用已原子生成 ${result.billCount || 0} 张账单，当前${result.bills?.every((bill) => bill.status === 'CONFIRMED') ? '已全部确认' : '为草稿状态'}，未发生部分成功。`}
              extra={
                <Space wrap>
                  {access.canConfirmFinanceBills &&
                    result.bills?.every((bill) => bill.status === 'DRAFT') && (
                      <Button
                        type="primary"
                        loading={confirming}
                        onClick={() => void confirmBatch()}
                      >
                        确认本批全部账单
                      </Button>
                    )}
                  <Button onClick={() => history.push('/finance/invoices')}>
                    前往开票 / 来票
                  </Button>
                  <Button
                    onClick={() => history.push('/finance/verifications')}
                  >
                    前往核销管理
                  </Button>
                </Space>
              }
            />
            <Descriptions
              bordered
              size="small"
              column={4}
              style={{ marginBottom: 16 }}
            >
              <Descriptions.Item label="批次号">
                {result.batchNo}
              </Descriptions.Item>
              <Descriptions.Item label="费用数">
                {result.feeCount}
              </Descriptions.Item>
              <Descriptions.Item label="账单数">
                {result.billCount}
              </Descriptions.Item>
              <Descriptions.Item label="本币合计">
                {result.totalBaseAmount} {result.baseCurrency}
              </Descriptions.Item>
            </Descriptions>
            <Table<API.FinanceBill>
              rowKey="id"
              size="small"
              bordered
              pagination={false}
              dataSource={result.bills || []}
              columns={[
                { title: '账单编号', dataIndex: 'billNo', width: 180 },
                {
                  title: '状态',
                  dataIndex: 'status',
                  width: 90,
                  render: (value) =>
                    value === 'CONFIRMED' ? (
                      <Tag color="green">已确认</Tag>
                    ) : (
                      <Tag color="gold">草稿</Tag>
                    ),
                },
                {
                  title: '方向',
                  dataIndex: 'direction',
                  width: 80,
                  render: (value) => directionText(String(value)),
                },
                { title: '结算单位', dataIndex: 'settlementPartyName' },
                { title: '对账抬头', dataIndex: 'statementTitle' },
                {
                  title: '金额',
                  align: 'right',
                  render: (_, row) => `${row.totalAmount} ${row.currency}`,
                },
                { title: '到期日', dataIndex: 'dueDate', width: 120 },
              ]}
            />
          </>
        ) : (
          <Empty />
        ))}
      </Drawer>

      <Modal
        title={`为【${quickAddPartnerName}】新增开票抬头`}
        open={quickAddOpen}
        confirmLoading={quickAddSaving}
        okText="保存并选用"
        cancelText="取消"
        onOk={() => void handleSaveQuickAddProfile()}
        onCancel={() => setQuickAddOpen(false)}
        destroyOnClose
        width={620}
      >
        <Form form={quickAddForm} layout="vertical" preserve={false}>
          <Alert
            type="info"
            showIcon
            message="新增的开票抬头将自动保存至该客户的主档案中，并自动选中为当前账单的对账抬头。"
            style={{ marginBottom: 16 }}
          />
          <Row gutter={16}>
            <Col span={24}>
              <Form.Item
                name="invoiceTitle"
                label="发票抬头全称"
                rules={[
                  {
                    required: true,
                    whitespace: true,
                    message: '请输入发票抬头全称',
                  },
                  { max: 200, message: '抬头全称不能超过 200 字' },
                ]}
              >
                <Input placeholder="公司注册全称或开票抬头" maxLength={200} />
              </Form.Item>
            </Col>
            <Col span={14}>
              <Form.Item
                name="taxpayerIdentificationNo"
                label="统一社会信用代码 / 税号"
                rules={[
                  {
                    required: true,
                    whitespace: true,
                    message: '请输入纳税人识别号/税号',
                  },
                  { max: 50, message: '税号不能超过 50 位' },
                ]}
              >
                <Input
                  placeholder="18 位纳税人识别号（自动大写）"
                  maxLength={50}
                />
              </Form.Item>
            </Col>
            <Col span={10}>
              <Form.Item
                name="defaultInvoiceType"
                label="默认发票类型"
                rules={[{ required: true, message: '请选择发票类型' }]}
              >
                <Select
                  options={[
                    { label: '增值税普通发票', value: 'NORMAL' },
                    { label: '增值税专用发票', value: 'SPECIAL' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="bankName" label="开户银行（选填）">
                <Input placeholder="例如：中国工商银行上海分行" maxLength={100} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="bankAccount" label="银行账号（选填）">
                <Input placeholder="开户行银行账号" maxLength={50} />
              </Form.Item>
            </Col>
            <Col span={15}>
              <Form.Item name="registeredAddress" label="开票地址（选填）">
                <Input placeholder="企业注册地址" maxLength={200} />
              </Form.Item>
            </Col>
            <Col span={9}>
              <Form.Item name="registeredPhone" label="开票电话（选填）">
                <Input placeholder="注册联系电话" maxLength={50} />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item name="isDefault" valuePropName="checked" style={{ marginBottom: 0 }}>
                <Checkbox>设为该客户的默认首选开票抬头</Checkbox>
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </>
  );
}
