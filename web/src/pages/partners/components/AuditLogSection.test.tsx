import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import AuditLogSection from './AuditLogSection';

const listAuditLogs = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartnerAuditLogs: listAuditLogs,
}));

describe('AuditLogSection', () => {
  it('加载失败时显示明确错误而不是空日志', async () => {
    listAuditLogs.mockRejectedValueOnce(new Error('审计服务不可用'));

    render(<AuditLogSection partnerId="partner-1" />);

    expect(await screen.findByText('操作日志加载失败')).toBeInTheDocument();
    expect(screen.getByText('审计服务不可用')).toBeInTheDocument();
    expect(screen.queryByText('暂无操作记录流水')).not.toBeInTheDocument();
  });
});
