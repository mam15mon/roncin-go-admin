import { WarningOutlined } from '@ant-design/icons';
import { Card, Flex, Tag, Tooltip, Tree, Typography } from 'antd';
import type { DataNode } from 'antd/es/tree';
import React, { useMemo, useState } from 'react';
import { formatAmount, trimDecimal } from '@/utils/format';

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

function groupLeafTitle(
  group: API.BillBatchPreviewGroup,
  invalid: boolean,
  splitByTaxRate: boolean,
  splitByOrder: boolean,
) {
  const amount =
    `${formatAmount(group.totalAmount)} ${group.currency || ''}`.trim();
  const fullTitle = `${group.settlementPartyName || group.settlementPartyId || '未命名结算单位'} · ${amount}`;
  return (
    <Flex
      align="center"
      wrap
      gap={8}
      style={{ width: '100%' }}
      title={fullTitle}
    >
      {invalid && (
        <Tooltip title="必填配置未完成，点击后在下方补全">
          <WarningOutlined style={{ color: '#d46b08' }} />
        </Tooltip>
      )}
      {splitByOrder && (
        <Tag style={{ margin: 0 }}>
          订单 {group.orderNo || group.orderId || '未配置'}
        </Tag>
      )}
      {splitByTaxRate && (
        <Tag style={{ margin: 0 }}>
          税率 {group.taxRate ? `${trimDecimal(group.taxRate)}%` : '未配置'}
        </Tag>
      )}
      {!splitByOrder && !splitByTaxRate && <Text>全部费用</Text>}
      <Text
        strong
        style={{
          marginLeft: 'auto',
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {amount}
      </Text>
    </Flex>
  );
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
  // 记录用户主动收起的节点；新出现的节点默认展开。
  const [collapsedKeys, setCollapsedKeys] = useState<Set<React.Key>>(new Set());
  const { treeData, allKeys } = useMemo(() => {
    const root: NavigatorBranch[] = [];
    const expanded = new Set<React.Key>();
    for (const group of groups) {
      const groupKey = group.groupKey;
      if (!groupKey) continue;
      const levels = [
        group.settlementPartyName ||
          group.settlementPartyId ||
          '未命名结算单位',
        group.currency || '币种未配置',
      ];
      let children = root;
      let path = '';
      for (const level of levels) {
        path = `${path}/${level}`;
        expanded.add(path);
        let branch = children.find((item) => item.key === path);
        if (!branch) {
          branch = {
            key: path,
            title: <Text strong={children === root}>{level}</Text>,
            children: [],
          };
          children.push(branch);
        }
        if (!branch.children) branch.children = [];
        children = branch.children;
      }
      children.push({
        key: groupKey,
        title: groupLeafTitle(
          group,
          invalidGroupKeys.has(groupKey),
          splitByTaxRate,
          splitByOrder,
        ),
        isLeaf: true,
      });
    }
    return { treeData: root, allKeys: [...expanded] };
  }, [groups, invalidGroupKeys, splitByOrder, splitByTaxRate]);

  return (
    <Card size="small" title="账单列表" style={{ marginBottom: 12 }}>
      <Tree
        blockNode
        expandedKeys={allKeys.filter((key) => !collapsedKeys.has(key))}
        onExpand={(keys) => {
          const open = new Set(keys);
          setCollapsedKeys(new Set(allKeys.filter((key) => !open.has(key))));
        }}
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
