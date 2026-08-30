import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Popconfirm } from 'antd';
import {
  FEE_BILLED,
  FEE_CONFIRMED,
  FEE_DRAFT,
  feeStatusCode,
} from './feeConstants';
import { feeBaseColumns } from './feeBaseColumns';

type OrderFeeColumnProps = {
  direction: number;
  financeLocked: boolean;
  onOpenModal: (direction: number, record?: API.OrderFee) => void;
  onConfirmFee: (record: API.OrderFee) => void;
  onReopenFee: (record: API.OrderFee) => void;
  onCancelFee: (record: API.OrderFee) => void;
};

export function getOrderFeeTableColumns({
  direction,
  financeLocked,
  onOpenModal,
  onConfirmFee,
  onReopenFee,
  onCancelFee,
}: OrderFeeColumnProps): ProColumns<API.OrderFee>[] {
  return [
    ...feeBaseColumns({ variant: 'workbench', direction }),
    {
      title: '操作',
      valueType: 'option',
      width: 110,
      fixed: 'right',
      render: (_, record) =>
        financeLocked
          ? []
          : [
              (feeStatusCode(record.status) === FEE_DRAFT ||
                feeStatusCode(record.status) === FEE_BILLED) && (
                <Button
                  key="edit"
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => onOpenModal(direction, record)}
                >
                  编辑
                </Button>
              ),
              feeStatusCode(record.status) === FEE_DRAFT && (
                <Popconfirm
                  key="confirm"
                  title="确认后该费用才能进入账单，确定继续？"
                  onConfirm={() => onConfirmFee(record)}
                >
                  <Button type="link" size="small">
                    确认
                  </Button>
                </Popconfirm>
              ),
              feeStatusCode(record.status) === FEE_CONFIRMED && (
                <Button
                  key="reopen"
                  type="link"
                  size="small"
                  onClick={() => onReopenFee(record)}
                >
                  撤回
                </Button>
              ),
              (feeStatusCode(record.status) === FEE_DRAFT ||
                feeStatusCode(record.status) === FEE_CONFIRMED) && (
                <Button
                  key="cancel"
                  type="link"
                  size="small"
                  danger
                  onClick={() => onCancelFee(record)}
                >
                  作废
                </Button>
              ),
            ],
    },
  ];
}
