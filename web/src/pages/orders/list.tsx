import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { App, Button, Popover, Result } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
import { OrderListTemplate } from '@/components/ui';
import type { OrderListItem } from '@/components/ui/order-list-template/types';
import {
  FeeReviewStatus,
  FeeReviewSummary,
} from '@/features/orders/fee-review';
import { history } from '@/router/history';
import {
  orderTagServiceBatchAssignOrderTags,
  orderTagServiceBatchRemoveOrderTags,
  orderTagServiceListOrderTagOptions,
} from '@/services/roncin/orderTagService';
import { isWriteResponse } from '@/utils/api';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import AttachmentDrawer, {
  type AttachmentDrawerRef,
} from './components/drawers/AttachmentDrawer';
import CargoItemDrawer, {
  type CargoItemDrawerRef,
} from './components/drawers/CargoItemDrawer';
import ConsolidationDrawer, {
  type ConsolidationDrawerRef,
} from './components/drawers/ConsolidationDrawer';
import ContainerDrawer, {
  type ContainerDrawerRef,
} from './components/drawers/ContainerDrawer';
import MilestoneDrawer, {
  type MilestoneDrawerRef,
} from './components/drawers/MilestoneDrawer';
import PersonnelDrawer, {
  type PersonnelDrawerRef,
} from './components/drawers/PersonnelDrawer';
import ShippingDocumentDrawer, {
  type ShippingDocumentDrawerRef,
} from './components/drawers/ShippingDocumentDrawer';
import { OrderFeeReviewDrawer } from './components/fees/fee-review/OrderFeeReview';
import OrderCommissionSummaryCell from './components/OrderCommissionSummaryCell';
import OrderCommissionSummaryModal from './components/OrderCommissionSummaryModal';
import {
  getDocumentsActionLabel,
  openOrderDocuments,
} from './list-documents-action';
import { queryOrderList } from './list-query';
import { useOrderListResources } from './list-resources';
import { getOrderKindDefinition } from './order-kinds/registry';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import { getOrderBusinessWritePolicy } from './use-order-lock-state';

/** 提成摘要下钻弹窗的行级上下文：只保存服务端已裁剪投影与单号。 */
interface CommissionSummaryModalState {
  orderNo: string;
  summary?: API.OrderCommissionSummary;
}

export default function OrderListPage() {
  const location = useLocation();
  const definition = getOrderKindDefinition(location.pathname);

  const actionRef = useRef<ActionType | undefined>(undefined);
  const milestoneDrawerRef = useRef<MilestoneDrawerRef | null>(null);
  const attachmentDrawerRef = useRef<AttachmentDrawerRef | null>(null);
  const personnelDrawerRef = useRef<PersonnelDrawerRef | null>(null);
  const containerDrawerRef = useRef<ContainerDrawerRef | null>(null);
  const consolidationDrawerRef = useRef<ConsolidationDrawerRef | null>(null);
  const cargoItemDrawerRef = useRef<CargoItemDrawerRef | null>(null);
  const shippingDocumentDrawerRef = useRef<ShippingDocumentDrawerRef | null>(
    null,
  );
  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);

  const access = useAccess();
  const { initialState } = useInitialState();
  const workspaceId = initialState?.currentUser?.currentOrganization?.id;
  const [activeOrder, setActiveOrder] = useState<API.Order>();
  const canOperateActiveOrder = access.canOperateOrganization(
    activeOrder?.organizationId,
  );
  const { message } = App.useApp();
  const [reviewOrderId, setReviewOrderId] = useState<string>();
  useEffect(() => {
    setReviewOrderId(undefined);
    setActiveOrder(undefined);
  }, [workspaceId]);
  const reviewColumns: ProColumns<OrderListItem>[] = [
    {
      title: '订单审核',
      key: 'feeReview',
      width: 180,
      render: (_, row) => (
        <Popover
          trigger={['hover', 'focus']}
          content={
            <FeeReviewSummary
              compact
              summary={row.rawRecord?.feeReview?.summary}
            />
          }
        >
          <Button
            type="text"
            size="small"
            disabled={
              !access.canReadFinanceFees &&
              !access.canOrder(row.rawRecord?.businessType ?? '', 'lock') &&
              !access.canOrder(row.rawRecord?.businessType ?? '', 'fee.read')
            }
            onClick={() => setReviewOrderId(row.id)}
          >
            <FeeReviewStatus
              status={row.rawRecord?.lockedAt ? 'APPROVED' : 'UNSUBMITTED'}
              pendingCorrection={row.rawRecord?.feeReview?.hasPendingCorrection}
            />
          </Button>
        </Popover>
      ),
    },
  ];
  const [tagModalOpen, setTagModalOpen] = useState(false);
  const [tagRows, setTagRows] = useState<OrderListItem[]>([]);
  const [tagFilterOptions, setTagFilterOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [commissionModalState, setCommissionModalState] =
    useState<CommissionSummaryModalState | null>(null);

  // 海运出口订单列表的提成摘要列：只消费服务端按当前用户权限裁剪的投影。
  const commissionExtraColumns: ProColumns<OrderListItem>[] = useMemo(
    () => [
      {
        title: '提成',
        dataIndex: 'commissionSummary',
        width: 220,
        render: (_, record) => (
          <OrderCommissionSummaryCell
            summary={record.rawRecord?.commissionSummary}
            onOpen={(summary) =>
              setCommissionModalState({
                orderNo: record.orderNo,
                summary,
              })
            }
          />
        ),
      },
    ],
    [],
  );

  useEffect(() => {
    if (!definition) return;
    void orderTagServiceListOrderTagOptions({
      businessType: definition.businessType as number,
      page: 1,
      pageSize: 200,
    }).then((response) => {
      setTagFilterOptions(
        (response.tags ?? []).map((tag) => ({
          label: tag.name ?? '',
          value: tag.id ?? '',
        })),
      );
    });
  }, [definition?.businessType]);
  const {
    ports,
    airports,
    customerMap,
    containerSpecOptions,
    containerSpecMap,
    searchCustomers,
    searchLocations,
    searchOrderPorts,
    searchOrderCarriers,
    searchOrderPersonnel,
  } = useOrderListResources(definition);

  if (!definition) {
    return (
      <PageContainer>
        <Result
          status="404"
          title="未知的业务类型"
          subTitle="当前路径未匹配到有效的订单业务类型配置"
        />
      </PageContainer>
    );
  }

  return (
    <>
      <OrderListTemplate
        key={definition.kind}
        queryStateIdentity={{
          tabKey: `/orders/${definition.kind}`,
          listId: `orders:${definition.kind}`,
        }}
        actionRef={actionRef}
        title={definition.title}
        subTitle={`统一维护${definition.title}全流程状态、主分单据、箱量配载、费用核算与业务履约轨迹`}
        extraColumns={commissionExtraColumns}
        trailingColumns={reviewColumns}
        options={{
          transportMode: definition.transportMode === 'air' ? 'air' : 'sea',
          loadPorts:
            definition.transportMode === 'air'
              ? searchLocations
              : searchOrderPorts,
          loadPartners: searchCustomers,
          loadCarriers: searchOrderCarriers,
          loadPersonnel: searchOrderPersonnel,
          tags: tagFilterOptions,
        }}
        onManageTags={
          access.canOrder(definition.businessType, 'update')
            ? (rows) => {
                if (
                  rows.some(
                    (row) =>
                      !access.canOperateOrganization(
                        row.rawRecord?.organizationId,
                      ),
                  )
                ) {
                  message.warning('只能在订单所属分公司工作台维护标签');
                  return;
                }
                setTagRows(rows);
                setTagModalOpen(true);
              }
            : undefined
        }
        queryOrders={(params) =>
          queryOrderList(params, definition, {
            ports,
            airports,
            customerMap,
            containerSpecMap,
          })
        }
        onCreateOrder={
          access.canOrder(definition.businessType, 'create')
            ? () => history.push(`/orders/${definition.kind}/new`)
            : undefined
        }
        onViewDetail={(item) =>
          history.push(
            `/orders/${item.orderKind || definition.kind}/${item.id}`,
          )
        }
        onOpenFees={(item) =>
          history.push(
            `/orders/${item.orderKind || definition.kind}/${item.id}/fees`,
          )
        }
        onOpenMilestones={(item) => {
          setActiveOrder(item.rawRecord);
          item.rawRecord && milestoneDrawerRef.current?.open(item.rawRecord);
        }}
        documentsActionLabel={getDocumentsActionLabel(definition.businessType)}
        onOpenDocuments={(item) =>
          openOrderDocuments(
            definition.businessType,
            definition.kind,
            item,
            (path) => history.push(path),
            (record) => {
              setActiveOrder(record);
              shippingDocumentDrawerRef.current?.open(record);
            },
          )
        }
        onOpenContainers={(item) => {
          setActiveOrder(item.rawRecord);
          item.rawRecord && containerDrawerRef.current?.open(item.rawRecord);
        }}
        onOpenCargo={(item) => {
          setActiveOrder(item.rawRecord);
          item.rawRecord && cargoItemDrawerRef.current?.open(item.rawRecord);
        }}
        onOpenAttachments={(item) => {
          setActiveOrder(item.rawRecord);
          item.rawRecord && attachmentDrawerRef.current?.open(item.rawRecord);
        }}
        onOpenPersonnel={(item) => {
          setActiveOrder(item.rawRecord);
          item.rawRecord && personnelDrawerRef.current?.open(item.rawRecord);
        }}
        onOpenConsolidations={(item) => {
          setActiveOrder(item.rawRecord);
          item.rawRecord &&
            consolidationDrawerRef.current?.open(item.rawRecord);
        }}
        onOpenAbnormal={
          access.canOrder(definition.businessType, 'abnormal_case.read')
            ? (item) => {
                setActiveOrder(item.rawRecord);
                item.rawRecord &&
                  abnormalCasePanelRef.current?.open(item.rawRecord);
              }
            : undefined
        }
      />
      <OrderFeeReviewDrawer
        key={reviewOrderId}
        orderId={reviewOrderId}
        onClose={() => setReviewOrderId(undefined)}
        onChanged={() => actionRef.current?.reload()}
      />
      <BusinessTagModal
        open={tagModalOpen}
        loadOptions={(params) =>
          orderTagServiceListOrderTagOptions({
            ...params,
            businessType: definition.businessType as number,
          })
        }
        targetCount={tagRows.length}
        existingTags={tagRows.flatMap(
          (row) => row.rawRecord?.tags ?? row.tags ?? [],
        )}
        canQuickCreate={Boolean(access.canCreateEnterpriseResources)}
        onSubmit={async (mode, tagIds): Promise<false | undefined> => {
          if (
            !tagRows.length ||
            tagRows.some(
              (row) =>
                !access.canOperateOrganization(row.rawRecord?.organizationId),
            )
          )
            return;
          const orderIds = tagRows.map((row) => row.id);
          if (mode === 'assign') {
            const writeResponse1 = await orderTagServiceBatchAssignOrderTags({
              businessType: definition.businessType as number,
              orderIds,
              tagIds,
            });
            if (!isWriteResponse(writeResponse1, 'count')) return false;
            message.success(`已为 ${orderIds.length} 个订单添加标签`);
          } else {
            const writeResponse2 = await orderTagServiceBatchRemoveOrderTags({
              businessType: definition.businessType as number,
              orderIds,
              tagIds,
            });
            if (!isWriteResponse(writeResponse2, 'count')) return false;
            message.success(`已从 ${orderIds.length} 个订单移除标签`);
          }
          actionRef.current?.reload();

          return undefined;
        }}
        onCancel={() => setTagModalOpen(false)}
      />
      <MilestoneDrawer
        ref={milestoneDrawerRef}
        canSet={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'milestone.set')
        }
      />
      <AttachmentDrawer
        ref={attachmentDrawerRef}
        canRegister={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'attachment.register')
        }
      />
      <PersonnelDrawer
        ref={personnelDrawerRef}
        canAssign={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'personnel.assign')
        }
        canRemove={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'personnel.remove')
        }
      />
      <ContainerDrawer
        ref={containerDrawerRef}
        canCreate={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'container.create')
        }
        canUpdate={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'container.update')
        }
        canRemove={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'container.delete')
        }
        containerSpecOptions={containerSpecOptions}
        containerSpecMap={containerSpecMap}
      />
      <ConsolidationDrawer ref={consolidationDrawerRef} />
      <CargoItemDrawer
        ref={cargoItemDrawerRef}
        canCreate={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'cargo_item.create')
        }
        canUpdate={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'cargo_item.update')
        }
        canRemove={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'cargo_item.delete')
        }
      />
      <ShippingDocumentDrawer
        ref={shippingDocumentDrawerRef}
        canManage={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'update')
        }
        transportMode={definition.transportMode}
      />
      <ReleasePodPanel
        ref={releasePodPanelRef}
        canManage={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'release_pod.create')
        }
      />
      <OrderCommissionSummaryModal
        open={commissionModalState !== null}
        orderNo={commissionModalState?.orderNo}
        summary={commissionModalState?.summary}
        canOpenLedger={access.canReadFinanceCommissions === true}
        onClose={() => setCommissionModalState(null)}
        onOpenLedger={() => {
          setCommissionModalState(null);
          history.push('/finance/commissions');
        }}
      />
      <AbnormalCasePanel
        ref={abnormalCasePanelRef}
        canRead={access.canOrder(definition.businessType, 'abnormal_case.read')}
        canCreate={access.canOrder(
          definition.businessType,
          'abnormal_case.create',
        )}
        canResolve={access.canOrder(
          definition.businessType,
          'abnormal_case.resolve',
        )}
        canDelete={access.canOrder(
          definition.businessType,
          'abnormal_case.delete',
        )}
        businessWriteBlockedReason={
          getOrderBusinessWritePolicy({
            state: {
              isLocked: Boolean(activeOrder?.lockedAt),
              businessType: activeOrder?.businessType,
            },
            loading: false,
            error: null,
            canOperate: canOperateActiveOrder,
            order: activeOrder,
          }).reason
        }
      />
    </>
  );
}
