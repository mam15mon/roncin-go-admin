/**
 * SE 表单隐藏区域错误定位前置链路。
 *
 * 提单页签与折叠备注中的字段平时不可见，但必须参与校验（forceRender /
 * display:none 常驻挂载）。提交校验失败时，通用模板会先调用
 * `revealSeaFormErrors` 通知各隐藏区域把首个错误字段变为可见，再执行
 * 滚动与聚焦；本模块是两侧之间的最小发布订阅桥，避免通用模板反向
 * 依赖 SE 页面模块。
 */

export interface SeaFormErrorField {
  name: (string | number)[];
  errors?: string[];
}

type RevealHandler = (fields: SeaFormErrorField[]) => void;

const handlers = new Set<RevealHandler>();

/** 注册隐藏区域监听；返回取消注册函数，供组件卸载时清理。 */
export function onSeaFormErrorReveal(handler: RevealHandler): () => void {
  handlers.add(handler);
  return () => {
    handlers.delete(handler);
  };
}

/** 按首个错误字段通知所有已注册隐藏区域切换页签或展开折叠备注。 */
export function revealSeaFormErrors(fields: SeaFormErrorField[]): void {
  for (const handler of handlers) {
    handler(fields);
  }
}

/** 判断错误字段是否落在指定前缀下（如 ['seaHouseBill', ...]）。 */
export function errorFieldUnderPrefix(
  fields: SeaFormErrorField[],
  prefix: string,
): boolean {
  return fields.some((field) => field.name[0] === prefix);
}
