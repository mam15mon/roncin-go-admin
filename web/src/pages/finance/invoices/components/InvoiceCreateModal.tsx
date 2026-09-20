import { ProTable } from '@ant-design/pro-components';
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
import { MODAL_SIZE } from '@/components/ui';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import {
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceListInvoiceCreationBills,
  settlementServiceListInvoiceProfilesForBill,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';

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
  onOk: () => Promise<void>;
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
  const queryClient = useQueryClient();
  const [organizationID, setOrganizationID] = useState<string>();
  const [selectedProfile, setSelectedProfile] =
    useState<API.FinanceInvoiceProfileOption>();

  // 可开票所属公司候选：弹窗打开即拉取；动态错误文案包装进 Error 交全局 onError。
  const organizationQuery = useQuery({
    queryKey: [...INVOICE_CREATE_QUERY_BASE, 'organization-options'],
    enabled: open,
    queryFn: async () => {
      try {
        const response = await settlementServiceListFinanceOrganizationOptions({
          purpose:
            FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_INVOICE_CREATE,
        });
        return response.data ?? [];
      } catch (error) {
        throw new Error(getErrorMessage(error, '加载可开票所属公司失败'));
      }
    },
  });

  // 开票抬头联动：跟随首张已选账单自动加载，账单变化即换 key 重查。
  const firstBillId = selectedBills[0]?.id;
  const profilesQuery = useQuery({
    queryKey: [
      ...INVOICE_CREATE_QUERY_BASE,
      'invoice-profiles',
      { billId: firstBillId },
    ],
    enabled: open && Boolean(firstBillId),
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
    const profiles = profilesQuery.data;
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
  }, [createForm, message, open, profilesQuery.data]);

  // 关闭弹窗即重置组织与抬头选择，并清除本弹窗查询缓存，
  // 避免在途/缓存响应在下次打开时回填上一次会话的数据。
  useEffect(() => {
    if (open) return;
    setOrganizationID(undefined);
    setSelectedProfile(undefined);
    createForm.setFieldValue('invoiceProfileId', undefined);
    queryClient.removeQueries({ queryKey: INVOICE_CREATE_QUERY_BASE });
  }, [createForm, open, queryClient]);

  const clearSelection = () => {
    setSelectedIDs([]);
    setSelectedBills([]);
    setSelectedProfile(undefined);
    createForm.setFieldValue('invoiceProfileId', undefined);
  };

  return (
    <Modal
      title="从已确认账单创建开票记录"
      open={open}
      width={MODAL_SIZE.LG}
      confirmLoading={submitting}
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={createForm} layout="inline" style={{ marginBottom: 12 }}>
        <Form.Item label="所属公司">
          <Select
            aria-label="所属公司"
            allowClear
            placeholder="请先选择可开票所属公司"
            style={{ width: 240 }}
            value={organizationID}
            options={(organizationQuery.data ?? []).map((item) => ({
              value: item.id,
              label: item.name ?? item.code ?? item.id,
            }))}
            onChange={(value) => {
              setOrganizationID(value);
              clearSelection();
            }}
          />
        </Form.Item>
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
            disabled={!selectedBills[0] || !organizationID}
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
        options={false}
        size="small"
        bordered
        columns={[
          { title: '账单编号', dataIndex: 'billNo' },
          {
            title: '方向',
            dataIndex: 'direction',
            renderText: (v) => (v === 'RECEIVABLE' ? '销项' : '进项'),
          },
          { title: '结算单位', dataIndex: 'settlementPartyName' },
          {
            title: '金额',
            render: (_, r) => `${r.totalAmount} ${r.currency}`,
          },
          { title: '税额', dataIndex: 'taxAmount' },
        ]}
        rowSelection={{
          selectedRowKeys: selectedIDs,
          preserveSelectedRowKeys: true,
          onChange: (keys, rows) => {
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
                Boolean(f) &&
                (r.direction !== f.direction ||
                  r.organizationId !== f.organizationId ||
                  r.settlementPartyId !== f.settlementPartyId ||
                  r.currency !== f.currency),
            };
          },
        }}
        request={async (p) => {
          if (!organizationID) {
            return { data: [], success: true, total: 0 };
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
