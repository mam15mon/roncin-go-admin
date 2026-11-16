import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { useQueryClient } from '@tanstack/react-query';
import { screen } from '@testing-library/react';
import { createContext, useContext } from 'react';
import { describe, expect, it } from 'vitest';

const Context = createContext('missing');
function Probe({ label }: { label: string }) {
  const client = useQueryClient();
  const value = useContext(Context);
  return (
    <output>
      {label}:{String(client.getQueryData(['identity']))}:{value}
    </output>
  );
}

describe('renderWithClient', () => {
  it('重新渲染沿用同一客户端并组合外部 wrapper，另一个用例实例独立', () => {
    const first = renderWithClient(<Probe label="初次" />, {
      wrapper: ({ children }) => (
        <Context.Provider value="外层上下文">{children}</Context.Provider>
      ),
    });
    first.queryClient.setQueryData(['identity'], '当前实例');
    first.rerender(<Probe label="再次" />);
    expect(screen.getByText('再次:当前实例:外层上下文')).toBeInTheDocument();
    first.unmount();
    const second = renderWithClient(<Probe label="独立" />);
    expect(second.queryClient).not.toBe(first.queryClient);
    expect(screen.getByText('独立:undefined:missing')).toBeInTheDocument();
  });
});
