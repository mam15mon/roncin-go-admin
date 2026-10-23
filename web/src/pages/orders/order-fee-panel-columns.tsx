import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Space } from 'antd';
import { BusinessTagList } from '@/components/business-tag/BusinessTagList';
import { feeBaseColumns } from './components/fees/feeBaseColumns';

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
          {canUpdate && record.canEdit === true && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => onEdit(record)}
            >
              编辑
            </Button>
          )}
          {/* 删除与修改分别消费服务端能力，账单占用和审核锁由后端统一判断。 */}
          {canDelete && record.canDelete === true && (
            <Button
              type="link"
              danger
              size="small"
              onClick={() => onCancelFee(record)}
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ];
}
