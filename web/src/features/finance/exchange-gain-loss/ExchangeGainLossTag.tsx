import { Tag } from 'antd';

type ExchangeGainLossTagProps = {
  value: string | number | null | undefined;
  baseCurrency?: string | null;
};

/**
 * 汇兑损益展示：正数收益绿标、负数损失红标、零显示 0.00 本位币。
 * 口径与计算留在调用方；此处只负责展示形态。
 */
export function ExchangeGainLossTag({
  value,
  baseCurrency,
}: ExchangeGainLossTagProps) {
  const numeric = Number(value || 0);
  if (numeric > 0) {
    return (
      <Tag color="green">
        +{value} {baseCurrency} (收益)
      </Tag>
    );
  }
  if (numeric < 0) {
    return (
      <Tag color="red">
        {value} {baseCurrency} (损失)
      </Tag>
    );
  }
  return <span>0.00 {baseCurrency}</span>;
}
