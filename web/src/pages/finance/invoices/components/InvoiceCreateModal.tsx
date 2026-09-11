import { ProTable } from '@ant-design/pro-components';
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
import React, { useEffect, useRef, useState } from 'react';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import {
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceListInvoiceCreationBills,
  settlementServiceListInvoiceProfilesForBill,
} from '@/services/roncin/settlementService';
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
  onOk,
}: InvoiceCreateModalProps) {
  const { message } = App.useApp();
  const [organizationID, setOrganizationID] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [availableProfiles, setAvailableProfiles] = useState<
    API.FinanceInvoiceProfileOption[]
  >([]);
  const [selectedProfile, setSelectedProfile] =
    useState<API.FinanceInvoiceProfileOption>();
  const profileRequestSequence = useRef(0);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_INVOICE_CREATE,
    })
      .then((response) => {
        if (!cancelled) setOrganizationOptions(response.data ?? []);
      })
      .catch((error: any) => {
        if (!cancelled) {
          setOrganizationOptions([]);
          message.error(error.message || '加载可开票所属公司失败');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [message, open]);

  useEffect(() => {
    if (!open) {
      profileRequestSequence.current += 1;
      setOrganizationID(undefined);
      setOrganizationOptions([]);
      setAvailableProfiles([]);
      setSelectedProfile(undefined);
      createForm.setFieldValue('invoiceProfileId', undefined);
    }
  }, [createForm, open]);

  const clearProfiles = () => {
    profileRequestSequence.current += 1;
    setAvailableProfiles([]);
    setSelectedProfile(undefined);
    createForm.setFieldValue('invoiceProfileId', undefined);
  };

  const loadSelectedProfiles = async (billID?: string) => {
    const requestSequence = ++profileRequestSequence.current;
    setAvailableProfiles([]);
    setSelectedProfile(undefined);
    createForm.setFieldValue('invoiceProfileId', undefined);
    if (!billID) return;
    try {
      const response = await settlementServiceListInvoiceProfilesForBill(
        { billId: billID },
        { skipErrorHandler: true },
      );
      if (requestSequence !== profileRequestSequence.current) return;
      const profiles = response.data?.data ?? [];
      setAvailableProfiles(profiles);
      const selected = profiles.find((item) => item.isDefault) || profiles[0];
      setSelectedProfile(selected);
      createForm.setFieldValue('invoiceProfileId', selected?.id);
      if (selected?.defaultInvoiceType) {
        createForm.setFieldValue('invoiceType', selected.defaultInvoiceType);
      }
      if (!selected) {
        message.warning(
          '该结算单位尚未配置可用开票抬头，请先到往来单位档案维护',
        );
      }
    } catch (error: any) {
      if (requestSequence !== profileRequestSequence.current) return;
      message.error(error.message || '加载开票抬头失败');
    }
  };

  const clearSelection = () => {
    setSelectedIDs([]);
    setSelectedBills([]);
    clearProfiles();
  };

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
        <Form.Item label="所属公司">
          <Select
            aria-label="所属公司"
            allowClear
            placeholder="请先选择可开票所属公司"
            style={{ width: 240 }}
            value={organizationID}
            options={organizationOptions.map((item) => ({
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
            const first = keys.map((key) => m.get(String(key))).find(Boolean);
            if (first?.id !== selectedBills[0]?.id) {
              void loadSelectedProfiles(first?.id);
            }
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
