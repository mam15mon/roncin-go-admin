import { SearchOutlined } from '@ant-design/icons';
import {
  Button,
  Card,
  Empty,
  Input,
  InputNumber,
  Pagination,
  Table,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React from 'react';
import {
  CANDIDATE_PAGE_SIZE,
  type CargoAllocationItem,
  type DraftAllocation,
} from './seaSharedContainerModels';

const { Text } = Typography;

type SeaSharedContainerAllocationTableProps = {
  candidates: API.SeaSharedContainerCandidateOrder[];
  flatCargoList: CargoAllocationItem[];
  candidatePage: number;
  candidateTotal: number;
  candidateSearching: string;
  isConfirmed: boolean;
  canUpdate: boolean;
  onSearchingChange: React.Dispatch<React.SetStateAction<string>>;
  onSearch: (value: string) => void;
  onPageChange: React.Dispatch<React.SetStateAction<number>>;
  onUpdateDraft: (
    identity: DraftAllocation,
    field: 'packageCount' | 'grossWeightKg' | 'volumeCbm',
    value: number | string | null,
  ) => void;
  onFillAllCargo: (record: CargoAllocationItem) => void;
};

/**
 * 跨订单货物分配：服务端按“订单”分页，本页订单的全部货物行完整展示，
 * 分页由独立的 Pagination 控制订单页，Table 不做本地二次分页
 */
export default function SeaSharedContainerAllocationTable({
  candidates,
  flatCargoList,
  candidatePage,
  candidateTotal,
  candidateSearching,
  isConfirmed,
  canUpdate,
  onSearchingChange,
  onSearch,
  onPageChange,
  onUpdateDraft,
  onFillAllCargo,
}: SeaSharedContainerAllocationTableProps) {
  const columns: ColumnsType<CargoAllocationItem> = [
    {
      title: '所属订单 / 分单',
      dataIndex: 'orderNo',
      width: 180,
      render: (_, record) => (
        <div>
          <Text strong>{record.orderNo}</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>
            HBL: {record.houseNo || '直单'}
          </Text>
        </div>
      ),
    },
    {
      title: '货物描述',
      dataIndex: 'cargoName',
      width: 160,
    },
    {
      title: '货物基准总量',
      width: 160,
      render: (_, record) => (
        <div style={{ fontSize: 12 }}>
          <div>件数: {record.totalPackageCount} PCS</div>
          <div>毛重: {record.totalGrossWeightKg} KG</div>
          <div>体积: {record.totalVolumeCbm} CBM</div>
        </div>
      ),
    },
    {
      title: '分配至本共享箱件数 (PCS)',
      width: 140,
      render: (_, record) => (
        <InputNumber
          min={0}
          max={record.totalPackageCount ?? 0}
          value={record.draft.packageCount}
          disabled={isConfirmed || !canUpdate}
          onChange={(val) => onUpdateDraft(record.draft, 'packageCount', val)}
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: '分配毛重 (KG)',
      width: 150,
      render: (_, record) => (
        <Input
          value={record.draft.grossWeightKg}
          disabled={isConfirmed || !canUpdate}
          onChange={(e) =>
            onUpdateDraft(record.draft, 'grossWeightKg', e.target.value)
          }
          placeholder="0.000"
        />
      ),
    },
    {
      title: '分配体积 (CBM)',
      width: 150,
      render: (_, record) => (
        <Input
          value={record.draft.volumeCbm}
          disabled={isConfirmed || !canUpdate}
          onChange={(e) =>
            onUpdateDraft(record.draft, 'volumeCbm', e.target.value)
          }
          placeholder="0.000000"
        />
      ),
    },
    {
      title: '快捷操作',
      width: 110,
      render: (_, record) =>
        !isConfirmed && canUpdate ? (
          <Button
            size="small"
            type="link"
            onClick={() => onFillAllCargo(record)}
          >
            全部填入
          </Button>
        ) : (
          '-'
        ),
    },
  ];

  return (
    <Card
      title={`同航次待分配 HOUSE 订单货物 (第 ${candidatePage} 页，共 ${candidateTotal} 票订单)`}
      size="small"
      extra={
        <Input.Search
          allowClear
          size="small"
          style={{ width: 260 }}
          placeholder="搜索订单号 / 业务号 / 分单号"
          prefix={<SearchOutlined />}
          value={candidateSearching}
          onChange={(e) => onSearchingChange(e.target.value)}
          onSearch={(value) => onSearch(value)}
        />
      }
    >
      {candidates.length === 0 ? (
        <Empty
          description="未找到符合条件的 HOUSE 订单可供拼箱"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      ) : (
        <>
          <Table
            columns={columns}
            dataSource={flatCargoList}
            rowKey={(r) => r.key}
            pagination={false}
            size="small"
            bordered
          />
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginTop: 12,
            }}
          >
            <Pagination
              current={candidatePage}
              pageSize={CANDIDATE_PAGE_SIZE}
              total={candidateTotal}
              showSizeChanger={false}
              showTotal={(total) => `共 ${total} 票订单`}
              onChange={(page) => onPageChange(page)}
            />
          </div>
        </>
      )}
    </Card>
  );
}
