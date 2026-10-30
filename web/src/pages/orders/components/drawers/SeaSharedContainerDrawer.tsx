import { PlusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Alert, App, Button, Drawer, Form, Space, Spin } from 'antd';
import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { DRAWER_SIZE } from '@/components/ui';
import { getRequestErrorStatus } from '@/requestErrorConfig';
import {
  seaSharedContainerServiceConfirmSeaSharedContainer,
  seaSharedContainerServiceCreateSeaSharedContainer,
  seaSharedContainerServiceDeleteSeaSharedContainer,
  seaSharedContainerServiceListSeaSharedContainerCandidates,
  seaSharedContainerServiceListSeaSharedContainers,
  seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft,
  seaSharedContainerServiceWithdrawSeaSharedContainer,
} from '@/services/roncin/seaSharedContainerService';
import { isWriteResponse } from '@/utils/api';
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

export default function SeaSharedContainerDrawer(
  props: SeaSharedContainerDrawerProps,
) {
  return (
    <SeaSharedContainerDrawerContent
      key={`${props.orderId}:${props.transportExecutionId}:${props.open}`}
      {...props}
    />
  );
}

function SeaSharedContainerDrawerContent({
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

  // 业务上下文键：订单 + 运输执行；外层按上下文和开关状态重新挂载。
  const contextKey =
    orderId && transportExecutionId
      ? `${orderId}:${transportExecutionId}`
      : null;

  const [submitting, setSubmitting] = useState(false);
  const [selectedContainerId, setSelectedContainerId] = useState<string | null>(
    null,
  );
  const { markDirty, resetDirty } = useUnsavedChangesGuard({
    enabled: open && Boolean(contextKey),
    identity: `${contextKey}:${selectedContainerId}`,
  });
  const [candidateKeyword, setCandidateKeyword] = useState('');
  const [candidateSearching, setCandidateSearching] = useState('');
  const [candidatePage, setCandidatePage] = useState(1);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();

  // 本地草稿：key = `${orderId}:${cargoItemId}`，仅属于当前业务上下文
  const [drafts, setDraftState] = useState<Record<string, DraftAllocation>>({});
  // 事件间同步推进提交快照；连续“填入→保存”不能读取上次渲染的闭包。
  const draftsRef = useRef(drafts);
  const setDrafts = useCallback(
    (next: React.SetStateAction<Record<string, DraftAllocation>>) => {
      const values =
        typeof next === 'function' ? next(draftsRef.current) : next;
      draftsRef.current = values;
      setDraftState(values);
    },
    [],
  );

  // 每次打开使用独立实例身份；关闭/切换订单即卸载，旧查询不会回写新工作台。
  const instanceId = React.useId();
  const activeContextRef = useRef<string | null>(contextKey);
  useEffect(() => {
    activeContextRef.current = contextKey;
    return () => {
      activeContextRef.current = null;
    };
  }, [contextKey]);
  const isCurrentContext = () =>
    open && Boolean(contextKey) && activeContextRef.current === contextKey;
  const enabled = open && Boolean(contextKey);
  const containersQuery = useQuery({
    queryKey: [
      'orders',
      'shared-containers',
      { orderId, transportExecutionId, instanceId },
    ],
    enabled,
    queryFn: async () => {
      const response = await seaSharedContainerServiceListSeaSharedContainers(
        { orderId, transportExecutionId, pageSize: CONTAINER_PAGE_SIZE },
        { skipErrorHandler: true },
      );
      if (
        !response ||
        response.success === false ||
        (!Array.isArray(response.data) &&
          !(response.success === true && response.data === undefined))
      )
        throw new Error('共享箱数据加载失败，请重试');
      // ProtoJSON 会省略空 repeated 字段；明确成功时缺省 data 即空列表。
      return response.data ?? [];
    },
    meta: { silent: true },
  });
  const candidatesQuery = useQuery({
    queryKey: [
      'orders',
      'shared-container-candidates',
      {
        orderId,
        transportExecutionId,
        instanceId,
        candidateKeyword,
        candidatePage,
      },
    ],
    enabled,
    queryFn: async () => {
      const response =
        await seaSharedContainerServiceListSeaSharedContainerCandidates(
          {
            orderId,
            transportExecutionId,
            page: candidatePage,
            pageSize: CANDIDATE_PAGE_SIZE,
            keyword: candidateKeyword || undefined,
          },
          { skipErrorHandler: true },
        );
      if (
        !response ||
        response.success === false ||
        (!Array.isArray(response.data) &&
          !(response.success === true && response.data === undefined))
      )
        throw new Error('候选订单加载失败，请重试');
      return { data: response.data ?? [], total: Number(response.total ?? 0) };
    },
    meta: { silent: true },
  });
  const containers = containersQuery.data ?? [];
  const candidates = candidatesQuery.data?.data ?? [];
  const candidateTotal = candidatesQuery.data?.total ?? 0;
  useEffect(() => {
    if (!containersQuery.isSuccess) return;
    setSelectedContainerId((previous) =>
      previous && containers.some((item) => item.id === previous)
        ? previous
        : (containers[0]?.id ?? null),
    );
  }, [containersQuery.data, containersQuery.isSuccess]);
  const loadContainers = () => containersQuery.refetch();
  const draftEdited = useRef(false);
  const initializedContainer = useRef<string | null>(null);

  // 当前选中的共享箱
  const selectedContainer = useMemo(() => {
    return containers.find((c) => c.id === selectedContainerId) || null;
  }, [containers, selectedContainerId]);

  // 在浏览器允许交互前初始化当前箱草稿，避免首个点击被迟到的 effect 回填覆盖。
  useLayoutEffect(() => {
    if (!selectedContainer) {
      setDrafts({});
      return;
    }
    const identity = `${selectedContainer.id}:${selectedContainer.version}`;
    if (
      initializedContainer.current === identity ||
      (draftEdited.current &&
        initializedContainer.current?.split(':')[0] === selectedContainer.id)
    )
      return;
    initializedContainer.current = identity;
    draftEdited.current = false;
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
      draftEdited.current = true;
      markDirty();
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
    [markDirty],
  );

  // 快捷填入该货物的全部总件重尺；首次操作（无既有草稿）时以行身份+版本兜底，
  // 避免产出缺少 ID 与期望版本的孤儿数量项
  const handleFillAllCargo = useCallback(
    (record: CargoAllocationItem) => {
      draftEdited.current = true;
      markDirty();
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
    },
    [markDirty],
  );

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

  const reloadCurrentContext = () => {
    void containersQuery.refetch();
    void candidatesQuery.refetch();
  };

  // 保存草稿
  const handleSaveDraft = async (): Promise<false | undefined> => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    setSubmitting(true);
    try {
      const writeResponse1 =
        await seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft(
          { id: containerId },
          {
            id: containerId,
            orderId: anchorOrderId,
            expectedVersion: String(container.version || 1),
            allocations: buildAllocationInputs(draftsRef.current),
          },
        );
      if (!isCurrentContext()) return false;
      if (!isWriteResponse(writeResponse1, true)) return false;
      draftEdited.current = false;
      initializedContainer.current = null;
      resetDirty();
      message.success('共享箱分配草稿已保存');
      reloadCurrentContext();
    } catch (err: unknown) {
      if (!isCurrentContext()) return false;
      message.error(err instanceof Error ? err.message : '保存草稿失败');
    } finally {
      if (isCurrentContext()) setSubmitting(false);
    }

    return undefined;
  };

  // 确认分配：携带全部草稿输入，在服务端单事务内保存并严格确认，杜绝部分成功
  const handleConfirm = async (): Promise<false | undefined> => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    if (
      !buildAllocationSummary(selectedContainer, draftsRef.current).isBalanced
    ) {
      message.warning('箱件重尺尚未平衡守恒，无法确认分配');
      return;
    }
    setSubmitting(true);
    try {
      const writeResponse2 =
        await seaSharedContainerServiceConfirmSeaSharedContainer(
          { id: containerId },
          {
            id: containerId,
            orderId: anchorOrderId,
            expectedVersion: String(container.version || 1),
            allocations: buildAllocationInputs(draftsRef.current),
          },
        );
      if (!isCurrentContext()) return false;
      if (!isWriteResponse(writeResponse2, true)) return false;
      draftEdited.current = false;
      initializedContainer.current = null;
      resetDirty();
      message.success('共享箱分配已确认生效');
      reloadCurrentContext();
    } catch (err: unknown) {
      if (!isCurrentContext()) return false;
      message.error(err instanceof Error ? err.message : '确认分配失败');
    } finally {
      if (isCurrentContext()) setSubmitting(false);
    }

    return undefined;
  };

  // 撤回确认
  const handleWithdraw = async (): Promise<false | undefined> => {
    const target = ensureSubmitContext();
    if (!target) return;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    setSubmitting(true);
    try {
      const writeResponse3 =
        await seaSharedContainerServiceWithdrawSeaSharedContainer(
          { id: containerId },
          {
            id: containerId,
            orderId: anchorOrderId,
            expectedVersion: String(container.version || 1),
          },
        );
      if (!isCurrentContext()) return false;
      if (!isWriteResponse(writeResponse3, true)) return false;
      message.success('已撤回至草稿状态');
      reloadCurrentContext();
    } catch (err: unknown) {
      if (!isCurrentContext()) return false;
      message.error(err instanceof Error ? err.message : '撤回失败');
      reloadCurrentContext();
    } finally {
      if (isCurrentContext()) setSubmitting(false);
    }

    return undefined;
  };

  // 删除共享箱
  const handleDelete = async (): Promise<false | undefined> => {
    const target = ensureSubmitContext();
    if (!target) return false;
    const { container, containerId } = target;
    const anchorOrderId = orderId ?? '';
    setSubmitting(true);
    try {
      const writeResponse4 =
        await seaSharedContainerServiceDeleteSeaSharedContainer({
          id: containerId,
          orderId: anchorOrderId,
          expectedVersion: String(container.version || 1),
        });
      if (!isCurrentContext()) return false;
      if (!isWriteResponse(writeResponse4)) return false;
      message.success('共享物理箱已删除');
      setSelectedContainerId(null);
      if (contextKey) {
        void loadContainers();
      }
    } catch (err: unknown) {
      if (!isCurrentContext()) return false;
      message.error(err instanceof Error ? err.message : '删除共享箱失败');
      if (contextKey) {
        void loadContainers();
      }
      return false;
    } finally {
      if (isCurrentContext()) setSubmitting(false);
    }

    return undefined;
  };

  // 新建共享箱提交
  const handleCreateSubmit = async (
    values: SeaSharedContainerFormValues,
  ): Promise<false | undefined> => {
    if (!transportExecutionId || !orderId) return;
    setSubmitting(true);
    try {
      const writeResponse5 =
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
      if (!isCurrentContext()) return false;
      if (!isWriteResponse(writeResponse5, true)) return false;
      message.success('共享物理箱已创建');
      setCreateModalOpen(false);
      createForm.resetFields();
      if (contextKey) {
        void loadContainers();
      }
    } catch (err: unknown) {
      if (!isCurrentContext()) return false;
      message.error(err instanceof Error ? err.message : '创建共享箱失败');
    } finally {
      if (isCurrentContext()) setSubmitting(false);
    }

    return undefined;
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
          {canCreate && containersQuery.isSuccess && (
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
        <Spin spinning={submitting}>
          <div style={{ marginBottom: 16 }}>
            <Alert
              type="info"
              showIcon
              title="共享箱业务原则"
              description="普通单票订单常态下直接使用独占箱。若多张 HOUSE 订单同航次拼装同一物理箱，请在下方选择或新建共享箱，逐票分配件重尺。确认前系统将严格检查箱容量与各订单货物件重尺守恒。"
            />
          </div>

          {containersQuery.isPending && (
            <div role="status">
              <Spin size="small" /> 正在加载共享箱…
            </div>
          )}
          {containersQuery.isError && (
            <SharedQueryError
              resource="共享箱"
              error={containersQuery.error}
              retry={() => void containersQuery.refetch()}
            />
          )}
          {candidatesQuery.isPending && (
            <div role="status">
              <Spin size="small" /> 正在加载候选订单…
            </div>
          )}
          {candidatesQuery.isError && (
            <SharedQueryError
              resource="候选订单"
              error={candidatesQuery.error}
              retry={() => void candidatesQuery.refetch()}
            />
          )}
          {containersQuery.isSuccess && (
            <SeaSharedContainerCardList
              containers={containers}
              selectedContainerId={selectedContainerId}
              canCreate={canCreate}
              onSelect={setSelectedContainerId}
              onCreate={() => setCreateModalOpen(true)}
            />
          )}

          {/* 选定共享箱详情与守恒监控 */}
          {selectedContainer && containersQuery.isSuccess && (
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
              {candidatesQuery.isSuccess && (
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
              )}
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

function SharedQueryError({
  resource,
  error,
  retry,
}: {
  resource: string;
  error: Error;
  retry: () => void;
}) {
  const forbidden = getRequestErrorStatus(error) === 403;
  return (
    <Alert
      showIcon
      type="error"
      style={{ marginBottom: 16 }}
      title={
        forbidden
          ? `无权读取${resource}，请联系管理员核对权限`
          : `${resource}加载失败，请重试`
      }
      action={
        <Button size="small" onClick={retry}>
          重试{resource}
        </Button>
      }
    />
  );
}
