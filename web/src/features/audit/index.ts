/**
 * 审计日志业务化展示能力公开入口：管理中心审计页与往来单位审计区块共用。
 * 只公开当前消费方实际使用的转换方法与类型。
 */
export {
  auditActionPresentation,
  auditActorName,
  auditBusinessObject,
  auditDetailLabel,
  auditDetailValue,
  isTechnicalAuditKey,
  parseRoleBadges,
} from './audit-presentation';
