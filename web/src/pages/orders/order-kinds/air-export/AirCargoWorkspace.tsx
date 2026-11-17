import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Descriptions,
  Form,
  Space,
  Spin,
  Typography,
} from 'antd';
import Decimal from 'decimal.js';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { trimDecimal } from '@/utils/format';
import AirWeightPanel from './AirWeightPanel';

/** 货物接口返回完整本票明细；摘要不能只加总某一分页。 */
export function summarizeAirCargo(
  items: API.OrderCargoItem[],
  orderId: string,
) {
  let packages = 0;
  let gross = new Decimal(0);
  let volume = new Decimal(0);
  for (const item of items) {
    if (!item.id || item.orderId !== orderId)
      throw new Error('货物明细与当前订单不一致，请重新读取。');
    // 三个非 optional 标量省略时分别表示零，不能当成未知货量。
    packages += item.packageCount ?? 0;
    gross = gross.plus(item.grossWeightKg ?? 0);
    volume = volume.plus(item.volumeCbm ?? 0);
  }
  return {
    packages,
    gross: gross.toString(),
    volume: volume.toString(),
    count: items.length,
  };
}

export default function AirCargoWorkspace({
  order,
  canReadCargo,
  canUpdate,
  onOpenCargo,
}: {
  order: API.Order;
  canReadCargo: boolean;
  canUpdate: boolean;
  onOpenCargo: () => void;
}) {
  const form = Form.useFormInstance();
  const values = Form.useWatch(
    (all) => [all.totalPackages, all.totalGrossWeightKg, all.totalVolumeCbm],
    { form, preserve: true },
  );
  const unsavedCargo = Boolean(
    values?.some(
      (value, index) =>
        value !==
        [order.totalPackages, order.totalGrossWeightKg, order.totalVolumeCbm][
          index
        ],
    ),
  );
  const cargo = useQuery({
    queryKey: [
      'air-weight',
      'cargo-summary',
      order.id,
      {
        organizationId: order.organizationId,
        businessType: order.businessType,
        version: order.version,
      },
    ],
    enabled: canReadCargo && Boolean(order.id && order.organizationId),
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await orderCargoItemServiceListCargoItems(
        { orderId: order.id || '' },
        { signal, skipErrorHandler: true },
      );
      if (
        !response ||
        response.success === false ||
        (response.data !== undefined && !Array.isArray(response.data))
      )
        throw new Error('实际货量加载失败');
      if (!response.data && response.success !== true)
        throw new Error('实际货量加载失败');
      return summarizeAirCargo(response.data ?? [], order.id || '');
    },
  });
  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <Space wrap>
        <Typography.Text strong>实际货物与计费重量</Typography.Text>
        {canReadCargo && (
          <Button disabled={false} onClick={onOpenCargo}>
            维护实际货物
          </Button>
        )}
      </Space>
      <Descriptions
        size="small"
        column={{ xs: 1, sm: 3 }}
        items={[
          {
            key: 'planned-pieces',
            label: '已保存委托件数',
            children:
              order.totalPackages === undefined
                ? '-'
                : `${order.totalPackages} ${order.totalPackageUnit || ''}`,
          },
          {
            key: 'planned-gross',
            label: '已保存委托毛重（KG）',
            children: trimDecimal(order.totalGrossWeightKg),
          },
          {
            key: 'planned-volume',
            label: '已保存委托体积（CBM）',
            children: trimDecimal(order.totalVolumeCbm),
          },
        ]}
      />
      {canReadCargo &&
        (cargo.isFetching ? (
          <Spin description="正在核对实际货量">
            <div style={{ minHeight: 40 }} />
          </Spin>
        ) : cargo.isError ? (
          <Alert
            type="error"
            title="实际货量加载失败"
            action={
              <Button disabled={false} onClick={() => void cargo.refetch()}>
                重试
              </Button>
            }
          />
        ) : (
          cargo.data && (
            <Descriptions
              size="small"
              column={{ xs: 1, sm: 3 }}
              items={[
                {
                  key: 'actual-pieces',
                  label: '实际总件数',
                  children: cargo.data.count
                    ? cargo.data.packages
                    : '暂无货物明细',
                },
                {
                  key: 'actual-gross',
                  label: '实际总毛重（KG）',
                  children: cargo.data.count
                    ? trimDecimal(cargo.data.gross)
                    : '-',
                },
                {
                  key: 'actual-volume',
                  label: '实际总体积（CBM）',
                  children: cargo.data.count
                    ? trimDecimal(cargo.data.volume)
                    : '-',
                },
              ]}
            />
          )
        ))}
      <AirWeightPanel
        scope="ORDER"
        targetId={order.id || ''}
        businessType={order.businessType}
        canUpdate={canUpdate}
        embedded
        unsavedCargo={unsavedCargo}
      />
    </Space>
  );
}
