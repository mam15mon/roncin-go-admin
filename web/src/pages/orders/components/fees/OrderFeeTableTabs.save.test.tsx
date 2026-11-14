import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { PAYABLE, RECEIVABLE } from './feeConstants';
import {
  listFees,
  makeFee,
  makeProps,
  pickSelectOption,
  renderTabs,
  resetBulkActionMocks,
  saveFees,
} from './OrderFeeTableTabs.bulkActions.fixtures';

/** 真实 wire 形态：非 optional 的零值/默认值省略，数量与单价为字符串。 */
function wireFee(
  id: string,
  direction: number,
  extra: Partial<API.OrderFee> = {},
) {
  return makeFee({
    id,
    direction,
    version: '2',
    canEdit: true,
    canDelete: true,
    canBill: true,
    feeSettingId: 'setting-of',
    settlementPartyId: 'party-old',
    billingUnitId: 'unit-piao',
    billingUnit: '票',
    quantity: '1.0000',
    unitPrice: '100.0000',
    note: `${id}备注`,
    ...extra,
  });
}

const rowOf = (key: string) =>
  document.querySelector(`tr[data-row-key="${key}"]`) as HTMLElement;
const noteInput = (key: string) =>
  within(rowOf(key)).getByPlaceholderText('备注（可选）');

async function mountBoth() {
  listFees.mockResolvedValue({
    data: [
      wireFee('r1', RECEIVABLE),
      wireFee('r2', RECEIVABLE),
      wireFee('p1', PAYABLE),
    ],
  } as never);
  const { props } = makeProps('order-1');
  await renderTabs(props);
  await waitFor(() => expect(rowOf('p1')).not.toBeNull());
  return props;
}

const clickSave = async (side: '应收' | '应付') => {
  await act(async () => {
    fireEvent.click(screen.getByText(`保存${side}费用`));
  });
};

function rejection(
  data: Record<string, unknown> | undefined,
  status = 400,
): Error {
  return Object.assign(new Error('Request failed'), {
    response: data ? { status, data: { code: status, ...data } } : undefined,
  });
}

describe('OrderFeeTableTabs 整表保存', () => {
  beforeEach(resetBulkActionMocks);

  it('两侧都有草稿时保存应收只提交应收改动行，应付草稿完整保留且只刷新本侧', async () => {
    saveFees.mockResolvedValue({
      success: true,
      data: [],
      feeContentVersion: '7',
    } as never);
    const props = await mountBoth();
    const loadsBefore = listFees.mock.calls.length;
    fireEvent.change(noteInput('r1'), { target: { value: '应收新备注' } });
    fireEvent.change(noteInput('p1'), { target: { value: '应付草稿' } });

    await clickSave('应收');

    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    const [params, body] = saveFees.mock.calls[0];
    expect(params).toEqual({ orderId: 'order-1' });
    expect(body).toMatchObject({ orderId: 'order-1', direction: RECEIVABLE });
    // r2 未改动不得提交；已保存行携带 feeId 与版本，不带未更改的税率。
    expect(body.rows).toHaveLength(1);
    expect(body.rows[0]).toMatchObject({
      clientRowId: 'r1',
      feeId: 'r1',
      expectedVersion: '2',
      note: '应收新备注',
      quantity: '1',
      unitPrice: '100',
    });
    expect(body.rows[0]).not.toHaveProperty('taxRate');
    expect(typeof body.requestKey).toBe('string');
    await waitFor(() =>
      expect(listFees.mock.calls.length).toBe(loadsBefore + 1),
    );
    expect(noteInput('p1')).toHaveValue('应付草稿');
    expect(props.onFeeSaved).toHaveBeenCalledWith('7');
  });

  it('保存期间只冻结本侧输入与新增操作，另一侧仍可继续编辑', async () => {
    let resolveSave!: (response: {
      success: boolean;
      data: API.OrderFee[];
    }) => void;
    saveFees.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveSave = resolve;
        }),
    );
    await mountBoth();
    fireEvent.change(noteInput('r1'), { target: { value: '本次提交' } });
    await clickSave('应收');
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(noteInput('r1')).toBeDisabled();
    expect(screen.getByText('新增应收费用').closest('button')).toBeDisabled();
    expect(noteInput('p1')).not.toBeDisabled();
    fireEvent.change(noteInput('p1'), {
      target: { value: '请求期间应付输入' },
    });
    await act(async () => {
      resolveSave({ success: true, data: [] });
    });
    await waitFor(() => expect(noteInput('r1')).not.toBeDisabled());
    expect(noteInput('p1')).toHaveValue('请求期间应付输入');
  });

  it('反向：保存应付只提交应付行，应收草稿保留', async () => {
    saveFees.mockResolvedValue({ success: true, data: [] } as never);
    await mountBoth();
    fireEvent.change(noteInput('r1'), { target: { value: '应收草稿' } });
    fireEvent.change(noteInput('p1'), { target: { value: '应付新备注' } });

    await clickSave('应付');

    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    const body = saveFees.mock.calls[0][1];
    expect(body.direction).toBe(PAYABLE);
    expect(body.rows.map((row) => row.clientRowId)).toEqual(['p1']);
    await waitFor(() => expect(noteInput('r1')).toHaveValue('应收草稿'));
  });

  it('没有改动时不发请求', async () => {
    await mountBoth();
    await clickSave('应收');
    expect(await screen.findByText('本侧费用没有需要保存的修改')).toBeVisible();
    expect(saveFees).not.toHaveBeenCalled();
  });

  it('保存失败保留全部输入，按错误 metadata 定位到行与字段展示', async () => {
    saveFees.mockRejectedValue(
      rejection({
        reason: 'ORDER_FEE_CURRENCY_INVALID',
        message: '币种不合法',
        metadata: {
          client_row_id: 'r1',
          field: 'currency',
          reason: 'ORDER_FEE_CURRENCY_INVALID',
        },
      }),
    );
    await mountBoth();
    fireEvent.change(noteInput('r1'), { target: { value: '保留我' } });
    fireEvent.change(noteInput('p1'), { target: { value: '应付也保留' } });

    await clickSave('应收');

    expect((await screen.findAllByText(/币种不合法/)).length).toBeGreaterThan(
      0,
    );
    expect(screen.getByText('本次保存失败，所有输入已保留')).toBeVisible();
    expect(noteInput('r1')).toHaveValue('保留我');
    expect(noteInput('p1')).toHaveValue('应付也保留');
    // 字段级错误挂到对应行的币种单元格。
    await waitFor(() =>
      expect(
        rowOf('r1').querySelector('.ant-form-item-has-error'),
      ).not.toBeNull(),
    );
  });

  it('多行同时失败时以 row_errors 逐行列出', async () => {
    saveFees.mockRejectedValue(
      rejection(
        {
          reason: 'ORDER_FEE_VERSION_CONFLICT',
          message: '已被更新，请刷新后重试',
          metadata: {
            client_row_id: 'r1',
            field: 'version',
            row_errors: JSON.stringify([
              {
                client_row_id: 'r1',
                field: 'version',
                reason: 'ORDER_FEE_VERSION_CONFLICT',
                message: '已被更新，请刷新后重试',
              },
              {
                client_row_id: 'r2',
                field: 'settlementPartyId',
                reason: 'ORDER_FEE_PARTY_INVALID',
                message: '结算单位无效',
              },
            ]),
          },
        },
        409,
      ),
    );
    await mountBoth();
    fireEvent.change(noteInput('r1'), { target: { value: 'x' } });
    fireEvent.change(noteInput('r2'), { target: { value: 'y' } });
    await clickSave('应收');
    expect(await screen.findByText(/结算单位无效/)).toBeVisible();
    expect(
      screen.getAllByText(/已被更新，请刷新后重试/).length,
    ).toBeGreaterThan(0);
  });

  it('request_key：原样重试沿用，修改内容换新，成功后下一次保存再换新', async () => {
    saveFees.mockRejectedValueOnce(rejection(undefined));
    saveFees.mockRejectedValueOnce(
      rejection({ reason: 'ORDER_FEE_INVALID', message: '失败' }),
    );
    saveFees.mockResolvedValue({ success: true, data: [] } as never);
    await mountBoth();
    fireEvent.change(noteInput('r1'), { target: { value: 'A' } });

    await clickSave('应收'); // 网络中断，结果未知
    expect(await screen.findByText(/保存结果未确认/)).toBeVisible();
    await clickSave('应收'); // 原样重试
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(2));
    expect(saveFees.mock.calls[1][1].requestKey).toBe(
      saveFees.mock.calls[0][1].requestKey,
    );

    fireEvent.change(noteInput('r1'), { target: { value: 'B' } }); // 修正输入
    await clickSave('应收');
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(3));
    const keyAfterEdit = saveFees.mock.calls[2][1].requestKey;
    expect(keyAfterEdit).not.toBe(saveFees.mock.calls[0][1].requestKey);

    // 成功之后即使再次得到同样内容，也必须是新的保存意图。
    fireEvent.change(noteInput('r1'), { target: { value: 'C' } });
    await clickSave('应收');
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(4));
    expect(saveFees.mock.calls[3][1].requestKey).not.toBe(keyAfterEdit);
  });

  it('新增行缺少必填项时不发请求；补齐后以 client_row_id 提交且不带 feeId', async () => {
    saveFees.mockResolvedValue({ success: true, data: [] } as never);
    listFees.mockResolvedValue({ data: [wireFee('r1', RECEIVABLE)] } as never);
    const { props } = makeProps('order-1');
    // 新增行默认带入客户为结算单位。
    await renderTabs({
      ...props,
      order: { id: 'order-1', customerId: 'party-old' } as API.Order,
      customerName: '原结算单位',
    });
    await waitFor(() => expect(rowOf('r1')).not.toBeNull());
    await act(async () => {
      fireEvent.click(screen.getByText('新增应收费用'));
    });
    await waitFor(() =>
      expect(document.querySelector('tr[data-row-key^="new_"]')).not.toBeNull(),
    );
    await clickSave('应收');
    expect(saveFees).not.toHaveBeenCalled();

    const newRow = document.querySelector(
      'tr[data-row-key^="new_"]',
    ) as HTMLElement;
    await pickSelectOption(
      within(newRow).getAllByRole('combobox')[0],
      '海运费 (OF)',
    );
    fireEvent.change(within(newRow).getByPlaceholderText('0.00'), {
      target: { value: '88.5' },
    });
    await clickSave('应收');
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    const row = saveFees.mock.calls[0][1].rows[0];
    expect(row.clientRowId).toMatch(/^new_/);
    expect(row).not.toHaveProperty('feeId');
    expect(row).not.toHaveProperty('expectedVersion');
    expect(row).toMatchObject({
      unitPrice: '88.5',
      feeSettingId: 'setting-of',
    });
    expect(saveFees.mock.calls[0][1].rows).toHaveLength(1);
  });

  it('已建账行保持只读且不提交，即使其他行修改后保存', async () => {
    listFees.mockResolvedValue({
      data: [
        wireFee('r1', RECEIVABLE),
        wireFee('billed', RECEIVABLE, {
          canEdit: false,
          canDelete: false,
          hasActiveBill: true,
        }),
      ],
    } as never);
    saveFees.mockResolvedValue({ success: true, data: [] } as never);
    const { props } = makeProps('order-1');
    await renderTabs(props);
    await waitFor(() => expect(rowOf('billed')).not.toBeNull());
    expect(
      within(rowOf('billed')).queryByPlaceholderText('备注（可选）'),
    ).toBeNull();
    fireEvent.change(noteInput('r1'), { target: { value: '只改我' } });
    await clickSave('应收');
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(saveFees.mock.calls[0][1].rows.map((r) => r.clientRowId)).toEqual([
      'r1',
    ]);
  });

  it('仅有新增权限（无修改权限）时整表只读：不可新增也不可保存', async () => {
    listFees.mockResolvedValue({ data: [wireFee('r1', RECEIVABLE)] } as never);
    const { props } = makeProps('order-1');
    props.canUpdateFees = false;
    await renderTabs(props);
    await waitFor(() => expect(rowOf('r1')).not.toBeNull());
    expect(
      within(rowOf('r1')).queryByPlaceholderText('备注（可选）'),
    ).toBeNull();
    expect(screen.getByText('保存应收费用').closest('button')).toBeDisabled();
    expect(screen.getByText('新增应收费用').closest('button')).toBeDisabled();
  });
});
