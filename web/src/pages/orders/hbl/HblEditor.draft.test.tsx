import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { getFormDraft, getFormDraftKey } from '@/components/layout/formDraft';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import { preview, renderEditor, save, state } from './HblEditor.fixtures';
import type { HblFormValues } from './model';

const scope = 'user-1:org-1';
const tabKey = '/orders/sea-export/order-1/hbl';
const draftKey = getFormDraftKey(
  tabKey,
  '/orders/sea-export/order-1/hbl',
  scope,
);

describe('HBL 制作草稿与原始版本', () => {
  it('首次运输印字初始化不标脏、不写草稿，直接输出已保存来源', async () => {
    renderEditor();
    expect(screen.getByLabelText('订舱号 (Booking No.)')).toHaveValue(
      'BOOK001',
    );
    expect(getFormDraft(draftKey)).toBeNull();
    expect(isTabDirty(tabKey, scope)).toBe(false);
    await preview();
    expect(state.save).not.toHaveBeenCalled();
    expect(getFormDraft(draftKey)).toBeNull();
  });

  it('首次显式带入实际箱就保存完整草稿，重挂后可恢复箱号和原版本', async () => {
    state.containers.mockResolvedValueOnce({
      success: true,
      data: [{ containerNo: 'BOX001', containerSpecId: '20gp' }],
    });
    const first = renderEditor();
    await act(async () => fireEvent.click(screen.getByText('带入实际箱')));
    await waitFor(() =>
      expect(getFormDraft<HblFormValues>(draftKey)).toMatchObject({
        source: { houseId: 'house-1', houseVersion: '3', linkVersion: '2' },
        printDetails: { containersText: 'BOX001 / 20GP' },
      }),
    );
    expect(isTabDirty(tabKey, scope)).toBe(true);
    first.unmount();
    renderEditor();
    expect(screen.getByLabelText('箱号／封号／箱型')).toHaveValue(
      'BOX001 / 20GP',
    );
  });

  it('旧草稿恢复不会借用最新HBL和关联版本，冲突响应保留输入与草稿', async () => {
    const first = renderEditor();
    fireEvent.change(screen.getByLabelText('发货人 (Shipper)'), {
      target: { value: '旧草稿内容' },
    });
    first.unmount();
    state.bundle = {
      ...state.bundle,
      documents: {
        ...state.bundle.documents,
        linkVersion: '8',
        houseBill: { ...state.bundle.documents.houseBill, version: '9' },
      },
    };
    state.save.mockResolvedValueOnce(undefined);
    renderEditor();
    await save();
    expect(state.save.mock.calls[0][1]).toMatchObject({
      expectedVersion: '3',
      expectedLinkVersion: '2',
      houseBill: { content: { shipperText: '旧草稿内容' } },
    });
    expect(screen.getByLabelText('发货人 (Shipper)')).toHaveValue('旧草稿内容');
    expect(getFormDraft<HblFormValues>(draftKey)?.source).toMatchObject({
      houseVersion: '3',
      linkVersion: '2',
    });
  });
});
