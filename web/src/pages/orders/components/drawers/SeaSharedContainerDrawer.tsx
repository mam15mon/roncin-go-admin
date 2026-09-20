import { PlusOutlined } from '@ant-design/icons';
import { Alert, App, Button, Drawer, Form, Space, Spin } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { DRAWER_SIZE } from '@/components/ui';
import {
  seaSharedContainerServiceConfirmSeaSharedContainer,
  seaSharedContainerServiceCreateSeaSharedContainer,
  seaSharedContainerServiceDeleteSeaSharedContainer,
  seaSharedContainerServiceListSeaSharedContainerCandidates,
  seaSharedContainerServiceListSeaSharedContainers,
  seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft,
  seaSharedContainerServiceWithdrawSeaSharedContainer,
} from '@/services/roncin/seaSharedContainerService';
import SeaSharedContainerAllocationTable from './SeaSharedContainerAllocationTable';
import SeaSharedContainerCardList from './SeaSharedContainerCardList';
import SeaSharedContainerCreateModal, {
  type SeaSharedContainerFormValues,
} from './SeaSharedContainerCreateModal';
import SeaSharedContainerDetailCard from './SeaSharedContainerDetailCard';
import {
  buildAllocationInputs,
  buildAllocationSummary,
  buildFlatCargoList,
  CANDIDATE_PAGE_SIZE,
  type CargoAllocationItem,
  CONTAINER_PAGE_SIZE,
  type DraftAllocation,
  STATUS_CONFIRMED,
} from './seaSharedContainerModels';

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
    orderId && transportExecutionId
      ? `${orderId}:${transportExecutionId}`
      : null;

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
        if (
          seq === containersSeqRef.current &&
          activeContextRef.current === context
        ) {
          message.error(
            err instanceof Error ? err.message : '加载共享箱数据失败',
          );
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
        if (
          seq === candidatesSeqRef.current &&
          activeContextRef.current === context
        ) {
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
  }, [
    open,
    contextKey,
    candidateKeyword,
    candidatePage,
    loadContainers,
    loadCandidates,
    resetContextState,
  ]);

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
  const flatCargoList = useMemo<CargoAllocationItem[]>(
    () => buildFlatCargoList(candidates, drafts),
    [candidates, drafts],
  );

  // 本地实时汇总已分配件重尺与剩余
  const summary = useMemo(
    () => buildAllocationSummary(selectedContainer, drafts),
    [selectedContainer, drafts],
  );

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
  }, [
    contextKey,
    loadContainers,
    loadCandidates,
    candidateKeyword,
    candidatePage,
  ]);

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
          allocations: buildAllocationInputs(drafts),
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
          allocations: buildAllocationInputs(drafts),
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
  const handleCreateSubmit = async (values: SeaSharedContainerFormValues) => {
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

  const isConfirmed = selectedContainer?.status === STATUS_CONFIRMED;

  return (
    <Drawer
      title="跨订单共享箱 / 客户拼货工作台"
      size={DRAWER_SIZE.LG}
      open={open}
      onClose={onClose}
      destroyOnHidden
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
          title="当前订单未关联实际运输执行"
          description="共享集装箱以同一实际航次（TransportExecution）为协同边界。请先为订单指定运输执行与船名航次后，再开展客户拼货与跨订单分配。"
        />
      ) : (
        <Spin spinning={loading || submitting}>
          <div style={{ marginBottom: 16 }}>
            <Alert
              type="info"
              showIcon
              title="共享箱业务原则"
              description="普通单票订单常态下直接使用独占箱。若多张 HOUSE 订单同航次拼装同一物理箱，请在下方选择或新建共享箱，逐票分配件重尺。确认前系统将严格检查箱容量与各订单货物件重尺守恒。"
            />
          </div>

          {/* 共享箱卡片选择栏 */}
          <SeaSharedContainerCardList
            containers={containers}
            selectedContainerId={selectedContainerId}
            canCreate={canCreate}
            onSelect={setSelectedContainerId}
            onCreate={() => setCreateModalOpen(true)}
          />

          {/* 选定共享箱详情与守恒监控 */}
          {selectedContainer && (
            <>
              <SeaSharedContainerDetailCard
                container={selectedContainer}
                summary={summary}
                isConfirmed={isConfirmed}
                canUpdate={canUpdate}
                canDelete={canDelete}
                submitting={submitting}
                onSaveDraft={handleSaveDraft}
                onConfirm={handleConfirm}
                onWithdraw={handleWithdraw}
                onDelete={handleDelete}
              />
              <SeaSharedContainerAllocationTable
                candidates={candidates}
                flatCargoList={flatCargoList}
                candidatePage={candidatePage}
                candidateTotal={candidateTotal}
                candidateSearching={candidateSearching}
                isConfirmed={isConfirmed}
                canUpdate={canUpdate}
                onSearchingChange={setCandidateSearching}
                onSearch={(value) => {
                  setCandidateKeyword(value.trim());
                  setCandidatePage(1);
                }}
                onPageChange={setCandidatePage}
                onUpdateDraft={handleUpdateDraft}
                onFillAllCargo={handleFillAllCargo}
              />
            </>
          )}
        </Spin>
      )}

      {/* 新建共享箱 Modal */}
      <SeaSharedContainerCreateModal
        open={createModalOpen}
        form={createForm}
        submitting={submitting}
        containerSpecOptions={containerSpecOptions}
        onCancel={() => setCreateModalOpen(false)}
        onFinish={handleCreateSubmit}
      />
    </Drawer>
  );
}
