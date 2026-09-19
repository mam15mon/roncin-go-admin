import {
  CheckSquareOutlined,
  EyeOutlined,
  MinusSquareOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { Button, Input, Space, Tag, Typography } from 'antd';

const { Text } = Typography;

type PermissionMatrixHeaderProps = {
  selectedCount: number;
  totalCount: number;
  readSelectedCount: number;
  writeSelectedCount: number;
  keyword: string;
  setKeyword: (value: string) => void;
  onSelectAllWrite: () => void;
  onSelectAllRead: () => void;
  onClearAll: () => void;
};

/** 权限矩阵顶部统计栏、全局快捷操作与搜索框。 */
export default function PermissionMatrixHeader({
  selectedCount,
  totalCount,
  readSelectedCount,
  writeSelectedCount,
  keyword,
  setKeyword,
  onSelectAllWrite,
  onSelectAllRead,
  onClearAll,
}: PermissionMatrixHeaderProps) {
  return (
    <>
      {/* Top Header & Global Actions */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 8,
        }}
      >
        <Space size={8}>
          <div
            style={{
              width: 3,
              height: 15,
              backgroundColor: '#1677ff',
              borderRadius: 2,
            }}
          />
          <Text strong style={{ fontSize: 13, color: '#1e293b' }}>
            功能权限配置（华为云 IAM 策略模式）
          </Text>
          <Tag color="blue" variant="filled">
            已选 {selectedCount} / {totalCount} 项
          </Tag>
          <Tag color="cyan" variant="filled">
            查看 (只读): {readSelectedCount} 项
          </Tag>
          <Tag color="orange" variant="filled">
            编辑/管理: {writeSelectedCount} 项
          </Tag>
        </Space>

        <Space size={6}>
          <Button
            size="small"
            type="primary"
            ghost
            icon={<CheckSquareOutlined />}
            onClick={onSelectAllWrite}
          >
            全选读写
          </Button>
          <Button size="small" icon={<EyeOutlined />} onClick={onSelectAllRead}>
            全选只读
          </Button>
          <Button
            size="small"
            danger
            ghost
            icon={<MinusSquareOutlined />}
            onClick={onClearAll}
          >
            清空
          </Button>
        </Space>
      </div>

      {/* Search Bar */}
      <Input
        placeholder="搜索功能对象、业务线或权限操作（如：海运、用户、账单、delete）..."
        prefix={<SearchOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />}
        allowClear
        size="small"
        value={keyword}
        onChange={(e) => setKeyword(e.target.value)}
        style={{ marginBottom: 10, backgroundColor: '#ffffff' }}
      />
    </>
  );
}
