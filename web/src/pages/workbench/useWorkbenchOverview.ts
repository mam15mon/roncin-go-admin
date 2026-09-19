import { useCallback, useEffect, useRef, useState } from 'react';
import { workbenchServiceGetWorkbenchOverview } from '@/services/roncin/workbenchService';

export type WorkbenchOverviewState = {
  loading: boolean;
  data?: API.GetWorkbenchOverviewData;
  error: boolean;
};

const INITIAL_STATE: WorkbenchOverviewState = {
  loading: true,
  data: undefined,
  error: false,
};

/**
 * 工作台 Overview 数据 Hook。
 *
 * 组织切换契约：
 * - organizationId 进入依赖数组，切换时整体重建状态（不残留上一组织的金额）；
 * - 每次发起请求前递增单调序号，清理函数再递增一次，使旧组织在途响应
 *   （成功或失败）因序号失配而被丢弃，迟到的旧组织响应不会覆盖当前页面。
 *
 * reload 返回 Promise：解析在本次刷新请求落定（成功或失败）之后，
 * 供月度申请提交等调用方「等待刷新完成再解除 loading」。
 */
export function useWorkbenchOverview(organizationId?: string) {
  const [state, setState] = useState<WorkbenchOverviewState>(INITIAL_STATE);
  const [reloadToken, setReloadToken] = useState(0);
  const sequenceRef = useRef(0);
  const reloadResolveRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    const sequence = ++sequenceRef.current;
    setState({ ...INITIAL_STATE });
    workbenchServiceGetWorkbenchOverview()
      .then((response) => {
        if (sequence !== sequenceRef.current) return;
        setState({
          loading: false,
          data: response.data,
          error: false,
        });
      })
      .catch(() => {
        if (sequence !== sequenceRef.current) return;
        setState({ loading: false, data: undefined, error: true });
      })
      .finally(() => {
        // 只有仍是最新一次请求时才释放等待方；被 supersede 的刷新由
        // reload() 自身或后续请求落定释放，不会永久挂起。
        if (sequence === sequenceRef.current) {
          reloadResolveRef.current?.();
          reloadResolveRef.current = null;
        }
      });
    return () => {
      // 失效在途响应：组织切换、重载与卸载后，旧响应一律丢弃。
      sequenceRef.current += 1;
    };
  }, [organizationId, reloadToken]);

  // 卸载时释放仍在等待的 reload promise，避免调用方永久挂起。
  useEffect(
    () => () => {
      reloadResolveRef.current?.();
      reloadResolveRef.current = null;
    },
    [],
  );

  const reload = useCallback(
    () =>
      new Promise<void>((resolve) => {
        // 连续 reload：前一个等待立即释放，由最新一次刷新接管。
        reloadResolveRef.current?.();
        reloadResolveRef.current = resolve;
        setReloadToken((token) => token + 1);
      }),
    [],
  );

  return { ...state, reload };
}
