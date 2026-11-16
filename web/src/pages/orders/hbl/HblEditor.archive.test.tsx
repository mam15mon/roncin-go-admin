import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OrderBusinessType, SeaHouseBillIssuerSource } from '@/enums.generated';
import { preview, renderEditor, state } from './HblEditor.fixtures';

function externalArchive() {
  state.bundle = {
    ...state.bundle,
    order: {
      ...state.bundle.order,
      businessType: OrderBusinessType.BUSINESS_TYPE_SI,
    },
    documents: {
      ...state.bundle.documents,
      houseBill: {
        ...state.bundle.documents.houseBill,
        issuerSource:
          SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER,
        issuerPartnerId: 'external-issuer',
        issuerPartnerName: '真实境外签发人',
        issuerOrganizationId: undefined,
        issuerOrganizationName: undefined,
        issuerOrganizationNameEn: undefined,
        printTemplate: 'EXTERNAL_ARCHIVE',
      },
    },
  };
}

describe('SI 外部 HBL 制作归档', () => {
  it('真实外部主体可维护和保存同一HBL，归档不提供本公司四类签发输出', async () => {
    externalArchive();
    renderEditor();
    expect(screen.queryByLabelText('HBL 输出版本')).toBeNull();
    expect(screen.getByText(/真实境外签发人.*外部提单登记资料/)).toBeVisible();
    fireEvent.change(screen.getByLabelText('发货人 (Shipper)'), {
      target: { value: '境外原单发货人' },
    });
    fireEvent.click(screen.getAllByRole('button', { name: /保存制作资料/ })[0]);
    await waitFor(() => expect(state.save).toHaveBeenCalledOnce());
    expect(state.save.mock.calls[0][1]).toMatchObject({
      id: 'house-1',
      expectedVersion: '3',
      expectedLinkVersion: '2',
      houseBill: {
        issuerPartnerId: 'external-issuer',
        content: { shipperText: '境外原单发货人' },
      },
    });
    await waitFor(() =>
      expect(
        screen.getAllByRole('button', { name: /保存制作资料/ })[0],
      ).toBeEnabled(),
    );
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].model).toMatchObject({
      outputKind: 'external-archive',
      issuerDisplayName: '真实境外签发人',
      savedVersion: '4',
      content: { shipperText: '境外原单发货人' },
    });
  });

  it('已审核仍可输出已保存归档且不借用公司英文名称', async () => {
    externalArchive();
    state.locked = true;
    renderEditor();
    expect(
      screen.getAllByRole('button', { name: /保存制作资料/ })[0],
    ).toBeDisabled();
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].model.issuerNameEn).toBe('');
    expect(state.preview.mock.calls.at(-1)?.[0].model.issuerDisplayName).toBe(
      '真实境外签发人',
    );
  });
});
