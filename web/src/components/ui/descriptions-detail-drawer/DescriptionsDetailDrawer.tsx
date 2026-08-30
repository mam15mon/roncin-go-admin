import {
  Descriptions,
  Drawer,
  type DescriptionsProps,
  type DrawerProps,
} from 'antd';
import {
  Children,
  Fragment,
  isValidElement,
  type ComponentProps,
  type ReactElement,
  type ReactNode,
} from 'react';

type DItemProps = ComponentProps<typeof Descriptions.Item>;

function detailContent(children: ReactNode) {
  return children === null || children === undefined || children === '' ? '-' : children;
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
      const { children: content, ...props } = (child as ReactElement<DItemProps>).props;
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
  return <Descriptions.Item {...props}>{detailContent(children)}</Descriptions.Item>;
}
