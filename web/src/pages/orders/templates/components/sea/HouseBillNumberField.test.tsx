import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { HouseBillNumberField } from './HouseBillIdentityFields';

function Harness({ exposeForm }: { exposeForm: (form: FormInstance) => void }) {
  const [form] = Form.useForm();
  exposeForm(form);
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form form={form}>
        <HouseBillNumberField fieldKey="seaHouseBill" />
      </Form>
    </ConfigProvider>
  );
}

describe('正式分单号校验', () => {
  it('空白号码必填且候选批次晚到时重验已填号码', async () => {
    let form!: FormInstance;
    render(
      <Harness
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: ' ' },
    });
    await screen.findByText('分单号不能为空');
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: ' hbl-used ' },
    });
    await waitFor(() =>
      expect(form.getFieldError(['seaHouseBill', 'houseNo'])).toEqual([]),
    );
    await act(async () => {
      form.setFieldValue('seaMasterBillBatchHouseNos', ['HBL-USED']);
    });
    expect(
      await screen.findByText(/分单号 HBL-USED 在该主单批次内已存在/),
    ).toBeVisible();
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-NEW' },
    });
    await act(async () => {
      await expect(form.validateFields()).resolves.toMatchObject({
        seaHouseBill: { houseNo: 'HBL-NEW' },
      });
    });
  });
});
