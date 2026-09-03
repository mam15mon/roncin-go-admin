import React, { useEffect, useState } from 'react';
import {
  App,
  Button,
  Col,
  Descriptions,
  Drawer,
  Empty,
  List,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { history } from '@umijs/max';
import {
  seaOrderChangeServiceListSeaOrderChangeEvents,
  seaOrderChangeServiceGetSeaOrderChangeEvent,
} from '@/services/roncin/seaOrderChangeService';

const { Text, Paragraph } = Typography;

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const messageValue = (error as { message?: unknown }).message;
    if (typeof messageValue === 'string' && messageValue) return messageValue;
  }
  return fallback;
}

function formatSnapshot(raw?: string): string {
  if (!raw) return '无';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return '快照 JSON 已损坏，无法展示；请联系管理员核查历史数据。';
  }
}

function renderEventSummary(record: API.SeaOrderChangeEventSummary) {
  if (record.eventType === 'SPLIT' && record.splitSummary) {
    return (
      <div>
        <Text strong>拆分结果：共 {record.splitSummary.resultCount} 票</Text>
        <div style={{ marginTop: 4 }}>
          {record.splitSummary.results?.map((result) => (
            <Tag
              key={result.orderId || result.orderNo}
              color={result.resultRole === 'ORIGINAL' ? 'default' : 'cyan'}
            >
              {result.resultRole === 'ORIGINAL' ? '原票' : '新票'}:{' '}
              {result.orderId ? (
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0 }}
                  onClick={() => history.push(`/orders/sea-export/${result.orderId}`)}
                >
                  {result.orderNo}
                </Button>
              ) : (
                result.orderNo
              )}{' '}
              ({result.packageCount} 件 / {result.grossWeightKg} KGS / {result.volumeCbm} CBM)
              {result.finalMasterNo ? ` · MBL ${result.finalMasterNo}` : ''}
            </Tag>
          ))}
        </div>
      </div>
    );
  }
  if (record.eventType === 'REASSIGNMENT' && record.reassignmentSummary) {
    const summary = record.reassignmentSummary;
    return (
      <div>
        <div>
          <Text type="secondary">原母单：</Text>
          <Text code>{summary.previousMasterNo || '无'}</Text>
          <span style={{ margin: '0 8px' }}>➔</span>
          <Text type="secondary">新母单：</Text>
          <Text code style={{ color: '#1677ff' }}>{summary.targetMasterNo || '新录入'}</Text>
        </div>
        <div style={{ marginTop: 4 }}>
          <Tag color="orange">责任：{summary.responsibilityType || '-'}</Tag>
          {summary.responsiblePartnerName && (
            <Text type="secondary">责任方：{summary.responsiblePartnerName}</Text>
          )}
        </div>
      </div>
    );
  }
  return record.noteOrReason || '-';
}

interface SeaOrderChangeHistorySectionProps {
  orderId: string;
  onOpenAll: () => void;
}

export const SeaOrderChangeHistorySection: React.FC<SeaOrderChangeHistorySectionProps> = ({
  orderId,
  onOpenAll,
}) => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [events, setEvents] = useState<API.SeaOrderChangeEventSummary[]>([]);

  useEffect(() => {
    if (!orderId) return;
    let active = true;
    setLoading(true);
    seaOrderChangeServiceListSeaOrderChangeEvents({ orderId, page: 1, pageSize: 5 })
      .then((response) => {
        if (active) setEvents(response?.data || []);
      })
      .catch((error: unknown) => {
        if (active) message.error(getErrorMessage(error, '加载拆票与改配记录失败'));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [message, orderId]);

  return (
    <Col span={24}>
      <Spin spinning={loading}>
        {events.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无拆票或改配记录" />
        ) : (
          <List
            size="small"
            dataSource={events}
            renderItem={(event) => (
              <List.Item
                actions={[
                  <Button key="all" type="link" size="small" onClick={onOpenAll}>
                    查看详情
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={
                    <Space>
                      <Tag color={event.eventType === 'SPLIT' ? 'purple' : 'blue'}>
                        {event.eventType === 'SPLIT' ? '部分拆票' : '整票改配'}
                      </Tag>
                      <Text type="secondary">{event.createdAt}</Text>
                      <Text type="secondary">{event.operatorName || '-'}</Text>
                    </Space>
                  }
                  description={renderEventSummary(event)}
                />
              </List.Item>
            )}
          />
        )}
        <div style={{ textAlign: 'right', marginTop: 8 }}>
          <Button type="link" onClick={onOpenAll}>查看全部拆票与改配历史</Button>
        </div>
      </Spin>
    </Col>
  );
};

interface SeaOrderChangeHistoryDrawerProps {
  orderId: string;
  open: boolean;
  onClose: () => void;
}

export const SeaOrderChangeHistoryDrawer: React.FC<SeaOrderChangeHistoryDrawerProps> = ({
  orderId,
  open,
  onClose,
}) => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [events, setEvents] = useState<API.SeaOrderChangeEventSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [currentDetail, setCurrentDetail] = useState<API.SeaOrderChangeEventDetailData | null>(null);

  const loadEvents = async () => {
    if (!orderId || !open) return;
    setLoading(true);
    try {
      const resp = await seaOrderChangeServiceListSeaOrderChangeEvents({
        orderId,
        page,
        pageSize,
      });
      if (resp?.data) {
        setEvents(resp.data || []);
        setTotal(resp.total || 0);
      }
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '加载拆票与改配历史失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      loadEvents();
    }
  }, [orderId, open, page, pageSize]);

  const viewDetail = async (record: API.SeaOrderChangeEventSummary) => {
    setDetailModalOpen(true);
    setDetailLoading(true);
    try {
      const resp = await seaOrderChangeServiceGetSeaOrderChangeEvent({
        orderId,
        eventId: record.id || '',
        eventType: record.eventType,
      });
      if (resp?.data) {
        setCurrentDetail(resp.data);
      }
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '加载变更事件详情失败'));
      setDetailModalOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const columns: ColumnsType<API.SeaOrderChangeEventSummary> = [
    {
      title: '事件类型',
      dataIndex: 'eventType',
      width: 100,
      render: (type: string) => {
        if (type === 'SPLIT') {
          return <Tag color="purple">部分拆票</Tag>;
        }
        return <Tag color="blue">整票改配</Tag>;
      },
    },
    {
      title: '发生时间',
      dataIndex: 'createdAt',
      width: 170,
    },
    {
      title: '操作人',
      dataIndex: 'operatorName',
      width: 120,
      render: (name: string) => name || '-',
    },
    {
      title: '变更内容摘要',
      render: (_, record) => renderEventSummary(record),
    },
    {
      title: '原因/备注',
      dataIndex: 'noteOrReason',
      width: 160,
      ellipsis: true,
      render: (val: string) => val || '-',
    },
    {
      title: '操作',
      width: 80,
      fixed: 'right',
      render: (_, record) => (
        <Button type="link" size="small" onClick={() => viewDetail(record)}>
          详情
        </Button>
      ),
    },
  ];

  return (
    <>
      <Drawer
        title="拆票与改配历史事件"
        open={open}
        onClose={onClose}
        width={850}
        destroyOnClose
      >
        <Table<API.SeaOrderChangeEventSummary>
          columns={columns}
          dataSource={events}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Drawer>

      <Modal
        title="变更事件前后快照详情"
        open={detailModalOpen}
        onCancel={() => {
          setDetailModalOpen(false);
          setCurrentDetail(null);
        }}
        footer={null}
        width={750}
        destroyOnClose
      >
        {detailLoading || !currentDetail ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Spin />
          </div>
        ) : (
          <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered size="small" column={2}>
              <Descriptions.Item label="事件类型">
                {currentDetail.eventType === 'SPLIT' ? (
                  <Tag color="purple">部分拆票</Tag>
                ) : (
                  <Tag color="blue">整票改配</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="发生时间">{currentDetail.createdAt}</Descriptions.Item>
              <Descriptions.Item label="操作人">{currentDetail.operatorName || '-'}</Descriptions.Item>
              <Descriptions.Item label="原因/备注">{currentDetail.noteOrReason || '-'}</Descriptions.Item>
            </Descriptions>

            {currentDetail.eventType === 'REASSIGNMENT' && (
              <>
                <Text strong>航程变更前后快照比对：</Text>
                <div style={{ display: 'flex', gap: 12 }}>
                  <div style={{ flex: 1, border: '1px solid #f0f0f0', borderRadius: 4, padding: 12 }}>
                    <Text type="secondary" strong>变更前快照：</Text>
                    <Paragraph style={{ marginTop: 8 }}>
                      <pre style={{ background: '#fafafa', padding: 8, fontSize: 12, borderRadius: 4 }}>
                        {formatSnapshot(currentDetail.beforeSnapshotJson)}
                      </pre>
                    </Paragraph>
                  </div>
                  <div style={{ flex: 1, border: '1px solid #e6f4ff', borderRadius: 4, padding: 12 }}>
                    <Text style={{ color: '#1677ff' }} strong>变更后快照：</Text>
                    <Paragraph style={{ marginTop: 8 }}>
                      <pre style={{ background: '#f6ffed', padding: 8, fontSize: 12, borderRadius: 4 }}>
                        {formatSnapshot(currentDetail.afterSnapshotJson)}
                      </pre>
                    </Paragraph>
                  </div>
                </div>
              </>
            )}

            {currentDetail.eventType === 'SPLIT' && (
              <>
                <Text strong>拆票前基线与守恒汇总：</Text>
                <div style={{ border: '1px solid #f0f0f0', borderRadius: 4, padding: 12 }}>
                  <Paragraph>
                    <pre style={{ background: '#fafafa', padding: 8, fontSize: 12, borderRadius: 4 }}>
                      {formatSnapshot(currentDetail.conservationSnapshotJson)}
                    </pre>
                  </Paragraph>
                </div>
              </>
            )}
          </Space>
        )}
      </Modal>
    </>
  );
};

export default SeaOrderChangeHistoryDrawer;
