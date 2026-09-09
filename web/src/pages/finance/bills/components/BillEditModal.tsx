import {
  DatePicker,
  Form,
  type FormInstance,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
} from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { settlementServiceListBillSettlementAccountUpdateCandidates } from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';
import type { BillFormValues } from './billConstants';

interface BillEditModalProps {
  open: boolean;
  editing?: API.FinanceBill;
  form: FormInstance<BillFormValues>;
  submitting: boolean;
  onCancel: () => void;
  onOk: () => Promise<void>;
}

export default function BillEditModal({
  open,
  editing,
  form,
  submitting,
  onCancel,
  onOk,
}: BillEditModalProps) {
  const [accountOptions, setAccountOptions] = useState<
    API.FinanceSettlementAccountOption[]
  >([]);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const requestSequenceRef = useRef(0);

  useEffect(() => {
    const requestSequence = ++requestSequenceRef.current;
    setAccountOptions([]);
    if (!open || !editing?.id) {
      setAccountsLoading(false);
      return undefined;
    }
    setAccountsLoading(true);
    void settlementServiceListBillSettlementAccountUpdateCandidates({
      billId: editing.id,
    })
      .then((response) => {
        if (requestSequence !== requestSequenceRef.current) return;
        const accounts = unwrapList(response);
        setAccountOptions(accounts);
        const selected = form.getFieldValue('settlementAccountId');
        const selectedStillAvailable = accounts.some(
          (account) => account.id === selected,
        );
        if (selected && selectedStillAvailable) return;

        const defaultAccount = accounts.find((account) => account.isDefault);
        form.setFieldValue('settlementAccountId', defaultAccount?.id);
      })
      .catch(() => {
        if (requestSequence === requestSequenceRef.current)
          setAccountOptions([]);
      })
      .finally(() => {
        if (requestSequence === requestSequenceRef.current) {
          setAccountsLoading(false);
        }
      });
    return () => {
      requestSequenceRef.current += 1;
    };
  }, [editing?.id, form, open]);

  return (
    <Modal
      title={`编辑账单 ${editing?.billNo || ''}`}
      open={open}
      width={680}
      destroyOnHidden
      confirmLoading={submitting}
      okText="保存"
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={form} layout="vertical">
        <Space size={16} align="start" wrap style={{ width: '100%' }}>
          <Form.Item
            name="statementTitle"
            label="对账抬头"
            rules={[
              { required: true, whitespace: true, message: '请输入对账抬头' },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
            style={{ minWidth: 260 }}
          >
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item
            name="billDate"
            label="账单日期"
            extra="修改账单日期将自动按新账单日重置 BILL 汇率快照"
            rules={[{ required: true, message: '请选择账单日期' }]}
          >
            <DatePicker allowClear={false} />
          </Form.Item>
          <Form.Item name="paymentTermsDays" label="账期（天，可选）">
            <InputNumber min={0} max={3650} precision={0} />
          </Form.Item>
          <Form.Item
            name="settlementAccountId"
            label="结算账户"
            rules={[{ required: true, message: '请选择结算账户' }]}
            style={{ minWidth: 620 }}
          >
            <Select
              allowClear
              loading={accountsLoading}
              placeholder="请选择可用于该账单的启用账户"
              options={accountOptions.map((account) => ({
                value: account.id,
                label: `${account.name || account.accountHolder || '未命名账户'}｜${account.bankName || '-'}${account.accountNo ? ` · ${account.accountNo}` : ''}｜${account.currency || '-'}`,
              }))}
            />
          </Form.Item>
          <Form.Item name="note" label="备注" style={{ minWidth: 620 }}>
            <Input maxLength={500} />
          </Form.Item>
        </Space>
      </Form>
    </Modal>
  );
}
