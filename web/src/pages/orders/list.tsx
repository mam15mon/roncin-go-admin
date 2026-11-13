import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { Button, Popover, Result } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { OrderListTemplate } from '@/components/ui';
import type { OrderListItem } from '@/components/ui/order-list-template/types';
import {
  FeeReviewStatus,
  FeeReviewSummary,
} from '@/features/orders/fee-review';
import { history } from '@/router/history';
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
import ShippingDocumentDrawer, {
  type ShippingDocumentDrawerRef,
} from './components/drawers/ShippingDocumentDrawer';
import { OrderFeeReviewDrawer } from './components/fees/fee-review/OrderFeeReview';
import {
  getDocumentsActionLabel,
  openOrderDocuments,
} from './list-documents-action';
import { queryOrderList } from './list-query';
import { useOrderListResources } from './list-resources';
import { getOrderKindDefinition } from './order-kinds/registry';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import {
  SEA_EXPORT_HIDDEN_COLUMNS,
  seaExportListColumns,
} from './sea-export-list-columns';
import { getOrderBusinessWritePolicy } from './use-order-lock-state';

export default function OrderListPage() {
  const location = useLocation();
  const definition = getOrderKindDefinition(location.pathname);

  const actionRef = useRef<ActionType | undefined>(undefined);
  const attachmentDrawerRef = useRef<AttachmentDrawerRef | null>(null);
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
  const [reviewOrderId, setReviewOrderId] = useState<string>();
  useEffect(() => {
    setReviewOrderId(undefined);
    setActiveOrder(undefined);
  }, [workspaceId]);
  const reviewColumns: ProColumns<OrderListItem>[] = [
    {
      title: '锁定状态',
      key: 'feeReview',
      width: 128,
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
            style={{
              height: 'auto',
              maxWidth: '100%',
              paddingInline: 0,
              flexWrap: 'wrap',
              justifyContent: 'flex-start',
              gap: 2,
              whiteSpace: 'normal',
            }}
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
        showBillRelations={definition.kind === 'sea-export'}
        subTitle={`统一维护${definition.title}全流程状态、主分单据、箱量配载、费用核算与业务履约轨迹`}
        columnSettingsKey={`orders:${definition.kind}`}
        extraColumns={
          definition.kind === 'sea-export' ? seaExportListColumns : []
        }
        defaultHiddenColumnKeys={
          definition.kind === 'sea-export' ? SEA_EXPORT_HIDDEN_COLUMNS : []
        }
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
        }}
        queryOrders={(params) =>
          queryOrderList(params, definition, {
            organizationId: workspaceId,
            ports,
            airports,
            customerMap,
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
      <AttachmentDrawer
        ref={attachmentDrawerRef}
        canRegister={
          canOperateActiveOrder &&
          access.canOrder(definition.businessType, 'attachment.register')
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
