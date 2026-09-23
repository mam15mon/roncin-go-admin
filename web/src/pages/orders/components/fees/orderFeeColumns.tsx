import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button } from 'antd';
import { feeBaseColumns } from './feeBaseColumns';
import {
  FEE_BILLED,
  FEE_UNBILLED,
  feeStatusCode,
} from './feeConstants';

type OrderFeeColumnProps = {
  direction: number;
  feeWritesDisabled: boolean;
  onOpenModal: (direction: number, record?: API.OrderFee) => void;
  onCancelFee: (record: API.OrderFee) => void;
};

export function getOrderFeeTableColumns({
  direction,
  feeWritesDisabled,
  onOpenModal,
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
        feeWritesDisabled
          ? []
          : [
              (feeStatusCode(record.status) === FEE_UNBILLED ||
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
              feeStatusCode(record.status) === FEE_UNBILLED && (
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
