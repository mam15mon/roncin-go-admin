import { render } from '@testing-library/react';
import React from 'react';
import { describe, expect, it } from 'vitest';
import { ListQueryStateProvider, useListQueryState } from './index';

type Filters = { search: Record<string, unknown> };
const base = {
  tabKey: '/t',
  listId: 'l',
  defaultPageSize: 20,
  defaultFilters: { search: {} } as Filters,
};
const dropHidden = (f: Filters): Filters => ({
  search: Object.fromEntries(
    Object.entries(f.search).filter(([key]) => key !== 'hidden'),
  ),
});

function Seed() {
  useListQueryState(base).commit({ search: { keep: 1, hidden: 2 } });
  return null;
}
function Probe({
  sanitize,
  out,
}: {
  sanitize: boolean;
  out: { filters?: Filters };
}) {
  const state = useListQueryState({
    ...base,
    sanitizeRestored: sanitize ? dropHidden : undefined,
  });
  out.filters = state.initial.filters;
  return null;
}

describe('页签恢复剔除隐藏筛选值', () => {
  it('提供 sanitizeRestored 时剔除隐藏字段，未提供时原样恢复', () => {
    const plain: { filters?: Filters } = {};
    const clean: { filters?: Filters } = {};
    const { rerender } = render(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    rerender(
      <ListQueryStateProvider enabled>
        <Probe sanitize={false} out={plain} />
      </ListQueryStateProvider>,
    );
    expect(plain.filters?.search).toEqual({ keep: 1, hidden: 2 });
    const second = render(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    second.rerender(
      <ListQueryStateProvider enabled>
        <Probe sanitize out={clean} />
      </ListQueryStateProvider>,
    );
    expect(clean.filters?.search).toEqual({ keep: 1 });
  });
});
