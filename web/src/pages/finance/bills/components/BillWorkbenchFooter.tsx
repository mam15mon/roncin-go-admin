import { ArrowLeftOutlined, FileDoneOutlined } from '@ant-design/icons';
import { Button, Space } from 'antd';
import React from 'react';
import type { BillCreationMode } from './billWorkbenchHelpers';

type BillWorkbenchFooterProps = {
  current: number;
  loading: boolean;
  mode: BillCreationMode;
  groupCount: number;
  nettingPairCount: number;
  onClose: () => void;
  onBack: () => void;
  onNext: () => void;
  onCreate: () => void;
};

/** 建账工作台吸底操作栏 */
export default function BillWorkbenchFooter({
  current,
  loading,
  mode,
  groupCount,
  nettingPairCount,
  onClose,
  onBack,
  onNext,
  onCreate,
}: BillWorkbenchFooterProps) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
      <Button onClick={onClose}>{current === 3 ? '关闭' : '取消'}</Button>
      {current < 3 && (
        <Space>
          {current > 0 && (
            <Button
              icon={<ArrowLeftOutlined />}
              disabled={loading}
              onClick={onBack}
            >
              上一步
            </Button>
          )}
          {current < 2 && (
            <Button type="primary" loading={loading} onClick={onNext}>
              下一步
            </Button>
          )}
          {current === 2 && (
            <Button
              type="primary"
              icon={<FileDoneOutlined />}
              loading={loading}
              onClick={onCreate}
            >
              {mode === 'NETTING'
                ? `原子生成 ${groupCount} 张账单与 ${nettingPairCount} 张对冲单`
                : `原子生成 ${groupCount} 张账单`}
            </Button>
          )}
        </Space>
      )}
    </div>
  );
}
