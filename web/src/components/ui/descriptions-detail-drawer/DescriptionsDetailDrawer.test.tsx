import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DItem, DescriptionsDetailDrawer } from './DescriptionsDetailDrawer';

describe('DItem', () => {
  it('空值显示短横线并保留零值', () => {
    render(
      <DescriptionsDetailDrawer
        open
        detail={{ id: 'detail-1' }}
        title="详情"
        onClose={() => undefined}
        descriptions={() => (
          <>
            <DItem label="空值">{undefined}</DItem>
            <DItem label="零值">{0}</DItem>
          </>
        )}
      />,
    );

    expect(screen.getByText('-')).toBeInTheDocument();
    expect(screen.getByText('0')).toBeInTheDocument();
  });
});
