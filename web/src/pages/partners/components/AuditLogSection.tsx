import { ClockCircleOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Button, Empty, Pagination, Space, Spin, Tag, Timeline, Typography } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { partnerServiceListPartnerAuditLogs } from '@/services/roncin/partnerService';
import { unwrapPage } from '@/utils/api';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

interface AuditLogSectionProps {
  partnerId?: string;
}

export default function AuditLogSection({ partnerId }: AuditLogSectionProps) {
  const [loading, setLoading] = useState(false);
  const [logs, setLogs] = useState<API.PartnerAuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loadError, setLoadError] = useState('');
  const requestSequenceRef = useRef(0);

  const fetchLogs = useCallback(async (currentPage = 1, size = 10) => {
    if (!partnerId) return;
    const requestSequence = ++requestSequenceRef.current;
    setLoading(true);
    setLoadError('');
    try {
      const res = await partnerServiceListPartnerAuditLogs({
        partnerId,
        page: currentPage,
        pageSize: size,
      });
      if (requestSequence !== requestSequenceRef.current) return;
      const result = unwrapPage(res);
      setLogs(result.data);
      setTotal(result.total);
    } catch (error) {
      if (requestSequence !== requestSequenceRef.current) return;
      setLogs([]);
      setTotal(0);
      setLoadError((error as Error).message || '操作日志加载失败');
    } finally {
      if (requestSequence === requestSequenceRef.current) {
        setLoading(false);
      }
    }
  }, [partnerId]);

  useEffect(() => {
    void fetchLogs(page, pageSize);
    return () => {
      requestSequenceRef.current += 1;
    };
  }, [fetchLogs, page, pageSize]);

  if (!partnerId) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description="保存客户档案后将自动记录操作与修改日志"
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
          description={loadError}
          action={<Button size="small" onClick={() => void fetchLogs(page, pageSize)}>重试</Button>}
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
              const detailString =
                log.details && Object.keys(log.details).length > 0
                  ? JSON.stringify(log.details, null, 2)
                  : '';
              return {
                key: log.id,
                color: log.result === 'success' ? 'blue' : 'red',
                children: (
                  <div style={{ fontSize: 13, marginBottom: 8 }}>
                    <Space size={8} wrap>
                      <Text strong style={{ color: '#262626' }}>
                        {log.action || '业务操作'}
                      </Text>
                      <Tag color="geekblue" style={{ margin: 0, fontSize: 11 }}>
                        <UserOutlined style={{ marginRight: 4 }} />
                        {log.userDisplayName || log.userId || '系统操作员'}
                      </Tag>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        <ClockCircleOutlined style={{ marginRight: 4 }} />
                        {formatDate(log.createdAt)}
                      </Text>
                    </Space>
                    {detailString && (
                      <div
                        style={{
                          backgroundColor: '#f5f5f5',
                          padding: '4px 8px',
                          borderRadius: 4,
                          marginTop: 4,
                          fontSize: 11,
                          fontFamily: 'monospace',
                          color: '#666',
                          maxHeight: 120,
                          overflowY: 'auto',
                          whiteSpace: 'pre-wrap',
                        }}
                      >
                        {detailString}
                      </div>
                    )}
                  </div>
                ),
              };
            })}
          />

          {total > pageSize && (
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8 }}>
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
