import { ClockCircleOutlined, UserOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Empty,
  Pagination,
  Space,
  Spin,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import React, { useState } from 'react';
import {
  auditActionPresentation,
  auditDetailLabel,
  isTechnicalAuditKey,
  parseRoleBadges,
} from '@/features/audit';
import { partnerServiceListPartnerAuditLogs } from '@/services/roncin/partnerService';
import { ensureListResponse, unwrapPage } from '@/utils/api';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

function renderDetailValue(key: string, value: unknown): React.ReactNode {
  if (value === undefined || value === null) return '—';

  // 角色列表格式化为中文标签
  if (
    [
      'roles',
      'from_roles',
      'to_roles',
      'roles_added',
      'roles_removed',
    ].includes(key)
  ) {
    const badges = parseRoleBadges(String(value));
    if (badges.length > 0) {
      return (
        <Space size={4} wrap>
          {badges.map((b) => (
            <Tag
              key={b.key}
              color={b.enabled ? 'blue' : 'default'}
              style={{ margin: 0, fontSize: 11, borderRadius: 3 }}
            >
              {b.label}
              {!b.enabled && '（已停用）'}
            </Tag>
          ))}
        </Space>
      );
    }
  }

  // 黑名单状态
  if (key === 'blacklisted') {
    const isBlack = String(value) === 'true';
    return (
      <Tag
        color={isBlack ? 'volcano' : 'green'}
        style={{ margin: 0, fontSize: 11, borderRadius: 3 }}
      >
        {isBlack ? '已列入黑名单' : '正常（未列入黑名单）'}
      </Tag>
    );
  }

  if (key === 'role_type') {
    const labels: Record<string, string> = {
      customer: '客户',
      supplier: '供应商',
      foreign_agent: '国外代理',
    };
    return labels[String(value)] ?? String(value);
  }

  // 启用状态
  if (key === 'enabled') {
    const isEnabled = String(value) === 'true';
    return (
      <Tag
        color={isEnabled ? 'success' : 'default'}
        style={{ margin: 0, fontSize: 11, borderRadius: 3 }}
      >
        {isEnabled ? '启用' : '停用'}
      </Tag>
    );
  }

  // 客户类型（正式/散客）
  if (key === 'is_casual') {
    const isCasual = String(value) === 'true';
    return (
      <Tag
        color={isCasual ? 'warning' : 'blue'}
        style={{ margin: 0, fontSize: 11, borderRadius: 3 }}
      >
        {isCasual ? '散客' : '签约正式'}
      </Tag>
    );
  }

  return (
    <span
      style={{
        color: '#1e293b',
        fontWeight: 500,
        wordBreak: 'break-all',
      }}
    >
      {String(value)}
    </span>
  );
}

interface AuditLogSectionProps {
  partnerId?: string;
  roleLabel?: string;
}

export default function AuditLogSection({
  partnerId,
  roleLabel,
}: AuditLogSectionProps) {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const auditQuery = useQuery({
    queryKey: ['partners', 'audit-logs', { partnerId, page, pageSize }],
    enabled: Boolean(partnerId),
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      if (!partnerId) throw new Error('缺少往来单位标识');
      const response = await partnerServiceListPartnerAuditLogs(
        { partnerId, page, pageSize },
        { signal, skipErrorHandler: true },
      );
      return unwrapPage(ensureListResponse(response, '操作日志加载失败'));
    },
  });
  const logs = auditQuery.data?.data ?? [];
  const total = auditQuery.data?.total ?? 0;
  const loading = auditQuery.isFetching;
  const loadError = auditQuery.error?.message;

  if (!partnerId) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={`保存${roleLabel || '企业'}档案后将自动记录操作与修改日志`}
        style={{ padding: '16px 0' }}
      />
    );
  }

  return (
    <Spin spinning={loading}>
      {loadError ? (
        <Alert
          type="error"
          showIcon
          title="操作日志加载失败"
          description={loadError === '操作日志加载失败' ? undefined : loadError}
          action={
            <Button size="small" onClick={() => void auditQuery.refetch()}>
              重试
            </Button>
          }
        />
      ) : logs.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="暂无操作记录流水"
          style={{ padding: '16px 0' }}
        />
      ) : (
        <div>
          <Timeline
            style={{ marginTop: 12 }}
            items={logs.map((log) => {
              const presentation = auditActionPresentation(log.action);
              const actionTitle =
                presentation.title !== '未识别的系统操作'
                  ? presentation.title
                  : log.action || '业务操作';

              const allEntries = Object.entries(log.details ?? {});
              const businessEntries = allEntries.filter(
                ([key, val]) =>
                  !isTechnicalAuditKey(key) &&
                  val !== undefined &&
                  val !== null &&
                  String(val).trim() !== '',
              );
              const hasDetails = allEntries.length > 0;
              const isSuccess = log.result === 'success';

              const itemContent = (
                <div key={log.id} style={{ fontSize: 13, marginBottom: 12 }}>
                  <Space size={8} wrap style={{ marginBottom: 4 }}>
                    <Text strong style={{ color: '#0f172a', fontSize: 13 }}>
                      {actionTitle}
                    </Text>
                    <Tag
                      color={presentation.color}
                      style={{
                        margin: 0,
                        fontSize: 11,
                        lineHeight: '18px',
                        borderRadius: 3,
                      }}
                    >
                      {presentation.category}
                    </Tag>
                    <Tag
                      color="geekblue"
                      style={{
                        margin: 0,
                        fontSize: 11,
                        lineHeight: '18px',
                        borderRadius: 3,
                      }}
                    >
                      <UserOutlined style={{ marginRight: 4 }} />
                      {log.userDisplayName || log.userId || '系统操作员'}
                    </Tag>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      <ClockCircleOutlined style={{ marginRight: 4 }} />
                      {formatDate(log.createdAt)}
                    </Text>
                    {!isSuccess && (
                      <Tag
                        color="error"
                        style={{ margin: 0, fontSize: 11, borderRadius: 3 }}
                      >
                        操作失败
                      </Tag>
                    )}
                  </Space>

                  {businessEntries.length > 0 && (
                    <div
                      style={{
                        backgroundColor: '#f8fafc',
                        border: '1px solid #e2e8f0',
                        borderRadius: 6,
                        padding: '8px 12px',
                        marginTop: 6,
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          flexWrap: 'wrap',
                          gap: '6px 16px',
                          alignItems: 'center',
                        }}
                      >
                        {businessEntries.map(([key, val]) => (
                          <div
                            key={key}
                            style={{
                              display: 'inline-flex',
                              alignItems: 'center',
                              gap: 4,
                              fontSize: 12,
                            }}
                          >
                            <span style={{ color: '#64748b' }}>
                              {auditDetailLabel(key)}:
                            </span>
                            {renderDetailValue(key, val)}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {hasDetails && (
                    <details
                      style={{
                        marginTop: 4,
                        fontSize: 11,
                        color: '#94a3b8',
                      }}
                    >
                      <summary
                        style={{
                          cursor: 'pointer',
                          userSelect: 'none',
                          display: 'inline-flex',
                          alignItems: 'center',
                          padding: '2px 0',
                        }}
                      >
                        技术明细
                      </summary>
                      <div
                        style={{
                          backgroundColor: '#f1f5f9',
                          border: '1px solid #e2e8f0',
                          padding: '6px 10px',
                          borderRadius: 4,
                          marginTop: 4,
                          fontFamily: 'monospace',
                          color: '#475569',
                          maxHeight: 140,
                          overflowY: 'auto',
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-all',
                        }}
                      >
                        {JSON.stringify(log.details, null, 2)}
                      </div>
                    </details>
                  )}
                </div>
              );

              return {
                key: log.id,
                color: isSuccess ? 'blue' : 'red',
                content: itemContent,
              };
            })}
          />

          {total > pageSize && (
            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                marginTop: 8,
              }}
            >
              <Pagination
                current={page}
                pageSize={pageSize}
                total={total}
                size="small"
                onChange={(p, ps) => {
                  setPage(p);
                  setPageSize(ps);
                }}
              />
            </div>
          )}
        </div>
      )}
    </Spin>
  );
}
