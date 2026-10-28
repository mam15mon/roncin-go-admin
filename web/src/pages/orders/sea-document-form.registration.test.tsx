import { ProForm, ProFormText } from '@ant-design/pro-components';
import { act, render, waitFor } from '@testing-library/react';
import { Form, type FormInstance } from 'antd';
import { expect, it, vi } from 'vitest';
import {
  buildSeaDocumentUpdate,
  seaDocumentFormValues,
} from './sea-document-form';

it('整页提交保留未展示的件重尺与单证备注，普通资料修改不误写单证', async () => {
  const documents: API.SeaOrderDocuments = {
    documentStructure: 3,
    linkVersion: '3',
    masterBill: {
      id: 'm',
      version: '5',
      content: { shipperText: 'MBL', packageCount: 12 },
    },
    houseBill: {
      id: 'h',
      version: '7',
      houseNo: 'HBL',
      issuerSource: 1,
      note: '已有备注',
      content: { shipperText: 'HBL', grossWeightKg: 123 },
    },
  };
  let form!: FormInstance;
  const submitted = vi.fn();
  function Harness() {
    [form] = Form.useForm();
    return (
      <ProForm
        form={form}
        initialValues={seaDocumentFormValues(documents)}
        onFinish={async (values) => {
          submitted(values);
          return true;
        }}
      >
        <ProFormText name={['seaMasterBillContent', 'shipperText']} />
        <ProFormText name={['seaHouseBill', 'houseNo']} />
        <ProFormText name={['seaHouseBill', 'issuerSource']} />
        <ProFormText name={['seaHouseBill', 'content', 'shipperText']} />
      </ProForm>
    );
  }
  render(<Harness />);
  await act(async () => form.submit());
  await waitFor(() => expect(submitted).toHaveBeenCalled());
  // Ant Form仅提交已注册字段；详情保存必须合并完整store再构造单证请求。
  expect(
    submitted.mock.calls[0][0].seaMasterBillContent.packageCount,
  ).toBeUndefined();
  expect(
    buildSeaDocumentUpdate(form.getFieldsValue(true), documents),
  ).toBeUndefined();
});
