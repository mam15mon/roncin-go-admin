import { describe, expect, it } from 'vitest';
import {
  auditActionPresentation,
  auditActorName,
  auditBusinessObject,
} from './audit-presentation';

describe('审计日志业务化展示', () => {
  it('将登录记录展示为当前账号操作', () => {
    const record = {
      action: 'auth.login',
      actorDisplayName: 'admin',
    } as API.AdminAuditLog;

    expect(auditActionPresentation(record.action).title).toBe('登录系统');
    expect(auditActorName(record)).toBe('admin');
    expect(auditBusinessObject(record)).toEqual({ name: '当前账号' });
  });

  it('展示管理员授权的具体人员', () => {
    const record = {
      action: 'admin.user.dingtalk.authorize',
      actorDisplayName: 'admin',
      targetDisplayName: '张冠楠',
    } as API.AdminAuditLog;

    expect(auditActionPresentation(record.action).title).toBe(
      '完成钉钉账号授权',
    );
    expect(auditBusinessObject(record)).toEqual({
      name: '张冠楠',
      type: '人员',
    });
  });

  it('优先展示业务编号', () => {
    const record = {
      action: 'order.update',
      details: { 'order.no': 'RNC-20260829-001' },
    } as API.AdminAuditLog;

    expect(auditBusinessObject(record)).toEqual({
      name: 'RNC-20260829-001',
      type: '订单',
    });
  });

  it('明确标识未知动作而不猜测含义', () => {
    expect(auditActionPresentation('example.unknown')).toEqual({
      title: '未识别的系统操作',
      category: '其他',
      color: 'default',
    });
  });
});
