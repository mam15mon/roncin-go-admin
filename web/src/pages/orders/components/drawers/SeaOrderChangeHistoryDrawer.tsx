import { paginationSizeChanger } from '@root/config/pagination';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Listy,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React, { useEffect, useState } from 'react';
import { DRAWER_SIZE, ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { history } from '@/router/history';
import {
  seaOrderChangeServiceGetSeaOrderChangeEvent,
  seaOrderChangeServiceListSeaOrderChangeEvents,
} from '@/services/roncin/seaOrderChangeService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import { formatDate } from '@/utils/format';

const { Text, Paragraph } = Typography;

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
                  onClick={() =>
                    history.push(`/orders/sea-export/${result.orderId}`)
                  }
                >
                  {result.orderNo}
                </Button>
              ) : (
                result.orderNo
              )}{' '}
              ({result.packageCount} 件 / {result.grossWeightKg} KGS /{' '}
              {result.volumeCbm} CBM)
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
          <Text code style={{ color: '#1677ff' }}>
            {summary.targetMasterNo || '新录入'}
          </Text>
        </div>
        <div style={{ marginTop: 4 }}>
          <Tag color="orange">责任：{summary.responsibilityType || '-'}</Tag>
          {summary.responsiblePartnerName && (
            <Text type="secondary">
              责任方：{summary.responsiblePartnerName}
            </Text>
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
  /**
   * 向「关联与记录」页签角标暴露服务端 total；未加载或失败时不上报，
   * 预览返回的五条不冒充总数。
   */
  onTotalChange?: (total: number | undefined) => void;
}

export const SeaOrderChangeHistorySection: React.FC<
  SeaOrderChangeHistorySectionProps
> = ({ orderId, onOpenAll, onTotalChange }) => {
  const query = useQuery({
    queryKey: ['orders', 'change-history-preview', { orderId }],
    enabled: Boolean(orderId),
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await seaOrderChangeServiceListSeaOrderChangeEvents(
        { orderId, page: 1, pageSize: 5 },
        { signal, skipErrorHandler: true },
      );
      return {
        events: unwrapList(
          ensureListResponse(response, '加载拆票与改配记录失败'),
        ),
        total: typeof response.total === 'number' ? response.total : undefined,
      };
    },
  });
  const events = query.data?.events ?? [];
  const loading = query.isPending && query.isFetching;
  useEffect(() => {
    onTotalChange?.(query.isSuccess ? query.data.total : undefined);
  }, [onTotalChange, query.data, query.isSuccess]);

  return (
    <Col span={24}>
      <Spin spinning={loading}>
        {query.isError ? (
          <Alert type="error" title="加载拆票与改配记录失败" />
        ) : events.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="暂无拆票或改配记录"
          />
        ) : (
          <Listy
            items={events}
            rowKey="id"
            itemRender={(event) => (
              <div
                style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}
              >
                <div style={{ flex: 'auto', minWidth: 0 }}>
                  <Space style={{ marginBottom: 4 }}>
                    <Tag
                      color={event.eventType === 'SPLIT' ? 'purple' : 'blue'}
                    >
                      {event.eventType === 'SPLIT' ? '部分拆票' : '整票改配'}
                    </Tag>
                    <Text type="secondary">{formatDate(event.createdAt)}</Text>
                    <Text type="secondary">{event.operatorName || '-'}</Text>
                  </Space>
                  <div>{renderEventSummary(event)}</div>
                </div>
                <Button type="link" size="small" onClick={onOpenAll}>
                  查看详情
                </Button>
              </div>
            )}
          />
        )}
        <div style={{ textAlign: 'right', marginTop: 8 }}>
          <Button type="link" onClick={onOpenAll}>
            查看全部拆票与改配历史
          </Button>
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

export const SeaOrderChangeHistoryDrawer: React.FC<
  SeaOrderChangeHistoryDrawerProps
> = ({ orderId, open, onClose }) => {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [detailSelection, setDetailSelection] = useState<{
    orderId: string;
    record: API.SeaOrderChangeEventSummary;
  }>();
  const detailTarget =
    detailSelection?.orderId === orderId ? detailSelection.record : undefined;
  const query = useQuery<{
    events: API.SeaOrderChangeEventSummary[];
    total: number;
  }>({
    queryKey: ['orders', 'change-history', { orderId, page, pageSize, open }],
    enabled: open && Boolean(orderId),
    placeholderData: (previous, previousQuery) =>
      open &&
      (
        previousQuery?.queryKey[2] as
          | { orderId?: string; open?: boolean }
          | undefined
      )?.open === true &&
      (previousQuery?.queryKey[2] as { orderId?: string } | undefined)
        ?.orderId === orderId
        ? previous
        : undefined,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await seaOrderChangeServiceListSeaOrderChangeEvents(
        { orderId, page, pageSize },
        { signal, skipErrorHandler: true },
      );
      return {
        events: unwrapList(
          ensureListResponse(response, '加载拆票与改配历史失败'),
        ),
        total: Number(response.total ?? 0),
      };
    },
  });
  const detailQuery = useQuery({
    queryKey: [
      'orders',
      'change-event-detail',
      {
        orderId,
        eventId: detailTarget?.id,
        eventType: detailTarget?.eventType,
        open,
      },
    ],
    enabled: open && Boolean(detailTarget?.id),
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await seaOrderChangeServiceGetSeaOrderChangeEvent(
        {
          orderId,
          eventId: detailTarget?.id ?? '',
          eventType: detailTarget?.eventType,
        },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data || response.success === false)
        throw new Error('加载变更事件详情失败');
      return response.data;
    },
  });
  const events = query.data?.events ?? [];
  const total = query.data?.total ?? 0;
  const loading = query.isFetching;
  const detailModalOpen = open && Boolean(detailTarget);
  const detailLoading = detailQuery.isFetching;
  const currentDetail = detailQuery.data;
  const viewDetail = (record: API.SeaOrderChangeEventSummary) =>
    setDetailSelection({ orderId, record });

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
      render: (value: string) => formatDate(value),
    },
    {
      title: '操作人',
      dataIndex: 'operatorName',
      width: 120,
      render: (name: string) => name || '-',
    },
    {
      title: '变更内容摘要',
      key: 'changeSummary',
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

  const columnSettings = useColumnSettings<
    ColumnsType<API.SeaOrderChangeEventSummary>[number]
  >({
    tableKey: 'orders:change-history',
    columns,
  });

  return (
    <>
      <Drawer
        title="拆票与改配历史事件"
        open={open}
        onClose={onClose}
        size={DRAWER_SIZE.MD}
        destroyOnHidden
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginBottom: 8,
          }}
        >
          {columnSettings.entry}
        </div>
        {query.isError && <Alert type="error" title="加载拆票与改配历史失败" />}
        <Table<API.SeaOrderChangeEventSummary>
          columns={columnSettings.columns}
          dataSource={events}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: paginationSizeChanger,
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
          setDetailSelection(undefined);
        }}
        footer={null}
        width={750}
        destroyOnHidden
      >
        {detailQuery.isError ? (
          <Alert type="error" title="加载变更事件详情失败" />
        ) : detailLoading || !currentDetail ? (
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
              <Descriptions.Item label="发生时间">
                {formatDate(currentDetail.createdAt)}
              </Descriptions.Item>
              <Descriptions.Item label="操作人">
                {currentDetail.operatorName || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="原因/备注">
                {currentDetail.noteOrReason || '-'}
              </Descriptions.Item>
            </Descriptions>

            {currentDetail.eventType === 'REASSIGNMENT' && (
              <>
                <Text strong>航程变更前后快照比对：</Text>
                <div style={{ display: 'flex', gap: 12 }}>
                  <div
                    style={{
                      flex: 1,
                      border: '1px solid #f0f0f0',
                      borderRadius: 4,
                      padding: 12,
                    }}
                  >
                    <Text type="secondary" strong>
                      变更前快照：
                    </Text>
                    <Paragraph style={{ marginTop: 8 }}>
                      <pre
                        style={{
                          background: '#fafafa',
                          padding: 8,
                          fontSize: 12,
                          borderRadius: 4,
                        }}
                      >
                        {formatSnapshot(currentDetail.beforeSnapshotJson)}
                      </pre>
                    </Paragraph>
                  </div>
                  <div
                    style={{
                      flex: 1,
                      border: '1px solid #e6f4ff',
                      borderRadius: 4,
                      padding: 12,
                    }}
                  >
                    <Text style={{ color: '#1677ff' }} strong>
                      变更后快照：
                    </Text>
                    <Paragraph style={{ marginTop: 8 }}>
                      <pre
                        style={{
                          background: '#f6ffed',
                          padding: 8,
                          fontSize: 12,
                          borderRadius: 4,
                        }}
                      >
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
                <div
                  style={{
                    border: '1px solid #f0f0f0',
                    borderRadius: 4,
                    padding: 12,
                  }}
                >
                  <Paragraph>
                    <pre
                      style={{
                        background: '#fafafa',
                        padding: 8,
                        fontSize: 12,
                        borderRadius: 4,
                      }}
                    >
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
