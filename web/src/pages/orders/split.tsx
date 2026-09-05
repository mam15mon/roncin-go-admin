import {
  CheckCircleOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { history, useAccess, useParams } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Input,
  Popconfirm,
  Radio,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import Decimal from 'decimal.js';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { SectionCard, StickyFooterBar } from '@/components/ui';
import OrderPageHeader from './components/OrderPageHeader';
import { OrderBusinessType } from '@/enums.generated';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaOrderSplit,
  seaOrderChangeServiceGetSeaOrderSplitContext,
  seaOrderChangeServicePreviewSeaOrderSplit,
} from '@/services/roncin/seaOrderChangeService';
import { computeCanonicalSha256 } from '@/utils/hash';
import { PARTNER_ROLES, searchPartnersByRole } from './common';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

const { Text } = Typography;
const { TextArea } = Input;

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const messageValue = (error as { message?: unknown }).message;
    if (typeof messageValue === 'string' && messageValue) return messageValue;
  }
  return fallback;
}

interface ResultConfig {
  key: string;
  role: 'ORIGINAL' | 'CREATED';
  title: string;
  targetType: 'CURRENT' | 'NEW' | 'CANDIDATE';
  candidateId?: string;
  candidateVersion?: string;
  candidateTeId?: string;
  candidateTeVersion?: string;
  masterNo?: string;
  issuerPartnerId?: string;
  carrierId?: string;
  vesselName?: string;
  voyageNo?: string;
  originLocationId?: string;
  dischargeLocationId?: string;
  transitLocationId?: string;
  etd?: string;
  eta?: string;
  internalReferenceNo?: string;
  bookingNotes?: string;
  allocationNotes?: string;
  operationNotes?: string;
}

interface FeeCurrencySummary {
  key: string;
  direction: string;
  currency: string;
  baseline: Decimal;
  assignedByResult: Record<string, Decimal>;
  remaining: Decimal;
}

export function calculateFeeCurrencySummaries(
  fees: API.SeaOrderSplitDraftFeeItem[],
  assignments: Record<string, string>,
  resultKeys: string[],
): FeeCurrencySummary[] {
  const summaries = new Map<string, FeeCurrencySummary>();

  for (const fee of fees) {
    if (!fee.id || !fee.direction || !fee.currency || !fee.totalAmount) {
      throw new Error('草稿费用缺少 ID、方向、币种或金额，无法计算费用守恒');
    }
    const key = `${fee.direction}:${fee.currency}`;
    let summary = summaries.get(key);
    if (!summary) {
      summary = {
        key,
        direction: fee.direction,
        currency: fee.currency,
        baseline: new Decimal(0),
        assignedByResult: Object.fromEntries(
          resultKeys.map((resultKey) => [resultKey, new Decimal(0)]),
        ),
        remaining: new Decimal(0),
      };
      summaries.set(key, summary);
    }

    const amount = new Decimal(fee.totalAmount);
    summary.baseline = summary.baseline.add(amount);
    const resultKey = assignments[fee.id];
    if (resultKey && summary.assignedByResult[resultKey]) {
      summary.assignedByResult[resultKey] =
        summary.assignedByResult[resultKey].add(amount);
    }
  }

  return Array.from(summaries.values())
    .map((summary) => ({
      ...summary,
      remaining: summary.baseline.sub(
        Object.values(summary.assignedByResult).reduce(
          (total, amount) => total.add(amount),
          new Decimal(0),
        ),
      ),
    }))
    .sort((left, right) => left.key.localeCompare(right.key));
}

export default function SeaOrderSplitPage() {
  const params = useParams<{ id: string }>();
  const orderId = params.id || '';
  const { message } = App.useApp();
  const access = useAccess();
  const canReassign = access.canOrder(
    OrderBusinessType.BUSINESS_TYPE_SE,
    'reassign',
  );
  const {
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
    refresh: refreshLockState,
  } = useOrderLockState(orderId);
  const lockWritePolicy = getOrderBusinessWritePolicy({
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
  });
  const lockWritePolicyRef = useRef(lockWritePolicy);
  lockWritePolicyRef.current = lockWritePolicy;

  const ensureSplitEditable = () => {
    const currentPolicy = lockWritePolicyRef.current;
    if (!currentPolicy.disabled) return true;
    message.warning(currentPolicy.reason || '订单当前不可拆票');
    return false;
  };

  const [loadingContext, setLoadingContext] = useState(false);
  const [splitContext, setSplitContext] =
    useState<API.SeaOrderSplitContextData | null>(null);

  // 拆票结果集：至少 1 个 ORIGINAL + 1 个 CREATED
  const [results, setResults] = useState<ResultConfig[]>([]);

  // 分配状态
  const [hblAssignments, setHblAssignments] = useState<Record<string, string>>(
    {},
  ); // hblId -> resultKey
  const [feeAssignments, setFeeAssignments] = useState<Record<string, string>>(
    {},
  ); // feeId -> resultKey
  const [attAssignments, setAttAssignments] = useState<
    Record<string, string[]>
  >({}); // attId -> resultKeys[]
  const initialPreviewTriggeredRef = useRef(false);
  const [note, setNote] = useState<string>('');

  // 预览与校验结果
  const [previewing, setPreviewing] = useState(false);
  const [previewData, setPreviewData] =
    useState<API.SeaOrderSplitPreviewData | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 下拉选项
  const [issuerOptions, setIssuerOptions] = useState<DefaultOptionType[]>([]);

  const feeCurrencySummaries = useMemo(
    () =>
      calculateFeeCurrencySummaries(
        splitContext?.draftFees || [],
        feeAssignments,
        results.map((result) => result.key),
      ),
    [splitContext?.draftFees, feeAssignments, results],
  );

  // 加载拆票上下文
  const loadContext = async () => {
    if (!orderId) return;
    setLoadingContext(true);
    try {
      const resp = await seaOrderChangeServiceGetSeaOrderSplitContext({
        orderId,
      });
      if (resp?.data) {
        const ctx = resp.data;
        setSplitContext(ctx);

        // 初始化结果
        const initialResults: ResultConfig[] = [
          {
            key: 'res-origin',
            role: 'ORIGINAL',
            title: `原票 (${ctx.orderNo})`,
            targetType: 'CURRENT',
            internalReferenceNo: ctx.internalReferenceNo,
            bookingNotes: ctx.bookingNotes,
            allocationNotes: ctx.allocationNotes,
            operationNotes: ctx.operationNotes,
          },
          {
            key: 'res-new-1',
            role: 'CREATED',
            title: '拆出新票 1',
            targetType: 'CURRENT',
            internalReferenceNo: '',
            bookingNotes: ctx.bookingNotes,
            allocationNotes: ctx.allocationNotes || '',
            operationNotes: ctx.operationNotes,
          },
        ];
        setResults(initialResults);

        // 初始化分配：默认第 1 个 HBL 留原票，其余到新票 1
        const initialHblMap: Record<string, string> = {};
        ctx.houseBills?.forEach(
          (h: API.SeaOrderSplitHouseBillItem, idx: number) => {
            if (h.id) {
              initialHblMap[h.id] = idx === 0 ? 'res-origin' : 'res-new-1';
            }
          },
        );
        setHblAssignments(initialHblMap);

        // 费用默认全部留原票
        const initialFeeMap: Record<string, string> = {};
        ctx.draftFees?.forEach((f: API.SeaOrderSplitDraftFeeItem) => {
          if (f.id) {
            initialFeeMap[f.id] = 'res-origin';
          }
        });
        setFeeAssignments(initialFeeMap);

        // 附件默认全部分配到原票
        const initialAttMap: Record<string, string[]> = {};
        ctx.attachments?.forEach((a: API.SeaOrderSplitAttachmentItem) => {
          if (a.id) {
            initialAttMap[a.id] = ['res-origin'];
          }
        });
        setAttAssignments(initialAttMap);
      }
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '加载拆票上下文失败'));
    } finally {
      setLoadingContext(false);
    }
  };

  useEffect(() => {
    loadContext();
  }, [orderId]);

  // 构造母单目标
  const buildTargets = (
    currentResults = results,
  ): API.SeaOrderSplitTargetInput[] => {
    return currentResults.map((r) => ({
      clientTargetKey: r.key,
      targetType: r.targetType,
      candidateId: r.targetType === 'CANDIDATE' ? r.candidateId : undefined,
      candidateVersion:
        r.targetType === 'CANDIDATE' ? r.candidateVersion : undefined,
      candidateTeId: r.targetType === 'CANDIDATE' ? r.candidateTeId : undefined,
      candidateTeVersion:
        r.targetType === 'CANDIDATE' ? r.candidateTeVersion : undefined,
      masterNo: r.masterNo,
      issuerPartnerId: r.issuerPartnerId,
      carrierId: r.carrierId,
      vesselName: r.vesselName,
      voyageNo: r.voyageNo,
      originLocationId: r.originLocationId,
      dischargeLocationId: r.dischargeLocationId,
      etd: r.etd,
      eta: r.eta,
    }));
  };

  // 构造拆票结果明细
  const buildSplitResults = (
    currentResults = results,
    currentHbls = hblAssignments,
    currentFees = feeAssignments,
    currentAtts = attAssignments,
  ): API.SeaOrderSplitResultInput[] => {
    return currentResults.map((r) => {
      const hblIds = Object.entries(currentHbls)
        .filter(([, resKey]) => resKey === r.key)
        .map(([hId]) => hId);

      const feeIds = Object.entries(currentFees)
        .filter(([, resKey]) => resKey === r.key)
        .map(([fId]) => fId);

      const attIds = Object.entries(currentAtts)
        .filter(([, resKeys]) => resKeys.includes(r.key))
        .map(([aId]) => aId);

      return {
        clientResultKey: r.key,
        resultRole: r.role,
        clientTargetKey: r.key,
        houseBillIds: hblIds,
        draftFeeIds: feeIds,
        attachmentReferenceIds: attIds,
        internalReferenceNo: r.internalReferenceNo,
        bookingNotes: r.bookingNotes,
        allocationNotes: r.allocationNotes,
        operationNotes: r.operationNotes,
      };
    });
  };

  // 构造全量预期版本字典
  const buildExpectedVersions = (
    currentResults = results,
  ): API.SeaOrderSplitExpectedVersions | undefined => {
    if (!splitContext) return undefined;
    if (
      !splitContext.orderVersion ||
      !splitContext.currentLinkVersion ||
      !splitContext.cargoAllocationVersion
    ) {
      return undefined;
    }
    const houseBillVersions: Record<string, string> = {};
    for (const h of splitContext.houseBills || []) {
      if (!h.id || !h.version) return undefined;
      houseBillVersions[h.id] = String(h.version);
    }
    const cargoItemVersions: Record<string, string> = {};
    for (const ci of splitContext.cargoItems || []) {
      if (!ci.id || !ci.version) return undefined;
      cargoItemVersions[ci.id] = String(ci.version);
    }
    const containerVersions: Record<string, string> = {};
    for (const c of splitContext.containers || []) {
      if (!c.id || !c.version) return undefined;
      containerVersions[c.id] = String(c.version);
    }
    const feeVersions: Record<string, string> = {};
    for (const f of splitContext.draftFees || []) {
      if (!f.id || !f.version) return undefined;
      feeVersions[f.id] = String(f.version);
    }
    const candidateMblVersions: Record<string, string> = {};
    const candidateTeVersions: Record<string, string> = {};
    for (const r of currentResults) {
      if (r.targetType === 'CANDIDATE') {
        if (
          !r.candidateId ||
          !r.candidateVersion ||
          !r.candidateTeId ||
          !r.candidateTeVersion
        ) {
          return undefined;
        }
        candidateMblVersions[r.candidateId] = String(r.candidateVersion);
        candidateTeVersions[r.candidateTeId] = String(r.candidateTeVersion);
      }
    }

    return {
      orderVersion: String(splitContext.orderVersion),
      linkVersion: String(splitContext.currentLinkVersion),
      allocationVersion: String(splitContext.cargoAllocationVersion),
      houseBillVersions,
      cargoItemVersions,
      containerVersions,
      feeVersions,
      candidateMblVersions,
      candidateTeVersions,
      attachmentReferenceFingerprint:
        splitContext.attachmentReferenceFingerprint,
    };
  };

  // 触发校验与预览
  const triggerPreview = async (
    currentResults = results,
    currentHbls = hblAssignments,
    currentFees = feeAssignments,
    currentAtts = attAssignments,
  ) => {
    if (
      lockWritePolicyRef.current.disabled ||
      !orderId ||
      !splitContext ||
      currentResults.length < 2
    )
      return;
    setPreviewing(true);
    setPreviewError(null);
    try {
      const targets = buildTargets(currentResults);
      const splitResults = buildSplitResults(
        currentResults,
        currentHbls,
        currentFees,
        currentAtts,
      );
      const expectedVersions = buildExpectedVersions(currentResults);
      if (!expectedVersions) {
        setPreviewError(
          '缺少完整版本控制信息或候选版本未获取，无法进行拆票校验',
        );
        setPreviewData(null);
        return;
      }

      const resp = await seaOrderChangeServicePreviewSeaOrderSplit(
        { orderId },
        {
          orderId,
          note: note ? note.trim() : undefined,
          targets,
          results: splitResults,
          expectedVersions,
        },
      );

      if (resp?.data) {
        setPreviewData(resp.data);
      }
    } catch (error: unknown) {
      setPreviewError(getErrorMessage(error, '拆票校验未通过'));
      setPreviewData(null);
    } finally {
      setPreviewing(false);
    }
  };

  // 依赖变化时防抖预览
  useEffect(() => {
    if (splitContext && results.length >= 2) {
      if (!initialPreviewTriggeredRef.current) {
        initialPreviewTriggeredRef.current = true;
        triggerPreview();
        return undefined;
      }
      const timer = setTimeout(() => {
        triggerPreview();
      }, 300);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [
    results,
    hblAssignments,
    feeAssignments,
    attAssignments,
    note,
    lockWritePolicy.disabled,
  ]);

  useEffect(() => {
    if (lockWritePolicy.disabled) {
      setPreviewData(null);
    }
  }, [lockWritePolicy.disabled]);

  // 添加新票
  const handleAddResult = () => {
    if (!ensureSplitEditable()) return;
    const nextIdx = results.filter((r) => r.role === 'CREATED').length + 1;
    const newKey = `res-new-${Date.now()}`;
    const newRes: ResultConfig = {
      key: newKey,
      role: 'CREATED',
      title: `拆出新票 ${nextIdx}`,
      targetType: 'CURRENT',
      internalReferenceNo: '',
      bookingNotes: splitContext?.bookingNotes,
      allocationNotes: splitContext?.allocationNotes || '',
      operationNotes: splitContext?.operationNotes,
    };
    const updated = [...results, newRes];
    setResults(updated);
  };

  // 移除新票
  const handleRemoveResult = (key: string) => {
    if (!ensureSplitEditable()) return;
    if (results.filter((r) => r.role === 'CREATED').length <= 1) {
      message.warning('拆票必须至少保留一个拆出新票');
      return;
    }
    const updated = results.filter((r) => r.key !== key);
    const nextHbls = { ...hblAssignments };
    Object.keys(nextHbls).forEach((hId) => {
      if (nextHbls[hId] === key) nextHbls[hId] = 'res-origin';
    });
    const nextFees = { ...feeAssignments };
    Object.keys(nextFees).forEach((fId) => {
      if (nextFees[fId] === key) nextFees[fId] = 'res-origin';
    });
    setResults(updated);
    setHblAssignments(nextHbls);
    setFeeAssignments(nextFees);
  };

  // 执行拆票提交
  const handleExecuteSplit = async () => {
    if (!ensureSplitEditable()) return;
    if (!previewData?.isValid || !previewData?.conservationPassed) {
      message.error(
        previewError || '拆票数据未满足守恒校验或存在门禁错误，请检查！',
      );
      return;
    }
    setSubmitting(true);
    try {
      const targets = buildTargets(results);
      const splitResults = buildSplitResults(
        results,
        hblAssignments,
        feeAssignments,
        attAssignments,
      );
      const expectedVersions = buildExpectedVersions(results);
      if (!expectedVersions) {
        message.error('未获取到有效拆票版本信息，请刷新重试！');
        return;
      }

      // 稳定指纹与幂等键：同参数输入生成稳定 SHA-256 哈希，参数变化生成新 key
      const payloadForHash = {
        orderId,
        targets,
        results: splitResults,
        note: note ? note.trim() : undefined,
        expectedVersions,
      };
      const hash = computeCanonicalSha256(payloadForHash);
      const fingerprint = `split-fp:${hash}`;
      const idempotencyKey = `split:${orderId}:${hash}`;

      const resp = await seaOrderChangeServiceExecuteSeaOrderSplit(
        { orderId },
        {
          orderId,
          idempotencyKey,
          requestFingerprint: fingerprint,
          note: note ? note.trim() : undefined,
          targets,
          results: splitResults,
          expectedVersions,
        },
      );

      const createdCount = resp?.data?.createdOrders?.length || 0;
      message.success(`拆票成功！原票已更新，成功生成 ${createdCount} 张新票`);
      history.push(`/orders/sea-export/${orderId}`);
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '执行拆票失败'));
    } finally {
      setSubmitting(false);
    }
  };

  // 分单分配列
  const hblColumns: ColumnsType<API.SeaOrderSplitHouseBillItem> = [
    {
      title: '分单号 (HBL No)',
      dataIndex: 'houseNo',
      render: (val) => <Text strong>{val}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (val) => (
        <Tag color={val === 'DRAFT' ? 'orange' : 'green'}>{val}</Tag>
      ),
    },
    {
      title: '集装箱与货物绑定情况',
      render: (_, hbl) => {
        const allocs =
          splitContext?.allocations?.filter((a) => a.houseBillId === hbl.id) ||
          [];
        const containerIds = Array.from(
          new Set(allocs.map((a) => a.containerId).filter(Boolean)),
        );
        const totalPkg = allocs.reduce(
          (acc, cur) => acc + (cur.packageCount || 0),
          0,
        );
        return (
          <Space orientation="vertical" size={2}>
            <Text type="secondary">
              关联装箱：{containerIds.length} 箱（合计分配 {totalPkg} 件）
            </Text>
          </Space>
        );
      },
    },
    {
      title: '归属结果票',
      width: 260,
      render: (_, hbl) => (
        <Select
          value={hbl.id ? hblAssignments[hbl.id] : undefined}
          style={{ width: '100%' }}
          onChange={(val) => {
            if (hbl.id) {
              setHblAssignments({ ...hblAssignments, [hbl.id]: val });
            }
          }}
          options={results.map((r) => ({
            label: (
              <span>
                <Tag color={r.role === 'ORIGINAL' ? 'default' : 'blue'}>
                  {r.role === 'ORIGINAL' ? '原' : '新'}
                </Tag>
                {r.title}
              </span>
            ),
            value: r.key,
          }))}
        />
      ),
    },
  ];

  // 费用分配列
  const feeColumns: ColumnsType<API.SeaOrderSplitDraftFeeItem> = [
    {
      title: '费用名称',
      dataIndex: 'feeName',
      render: (val, r) => (
        <span>
          <Tag color={r.direction === 'RECEIVABLE' ? 'green' : 'red'}>
            {r.direction === 'RECEIVABLE' ? '应收' : '应付'}
          </Tag>
          {val}
        </span>
      ),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      ellipsis: true,
      render: (val) => val || '-',
    },
    {
      title: '费用金额',
      render: (_, f) => (
        <Text strong>
          {f.currency} {f.totalAmount}
        </Text>
      ),
    },
    {
      title: '整行归属结果票',
      width: 260,
      render: (_, fee) => (
        <Select
          value={fee.id ? feeAssignments[fee.id] : undefined}
          style={{ width: '100%' }}
          onChange={(val) => {
            if (fee.id) {
              setFeeAssignments({ ...feeAssignments, [fee.id]: val });
            }
          }}
          options={results.map((r) => ({
            label: (
              <span>
                <Tag color={r.role === 'ORIGINAL' ? 'default' : 'blue'}>
                  {r.role === 'ORIGINAL' ? '原' : '新'}
                </Tag>
                {r.title}
              </span>
            ),
            value: r.key,
          }))}
        />
      ),
    },
  ];

  // 附件继承列
  const attColumns: ColumnsType<API.SeaOrderSplitAttachmentItem> = [
    {
      title: '单证附件',
      dataIndex: 'fileName',
      render: (val, r) => (
        <span>
          <Tag color="geekblue">{r.docType}</Tag>
          {val}
        </span>
      ),
    },
    {
      title: '文件大小',
      dataIndex: 'fileSize',
      width: 120,
      render: (s) => `${s} 字节`,
    },
    {
      title: '共享引用至结果票',
      render: (_, att) => {
        if (!att.id) return null;
        const currentKeys = attAssignments[att.id] || [];
        return (
          <Checkbox.Group
            value={currentKeys}
            onChange={(checkedValues) => {
              const nextValues = checkedValues as string[];
              if (!nextValues.includes('res-origin')) {
                nextValues.push('res-origin');
              }
              setAttAssignments({
                ...attAssignments,
                [att.id as string]: nextValues,
              });
            }}
          >
            {results.map((r) => (
              <Checkbox
                key={r.key}
                value={r.key}
                disabled={r.role === 'ORIGINAL'}
              >
                {r.title}
              </Checkbox>
            ))}
          </Checkbox.Group>
        );
      },
    },
  ];

  return (
    <Spin spinning={loadingContext}>
      {/* 顶部 OrderPageHeader */}
      <OrderPageHeader
        page="split"
        orderKind="sea-export"
        orderId={orderId}
        orderNo={splitContext?.orderNo}
        subTitle="支持整单部分拆票、HBL/箱货零误差守恒切分、草稿费用整行归属及多票并行派生"
        tags={<Tag color="blue">海运出口 (HOUSE)</Tag>}
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void loadContext();
              void refreshLockState();
            }}
          >
            刷新数据
          </Button>
        }
      />

      <div style={{ padding: '0 24px 80px' }}>
        {lockWritePolicy.disabled && (
          <Alert
            type={
              lockWritePolicy.reason?.includes('已锁定') ? 'warning' : 'error'
            }
            showIcon
            message="拆票操作当前不可用"
            description={lockWritePolicy.reason}
            action={
              <Button size="small" onClick={() => void refreshLockState()}>
                重试锁状态
              </Button>
            }
            style={{ marginBottom: 16 }}
          />
        )}
        <div
          aria-disabled={lockWritePolicy.disabled}
          style={
            lockWritePolicy.disabled
              ? { pointerEvents: 'none', opacity: 0.72 }
              : undefined
          }
        >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          {previewError && (
            <Alert
              type="error"
              showIcon
              message="拆票校验未通过"
              description={previewError}
            />
          )}

          {/* 1. 顶部基线统计区块 */}
          <SectionCard
            title="原始订单基线汇总"
            collapsible
            defaultCollapsed={false}
          >
            <Row gutter={16}>
              <Col span={4}>
                <Statistic
                  title="总件数 (Packages)"
                  value={previewData?.baseline?.packageCount ?? '-'}
                  suffix="件"
                />
              </Col>
              <Col span={4}>
                <Statistic
                  title="总毛重 (Gross Weight)"
                  value={previewData?.baseline?.grossWeightKg ?? '-'}
                  suffix="KGS"
                />
              </Col>
              <Col span={4}>
                <Statistic
                  title="总体积 (Volume)"
                  value={previewData?.baseline?.volumeCbm ?? '-'}
                  suffix="CBM"
                />
              </Col>
              <Col span={4}>
                <Statistic
                  title="集装箱总数"
                  value={splitContext?.containers?.length || 0}
                  suffix="箱"
                />
              </Col>
              <Col span={4}>
                <Statistic
                  title="分单总数 (HBL)"
                  value={splitContext?.houseBills?.length || 0}
                  suffix="票"
                />
              </Col>
              <Col span={4}>
                <Statistic
                  title="可分配草稿费用"
                  value={splitContext?.draftFees?.length || 0}
                  suffix="笔"
                />
              </Col>
            </Row>
          </SectionCard>

          {/* 2. 拆票结果集规划区块 */}
          <SectionCard
            title={`拆票目标集规划（共 ${results.length} 票）`}
            extra={
              <Button
                type="dashed"
                size="small"
                icon={<PlusOutlined />}
                onClick={handleAddResult}
              >
                添加拆出新票
              </Button>
            }
          >
            <Row gutter={[16, 16]}>
              {results.map((res, index) => (
                <Col span={24} key={res.key}>
                  <Card
                    size="small"
                    style={{
                      borderColor:
                        res.role === 'ORIGINAL' ? '#d9d9d9' : '#91caff',
                      background:
                        res.role === 'ORIGINAL' ? '#fafafa' : '#f0f7ff',
                    }}
                    title={
                      <Space>
                        <Tag
                          color={
                            res.role === 'ORIGINAL' ? 'default' : 'geekblue'
                          }
                        >
                          {res.role === 'ORIGINAL'
                            ? '保留原票'
                            : `新票 ${index}`}
                        </Tag>
                        <Text strong>{res.title}</Text>
                      </Space>
                    }
                    extra={
                      res.role === 'CREATED' && (
                        <Button
                          type="text"
                          danger
                          size="small"
                          icon={<DeleteOutlined />}
                          onClick={() => handleRemoveResult(res.key)}
                        >
                          删除此新票
                        </Button>
                      )
                    }
                  >
                    <Row gutter={16}>
                      <Col span={6}>
                        <div style={{ marginBottom: 4 }}>
                          <Text type="secondary">母单配载策略：</Text>
                        </div>
                        <Radio.Group
                          value={res.targetType}
                          onChange={(e) => {
                            const val = e.target.value;
                            const updated = [...results];
                            let newAllocNotes = res.allocationNotes;
                            if (val === 'CURRENT') {
                              newAllocNotes =
                                splitContext?.allocationNotes || '';
                            } else if (res.targetType === 'CURRENT') {
                              newAllocNotes = '';
                            }
                            updated[index] = {
                              ...res,
                              targetType: val,
                              allocationNotes: newAllocNotes,
                              candidateId: undefined,
                              candidateVersion: undefined,
                              candidateTeId: undefined,
                              candidateTeVersion: undefined,
                            };
                            setResults(updated);
                          }}
                        >
                          <Radio value="CURRENT">沿用当前母单</Radio>
                          {canReassign && <Radio value="NEW">录入新母单</Radio>}
                          {canReassign && (
                            <Radio value="CANDIDATE">选择已有母单</Radio>
                          )}
                        </Radio.Group>
                      </Col>
                      <Col span={6}>
                        <div style={{ marginBottom: 4 }}>
                          <Text type="secondary">内部单号：</Text>
                        </div>
                        <Input
                          placeholder="内部单号（可留空）"
                          value={res.internalReferenceNo}
                          onChange={(e) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              internalReferenceNo: e.target.value,
                            };
                            setResults(updated);
                          }}
                        />
                      </Col>
                      <Col span={6}>
                        <div style={{ marginBottom: 4 }}>
                          <Text type="secondary">订舱 / 排载备注：</Text>
                        </div>
                        <Input
                          placeholder="订舱备注"
                          value={res.bookingNotes}
                          onChange={(e) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              bookingNotes: e.target.value,
                            };
                            setResults(updated);
                          }}
                        />
                      </Col>
                      <Col span={6}>
                        <div style={{ marginBottom: 4 }}>
                          <Text type="secondary">配载 / 分配备注：</Text>
                        </div>
                        <Input
                          placeholder="配载备注"
                          value={res.allocationNotes}
                          onChange={(e) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              allocationNotes: e.target.value,
                            };
                            setResults(updated);
                          }}
                        />
                      </Col>
                    </Row>

                    <Row gutter={16} style={{ marginTop: 12 }}>
                      <Col span={24}>
                        <div style={{ marginBottom: 4 }}>
                          <Text type="secondary">操作备注：</Text>
                        </div>
                        <Input
                          placeholder="操作备注（可修改或清空）"
                          value={res.operationNotes}
                          onChange={(e) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              operationNotes: e.target.value,
                            };
                            setResults(updated);
                          }}
                        />
                      </Col>
                    </Row>

                    {res.targetType === 'CANDIDATE' && (
                      <div
                        style={{
                          marginTop: 12,
                          padding: 12,
                          background: '#ffffff',
                          borderRadius: 4,
                        }}
                      >
                        <Space style={{ width: '100%' }}>
                          <Input
                            placeholder="输入已有草稿提单号 (MBL No)"
                            style={{ width: 260 }}
                            value={res.masterNo}
                            onChange={(e) => {
                              const updated = [...results];
                              updated[index] = {
                                ...res,
                                masterNo: e.target.value,
                                candidateId: undefined,
                                candidateVersion: undefined,
                                candidateTeId: undefined,
                                candidateTeVersion: undefined,
                              };
                              setResults(updated);
                            }}
                          />
                          <Button
                            onClick={async () => {
                              if (!res.masterNo) {
                                message.warning('请先输入提单号');
                                return;
                              }
                              if (!/^[A-Za-z0-9]+$/.test(res.masterNo)) {
                                message.warning(
                                  '提单号只能包含英文字母和阿拉伯数字，不能包含空格或符号',
                                );
                                return;
                              }
                              if (!res.issuerPartnerId) {
                                message.warning('请先选择签发方');
                                return;
                              }
                              try {
                                const resp =
                                  await orderServiceMatchSeaMasterBillCandidate(
                                    {
                                      masterNo: res.masterNo,
                                      issuerPartnerId: res.issuerPartnerId,
                                    },
                                  );
                                if (resp?.matched && resp.candidate) {
                                  const c = resp.candidate;
                                  const te = c.transportExecution;
                                  if (
                                    !c.id ||
                                    !c.version ||
                                    !te?.id ||
                                    !te.version
                                  ) {
                                    message.error(
                                      '候选母单或运输执行缺少版本信息，无法选择！',
                                    );
                                    return;
                                  }
                                  const updated = [...results];
                                  updated[index] = {
                                    ...res,
                                    masterNo: c.masterNo,
                                    candidateId: c.id,
                                    candidateVersion: String(c.version),
                                    candidateTeId: te.id,
                                    candidateTeVersion: String(te.version),
                                    issuerPartnerId: c.issuerPartnerId,
                                    carrierId: te?.carrierId,
                                    vesselName: te?.vesselName,
                                    voyageNo: te?.voyageNo,
                                    originLocationId: te?.originLocationId,
                                    dischargeLocationId:
                                      te?.dischargeLocationId,
                                    transitLocationId: te?.transitLocationId,
                                    etd: te?.etd
                                      ? dayjs(te.etd).format(
                                          'YYYY-MM-DD HH:mm:ss',
                                        )
                                      : undefined,
                                    eta: te?.eta
                                      ? dayjs(te.eta).format(
                                          'YYYY-MM-DD HH:mm:ss',
                                        )
                                      : undefined,
                                  };
                                  setResults(updated);
                                  message.success(
                                    `成功匹配到共享母单 [${c.masterNo}]，版本: ${c.version}`,
                                  );
                                  triggerPreview(updated);
                                } else {
                                  message.warning('未找到匹配的草稿候选母单');
                                }
                              } catch (error: unknown) {
                                message.error(
                                  getErrorMessage(error, '匹配候选母单失败'),
                                );
                              }
                            }}
                          >
                            匹配已有母单
                          </Button>
                          {res.candidateId && (
                            <Tag color="success">
                              已匹配 ID: {res.candidateId.slice(0, 8)} (v
                              {res.candidateVersion}) {res.vesselName}{' '}
                              {res.voyageNo}
                            </Tag>
                          )}
                        </Space>
                      </div>
                    )}

                    {res.targetType === 'NEW' && (
                      <div
                        style={{
                          marginTop: 12,
                          padding: 12,
                          background: '#ffffff',
                          borderRadius: 4,
                        }}
                      >
                        <Row gutter={12}>
                          <Col span={6}>
                            <Input
                              placeholder="新母单号 (MBL No)"
                              value={res.masterNo}
                              onChange={(e) => {
                                const updated = [...results];
                                updated[index] = {
                                  ...res,
                                  masterNo: e.target.value,
                                };
                                setResults(updated);
                              }}
                            />
                          </Col>
                          <Col span={6}>
                            <Select
                              showSearch
                              placeholder="选择发单人 / 船代"
                              style={{ width: '100%' }}
                              value={res.issuerPartnerId}
                              options={issuerOptions}
                              onSearch={async (k) => {
                                const opts = await searchPartnersByRole(
                                  PARTNER_ROLES.SUPPLIER,
                                  k,
                                );
                                setIssuerOptions(opts);
                              }}
                              onChange={(val) => {
                                const updated = [...results];
                                updated[index] = {
                                  ...res,
                                  issuerPartnerId: val,
                                };
                                setResults(updated);
                              }}
                            />
                          </Col>
                          <Col span={6}>
                            <Input
                              placeholder="船名"
                              value={res.vesselName}
                              onChange={(e) => {
                                const updated = [...results];
                                updated[index] = {
                                  ...res,
                                  vesselName: e.target.value,
                                };
                                setResults(updated);
                              }}
                            />
                          </Col>
                          <Col span={6}>
                            <Input
                              placeholder="航次"
                              value={res.voyageNo}
                              onChange={(e) => {
                                const updated = [...results];
                                updated[index] = {
                                  ...res,
                                  voyageNo: e.target.value,
                                };
                                setResults(updated);
                              }}
                            />
                          </Col>
                        </Row>
                      </div>
                    )}
                  </Card>
                </Col>
              ))}
            </Row>
          </SectionCard>

          {/* 3. 分单分配区块 */}
          <SectionCard
            title={
              <Space>
                <Text strong>分单 (HBL) 分配与集装箱归属</Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  （每个分单必须唯一指派到一个结果票，且同一集装箱内货物不得跨票分配）
                </Text>
              </Space>
            }
          >
            <Table<API.SeaOrderSplitHouseBillItem>
              columns={hblColumns}
              dataSource={splitContext?.houseBills || []}
              rowKey="id"
              pagination={false}
              size="middle"
            />
          </SectionCard>

          {/* 4. 草稿费用分配区块 */}
          <SectionCard
            title={
              <Space>
                <Text strong>草稿费用整行归属</Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  （仅未确认/未账单化的草稿费用可整行转移，税费与汇率快照全量保留）
                </Text>
              </Space>
            }
          >
            <Table<API.SeaOrderSplitDraftFeeItem>
              columns={feeColumns}
              dataSource={splitContext?.draftFees || []}
              rowKey="id"
              pagination={false}
              size="middle"
            />
            {feeCurrencySummaries.length > 0 && (
              <div style={{ marginTop: 16 }}>
                <Text strong>各币种费用实时守恒：</Text>
                <Row gutter={[12, 12]} style={{ marginTop: 8 }}>
                  {feeCurrencySummaries.map((summary) => {
                    const assigned = Object.values(
                      summary.assignedByResult,
                    ).reduce(
                      (total, amount) => total.add(amount),
                      new Decimal(0),
                    );
                    const remainingColor = summary.remaining.isZero()
                      ? 'success'
                      : summary.remaining.isPositive()
                        ? 'processing'
                        : 'error';
                    return (
                      <Col span={12} key={summary.key}>
                        <Card
                          size="small"
                          type="inner"
                          title={`${summary.direction === 'RECEIVABLE' ? '应收' : '应付'} ${summary.currency}`}
                          extra={
                            <Tag color={remainingColor}>
                              {summary.remaining.isZero()
                                ? '已完整归属'
                                : '存在归属差额'}
                            </Tag>
                          }
                        >
                          <div>
                            基准：{summary.currency}{' '}
                            {summary.baseline.toString()}
                          </div>
                          <div>
                            已分配：{summary.currency} {assigned.toString()}
                          </div>
                          <div>
                            剩余：{summary.currency}{' '}
                            {summary.remaining.toString()}
                          </div>
                          <div style={{ marginTop: 6 }}>
                            {results.map((result) => (
                              <Tag key={result.key}>
                                {result.title}：{summary.currency}{' '}
                                {summary.assignedByResult[
                                  result.key
                                ]?.toString() ?? '0'}
                              </Tag>
                            ))}
                          </div>
                        </Card>
                      </Col>
                    );
                  })}
                </Row>
              </div>
            )}
          </SectionCard>

          {/* 5. 附件引用继承区块 */}
          <SectionCard
            title={
              <Space>
                <Text strong>单证附件共享引用</Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  （勾选后新票将建立对该物理资产的关联引用，解除任一单票引用不影响底层文件）
                </Text>
              </Space>
            }
          >
            <Table<API.SeaOrderSplitAttachmentItem>
              columns={attColumns}
              dataSource={splitContext?.attachments || []}
              rowKey="id"
              pagination={false}
              size="middle"
            />
          </SectionCard>

          {/* 6. 拆票说明区块 */}
          <SectionCard title="拆票说明（可选）">
            <TextArea
              rows={2}
              maxLength={500}
              showCount
              placeholder="可填写本次拆票说明（将永久记录在不可变拆票事件历史中）"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </SectionCard>

          {/* 7. 实时守恒与重算校验区块 */}
          <SectionCard
            title="实时守恒与配载重算校验"
            extra={
              previewing ? (
                <Spin size="small" />
              ) : previewData?.conservationPassed && previewData?.isValid ? (
                <Tag color="success" icon={<CheckCircleOutlined />}>
                  守恒与门禁校验通过
                </Tag>
              ) : (
                <Tag color="error" icon={<ExclamationCircleOutlined />}>
                  校验未通过
                </Tag>
              )
            }
          >
            {previewData?.validationErrors &&
              previewData.validationErrors.length > 0 && (
                <Alert
                  type="error"
                  showIcon
                  message="阻断原因提示"
                  description={
                    <div>
                      {previewData.validationErrors.map((err) => (
                        <div key={`${err.reason}-${err.message}`}>
                          <Text strong>[{err.reason}]</Text> <span>{err.message}</span>
                        </div>
                      ))}
                    </div>
                  }
                  style={{ marginBottom: 16 }}
                />
              )}

            <Row gutter={16}>
              <Col span={8}>
                <Card size="small" title="原始基线总量" type="inner">
                  <div>
                    件数：{previewData?.baseline?.packageCount ?? '-'} 件
                  </div>
                  <div>
                    毛重：{previewData?.baseline?.grossWeightKg ?? '-'} KGS
                  </div>
                  <div>体积：{previewData?.baseline?.volumeCbm ?? '-'} CBM</div>
                </Card>
              </Col>
              <Col span={8}>
                <Card size="small" title="各票分配累计" type="inner">
                  <div>
                    件数：{previewData?.allocated?.packageCount ?? '-'} 件
                  </div>
                  <div>
                    毛重：{previewData?.allocated?.grossWeightKg ?? '-'} KGS
                  </div>
                  <div>
                    体积：{previewData?.allocated?.volumeCbm ?? '-'} CBM
                  </div>
                </Card>
              </Col>
              <Col span={8}>
                {(() => {
                  const remPkg = previewData?.remaining?.packageCount;
                  let pkgColor = '#8c8c8c';
                  let pkgBg = '#fafafa';
                  let pkgStatusText = '未计算';
                  if (remPkg !== undefined && remPkg !== null) {
                    if (remPkg === 0) {
                      pkgColor = '#52c41a';
                      pkgBg = '#f6ffed';
                      pkgStatusText = '已完全分配 (守恒)';
                    } else if (remPkg > 0) {
                      pkgColor = '#1677ff';
                      pkgBg = '#e6f4ff';
                      pkgStatusText = `分配进行中 (待分配 ${remPkg} 件)`;
                    } else {
                      pkgColor = '#f5222d';
                      pkgBg = '#fff1f0';
                      pkgStatusText = `分配超出 (超出 ${Math.abs(remPkg)} 件)`;
                    }
                  }

                  const remainingWeightValue =
                    previewData?.remaining?.grossWeightKg;
                  const remWt =
                    remainingWeightValue === undefined
                      ? undefined
                      : new Decimal(remainingWeightValue);
                  let wtColor = '#8c8c8c';
                  let wtStatusText = '未计算';
                  if (remWt) {
                    if (remWt.isZero()) {
                      wtColor = '#52c41a';
                      wtStatusText = '已完全分配 (守恒)';
                    } else if (remWt.isPositive()) {
                      wtColor = '#1677ff';
                      wtStatusText = `进行中 (待分配 ${remainingWeightValue} KGS)`;
                    } else {
                      wtColor = '#f5222d';
                      wtStatusText = `超出分配 (超出 ${remWt.abs().toFixed(3)} KGS)`;
                    }
                  }

                  const remainingVolumeValue =
                    previewData?.remaining?.volumeCbm;
                  const remVol =
                    remainingVolumeValue === undefined
                      ? undefined
                      : new Decimal(remainingVolumeValue);
                  let volColor = '#8c8c8c';
                  let volStatusText = '未计算';
                  if (remVol) {
                    if (remVol.isZero()) {
                      volColor = '#52c41a';
                      volStatusText = '已完全分配 (守恒)';
                    } else if (remVol.isPositive()) {
                      volColor = '#1677ff';
                      volStatusText = `进行中 (待分配 ${remainingVolumeValue} CBM)`;
                    } else {
                      volColor = '#f5222d';
                      volStatusText = `超出分配 (超出 ${remVol.abs().toFixed(6)} CBM)`;
                    }
                  }

                  return (
                    <Card
                      size="small"
                      title="未分配差额 (零误差守恒校验)"
                      type="inner"
                      style={{ background: pkgBg }}
                    >
                      <div style={{ marginBottom: 4 }}>
                        件数差额：
                        <Text strong style={{ color: pkgColor }}>
                          {previewData?.remaining?.packageCount ?? '-'} 件
                        </Text>
                        <Tag
                          color={
                            remPkg === 0
                              ? 'success'
                              : remPkg && remPkg > 0
                                ? 'processing'
                                : 'error'
                          }
                          style={{ marginLeft: 8 }}
                        >
                          {pkgStatusText}
                        </Tag>
                      </div>
                      <div style={{ marginBottom: 4 }}>
                        毛重差额：
                        <Text strong style={{ color: wtColor }}>
                          {previewData?.remaining?.grossWeightKg ?? '-'} KGS
                        </Text>
                        <Tag
                          color={
                            remWt?.isZero()
                              ? 'success'
                              : remWt?.isPositive()
                                ? 'processing'
                                : 'error'
                          }
                          style={{ marginLeft: 8 }}
                        >
                          {wtStatusText}
                        </Tag>
                      </div>
                      <div>
                        体积差额：
                        <Text strong style={{ color: volColor }}>
                          {previewData?.remaining?.volumeCbm ?? '-'} CBM
                        </Text>
                        <Tag
                          color={
                            remVol?.isZero()
                              ? 'success'
                              : remVol?.isPositive()
                                ? 'processing'
                                : 'error'
                          }
                          style={{ marginLeft: 8 }}
                        >
                          {volStatusText}
                        </Tag>
                      </div>
                    </Card>
                  );
                })()}
              </Col>
            </Row>

            {previewData?.results && (
              <div style={{ marginTop: 16 }}>
                <Text
                  strong
                  style={{ fontSize: 13, marginBottom: 8, display: 'block' }}
                >
                  拆票结果票规划与集装箱计划自动重算：
                </Text>
                <Row gutter={[12, 12]}>
                  {previewData.results.map((pr) => (
                    <Col span={12} key={pr.clientResultKey}>
                      <Card
                        size="small"
                        type="inner"
                        title={`${pr.resultRole === 'ORIGINAL' ? '原票' : '新票'}: ${pr.clientResultKey}`}
                      >
                        <div>
                          分配货物：{pr.packageCount} 件 / {pr.grossWeightKg}{' '}
                          KGS / {pr.volumeCbm} CBM
                        </div>
                        <div>
                          分配分单：{pr.houseBillCount} 票 | 归属费用：
                          {pr.feeCount} 笔
                        </div>
                        <div style={{ marginTop: 6 }}>
                          <Text type="secondary">自动重算箱计划：</Text>
                          {pr.containerPlans && pr.containerPlans.length > 0 ? (
                            pr.containerPlans.map((cp) => (
                              <Tag
                                key={cp.containerSpecId || cp.containerSpecName}
                                color="blue"
                              >
                                {cp.containerSpecName}: {cp.quantity} 箱
                              </Tag>
                            ))
                          ) : (
                            <Text type="secondary">无箱计划</Text>
                          )}
                        </div>
                      </Card>
                    </Col>
                  ))}
                </Row>
              </div>
            )}
          </SectionCard>
        </Space>
        </div>
      </div>

      {/* 底部吸底操作栏 StickyFooterBar */}
      <StickyFooterBar
        info={
          <Space size="large">
            <Text strong>拆票规划：{results.length} 票</Text>
            <Text type="secondary">
              分配进度：{previewData?.allocated?.packageCount ?? 0} /{' '}
              {previewData?.baseline?.packageCount ?? 0} 件
            </Text>
            {previewData?.conservationPassed && previewData?.isValid ? (
              <Tag color="success">守恒验证通过</Tag>
            ) : (
              <Tag color="error">等待满足守恒条件</Tag>
            )}
          </Space>
        }
      >
        <Button onClick={() => history.push(`/orders/sea-export/${orderId}`)}>
          取消返回
        </Button>
        <Popconfirm
          title="确认提交拆票"
          description="拆票将原子锁定订单、分批移转货物、分单与费用并创建新订单。确定提交执行？"
          okText="确认执行"
          cancelText="取消"
          onConfirm={handleExecuteSplit}
          disabled={
            lockWritePolicy.disabled ||
            !previewData?.isValid ||
            !previewData?.conservationPassed
          }
        >
          <Button
            type="primary"
            loading={submitting}
            disabled={
              lockWritePolicy.disabled ||
              !previewData?.isValid ||
              !previewData?.conservationPassed
            }
          >
            确认执行拆票
          </Button>
        </Popconfirm>
      </StickyFooterBar>
    </Spin>
  );
}
