import type { ProColumns } from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Descriptions,
  Form,
  type FormInstance,
  Input,
  Modal,
  Select,
  Typography,
} from 'antd';
import React, { useEffect, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { MODAL_SIZE, ResponsiveProTable as ProTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  settlementServiceListInvoiceCreationBills,
  settlementServiceListInvoiceProfilesForBill,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatAmount } from '@/utils/format';

const { Text } = Typography;

interface InvoiceCreateModalProps {
  open: boolean;
  onCancel: () => void;
  submitting: boolean;
  createForm: FormInstance;
  selectedBills: API.FinanceBill[];
  selectedIDs: React.Key[];
  setSelectedIDs: (keys: React.Key[]) => void;
  setSelectedBills: (bills: API.FinanceBill[]) => void;
  onOk: () => Promise<unknown>;
}

/** 服务端状态域前缀：开票创建弹窗查询的统一 key 前缀。 */
const INVOICE_CREATE_QUERY_BASE = ['finance', 'invoice-create'] as const;

export default function InvoiceCreateModal({
  open,
  onCancel,
  submitting,
  createForm,
  selectedBills,
  selectedIDs,
  setSelectedIDs,
  setSelectedBills,
  onOk,
}: InvoiceCreateModalProps) {
  const { message } = App.useApp();
  const { markDirty } = useUnsavedChangesGuard({ enabled: open });
  const queryClient = useQueryClient();
  const { initialState } = useInitialState();
  const access = useAccess();
  const organizationID = initialState?.currentUser?.currentOrganization?.id;
  const canOperate =
    Boolean(initialState?.currentUser?.id) &&
    access.canCreateFinanceInvoices &&
    access.canOperateOrganization(organizationID);
  const validSelection = selectedBills.every(
    (bill) => Boolean(organizationID) && bill.organizationId === organizationID,
  );
  const [selectedProfile, setSelectedProfile] =
    useState<API.FinanceInvoiceProfileOption>();

  // 开票抬头联动：跟随首张已选账单自动加载，账单变化即换 key 重查。
  const firstBillId = selectedBills[0]?.id;
  const profilesQuery = useQuery({
    queryKey: [
      ...INVOICE_CREATE_QUERY_BASE,
      'invoice-profiles',
      { organizationId: organizationID, billId: firstBillId },
    ],
    enabled: open && canOperate && validSelection && Boolean(firstBillId),
    queryFn: async () => {
      try {
        const response = await settlementServiceListInvoiceProfilesForBill(
          { billId: firstBillId as string },
          { skipErrorHandler: true },
        );
        return response.data?.data ?? [];
      } catch (error) {
        throw new Error(getErrorMessage(error, '加载开票抬头失败'));
      }
    },
  });

  // 抬头就绪后回填默认选择；账单未定或加载中时清空选择，旧抬头不残留。
  useEffect(() => {
    if (!open) return;
    const profiles =
      canOperate && validSelection ? profilesQuery.data : undefined;
    if (!profiles) {
      setSelectedProfile(undefined);
      createForm.setFieldValue('invoiceProfileId', undefined);
      return;
    }
    const selected = profiles.find((item) => item.isDefault) || profiles[0];
    setSelectedProfile(selected);
    createForm.setFieldValue('invoiceProfileId', selected?.id);
    if (selected?.defaultInvoiceType) {
      createForm.setFieldValue('invoiceType', selected.defaultInvoiceType);
    }
    if (!selected) {
      message.warning('该结算单位尚未配置可用开票抬头，请先到往来单位档案维护');
    }
  }, [
    canOperate,
    validSelection,
    createForm,
    message,
    open,
    profilesQuery.data,
  ]);

  // 关闭弹窗即重置抬头选择，并清除本弹窗查询缓存，
  // 避免在途/缓存响应在下次打开时回填上一次会话的数据。
  useEffect(() => {
    if (open) return;
    setSelectedProfile(undefined);
    createForm.setFieldValue('invoiceProfileId', undefined);
    queryClient.removeQueries({ queryKey: INVOICE_CREATE_QUERY_BASE });
  }, [createForm, open, queryClient]);

  const columns: ProColumns<API.FinanceBill>[] = [
    { title: '账单编号', dataIndex: 'billNo' },
    {
      title: '方向',
      dataIndex: 'direction',
      renderText: (v) => (v === 'RECEIVABLE' ? '销项' : '进项'),
    },
    { title: '结算单位', dataIndex: 'settlementPartyName' },
    {
      title: '金额',
      key: 'totalAmount',
      render: (_, r) => `${formatAmount(r.invoiceableAmount)} ${r.currency}`,
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      render: (_, r) => formatAmount(r.invoiceableTaxAmount),
    },
  ];

  const columnSettings = useColumnSettings<ProColumns<API.FinanceBill>>({
    tableKey: 'finance:invoice-candidate-bills',
    columns,
  });

  return (
    <Modal
      title="从已确认账单创建开票记录"
      open={open}
      width={MODAL_SIZE.LG}
      confirmLoading={submitting}
      onCancel={onCancel}
      onOk={() => {
        if (!canOperate || !validSelection || selectedBills.length === 0) {
          message.warning('请选择当前公司可开票的账单');
          return;
        }
        void onOk();
      }}
    >
      <Form
        form={createForm}
        layout="inline"
        style={{ marginBottom: 12 }}
        onValuesChange={markDirty}
      >
        <Form.Item
          name="invoiceProfileId"
          label="开票抬头"
          rules={[{ required: true, message: '请选择开票抬头' }]}
        >
          <Select
            style={{ width: 300 }}
            placeholder={
              selectedBills[0] ? '请选择该客户的开票抬头' : '请先选择账单'
            }
            disabled={!selectedBills[0] || !canOperate || !validSelection}
            options={(profilesQuery.data ?? []).map((item) => ({
              value: item.id,
              label: `${item.invoiceTitle}${item.isDefault ? '（默认）' : ''}`,
            }))}
            onChange={(id) => {
              const profile = (profilesQuery.data ?? []).find(
                (item) => item.id === id,
              );
              setSelectedProfile(profile);
              if (profile?.defaultInvoiceType) {
                createForm.setFieldValue(
                  'invoiceType',
                  profile.defaultInvoiceType,
                );
              }
            }}
          />
        </Form.Item>
        <Form.Item
          name="invoiceType"
          label="发票类型"
          rules={[{ required: true }]}
        >
          <Select
            style={{ width: 150 }}
            options={[
              { value: 'NORMAL', label: '普通发票' },
              { value: 'SPECIAL', label: '专用发票' },
            ]}
          />
        </Form.Item>
        <Form.Item name="note" label="备注">
          <Input style={{ width: 360 }} maxLength={500} />
        </Form.Item>
      </Form>
      {selectedBills[0] && (
        <Descriptions
          bordered
          size="small"
          column={3}
          style={{ marginBottom: 12 }}
        >
          <Descriptions.Item label="已选开票抬头" span={2}>
            {selectedProfile?.invoiceTitle || <Text type="danger">未配置</Text>}
          </Descriptions.Item>
          <Descriptions.Item label="默认票种">
            {selectedProfile?.defaultInvoiceType === 'SPECIAL'
              ? '专用发票'
              : selectedProfile
                ? '普通发票'
                : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="纳税人识别号" span={3}>
            {selectedProfile?.taxpayerIdentificationNo || '-'}
          </Descriptions.Item>
        </Descriptions>
      )}
      <ProTable<API.FinanceBill>
        key={organizationID || 'no-organization'}
        rowKey="id"
        options={{ reload: true, density: true, setting: false }}
        size="small"
        bordered
        columns={columnSettings.columns}
        toolBarRender={() => [columnSettings.entry]}
        rowSelection={{
          selectedRowKeys: selectedIDs,
          preserveSelectedRowKeys: true,
          onChange: (keys, rows) => {
            markDirty();
            const m = new Map(selectedBills.map((x) => [x.id, x]));
            rows.forEach((x) => {
              m.set(x.id, x);
            });
            setSelectedIDs(keys);
            setSelectedBills(
              keys
                .map((k) => m.get(String(k)))
                .filter(Boolean) as API.FinanceBill[],
            );
            // 抬头查询由首张账单驱动的 queryKey 自动联动，这里不再手动拉取。
          },
          getCheckboxProps: (r) => {
            const f = selectedBills[0];
            return {
              disabled:
                !canOperate ||
                r.organizationId !== organizationID ||
                (Boolean(f) &&
                  (r.direction !== f?.direction ||
                    r.organizationId !== f?.organizationId ||
                    r.settlementPartyId !== f?.settlementPartyId ||
                    r.currency !== f?.currency)),
            };
          },
        }}
        request={async (p) => {
          if (!canOperate || !organizationID) {
            return { data: [], success: false, total: 0 };
          }
          const r = await settlementServiceListInvoiceCreationBills({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            organizationId: organizationID,
          });
          return toTableRequest(r);
        }}
      />
    </Modal>
  );
}
