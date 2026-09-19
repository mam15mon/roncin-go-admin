import {
  CheckCircleOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  RollbackOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  Button,
  Card,
  Descriptions,
  Divider,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from 'antd';
import React from 'react';
import type { AllocationSummary } from './seaSharedContainerModels';

const { Text } = Typography;

type SeaSharedContainerDetailCardProps = {
  container: API.SeaSharedContainer;
  summary: AllocationSummary;
  isConfirmed: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  submitting: boolean;
  onSaveDraft: () => void;
  onConfirm: () => void;
  onWithdraw: () => void;
  onDelete: () => void;
};

/** 选定共享箱详情与守恒监控卡 */
export default function SeaSharedContainerDetailCard({
  container,
  summary,
  isConfirmed,
  canUpdate,
  canDelete,
  submitting,
  onSaveDraft,
  onConfirm,
  onWithdraw,
  onDelete,
}: SeaSharedContainerDetailCardProps) {
  return (
    <>
      <Divider style={{ margin: '16px 0' }} />
      <Card
        title={
          <Space>
            <span>共享物理箱：</span>
            <Text
              style={{
                fontFamily: 'monospace',
                fontWeight: 600,
                fontSize: 16,
              }}
            >
              {container.containerNo}
            </Text>
            <Tag color="blue">
              {container.containerSpecName || container.containerSpecId}
            </Tag>
            {isConfirmed ? (
              <Tag color="success" icon={<CheckCircleOutlined />}>
                已确认生效
              </Tag>
            ) : (
              <Tag color="warning" icon={<ExclamationCircleOutlined />}>
                草稿待确认
              </Tag>
            )}
            {summary.isBalanced ? (
              <Tag color="cyan">件重尺完全守恒</Tag>
            ) : (
              <Tag color="red">
                差额: {summary.diffPackages} PCS / {summary.diffWeight} KG /{' '}
                {summary.diffVolume} CBM
              </Tag>
            )}
          </Space>
        }
        extra={
          <Space>
            {!isConfirmed && canUpdate && (
              <>
                <Button
                  icon={<SaveOutlined />}
                  onClick={onSaveDraft}
                  loading={submitting}
                >
                  保存草稿
                </Button>
                <Button
                  type="primary"
                  icon={<CheckCircleOutlined />}
                  onClick={onConfirm}
                  loading={submitting}
                  disabled={!summary.isBalanced}
                >
                  确认分配
                </Button>
              </>
            )}
            {isConfirmed && canUpdate && (
              <Button
                icon={<RollbackOutlined />}
                onClick={onWithdraw}
                loading={submitting}
              >
                撤回至草稿
              </Button>
            )}
            {!isConfirmed && canDelete && (
              <Popconfirm
                title="确定删除此共享物理箱？"
                description="删除后相关草稿分配将一并清理。"
                onConfirm={onDelete}
                okText="确定"
                cancelText="取消"
              >
                <Button danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            )}
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        <Descriptions size="small" column={{ xs: 1, sm: 3 }}>
          <Descriptions.Item label="箱体总容量">
            {summary.totalPackages} PCS / {summary.totalWeight} KG /{' '}
            {summary.totalVolume} CBM
          </Descriptions.Item>
          <Descriptions.Item label="已分配总量">
            {summary.allocPackages} PCS / {summary.allocWeight} KG /{' '}
            {summary.allocVolume} CBM
          </Descriptions.Item>
          <Descriptions.Item label="剩余待分配">
            <Text
              type={summary.diffPackages === 0 ? 'secondary' : 'danger'}
              strong
            >
              {summary.diffPackages} PCS / {summary.diffWeight} KG /{' '}
              {summary.diffVolume} CBM
            </Text>
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </>
  );
}
