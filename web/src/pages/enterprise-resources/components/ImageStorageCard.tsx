import { Card, Space, Tag } from 'antd';
import { formatStorageSize } from '../resourceConstants';

type ImageStorageCardProps = {
  capabilities?: API.GetEnterpriseResourceCapabilitiesResponse;
};

/** 图片管理页签顶部的组织图片存储空间用量展示卡片。 */
export default function ImageStorageCard({
  capabilities,
}: ImageStorageCardProps) {
  return (
    <Card
      styles={{ body: { padding: '12px 16px' } }}
      style={{
        marginBottom: 16,
        borderRadius: 8,
        border: '1px solid #f0f0f0',
      }}
    >
      <Space>
        <span style={{ color: 'rgba(0, 0, 0, 0.65)', fontSize: 13 }}>
          组织图片存储空间：
        </span>
        <Tag color="blue">
          已用 {formatStorageSize(capabilities?.imageUsedStorageBytes)}
        </Tag>
        {Number(capabilities?.imageStorageQuotaBytes) > 0 ? (
          <>
            <Tag>
              总额 {formatStorageSize(capabilities?.imageStorageQuotaBytes)}
            </Tag>
            <Tag color="cyan">
              {Math.min(
                100,
                (Number(capabilities?.imageUsedStorageBytes) /
                  Number(capabilities?.imageStorageQuotaBytes)) *
                  100,
              ).toFixed(2)}
              %
            </Tag>
          </>
        ) : (
          <Tag>不限额</Tag>
        )}
      </Space>
    </Card>
  );
}
