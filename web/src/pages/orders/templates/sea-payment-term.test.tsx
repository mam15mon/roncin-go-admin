import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, within } from '@testing-library/react';
import { ConfigProvider, type FormInstance } from 'antd';
import { expect, it } from 'vitest';
import { PaymentTerm } from '@/enums.generated';
import { SectionCards } from './sea-create-layout.fixtures';

it('共用海运模板仅注册一个可编辑业务付款方式，切换不改正式单证运费条款', async () => {
  let form!: FormInstance;
  const view = render(
    <ConfigProvider virtual={false}>
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['cargoCommercial']}
        initialValues={{
          paymentTerm: PaymentTerm.PAYMENT_TERM_PREPAID,
          seaMasterBillContent: { freightTerms: 'EXTERNAL ORIGINAL TEXT' },
        }}
      />
    </ConfigProvider>,
  );
  const select = screen.getByLabelText('业务付款方式');
  expect(view.container.querySelectorAll('#paymentTerm')).toHaveLength(1);
  fireEvent.mouseDown(select);
  const popup = document.querySelector('.ant-select-dropdown') as HTMLElement;
  fireEvent.click(within(popup).getByText('到付 (CC)'));
  await act(async () => {});
  expect(form.getFieldValue('paymentTerm')).toBe(
    PaymentTerm.PAYMENT_TERM_COLLECT,
  );
  expect(form.getFieldValue(['seaMasterBillContent', 'freightTerms'])).toBe(
    'EXTERNAL ORIGINAL TEXT',
  );
});
