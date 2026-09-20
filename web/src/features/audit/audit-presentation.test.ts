import { describe, expect, it } from 'vitest';
import {
  auditActionPresentation,
  auditActorName,
  auditBusinessObject,
  auditDetailValue,
  isTechnicalAuditKey,
  parseRoleBadges,
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

  it('正确识别企业资源相关操作', () => {
    const create = auditActionPresentation('enterprise_resource.create');
    expect(create.title).toBe('新增企业资源');
    expect(create.category).toBe('企业资源');

    const link = auditActionPresentation(
      'enterprise_resource.partner.batch_link',
    );
    expect(link.title).toBe('批量关联企业');
    expect(link.category).toBe('企业资源');

    const tagGroup = auditActionPresentation('enterprise_tag_group.create');
    expect(tagGroup.title).toBe('创建标签组');
  });

  it('正确解析客商身份角色与技术字段', () => {
    const badges = parseRoleBadges(
      'customer:true,supplier:true,foreign_agent:false',
    );
    expect(badges).toEqual([
      { key: 'customer', label: '客户', enabled: true },
      { key: 'supplier', label: '供应商', enabled: true },
      { key: 'foreign_agent', label: '国外代理', enabled: false },
    ]);

    expect(isTechnicalAuditKey('partner.id')).toBe(true);
    expect(isTechnicalAuditKey('resource_id')).toBe(true);
    expect(isTechnicalAuditKey('trace_id')).toBe(true);
    expect(isTechnicalAuditKey('legal_name')).toBe(false);
    expect(isTechnicalAuditKey('partner.code')).toBe(false);
    expect(isTechnicalAuditKey('roles')).toBe(false);
    expect(auditDetailValue('role_type', 'customer')).toBe('客户');
    expect(auditDetailValue('role_type', 'supplier')).toBe('供应商');
    expect(auditDetailValue('role_type', 'foreign_agent')).toBe('国外代理');
  });
});
