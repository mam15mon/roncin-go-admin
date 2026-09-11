import { Card, Tree, Typography } from 'antd';
import type { DataNode } from 'antd/es/tree';
import React, { useMemo } from 'react';

const { Text } = Typography;

type BillGroupNavigatorProps = {
  groups: API.BillBatchPreviewGroup[];
  splitByTaxRate: boolean;
  splitByOrder: boolean;
  activeGroupKey?: string;
  invalidGroupKeys: Set<string>;
  onSelect: (groupKey: string) => void;
};

type NavigatorBranch = DataNode & { children?: NavigatorBranch[] };

function groupLeafTitle(group: API.BillBatchPreviewGroup, invalid: boolean) {
  const amount = `${group.totalAmount || '-'} ${group.currency || ''}`.trim();
  return `${invalid ? '⚠ ' : ''}${group.settlementPartyName || group.settlementPartyId || '未命名结算单位'} · ${amount}`;
}

/** 层级仅是服务端扁平叶子的导航投影，不参与任何金额或分组判断。 */
export default function BillGroupNavigator({
  groups,
  splitByTaxRate,
  splitByOrder,
  activeGroupKey,
  invalidGroupKeys,
  onSelect,
}: BillGroupNavigatorProps) {
  const { treeData, expandedKeys } = useMemo(() => {
    const root: NavigatorBranch[] = [];
    const expanded = new Set<React.Key>();
    for (const group of groups) {
      const groupKey = group.groupKey;
      if (!groupKey) continue;
      const levels = [
        `结算单位：${group.settlementPartyName || group.settlementPartyId || '-'}`,
        `币种：${group.currency || '未配置'}`,
        ...(splitByTaxRate ? [`税率：${group.taxRate ?? '未配置'}`] : []),
        ...(splitByOrder
          ? [`订单：${group.orderNo || group.orderId || '未配置'}`]
          : []),
      ];
      let children = root;
      let path = '';
      for (const level of levels) {
        path = `${path}/${level}`;
        expanded.add(path);
        let branch = children.find((item) => item.key === path);
        if (!branch) {
          branch = { key: path, title: level, children: [] };
          children.push(branch);
        }
        if (!branch.children) branch.children = [];
        children = branch.children;
      }
      children.push({
        key: groupKey,
        title: groupLeafTitle(group, invalidGroupKeys.has(groupKey)),
        isLeaf: true,
      });
    }
    return { treeData: root, expandedKeys: [...expanded] };
  }, [groups, invalidGroupKeys, splitByOrder, splitByTaxRate]);

  return (
    <Card size="small" title="拟生成账单导航" style={{ marginBottom: 16 }}>
      <Text type="secondary">
        结算单位和费用币种为固定边界；仅展示当前服务端预览叶子，带 ⚠
        的叶子尚有必填配置缺失。
      </Text>
      <Tree
        blockNode
        expandedKeys={expandedKeys}
        selectedKeys={activeGroupKey ? [activeGroupKey] : []}
        treeData={treeData}
        onSelect={(keys, info) => {
          if (!info.node.isLeaf) return;
          const groupKey = String(keys[0] || '');
          if (groupKey) onSelect(groupKey);
        }}
      />
    </Card>
  );
}
