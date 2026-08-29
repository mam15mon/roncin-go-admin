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
  Space,
  Steps,
  Switch,
  Tag,
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
import BillCreationResultTable from './BillCreationResultTable';
import BillGroupCard from './BillGroupCard';
import BillSplitStrategyCards from './BillSplitStrategyCards';
import QuickAddInvoiceProfileModal from './QuickAddInvoiceProfileModal';
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
  const splitByOrderRef = useRef(splitByOrder);
  const splitByTaxRateRef = useRef(splitByTaxRate);

  useEffect(() => {
    selectedIdsRef.current = selectedIds;
    splitByOrderRef.current = splitByOrder;
    splitByTaxRateRef.current = splitByTaxRate;
  }, [selectedIds, splitByOrder, splitByTaxRate]);

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
              <BillGroupCard
                key={group.groupKey}
                group={group}
                index={index}
                invoiceProfilesMap={invoiceProfilesMap}
                feeColumns={getPreviewFeeColumns(handleRemoveFee)}
                directionText={directionText}
                onOpenQuickAddProfile={handleOpenQuickAddProfile}
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

      <QuickAddInvoiceProfileModal
        open={quickAddOpen}
        partnerName={quickAddPartnerName}
        saving={quickAddSaving}
        form={quickAddForm}
        onOk={() => void handleSaveQuickAddProfile()}
        onCancel={() => setQuickAddOpen(false)}
      />
    </>
  );
}
