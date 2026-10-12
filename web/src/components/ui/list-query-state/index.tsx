import { createContext, type ReactNode, useContext, useRef } from 'react';

export type ListQuerySnapshot<T> = {
  filters: T;
  labels: Record<string, string>;
  current: number;
  pageSize: number;
};
type Store = {
  snapshots: Map<string, Map<string, ListQuerySnapshot<unknown>>>;
  generations: Map<string, number>;
  enabled: boolean;
};
const Context = createContext<Store | undefined>(undefined);

/** 快照只属于当前工作区实例；身份不完整时禁止共享保存。 */
export function ListQueryStateProvider({
  enabled,
  children,
}: {
  enabled: boolean;
  children: ReactNode;
}) {
  const store = useRef<Store>({
    snapshots: new Map(),
    generations: new Map(),
    enabled,
  });
  store.current.enabled = enabled;
  return <Context.Provider value={store.current}>{children}</Context.Provider>;
}

export function useClearListQueries() {
  const store = useContext(Context);
  return (tabKeys: string[]) => {
    if (!store?.enabled) return;
    for (const tabKey of tabKeys) {
      store.snapshots.delete(tabKey);
      store.generations.set(tabKey, (store.generations.get(tabKey) ?? 0) + 1);
    }
  };
}

/** 显式列表身份在挂载期间固定；事件同步记录，卸载不回写。 */
export function useListQueryState<T>({
  tabKey,
  listId,
  defaultPageSize,
  defaultFilters,
}: {
  tabKey: string;
  listId: string;
  defaultPageSize: number;
  defaultFilters: T;
}) {
  const store = useContext(Context);
  const lease = useRef(store?.generations.get(tabKey) ?? 0);
  const initial = useRef(
    (store?.enabled ? store.snapshots.get(tabKey)?.get(listId) : undefined) as
      | ListQuerySnapshot<T>
      | undefined,
  );
  const snapshotRef = useRef<ListQuerySnapshot<T>>(
    initial.current ?? {
      filters: defaultFilters,
      labels: {},
      current: 1,
      pageSize: defaultPageSize,
    },
  );
  const persist = () => {
    if (
      !tabKey ||
      !listId ||
      !store?.enabled ||
      (store.generations.get(tabKey) ?? 0) !== lease.current
    )
      return;
    let lists = store.snapshots.get(tabKey);
    if (!lists) {
      lists = new Map();
      store.snapshots.set(tabKey, lists);
    }
    lists.set(listId, snapshotRef.current);
  };
  const commit = (
    filters: T,
    labels = snapshotRef.current.labels,
    resetPage = true,
  ) => {
    snapshotRef.current = {
      ...snapshotRef.current,
      filters,
      labels,
      current: resetPage ? 1 : snapshotRef.current.current,
    };
    persist();
  };
  const onPageChange = (current: number, pageSize: number) => {
    snapshotRef.current = { ...snapshotRef.current, current, pageSize };
    persist();
  };
  return {
    initial: initial.current ?? {
      filters: defaultFilters,
      labels: {},
      current: 1,
      pageSize: defaultPageSize,
    },
    snapshotRef,
    commit,
    pagination: {
      defaultCurrent: snapshotRef.current.current,
      defaultPageSize: snapshotRef.current.pageSize,
      onChange: onPageChange,
    },
  };
}
