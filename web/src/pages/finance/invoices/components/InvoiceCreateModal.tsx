import { ProTable } from '@ant-design/pro-components';
import {
  Descriptions,
  Form,
  type FormInstance,
  Input,
  Modal,
  Select,
  Typography,
} from 'antd';
import React from 'react';
import { FinanceBillStatus } from '@/enums.generated';
import { settlementServiceListBills } from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';

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
  availableProfiles: API.PartnerInvoiceProfile[];
  selectedProfile?: API.PartnerInvoiceProfile;
  setSelectedProfile: (profile?: API.PartnerInvoiceProfile) => void;
  loadSelectedProfiles: (partyId?: string) => Promise<void>;
  onOk: () => Promise<void>;
}

export default function InvoiceCreateModal({
  open,
  onCancel,
  submitting,
  createForm,
  selectedBills,
  selectedIDs,
  setSelectedIDs,
  setSelectedBills,
  availableProfiles,
  selectedProfile,
  setSelectedProfile,
  loadSelectedProfiles,
  onOk,
}: InvoiceCreateModalProps) {
  return (
    <Modal
      title="从已确认账单创建开票记录"
      open={open}
      width={1050}
      confirmLoading={submitting}
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={createForm} layout="inline" style={{ marginBottom: 12 }}>
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
            disabled={!selectedBills[0]}
            options={availableProfiles.map((item) => ({
              value: item.id,
              label: `${item.invoiceTitle}${item.isDefault ? '（默认）' : ''}`,
            }))}
            onChange={(id) => {
              const profile = availableProfiles.find((item) => item.id === id);
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
            {selectedProfile?.invoiceTitle || (
              <Text type="danger">未配置</Text>
            )}
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
            const first = keys.map((key) => m.get(String(key))).find(Boolean);
            if (
              first?.settlementPartyId !==
              selectedBills[0]?.settlementPartyId
            ) {
              void loadSelectedProfiles(first?.settlementPartyId);
            } else if (!first) {
              void loadSelectedProfiles(undefined);
            }
          },
          getCheckboxProps: (r) => {
            const f = selectedBills[0];
            return {
              disabled:
                Boolean(f) &&
                (r.direction !== f.direction ||
                  r.settlementPartyId !== f.settlementPartyId ||
                  r.currency !== f.currency),
            };
          },
        }}
        request={async (p) => {
          const r = await settlementServiceListBills({
            page: p.current,
            pageSize: p.pageSize,
            status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
          });
          return toTableRequest(r);
        }}
      />
    </Modal>
  );
}
