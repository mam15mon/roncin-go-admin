import { Tag } from 'antd';
import React from 'react';

export type PartnerOptionContractData = {
  isCasual?: boolean;
  creditExceeded?: boolean;
};

/**
 * 往来单位选择器候选项的业务标注：按契约字段（isCasual / creditExceeded）
 * 动态渲染标签。label 只保留名称与编码，禁止把标注文本拼进 label，
 * 防止单证 / 合同字符串被污染。
 */
export function PartnerSelectOptionTags({
  data,
}: {
  data?: PartnerOptionContractData;
}) {
  if (!data?.isCasual && !data?.creditExceeded) {
    return null;
  }
  return (
    <span style={{ display: 'inline-flex', gap: 4, flexShrink: 0 }}>
      {data.isCasual ? (
        <Tag color="orange" style={{ marginInlineEnd: 0 }}>
          散客
        </Tag>
      ) : null}
      {data.creditExceeded ? (
        <Tag color="error" style={{ marginInlineEnd: 0 }}>
          已超信用额度
        </Tag>
      ) : null}
    </span>
  );
}

export default PartnerSelectOptionTags;
