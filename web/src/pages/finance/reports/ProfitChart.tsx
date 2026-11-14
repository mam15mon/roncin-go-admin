import { Empty, Select, Space, Table, Typography } from 'antd';
import React, { useId, useState } from 'react';
import { SectionCard } from '@/components/ui';
import { formatAmount } from '@/utils/format';

/** 按完整匹配月序列绘图；金额运算由服务端完成，Number 只用于图形坐标。 */
export default function ProfitChart({
  response,
}: {
  response: API.ListProfitReportResponse;
}) {
  const currencies = (response.totals ?? [])
    .map((item) => item.baseCurrency)
    .filter((value): value is string => Boolean(value));
  const [selected, setSelected] = useState<string>();
  const currency =
    selected && currencies.includes(selected) ? selected : currencies[0];
  const titleId = useId();
  const data = (response.months ?? []).map((month) => {
    const amount = month.amounts?.find(
      (item) => item.baseCurrency === currency,
    );
    return {
      month: month.month ?? '',
      gross: amount?.profitAmount ?? '0',
      net: amount?.netProfitAmount ?? '0',
    };
  });
  const max = Math.max(
    1,
    ...data.flatMap((row) => [
      Math.abs(Number(row.gross)),
      Math.abs(Number(row.net)),
    ]),
  );
  const width = Math.max(640, data.length * 74 + 70);
  const zero = 150;
  return (
    <SectionCard
      title="月度利润分析"
      extra={
        <Space>
          <Typography.Text>本币</Typography.Text>
          <Select
            aria-label="图表本币"
            value={currency}
            options={currencies.map((value) => ({ value, label: value }))}
            onChange={setSelected}
            style={{ minWidth: 100 }}
          />
        </Space>
      }
    >
      {data.length && currency ? (
        <>
          <div style={{ overflowX: 'auto' }}>
            <svg
              role="img"
              aria-labelledby={titleId}
              viewBox={`0 0 ${width} 320`}
              style={{ width: '100%', minWidth: width, height: 320 }}
            >
              <title
                id={titleId}
              >{`完整筛选范围月度含税与未税利润（${currency}）`}</title>
              <line
                x1="50"
                x2={width - 10}
                y1={zero}
                y2={zero}
                stroke="#8c98a8"
              />
              <text x="6" y="16" fill="#5b6b82" fontSize="11">
                {formatAmount(max)}
              </text>
              <text x="6" y={zero - 5} fill="#5b6b82" fontSize="11">
                0.00
              </text>
              {data.map((row, index) => (
                <g key={row.month}>
                  {[
                    {
                      value: row.gross,
                      color: '#1677ff',
                      offset: 0,
                      label: '含税',
                    },
                    {
                      value: row.net,
                      color: '#52c41a',
                      offset: 24,
                      label: '未税',
                    },
                  ].map((item) => {
                    const value = Number(item.value);
                    const height = (Math.abs(value) / max) * 125;
                    return (
                      <rect
                        key={item.label}
                        x={64 + index * 74 + item.offset}
                        y={value >= 0 ? zero - height : zero}
                        width="21"
                        height={height}
                        fill={item.color}
                      >
                        <title>{`${row.month === 'UNDATED' ? '未设置业务日期' : row.month} ${item.label}利润：${formatAmount(item.value)} ${currency}`}</title>
                      </rect>
                    );
                  })}
                  <text
                    x={86 + index * 74}
                    y="298"
                    textAnchor="middle"
                    fill="#5b6b82"
                    fontSize="11"
                  >
                    {row.month === 'UNDATED' ? '未设置日期' : row.month}
                  </text>
                </g>
              ))}
            </svg>
          </div>
          <Space style={{ marginBottom: 8 }}>
            <Typography.Text style={{ color: '#1677ff' }}>
              ■ 含税利润
            </Typography.Text>
            <Typography.Text style={{ color: '#389e0d' }}>
              ■ 未税利润
            </Typography.Text>
          </Space>
          <Table
            size="small"
            rowKey="month"
            pagination={false}
            dataSource={data}
            columns={[
              {
                title: '月份',
                dataIndex: 'month',
                render: (value) =>
                  value === 'UNDATED' ? '未设置业务日期' : value,
              },
              {
                title: `含税利润（${currency}）`,
                dataIndex: 'gross',
                align: 'right',
                render: (value) => formatAmount(value),
              },
              {
                title: `未税利润（${currency}）`,
                dataIndex: 'net',
                align: 'right',
                render: (value) => formatAmount(value),
              },
            ]}
          />
        </>
      ) : (
        <Empty
          description="当前筛选没有可分析的费用"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      )}
    </SectionCard>
  );
}
