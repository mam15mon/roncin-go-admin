import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { preview, renderEditor, save, state } from './HblEditor.fixtures';

describe('HBL 签发公司抬头', () => {
  it('使用真实签发公司英文名，忽略当前订单归属公司名称', async () => {
    state.bundle.order.organizationName = '订单归属公司';
    state.bundle.documents.houseBill = {
      ...state.bundle.documents.houseBill,
      issuerOrganizationId: 'different-issuer',
      issuerOrganizationName: '其他自签公司',
      issuerOrganizationNameEn: 'Confirmed Chengdu Test Company Co., Ltd',
    };
    renderEditor();
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].model).toMatchObject({
      issuerNameEn: 'Confirmed Chengdu Test Company Co., Ltd',
      houseNo: 'HBL001',
    });
  });

  it('缺英文名时四版本不可输出，但可修改和保存同一HBL', async () => {
    state.bundle.documents.houseBill = {
      ...state.bundle.documents.houseBill,
      issuerOrganizationNameEn: undefined,
    };
    renderEditor();
    expect(
      screen.getByText('签发公司尚未维护英文名称，请在公司档案中补齐后输出。'),
    ).toBeVisible();
    for (const variant of ['草稿', '正本', '电放', '副本']) {
      fireEvent.click(
        screen.getByText(variant, { selector: '.ant-segmented-item-label' }),
      );
      expect(
        screen.getAllByRole('button', { name: /预览／打印／下载/ })[0],
      ).toBeDisabled();
    }
    const input = screen.getByLabelText('发货人 (Shipper)');
    expect(input).toBeEnabled();
    fireEvent.change(input, { target: { value: '缺名仍可维护正文' } });
    await save();
    expect(state.save.mock.calls[0][1].houseBill.content.shipperText).toBe(
      '缺名仍可维护正文',
    );
    expect(state.preview).not.toHaveBeenCalled();
  });
});
