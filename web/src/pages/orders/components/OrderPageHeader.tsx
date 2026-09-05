import { history } from '@umijs/max';
import React, { type ReactNode } from 'react';
import { PageHeaderShell } from '@/components/ui';
import { ORDER_KIND_CONFIGS, type OrderKind } from '../common';

export type OrderPageKind = 'create' | 'detail' | 'fees' | 'split';

export interface OrderPageHeaderProps {
  page: OrderPageKind;
  orderKind: OrderKind;
  orderId?: string;
  orderNo?: string;
  tags?: ReactNode;
  extra?: ReactNode;
  subTitle?: ReactNode;
}

export const OrderPageHeader: React.FC<OrderPageHeaderProps> = ({
  page,
  orderKind,
  orderId,
  orderNo,
  tags,
  extra,
  subTitle,
}) => {
  const config = ORDER_KIND_CONFIGS[orderKind];
  const navTitle = config?.navigationTitle || '海运出口';
  const listPath = `/orders/${orderKind}`;
  const detailPath = orderId ? `/orders/${orderKind}/${orderId}` : listPath;
  const orderIdentifier = orderNo || orderId || '订单详情';

  const breadcrumbs: Array<{ label: string; href?: string }> = [
    { label: '订单管理', href: '/orders' },
    { label: navTitle, href: listPath },
  ];

  let title: ReactNode = '';
  let backText = '返回列表';
  let onBack = () => history.push(listPath);

  switch (page) {
    case 'create':
      title = '新建订单';
      backText = '返回列表';
      onBack = () => history.push(listPath);
      break;

    case 'detail':
      title = orderIdentifier;
      backText = '返回列表';
      onBack = () => history.push(listPath);
      break;

    case 'fees':
      breadcrumbs.push({
        label: orderIdentifier,
        href: detailPath,
      });
      title = '费用录入';
      backText = '返回订单详情';
      onBack = () => history.push(detailPath);
      break;

    case 'split':
      breadcrumbs.push({
        label: orderIdentifier,
        href: detailPath,
      });
      title = '拆票';
      backText = '返回订单详情';
      onBack = () => history.push(detailPath);
      break;
  }

  return (
    <PageHeaderShell
      title={title}
      subTitle={subTitle}
      onBack={onBack}
      backText={backText}
      breadcrumbs={breadcrumbs}
      tags={tags}
      extra={extra}
    />
  );
};

export default OrderPageHeader;
