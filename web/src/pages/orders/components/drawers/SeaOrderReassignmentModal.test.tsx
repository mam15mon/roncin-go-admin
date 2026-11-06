import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  candidate,
  enterDate,
  enterMasterNo,
  execute,
  fillConfirmation,
  matchCandidate,
  openExecutionConfirmation,
  outerConfirmButton,
  preview,
  renderReassignment,
  setupReassignmentTests,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();

describe('SeaOrderReassignmentModal', () => {
  it('打开改配弹窗并展示母单变更对比差异项', async () => {
    renderReassignment();
    expect(outerConfirmButton()).toBeDisabled();
    expect(preview).not.toHaveBeenCalled();
    await enterMasterNo();

    await waitFor(() => {
      expect(screen.getByText(/订单整票改配/)).toBeInTheDocument();
      expect(screen.getByText('手工录入新母单')).toBeInTheDocument();
      expect(screen.getByText('匹配已有共享母单')).toBeInTheDocument();
      expect(screen.getByText('改配原因说明')).toBeInTheDocument();
      expect(screen.getByText('责任归属类型')).toBeInTheDocument();
      expect(screen.getByText('外部确认方')).toBeInTheDocument();
      expect(screen.getByText('确认时间')).toBeInTheDocument();
      expect(screen.getByText('确认说明')).toBeInTheDocument();
      expect(screen.getByText('船公司')).toBeInTheDocument();
      expect(screen.queryByText('发单人 / 船代')).not.toBeInTheDocument();
      expect(screen.queryByText('承运人 / 船东')).not.toBeInTheDocument();
    });
    await waitFor(() => expect(screen.getByText('OLDMBL123')).toBeVisible());
    expect(screen.getByText('NEWMBL456')).toBeVisible();
  });

  it('真实母单输入与 RFC3339 航程进入预览，并以相同目标和版本完成 NEW 改配', async () => {
    const { props } = renderReassignment();
    await enterMasterNo();
    fireEvent.change(screen.getByPlaceholderText('船名'), {
      target: { value: 'NEW VESSEL' },
    });
    fireEvent.change(screen.getByPlaceholderText('航次'), {
      target: { value: 'VOYAGE001' },
    });
    enterDate('预计离港时间 (ETD)', '2026-11-10 12:30:00');
    enterDate('预计到港时间 (ETA)', '2026-11-15 13:30:00');
    await fillConfirmation();
    const expectedTarget = {
      targetType: 'NEW',
      masterNo: 'NEWMBL456',
      shippingLineId: 'carrier-1',
      vesselName: 'NEW VESSEL',
      voyageNo: 'VOYAGE001',
      etd: dayjs('2026-11-10 12:30:00').toISOString(),
      eta: dayjs('2026-11-15 13:30:00').toISOString(),
    };
    await waitFor(() =>
      expect(preview).toHaveBeenLastCalledWith(
        { orderId: 'order-123' },
        {
          orderId: 'order-123',
          target: expect.objectContaining(expectedTarget),
        },
        expect.objectContaining({ skipErrorHandler: true }),
      ),
    );
    const confirmation = await openExecutionConfirmation();
    expect(screen.getByPlaceholderText('请输入目标提单号')).toBeDisabled();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(execute).toHaveBeenCalledOnce());
    expect(execute).toHaveBeenCalledWith(
      { orderId: 'order-123' },
      expect.objectContaining({
        target: expect.objectContaining(expectedTarget),
        expectedOrderVersion: '7',
        expectedLinkVersion: '4',
        reason: '客户要求改配',
        confirmation: expect.objectContaining({ confirmedByParty: '测试船代' }),
      }),
      expect.objectContaining({ skipErrorHandler: true }),
    );
    await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce());
    expect(props.onClose).toHaveBeenCalledOnce();
  });

  it('明确匹配候选母单后使用其运输执行和版本完成 CANDIDATE 改配', async () => {
    matchCandidate.mockResolvedValue({
      success: true,
      matched: true,
      candidate,
    });
    renderReassignment();
    await enterMasterNo('EXISTING123');
    fireEvent.click(screen.getByRole('button', { name: '检查/匹配' }));
    await waitFor(() =>
      expect(screen.getByText('已匹配现有共享母单：EXISTING123')).toBeVisible(),
    );
    await waitFor(() =>
      expect(screen.getByPlaceholderText('船名')).toHaveValue(
        'CANDIDATE VESSEL',
      ),
    );
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(execute).toHaveBeenCalledOnce());
    const payload = execute.mock.calls[0][1];
    expect(payload).toMatchObject({
      expectedCandidateMblVersion: '8',
      expectedCandidateTeVersion: '9',
      target: {
        targetType: 'CANDIDATE',
        candidateId: 'candidate-mbl',
        candidateVersion: '8',
        candidateTeId: 'candidate-te',
        candidateTeVersion: '9',
        etd: candidate.transportExecutions?.[0].etd,
        eta: candidate.transportExecutions?.[0].eta,
      },
    });
    expect(preview.mock.calls.at(-1)?.[1].target).toEqual(payload.target);
  });
});
