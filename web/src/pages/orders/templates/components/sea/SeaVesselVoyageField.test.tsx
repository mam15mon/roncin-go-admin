import { ProForm } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Form, type FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { props } from '../../sea-create-layout.fixtures';
import { getSeaTransportFields } from './SeaTransportSection';

function setup(initialValues: Record<string, unknown> = {}) {
  let form!: FormInstance;
  function TestForm() {
    [form] = Form.useForm();
    return (
      <ProForm form={form} initialValues={initialValues} submitter={false}>
        {getSeaTransportFields(props).vessel}
      </ProForm>
    );
  }
  render(<TestForm />);
  return { form };
}

describe('船在哪儿直接查询', () => {
  it.each([
    ['UNI-PROSPER', 'UNI-PROSPER'],
    ['EVER GIVEN', 'EVER GIVEN'],
    ['  EVER GIVEN  ', 'EVER GIVEN'],
    ['远洋 + 和平', '远洋 + 和平'],
  ])('按当前输入 %s 查询船名 %s', (inputName, vesselName) => {
    setup({ vesselName: inputName, voyageNo: '001W' });
    const link = screen.getByRole('link', { name: '船在哪儿' });
    expect(link).toHaveAttribute(
      'href',
      `https://www.shipxy.com/?kw=${encodeURIComponent(vesselName)}`,
    );
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    expect(link).not.toHaveAttribute('tabindex', '-1');
  });

  it('修改后查询新输入，只有空格时打开首页手动查询', async () => {
    const { form } = setup({ vesselName: 'UNI-PROSPER', voyageNo: '001W' });
    const input = screen.getByLabelText('船名');
    fireEvent.change(input, { target: { value: 'EVER GIVEN' } });
    await waitFor(() =>
      expect(screen.getByRole('link', { name: '船在哪儿' })).toHaveAttribute(
        'href',
        'https://www.shipxy.com/?kw=EVER%20GIVEN',
      ),
    );
    expect(screen.getByLabelText('航次')).toHaveValue('001W');
    fireEvent.change(screen.getByLabelText('航次'), {
      target: { value: '002E' },
    });
    expect(form.getFieldValue('vesselName')).toBe('EVER GIVEN');
    expect(screen.getByRole('link', { name: '船在哪儿' })).toHaveAttribute(
      'href',
      'https://www.shipxy.com/?kw=EVER%20GIVEN',
    );
    fireEvent.change(input, { target: { value: '   ' } });
    await waitFor(() =>
      expect(screen.getByRole('link', { name: '船在哪儿' })).toHaveAttribute(
        'href',
        'https://www.shipxy.com/',
      ),
    );
    expect(form.getFieldValue('voyageNo')).toBe('002E');
  });

  it('尚未填写船名时仍可打开首页手动查询', () => {
    setup();
    expect(screen.getByRole('link', { name: '船在哪儿' })).toHaveAttribute(
      'href',
      'https://www.shipxy.com/',
    );
  });

  it('共享航程船名只读时仍可查询，表单回填后更新链接', async () => {
    const { form } = setup({
      vesselName: 'UNI-PROSPER',
      voyageNo: '001W',
      shippingLineId: 'line-1',
      seaMasterBillMasterNo: 'MBL123',
      seaMasterBillSharedVoyage: {
        identity: 'line-1:MBL123',
        voyage: { vesselName: 'UNI-PROSPER', voyageNo: '001W' },
      },
    });
    expect(screen.getByLabelText('船名')).toBeDisabled();
    expect(screen.getByLabelText('航次')).toBeDisabled();
    expect(screen.getByRole('link', { name: '船在哪儿' })).toHaveAttribute(
      'href',
      'https://www.shipxy.com/?kw=UNI-PROSPER',
    );
    await act(async () => {
      form.setFieldsValue({ vesselName: 'EVER GIVEN', voyageNo: '002E' });
    });
    expect(screen.getByRole('link', { name: '船在哪儿' })).toHaveAttribute(
      'href',
      'https://www.shipxy.com/?kw=EVER%20GIVEN',
    );
    expect(form.getFieldValue('vesselName')).toBe('EVER GIVEN');
    expect(form.getFieldValue('voyageNo')).toBe('002E');
  });
});
