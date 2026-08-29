import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Popconfirm, Space, Tag } from 'antd';
import {
  FEE_BILLED,
  FEE_CANCELLED,
  FEE_CONFIRMED,
  FEE_DRAFT,
  PAYABLE,
  feeDirectionCode,
  feeStatusCode,
} from './components/fees/feeConstants';
import { trimExactDecimal } from './order-fee-decimal';

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
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => {
        if (feeStatusCode(record.status) === FEE_CONFIRMED)
          return <Tag color="green">已确认</Tag>;
        if (feeStatusCode(record.status) === FEE_BILLED)
          return <Tag color="blue">已进账单</Tag>;
        if (feeStatusCode(record.status) === FEE_CANCELLED)
          return <Tag>已作废</Tag>;
        return <Tag color="gold">草稿</Tag>;
      },
    },
    {
      title: '收付方向',
      dataIndex: 'direction',
      width: 90,
      render: (_, record) =>
        feeDirectionCode(record.direction) === PAYABLE ? (
          <Tag color="volcano">应付</Tag>
        ) : (
          <Tag color="green">应收</Tag>
        ),
    },
    {
      title: '费用代码',
      dataIndex: 'feeCode',
      width: 130,
      copyable: true,
    },
    {
      title: '费用名称',
      dataIndex: 'feeName',
      width: 150,
      ellipsis: true,
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 190,
      ellipsis: true,
    },
    {
      title: '计费单位',
      dataIndex: 'billingUnit',
      width: 90,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      width: 110,
      align: 'right',
      render: (_, record) => trimExactDecimal(record.quantity),
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      width: 130,
      align: 'right',
      render: (_, record) => trimExactDecimal(record.unitPrice),
    },
    {
      title: '总金额',
      dataIndex: 'totalAmount',
      width: 150,
      align: 'right',
      render: (_, record) => (
        <strong>
          {trimExactDecimal(record.totalAmount)} {record.currency}
        </strong>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 160,
      align: 'right',
      render: (_, record) => (
        <Space size={4}>
          {trimExactDecimal(record.exchangeRate)}
          {record.exchangeRateSource === 'MANUAL' && (
            <Tag color="gold">手工</Tag>
          )}
          {record.exchangeRateSource === 'SYSTEM' && (
            <Tag color="blue">系统</Tag>
          )}
          {record.exchangeRateSource === 'BASE_CURRENCY' && <Tag>本币</Tag>}
        </Space>
      ),
    },
    {
      title: '费用日期',
      dataIndex: 'expenseDate',
      width: 110,
    },
    {
      title: '备注',
      dataIndex: 'note',
      width: 180,
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
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
