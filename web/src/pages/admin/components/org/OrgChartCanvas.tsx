import { OrganizationChart } from '@ant-design/graphs';
import { ApartmentOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Space, Spin, Tag, Tooltip } from 'antd';
import React, { useRef } from 'react';
import { getOrganizationKindMeta } from './types';

type OrgChartCanvasProps = {
  loading: boolean;
  graphData: { nodes: any[]; edges: any[] };
  chartDirection: 'vertical' | 'horizontal';
  selectedId: string;
  onSelectNode: (id: string) => void;
  onOpenDrawer: () => void;
};

export default function OrgChartCanvas({
  loading,
  graphData,
  chartDirection,
  selectedId,
  onSelectNode,
  onOpenDrawer,
}: OrgChartCanvasProps) {
  const graphRef = useRef<any>(null);

  return (
    <Card
      styles={{ body: { padding: 0 } }}
      style={{
        minHeight: 640,
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      <Spin spinning={loading}>
        {graphData.nodes.length > 0 ? (
          <div
            style={{ height: 'calc(100vh - 270px)', minHeight: 600 }}
            onMouseDown={(e) => {
              if (e.button === 1) {
                e.preventDefault();
              }
            }}
          >
            <OrganizationChart
              ref={graphRef}
              data={graphData}
              direction={chartDirection}
              autoFit="center"
              node={{
                style: {
                  size: [210, 80],
                  ports:
                    chartDirection === 'vertical'
                      ? [
                          { key: 'in', placement: 'top' },
                          { key: 'out', placement: 'bottom' },
                        ]
                      : [
                          { key: 'in', placement: 'left' },
                          { key: 'out', placement: 'right' },
                        ],
                  component: (nodeData: Record<string, unknown>) => {
                    const item = (nodeData.data || nodeData) as {
                      id: string;
                      name: string;
                      code: string;
                      kind: number;
                      enabled: boolean;
                      parentId?: string;
                      childrenCount?: number;
                    };
                    const isCurrentSelected = item.id === selectedId;

                    return (
                      <div
                        onClick={() => {
                          if (item.id) {
                            onSelectNode(item.id);
                            onOpenDrawer();
                          }
                        }}
                        style={{
                          width: 210,
                          height: 80,
                          backgroundColor: '#ffffff',
                          borderRadius: 8,
                          border: isCurrentSelected
                            ? '2px solid #1677ff'
                            : '1px solid #e2e8f0',
                          boxShadow: isCurrentSelected
                            ? '0 4px 14px rgba(22, 119, 255, 0.25)'
                            : '0 2px 6px rgba(0, 0, 0, 0.04)',
                          padding: '9px 12px',
                          display: 'flex',
                          flexDirection: 'column',
                          justifyContent: 'space-between',
                          cursor: 'pointer',
                          boxSizing: 'border-box',
                          position: 'relative',
                          transition: 'all 0.15s ease',
                        }}
                      >
                        {item.parentId && (
                          <div
                            style={{
                              position: 'absolute',
                              ...(chartDirection === 'vertical'
                                ? {
                                    top: -4,
                                    left: '50%',
                                    transform: 'translateX(-50%)',
                                  }
                                : {
                                    left: -4,
                                    top: '50%',
                                    transform: 'translateY(-50%)',
                                  }),
                              width: 8,
                              height: 8,
                              borderRadius: '50%',
                              backgroundColor: '#1677ff',
                              border: '2px solid #ffffff',
                              boxShadow: '0 1px 3px rgba(0, 0, 0, 0.2)',
                              zIndex: 5,
                            }}
                          />
                        )}

                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                          }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 6,
                              minWidth: 0,
                            }}
                          >
                            <ApartmentOutlined
                              style={{
                                color: isCurrentSelected
                                  ? '#1677ff'
                                  : 'rgba(0, 0, 0, 0.45)',
                                fontSize: 14,
                              }}
                            />
                            <span
                              style={{
                                fontWeight: 600,
                                fontSize: 13,
                                color: isCurrentSelected
                                  ? '#1677ff'
                                  : 'rgba(0, 0, 0, 0.88)',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                maxWidth: 120,
                              }}
                              title={item.name}
                            >
                              {item.name}
                            </span>
                          </div>
                          {item.enabled ? (
                            <Tag
                              color="success"
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 10,
                                lineHeight: '16px',
                                padding: '0 4px',
                              }}
                            >
                              启用
                            </Tag>
                          ) : (
                            <Tag
                              color="default"
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 10,
                                lineHeight: '16px',
                                padding: '0 4px',
                              }}
                            >
                              停用
                            </Tag>
                          )}
                        </div>

                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            fontSize: 11,
                            color: 'rgba(0, 0, 0, 0.45)',
                          }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 4,
                            }}
                          >
                            <Tag
                              color={getOrganizationKindMeta(item.kind)?.color}
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 10,
                                lineHeight: '16px',
                                padding: '0 4px',
                              }}
                            >
                              {getOrganizationKindMeta(item.kind)?.label}
                            </Tag>
                            <span
                              style={{
                                fontFamily: 'monospace',
                                color: 'rgba(0, 0, 0, 0.45)',
                              }}
                            >
                              {item.code}
                            </span>
                          </div>

                          {(item.childrenCount ?? 0) > 0 && (
                            <span style={{ color: '#1677ff', fontWeight: 500 }}>
                              {item.childrenCount} 个下级
                            </span>
                          )}
                        </div>

                        {(item.childrenCount ?? 0) > 0 && (
                          <div
                            style={{
                              position: 'absolute',
                              ...(chartDirection === 'vertical'
                                ? {
                                    bottom: -4,
                                    left: '50%',
                                    transform: 'translateX(-50%)',
                                  }
                                : {
                                    right: -4,
                                    top: '50%',
                                    transform: 'translateY(-50%)',
                                  }),
                              width: 8,
                              height: 8,
                              borderRadius: '50%',
                              backgroundColor: '#1677ff',
                              border: '2px solid #ffffff',
                              boxShadow: '0 1px 3px rgba(0, 0, 0, 0.2)',
                              zIndex: 5,
                            }}
                          />
                        )}
                      </div>
                    );
                  },
                },
              }}
              edge={{
                style: {
                  stroke: '#94a3b8',
                  lineWidth: 1.5,
                  strokeOpacity: 0.85,
                  endArrow: true,
                  router: {
                    type: 'orth',
                  },
                },
              }}
              behaviors={[
                'drag-canvas',
                'zoom-canvas',
                'drag-element',
                'collapse-expand',
              ]}
            />
          </div>
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="暂无组织架构数据"
            style={{ margin: '80px 0' }}
          />
        )}
      </Spin>

      {/* Floating Canvas Controls */}
      <div
        style={{
          position: 'absolute',
          bottom: 16,
          right: 16,
          zIndex: 10,
          backgroundColor: 'rgba(255, 255, 255, 0.95)',
          padding: '4px 8px',
          borderRadius: 6,
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.12)',
          border: '1px solid #e2e8f0',
        }}
      >
        <Space size={4}>
          <Tooltip title="自适应画布居中">
            <Button
              size="small"
              type="text"
              onClick={() => {
                graphRef.current?.fitCenter?.();
                graphRef.current?.fitView?.();
              }}
            >
              居中适界
            </Button>
          </Tooltip>
          <Tooltip title="放大">
            <Button
              size="small"
              type="text"
              onClick={() => {
                const currentZoom = graphRef.current?.getZoom?.() || 1;
                graphRef.current?.zoomTo?.(currentZoom * 1.2);
              }}
            >
              +
            </Button>
          </Tooltip>
          <Tooltip title="缩小">
            <Button
              size="small"
              type="text"
              onClick={() => {
                const currentZoom = graphRef.current?.getZoom?.() || 1;
                graphRef.current?.zoomTo?.(currentZoom * 0.8);
              }}
            >
              -
            </Button>
          </Tooltip>
          <Tooltip title="重置缩放到 100%">
            <Button
              size="small"
              type="text"
              onClick={() => graphRef.current?.zoomTo?.(1)}
            >
              1:1
            </Button>
          </Tooltip>
        </Space>
      </div>
    </Card>
  );
}
