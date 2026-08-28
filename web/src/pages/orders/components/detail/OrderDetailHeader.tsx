import {
  AlertOutlined,
  DollarOutlined,
  DownOutlined,
  FileDoneOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import { history } from '@umijs/max';
import { Button, Dropdown, type MenuProps } from 'antd';
import React from 'react';
import { DocumentDetailLayout } from '@/components/ui/document-detail-layout';

type OrderDetailHeaderProps = {
  kind: string;
  orderId: string;
  configTitle: string;
  businessType: string;
  order: API.Order;
  saving: boolean;
  canManageFee: boolean;
  canCreatePod: boolean;
  canCreateAbnormal: boolean;
  moreMenuItems: MenuProps['items'];
  hasAction: (action: number) => boolean;
  onSave: () => void;
  onConfirmTermination: (targetStatus: number) => void;
  onConfirmClosure: (targetStatus: number) => void;
  onOpenReleasePod: () => void;
  onOpenAbnormalCase: () => void;
};

export default function OrderDetailHeader({
  kind,
  orderId,
  configTitle,
  order,
  saving,
  canManageFee,
  canCreatePod,
  canCreateAbnormal,
  moreMenuItems,
  hasAction,
  onSave,
  onConfirmTermination,
  onConfirmClosure,
  onOpenReleasePod,
  onOpenAbnormalCase,
}: OrderDetailHeaderProps) {
  return (
    <DocumentDetailLayout
      breadcrumbs={[
        { label: configTitle, path: `/orders/${kind}` },
        { label: `${configTitle}详情` },
      ]}
      code={order.orderNo}
      actions={
        <>
          {/* 实心蓝底主保存按钮 */}
          {hasAction(1) && (
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={saving}
              onClick={onSave}
              style={{ fontWeight: 500 }}
            >
              保存
            </Button>
          )}

          {hasAction(3) && (
            <Button danger onClick={() => onConfirmTermination(2)}>
              发起退关
            </Button>
          )}
          {hasAction(4) && (
            <Button
              danger
              type="primary"
              onClick={() => onConfirmTermination(3)}
            >
              完成退关
            </Button>
          )}
          {hasAction(5) && (
            <Button onClick={() => onConfirmTermination(1)}>取消退关</Button>
          )}
          {hasAction(6) && (
            <Button type="primary" onClick={() => onConfirmClosure(2)}>
              完结订单
            </Button>
          )}
          {hasAction(7) && (
            <Button onClick={() => onConfirmClosure(1)}>反结案</Button>
          )}

          {/* 费用录入（直达独立全屏费用工作台页面） */}
          {canManageFee && (
            <Button
              type="primary"
              icon={<DollarOutlined />}
              onClick={() => history.push(`/orders/${kind}/${orderId}/fees`)}
              style={{ fontWeight: 500 }}
            >
              费用录入
            </Button>
          )}

          {/* 导出单证 / 放货凭证 POD */}
          {canCreatePod && (
            <Button
              style={{ color: '#1677ff', borderColor: '#1677ff' }}
              icon={<FileDoneOutlined />}
              onClick={onOpenReleasePod}
            >
              导出单证 (POD)
            </Button>
          )}

          {/* 异常情况 */}
          {canCreateAbnormal && (
            <Button
              style={{ color: '#ff4d4f', borderColor: '#ff4d4f' }}
              icon={<AlertOutlined />}
              onClick={onOpenAbnormalCase}
            >
              异常情况
            </Button>
          )}

          {/* 更多操作 */}
          <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
            <Button style={{ color: '#64748b', borderColor: '#d9d9d9' }}>
              更多操作 <DownOutlined style={{ fontSize: 10 }} />
            </Button>
          </Dropdown>
        </>
      }
    >
      {null}
    </DocumentDetailLayout>
  );
}
