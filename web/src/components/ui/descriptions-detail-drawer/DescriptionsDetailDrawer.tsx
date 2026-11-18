import {
  Button,
  Descriptions,
  type DescriptionsProps,
  Drawer,
  type DrawerProps,
  Modal,
  Space,
} from 'antd';
import {
  Children,
  type ComponentProps,
  Fragment,
  isValidElement,
  type ReactElement,
  type ReactNode,
} from 'react';

type DItemProps = ComponentProps<typeof Descriptions.Item>;

function detailContent(children: ReactNode) {
  return children === null || children === undefined || children === ''
    ? '-'
    : children;
}

function resolveDescriptionItems(children: ReactNode): ReactNode {
  return Children.map(children, (child) => {
    if (!isValidElement(child)) return child;
    if (child.type === Fragment) {
      return resolveDescriptionItems(
        (child as ReactElement<{ children?: ReactNode }>).props.children,
      );
    }
    if (child.type === DItem) {
      const { children: content, ...props } = (
        child as ReactElement<DItemProps>
      ).props;
      return (
        <Descriptions.Item {...props}>
          {detailContent(content)}
        </Descriptions.Item>
      );
    }
    return child;
  });
}

export type DescriptionsDetailDrawerProps<T> = {
  open: boolean;
  detail?: T | null;
  loading?: boolean;
  title: ReactNode | ((detail?: T | null) => ReactNode);
  size?: DrawerProps['size'];
  column?: DescriptionsProps['column'];
  extra?: ReactNode | ((detail: T) => ReactNode);
  descriptions: (detail: T) => ReactNode;
  children?: (detail: T) => ReactNode;
  onClose: () => void;
};

/** 只读详情抽屉的统一外壳，业务字段和明细表格由调用方提供。 */
export function DescriptionsDetailDrawer<T>({
  open,
  detail,
  loading = false,
  title,
  size,
  column = 2,
  extra,
  descriptions,
  children,
  onClose,
}: DescriptionsDetailDrawerProps<T>) {
  return (
    <Drawer
      title={typeof title === 'function' ? title(detail) : title}
      open={open}
      size={size}
      loading={loading}
      extra={
        typeof extra === 'function'
          ? detail
            ? extra(detail)
            : undefined
          : extra
      }
      onClose={onClose}
    >
      {detail ? (
        <>
          <Descriptions bordered size="small" column={column}>
            {resolveDescriptionItems(descriptions(detail))}
          </Descriptions>
          {children?.(detail)}
        </>
      ) : null}
    </Drawer>
  );
}

/** 详情字段空值统一显示短横线，保留 0 与 false。 */
export function DItem({ children, ...props }: DItemProps) {
  return (
    <Descriptions.Item {...props}>{detailContent(children)}</Descriptions.Item>
  );
}

export type DescriptionsDetailModalProps<T> = Omit<
  DescriptionsDetailDrawerProps<T>,
  'size' | 'extra'
> & {
  width?: number;
  /** 弹窗底部操作区；默认只有「关闭」。 */
  footer?: ReactNode | ((detail: T) => ReactNode);
  /** 字段表上方的标题行，可放与字段直接相关的操作（如编辑）。 */
  header?: ReactNode | ((detail: T) => ReactNode);
};

/** 只读详情弹窗外壳，字段与明细由调用方提供；内容超高时在弹窗内滚动。 */
export function DescriptionsDetailModal<T>({
  open,
  detail,
  loading = false,
  title,
  width = 1200,
  column = 2,
  footer,
  header,
  descriptions,
  children,
  onClose,
}: DescriptionsDetailModalProps<T>) {
  const extraFooter =
    typeof footer === 'function' ? (detail ? footer(detail) : null) : footer;
  return (
    <Modal
      title={typeof title === 'function' ? title(detail) : title}
      open={open}
      width={width}
      loading={loading}
      destroyOnHidden
      mask={{ closable: false }}
      styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
      footer={
        <Space>
          <Button onClick={onClose}>关闭</Button>
          {extraFooter}
        </Space>
      }
      onCancel={onClose}
    >
      {detail ? (
        <>
          {typeof header === 'function' ? header(detail) : header}
          <Descriptions bordered size="small" column={column}>
            {resolveDescriptionItems(descriptions(detail))}
          </Descriptions>
          {children?.(detail)}
        </>
      ) : null}
    </Modal>
  );
}
