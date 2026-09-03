import {
  CheckCircleOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
  RollbackOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Drawer,
  Empty,
  InputNumber,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import Decimal from 'decimal.js';
import React, { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import { SeaCargoAllocationStatus, SeaDocumentStructure } from '@/enums.generated';
import {
  seaCargoAllocationServiceApplySeaHouseBillAllocationSummary,
  seaCargoAllocationServiceConfirmSeaCargoAllocation,
  seaCargoAllocationServiceGetSeaCargoAllocation,
  seaCargoAllocationServiceSaveSeaCargoAllocationDraft,
  seaCargoAllocationServiceWithdrawSeaCargoAllocation,
} from '@/services/roncin/seaCargoAllocationService';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

export type SeaCargoAllocationDrawerRef = {
  open: (order: API.Order) => void;
};

type SeaCargoAllocationDrawerProps = {
  onSuccess?: () => void;
  canManage?: boolean;
};

export type EditableAllocationRow = {
  key: string;
  id?: string;
  cargoItemId: string;
  houseBillId: string;
  containerId?: string;
  packageCount: number;
  grossWeightKg: string;
  volumeCbm: string;
};

const SeaCargoAllocationDrawer = forwardRef<
  SeaCargoAllocationDrawerRef,
  SeaCargoAllocationDrawerProps
>(function SeaCargoAllocationDrawer({ onSuccess, canManage = false }, ref) {
  const { message } = App.useApp();

  const [visible, setVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [withdrawing, setWithdrawing] = useState(false);
  const [applyingHblId, setApplyingHblId] = useState<string | null>(null);

  const [order, setOrder] = useState<API.Order | null>(null);
  const [aggregate, setAggregate] =
    useState<API.SeaCargoAllocationAggregate | null>(null);
  const [rows, setRows] = useState<EditableAllocationRow[]>([]);
  const [serverError, setServerError] = useState<{
    message: string;
    metadata?: Record<string, string>;
  } | null>(null);
  const [activeTab, setActiveTab] = useState('rows');
  const contentRef = useRef<HTMLDivElement>(null);

  const rowsFromAggregate = (agg: API.SeaCargoAllocationAggregate): EditableAllocationRow[] =>
    (agg.allocations || []).map((a, idx) => ({
      key: a.id || `alloc-${idx}`,
      id: a.id,
      cargoItemId: a.cargoItemId || '',
      houseBillId: a.houseBillId || '',
      containerId: a.containerId || undefined,
      packageCount: a.packageCount || 0,
      grossWeightKg: a.grossWeightKg || '0.000',
      volumeCbm: a.volumeCbm || '0.000000',
    }));

  const loadData = async (orderId: string) => {
    setLoading(true);
    setServerError(null);
    setAggregate(null);
    setRows([]);
    try {
      const res = await seaCargoAllocationServiceGetSeaCargoAllocation({
        orderId,
      });
      if (!res.data) {
        throw new Error('未获取到箱货分配信息');
      }
      const agg = res.data;
      setAggregate(agg);
      setRows(rowsFromAggregate(agg));
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载箱货分配失败');
    } finally {
      setLoading(false);
    }
  };

  useImperativeHandle(ref, () => ({
    open: (targetOrder: API.Order) => {
      setOrder(targetOrder);
      setAggregate(null);
      setRows([]);
      setActiveTab('rows');
      setVisible(true);
      if (targetOrder.id) {
        loadData(targetOrder.id);
      }
    },
  }));

  const isFCL = aggregate?.shipmentType === 'FCL';
  const isConfirmed =
    aggregate?.allocationStatus === SeaCargoAllocationStatus.SEA_CARGO_ALLOCATION_STATUS_CONFIRMED;
  const isDirect = aggregate?.documentStructure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

  useEffect(() => {
    if (!serverError?.metadata) return;
    const targetID = serverError.metadata.cargo_item_id || serverError.metadata.house_bill_id || serverError.metadata.container_id;
    if (!targetID) return;
    const row = rows.find((item) => item.cargoItemId === targetID || item.houseBillId === targetID || item.containerId === targetID);
    if (!row) return;
    setActiveTab('rows');
    requestAnimationFrame(() => {
      const element = contentRef.current?.querySelector<HTMLElement>(`[data-row-key="${row.key}"]`);
      element?.scrollIntoView({ block: 'center', behavior: 'smooth' });
      element?.querySelector<HTMLElement>('input, [role="combobox"]')?.focus();
    });
  }, [serverError, rows]);

  // 实时进度与守恒计算
  const progressCalculations = useMemo(() => {
    const cargoItems = aggregate?.cargoItems || [];
    const containers = aggregate?.containers || [];
    const houseBills = aggregate?.houseBills || [];

    // 货物行汇总
    const cargoSummaries = cargoItems.map((ci) => {
      const basePkg = new Decimal(ci.packageCount || 0);
      const baseWeight = new Decimal(ci.grossWeightKg || 0);
      const baseVol = new Decimal(ci.volumeCbm || 0);

      let allocPkg = new Decimal(0);
      let allocWeight = new Decimal(0);
      let allocVol = new Decimal(0);

      for (const r of rows) {
        if (r.cargoItemId === ci.id) {
          allocPkg = allocPkg.plus(new Decimal(r.packageCount || 0));
          allocWeight = allocWeight.plus(new Decimal(r.grossWeightKg || 0));
          allocVol = allocVol.plus(new Decimal(r.volumeCbm || 0));
        }
      }

      const remPkg = basePkg.minus(allocPkg);
      const remWeight = baseWeight.minus(allocWeight);
      const remVol = baseVol.minus(allocVol);

      let status = 'IN_PROGRESS';
      if (remPkg.isNegative() || remWeight.isNegative() || remVol.isNegative()) {
        status = 'EXCEEDED';
      } else if (remPkg.isZero() && remWeight.isZero() && remVol.isZero()) {
        status = 'COMPLETED';
      }

      return {
        cargoItemId: ci.id,
        cargoName: ci.cargoName,
        basePkg: basePkg.toNumber(),
        allocPkg: allocPkg.toNumber(),
        remPkg: remPkg.toNumber(),
        baseWeight: baseWeight.toFixed(3),
        allocWeight: allocWeight.toFixed(3),
        remWeight: remWeight.toFixed(3),
        baseVol: baseVol.toFixed(6),
        allocVol: allocVol.toFixed(6),
        remVol: remVol.toFixed(6),
        status,
      };
    });

    // 集装箱汇总
    const containerSummaries = containers.map((cntr) => {
      const basePkg = new Decimal(cntr.packageCount || 0);
      const baseWeight = new Decimal(cntr.grossWeightKg || 0);
      const baseVol = new Decimal(cntr.volumeCbm || 0);

      let allocPkg = new Decimal(0);
      let allocWeight = new Decimal(0);
      let allocVol = new Decimal(0);

      for (const r of rows) {
        if (r.containerId === cntr.id) {
          allocPkg = allocPkg.plus(new Decimal(r.packageCount || 0));
          allocWeight = allocWeight.plus(new Decimal(r.grossWeightKg || 0));
          allocVol = allocVol.plus(new Decimal(r.volumeCbm || 0));
        }
      }

      const remPkg = basePkg.minus(allocPkg);
      const remWeight = baseWeight.minus(allocWeight);
      const remVol = baseVol.minus(allocVol);

      let status = 'IN_PROGRESS';
      if (remPkg.isNegative() || remWeight.isNegative() || remVol.isNegative()) {
        status = 'EXCEEDED';
      } else if (remPkg.isZero() && remWeight.isZero() && remVol.isZero()) {
        status = 'COMPLETED';
      }

      return {
        containerId: cntr.id,
        containerNo: cntr.containerNo,
        basePkg: basePkg.toNumber(),
        allocPkg: allocPkg.toNumber(),
        remPkg: remPkg.toNumber(),
        baseWeight: baseWeight.toFixed(3),
        allocWeight: allocWeight.toFixed(3),
        remWeight: remWeight.toFixed(3),
        baseVol: baseVol.toFixed(6),
        allocVol: allocVol.toFixed(6),
        remVol: remVol.toFixed(6),
        status,
      };
    });

    // 分单汇总与当前显示值对比
    const houseBillSummaries = houseBills.map((hb) => {
      let allocPkg = new Decimal(0);
      let allocWeight = new Decimal(0);
      let allocVol = new Decimal(0);
      let rowCount = 0;

      for (const r of rows) {
        if (r.houseBillId === hb.id) {
          rowCount++;
          allocPkg = allocPkg.plus(new Decimal(r.packageCount || 0));
          allocWeight = allocWeight.plus(new Decimal(r.grossWeightKg || 0));
          allocVol = allocVol.plus(new Decimal(r.volumeCbm || 0));
        }
      }

      const dispPkg = new Decimal(hb.content?.packageCount ?? 0);
      const dispWeight = new Decimal(hb.content?.grossWeightKg ?? 0);
      const dispVol = new Decimal(hb.content?.volumeCbm ?? 0);

      const diffPkg = allocPkg.minus(dispPkg);
      const diffWeight = allocWeight.minus(dispWeight);
      const diffVol = allocVol.minus(dispVol);

      const matches =
        diffPkg.isZero() && diffWeight.isZero() && diffVol.isZero();

      return {
        houseBillId: hb.id,
        houseNo: hb.houseNo,
        version: hb.version,
        rowCount,
        allocPkg: allocPkg.toNumber(),
        allocWeight: allocWeight.toFixed(3),
        allocVol: allocVol.toFixed(6),
        dispPkg: dispPkg.toNumber(),
        dispWeight: dispWeight.toFixed(3),
        dispVol: dispVol.toFixed(6),
        diffPkg: diffPkg.toNumber(),
        diffWeight: diffWeight.toFixed(3),
        diffVol: diffVol.toFixed(6),
        matches,
      };
    });

    // 检查是否有任何输入行是非法的或超分
    let hasRowError = false;
    for (const r of rows) {
      if (!r.cargoItemId || !r.houseBillId) {
        hasRowError = true;
        break;
      }
      if (isFCL && !r.containerId) {
        // FCL 保存草稿允许未选箱，但确认时需要
      }
      const p = r.packageCount;
      let w: Decimal;
      let v: Decimal;
      try {
        w = new Decimal(r.grossWeightKg);
        v = new Decimal(r.volumeCbm);
      } catch {
        hasRowError = true;
        break;
      }
      if (!Number.isInteger(p) || p <= 0 || !w.isPositive() || w.decimalPlaces() > 3 || !v.isPositive() || v.decimalPlaces() > 6) {
        hasRowError = true;
        break;
      }
    }

    const hasCargoExceeded = cargoSummaries.some(
      (c) => c.status === 'EXCEEDED',
    );
    const hasContainerExceeded = isFCL && containerSummaries.some(
      (c) => c.status === 'EXCEEDED',
    );
    const hasExceeded = hasCargoExceeded || hasContainerExceeded;

    // 是否完全完成分配（用于确认按钮门禁）
    const allCargoCompleted =
      cargoSummaries.length > 0 &&
      cargoSummaries.every((c) => c.status === 'COMPLETED');
    const allContainersCompleted =
      !isFCL ||
      (containerSummaries.length > 0 &&
        containerSummaries.every((c) => c.status === 'COMPLETED'));
    const allHouseBillsHaveAllocations =
      houseBillSummaries.length > 0 &&
      houseBillSummaries.every((h) => h.rowCount > 0);
    const allRowsHaveContainersIfFCL =
      !isFCL || (rows.length > 0 && rows.every((r) => !!r.containerId));

    const isFullyCompleted =
      !hasExceeded &&
      !hasRowError &&
      allCargoCompleted &&
      allContainersCompleted &&
      allHouseBillsHaveAllocations &&
      allRowsHaveContainersIfFCL;

    // 未完成原因清单
    const incompleteReasons: string[] = [];
    for (const c of cargoSummaries) {
      if (c.status === 'IN_PROGRESS') {
        incompleteReasons.push(
          `货物【${c.cargoName}】未分完 (剩余件数: ${c.remPkg}, 毛重: ${c.remWeight}kg, 体积: ${c.remVol}cbm)`,
        );
      }
    }
    for (const h of houseBillSummaries) {
      if (h.rowCount === 0) {
        incompleteReasons.push(`分单【${h.houseNo}】尚未分配任何货物`);
      }
    }
    if (isFCL) {
      for (const cntr of containerSummaries) {
        if (cntr.status === 'IN_PROGRESS') {
          incompleteReasons.push(
            `集装箱【${cntr.containerNo}】未分完 (剩余件数: ${cntr.remPkg}, 毛重: ${cntr.remWeight}kg, 体积: ${cntr.remVol}cbm)`,
          );
        }
      }
      if (rows.some((r) => !r.containerId)) {
        incompleteReasons.push('存在未指定实际箱的分配行 (整箱确认必须指定实际箱)');
      }
    }

    return {
      cargoSummaries,
      containerSummaries,
      houseBillSummaries,
      hasRowError,
      hasExceeded,
      isFullyCompleted,
      incompleteReasons,
    };
  }, [aggregate, rows, isFCL]);

  const handleRowChange = (
    key: string,
    field: keyof EditableAllocationRow,
    value: EditableAllocationRow[keyof EditableAllocationRow],
  ) => {
    setRows((prev) =>
      prev.map((row) => (row.key === key ? { ...row, [field]: value } : row)),
    );
  };

  const handleAddRow = () => {
    const defaultCargo = aggregate?.cargoItems?.[0]?.id || '';
    const defaultHb = aggregate?.houseBills?.[0]?.id || '';
    const defaultCntr = isFCL ? aggregate?.containers?.[0]?.id : undefined;

    const newRow: EditableAllocationRow = {
      key: `alloc-new-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      cargoItemId: defaultCargo,
      houseBillId: defaultHb,
      containerId: defaultCntr,
      packageCount: 0,
      grossWeightKg: '0.000',
      volumeCbm: '0.000000',
    };
    setRows((prev) => [...prev, newRow]);
  };

  const handleRemoveRow = (key: string) => {
    setRows((prev) => prev.filter((r) => r.key !== key));
  };

  // 保存草稿
  const handleSaveDraft = async () => {
    if (!order?.id || !aggregate) return;
    if (!aggregate.allocationVersion) {
      message.error('箱货分配版本缺失，请刷新后重试');
      return;
    }
    if (progressCalculations.hasExceeded) {
      message.error('当前分配存在超分，禁止保存草稿！请调整分配数量后再试。');
      return;
    }
    setSaving(true);
    setServerError(null);
    try {
      const inputs: API.SeaCargoAllocationInput[] = rows.map((r) => ({
        id: r.id,
        cargoItemId: r.cargoItemId,
        houseBillId: r.houseBillId,
        containerId: r.containerId || undefined,
        packageCount: Number(r.packageCount),
        grossWeightKg: new Decimal(r.grossWeightKg).toString(),
        volumeCbm: new Decimal(r.volumeCbm).toString(),
      }));

      const res = await seaCargoAllocationServiceSaveSeaCargoAllocationDraft(
        { orderId: order.id },
        {
          orderId: order.id,
          expectedAllocationVersion: aggregate.allocationVersion,
          allocations: inputs,
        },
      );
      message.success('保存箱货分配草稿成功');
      if (res.data) {
        setAggregate(res.data);
        setRows(rowsFromAggregate(res.data));
      } else {
        await loadData(order.id);
      }
      onSuccess?.();
    } catch (err: any) {
      const errorMsg = err.message || '保存草稿失败';
      setServerError({
        message: errorMsg,
        metadata: err.info?.metadata || err.metadata,
      });
      message.error(errorMsg);
    } finally {
      setSaving(false);
    }
  };

  // 确认分配
  const handleConfirm = async () => {
    if (!order?.id || !aggregate) return;
    if (!aggregate.allocationVersion) {
      message.error('箱货分配版本缺失，请刷新后重试');
      return;
    }
    if (!progressCalculations.isFullyCompleted) {
      message.error('尚未完整分配守恒，无法确认分配！');
      return;
    }
    setConfirming(true);
    setServerError(null);
    try {
      const res = await seaCargoAllocationServiceConfirmSeaCargoAllocation(
        { orderId: order.id },
        {
          orderId: order.id,
          expectedAllocationVersion: aggregate.allocationVersion,
        },
      );
      message.success('箱货分配已确认！所有货物和集装箱已严格守恒锁定。');
      if (res.data) {
        setAggregate(res.data);
      } else {
        await loadData(order.id);
      }
      onSuccess?.();
    } catch (err: any) {
      const errorMsg = err.message || '确认分配失败';
      setServerError({
        message: errorMsg,
        metadata: err.info?.metadata || err.metadata,
      });
      message.error(errorMsg);
    } finally {
      setConfirming(false);
    }
  };

  // 撤回确认
  const handleWithdraw = async () => {
    if (!order?.id || !aggregate) return;
    if (!aggregate.allocationVersion) {
      message.error('箱货分配版本缺失，请刷新后重试');
      return;
    }
    setWithdrawing(true);
    setServerError(null);
    try {
      const res = await seaCargoAllocationServiceWithdrawSeaCargoAllocation(
        { orderId: order.id },
        {
          orderId: order.id,
          expectedAllocationVersion: aggregate.allocationVersion,
        },
      );
      message.success('已成功撤回确认状态，恢复草稿编辑！');
      if (res.data) {
        setAggregate(res.data);
      } else {
        await loadData(order.id);
      }
      onSuccess?.();
    } catch (err: any) {
      const errorMsg = err.message || '撤回确认失败';
      setServerError({
        message: errorMsg,
        metadata: err.info?.metadata || err.metadata,
      });
      message.error(errorMsg);
    } finally {
      setWithdrawing(false);
    }
  };

  // 显式用分配汇总填入目标 HBL
  const handleApplyHouseBillSummary = async (hbId: string, hbVersion: number) => {
    if (!order?.id || !aggregate) return;
    if (!aggregate.allocationVersion || !Number.isInteger(hbVersion) || hbVersion <= 0) {
      message.error('箱货分配或分单版本缺失，请刷新后重试');
      return;
    }
    setApplyingHblId(hbId);
    try {
      await seaCargoAllocationServiceApplySeaHouseBillAllocationSummary(
        { orderId: order.id, houseBillId: hbId },
        {
          orderId: order.id,
          houseBillId: hbId,
          expectedAllocationVersion: aggregate.allocationVersion,
          expectedHouseBillVersion: String(hbVersion || 1),
        },
      );
      message.success('已将分配汇总件重尺填入该分单！');
      await loadData(order.id);
      onSuccess?.();
    } catch (err: any) {
      message.error(err.message || '填入分单汇总失败');
    } finally {
      setApplyingHblId(null);
    }
  };

  // 状态显示渲染
  const renderStatusBadge = (status: string) => {
    switch (status) {
      case 'COMPLETED':
        return <Tag color="success">已分完</Tag>;
      case 'EXCEEDED':
        return <Tag color="error">已超分</Tag>;
      default:
        return <Tag color="processing">分配中</Tag>;
    }
  };

  const cargoOptions = (aggregate?.cargoItems || []).map((c) => ({
    label: `${c.cargoName || '货物'} (${c.packageCount ?? 0}件 / ${c.grossWeightKg ?? 0}kg / ${c.volumeCbm ?? 0}cbm)`,
    value: c.id || '',
  }));

  const houseBillOptions = (aggregate?.houseBills || []).map((h) => ({
    label: `分单: ${h.houseNo || h.id}`,
    value: h.id || '',
  }));

  const containerOptions = (aggregate?.containers || []).map((cntr) => ({
    label: `集装箱: ${cntr.containerNo || cntr.id}`,
    value: cntr.id || '',
  }));

  return (
    <Drawer
      title={
        <Space size="middle" align="center">
          <span>海运出口箱货定量分配</span>
          {aggregate ? (
            <>
              <Tag color="blue">订单: {order?.orderNo || order?.id}</Tag>
              <Tag color={isConfirmed ? 'success' : 'cyan'}>
                {isConfirmed ? '已确认 (CONFIRMED)' : '草稿 (DRAFT)'}
              </Tag>
              <Tag>v{aggregate.allocationVersion}</Tag>
              {isConfirmed && aggregate.confirmedByName ? (
                <Text type="secondary" style={{ fontSize: 13 }}>
                  确认人: {aggregate.confirmedByName} (
                  {formatDate(aggregate.confirmedAt)})
                </Text>
              ) : null}
            </>
          ) : null}
        </Space>
      }
      width={1100}
      open={visible}
      onClose={() => setVisible(false)}
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => order?.id && loadData(order.id)}
            disabled={loading}
          >
            刷新
          </Button>
          {canManage && !isDirect && !isConfirmed ? (
            <>
              <Button
                icon={<SaveOutlined />}
                loading={saving}
                disabled={
                  loading ||
                  progressCalculations.hasExceeded ||
                  progressCalculations.hasRowError
                }
                onClick={handleSaveDraft}
              >
                保存草稿
              </Button>
              <Tooltip
                title={
                  !progressCalculations.isFullyCompleted ? (
                    <div>
                      <div style={{ fontWeight: 600, marginBottom: 4 }}>
                        尚未达到守恒确认条件：
                      </div>
                      {progressCalculations.incompleteReasons.map((r) => (
                        <div key={r}>• {r}</div>
                      ))}
                    </div>
                  ) : undefined
                }
              >
                <span>
                  <Button
                    type="primary"
                    icon={<CheckCircleOutlined />}
                    loading={confirming}
                    disabled={
                      loading ||
                      !progressCalculations.isFullyCompleted ||
                      progressCalculations.hasExceeded
                    }
                    onClick={handleConfirm}
                  >
                    确认分配 (守恒门禁)
                  </Button>
                </span>
              </Tooltip>
            </>
          ) : canManage && !isDirect && isConfirmed ? (
            <Popconfirm
              title="确定要撤回确认状态吗？"
              description="撤回后将解除货物、集装箱与分单的锁定，恢复可编辑草稿状态。"
              onConfirm={handleWithdraw}
              okText="确定撤回"
              cancelText="取消"
            >
              <Button danger icon={<RollbackOutlined />} loading={withdrawing}>
                撤回确认
              </Button>
            </Popconfirm>
          ) : null}
        </Space>
      }
    >
      <Spin spinning={loading}>
        <div ref={contentRef}>
        {serverError ? (
          <Alert
            type="error"
            showIcon
            closable
            onClose={() => setServerError(null)}
            title="操作失败"
            description={
              <div>
                <div>{serverError.message}</div>
                {serverError.metadata ? (
                  <div
                    style={{
                      marginTop: 6,
                      fontSize: 12,
                      fontFamily: 'monospace',
                    }}
                  >
                    定位信息: {JSON.stringify(serverError.metadata)}
                  </div>
                ) : null}
              </div>
            }
            style={{ marginBottom: 16 }}
          />
        ) : null}

        {isDirect ? (
          <Alert type="info" showIcon title="直单（DIRECT）不使用箱货分配" description="如需添加 HBL 并分配箱货，请先取消直单标记，再切换为分单结构。" style={{ marginBottom: 16 }} />
        ) : isConfirmed ? (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            title="当前箱货分配已确认锁定"
            description="所有货物明细、集装箱与分单的定量分配已严格守恒。如需增删改货物、箱号、分单或重新分配，请点击右上角『撤回确认』恢复草稿。"
          />
        ) : progressCalculations.hasExceeded ? (
          <Alert
            type="error"
            showIcon
            style={{ marginBottom: 16 }}
            title="分配数量超出允许范围 (已超分)"
            description="存在分配量大于基准量的维度，保存草稿与确认按钮已自动禁用。请核对下方红色的超分项并调减分配数量。"
          />
        ) : !progressCalculations.isFullyCompleted ? (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            title="草稿处于部分分配状态"
            description="您可以随时保存当前草稿；但必须在每项货物、每只实际箱完全守恒，且每张分单均有分配后，方可执行『确认分配』。"
          />
        ) : (
          <Alert
            type="success"
            showIcon
            style={{ marginBottom: 16 }}
            title="定量守恒校验全部通过！"
            description="所有货物、实际箱与分单的数量已完美配平，您可以点击右上角『确认分配』锁定本单分配事实。"
          />
        )}

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          style={{ display: isDirect ? 'none' : undefined }}
          items={[
            {
              key: 'rows',
              label: `分配明细维护 (${rows.length}行)`,
              children: (
                <Card
                  size="small"
                  variant="borderless"
                  style={{ background: '#fafafa' }}
                >
                  <Space wrap style={{ marginBottom: 12 }} data-testid="live-allocation-progress">
                    {progressCalculations.cargoSummaries.map((summary) => (
                      <Tag
                        key={summary.cargoItemId}
                        color={summary.status === 'EXCEEDED' ? 'error' : summary.status === 'COMPLETED' ? 'success' : 'processing'}
                      >
                        {summary.cargoName || '货物'}：剩余 {summary.remPkg} 件 / {summary.remWeight} KG / {summary.remVol} CBM
                      </Tag>
                    ))}
                    {isFCL && progressCalculations.containerSummaries.map((summary) => (
                      <Tag
                        key={summary.containerId}
                        color={summary.status === 'EXCEEDED' ? 'error' : summary.status === 'COMPLETED' ? 'success' : 'processing'}
                      >
                        箱号 {summary.containerNo}：剩余 {summary.remPkg} 件 / {summary.remWeight} KG / {summary.remVol} CBM
                      </Tag>
                    ))}
                  </Space>
                  <div style={{ marginBottom: 12 }}>
                    {canManage && !isConfirmed ? (
                      <Button
                        type="dashed"
                        icon={<PlusOutlined />}
                        onClick={handleAddRow}
                        disabled={
                          (aggregate?.cargoItems?.length || 0) === 0 ||
                          (aggregate?.houseBills?.length || 0) === 0
                        }
                      >
                        添加分配行
                      </Button>
                    ) : null}
                  </div>

                  <Table<EditableAllocationRow>
                    rowKey="key"
                    dataSource={rows}
                    pagination={false}
                    size="small"
                    rowClassName={(record) => {
                      const metadata = serverError?.metadata;
                      return metadata && (record.cargoItemId === metadata.cargo_item_id || record.houseBillId === metadata.house_bill_id || record.containerId === metadata.container_id)
                        ? 'sea-cargo-allocation-error-row'
                        : '';
                    }}
                    locale={{
                      emptyText: (
                        <Empty
                          description={
                            (aggregate?.cargoItems?.length || 0) === 0
                              ? '订单暂无货物明细，无法进行箱货分配'
                              : (aggregate?.houseBills?.length || 0) === 0
                                ? '订单暂无分单 (HBL)，无法进行箱货分配'
                                : '暂无分配行，请点击上方『添加分配行』开始定量分配'
                          }
                        />
                      ),
                    }}
                    columns={[
                      {
                        title: '货物明细',
                        dataIndex: 'cargoItemId',
                        width: 220,
                        render: (val, record) => (
                          <Select
                            value={val}
                            disabled={isConfirmed || !canManage}
                            style={{ width: '100%' }}
                            options={cargoOptions}
                            onChange={(v) =>
                              handleRowChange(record.key, 'cargoItemId', v)
                            }
                          />
                        ),
                      },
                      {
                        title: '归属分单 (HBL)',
                        dataIndex: 'houseBillId',
                        width: 200,
                        render: (val, record) => (
                          <Select
                            value={val}
                            disabled={isConfirmed || !canManage}
                            style={{ width: '100%' }}
                            options={houseBillOptions}
                            onChange={(v) =>
                              handleRowChange(record.key, 'houseBillId', v)
                            }
                          />
                        ),
                      },
                      ...(isFCL
                        ? [
                            {
                              title: '承载实际箱',
                              dataIndex: 'containerId',
                              width: 200,
                              render: (val: string | undefined, record: EditableAllocationRow) => (
                                <Select
                                  value={val}
                                  placeholder="选择箱号 (确认必选)"
                                  allowClear
                                  disabled={isConfirmed || !canManage}
                                  style={{ width: '100%' }}
                                  options={containerOptions}
                                  onChange={(v) =>
                                    handleRowChange(
                                      record.key,
                                      'containerId',
                                      v,
                                    )
                                  }
                                />
                              ),
                            },
                          ]
                        : []),
                      {
                        title: '件数 (PCS)',
                        dataIndex: 'packageCount',
                        width: 120,
                        render: (val, record) => (
                          <InputNumber
                            value={val}
                            min={1}
                            precision={0}
                            disabled={isConfirmed || !canManage}
                            style={{ width: '100%' }}
                            onChange={(v) =>
                              handleRowChange(
                                record.key,
                                'packageCount',
                                v ?? 0,
                              )
                            }
                          />
                        ),
                      },
                      {
                        title: '毛重 (KG)',
                        dataIndex: 'grossWeightKg',
                        width: 140,
                        render: (val, record) => (
                          <InputNumber
                            value={val}
                            stringMode
                            min={0.001}
                            precision={3}
                            disabled={isConfirmed || !canManage}
                            style={{ width: '100%' }}
                            onChange={(v) =>
                              handleRowChange(
                                record.key,
                                'grossWeightKg',
                                v !== null ? v : '0.000',
                              )
                            }
                          />
                        ),
                      },
                      {
                        title: '体积 (CBM)',
                        dataIndex: 'volumeCbm',
                        width: 140,
                        render: (val, record) => (
                          <InputNumber
                            value={val}
                            stringMode
                            min={0.000001}
                            precision={6}
                            disabled={isConfirmed || !canManage}
                            style={{ width: '100%' }}
                            onChange={(v) =>
                              handleRowChange(
                                record.key,
                                'volumeCbm',
                                v !== null ? v : '0.000000',
                              )
                            }
                          />
                        ),
                      },
                      ...(canManage && !isConfirmed
                        ? [
                            {
                              title: '操作',
                              key: 'action',
                              width: 60,
                            render: (_: unknown, record: EditableAllocationRow) => (
                                <Button
                                  type="text"
                                  danger
                                  icon={<DeleteOutlined />}
                                  onClick={() => handleRemoveRow(record.key)}
                                />
                              ),
                            },
                          ]
                        : []),
                    ]}
                  />
                </Card>
              ),
            },
            {
              key: 'balance',
              label: '守恒进度总览',
              children: (
                <Row gutter={[16, 16]}>
                  <Col span={24}>
                    <Card size="small" title="货物分配守恒进度">
                      <Table
                        size="small"
                        pagination={false}
                        rowKey="cargoItemId"
                        dataSource={progressCalculations.cargoSummaries}
                        columns={[
                          { title: '货物名称', dataIndex: 'cargoName' },
                          {
                            title: '件数 (基准 / 已分 / 剩余)',
                            render: (_, r) => (
                              <Space orientation="vertical" size={0}>
                                <span>
                                  基准: {r.basePkg} / 已分: {r.allocPkg}
                                </span>
                                <Text
                                  style={{
                                    color:
                                      r.remPkg < 0
                                        ? '#ff4d4f'
                                        : r.remPkg === 0
                                          ? '#52c41a'
                                          : '#1677ff',
                                    fontWeight: 600,
                                  }}
                                >
                                  剩余: {r.remPkg}
                                </Text>
                              </Space>
                            ),
                          },
                          {
                            title: '毛重 KG (基准 / 已分 / 剩余)',
                            render: (_, r) => (
                              <Space orientation="vertical" size={0}>
                                <span>
                                  基准: {r.baseWeight} / 已分: {r.allocWeight}
                                </span>
                                <Text
                                  style={{
                                    color:
                                      Number(r.remWeight) < 0
                                        ? '#ff4d4f'
                                        : Number(r.remWeight) === 0
                                          ? '#52c41a'
                                          : '#1677ff',
                                    fontWeight: 600,
                                  }}
                                >
                                  剩余: {r.remWeight}
                                </Text>
                              </Space>
                            ),
                          },
                          {
                            title: '体积 CBM (基准 / 已分 / 剩余)',
                            render: (_, r) => (
                              <Space orientation="vertical" size={0}>
                                <span>
                                  基准: {r.baseVol} / 已分: {r.allocVol}
                                </span>
                                <Text
                                  style={{
                                    color:
                                      Number(r.remVol) < 0
                                        ? '#ff4d4f'
                                        : Number(r.remVol) === 0
                                          ? '#52c41a'
                                          : '#1677ff',
                                    fontWeight: 600,
                                  }}
                                >
                                  剩余: {r.remVol}
                                </Text>
                              </Space>
                            ),
                          },
                          {
                            title: '状态',
                            dataIndex: 'status',
                            width: 100,
                            render: (s) => renderStatusBadge(s),
                          },
                        ]}
                      />
                    </Card>
                  </Col>

                  {isFCL ? (
                    <Col span={24}>
                      <Card size="small" title="集装箱分配守恒进度 (整箱 FCL)">
                        <Table
                          size="small"
                          pagination={false}
                          rowKey="containerId"
                          dataSource={progressCalculations.containerSummaries}
                          columns={[
                            { title: '箱号', dataIndex: 'containerNo' },
                            {
                              title: '件数 (基准 / 已分 / 剩余)',
                              render: (_, r) => (
                                <Space orientation="vertical" size={0}>
                                  <span>
                                    基准: {r.basePkg} / 已分: {r.allocPkg}
                                  </span>
                                  <Text
                                    style={{
                                      color:
                                        r.remPkg < 0
                                        ? '#ff4d4f'
                                        : r.remPkg === 0
                                          ? '#52c41a'
                                          : '#1677ff',
                                      fontWeight: 600,
                                    }}
                                  >
                                    剩余: {r.remPkg}
                                  </Text>
                                </Space>
                              ),
                            },
                            {
                              title: '毛重 KG (基准 / 已分 / 剩余)',
                              render: (_, r) => (
                                <Space orientation="vertical" size={0}>
                                  <span>
                                    基准: {r.baseWeight} / 已分: {r.allocWeight}
                                  </span>
                                  <Text
                                    style={{
                                      color:
                                        Number(r.remWeight) < 0
                                        ? '#ff4d4f'
                                        : Number(r.remWeight) === 0
                                          ? '#52c41a'
                                          : '#1677ff',
                                      fontWeight: 600,
                                    }}
                                  >
                                    剩余: {r.remWeight}
                                  </Text>
                                </Space>
                              ),
                            },
                            {
                              title: '体积 CBM (基准 / 已分 / 剩余)',
                              render: (_, r) => (
                                <Space orientation="vertical" size={0}>
                                  <span>
                                    基准: {r.baseVol} / 已分: {r.allocVol}
                                  </span>
                                  <Text
                                    style={{
                                      color:
                                        Number(r.remVol) < 0
                                        ? '#ff4d4f'
                                        : Number(r.remVol) === 0
                                          ? '#52c41a'
                                          : '#1677ff',
                                      fontWeight: 600,
                                    }}
                                  >
                                    剩余: {r.remVol}
                                  </Text>
                                </Space>
                              ),
                            },
                            {
                              title: '状态',
                              dataIndex: 'status',
                              width: 100,
                              render: (s) => renderStatusBadge(s),
                            },
                          ]}
                        />
                      </Card>
                    </Col>
                  ) : null}
                </Row>
              ),
            },
            {
              key: 'houseBills',
              label: '分单汇总与提单对比',
              children: (
                <Card
                  size="small"
                  title="分单分配汇总与提单显示值对比（显式填入）"
                >
                  <Table
                    size="small"
                    pagination={false}
                    rowKey="houseBillId"
                    dataSource={progressCalculations.houseBillSummaries}
                    columns={[
                      {
                        title: '分单号',
                        dataIndex: 'houseNo',
                        render: (val, r) => (
                          <Space>
                            <Text strong>{val}</Text>
                            <Tag>v{r.version}</Tag>
                          </Space>
                        ),
                      },
                      {
                        title: '分配汇总 (件 / 重 / 尺)',
                        render: (_, r) => (
                          <div>
                            {r.allocPkg} 件 / {r.allocWeight} kg / {r.allocVol} cbm
                          </div>
                        ),
                      },
                      {
                        title: '提单当前显示值 (件 / 重 / 尺)',
                        render: (_, r) => (
                          <div>
                            {r.dispPkg} 件 / {r.dispWeight} kg / {r.dispVol} cbm
                          </div>
                        ),
                      },
                      {
                        title: '差额 (已分 - 当前值)',
                        render: (_, r) => (
                          <Space orientation="vertical" size={0}>
                            <Text
                              style={{
                                color: r.matches ? '#52c41a' : '#fa8c16',
                                fontWeight: 600,
                              }}
                            >
                              {r.matches
                                ? '完全一致'
                                : `差 ${r.diffPkg}件 / ${r.diffWeight}kg / ${r.diffVol}cbm`}
                            </Text>
                          </Space>
                        ),
                      },
                      {
                        title: '操作',
                        key: 'action',
                        width: 220,
                        render: (_, r) => (
                          <Button
                            size="small"
                            type="primary"
                            ghost
                            loading={applyingHblId === r.houseBillId}
                            disabled={!canManage || loading || r.rowCount === 0 || !r.houseBillId}
                            onClick={() => {
                              if (r.houseBillId) {
                                handleApplyHouseBillSummary(
                                  r.houseBillId,
                                  Number(r.version),
                                );
                              }
                            }}
                          >
                            用分配汇总填入本张 HBL
                          </Button>
                        ),
                      },
                    ]}
                  />
                </Card>
              ),
            },
          ]}
        />
        <style>{`.sea-cargo-allocation-error-row > td { background: #fff1f0 !important; outline: 1px solid #ff4d4f; }`}</style>
        </div>
      </Spin>
    </Drawer>
  );
});

export default SeaCargoAllocationDrawer;
