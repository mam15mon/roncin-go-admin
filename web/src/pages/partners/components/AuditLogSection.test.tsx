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

  it('将操作记录呈现为面向业务人员的友好格式并隐藏技术 UUID', async () => {
    listAuditLogs.mockResolvedValueOnce({
      data: [
        {
          id: 'log-1',
          action: 'partner.create',
          userDisplayName: '系统管理员',
          result: 'success',
          createdAt: '2026-09-14T01:26:16Z',
          details: {
            legal_name: '测试客商',
            'partner.code': 'P8PSJDZ83',
            'partner.id': '01a09bce-3165-75b8-9f75-36a7dcff0c3c',
            roles: 'customer:true',
          },
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
    });

    render(<AuditLogSection partnerId="partner-1" />);

    // 应该显示业务化操作标题与分类，而不是原始英文 partner.create
    expect(await screen.findByText('新增往来单位')).toBeInTheDocument();
    expect(screen.getByText('往来单位')).toBeInTheDocument();
    expect(screen.getByText('系统管理员')).toBeInTheDocument();

    // 应该展示中文业务键值，并将身份角色解析为标签
    expect(screen.getByText('单位名称:')).toBeInTheDocument();
    expect(screen.getByText('测试客商')).toBeInTheDocument();
    expect(screen.getByText('单位编号:')).toBeInTheDocument();
    expect(screen.getByText('P8PSJDZ83')).toBeInTheDocument();
    expect(screen.getByText('身份角色:')).toBeInTheDocument();
    expect(screen.getByText('客户')).toBeInTheDocument();

    // 默认业务卡片内不应该出现裸露的技术标签 "单位 ID:"
    expect(screen.queryByText('单位 ID:')).not.toBeInTheDocument();

    // 应该提供折叠的技术明细
    expect(screen.getByText('技术明细')).toBeInTheDocument();
  });
});
