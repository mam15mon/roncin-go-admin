import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { describe, expect, it } from 'vitest';
import BillDueSettings from './BillDueSettings';
import {
  type BillDueSettingsValue,
  billDuePayload,
  initialBillDueSettings,
} from './billDueSettings';

let form: FormInstance<BillDueSettingsValue>;
function Scene({
  source = {},
  fieldsProtected = false,
  termsRequired = false,
  disabled = false,
}: {
  source?: { paymentTermsDays?: number; dueDate?: string };
  fieldsProtected?: boolean;
  termsRequired?: boolean;
  disabled?: boolean;
}) {
  [form] = Form.useForm();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form
        form={form}
        initialValues={initialBillDueSettings(source)}
        disabled={disabled}
      >
        <BillDueSettings
          billDate="2026-11-18"
          fieldsProtected={fieldsProtected}
          hasPaymentTerms={source.paymentTermsDays != null}
          termsRequired={termsRequired}
        />
      </Form>
    </ConfigProvider>
  );
}

describe('共享到期设置表单', () => {
  it('切换方式与折叠保留原输入，校验并只提交当前方式', async () => {
    render(<Scene source={{ paymentTermsDays: 30 }} />);
    expect(screen.getByText('2026-12-18 到期 · 账期 30 天')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    fireEvent.click(screen.getByRole('radio', { name: '指定日期' }));
    await act(async () => {
      form.setFieldValue('dueDate', dayjs('2026-12-01'));
    });
    fireEvent.click(screen.getByRole('button', { name: '收起到期设置' }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '调整到期日' })).toBeVisible(),
    );
    let value: BillDueSettingsValue | undefined;
    await act(async () => {
      value = await form.validateFields();
    });
    expect(billDuePayload(value ?? {})).toEqual({ dueDate: '2026-12-01' });
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    fireEvent.click(screen.getByRole('radio', { name: '按账期天数' }));
    expect(screen.getByLabelText('账期（天）')).toHaveValue('30');
    expect(form.getFieldValue('dueDate').format('YYYY-MM-DD')).toBe(
      '2026-12-01',
    );
  });

  it('指定日期为空不能收起；早于账单日期不能提交', async () => {
    render(<Scene />);
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    fireEvent.click(screen.getByRole('radio', { name: '指定日期' }));
    fireEvent.click(screen.getByRole('button', { name: '收起到期设置' }));
    expect(await screen.findByText('请选择指定到期日')).toBeVisible();
    expect(screen.getByRole('button', { name: '收起到期设置' })).toBeVisible();
    await act(async () => {
      form.setFieldValue('dueDate', dayjs('2026-11-17'));
    });
    await act(async () => {
      await expect(form.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ errors: ['到期日不能早于创建日期'] }),
        ]),
      });
    });
  });

  it('客户有默认账期时不能清空成未约定', async () => {
    render(<Scene source={{ paymentTermsDays: 30 }} termsRequired />);
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    fireEvent.change(screen.getByLabelText('账期（天）'), {
      target: { value: '' },
    });
    fireEvent.click(screen.getByRole('button', { name: '收起到期设置' }));
    expect(
      await screen.findByText('请填写账期天数，或改为指定到期日'),
    ).toBeVisible();
  });

  it('受保护且有账期时禁用调整；无账期仍可调整日期但不能新增账期', async () => {
    const view = render(
      <Scene source={{ paymentTermsDays: 30 }} fieldsProtected />,
    );
    expect(screen.getByRole('button', { name: '调整到期日' })).toBeDisabled();
    view.unmount();
    render(<Scene source={{ dueDate: '2026-12-01' }} fieldsProtected />);
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    expect(screen.getByLabelText('指定到期日')).toBeEnabled();
    expect(screen.getByRole('radio', { name: '按账期天数' })).toBeDisabled();
    await act(async () => {
      form.setFieldValue('dueDate', dayjs('2026-12-02'));
    });
    let value: BillDueSettingsValue | undefined;
    await act(async () => {
      value = await form.validateFields();
    });
    expect(billDuePayload(value ?? {})).toEqual({ dueDate: '2026-12-02' });
  });

  it('继承父表单冻结状态', () => {
    render(<Scene source={{ paymentTermsDays: 30 }} disabled />);
    expect(screen.getByRole('button', { name: '调整到期日' })).toBeDisabled();
  });
});
