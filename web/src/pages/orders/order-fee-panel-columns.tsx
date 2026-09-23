import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Space } from 'antd';
import { BusinessTagList } from '@/components/business-tag/BusinessTagList';
import { feeBaseColumns } from './components/fees/feeBaseColumns';
import {
  FEE_BILLED,
  FEE_UNBILLED,
  feeStatusCode,
} from './components/fees/feeConstants';

interface OrderFeePanelColumnsDeps {
  canUpdate: boolean;
  canDelete: boolean;
  onEdit: (fee: API.OrderFee) => void;
  onCancelFee: (fee: API.OrderFee) => void;
}

/** 订单费用抽屉面板的表格列（较费用工作台为精简版）。 */
export function buildOrderFeePanelColumns({
  canUpdate,
  canDelete,
  onEdit,
  onCancelFee,
}: OrderFeePanelColumnsDeps): ProColumns<API.OrderFee>[] {
  return [
    {
      title: '标签',
      dataIndex: 'tags',
      width: 120,
      render: (_, row) => <BusinessTagList tags={row.tags} />,
    },
    ...feeBaseColumns({ variant: 'panel' }),
    {
      title: '操作',
      valueType: 'option',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          {canUpdate &&
            (feeStatusCode(record.status) === FEE_UNBILLED ||
              feeStatusCode(record.status) === FEE_BILLED) && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => onEdit(record)}
              >
                编辑
              </Button>
            )}
          {canDelete && feeStatusCode(record.status) === FEE_UNBILLED && (
            <Button
              type="link"
              danger
              size="small"
              onClick={() => onCancelFee(record)}
            >
              作废
            </Button>
          )}
        </Space>
      ),
    },
  ];
}
