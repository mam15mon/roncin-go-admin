// 研究实验：使用仓库实际安装的 ProTable useFetchData 验证 request 副作用不能被取消保护。
import { act, renderHook, waitFor } from '/home/vince/roncin-go-admin/web/node_modules/@testing-library/react/dist/index.js';
import { useState } from '/home/vince/roncin-go-admin/web/node_modules/react/index.js';
import { expect, it } from '/home/vince/roncin-go-admin/web/node_modules/vitest/dist/index.js';
import useFetchData from '/home/vince/roncin-go-admin/web/node_modules/@ant-design/pro-components/es/table/useFetchData.js';

it('旧请求已被表格取消，但 request 内的指标副作用仍覆盖新指标', async () => {
  const resolvers = [];
  const accepted = [];
  const getData = () => new Promise((resolve) => resolvers.push(resolve));
  const { result, rerender } = renderHook(({ filter }) => {
    const [metric, setMetric] = useState('initial');
    const action = useFetchData(async () => {
      const response = await getData();
      setMetric(response.summary);
      return response;
    }, [], { effects: [filter], debounceTime: 1, onLoad: (rows) => accepted.push(rows) });
    return { action, metric };
  }, { initialProps: { filter: 'A' } });
  await waitFor(() => expect(resolvers).toHaveLength(1));
  rerender({ filter: 'B' });
  await waitFor(() => expect(resolvers).toHaveLength(2));
  await act(async () => resolvers[1]({ data: [{ id: 'B' }], success: true, total: 1, summary: 'B-200' }));
  await waitFor(() => expect(result.current.action.dataSource).toEqual([{ id: 'B' }]));
  expect(result.current.metric).toBe('B-200');
  await act(async () => resolvers[0]({ data: [{ id: 'A' }], success: true, total: 1, summary: 'A-100' }));
  await waitFor(() => expect(result.current.metric).toBe('A-100'));
  expect(result.current.action.dataSource).toEqual([{ id: 'B' }]);
  expect(accepted).toEqual([[{ id: 'B' }]]);
});
