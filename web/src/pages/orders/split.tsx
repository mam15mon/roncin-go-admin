import {
  CheckCircleOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Input,
  InputNumber,
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
import { OrderBusinessType } from '@/enums.generated';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaOrderSplit,
  seaOrderChangeServiceGetSeaOrderSplitContext,
  seaOrderChangeServicePreviewSeaOrderSplit,
} from '@/services/roncin/seaOrderChangeService';
import { computeCanonicalSha256 } from '@/utils/hash';
import { searchShippingLineOptions } from '@/utils/options';
import OrderPageHeader from './components/OrderPageHeader';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from './templates/components/sea/SeaExternalConfirmationFields';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

const { Text } = Typography;
const { TextArea } = Input;

// 纯函数与类型已抽至 splitUtils；此处 re-export 维持测试与既有导入路径稳定。
export {
  buildSeaOrderSplitTargets,
  calculateFeeCurrencySummaries,
  type ResultConfig,
} from './splitUtils';

import {
  buildSeaOrderSplitTargets,
  calculateFeeCurrencySummaries,
  type FeeCurrencySummary,
  getErrorMessage,
  type ResultConfig,
} from './splitUtils';

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
  const [containerAssignments, setContainerAssignments] = useState<
    Record<string, string>
  >({}); // containerId -> resultKey
  const [cargoAllocations, setCargoAllocations] = useState<
    Record<
      string,
      Record<
        string,
        { packageCount: number; grossWeightKg: string; volumeCbm: string }
      >
    >
  >({}); // cargoItemId -> resultKey -> { packageCount, grossWeightKg, volumeCbm }
  const [sharedAllocations, setSharedAllocations] = useState<
    Record<
      string,
      Record<
        string,
        { packageCount: number; grossWeightKg: string; volumeCbm: string }
      >
    >
  >({}); // allocationId -> resultKey -> { packageCount, grossWeightKg, volumeCbm }
  const [feeAssignments, setFeeAssignments] = useState<Record<string, string>>(
    {},
  ); // feeId -> resultKey
  const [attAssignments, setAttAssignments] = useState<
    Record<string, string[]>
  >({}); // attId -> resultKeys[]
  const initialPreviewTriggeredRef = useRef(false);
  const [note, setNote] = useState<string>('');
  const [confirmationForm] = Form.useForm<SeaExternalConfirmationFormValues>();

  // 预览与校验结果
  const [previewing, setPreviewing] = useState(false);
  const [previewData, setPreviewData] =
    useState<API.SeaOrderSplitPreviewData | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // 下拉选项
  const [carrierOptions, setCarrierOptions] = useState<DefaultOptionType[]>([]);

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
        if (ctx.currentMasterBill?.shippingLineId) {
          setCarrierOptions([
            {
              label:
                ctx.currentMasterBill.shippingLineName ||
                ctx.currentMasterBill.shippingLineId,
              value: ctx.currentMasterBill.shippingLineId,
            },
          ]);
        }

        const defaultHouseNo = ctx.currentHouseBill?.houseNo
          ? `${ctx.currentHouseBill.houseNo}-1`
          : 'HBL-1';

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
            houseNo: defaultHouseNo,
            issuerSource: 'SELF_ORGANIZATION',
          },
        ];
        setResults(initialResults);

        // 初始化集装箱整箱归属：默认全在原票
        const initialCntrMap: Record<string, string> = {};
        ctx.containers?.forEach((c: API.SeaOrderSplitContainerItem) => {
          if (c.id) {
            initialCntrMap[c.id] = 'res-origin';
          }
        });
        setContainerAssignments(initialCntrMap);

        // 初始化货物件重尺分配：原票全量，新票 0
        const initialCargoAlloc: Record<
          string,
          Record<
            string,
            { packageCount: number; grossWeightKg: string; volumeCbm: string }
          >
        > = {};
        ctx.cargoItems?.forEach((ci: API.SeaOrderSplitCargoItem) => {
          if (ci.id) {
            initialCargoAlloc[ci.id] = {
              'res-origin': {
                packageCount: ci.packageCount || 0,
                grossWeightKg: String(ci.grossWeightKg || '0'),
                volumeCbm: String(ci.volumeCbm || '0'),
              },
              'res-new-1': {
                packageCount: 0,
                grossWeightKg: '0',
                volumeCbm: '0',
              },
            };
          }
        });
        setCargoAllocations(initialCargoAlloc);

        // 初始化共享箱分配：原票全量，新票 0
        const initialSharedAlloc: Record<
          string,
          Record<
            string,
            { packageCount: number; grossWeightKg: string; volumeCbm: string }
          >
        > = {};
        ctx.sharedContainerAllocations?.forEach(
          (sa: API.SeaOrderSplitSharedContainerAllocationItem) => {
            if (sa.allocationId) {
              initialSharedAlloc[sa.allocationId] = {
                'res-origin': {
                  packageCount: sa.packageCount || 0,
                  grossWeightKg: String(sa.grossWeightKg || '0'),
                  volumeCbm: String(sa.volumeCbm || '0'),
                },
                'res-new-1': {
                  packageCount: 0,
                  grossWeightKg: '0',
                  volumeCbm: '0',
                },
              };
            }
          },
        );
        setSharedAllocations(initialSharedAlloc);

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

  // 页签占位标题为中性「订单拆票」，加载成功后回填带单号的真实标题。
  useEffect(() => {
    if (splitContext?.orderNo && orderId && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: `/orders/sea-export/${orderId}/split`,
            title: `${splitContext.orderNo}_拆票`,
          },
        }),
      );
    }
  }, [splitContext?.orderNo, orderId]);

  // 构造母单目标
  const buildTargets = (
    currentResults = results,
  ): API.SeaOrderSplitTargetInput[] =>
    buildSeaOrderSplitTargets(currentResults);

  // 构造拆票结果明细
  const buildSplitResults = (
    currentResults = results,
    currentContainers = containerAssignments,
    currentCargoAllocs = cargoAllocations,
    currentSharedAllocs = sharedAllocations,
    currentFees = feeAssignments,
    currentAtts = attAssignments,
  ): API.SeaOrderSplitResultInput[] => {
    return currentResults.map((r) => {
      const feeIds = Object.entries(currentFees)
        .filter(([, resKey]) => resKey === r.key)
        .map(([fId]) => fId);

      const attIds = Object.entries(currentAtts)
        .filter(([, resKeys]) => resKeys.includes(r.key))
        .map(([aId]) => aId);

      const cIds = Object.entries(currentContainers)
        .filter(([, resKey]) => resKey === r.key)
        .map(([cId]) => cId);

      const cargoAllocs = (splitContext?.cargoItems || []).map((ci) => {
        const a = currentCargoAllocs[ci.id || '']?.[r.key] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        return {
          cargoItemId: ci.id || '',
          packageCount: Number(a.packageCount) || 0,
          grossWeightKg: String(a.grossWeightKg || '0'),
          volumeCbm: String(a.volumeCbm || '0'),
        };
      });

      const sharedAllocs = (splitContext?.sharedContainerAllocations || []).map(
        (sa) => {
          const a = currentSharedAllocs[sa.allocationId || '']?.[r.key] || {
            packageCount: 0,
            grossWeightKg: '0',
            volumeCbm: '0',
          };
          return {
            allocationId: sa.allocationId || '',
            packageCount: Number(a.packageCount) || 0,
            grossWeightKg: String(a.grossWeightKg || '0'),
            volumeCbm: String(a.volumeCbm || '0'),
          };
        },
      );

      let houseBill: API.SeaOrderSplitHouseBillInput | undefined;
      if (r.role === 'CREATED' && splitContext?.documentStructure === 'HOUSE') {
        houseBill = {
          houseNo: r.houseNo || '',
          issuerSource: r.issuerSource || 'SELF_ORGANIZATION',
          issuerPartnerId: r.issuerPartnerId,
          note: r.houseBillNote,
        };
      }

      return {
        clientResultKey: r.key,
        resultRole: r.role,
        clientTargetKey: r.key,
        draftFeeIds: feeIds,
        attachmentReferenceIds: attIds,
        containerIds: cIds,
        cargoAllocations: cargoAllocs,
        sharedContainerAllocations:
          sharedAllocs.length > 0 ? sharedAllocs : undefined,
        houseBill,
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
    if (!splitContext.orderVersion || !splitContext.currentLinkVersion) {
      return undefined;
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
    const sharedContainerVersions: Record<string, string> = {};
    for (const sa of splitContext.sharedContainerAllocations || []) {
      if (sa.sharedContainerId && sa.sharedContainerVersion) {
        sharedContainerVersions[sa.sharedContainerId] = String(
          sa.sharedContainerVersion,
        );
      }
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
      currentHblVersion: splitContext.currentHouseBill?.version
        ? String(splitContext.currentHouseBill.version)
        : undefined,
      cargoItemVersions,
      containerVersions,
      feeVersions,
      sharedContainerVersions:
        Object.keys(sharedContainerVersions).length > 0
          ? sharedContainerVersions
          : undefined,
      candidateMblVersions,
      candidateTeVersions,
      attachmentReferenceFingerprint:
        splitContext.attachmentReferenceFingerprint,
    };
  };

  // 触发校验与预览
  const triggerPreview = async (
    currentResults = results,
    currentContainers = containerAssignments,
    currentCargoAllocs = cargoAllocations,
    currentSharedAllocs = sharedAllocations,
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
        currentContainers,
        currentCargoAllocs,
        currentSharedAllocs,
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
    containerAssignments,
    cargoAllocations,
    sharedAllocations,
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
    const defaultHouseNo = splitContext?.currentHouseBill?.houseNo
      ? `${splitContext.currentHouseBill.houseNo}-${nextIdx}`
      : `HBL-${nextIdx}`;
    const newRes: ResultConfig = {
      key: newKey,
      role: 'CREATED',
      title: `拆出新票 ${nextIdx}`,
      targetType: 'CURRENT',
      internalReferenceNo: '',
      bookingNotes: splitContext?.bookingNotes,
      allocationNotes: splitContext?.allocationNotes || '',
      operationNotes: splitContext?.operationNotes,
      houseNo: defaultHouseNo,
      issuerSource: 'SELF_ORGANIZATION',
    };
    const updated = [...results, newRes];
    setResults(updated);

    setCargoAllocations((prev) => {
      const next = { ...prev };
      for (const ciId of Object.keys(next)) {
        next[ciId] = {
          ...next[ciId],
          [newKey]: { packageCount: 0, grossWeightKg: '0', volumeCbm: '0' },
        };
      }
      return next;
    });

    setSharedAllocations((prev) => {
      const next = { ...prev };
      for (const saId of Object.keys(next)) {
        next[saId] = {
          ...next[saId],
          [newKey]: { packageCount: 0, grossWeightKg: '0', volumeCbm: '0' },
        };
      }
      return next;
    });
  };

  // 移除新票
  const handleRemoveResult = (key: string) => {
    if (!ensureSplitEditable()) return;
    if (results.filter((r) => r.role === 'CREATED').length <= 1) {
      message.warning('拆票必须至少保留一个拆出新票');
      return;
    }
    const updated = results.filter((r) => r.key !== key);
    const nextContainers = { ...containerAssignments };
    Object.keys(nextContainers).forEach((cId) => {
      if (nextContainers[cId] === key) nextContainers[cId] = 'res-origin';
    });
    const nextFees = { ...feeAssignments };
    Object.keys(nextFees).forEach((fId) => {
      if (nextFees[fId] === key) nextFees[fId] = 'res-origin';
    });
    setResults(updated);
    setContainerAssignments(nextContainers);
    setFeeAssignments(nextFees);

    setCargoAllocations((prev) => {
      const next = { ...prev };
      for (const ci of splitContext?.cargoItems || []) {
        if (!ci.id || !next[ci.id]) continue;
        const removed = next[ci.id][key] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const origin = next[ci.id]['res-origin'] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const pkg =
          (Number(origin.packageCount) || 0) +
          (Number(removed.packageCount) || 0);
        const wt = new Decimal(origin.grossWeightKg || '0')
          .add(new Decimal(removed.grossWeightKg || '0'))
          .toString();
        const vol = new Decimal(origin.volumeCbm || '0')
          .add(new Decimal(removed.volumeCbm || '0'))
          .toString();
        const { [key]: _, ...rest } = next[ci.id];
        next[ci.id] = {
          ...rest,
          'res-origin': {
            packageCount: pkg,
            grossWeightKg: wt,
            volumeCbm: vol,
          },
        };
      }
      return next;
    });

    setSharedAllocations((prev) => {
      const next = { ...prev };
      for (const sa of splitContext?.sharedContainerAllocations || []) {
        if (!sa.allocationId || !next[sa.allocationId]) continue;
        const removed = next[sa.allocationId][key] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const origin = next[sa.allocationId]['res-origin'] || {
          packageCount: 0,
          grossWeightKg: '0',
          volumeCbm: '0',
        };
        const pkg =
          (Number(origin.packageCount) || 0) +
          (Number(removed.packageCount) || 0);
        const wt = new Decimal(origin.grossWeightKg || '0')
          .add(new Decimal(removed.grossWeightKg || '0'))
          .toString();
        const vol = new Decimal(origin.volumeCbm || '0')
          .add(new Decimal(removed.volumeCbm || '0'))
          .toString();
        const { [key]: _, ...rest } = next[sa.allocationId];
        next[sa.allocationId] = {
          ...rest,
          'res-origin': {
            packageCount: pkg,
            grossWeightKg: wt,
            volumeCbm: vol,
          },
        };
      }
      return next;
    });
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
    // 任一结果换入其他母单（含新建）即产生内嵌改配，必须携带外部确认
    let confirmation: API.SeaExternalConfirmationInput | undefined;
    if (results.some((r) => r.targetType !== 'CURRENT')) {
      try {
        confirmation = buildSeaExternalConfirmation(
          await confirmationForm.validateFields(),
        );
      } catch {
        message.error('请完整填写承运方外部确认信息后再执行拆票');
        return;
      }
    }
    setSubmitting(true);
    try {
      const targets = buildTargets(results);
      const splitResults = buildSplitResults(
        results,
        containerAssignments,
        cargoAllocations,
        sharedAllocations,
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
        confirmation,
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
          confirmation,
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

  // 独占箱整箱归属列
  const containerColumns: ColumnsType<API.SeaOrderSplitContainerItem> = [
    {
      title: '箱号',
      dataIndex: 'containerNo',
      render: (val) => <Text strong>{val || '-'}</Text>,
    },
    {
      title: '箱型规格',
      dataIndex: 'containerSpecName',
      render: (val) => val || '-',
    },
    {
      title: '货物统计 (件/重/尺)',
      render: (_, c) => (
        <span>
          {c.packageCount ?? 0} 件 / {c.grossWeightKg ?? 0} KGS /{' '}
          {c.volumeCbm ?? 0} CBM
        </span>
      ),
    },
    {
      title: '整箱归属结果票',
      width: 260,
      render: (_, c) => (
        <Select
          value={c.id ? containerAssignments[c.id] : undefined}
          style={{ width: '100%' }}
          onChange={(val) => {
            if (c.id) {
              setContainerAssignments({
                ...containerAssignments,
                [c.id]: val,
              });
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
    <PageContainer
      title={false}
      breadcrumbRender={false}
      header={{ title: false, style: { padding: 0 } }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      <Spin spinning={loadingContext}>
        {/* 顶部 OrderPageHeader */}
        <OrderPageHeader
          page="split"
          orderKind="sea-export"
          navigationTitle="海运出口"
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

        <div style={{ paddingBottom: 80 }}>
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
                      title="货物项总数"
                      value={splitContext?.cargoItems?.length || 0}
                      suffix="项"
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
                                  shippingLineId:
                                    val === 'CURRENT'
                                      ? undefined
                                      : res.shippingLineId ||
                                        splitContext?.currentMasterBill
                                          ?.shippingLineId,
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
                              {canReassign && (
                                <Radio value="NEW">录入新母单</Radio>
                              )}
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

                        {res.role === 'ORIGINAL' ? (
                          <div
                            style={{
                              marginTop: 12,
                              padding: 8,
                              background: '#f5f5f5',
                              borderRadius: 4,
                            }}
                          >
                            <Text type="secondary">
                              保留单分单号 (HBL No)：
                            </Text>
                            <Text strong style={{ marginLeft: 8 }}>
                              {splitContext?.currentHouseBill?.houseNo ||
                                '无分单（直单模式）'}
                            </Text>
                          </div>
                        ) : (
                          splitContext?.documentStructure === 'HOUSE' && (
                            <div
                              style={{
                                marginTop: 12,
                                padding: 12,
                                background: '#ffffff',
                                borderRadius: 4,
                                border: '1px dashed #91caff',
                              }}
                            >
                              <Row gutter={12}>
                                <Col span={8}>
                                  <div style={{ marginBottom: 4 }}>
                                    <Text strong>新分单号 (HBL No) *</Text>
                                  </div>
                                  <Input
                                    placeholder="新分单号"
                                    value={res.houseNo}
                                    onChange={(e) => {
                                      const updated = [...results];
                                      updated[index] = {
                                        ...res,
                                        houseNo: e.target.value,
                                      };
                                      setResults(updated);
                                    }}
                                  />
                                </Col>
                                <Col span={8}>
                                  <div style={{ marginBottom: 4 }}>
                                    <Text strong>签发主体</Text>
                                  </div>
                                  <Select
                                    value={
                                      res.issuerSource || 'SELF_ORGANIZATION'
                                    }
                                    onChange={(val) => {
                                      const updated = [...results];
                                      updated[index] = {
                                        ...res,
                                        issuerSource: val,
                                      };
                                      setResults(updated);
                                    }}
                                    style={{ width: '100%' }}
                                    options={[
                                      {
                                        label: '组织自签 (SELF_ORGANIZATION)',
                                        value: 'SELF_ORGANIZATION',
                                      },
                                      {
                                        label: '客户代签 (CUSTOMER_PARTNER)',
                                        value: 'CUSTOMER_PARTNER',
                                      },
                                      {
                                        label: '第三方代签 (OTHER_PARTNER)',
                                        value: 'OTHER_PARTNER',
                                      },
                                    ]}
                                  />
                                </Col>
                                <Col span={8}>
                                  <div style={{ marginBottom: 4 }}>
                                    <Text strong>分单备注</Text>
                                  </div>
                                  <Input
                                    placeholder="可选分单备注"
                                    value={res.houseBillNote}
                                    onChange={(e) => {
                                      const updated = [...results];
                                      updated[index] = {
                                        ...res,
                                        houseBillNote: e.target.value,
                                      };
                                      setResults(updated);
                                    }}
                                  />
                                </Col>
                              </Row>
                            </div>
                          )
                        )}

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
                              <Select
                                showSearch
                                placeholder="选择船公司"
                                style={{ width: 220 }}
                                value={res.shippingLineId}
                                options={carrierOptions}
                                filterOption={false}
                                onSearch={async (keyword) => {
                                  const options =
                                    await searchShippingLineOptions(keyword);
                                  setCarrierOptions(options);
                                }}
                                onChange={(value) => {
                                  const updated = [...results];
                                  updated[index] = {
                                    ...res,
                                    shippingLineId: value,
                                    candidateId: undefined,
                                    candidateVersion: undefined,
                                    candidateTeId: undefined,
                                    candidateTeVersion: undefined,
                                  };
                                  setResults(updated);
                                }}
                              />
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
                                  if (!res.shippingLineId) {
                                    message.warning('请先选择船公司');
                                    return;
                                  }
                                  try {
                                    const resp =
                                      await orderServiceMatchSeaMasterBillCandidate(
                                        {
                                          masterNo: res.masterNo,
                                          shippingLineId: res.shippingLineId,
                                        },
                                      );
                                    if (resp?.matched && resp.candidate) {
                                      const c = resp.candidate;
                                      const transportExecutions =
                                        c.transportExecutions ?? [];
                                      if (transportExecutions.length !== 1) {
                                        message.error(
                                          '该 MBL 存在多个实际航次，请改用整票改配明确选择目标航次',
                                        );
                                        return;
                                      }
                                      const te = transportExecutions[0];
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
                                        shippingLineId: te?.shippingLineId,
                                        vesselName: te?.vesselName,
                                        voyageNo: te?.voyageNo,
                                        originLocationId: te?.originLocationId,
                                        dischargeLocationId:
                                          te?.dischargeLocationId,
                                        transitLocationId:
                                          te?.transitLocationId,
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
                                      message.warning(
                                        '未找到匹配的草稿候选母单',
                                      );
                                    }
                                  } catch (error: unknown) {
                                    message.error(
                                      getErrorMessage(
                                        error,
                                        '匹配候选母单失败',
                                      ),
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
                                  placeholder="选择船公司"
                                  style={{ width: '100%' }}
                                  value={res.shippingLineId}
                                  options={carrierOptions}
                                  filterOption={false}
                                  onSearch={async (keyword) => {
                                    const opts =
                                      await searchShippingLineOptions(keyword);
                                    setCarrierOptions(opts);
                                  }}
                                  onChange={(val) => {
                                    const updated = [...results];
                                    updated[index] = {
                                      ...res,
                                      shippingLineId: val,
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

              {/* 3. 集装箱与货物分配切分区块 */}
              <SectionCard
                title={
                  <Space>
                    <Text strong>集装箱与货物明细切分</Text>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      （独占箱整箱归属单一结果票；各货物项件重尺需严格守恒切分）
                    </Text>
                  </Space>
                }
              >
                {splitContext?.containers &&
                  splitContext.containers.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Text
                        strong
                        style={{ marginBottom: 8, display: 'block' }}
                      >
                        独占集装箱整箱归属：
                      </Text>
                      <Table<API.SeaOrderSplitContainerItem>
                        columns={containerColumns}
                        dataSource={splitContext.containers}
                        rowKey="id"
                        pagination={false}
                        size="small"
                      />
                    </div>
                  )}

                <div>
                  <Text strong style={{ marginBottom: 8, display: 'block' }}>
                    货物明细与件重尺分配：
                  </Text>
                  <Space
                    direction="vertical"
                    style={{ width: '100%' }}
                    size="middle"
                  >
                    {(splitContext?.cargoItems || []).map((ci) => {
                      if (!ci.id) return null;
                      const currentAllocMap = cargoAllocations[ci.id] || {};
                      const totalAllocPkg = results.reduce(
                        (acc, r) =>
                          acc +
                          (Number(currentAllocMap[r.key]?.packageCount) || 0),
                        0,
                      );
                      const totalAllocWt = results.reduce(
                        (acc, r) =>
                          acc.add(
                            new Decimal(
                              currentAllocMap[r.key]?.grossWeightKg || '0',
                            ),
                          ),
                        new Decimal(0),
                      );
                      const totalAllocVol = results.reduce(
                        (acc, r) =>
                          acc.add(
                            new Decimal(
                              currentAllocMap[r.key]?.volumeCbm || '0',
                            ),
                          ),
                        new Decimal(0),
                      );

                      const baselinePkg = ci.packageCount || 0;
                      const baselineWt = new Decimal(ci.grossWeightKg || '0');
                      const baselineVol = new Decimal(ci.volumeCbm || '0');

                      const isPkgEqual = totalAllocPkg === baselinePkg;
                      const isWtEqual = totalAllocWt.equals(baselineWt);
                      const isVolEqual = totalAllocVol.equals(baselineVol);
                      const isAllConserved =
                        isPkgEqual && isWtEqual && isVolEqual;

                      const fillRemaining = (targetKey: string) => {
                        const otherPkg = results
                          .filter((r) => r.key !== targetKey)
                          .reduce(
                            (acc, r) =>
                              acc +
                              (Number(currentAllocMap[r.key]?.packageCount) ||
                                0),
                            0,
                          );
                        const otherWt = results
                          .filter((r) => r.key !== targetKey)
                          .reduce(
                            (acc, r) =>
                              acc.add(
                                new Decimal(
                                  currentAllocMap[r.key]?.grossWeightKg || '0',
                                ),
                              ),
                            new Decimal(0),
                          );
                        const otherVol = results
                          .filter((r) => r.key !== targetKey)
                          .reduce(
                            (acc, r) =>
                              acc.add(
                                new Decimal(
                                  currentAllocMap[r.key]?.volumeCbm || '0',
                                ),
                              ),
                            new Decimal(0),
                          );

                        const remPkg = Math.max(0, baselinePkg - otherPkg);
                        const remWt = Decimal.max(
                          0,
                          baselineWt.sub(otherWt),
                        ).toFixed(3);
                        const remVol = Decimal.max(
                          0,
                          baselineVol.sub(otherVol),
                        ).toFixed(6);

                        setCargoAllocations((prev) => ({
                          ...prev,
                          [ci.id as string]: {
                            ...(prev[ci.id as string] || {}),
                            [targetKey]: {
                              packageCount: remPkg,
                              grossWeightKg: remWt,
                              volumeCbm: remVol,
                            },
                          },
                        }));
                      };

                      return (
                        <Card
                          key={ci.id}
                          size="small"
                          type="inner"
                          title={
                            <Space>
                              <Text strong>{ci.cargoName || '货物项'}</Text>
                              <Text type="secondary">
                                （基准总量：{baselinePkg} 件 /{' '}
                                {ci.grossWeightKg} KGS / {ci.volumeCbm} CBM）
                              </Text>
                            </Space>
                          }
                          extra={
                            isAllConserved ? (
                              <Tag
                                color="success"
                                icon={<CheckCircleOutlined />}
                              >
                                件重尺守恒
                              </Tag>
                            ) : (
                              <Tag color="error">
                                差额: {baselinePkg - totalAllocPkg} 件 /{' '}
                                {baselineWt.sub(totalAllocWt).toFixed(3)} KGS /{' '}
                                {baselineVol.sub(totalAllocVol).toFixed(6)} CBM
                              </Tag>
                            )
                          }
                        >
                          <Table
                            dataSource={results}
                            rowKey="key"
                            pagination={false}
                            size="small"
                            columns={[
                              {
                                title: '结果票',
                                width: 200,
                                render: (_, r) => (
                                  <span>
                                    <Tag
                                      color={
                                        r.role === 'ORIGINAL'
                                          ? 'default'
                                          : 'blue'
                                      }
                                    >
                                      {r.role === 'ORIGINAL' ? '原' : '新'}
                                    </Tag>
                                    {r.title}
                                  </span>
                                ),
                              },
                              {
                                title: '分配件数',
                                width: 160,
                                render: (_, r) => (
                                  <InputNumber
                                    min={0}
                                    max={baselinePkg}
                                    value={
                                      currentAllocMap[r.key]?.packageCount ?? 0
                                    }
                                    onChange={(val) => {
                                      setCargoAllocations((prev) => ({
                                        ...prev,
                                        [ci.id as string]: {
                                          ...(prev[ci.id as string] || {}),
                                          [r.key]: {
                                            packageCount: Number(val) || 0,
                                            grossWeightKg:
                                              currentAllocMap[r.key]
                                                ?.grossWeightKg ?? '0',
                                            volumeCbm:
                                              currentAllocMap[r.key]
                                                ?.volumeCbm ?? '0',
                                          },
                                        },
                                      }));
                                    }}
                                  />
                                ),
                              },
                              {
                                title: '分配毛重 (KGS)',
                                width: 180,
                                render: (_, r) => (
                                  <Input
                                    value={
                                      currentAllocMap[r.key]?.grossWeightKg ??
                                      '0'
                                    }
                                    onChange={(e) => {
                                      const val = e.target.value;
                                      setCargoAllocations((prev) => ({
                                        ...prev,
                                        [ci.id as string]: {
                                          ...(prev[ci.id as string] || {}),
                                          [r.key]: {
                                            packageCount:
                                              currentAllocMap[r.key]
                                                ?.packageCount ?? 0,
                                            grossWeightKg: val,
                                            volumeCbm:
                                              currentAllocMap[r.key]
                                                ?.volumeCbm ?? '0',
                                          },
                                        },
                                      }));
                                    }}
                                  />
                                ),
                              },
                              {
                                title: '分配体积 (CBM)',
                                width: 180,
                                render: (_, r) => (
                                  <Input
                                    value={
                                      currentAllocMap[r.key]?.volumeCbm ?? '0'
                                    }
                                    onChange={(e) => {
                                      const val = e.target.value;
                                      setCargoAllocations((prev) => ({
                                        ...prev,
                                        [ci.id as string]: {
                                          ...(prev[ci.id as string] || {}),
                                          [r.key]: {
                                            packageCount:
                                              currentAllocMap[r.key]
                                                ?.packageCount ?? 0,
                                            grossWeightKg:
                                              currentAllocMap[r.key]
                                                ?.grossWeightKg ?? '0',
                                            volumeCbm: val,
                                          },
                                        },
                                      }));
                                    }}
                                  />
                                ),
                              },
                              {
                                title: '快捷操作',
                                render: (_, r) => (
                                  <Button
                                    size="small"
                                    type="link"
                                    onClick={() => fillRemaining(r.key)}
                                  >
                                    填入剩余
                                  </Button>
                                ),
                              },
                            ]}
                          />
                        </Card>
                      );
                    })}
                  </Space>
                </div>

                {splitContext?.sharedContainerAllocations &&
                  splitContext.sharedContainerAllocations.length > 0 && (
                    <div style={{ marginTop: 24 }}>
                      <Text
                        strong
                        style={{ marginBottom: 8, display: 'block' }}
                      >
                        跨订单共享箱分配切分：
                      </Text>
                      <Space
                        direction="vertical"
                        style={{ width: '100%' }}
                        size="middle"
                      >
                        {splitContext.sharedContainerAllocations.map((sa) => {
                          if (!sa.allocationId) return null;
                          const currentAllocMap =
                            sharedAllocations[sa.allocationId] || {};
                          const totalAllocPkg = results.reduce(
                            (acc, r) =>
                              acc +
                              (Number(currentAllocMap[r.key]?.packageCount) ||
                                0),
                            0,
                          );
                          const totalAllocWt = results.reduce(
                            (acc, r) =>
                              acc.add(
                                new Decimal(
                                  currentAllocMap[r.key]?.grossWeightKg || '0',
                                ),
                              ),
                            new Decimal(0),
                          );
                          const totalAllocVol = results.reduce(
                            (acc, r) =>
                              acc.add(
                                new Decimal(
                                  currentAllocMap[r.key]?.volumeCbm || '0',
                                ),
                              ),
                            new Decimal(0),
                          );

                          const baselinePkg = sa.packageCount || 0;
                          const baselineWt = new Decimal(
                            sa.grossWeightKg || '0',
                          );
                          const baselineVol = new Decimal(sa.volumeCbm || '0');

                          const isAllConserved =
                            totalAllocPkg === baselinePkg &&
                            totalAllocWt.equals(baselineWt) &&
                            totalAllocVol.equals(baselineVol);

                          const fillSharedRemaining = (targetKey: string) => {
                            const otherPkg = results
                              .filter((r) => r.key !== targetKey)
                              .reduce(
                                (acc, r) =>
                                  acc +
                                  (Number(
                                    currentAllocMap[r.key]?.packageCount,
                                  ) || 0),
                                0,
                              );
                            const otherWt = results
                              .filter((r) => r.key !== targetKey)
                              .reduce(
                                (acc, r) =>
                                  acc.add(
                                    new Decimal(
                                      currentAllocMap[r.key]?.grossWeightKg ||
                                        '0',
                                    ),
                                  ),
                                new Decimal(0),
                              );
                            const otherVol = results
                              .filter((r) => r.key !== targetKey)
                              .reduce(
                                (acc, r) =>
                                  acc.add(
                                    new Decimal(
                                      currentAllocMap[r.key]?.volumeCbm || '0',
                                    ),
                                  ),
                                new Decimal(0),
                              );

                            const remPkg = Math.max(0, baselinePkg - otherPkg);
                            const remWt = Decimal.max(
                              0,
                              baselineWt.sub(otherWt),
                            ).toFixed(3);
                            const remVol = Decimal.max(
                              0,
                              baselineVol.sub(otherVol),
                            ).toFixed(6);

                            setSharedAllocations((prev) => ({
                              ...prev,
                              [sa.allocationId as string]: {
                                ...(prev[sa.allocationId as string] || {}),
                                [targetKey]: {
                                  packageCount: remPkg,
                                  grossWeightKg: remWt,
                                  volumeCbm: remVol,
                                },
                              },
                            }));
                          };

                          return (
                            <Card
                              key={sa.allocationId}
                              size="small"
                              type="inner"
                              title={
                                <Space>
                                  <Text strong>
                                    共享箱: {sa.containerNo || '待配箱号'} (
                                    {sa.containerSpecName || '-'})
                                  </Text>
                                  <Text type="secondary">
                                    （分配基准：{baselinePkg} 件 /{' '}
                                    {sa.grossWeightKg} KGS / {sa.volumeCbm}{' '}
                                    CBM）
                                  </Text>
                                </Space>
                              }
                              extra={
                                isAllConserved ? (
                                  <Tag
                                    color="success"
                                    icon={<CheckCircleOutlined />}
                                  >
                                    守恒满足
                                  </Tag>
                                ) : (
                                  <Tag color="error">
                                    差额: {baselinePkg - totalAllocPkg} 件 /{' '}
                                    {baselineWt.sub(totalAllocWt).toFixed(3)}{' '}
                                    KGS /{' '}
                                    {baselineVol.sub(totalAllocVol).toFixed(6)}{' '}
                                    CBM
                                  </Tag>
                                )
                              }
                            >
                              <Table
                                dataSource={results}
                                rowKey="key"
                                pagination={false}
                                size="small"
                                columns={[
                                  {
                                    title: '结果票',
                                    width: 200,
                                    render: (_, r) => (
                                      <span>
                                        <Tag
                                          color={
                                            r.role === 'ORIGINAL'
                                              ? 'default'
                                              : 'blue'
                                          }
                                        >
                                          {r.role === 'ORIGINAL' ? '原' : '新'}
                                        </Tag>
                                        {r.title}
                                      </span>
                                    ),
                                  },
                                  {
                                    title: '分配件数',
                                    width: 160,
                                    render: (_, r) => (
                                      <InputNumber
                                        min={0}
                                        max={baselinePkg}
                                        value={
                                          currentAllocMap[r.key]
                                            ?.packageCount ?? 0
                                        }
                                        onChange={(val) => {
                                          setSharedAllocations((prev) => ({
                                            ...prev,
                                            [sa.allocationId as string]: {
                                              ...(prev[
                                                sa.allocationId as string
                                              ] || {}),
                                              [r.key]: {
                                                packageCount: Number(val) || 0,
                                                grossWeightKg:
                                                  currentAllocMap[r.key]
                                                    ?.grossWeightKg ?? '0',
                                                volumeCbm:
                                                  currentAllocMap[r.key]
                                                    ?.volumeCbm ?? '0',
                                              },
                                            },
                                          }));
                                        }}
                                      />
                                    ),
                                  },
                                  {
                                    title: '分配毛重 (KGS)',
                                    width: 180,
                                    render: (_, r) => (
                                      <Input
                                        value={
                                          currentAllocMap[r.key]
                                            ?.grossWeightKg ?? '0'
                                        }
                                        onChange={(e) => {
                                          const val = e.target.value;
                                          setSharedAllocations((prev) => ({
                                            ...prev,
                                            [sa.allocationId as string]: {
                                              ...(prev[
                                                sa.allocationId as string
                                              ] || {}),
                                              [r.key]: {
                                                packageCount:
                                                  currentAllocMap[r.key]
                                                    ?.packageCount ?? 0,
                                                grossWeightKg: val,
                                                volumeCbm:
                                                  currentAllocMap[r.key]
                                                    ?.volumeCbm ?? '0',
                                              },
                                            },
                                          }));
                                        }}
                                      />
                                    ),
                                  },
                                  {
                                    title: '分配体积 (CBM)',
                                    width: 180,
                                    render: (_, r) => (
                                      <Input
                                        value={
                                          currentAllocMap[r.key]?.volumeCbm ??
                                          '0'
                                        }
                                        onChange={(e) => {
                                          const val = e.target.value;
                                          setSharedAllocations((prev) => ({
                                            ...prev,
                                            [sa.allocationId as string]: {
                                              ...(prev[
                                                sa.allocationId as string
                                              ] || {}),
                                              [r.key]: {
                                                packageCount:
                                                  currentAllocMap[r.key]
                                                    ?.packageCount ?? 0,
                                                grossWeightKg:
                                                  currentAllocMap[r.key]
                                                    ?.grossWeightKg ?? '0',
                                                volumeCbm: val,
                                              },
                                            },
                                          }));
                                        }}
                                      />
                                    ),
                                  },
                                  {
                                    title: '快捷操作',
                                    render: (_, r) => (
                                      <Button
                                        size="small"
                                        type="link"
                                        onClick={() =>
                                          fillSharedRemaining(r.key)
                                        }
                                      >
                                        填入剩余
                                      </Button>
                                    ),
                                  },
                                ]}
                              />
                            </Card>
                          );
                        })}
                      </Space>
                    </div>
                  )}
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

              {/* 7. 外部确认区块：任一结果换入其他母单（内嵌改配）时必填 */}
              {results.some((r) => r.targetType !== 'CURRENT') && (
                <SectionCard
                  title={
                    <Space>
                      <Text strong>承运方外部确认</Text>
                      <Text type="danger" style={{ fontSize: 12 }}>
                        （检测到结果票换入其他母单，将产生内嵌改配，必须记录外部确认）
                      </Text>
                    </Space>
                  }
                >
                  <Form form={confirmationForm} layout="vertical">
                    <SeaExternalConfirmationFields orderId={orderId} />
                  </Form>
                </SectionCard>
              )}

              {/* 7. 实时守恒与重算校验区块 */}
              <SectionCard
                title="实时守恒与配载重算校验"
                extra={
                  previewing ? (
                    <Spin size="small" />
                  ) : previewData?.conservationPassed &&
                    previewData?.isValid ? (
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
                              <Text strong>[{err.reason}]</Text>{' '}
                              <span>{err.message}</span>
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
                      <div>
                        体积：{previewData?.baseline?.volumeCbm ?? '-'} CBM
                      </div>
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
                      style={{
                        fontSize: 13,
                        marginBottom: 8,
                        display: 'block',
                      }}
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
                              分配货物：{pr.packageCount} 件 /{' '}
                              {pr.grossWeightKg} KGS / {pr.volumeCbm} CBM
                            </div>
                            <div>
                              分单号：{pr.houseNo || '无'} | 归属费用：
                              {pr.feeCount} 笔
                            </div>
                            <div style={{ marginTop: 6 }}>
                              <Text type="secondary">自动重算箱计划：</Text>
                              {pr.containerPlans &&
                              pr.containerPlans.length > 0 ? (
                                pr.containerPlans.map((cp) => (
                                  <Tag
                                    key={
                                      cp.containerSpecId || cp.containerSpecName
                                    }
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
    </PageContainer>
  );
}
