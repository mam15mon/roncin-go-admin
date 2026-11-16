import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { AirWeightSummary } from './air-list-columns';

afterEach(cleanup);

describe('航空列表确认重量', () => {
  it('无确认保留未确认，不伪造零', () => {
    render(<AirWeightSummary summaries={[]} />);
    expect(screen.getByText('未确认')).toBeInTheDocument();
  });
  it('两个来源独立显示，确认零及过期原值不丢失', () => {
    render(
      <AirWeightSummary
        summaries={[
          {
            source: 'PLANNED',
            confirmedWeightKg: '0',
            confirmationStale: false,
          },
          {
            source: 'ACTUAL',
            confirmedWeightKg: '120.50000000',
            confirmationStale: true,
          },
        ]}
      />,
    );
    expect(screen.getByText('委托：0 KG')).toBeInTheDocument();
    expect(screen.getByText('实际：120.5 KG')).toBeInTheDocument();
    expect(screen.getAllByText('需重确认')).toHaveLength(1);
  });
});
