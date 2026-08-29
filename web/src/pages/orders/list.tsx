import type { ActionType } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { history, useAccess, useLocation } from '@umijs/max';
import { OrderListTemplate } from '@/components/ui';
import { Result } from 'antd';
import React, { useRef } from 'react';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  paymentTermOptions,
  parseOrderKind,
  shipmentTypeOptions,
  tradeDirectionOptions,
  tradeTermOptions,
} from './common';
import { seStatusTabs } from './list-constants';
import { queryOrderList } from './list-query';
import { useOrderListResources } from './list-resources';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
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
import EditOrderModal, {
  type EditOrderModalRef,
} from './components/modals/EditOrderModal';
import TransitionModal, {
  type TransitionModalRef,
} from './components/modals/TransitionModal';

export default function OrderListPage() {
  const location = useLocation();
  const config = parseOrderKind(location.pathname);

  const actionRef = useRef<ActionType | undefined>(undefined);
  const editOrderModalRef = useRef<EditOrderModalRef | null>(null);
  const transitionModalRef = useRef<TransitionModalRef | null>(null);
  const milestoneDrawerRef = useRef<MilestoneDrawerRef | null>(null);
  const attachmentDrawerRef = useRef<AttachmentDrawerRef | null>(null);
  const personnelDrawerRef = useRef<PersonnelDrawerRef | null>(null);
  const containerDrawerRef = useRef<ContainerDrawerRef | null>(null);
  const consolidationDrawerRef = useRef<ConsolidationDrawerRef | null>(null);
  const cargoItemDrawerRef = useRef<CargoItemDrawerRef | null>(null);
  const shippingDocumentDrawerRef =
    useRef<ShippingDocumentDrawerRef | null>(null);
  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  const access = useAccess();
  const {
    masterOptions,
    ports,
    airports,
    customerMap,
    containerSpecOptions,
    containerSpecMap,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchCustomers,
    searchOrderPorts,
    searchOrderCarriers,
    searchOrderPersonnel,
  } = useOrderListResources(config);

  if (!config) {
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
        actionRef={actionRef}
        orderKind={config.kind as any}
        title={config.title}
        subTitle={`统一维护${config.title}全流程状态、主分单据、箱量配载、费用核算与业务履约轨迹`}
        statusTabs={seStatusTabs}
        options={{
          loadPorts: searchOrderPorts,
          loadPartners: searchCustomers,
          loadCarriers: searchOrderCarriers,
          loadPersonnel: searchOrderPersonnel,
        }}
        queryOrders={(params) =>
          queryOrderList(params, config, {
            ports,
            airports,
            customerMap,
            containerSpecMap,
          })
        }
        onCreateOrder={() => history.push(`/orders/${config.kind}/new`)}
        onViewDetail={(item) =>
          history.push(`/orders/${item.orderKind || config.kind}/${item.id}`)
        }
        onEditOrder={(item) =>
          item.rawRecord && editOrderModalRef.current?.open(item.rawRecord)
        }
        onOpenFees={(item) =>
          item.rawRecord && orderFeePanelRef.current?.open(item.rawRecord)
        }
        onOpenMilestones={(item) =>
          item.rawRecord && milestoneDrawerRef.current?.open(item.rawRecord)
        }
        onOpenDocuments={(item) =>
          item.rawRecord &&
          shippingDocumentDrawerRef.current?.open(item.rawRecord)
        }
        onOpenContainers={(item) =>
          item.rawRecord && containerDrawerRef.current?.open(item.rawRecord)
        }
        onOpenCargo={(item) =>
          item.rawRecord && cargoItemDrawerRef.current?.open(item.rawRecord)
        }
        onOpenAttachments={(item) =>
          item.rawRecord && attachmentDrawerRef.current?.open(item.rawRecord)
        }
        onOpenPersonnel={(item) =>
          item.rawRecord && personnelDrawerRef.current?.open(item.rawRecord)
        }
        onOpenConsolidations={(item) =>
          item.rawRecord &&
          consolidationDrawerRef.current?.open(item.rawRecord)
        }
        onOpenAbnormal={(item) =>
          item.rawRecord && abnormalCasePanelRef.current?.open(item.rawRecord)
        }
        onTransitionStatus={(item) =>
          item.rawRecord && transitionModalRef.current?.open(item.rawRecord)
        }
      />

      <EditOrderModal
        ref={editOrderModalRef}
        category={config.category}
        tradeDirectionOptions={tradeDirectionOptions}
        tradeTermOptions={tradeTermOptions}
        paymentTermOptions={paymentTermOptions}
        shipmentTypeOptions={shipmentTypeOptions}
        locationOptions={locationOptions}
        serviceTypeOptions={serviceTypeOptions}
        cargoCategoryOptions={cargoCategoryOptions}
        containerSpecOptions={containerSpecOptions}
        searchCustomers={searchCustomers}
        onSuccess={() => actionRef.current?.reload()}
      />
      <TransitionModal
        ref={transitionModalRef}
        onSuccess={() => actionRef.current?.reload()}
      />
      <MilestoneDrawer
        ref={milestoneDrawerRef}
        canSet={access.canOrder(config.businessType, 'milestone.set')}
      />
      <AttachmentDrawer
        ref={attachmentDrawerRef}
        canRegister={access.canOrder(
          config.businessType,
          'attachment.register',
        )}
      />
      <PersonnelDrawer
        ref={personnelDrawerRef}
        canAssign={access.canOrder(config.businessType, 'personnel.assign')}
        canRemove={access.canOrder(config.businessType, 'personnel.remove')}
      />
      <ContainerDrawer
        ref={containerDrawerRef}
        canCreate={access.canOrder(config.businessType, 'container.create')}
        canUpdate={access.canOrder(config.businessType, 'container.update')}
        canRemove={access.canOrder(config.businessType, 'container.delete')}
        containerSpecOptions={containerSpecOptions}
        containerSpecMap={containerSpecMap}
      />
      <ConsolidationDrawer ref={consolidationDrawerRef} />
      <CargoItemDrawer
        ref={cargoItemDrawerRef}
        canCreate={access.canOrder(config.businessType, 'cargo_item.create')}
        canUpdate={access.canOrder(config.businessType, 'cargo_item.update')}
        canRemove={access.canOrder(config.businessType, 'cargo_item.delete')}
      />
      <ShippingDocumentDrawer
        ref={shippingDocumentDrawerRef}
        canManage={access.canOrder(config.businessType, 'update')}
        category={config.category}
      />
      <ReleasePodPanel
        ref={releasePodPanelRef}
        canManage={access.canOrder(config.businessType, 'release_pod.create')}
      />
      <OrderFeePanel ref={orderFeePanelRef} />
      <AbnormalCasePanel
        ref={abnormalCasePanelRef}
        canManage={access.canOrder(config.businessType, 'abnormal_case.create')}
        masterOptions={masterOptions}
      />
    </>
  );
}
