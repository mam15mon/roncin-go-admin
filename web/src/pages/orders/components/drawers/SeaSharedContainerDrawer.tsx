import {
  CheckCircleOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  RollbackOutlined,
  SaveOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Pagination,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import Decimal from 'decimal.js';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { SeaSharedContainerStatus } from '@/enums.generated';
import {
  seaSharedContainerServiceConfirmSeaSharedContainer,
  seaSharedContainerServiceCreateSeaSharedContainer,
  seaSharedContainerServiceDeleteSeaSharedContainer,
  seaSharedContainerServiceListSeaSharedContainerCandidates,
  seaSharedContainerServiceListSeaSharedContainers,
  seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft,
  seaSharedContainerServiceWithdrawSeaSharedContainer,
} from '@/services/roncin/seaSharedContainerService';

const { Text, Title } = Typography;

// 候选订单服务端分页大小（分页单位是订单，不是货物行）；共享箱列表一次取分页上限
const CANDIDATE_PAGE_SIZE = 20;
const CONTAINER_PAGE_SIZE = 200;
const STATUS_CONFIRMED = SeaSharedContainerStatus.SEA_SHARED_CONTAINER_STATUS_CONFIRMED;

export type SeaSharedContainerDrawerProps = {
  open: boolean;
  onClose: () => void;
  transportExecutionId?: string;
  orderId?: string;
  orderNo?: string;
  canCreate?: boolean;
  canUpdate?: boolean;
  canDelete?: boolean;
  containerSpecOptions?: { label: string; value: string | number }[];
};

// 本地草稿分配项：携带完整身份与期望版本，翻页或跨页编辑后仍可正确提交
type DraftAllocation = {
  orderId: string;
  houseBillId: string;
  cargoItemId: string;
  expectedOrderVersion: string;
  expectedLinkVersion: string;
  expectedHouseBillVersion: string;
  expectedCargoItemVersion: string;
  packageCount: number;
  grossWeightKg: string;
  volumeCbm: string;
};

type CargoAllocationItem = {
  key: string;
  draft: DraftAllocation;
  orderNo: string;
  houseNo: string;
  cargoName: string;
  totalPackageCount: number;
  totalGrossWeightKg: string;
  totalVolumeCbm: string;
};

const isZeroQuantity = (pkg: number, weight: string, volume: string) =>
  pkg <= 0 &&
  new Decimal(weight || 0).lte(0) &&
  new Decimal(volume || 0).lte(0);

export default function SeaSharedContainerDrawer({
  open,
  onClose,
  transportExecutionId,
  orderId,
  orderNo: _currentOrderNo,
  canCreate = false,
  canUpdate = false,
  canDelete = false,
  containerSpecOptions = [],
}: SeaSharedContainerDrawerProps) {
  const { message } = App.useApp();

  // 业务上下文键：订单 + 运输执行。父组件以该键重新挂载，抽屉内部再以请求序号防迟到覆盖
  const contextKey =
    orderId && transportExecutionId ? `${orderId}:${transportExecutionId}` : null;

  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [containers, setContainers] = useState<API.SeaSharedContainer[]>([]);
  const [selectedContainerId, setSelectedContainerId] = useState<string | null>(
    null,
  );
  const [candidates, setCandidates] = useState<
    API.SeaSharedContainerCandidateOrder[]
  >([]);
  const [candidateKeyword, setCandidateKeyword] = useState('');
  const [candidateSearching, setCandidateSearching] = useState('');
  const [candidatePage, setCandidatePage] = useState(1);
  const [candidateTotal, setCandidateTotal] = useState(0);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();

  // 本地草稿：key = `${orderId}:${cargoItemId}`，仅属于当前业务上下文
  const [drafts, setDrafts] = useState<Record<string, DraftAllocation>>({});

  // 迟到响应防护：单调递增请求序号 + 当前上下文键，仅最新上下文的最新请求可写入状态
  const containersSeqRef = useRef(0);
  const candidatesSeqRef = useRef(0);
  const activeContextRef = useRef<string | null>(null);
  const loadedContextRef = useRef<string | null>(null);
  const lastCandidateQueryRef = useRef<string | null>(null);

  const resetContextState = useCallback(() => {
    setContainers([]);
    setCandidates([]);
    setCandidateTotal(0);
    setSelectedContainerId(null);
    setDrafts({});
  }, []);

  const loadContainers = useCallback(
    async (context: string) => {
      if (!orderId || !transportExecutionId) return;
      const seq = ++containersSeqRef.current;
      setLoading(true);
      try {
        const resp = await seaSharedContainerServiceListSeaSharedContainers({
          orderId,
          transportExecutionId,
          pageSize: CONTAINER_PAGE_SIZE,
        });
        if (
          seq !== containersSeqRef.current ||
          activeContextRef.current !== context
        ) {
          return;
        }
        const containerList = resp?.data || [];
        setContainers(containerList);
        setSelectedContainerId((prev) => {
          if (prev && containerList.some((c) => c.id === prev)) {
            return prev;
          }
          return containerList[0]?.id || null;
        });
      } catch (err: unknown) {
        if (seq === containersSeqRef.current && activeContextRef.current === context) {
          message.error(err instanceof Error ? err.message : '加载共享箱数据失败');
        }
      } finally {
        if (seq === containersSeqRef.current) {
          setLoading(false);
        }
      }
    },
    [orderId, transportExecutionId, message],
  );

  const loadCandidates = useCallback(
    async (context: string, keyword: string, page: number) => {
      if (!orderId || !transportExecutionId) return;
      const seq = ++candidatesSeqRef.current;
      try {
        const resp =
          await seaSharedContainerServiceListSeaSharedContainerCandidates({
            orderId,
            transportExecutionId,
            page,
            pageSize: CANDIDATE_PAGE_SIZE,
            keyword: keyword || undefined,
          });
        if (
          seq !== candidatesSeqRef.current ||
          activeContextRef.current !== context
        ) {
          return;
        }
        setCandidates(resp?.data || []);
        setCandidateTotal(Number(resp?.total || 0));
      } catch (err: unknown) {
        if (seq === candidatesSeqRef.current && activeContextRef.current === context) {
          message.error(
            err instanceof Error ? err.message : '加载候选订单失败',
          );
        }
      }
    },
    [orderId, transportExecutionId, message],
  );

  // 打开或上下文/查询条件变化：新上下文立即清空旧状态并重置查询；
  // 同一上下文内仅关键字或页码变化时按需重新查询候选订单
  useEffect(() => {
    if (!open || !contextKey) {
      activeContextRef.current = null;
      loadedContextRef.current = null;
      lastCandidateQueryRef.current = null;
      containersSeqRef.current += 1;
      candidatesSeqRef.current += 1;
      resetContextState();
      return;
    }
    activeContextRef.current = contextKey;
    if (loadedContextRef.current !== contextKey) {
      loadedContextRef.current = contextKey;
      lastCandidateQueryRef.current = `${contextKey}||:1`;
      setCandidateKeyword('');
      setCandidateSearching('');
      setCandidatePage(1);
      containersSeqRef.current += 1;
      candidatesSeqRef.current += 1;
      resetContextState();
      void loadContainers(contextKey);
      void loadCandidates(contextKey, '', 1);
      return;
    }
    const queryKey = `${contextKey}||${candidateKeyword}:${candidatePage}`;
    if (lastCandidateQueryRef.current === queryKey) return;
    lastCandidateQueryRef.current = queryKey;
    void loadCandidates(contextKey, candidateKeyword, candidatePage);
  }, [open, contextKey, candidateKeyword, candidatePage, loadContainers, loadCandidates, resetContextState]);

  // 当前选中的共享箱
  const selectedContainer = useMemo(() => {
    return containers.find((c) => c.id === selectedContainerId) || null;
  }, [containers, selectedContainerId]);

  // 选定共享箱变化时，以其既有分配初始化本地草稿
  useEffect(() => {
    if (!selectedContainer) {
      setDrafts({});
      return;
    }
    const map: Record<string, DraftAllocation> = {};
    for (const alloc of selectedContainer.allocations ?? []) {
      if (!alloc.orderId || !alloc.cargoItemId || !alloc.houseBillId) continue;
      map[`${alloc.orderId}:${alloc.cargoItemId}`] = {
        orderId: alloc.orderId,
        houseBillId: alloc.houseBillId,
        cargoItemId: alloc.cargoItemId,
        expectedOrderVersion: String(alloc.orderVersion || 1),
        expectedLinkVersion: String(alloc.linkVersion || 1),
        expectedHouseBillVersion: String(alloc.houseBillVersion || 1),
        expectedCargoItemVersion: String(alloc.cargoItemVersion || 1),
        packageCount: alloc.packageCount || 0,
        grossWeightKg: alloc.grossWeightKg || '0.000',
        volumeCbm: alloc.volumeCbm || '0.000000',
      };
    }
    setDrafts(map);
  }, [selectedContainer]);

  // 展开当前候选页订单与货物行；草稿值从 drafts 读取（含不在本页展示的历史编辑）
  const flatCargoList = useMemo<CargoAllocationItem[]>(() => {
    const list: CargoAllocationItem[] = [];
    for (const order of candidates) {
      if (!order.cargoItems) continue;
      for (const cargo of order.cargoItems) {
        if (!order.orderId || !cargo.id || !order.houseBillId) continue;
        const key = `${order.orderId}:${cargo.id}`;
        const draft =
          drafts[key] ||
          ({
            orderId: order.orderId,
            houseBillId: order.houseBillId,
            cargoItemId: cargo.id,
            expectedOrderVersion: String(order.orderVersion || 1),
            expectedLinkVersion: String(order.linkVersion || 1),
            expectedHouseBillVersion: String(order.houseBillVersion || 1),
            expectedCargoItemVersion: String(cargo.version || 1),
            packageCount: 0,
            grossWeightKg: '0.000',
            volumeCbm: '0.000000',
          } satisfies DraftAllocation);
        list.push({
          key,
          draft,
          orderNo: order.orderNo || '',
          houseNo: order.houseNo || '',
          cargoName: cargo.cargoName || '未命名货物',
          totalPackageCount: cargo.packageCount || 0,
          totalGrossWeightKg: cargo.grossWeightKg || '0.000',
          totalVolumeCbm: cargo.volumeCbm || '0.000000',
        });
      }
    }
    return list;
  }, [candidates, drafts]);

  // 本地实时汇总已分配件重尺与剩余
  const summary = useMemo(() => {
    if (!selectedContainer) {
      return {
        totalPackages: 0,
        totalWeight: '0.000',
        totalVolume: '0.000000',
        allocPackages: 0,
        allocWeight: '0.000',
        allocVolume: '0.000000',
        diffPackages: 0,
        diffWeight: '0.000',
        diffVolume: '0.000000',
        isBalanced: false,
      };
    }

    let allocPackages = 0;
    let allocWeight = new Decimal(0);
    let allocVolume = new Decimal(0);

    for (const item of Object.values(drafts)) {
      allocPackages += Number(item.packageCount || 0);
      allocWeight = allocWeight.plus(new Decimal(item.grossWeightKg || 0));
      allocVolume = allocVolume.plus(new Decimal(item.volumeCbm || 0));
    }

    const totalPackages = selectedContainer.packageCount || 0;
    const totalWeight = new Decimal(selectedContainer.grossWeightKg || 0);
    const totalVolume = new Decimal(selectedContainer.volumeCbm || 0);

    const diffPackages = totalPackages - allocPackages;
    const diffWeight = totalWeight.minus(allocWeight);
    const diffVolume = totalVolume.minus(allocVolume);

    const isBalanced =
      diffPackages === 0 && diffWeight.isZero() && diffVolume.isZero();

    return {
      totalPackages,
      totalWeight: totalWeight.toFixed(3),
      totalVolume: totalVolume.toFixed(6),
      allocPackages,
      allocWeight: allocWeight.toFixed(3),
      allocVolume: allocVolume.toFixed(6),
      diffPackages,
      diffWeight: diffWeight.toFixed(3),
      diffVolume: diffVolume.toFixed(6),
      isBalanced,
    };
  }, [selectedContainer, drafts]);

  // 编辑草稿：以当前行身份+版本为基础合并数量
  const handleUpdateDraft = useCallback(
    (
      identity: DraftAllocation,
      field: 'packageCount' | 'grossWeightKg' | 'volumeCbm',
      value: number | string | null,
    ) => {
      const key = `${identity.orderId}:${identity.cargoItemId}`;
      setDrafts((prev) => {
        const existing = prev[key] ?? identity;
        return {
          ...prev,
          [key]: {
            ...existing,
            [field]:
              field === 'packageCount'
                ? Number(value ?? 0)
                : String(value ?? '0.000'),
          },
        };
      });
    },
    [],
  );

  // 快捷填入该货物的全部总件重尺；首次操作（无既有草稿）时以行身份+版本兜底，
  // 避免产出缺少 ID 与期望版本的孤儿数量项
  const handleFillAllCargo = useCallback((record: CargoAllocationItem) => {
    const key = `${record.draft.orderId}:${record.draft.cargoItemId}`;
    setDrafts((prev) => ({
      ...prev,
      [key]: {
        ...(prev[key] ?? record.draft),
        packageCount: record.totalPackageCount,
        grossWeightKg: record.totalGrossWeightKg,
        volumeCbm: record.totalVolumeCbm,
      },
    }));
  }, []);

  // 构建提交 payload：以本地全部草稿为准（跨页新录入同样保留），零值项剔除
  const buildAllocationInputs = ():
    | API.SeaSharedContainerAllocationInput[]
    | undefined => {
    const inputs: API.SeaSharedContainerAllocationInput[] = [];
    for (const draft of Object.values(drafts)) {
      if (isZeroQuantity(draft.packageCount, draft.grossWeightKg, draft.volumeCbm)) {
        continue;
      }
      inputs.push({
        orderId: draft.orderId,
        houseBillId: draft.houseBillId,
        cargoItemId: draft.cargoItemId,
        packageCount: draft.packageCount,
        grossWeightKg: draft.grossWeightKg,
        volumeCbm: draft.volumeCbm,
        expectedOrderVersion: draft.expectedOrderVersion,
        expectedLinkVersion: draft.expectedLinkVersion,
        expectedHouseBillVersion: draft.expectedHouseBillVersion,
        expectedCargoItemVersion: draft.expectedCargoItemVersion,
      });
    }
    return inputs;
  };

  // 提交前校验选中共享箱仍属于当前上下文，通过时返回该共享箱与收窄后的 ID
  const ensureSubmitContext = (): {
    container: API.SeaSharedContainer;
    containerId: string;
  } | null => {
    if (!contextKey || activeContextRef.current !== contextKey) {
      message.error('共享箱上下文已变化，请重新打开工作台后再操作');
      return null;
    }
    const containerId = selectedContainer?.id;
    if (!selectedContainer || !containerId) {
      message.error('未选择共享物理箱');
      return null;
    }
    if (
      selectedContainer.transportExecutionId &&
      selectedContainer.transportExecutionId !== transportExecutionId
    ) {
      message.error('选中的共享箱不属于当前运输执行，请刷新后重试');
      return null;
    }
    return { container: selectedContainer, containerId };
  };

  const reloadCurrentContext = useCallback(() => {
    if (!contextKey) return;
    void loadContainers(contextKey);
    void loadCandidates(contextKey, candidateKeyword, candidatePage);
  }, [contextKey, loadContainers, loadCandidates, candidateKeyword, candidatePage]);

  // 保存草稿
  const handleSaveDraft = async () => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    setSubmitting(true);
    try {
      await seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft(
        { id: containerId },
        {
          id: containerId,
          orderId: anchorOrderId,
          expectedVersion: String(container.version || 1),
          allocations: buildAllocationInputs(),
        },
      );
      message.success('共享箱分配草稿已保存');
      reloadCurrentContext();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '保存草稿失败');
      reloadCurrentContext();
    } finally {
      setSubmitting(false);
    }
  };

  // 确认分配：携带全部草稿输入，在服务端单事务内保存并严格确认，杜绝部分成功
  const handleConfirm = async () => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    if (!summary.isBalanced) {
      message.warning('箱件重尺尚未平衡守恒，无法确认分配');
      return;
    }
    setSubmitting(true);
    try {
      await seaSharedContainerServiceConfirmSeaSharedContainer(
        { id: containerId },
        {
          id: containerId,
          orderId: anchorOrderId,
          expectedVersion: String(container.version || 1),
          allocations: buildAllocationInputs(),
        },
      );
      message.success('共享箱分配已确认生效');
      reloadCurrentContext();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '确认分配失败');
      reloadCurrentContext();
    } finally {
      setSubmitting(false);
    }
  };

  // 撤回确认
  const handleWithdraw = async () => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    setSubmitting(true);
    try {
      await seaSharedContainerServiceWithdrawSeaSharedContainer(
        { id: containerId },
        {
          id: containerId,
          orderId: anchorOrderId,
          expectedVersion: String(container.version || 1),
        },
      );
      message.success('已撤回至草稿状态');
      reloadCurrentContext();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '撤回失败');
      reloadCurrentContext();
    } finally {
      setSubmitting(false);
    }
  };

  // 删除共享箱
  const handleDelete = async () => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    setSubmitting(true);
    try {
      await seaSharedContainerServiceDeleteSeaSharedContainer({
        id: containerId,
        orderId: anchorOrderId,
        expectedVersion: String(container.version || 1),
      });
      message.success('共享物理箱已删除');
      setSelectedContainerId(null);
      if (contextKey) {
        void loadContainers(contextKey);
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除共享箱失败');
      if (contextKey) {
        void loadContainers(contextKey);
      }
    } finally {
      setSubmitting(false);
    }
  };

  // 新建共享箱提交
  const handleCreateSubmit = async (values: any) => {
    if (!transportExecutionId || !orderId) return;
    setSubmitting(true);
    try {
      await seaSharedContainerServiceCreateSeaSharedContainer({
        input: {
          transportExecutionId,
          containerNo: values.containerNo.trim().toUpperCase(),
          containerSpecId: values.containerSpecId,
          sealNo: values.sealNo?.trim() || undefined,
          packageCount: Number(values.packageCount || 0),
          grossWeightKg: String(values.grossWeightKg || '0.000'),
          volumeCbm: String(values.volumeCbm || '0.000000'),
          note: values.note?.trim() || undefined,
        },
        orderId,
      });
      message.success('共享物理箱已创建');
      setCreateModalOpen(false);
      createForm.resetFields();
      if (contextKey) {
        void loadContainers(contextKey);
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '创建共享箱失败');
    } finally {
      setSubmitting(false);
    }
  };

  const isConfirmed =
    selectedContainer?.status === STATUS_CONFIRMED;

  const columns: ColumnsType<CargoAllocationItem> = [
    {
      title: '所属订单 / 分单',
      dataIndex: 'orderNo',
      width: 180,
      render: (_, record) => (
        <div>
          <Text strong>{record.orderNo}</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>
            HBL: {record.houseNo || '直单'}
          </Text>
        </div>
      ),
    },
    {
      title: '货物描述',
      dataIndex: 'cargoName',
      width: 160,
    },
    {
      title: '货物基准总量',
      width: 160,
      render: (_, record) => (
        <div style={{ fontSize: 12 }}>
          <div>件数: {record.totalPackageCount} PCS</div>
          <div>毛重: {record.totalGrossWeightKg} KG</div>
          <div>体积: {record.totalVolumeCbm} CBM</div>
        </div>
      ),
    },
    {
      title: '分配至本共享箱件数 (PCS)',
      width: 140,
      render: (_, record) => (
        <InputNumber
          min={0}
          max={record.totalPackageCount ?? 0}
          value={record.draft.packageCount}
          disabled={isConfirmed || !canUpdate}
          onChange={(val) =>
            handleUpdateDraft(record.draft, 'packageCount', val)
          }
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: '分配毛重 (KG)',
      width: 150,
      render: (_, record) => (
        <Input
          value={record.draft.grossWeightKg}
          disabled={isConfirmed || !canUpdate}
          onChange={(e) =>
            handleUpdateDraft(record.draft, 'grossWeightKg', e.target.value)
          }
          placeholder="0.000"
        />
      ),
    },
    {
      title: '分配体积 (CBM)',
      width: 150,
      render: (_, record) => (
        <Input
          value={record.draft.volumeCbm}
          disabled={isConfirmed || !canUpdate}
          onChange={(e) =>
            handleUpdateDraft(record.draft, 'volumeCbm', e.target.value)
          }
          placeholder="0.000000"
        />
      ),
    },
    {
      title: '快捷操作',
      width: 110,
      render: (_, record) =>
        !isConfirmed && canUpdate ? (
          <Button
            size="small"
            type="link"
            onClick={() => handleFillAllCargo(record)}
          >
            全部填入
          </Button>
        ) : (
          '-'
        ),
    },
  ];

  return (
    <Drawer
      title="跨订单共享箱 / 客户拼货工作台"
      width={1100}
      open={open}
      onClose={onClose}
      destroyOnClose
      extra={
        <Space>
          {canCreate && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalOpen(true)}
            >
              新建共享物理箱
            </Button>
          )}
        </Space>
      }
    >
      {!contextKey ? (
        <Alert
          type="warning"
          showIcon
          message="当前订单未关联实际运输执行"
          description="共享集装箱以同一实际航次（TransportExecution）为协同边界。请先为订单指定运输执行与船名航次后，再开展客户拼货与跨订单分配。"
        />
      ) : (
        <Spin spinning={loading || submitting}>
          <div style={{ marginBottom: 16 }}>
            <Alert
              type="info"
              showIcon
              message="共享箱业务原则"
              description="普通单票订单常态下直接使用独占箱。若多张 HOUSE 订单同航次拼装同一物理箱，请在下方选择或新建共享箱，逐票分配件重尺。确认前系统将严格检查箱容量与各订单货物件重尺守恒。"
            />
          </div>

          {/* 共享箱卡片选择栏 */}
          <div style={{ marginBottom: 16 }}>
            <Title level={5}>本航次共享物理箱 ({containers.length})</Title>
            {containers.length === 0 ? (
              <Empty
                description="当前航次暂无共享物理箱"
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              >
                {canCreate && (
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => setCreateModalOpen(true)}
                  >
                    立即创建共享箱
                  </Button>
                )}
              </Empty>
            ) : (
              <Row gutter={[12, 12]}>
                {containers.map((cntr) => {
                  const isSelected = cntr.id === selectedContainerId;
                  const cntrConfirmed = cntr.status === STATUS_CONFIRMED;
                  return (
                    <Col xs={24} sm={12} md={8} key={cntr.id}>
                      <Card
                        size="small"
                        hoverable
                        onClick={() => setSelectedContainerId(cntr.id || null)}
                        style={{
                          borderColor: isSelected ? '#1677ff' : '#f0f0f0',
                          borderWidth: isSelected ? 2 : 1,
                          backgroundColor: isSelected ? '#f6ffed' : '#ffffff',
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                          }}
                        >
                          <Text
                            strong
                            style={{ fontFamily: 'monospace', fontSize: 15 }}
                          >
                            {cntr.containerNo}
                          </Text>
                          {cntrConfirmed ? (
                            <Tag color="success">已确认</Tag>
                          ) : (
                            <Tag color="warning">草稿</Tag>
                          )}
                        </div>
                        <div
                          style={{
                            marginTop: 4,
                            fontSize: 12,
                            color: 'rgba(0,0,0,0.65)',
                          }}
                        >
                          <div>
                            规格: {cntr.containerSpecName || cntr.containerSpecId}
                            {cntr.sealNo ? ` | 封号: ${cntr.sealNo}` : ''}
                          </div>
                          <div>
                            容量: {cntr.packageCount} PCS / {cntr.grossWeightKg}{' '}
                            KG / {cntr.volumeCbm} CBM
                          </div>
                        </div>
                      </Card>
                    </Col>
                  );
                })}
              </Row>
            )}
          </div>

          {/* 选定共享箱详情与守恒监控 */}
          {selectedContainer && (
            <>
              <Divider style={{ margin: '16px 0' }} />
              <Card
                title={
                  <Space>
                    <span>共享物理箱：</span>
                    <Text
                      style={{
                        fontFamily: 'monospace',
                        fontWeight: 600,
                        fontSize: 16,
                      }}
                    >
                      {selectedContainer.containerNo}
                    </Text>
                    <Tag color="blue">
                      {selectedContainer.containerSpecName ||
                        selectedContainer.containerSpecId}
                    </Tag>
                    {isConfirmed ? (
                      <Tag color="success" icon={<CheckCircleOutlined />}>
                        已确认生效
                      </Tag>
                    ) : (
                      <Tag color="warning" icon={<ExclamationCircleOutlined />}>
                        草稿待确认
                      </Tag>
                    )}
                    {summary.isBalanced ? (
                      <Tag color="cyan">件重尺完全守恒</Tag>
                    ) : (
                      <Tag color="red">
                        差额: {summary.diffPackages} PCS / {summary.diffWeight} KG
                        / {summary.diffVolume} CBM
                      </Tag>
                    )}
                  </Space>
                }
                extra={
                  <Space>
                    {!isConfirmed && canUpdate && (
                      <>
                        <Button
                          icon={<SaveOutlined />}
                          onClick={handleSaveDraft}
                          loading={submitting}
                        >
                          保存草稿
                        </Button>
                        <Button
                          type="primary"
                          icon={<CheckCircleOutlined />}
                          onClick={handleConfirm}
                          loading={submitting}
                          disabled={!summary.isBalanced}
                        >
                          确认分配
                        </Button>
                      </>
                    )}
                    {isConfirmed && canUpdate && (
                      <Button
                        icon={<RollbackOutlined />}
                        onClick={handleWithdraw}
                        loading={submitting}
                      >
                        撤回至草稿
                      </Button>
                    )}
                    {!isConfirmed && canDelete && (
                      <Popconfirm
                        title="确定删除此共享物理箱？"
                        description="删除后相关草稿分配将一并清理。"
                        onConfirm={handleDelete}
                        okText="确定"
                        cancelText="取消"
                      >
                        <Button danger icon={<DeleteOutlined />}>
                          删除
                        </Button>
                      </Popconfirm>
                    )}
                  </Space>
                }
                style={{ marginBottom: 16 }}
              >
                <Descriptions size="small" column={{ xs: 1, sm: 3 }}>
                  <Descriptions.Item label="箱体总容量">
                    {summary.totalPackages} PCS / {summary.totalWeight} KG /{' '}
                    {summary.totalVolume} CBM
                  </Descriptions.Item>
                  <Descriptions.Item label="已分配总量">
                    {summary.allocPackages} PCS / {summary.allocWeight} KG /{' '}
                    {summary.allocVolume} CBM
                  </Descriptions.Item>
                  <Descriptions.Item label="剩余待分配">
                    <Text
                      type={
                        summary.diffPackages === 0 ? 'secondary' : 'danger'
                      }
                      strong
                    >
                      {summary.diffPackages} PCS / {summary.diffWeight} KG /{' '}
                      {summary.diffVolume} CBM
                    </Text>
                  </Descriptions.Item>
                </Descriptions>
              </Card>

              {/* 跨订单货物分配：服务端按“订单”分页，本页订单的全部货物行完整展示，
                  分页由独立的 Pagination 控制订单页，Table 不做本地二次分页 */}
              <Card
                title={`同航次待分配 HOUSE 订单货物 (第 ${candidatePage} 页，共 ${candidateTotal} 票订单)`}
                size="small"
                extra={
                  <Input.Search
                    allowClear
                    size="small"
                    style={{ width: 260 }}
                    placeholder="搜索订单号 / 业务号 / 分单号"
                    prefix={<SearchOutlined />}
                    value={candidateSearching}
                    onChange={(e) => setCandidateSearching(e.target.value)}
                    onSearch={(value) => {
                      setCandidateKeyword(value.trim());
                      setCandidatePage(1);
                    }}
                  />
                }
              >
                {candidates.length === 0 ? (
                  <Empty
                    description="未找到符合条件的 HOUSE 订单可供拼箱"
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                  />
                ) : (
                  <>
                    <Table
                      columns={columns}
                      dataSource={flatCargoList}
                      rowKey={(r) => r.key}
                      pagination={false}
                      size="small"
                      bordered
                    />
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'flex-end',
                        marginTop: 12,
                      }}
                    >
                      <Pagination
                        current={candidatePage}
                        pageSize={CANDIDATE_PAGE_SIZE}
                        total={candidateTotal}
                        showSizeChanger={false}
                        showTotal={(total) => `共 ${total} 票订单`}
                        onChange={(page) => setCandidatePage(page)}
                      />
                    </div>
                  </>
                )}
              </Card>
            </>
          )}
        </Spin>
      )}

      {/* 新建共享箱 Modal */}
      <Modal
        title="新建跨订单共享物理箱"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form
          form={createForm}
          layout="vertical"
          onFinish={handleCreateSubmit}
          initialValues={{
            packageCount: 0,
            grossWeightKg: '0.000',
            volumeCbm: '0.000000',
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="containerNo"
                label="箱号 (Container No.)"
                rules={[
                  { required: true, message: '请输入箱号' },
                  {
                    pattern: /^[A-Z0-9]+$/i,
                    message: '箱号仅允许英文与数字',
                  },
                ]}
              >
                <Input placeholder="例如 COSU1234567" maxLength={30} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="containerSpecId"
                label="集装箱规格"
                rules={[{ required: true, message: '请选择箱型规格' }]}
              >
                <Select
                  placeholder="选择规格"
                  options={containerSpecOptions}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sealNo" label="铅封号 (Seal No.)">
                <Input placeholder="可选填铅封号" maxLength={50} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="packageCount"
                label="总件数 (PCS)"
                rules={[{ required: true, message: '请输入总件数' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="grossWeightKg"
                label="总毛重 (KG)"
                rules={[{ required: true, message: '请输入总毛重' }]}
              >
                <Input placeholder="例如 20000.000" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="volumeCbm"
                label="总体积 (CBM)"
                rules={[{ required: true, message: '请输入总体积' }]}
              >
                <Input placeholder="例如 65.000000" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="note" label="备注说明">
            <Input.TextArea rows={2} placeholder="拼箱注意事项或客户说明" />
          </Form.Item>
        </Form>
      </Modal>
    </Drawer>
  );
}
