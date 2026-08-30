import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Popconfirm, Space, Tag } from 'antd';
import {
  FEE_BILLED,
  FEE_CONFIRMED,
  FEE_DRAFT,
  feeStatusCode,
} from './components/fees/feeConstants';
import { feeBaseColumns } from './components/fees/feeBaseColumns';

interface OrderFeePanelColumnsDeps {
  canUpdate: boolean;
  canDelete: boolean;
  onEdit: (fee: API.OrderFee) => void;
  onConfirmFee: (fee: API.OrderFee) => void | Promise<void>;
  onReopenFee: (fee: API.OrderFee) => void;
  onCancelFee: (fee: API.OrderFee) => void;
}

/** 订单费用抽屉面板的表格列（较费用工作台为精简版）。 */
export function buildOrderFeePanelColumns({
  canUpdate,
  canDelete,
  onEdit,
  onConfirmFee,
  onReopenFee,
  onCancelFee,
}: OrderFeePanelColumnsDeps): ProColumns<API.OrderFee>[] {
  return [
    {
      title: '标签',
      dataIndex: 'tags',
      width: 120,
      render: (_, row) =>
        row.tags?.length ? (
          row.tags.map((tag) => (
            <Tag
              key={tag.id}
              style={
                tag.groupColor
                  ? {
                      color: tag.groupColor,
                      borderColor: tag.groupColor,
                      marginInlineEnd: 4,
                    }
                  : { marginInlineEnd: 4 }
              }
            >
              {tag.name}
            </Tag>
          ))
        ) : (
          '-'
        ),
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
            (feeStatusCode(record.status) === FEE_DRAFT ||
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
          {canUpdate && feeStatusCode(record.status) === FEE_DRAFT && (
            <Popconfirm
              title="确认后该费用才能进入账单，确定继续？"
              onConfirm={async () => {
                await onConfirmFee(record);
              }}
            >
              <Button type="link" size="small">
                确认
              </Button>
            </Popconfirm>
          )}
          {canUpdate && feeStatusCode(record.status) === FEE_CONFIRMED && (
            <Button
              type="link"
              size="small"
              onClick={() => onReopenFee(record)}
            >
              撤回
            </Button>
          )}
          {canDelete &&
            (feeStatusCode(record.status) === FEE_DRAFT ||
              feeStatusCode(record.status) === FEE_CONFIRMED) && (
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
