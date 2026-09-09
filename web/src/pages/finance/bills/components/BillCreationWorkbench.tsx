import {
  ArrowLeftOutlined,
  FileDoneOutlined,
  ReloadOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Drawer,
  Form,
  Row,
  Select,
  Space,
  Steps,
  Switch,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { FinanceOrganizationPurpose, OrderFeeStatus } from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceConfirmBillBatch,
  settlementServiceCreateBillBatch,
  settlementServiceListBillCreationCandidates,
  settlementServiceListFinanceOrganizationOptions,
  settlementServicePreviewBillBatch,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { longRequestOptions } from '@/utils/requestTimeout';
import { generateUUID } from '@/utils/uuid';
import BillCreationResultTable from './BillCreationResultTable';
import BillGroupCard from './BillGroupCard';
import BillSplitStrategyCards from './BillSplitStrategyCards';
import {
  getPreviewFeeColumns,
  selectionFeeColumns,
} from './billWorkbenchFeeColumns';

const { Text } = Typography;

type GroupFormValue = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
};

type WorkbenchFormValue = {
  groups: GroupFormValue[];
};

type RequestError = Error & {
  data?: { reason?: string; message?: string };
  response?: { data?: { reason?: string; message?: string } };
};

export type BillCreationWorkbenchProps = {
  open: boolean;
  initialFeeIds?: string[];
  initialOrganizationId?: string;
  initialOrganizationName?: string;
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
  if (reason === financeErrorReasons.FINANCE_BILL_FEE_INVALID) {
    return '所选费用必须为已确认状态且尚未进入其他账单';
  }
  if (reason === financeErrorReasons.FINANCE_BILL_PREVIEW_STALE) {
    return '费用已发生变化，请重新预览后再生成账单';
  }
  return fallback;
}

export default function BillCreationWorkbench({
  open,
  initialFeeIds = [],
  initialOrganizationId,
  initialOrganizationName,
  sourceLabel,
  onClose,
  onCreated,
}: BillCreationWorkbenchProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<WorkbenchFormValue>();
  const [current, setCurrent] = useState(0);
  const [selectedFeeIds, setSelectedFeeIds] = useState<React.Key[]>([]);
  const [splitByOrder, setSplitByOrder] = useState(false);
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
  const previewInitKeyRef = useRef<string | undefined>(undefined);
  const previewErrorKeyRef = useRef<string | undefined>(undefined);
  const previewRequestTokenRef = useRef(0);
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
      .catch((error: any) => {
        if (!cancelled) {
          setOrganizationOptions([]);
          message.error(error.message || '加载可建账所属公司失败');
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
  }, [organizationId, selectedIds, splitByOrder, splitByTaxRate]);

  const loadPreview = useCallback(
    async (
      overrideIds?: string[],
      policyOverride?: { splitByOrder: boolean; splitByTaxRate: boolean },
      organizationIdOverride?: string,
    ) => {
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
      const requestToken = ++previewRequestTokenRef.current;
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
            organizationId: requestedOrganizationId,
          },
          { ...longRequestOptions, skipErrorHandler: true },
        );
        if (
          requestToken !== previewRequestTokenRef.current ||
          requestedOrganizationId !== organizationIdRef.current
        ) {
          return false;
        }
        const groups = unwrapList(response);
        if (!response.previewToken || groups.length === 0) {
          throw new Error('服务端未返回有效的拆单预览');
        }
        previewErrorKeyRef.current = undefined;
        setPreview(response);

        form.setFieldsValue({
          groups: groups.map((group) => {
            return {
              statementTitle: group.settlementPartyName || '',
              billDate: dayjs(),
              paymentTermsDays: undefined,
              note: undefined,
            };
          }),
        });
        return true;
      } catch (rawError: unknown) {
        if (
          requestToken !== previewRequestTokenRef.current ||
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
    [form, message],
  );

  // 初始化或当从业务页面进入时，自动快速预览并直达账单资料页
  useEffect(() => {
    if (!open) {
      previewRequestTokenRef.current += 1;
      previewInitKeyRef.current = undefined;
      previewErrorKeyRef.current = undefined;
      previewPendingRef.current = null;
      return;
    }
    const initialIds = initialFeeKey
      ? initialFeeKey.split('|').filter(Boolean)
      : [];
    const initKey = `${initialFeeKey}:${initialOrganizationId || ''}:${open ? 'open' : 'closed'}`;
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
    previewRequestTokenRef.current += 1;
    setSelectedFeeIds(initialIds);
    const nextOrganizationId =
      initialIds.length > 0 ? initialOrganizationId : undefined;
    organizationIdRef.current = nextOrganizationId;
    setOrganizationId(nextOrganizationId);
    setSplitByOrder(false);
    setSplitByTaxRate(false);
    setPreview(undefined);
    setResult(undefined);
    setLoading(false);
    setConfirming(false);
    setIdempotencyKey(generateUUID());
    form.resetFields();

    if (initialIds.length > 0 && initialOrganizationId) {
      // 极速模式：从单票/多选费用带入时，直接拉取预览并切到账单资料页
      const previewPromise = loadPreview(
        initialIds,
        {
          splitByOrder: false,
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
  }, [open, initialFeeKey, initialOrganizationId, form, loadPreview]);

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
      if (await loadPreview()) setCurrent(2);
    }
  };

  const createBatch = async () => {
    if (!organizationId) {
      message.warning('请先选择可建账所属公司');
      return;
    }
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
          organizationId,
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
    <Drawer
      title="费用批量转账单"
      open={open}
      size="min(1280px, 96vw)"
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

      {current === 0 &&
        (fixedSelection ? (
          <Card>
            <Alert
              type="info"
              showIcon
              title={`已从${sourceLabel || '业务页面'}带入 ${selectedIds.length} 笔已确认费用`}
              description="费用状态、结算维度和金额快照将在预览及最终建单事务中由服务端再次校验。"
            />
            <div style={{ marginTop: 12 }}>
              来源公司：
              {initialOrganizationName ||
                organizationOptions.find(
                  (item) => item.id === initialOrganizationId,
                )?.name ||
                initialOrganizationId ||
                '-'}
            </div>
          </Card>
        ) : (
          <>
            <Select
              aria-label="所属公司"
              allowClear
              placeholder="请先选择可建账所属公司"
              style={{ width: 260, marginBottom: 12 }}
              value={organizationId}
              options={organizationOptions.map((item) => ({
                value: item.id,
                label: item.name || item.code || item.id,
              }))}
              onChange={(value) => {
                previewRequestTokenRef.current += 1;
                organizationIdRef.current = value;
                setOrganizationId(value);
                setSelectedFeeIds([]);
                setPreview(undefined);
                setResult(undefined);
                setCurrent(0);
              }}
            />
            <ProTable<API.FeeLedgerItem>
              key={organizationId || 'no-organization'}
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
                    record.status ===
                      OrderFeeStatus.ORDER_FEE_STATUS_CONFIRMED &&
                    !record.billNo;
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
                if (!organizationId)
                  return { data: [], success: true, total: 0 };
                const response =
                  await settlementServiceListBillCreationCandidates({
                    page: params.current,
                    pageSize: params.pageSize,
                    keyword: params.keyword,
                    direction: params.direction,
                    organizationId,
                  });
                return toTableRequest(response);
              }}
            />
          </>
        ))}

      {current === 1 && (
        <BillSplitStrategyCards
          splitByOrder={splitByOrder}
          setSplitByOrder={setSplitByOrder}
          splitByTaxRate={splitByTaxRate}
          setSplitByTaxRate={setSplitByTaxRate}
          selectedCount={selectedIds.length}
        />
      )}

      {current === 2 && preview?.data && (
        <Form form={form} layout="vertical">
          <Card
            size="small"
            style={{
              marginBottom: 16,
              background: '#fafafa',
              border: '1px solid #f0f0f0',
            }}
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
                  <Tag color="blue">共 {preview.data.length} 张拟生成账单</Tag>
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
            <BillGroupCard
              key={group.groupKey}
              group={group}
              index={index}
              feeColumns={getPreviewFeeColumns(handleRemoveFee)}
              directionText={directionText}
            />
          ))}
        </Form>
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
