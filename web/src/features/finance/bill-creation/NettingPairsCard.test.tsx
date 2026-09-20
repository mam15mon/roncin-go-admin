import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import NettingPairsCard from './NettingPairsCard';

describe('对冲抵销预览卡片', () => {
  it('展示毛额、抵销额与净应收/净应付，金额带共同账单币种', () => {
    render(
      <NettingPairsCard
        pairs={[
          {
            settlementPartyId: 'party-1',
            settlementPartyName: '验收同行',
            currency: 'CNY',
            receivableGrossAmount: '100.00000000',
            payableGrossAmount: '70.00000000',
            offsetAmount: '70.00000000',
            netReceivableAmount: '30.00000000',
            netPayableAmount: '0.00000000',
          },
          {
            settlementPartyId: 'party-1',
            settlementPartyName: '验收同行',
            currency: 'USD',
            receivableGrossAmount: '10.00000000',
            payableGrossAmount: '25.00000000',
            offsetAmount: '10.00000000',
            netReceivableAmount: '0.00000000',
            netPayableAmount: '15.00000000',
          },
        ]}
      />,
    );

    expect(screen.getAllByText('验收同行')).toHaveLength(2);
    expect(screen.getByText('100.00000000 CNY')).toBeInTheDocument();
    // 应付毛额与抵销额同为 70：毛额、抵销各出现一次
    expect(screen.getAllByText('70.00000000 CNY')).toHaveLength(2);
    // 净应收为正数展示金额，另一边为零时显示占位符
    expect(screen.getByText('30.00000000 CNY')).toBeInTheDocument();
    expect(screen.getByText('15.00000000 USD')).toBeInTheDocument();
    // 抵销额 = 双方毛额较小值提示存在
    expect(screen.getByText('抵销额 = 双方毛额较小值')).toBeInTheDocument();
  });

  it('无对冲汇总时不渲染', () => {
    const { container } = render(<NettingPairsCard pairs={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
