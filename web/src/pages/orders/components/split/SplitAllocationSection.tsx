import { CheckCircleOutlined } from '@ant-design/icons';
import {
  Button,
  Card,
  Input,
  InputNumber,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import Decimal from 'decimal.js';
import type { Dispatch, SetStateAction } from 'react';
import { SectionCard } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import type { ResultConfig } from '../../splitUtils';

const { Text } = Typography;

/** 货物/共享箱在各结果票间的件重尺分配表：cargoItemId -> resultKey -> 分配值。 */
export type SplitAllocationMap = Record<
  string,
  Record<
    string,
    { packageCount: number; grossWeightKg: string; volumeCbm: string }
  >
>;

interface SplitAllocationSectionProps {
  splitContext: API.SeaOrderSplitContextData | null;
  results: ResultConfig[];
  containerAssignments: Record<string, string>;
  setContainerAssignments: Dispatch<SetStateAction<Record<string, string>>>;
  cargoAllocations: SplitAllocationMap;
  setCargoAllocations: Dispatch<SetStateAction<SplitAllocationMap>>;
  sharedAllocations: SplitAllocationMap;
  setSharedAllocations: Dispatch<SetStateAction<SplitAllocationMap>>;
}

/** 拆票页区块 3：独占箱整箱归属与货物件重尺守恒切分（含跨订单共享箱）。 */
export default function SplitAllocationSection({
  splitContext,
  results,
  containerAssignments,
  setContainerAssignments,
  cargoAllocations,
  setCargoAllocations,
  sharedAllocations,
  setSharedAllocations,
}: SplitAllocationSectionProps) {
  // 独占箱整箱归属列
  const containerColumns: ColumnsType<API.SeaOrderSplitContainerItem> = [
    {
      title: '箱号',
      dataIndex: 'containerNo',
      render: (val) => <Text strong>{val || '-'}</Text>,
    },
    {
      title: '箱型规格',
      dataIndex: 'containerSpecName',
      render: (val) => val || '-',
    },
    {
      title: '货物统计 (件/重/尺)',
      key: 'cargoSummary',
      render: (_, c) => (
        <span>
          {c.packageCount ?? 0} 件 / {c.grossWeightKg ?? 0} KGS /{' '}
          {c.volumeCbm ?? 0} CBM
        </span>
      ),
    },
    {
      title: '整箱归属结果票',
      key: 'assignment',
      width: 260,
      render: (_, c) => (
        <Select
          value={c.id ? containerAssignments[c.id] : undefined}
          style={{ width: '100%' }}
          onChange={(val) => {
            if (c.id) {
              setContainerAssignments({
                ...containerAssignments,
                [c.id]: val,
              });
            }
          }}
          options={results.map((r) => ({
            label: (
              <span>
                <Tag color={r.role === 'ORIGINAL' ? 'default' : 'blue'}>
                  {r.role === 'ORIGINAL' ? '原' : '新'}
                </Tag>
                {r.title}
              </span>
            ),
            value: r.key,
          }))}
        />
      ),
    },
  ];

  const containerSettings =
    useColumnSettings<ColumnsType<API.SeaOrderSplitContainerItem>[number]>({
      tableKey: 'orders:split-containers',
      columns: containerColumns,
    });

  return (
    <SectionCard
      title={
        <Space>
          <Text strong>集装箱与货物明细切分</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            （独占箱整箱归属单一结果票；各货物项件重尺需严格守恒切分）
          </Text>
        </Space>
      }
    >
      {splitContext?.containers && splitContext.containers.length > 0 && (
        <div style={{ marginBottom: 24 }}>
          <Text strong style={{ marginBottom: 8, display: 'block' }}>
            独占集装箱整箱归属：
          </Text>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
            {containerSettings.entry}
          </div>
          <Table<API.SeaOrderSplitContainerItem>
            columns={containerSettings.columns}
            dataSource={splitContext.containers}
            rowKey="id"
            pagination={false}
            size="small"
          />
          {containerSettings.modal}
        </div>
      )}

      <div>
        <Text strong style={{ marginBottom: 8, display: 'block' }}>
          货物明细与件重尺分配：
        </Text>
        <Space orientation="vertical" style={{ width: '100%' }} size="middle">
          {(splitContext?.cargoItems || []).map((ci) => {
            if (!ci.id) return null;
            const currentAllocMap = cargoAllocations[ci.id] || {};
            const totalAllocPkg = results.reduce(
              (acc, r) =>
                acc + (Number(currentAllocMap[r.key]?.packageCount) || 0),
              0,
            );
            const totalAllocWt = results.reduce(
              (acc, r) =>
                acc.add(
                  new Decimal(currentAllocMap[r.key]?.grossWeightKg || '0'),
                ),
              new Decimal(0),
            );
            const totalAllocVol = results.reduce(
              (acc, r) =>
                acc.add(new Decimal(currentAllocMap[r.key]?.volumeCbm || '0')),
              new Decimal(0),
            );

            const baselinePkg = ci.packageCount || 0;
            const baselineWt = new Decimal(ci.grossWeightKg || '0');
            const baselineVol = new Decimal(ci.volumeCbm || '0');

            const isPkgEqual = totalAllocPkg === baselinePkg;
            const isWtEqual = totalAllocWt.equals(baselineWt);
            const isVolEqual = totalAllocVol.equals(baselineVol);
            const isAllConserved = isPkgEqual && isWtEqual && isVolEqual;

            const fillRemaining = (targetKey: string) => {
              const otherPkg = results
                .filter((r) => r.key !== targetKey)
                .reduce(
                  (acc, r) =>
                    acc + (Number(currentAllocMap[r.key]?.packageCount) || 0),
                  0,
                );
              const otherWt = results
                .filter((r) => r.key !== targetKey)
                .reduce(
                  (acc, r) =>
                    acc.add(
                      new Decimal(currentAllocMap[r.key]?.grossWeightKg || '0'),
                    ),
                  new Decimal(0),
                );
              const otherVol = results
                .filter((r) => r.key !== targetKey)
                .reduce(
                  (acc, r) =>
                    acc.add(
                      new Decimal(currentAllocMap[r.key]?.volumeCbm || '0'),
                    ),
                  new Decimal(0),
                );

              const remPkg = Math.max(0, baselinePkg - otherPkg);
              const remWt = Decimal.max(0, baselineWt.sub(otherWt)).toFixed(3);
              const remVol = Decimal.max(0, baselineVol.sub(otherVol)).toFixed(
                6,
              );

              setCargoAllocations((prev) => ({
                ...prev,
                [ci.id as string]: {
                  ...(prev[ci.id as string] || {}),
                  [targetKey]: {
                    packageCount: remPkg,
                    grossWeightKg: remWt,
                    volumeCbm: remVol,
                  },
                },
              }));
            };

            return (
              <Card
                key={ci.id}
                size="small"
                type="inner"
                title={
                  <Space>
                    <Text strong>{ci.cargoName || '货物项'}</Text>
                    <Text type="secondary">
                      （基准总量：{baselinePkg} 件 / {ci.grossWeightKg} KGS /{' '}
                      {ci.volumeCbm} CBM）
                    </Text>
                  </Space>
                }
                extra={
                  isAllConserved ? (
                    <Tag color="success" icon={<CheckCircleOutlined />}>
                      件重尺守恒
                    </Tag>
                  ) : (
                    <Tag color="error">
                      差额: {baselinePkg - totalAllocPkg} 件 /{' '}
                      {baselineWt.sub(totalAllocWt).toFixed(3)} KGS /{' '}
                      {baselineVol.sub(totalAllocVol).toFixed(6)} CBM
                    </Tag>
                  )
                }
              >
                <SplitResultAllocationTable
                  tableKey="orders:split-allocations"
                  results={results}
                  allocMap={currentAllocMap}
                  baselinePkg={baselinePkg}
                  onChangeAllocation={(resultKey, patch) => {
                    setCargoAllocations((prev) => ({
                      ...prev,
                      [ci.id as string]: {
                        ...(prev[ci.id as string] || {}),
                        [resultKey]: {
                          packageCount:
                            patch.packageCount ??
                            currentAllocMap[resultKey]?.packageCount ??
                            0,
                          grossWeightKg:
                            patch.grossWeightKg ??
                            currentAllocMap[resultKey]?.grossWeightKg ??
                            '0',
                          volumeCbm:
                            patch.volumeCbm ??
                            currentAllocMap[resultKey]?.volumeCbm ??
                            '0',
                        },
                      },
                    }));
                  }}
                  onFillRemaining={fillRemaining}
                />
              </Card>
            );
          })}
        </Space>
      </div>

      {splitContext?.sharedContainerAllocations &&
        splitContext.sharedContainerAllocations.length > 0 && (
          <div style={{ marginTop: 24 }}>
            <Text strong style={{ marginBottom: 8, display: 'block' }}>
              跨订单共享箱分配切分：
            </Text>
            <Space orientation="vertical" style={{ width: '100%' }} size="middle">
              {splitContext.sharedContainerAllocations.map((sa) => {
                if (!sa.allocationId) return null;
                const currentAllocMap =
                  sharedAllocations[sa.allocationId] || {};
                const totalAllocPkg = results.reduce(
                  (acc, r) =>
                    acc + (Number(currentAllocMap[r.key]?.packageCount) || 0),
                  0,
                );
                const totalAllocWt = results.reduce(
                  (acc, r) =>
                    acc.add(
                      new Decimal(currentAllocMap[r.key]?.grossWeightKg || '0'),
                    ),
                  new Decimal(0),
                );
                const totalAllocVol = results.reduce(
                  (acc, r) =>
                    acc.add(
                      new Decimal(currentAllocMap[r.key]?.volumeCbm || '0'),
                    ),
                  new Decimal(0),
                );

                const baselinePkg = sa.packageCount || 0;
                const baselineWt = new Decimal(sa.grossWeightKg || '0');
                const baselineVol = new Decimal(sa.volumeCbm || '0');

                const isAllConserved =
                  totalAllocPkg === baselinePkg &&
                  totalAllocWt.equals(baselineWt) &&
                  totalAllocVol.equals(baselineVol);

                const fillSharedRemaining = (targetKey: string) => {
                  const otherPkg = results
                    .filter((r) => r.key !== targetKey)
                    .reduce(
                      (acc, r) =>
                        acc +
                        (Number(currentAllocMap[r.key]?.packageCount) || 0),
                      0,
                    );
                  const otherWt = results
                    .filter((r) => r.key !== targetKey)
                    .reduce(
                      (acc, r) =>
                        acc.add(
                          new Decimal(
                            currentAllocMap[r.key]?.grossWeightKg || '0',
                          ),
                        ),
                      new Decimal(0),
                    );
                  const otherVol = results
                    .filter((r) => r.key !== targetKey)
                    .reduce(
                      (acc, r) =>
                        acc.add(
                          new Decimal(currentAllocMap[r.key]?.volumeCbm || '0'),
                        ),
                      new Decimal(0),
                    );

                  const remPkg = Math.max(0, baselinePkg - otherPkg);
                  const remWt = Decimal.max(0, baselineWt.sub(otherWt)).toFixed(
                    3,
                  );
                  const remVol = Decimal.max(
                    0,
                    baselineVol.sub(otherVol),
                  ).toFixed(6);

                  setSharedAllocations((prev) => ({
                    ...prev,
                    [sa.allocationId as string]: {
                      ...(prev[sa.allocationId as string] || {}),
                      [targetKey]: {
                        packageCount: remPkg,
                        grossWeightKg: remWt,
                        volumeCbm: remVol,
                      },
                    },
                  }));
                };

                return (
                  <Card
                    key={sa.allocationId}
                    size="small"
                    type="inner"
                    title={
                      <Space>
                        <Text strong>
                          共享箱: {sa.containerNo || '待配箱号'} (
                          {sa.containerSpecName || '-'})
                        </Text>
                        <Text type="secondary">
                          （分配基准：{baselinePkg} 件 / {sa.grossWeightKg} KGS
                          / {sa.volumeCbm} CBM）
                        </Text>
                      </Space>
                    }
                    extra={
                      isAllConserved ? (
                        <Tag color="success" icon={<CheckCircleOutlined />}>
                          守恒满足
                        </Tag>
                      ) : (
                        <Tag color="error">
                          差额: {baselinePkg - totalAllocPkg} 件 /{' '}
                          {baselineWt.sub(totalAllocWt).toFixed(3)} KGS /{' '}
                          {baselineVol.sub(totalAllocVol).toFixed(6)} CBM
                        </Tag>
                      )
                    }
                  >
                    <SplitResultAllocationTable
                      tableKey="orders:split-result"
                      results={results}
                      allocMap={currentAllocMap}
                      baselinePkg={baselinePkg}
                      onChangeAllocation={(resultKey, patch) => {
                        setSharedAllocations((prev) => ({
                          ...prev,
                          [sa.allocationId as string]: {
                            ...(prev[sa.allocationId as string] || {}),
                            [resultKey]: {
                              packageCount:
                                patch.packageCount ??
                                currentAllocMap[resultKey]?.packageCount ??
                                0,
                              grossWeightKg:
                                patch.grossWeightKg ??
                                currentAllocMap[resultKey]?.grossWeightKg ??
                                '0',
                              volumeCbm:
                                patch.volumeCbm ??
                                currentAllocMap[resultKey]?.volumeCbm ??
                                '0',
                            },
                          },
                        }));
                      }}
                      onFillRemaining={fillSharedRemaining}
                    />
                  </Card>
                );
              })}
            </Space>
          </div>
        )}
    </SectionCard>
  );
}

interface SplitResultAllocationTableProps {
  /** 列设置表格标识：货物项与共享箱分配表按语义分别隔离。 */
  tableKey: string;
  results: ResultConfig[];
  allocMap: Record<
    string,
    { packageCount: number; grossWeightKg: string; volumeCbm: string }
  >;
  baselinePkg: number;
  onChangeAllocation: (
    resultKey: string,
    patch: { packageCount?: number; grossWeightKg?: string; volumeCbm?: string },
  ) => void;
  onFillRemaining: (resultKey: string) => void;
}

/** 结果票件重尺分配表：各货物项与共享箱按行渲染，列设置在同结构表间共享。 */
function SplitResultAllocationTable({
  tableKey,
  results,
  allocMap,
  baselinePkg,
  onChangeAllocation,
  onFillRemaining,
}: SplitResultAllocationTableProps) {
  const columns: ColumnsType<ResultConfig> = [
    {
      title: '结果票',
      key: 'result',
      width: 200,
      render: (_, r) => (
        <span>
          <Tag color={r.role === 'ORIGINAL' ? 'default' : 'blue'}>
            {r.role === 'ORIGINAL' ? '原' : '新'}
          </Tag>
          {r.title}
        </span>
      ),
    },
    {
      title: '分配件数',
      key: 'packageCount',
      width: 160,
      render: (_, r) => (
        <InputNumber
          min={0}
          max={baselinePkg}
          value={allocMap[r.key]?.packageCount ?? 0}
          onChange={(val) => {
            onChangeAllocation(r.key, { packageCount: Number(val) || 0 });
          }}
        />
      ),
    },
    {
      title: '分配毛重 (KGS)',
      key: 'grossWeightKg',
      width: 180,
      render: (_, r) => (
        <Input
          value={allocMap[r.key]?.grossWeightKg ?? '0'}
          onChange={(e) => {
            onChangeAllocation(r.key, { grossWeightKg: e.target.value });
          }}
        />
      ),
    },
    {
      title: '分配体积 (CBM)',
      key: 'volumeCbm',
      width: 180,
      render: (_, r) => (
        <Input
          value={allocMap[r.key]?.volumeCbm ?? '0'}
          onChange={(e) => {
            onChangeAllocation(r.key, { volumeCbm: e.target.value });
          }}
        />
      ),
    },
    {
      title: '快捷操作',
      key: 'quickFill',
      render: (_, r) => (
        <Button
          size="small"
          type="link"
          onClick={() => onFillRemaining(r.key)}
        >
          填入剩余
        </Button>
      ),
    },
  ];

  const columnSettings = useColumnSettings<ColumnsType<ResultConfig>[number]>({
    tableKey,
    columns,
  });

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
        {columnSettings.entry}
      </div>
      <Table
        dataSource={results}
        rowKey="key"
        pagination={false}
        size="small"
        columns={columnSettings.columns}
      />
      {columnSettings.modal}
    </>
  );
}
