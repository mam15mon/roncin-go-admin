import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { Form, type FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { serializeAirWaybillContent } from '../../../order-kinds/air-export/form-adapter';
import { AirWaybillFields } from './AirWaybillFields';

function setup(content: API.AirWaybillContent = {}, disabled = false) {
  let current!: FormInstance;
  function Editor() {
    const [form] = Form.useForm();
    current = form;
    return (
      <ProForm form={form} initialValues={{ content }} submitter={false}>
        <AirWaybillFields prefix={['content']} disabled={disabled} />
      </ProForm>
    );
  }
  renderWithApp(<Editor />);
  return () => current;
}

describe('航空结构化正文控件', () => {
  it('打开及编辑分节保留未挂载正文，长文与换行不被修剪', async () => {
    const longText = `品名\n${'货'.repeat(9997)}`;
    const readForm = setup({
      shipperText: '第一行\n第二行',
      goodsDescription: longText,
      remarks: ' 原始备注\n',
    });
    fireEvent.click(screen.getByRole('button', { name: '收发货人与代理' }));
    const shipper = await screen.findByRole('textbox', {
      name: '发货人名称与地址',
    });
    expect(shipper).toHaveValue('第一行\n第二行');
    fireEvent.change(shipper, { target: { value: '  新发货人\n地址  ' } });
    const form = readForm();
    expect(form.getFieldValue(['content', 'shipperText'])).toBe(
      '  新发货人\n地址  ',
    );
    expect(form.getFieldValue(['content', 'goodsDescription'])).toBe(longText);
    expect(form.getFieldValue(['content', 'remarks'])).toBe(' 原始备注\n');
    const submitted = await form.validateFields();
    expect(submitted.content.goodsDescription).toBe(longText);
    expect(submitted.content.remarks).toBe(' 原始备注\n');
    fireEvent.click(screen.getByRole('button', { name: '签发与备注' }));
    const date = await screen.findByLabelText('签发日期');
    fireEvent.change(date, { target: { value: '2026-11-16' } });
    expect(form.getFieldValue(['content', 'issueDate'])).toBe('2026-11-16');
  });

  it('声明文本与数值明确区分，切换NVD不夹带原数值', async () => {
    const readForm = setup({ carriageValue: { kind: 'AMOUNT', amount: '0' } });
    fireEvent.click(screen.getByRole('button', { name: '币种、声明值与付款' }));
    expect(
      await screen.findByRole('textbox', { name: '运输声明价值数值' }),
    ).toHaveValue('0');
    fireEvent.mouseDown(screen.getByRole('combobox', { name: '运输声明价值' }));
    const options = await screen.findAllByText('NVD');
    fireEvent.click(options[options.length - 1]);
    expect(
      readForm().getFieldValue(['content', 'carriageValue']),
    ).toMatchObject({ kind: 'NVD' });
    expect(
      readForm().getFieldValue(['content', 'carriageValue', 'amount']),
    ).toBeUndefined();
    await waitFor(() =>
      expect(
        screen.queryByRole('textbox', { name: '运输声明价值数值' }),
      ).toBeNull(),
    );
  });

  it('运价行绑定当前正文路径并保留小数原值，AS ARRANGED独立于行金额', async () => {
    const readForm = setup();
    fireEvent.click(screen.getByRole('button', { name: '运价明细' }));
    fireEvent.click(await screen.findByRole('button', { name: '添加运价行' }));
    fireEvent.change(screen.getByRole('textbox', { name: '运价行金额' }), {
      target: { value: '123.45678901' },
    });
    fireEvent.change(screen.getByRole('textbox', { name: '计费重（KG）' }), {
      target: { value: '0' },
    });
    fireEvent.click(screen.getByRole('checkbox', { name: /AS ARRANGED/ }));
    expect(readForm().getFieldValue(['content', 'rateLines'])).toMatchObject([
      { amount: '123.45678901', chargeableWeightKg: '0' },
    ]);
    expect(readForm().getFieldValue(['content', 'asArranged'])).toBe(true);
    expect(readForm().getFieldValue('rateLines')).toBeUndefined();
    fireEvent.change(screen.getByRole('textbox', { name: '运价行金额' }), {
      target: { value: 'AS ARRANGED' },
    });
    await act(async () => {
      await expect(
        readForm().validateFields([['content', 'rateLines', 0, 'amount']]),
      ).rejects.toMatchObject({
        errorFields: [{ name: ['content', 'rateLines', 0, 'amount'] }],
      });
    });
  });

  it('只打开费用分节不产生虚假费用，禁用正文保留内容且无增删入口', async () => {
    const readForm = setup({ remarks: '只读备注' }, true);
    fireEvent.click(screen.getByRole('button', { name: '其他费用与汇总' }));
    expect(
      await screen.findByRole('textbox', { name: '预付估价费' }),
    ).toBeDisabled();
    expect(screen.queryByRole('button', { name: '添加其他费用' })).toBeNull();
    expect(
      serializeAirWaybillContent(readForm().getFieldValue('content'))?.prepaid,
    ).toBeUndefined();
    fireEvent.click(screen.getByRole('button', { name: '签发与备注' }));
    const remarks = await screen.findByRole('textbox', { name: '单证备注' });
    expect(remarks).toBeDisabled();
    expect(remarks).toHaveValue('只读备注');
  });

  it('其他费用提交保持嵌套路径、明确零与未打开的运价分节', async () => {
    const readForm = setup({
      currency: 'CNY',
      weightPayment: 'PPD',
      rateLines: [{ pieces: '2', amount: '100.12345678' }],
      carriageValue: { kind: 'NVD' },
      otherCharges: [
        {
          name: '处理费',
          currency: 'CNY',
          amount: '0',
          quantity: '3',
          dueTo: 'AGENT',
          payment: 'PPD',
        },
      ],
    });
    fireEvent.click(screen.getByRole('button', { name: '其他费用与汇总' }));
    fireEvent.change(await screen.findByRole('textbox', { name: '数量' }), {
      target: { value: '3.12345678' },
    });
    const submitted = await readForm().validateFields();
    expect(serializeAirWaybillContent(submitted.content)).toMatchObject({
      carriageValue: { kind: 'NVD' },
      rateLines: [{ pieces: '2', amount: '100.12345678' }],
      otherCharges: [{ quantity: '3.12345678', amount: '0', currency: 'CNY' }],
    });
    expect(submitted.otherCharges).toBeUndefined();
  });
});
