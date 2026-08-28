import { CheckOutlined } from '@ant-design/icons';
import { Button, Card, Col, Row, Space, Tag } from 'antd';
import React from 'react';

export const PRESET_COLORS = [
  '#FFF7E6', // 浅橙黄
  '#FFFBE6', // 浅黄
  '#E6F4FF', // 浅蓝
  '#F9F0FF', // 浅紫
  '#F6FFED', // 浅绿
  '#FFF0F6', // 浅粉
  '#F0F5FF', // 浅靛
  '#FCFFE6', // 浅柠
  '#FFFFFF', // 纯白（无高亮）
];

export interface RowColorsConfig {
  unbilled: string;
  unverifiedUninvoiced: string;
  invoicedUnverified: string;
  invoicedPartiallyVerified: string;
  partiallyVerifiedUninvoiced: string;
  verifiedUninvoiced: string;
  completed: string;
}

const COLOR_STATUS_ITEMS: {
  key: keyof RowColorsConfig;
  tagColor: string;
  label: string;
  desc: string;
}[] = [
  {
    key: 'unbilled',
    tagColor: 'gold',
    label: '账单未建立',
    desc: '（草稿/未出账单）',
  },
  {
    key: 'unverifiedUninvoiced',
    tagColor: 'orange',
    label: '未核销未开票',
    desc: '（已确认待处理）',
  },
  {
    key: 'invoicedUnverified',
    tagColor: 'blue',
    label: '已开票未核销',
    desc: '（发票已开待收付款）',
  },
  {
    key: 'invoicedPartiallyVerified',
    tagColor: 'cyan',
    label: '已开票部分核销',
    desc: '（已开发票且部分收付款）',
  },
  {
    key: 'partiallyVerifiedUninvoiced',
    tagColor: 'purple',
    label: '部分核销未开票',
    desc: '（未开发票但部分收付款）',
  },
  {
    key: 'verifiedUninvoiced',
    tagColor: 'purple',
    label: '已核销未开票',
    desc: '（资金已全收付待开票）',
  },
  {
    key: 'completed',
    tagColor: 'green',
    label: '已完成',
    desc: '（已全额核销并开票完毕）',
  },
];

type RowColorSettingsProps = {
  rowColors: RowColorsConfig;
  onColorChange: (key: keyof RowColorsConfig, color: string) => void;
  onResetColors: () => void;
};

export default function RowColorSettings({
  rowColors,
  onColorChange,
  onResetColors,
}: RowColorSettingsProps) {
  return (
    <Card
      size="small"
      title={
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <span>7 类费用财务进度 行背景高亮颜色设置</span>
          <Button size="small" onClick={onResetColors}>
            重置默认颜色
          </Button>
        </div>
      }
      style={{ background: '#fafafa' }}
    >
      <Row gutter={[16, 12]}>
        {COLOR_STATUS_ITEMS.map((item) => (
          <Col span={12} key={item.key}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '8px 12px',
                background: rowColors[item.key],
                border: '1px solid #d9d9d9',
                borderRadius: 4,
              }}
            >
              <Space>
                <Tag color={item.tagColor}>{item.label}</Tag>
                <span style={{ fontSize: 12, color: '#595959' }}>
                  {item.desc}
                </span>
              </Space>
              <Space size={4}>
                {PRESET_COLORS.map((c) => (
                  <div
                    key={c}
                    onClick={() => onColorChange(item.key, c)}
                    style={{
                      width: 18,
                      height: 18,
                      background: c,
                      border:
                        rowColors[item.key] === c
                          ? '2px solid #1677ff'
                          : '1px solid #d9d9d9',
                      borderRadius: 3,
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    {rowColors[item.key] === c && (
                      <CheckOutlined
                        style={{ fontSize: 10, color: '#1677ff' }}
                      />
                    )}
                  </div>
                ))}
              </Space>
            </div>
          </Col>
        ))}
      </Row>
    </Card>
  );
}
