import {
  HistoryOutlined,
  ReloadOutlined,
  ScissorOutlined,
  ShareAltOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { history } from '@/router/history';
import { App, Button, type MenuProps, Tooltip } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import SameBatchOrdersSection from '../../components/detail/SameBatchOrdersSection';
import SeaOrderChangeHistoryDrawer, {
  SeaOrderChangeHistorySection,
} from '../../components/drawers/SeaOrderChangeHistoryDrawer';
import SeaOrderReassignmentModal from '../../components/drawers/SeaOrderReassignmentModal';
import SeaSharedContainerDrawer from '../../components/drawers/SeaSharedContainerDrawer';
import SeaTransportExecutionUpdateModal from '../../components/drawers/SeaTransportExecutionUpdateModal';
import type {
  OrderDetailFeatureContribution,
  OrderDetailFeaturesProps,
} from '../types';

/**
 * 海运出口详情扩展：以正常 React 组件持有拆票/改配动作资格、共享航次、
 * 共享箱、同批订单与拆票/改配历史的请求与本地状态，通过 render-prop
 * 向通用详情布局贡献 UI。订单身份变化由页面按 orderFormIdentity 重挂载。
 */
export default function SeaExportDetailFeatures({
  context,
  children,
}: OrderDetailFeaturesProps) {
  const { message } = App.useApp();
  const {
    kind,
    orderId,
    order,
    orderFormIdentity,
    businessWritesDisabled,
    businessWriteBlockedReason,
    canOrder,
    searchShippingLines,
    searchLocations,
    containerSpecOptions,
    refreshOrderAndLock,
    ensureBusinessWriteAllowed,
  } = context;

  // —— 拆票/改配动作资格：请求序号 + 目标身份 + 实例存活三重门禁 ——
  // 订单切换由页面按 orderFormIdentity 重挂载扩展；旧实例卸载后，仍在等待的
  // 同步任务续体不得再对旧订单发起请求或写回状态。
  const aliveRef = useRef(true);
  const changeActionsTargetKey = orderFormIdentity;
  const activeChangeActionsTargetRef = useRef(changeActionsTargetKey);
  activeChangeActionsTargetRef.current = changeActionsTargetKey;
  const changeActionsRequestIdRef = useRef(0);
  const [changeActionsState, setChangeActionsState] = useState<{
    targetKey: string;
    data: API.SeaOrderChangeActionsData;
  } | null>(null);
  const changeActions =
    changeActionsState &&
    changeActionsState.targetKey === changeActionsTargetKey
      ? changeActionsState.data
      : null;

  const [reassignModalOpen, setReassignModalOpen] = useState(false);
  const [voyageUpdateModalOpen, setVoyageUpdateModalOpen] = useState(false);
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [sharedContainerDrawerOpen, setSharedContainerDrawerOpen] =
    useState(false);
  const [sharedContainerTEId, setSharedContainerTEId] = useState<
    string | undefined
  >(undefined);
  const [sharedContainerOrderId, setSharedContainerOrderId] = useState<
    string | undefined
  >(undefined);

  const loadChangeActions = useCallback(async () => {
    const requestOrderId = orderId;
    const requestTargetKey = changeActionsTargetKey;
    if (!requestOrderId || !requestTargetKey) {
      changeActionsRequestIdRef.current += 1;
      setChangeActionsState(null);
      return;
    }
    // 等待其他刷新任务结束的旧闭包不得使当前订单请求失效。
    if (
      requestTargetKey !== activeChangeActionsTargetRef.current ||
      !aliveRef.current
    ) {
      return;
    }
    const requestId = ++changeActionsRequestIdRef.current;
    try {
      const resp = await seaOrderChangeServiceGetSeaOrderChangeActions({
        orderId: requestOrderId,
      });
      if (
        requestId !== changeActionsRequestIdRef.current ||
        requestTargetKey !== activeChangeActionsTargetRef.current ||
        !aliveRef.current
      ) {
        return;
      }

      setChangeActionsState(
        resp?.data ? { targetKey: requestTargetKey, data: resp.data } : null,
      );
    } catch (error: unknown) {
      if (
        requestId !== changeActionsRequestIdRef.current ||
        requestTargetKey !== activeChangeActionsTargetRef.current ||
        !aliveRef.current
      ) {
        return;
      }
      setChangeActionsState(null);
      message.error(
        error instanceof Error ? error.message : '加载拆票与改配动作失败',
      );
    }
  }, [changeActionsTargetKey, message, orderId]);

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  useEffect(() => {
    setChangeActionsState(null);
    void loadChangeActions();

    return () => {
      changeActionsRequestIdRef.current += 1;
    };
  }, [loadChangeActions, order?.version]);

  useEffect(() => {
    if (businessWritesDisabled) {
      setReassignModalOpen(false);
    }
  }, [businessWritesDisabled]);

  // —— 更多菜单：共享航次、共享箱、拆票/改配历史 ——
  const moreMenuItems: MenuProps['items'] = useMemo(
    () => [
      ...(!businessWritesDisabled && canOrder('reassign')
        ? [
            {
              key: 'shared-voyage-update',
              icon: <ReloadOutlined />,
              label: '共享航次调整',
              onClick: () => setVoyageUpdateModalOpen(true),
            },
          ]
        : []),
      {
        key: 'shared-container-workbench',
        icon: <ShareAltOutlined />,
        label: '跨订单共享箱工作台',
        disabled: !canOrder('container.read'),
        onClick: () => {
          const teId = order.seaMasterBill?.transportExecutionId;
          if (!teId) {
            message.warning('当前订单尚未关联实际运输执行，无法开展跨订单拼箱');
            return;
          }
          setSharedContainerTEId(teId);
          setSharedContainerOrderId(orderId);
          setSharedContainerDrawerOpen(true);
        },
      },
      {
        key: 'change-history',
        icon: <HistoryOutlined />,
        label: '拆票与改配历史',
        onClick: () => setHistoryDrawerOpen(true),
      },
    ],
    [businessWritesDisabled, canOrder, message, order, orderId],
  );

  // —— 头部动作：拆票与改配（保持通用 Header 中的位置、样式与禁用语义）——
  const canSplit = canOrder('split');
  const canReassign = canOrder('reassign');
  const splitDisabled = !changeActions?.canSplit;
  const splitBlockedReasons = changeActions?.splitBlockedReasons;
  const reassignDisabled = !changeActions?.canReassign;
  const reassignBlockedReasons = changeActions?.reassignBlockedReasons;

  const headerActionsNode = (
    <>
      {canSplit && (
        <Tooltip
          title={
            businessWritesDisabled
              ? businessWriteBlockedReason
              : splitDisabled &&
                  splitBlockedReasons &&
                  splitBlockedReasons.length > 0
                ? splitBlockedReasons.join('；')
                : undefined
          }
        >
          <span>
            <Button
              style={{ color: '#722ed1', borderColor: '#722ed1' }}
              icon={<ScissorOutlined />}
              disabled={splitDisabled || businessWritesDisabled}
              onClick={() => {
                if (ensureBusinessWriteAllowed()) {
                  history.push(`/orders/${kind}/${orderId}/split`);
                }
              }}
            >
              拆票
            </Button>
          </span>
        </Tooltip>
      )}

      {canReassign && (
        <Tooltip
          title={
            businessWritesDisabled
              ? businessWriteBlockedReason
              : reassignDisabled &&
                  reassignBlockedReasons &&
                  reassignBlockedReasons.length > 0
                ? reassignBlockedReasons.join('；')
                : undefined
          }
        >
          <span>
            <Button
              style={{ color: '#fa8c16', borderColor: '#fa8c16' }}
              icon={<SwapOutlined />}
              disabled={reassignDisabled || businessWritesDisabled}
              onClick={() => {
                setReassignModalOpen(true);
              }}
            >
              改配
            </Button>
          </span>
        </Tooltip>
      )}
    </>
  );

  // —— 后置区块：同批订单与拆票/改配记录 ——
  const appendSections = useMemo(
    () => [
      {
        key: 'same-batch-orders',
        title: '同批订单',
        content: <SameBatchOrdersSection orderId={orderId} orderKind={kind} />,
      },
      {
        key: 'sea-order-change-history',
        title: '拆票与改配记录',
        content: (
          <SeaOrderChangeHistorySection
            orderId={orderId}
            onOpenAll={() => setHistoryDrawerOpen(true)}
          />
        ),
      },
    ],
    [kind, orderId],
  );

  const contribution: OrderDetailFeatureContribution = {
    headerActions: headerActionsNode,
    moreMenuItems,
    appendSections,
    refreshTypeState: loadChangeActions,
    overlays: (
      <>
        <SeaOrderReassignmentModal
          orderId={orderId}
          orderNo={order.orderNo}
          open={reassignModalOpen}
          disabled={changeActions?.canReassign === false}
          disabledReason={changeActions?.reassignBlockedReasons?.join('；')}
          onClose={() => setReassignModalOpen(false)}
          onSuccess={async () => {
            await refreshOrderAndLock();
            await loadChangeActions();
          }}
          searchShippingLines={searchShippingLines}
          searchLocations={searchLocations}
          initialShippingLineId={order.shippingLineId}
          initialShippingLineName={order.seaMasterBill?.shippingLineName}
        />
        <SeaTransportExecutionUpdateModal
          order={order}
          open={voyageUpdateModalOpen}
          onClose={() => setVoyageUpdateModalOpen(false)}
          onSuccess={async () => {
            await refreshOrderAndLock();
            await loadChangeActions();
          }}
          searchLocations={searchLocations}
        />
        <SeaOrderChangeHistoryDrawer
          orderId={orderId}
          open={historyDrawerOpen}
          onClose={() => setHistoryDrawerOpen(false)}
        />
        <SeaSharedContainerDrawer
          key={`shared-container:${orderId}:${sharedContainerTEId ?? ''}`}
          open={
            sharedContainerDrawerOpen &&
            sharedContainerOrderId === orderId &&
            !!sharedContainerTEId
          }
          onClose={() => setSharedContainerDrawerOpen(false)}
          transportExecutionId={sharedContainerTEId}
          orderId={orderId}
          orderNo={order.orderNo}
          canCreate={!businessWritesDisabled && canOrder('container.create')}
          canUpdate={!businessWritesDisabled && canOrder('container.update')}
          canDelete={!businessWritesDisabled && canOrder('container.delete')}
          containerSpecOptions={containerSpecOptions}
        />
      </>
    ),
  };

  return <>{children(contribution)}</>;
}
