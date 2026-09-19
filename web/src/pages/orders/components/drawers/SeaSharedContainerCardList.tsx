import { PlusOutlined } from '@ant-design/icons';
import { Button, Card, Col, Empty, Row, Tag, Typography } from 'antd';
import React from 'react';
import { STATUS_CONFIRMED } from './seaSharedContainerModels';

const { Text, Title } = Typography;

type SeaSharedContainerCardListProps = {
  containers: API.SeaSharedContainer[];
  selectedContainerId: string | null;
  canCreate: boolean;
  onSelect: (containerId: string | null) => void;
  onCreate: () => void;
};

/** 本航次共享物理箱卡片选择栏 */
export default function SeaSharedContainerCardList({
  containers,
  selectedContainerId,
  canCreate,
  onSelect,
  onCreate,
}: SeaSharedContainerCardListProps) {
  return (
    <div style={{ marginBottom: 16 }}>
      <Title level={5}>本航次共享物理箱 ({containers.length})</Title>
      {containers.length === 0 ? (
        <Empty
          description="当前航次暂无共享物理箱"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        >
          {canCreate && (
            <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>
              立即创建共享箱
            </Button>
          )}
        </Empty>
      ) : (
        <Row gutter={[12, 12]}>
          {containers.map((cntr) => {
            const isSelected = cntr.id === selectedContainerId;
            const cntrConfirmed = cntr.status === STATUS_CONFIRMED;
            return (
              <Col xs={24} sm={12} md={8} key={cntr.id}>
                <Card
                  size="small"
                  hoverable
                  onClick={() => onSelect(cntr.id || null)}
                  style={{
                    borderColor: isSelected ? '#1677ff' : '#f0f0f0',
                    borderWidth: isSelected ? 2 : 1,
                    backgroundColor: isSelected ? '#f6ffed' : '#ffffff',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <Text
                      strong
                      style={{ fontFamily: 'monospace', fontSize: 15 }}
                    >
                      {cntr.containerNo}
                    </Text>
                    {cntrConfirmed ? (
                      <Tag color="success">已确认</Tag>
                    ) : (
                      <Tag color="warning">草稿</Tag>
                    )}
                  </div>
                  <div
                    style={{
                      marginTop: 4,
                      fontSize: 12,
                      color: 'rgba(0,0,0,0.65)',
                    }}
                  >
                    <div>
                      规格: {cntr.containerSpecName || cntr.containerSpecId}
                      {cntr.sealNo ? ` | 封号: ${cntr.sealNo}` : ''}
                    </div>
                    <div>
                      容量: {cntr.packageCount} PCS / {cntr.grossWeightKg} KG /{' '}
                      {cntr.volumeCbm} CBM
                    </div>
                  </div>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}
    </div>
  );
}
