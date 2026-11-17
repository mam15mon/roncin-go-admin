import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { revealSeaFormErrors } from '../../sea-form-error-reveal';
import { renderAirOrderContent } from './AirOrderWaybillFields.fixtures';

describe('航空订单正文常用布局', () => {
  it('通知人切换保留两份文本和显式清空，常用抬头四行货描五行', async () => {
    const view = renderAirOrderContent({
      notifyPartyText: '通知人A',
      secondNotifyPartyText: '通知人B',
    });
    expect(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
    ).toHaveAttribute('rows', '4');
    expect(screen.getByRole('textbox', { name: '运单品名' })).toHaveAttribute(
      'rows',
      '5',
    );
    expect(screen.queryByRole('textbox', { name: '第二通知人' })).toBeNull();
    fireEvent.click(screen.getByText('第二通知人 (Second Notify Party) · 已填写'));
    fireEvent.change(screen.getByRole('textbox', { name: '第二通知人' }), {
      target: { value: '' },
    });
    fireEvent.click(screen.getByText('通知人 (Notify Party)', { exact: true }));
    expect(screen.getByRole('textbox', { name: '通知人' })).toHaveValue(
      '通知人A',
    );
    await act(async () => {
      const values = await view.readForm().validateFields();
      expect(values.content).toMatchObject({
        notifyPartyText: '通知人A',
        secondNotifyPartyText: '',
      });
    });
  });

  it('低频错误展开两层更多资料，通知人错误选择对应控件', async () => {
    renderAirOrderContent({ currency: 'CNY', secondNotifyPartyText: '第二人' });
    expect(screen.queryByRole('textbox', { name: '运单币种' })).toBeNull();
    act(() => revealSeaFormErrors([{ name: ['content', 'currency'] }]));
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: '运单币种' })).toBeVisible(),
    );
    act(() =>
      revealSeaFormErrors([{ name: ['content', 'secondNotifyPartyText'] }]),
    );
    expect(screen.getByRole('textbox', { name: '第二通知人' })).toBeVisible();
    expect(screen.getByRole('textbox', { name: '第二通知人' })).toHaveValue(
      '第二人',
    );
  });

  it('程序带入超长通知人也会校验失败并定位，不截断已保存长文', async () => {
    const text = '通'.repeat(5001);
    const view = renderAirOrderContent({ secondNotifyPartyText: text });
    await act(async () => {
      await expect(
        view.readForm().validateFields([['content', 'secondNotifyPartyText']]),
      ).rejects.toMatchObject({
        errorFields: [{ name: ['content', 'secondNotifyPartyText'] }],
      });
      revealSeaFormErrors([{ name: ['content', 'secondNotifyPartyText'] }]);
    });
    expect(screen.getByRole('textbox', { name: '第二通知人' })).toHaveValue(
      text,
    );
    expect(
      view.readForm().getFieldValue(['content', 'secondNotifyPartyText']),
    ).toHaveLength(5001);
  });
});
