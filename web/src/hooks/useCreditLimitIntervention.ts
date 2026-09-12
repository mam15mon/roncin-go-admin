import { useEffect, useState } from 'react';
import { settlementServiceGetCreditLimitControlPolicy } from '@/services/roncin/settlementService';

/**
 * 读取组织信用额度管控策略并返回是否处于「直接干预」模式。
 *
 * - 仅当策略已保存且「超额后允许选择」为 false 时返回 true；该布尔为 false 时
 *   JSON 序列化会整体省略，必须用 `!== true` 判定，禁止 `=== false` 死分支。
 * - 默认（未保存策略 / 读取失败 / 未激活）返回 false：直接干预是显式管理动作，
 *   读取失败时回退仅提醒模式，不阻断任何选择。
 */
export function useCreditLimitIntervention(enabled = true): boolean {
  const [interventionActive, setInterventionActive] = useState(false);
  useEffect(() => {
    if (!enabled) {
      setInterventionActive(false);
      return undefined;
    }
    let cancelled = false;
    settlementServiceGetCreditLimitControlPolicy({})
      .then((response) => {
        if (cancelled) return;
        setInterventionActive(
          response.data?.allowSelectionWhenCreditExceeded !== true,
        );
      })
      .catch(() => {
        if (!cancelled) setInterventionActive(false);
      });
    return () => {
      cancelled = true;
    };
  }, [enabled]);
  return interventionActive;
}
