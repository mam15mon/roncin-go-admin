import {
  EditOutlined,
  FileTextOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Space, Typography } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import { partnerContractStatusMeta, statusTag } from '@/constants/statusMeta';
import { PartnerContractStatus } from '@/enums.generated';
import {
  partnerServiceCreatePartnerContract,
  partnerServiceListPartnerContracts,
  partnerServiceUpdatePartnerContract,
} from '@/services/roncin/partnerService';
import { toTableRequest } from '@/utils/api';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

const contractStatusOptions = [
  ...Object.entries(partnerContractStatusMeta).map(([value, meta]) => ({
    label: meta.text,
    value: Number(value),
  })),
];

type ContractFormValues = {
  contractNo?: string;
  name?: string;
  status?: number;
  dateRange?: [Dayjs, Dayjs];
  paymentTerms?: string;
  disputeResolution?: string;
  otherNotes?: string;
};

type ContractsPanelProps = {
  partner?: API.Partner;
  canManage: boolean;
};

function availableContractStatuses(contract?: API.PartnerContract) {
  if (!contract) return contractStatusOptions.slice(0, 2);
  const allowedStatuses = new Set(contract.allowedStatuses ?? []);
  return contractStatusOptions.filter((option) =>
    allowedStatuses.has(option.value),
  );
}

export default function ContractsPanel({
  partner,
  canManage,
}: ContractsPanelProps) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingContract, setEditingContract] = useState<API.PartnerContract>();

  const openForm = (contract?: API.PartnerContract) => {
    setEditingContract(contract);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const columns: ProColumns<API.PartnerContract>[] = [
    {
      title: '合同编号',
      dataIndex: 'contractNo',
      width: 150,
      copyable: true,
      render: (no) => (
        <Text style={{ fontFamily: 'monospace', fontWeight: 500 }}>{no}</Text>
      ),
    },
    {
      title: '合同名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (name) => <Text strong>{name}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (_, record) =>
        statusTag(partnerContractStatusMeta, record.status ?? 0, '未知'),
    },
    {
      title: '生效起止日期',
      dataIndex: 'startDate',
      width: 220,
      render: (_, record) => (
        <span>
          {formatDate(record.startDate, 'date')}
          {' ~ '}
          {formatDate(record.endDate, 'date')}
        </span>
      ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      fixed: 'right',
      render: (_, record) =>
        canManage ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            style={{ padding: 0 }}
            onClick={() => openForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  return (
    <>
      <ProTable<API.PartnerContract>
        headerTitle={
          <Space size={6}>
            <FileTextOutlined style={{ color: '#1677ff' }} />
            <span>商务框架合同列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        search={false}
        pagination={false}
        request={async () => {
          if (!partner?.id) return { data: [], success: true };
          const response = await partnerServiceListPartnerContracts({
            partnerId: partner.id,
          });
          return toTableRequest(response);
        }}
        toolBarRender={() =>
          canManage
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => openForm()}
                >
                  新增合同
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<ContractFormValues>
        title={editingContract ? '编辑商务合同' : '新增商务合同'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingContract
            ? {
                ...editingContract,
                dateRange:
                  editingContract.startDate && editingContract.endDate
                    ? [
                        dayjs(editingContract.startDate),
                        dayjs(editingContract.endDate),
                      ]
                    : undefined,
              }
            : {
                status:
                  PartnerContractStatus.PARTNER_CONTRACT_STATUS_PENDING,
              }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (
            !partner?.id ||
            !values.name ||
            !values.status ||
            !values.dateRange
          ) {
            return false;
          }
          const common = {
            name: values.name.trim(),
            status: values.status,
            startDate: values.dateRange[0].startOf('day').toISOString(),
            endDate: values.dateRange[1].startOf('day').toISOString(),
            paymentTerms: values.paymentTerms,
            disputeResolution: values.disputeResolution,
            otherNotes: values.otherNotes,
          };
          if (editingContract?.id) {
            await partnerServiceUpdatePartnerContract(
              { partnerId: partner.id, id: editingContract.id },
              {
                partnerId: partner.id,
                id: editingContract.id,
                contract: common,
              },
            );
            message.success('合同已成功更新');
          } else {
            if (!values.contractNo) return false;
            await partnerServiceCreatePartnerContract(
              { partnerId: partner.id },
              {
                partnerId: partner.id,
                contract: { ...common, contractNo: values.contractNo.trim() },
              },
            );
            message.success('合同已成功创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="contractNo"
          label="合同唯一编号"
          placeholder="例如：CT-2026-0001"
          disabled={Boolean(editingContract)}
          rules={[{ required: !editingContract, message: '请输入合同编号' }]}
        />
        <ProFormText
          name="name"
          label="合同名称"
          placeholder="例如：2026年度国际海运出口货代服务框架协议"
          rules={[{ required: true, message: '请输入合同名称' }]}
        />
        <ProFormSelect
          name="status"
          label="合同生命周期状态"
          options={availableContractStatuses(editingContract)}
          rules={[{ required: true, message: '请选择合同状态' }]}
        />
        <ProFormDateRangePicker
          name="dateRange"
          label="合同有效起止期限"
          rules={[{ required: true, message: '请选择合同期限' }]}
          fieldProps={{ allowEmpty: [false, false] }}
        />
        <ProFormTextArea
          name="paymentTerms"
          label="付款与账期约定"
          placeholder="请输入结算账期、支付方式与违约金约定"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
        <ProFormTextArea
          name="disputeResolution"
          label="争议管辖与解决"
          placeholder="例如：提交上海国际仲裁中心仲裁"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
        <ProFormTextArea
          name="otherNotes"
          label="补充约定事项"
          placeholder="其他补充约定"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
      </ModalForm>
    </>
  );
}
