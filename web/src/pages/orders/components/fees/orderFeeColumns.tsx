import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button } from 'antd';
import { feeBaseColumns } from './feeBaseColumns';

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
              <Button
                key="edit"
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => onOpenModal(direction, record)}
              >
                编辑
              </Button>,
              // 已建账（含草稿账单占用）费用不可删除，需先经财务链路取消账单；
              // 补录生成费用由服务端拒绝普通删除并提示前往补录申请撤销。
              record.hasActiveBill !== true && (
                <Button
                  key="cancel"
                  type="link"
                  size="small"
                  danger
                  onClick={() => onCancelFee(record)}
                >
                  删除
                </Button>
              ),
            ].filter(Boolean),
    },
  ];
}
