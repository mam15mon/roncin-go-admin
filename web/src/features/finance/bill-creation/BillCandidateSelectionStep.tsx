import type { ProColumns } from '@ant-design/pro-components';
import { paginationSizeChanger } from '@root/config/pagination';
import { Alert, Card } from 'antd';
import React from 'react';
import { ResponsiveProTable as ProTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { settlementServiceListBillCreationCandidates } from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { selectionFeeColumns } from './billWorkbenchFeeColumns';

type BillCandidateSelectionStepProps = {
  fixedSelection: boolean;
  sourceLabel?: string;
  selectedIds: string[];
  selectedFeeIds: React.Key[];
  onSelectedFeeIdsChange: React.Dispatch<React.SetStateAction<React.Key[]>>;
  organizationId?: string;
  canOperate: boolean;
};

/** 建账工作台第 1 步：固定费用带入提示或候选费用列表 */
export default function BillCandidateSelectionStep({
  fixedSelection,
  sourceLabel,
  selectedIds,
  selectedFeeIds,
  onSelectedFeeIdsChange,
  organizationId,
  canOperate,
}: BillCandidateSelectionStepProps) {
  const candidateSettings = useColumnSettings<ProColumns<API.FeeLedgerItem>>({
    tableKey: 'finance:bill-candidate-fees',
    columns: selectionFeeColumns,
  });

  return fixedSelection ? (
    <Card>
      <Alert
        type="info"
        showIcon
        title={`已从${sourceLabel || '业务页面'}带入 ${selectedIds.length} 笔未建账费用`}
        description="仅已批准且没有待审核补录的费用可以转账单，生成前会再核对一次。"
      />
    </Card>
  ) : (
    <ProTable<API.FeeLedgerItem>
      key={organizationId || 'no-organization'}
      rowKey="id"
      headerTitle="选择待结算费用"
      columns={candidateSettings.columns}
      size="small"
      bordered
      options={{ reload: true, density: true, setting: false }}
      toolBarRender={() => [candidateSettings.entry]}
      pagination={{
        defaultPageSize: 15,
        showSizeChanger: paginationSizeChanger,
      }}
      rowSelection={{
        selectedRowKeys: selectedFeeIds,
        preserveSelectedRowKeys: true,
        onChange: onSelectedFeeIdsChange,
        getCheckboxProps: (record) => {
          // 候选接口已按有效账单关联过滤未建账费用；billNo 为空作为展示层兜底，
          // 不在前端重建第二套状态判定。
          const isSelectable =
            canOperate &&
            record.organizationId === organizationId &&
            record.canBill === true;
          return {
            disabled: !isSelectable,
            title: !isSelectable
              ? record.blockedReasons?.join('；') || '费用尚不符合建账条件'
              : undefined,
          };
        },
      }}
      tableAlertRender={false}
      request={async (params) => {
        if (!canOperate || !organizationId)
          return { data: [], success: false, total: 0 };
        const response = await settlementServiceListBillCreationCandidates({
          page: params.current,
          pageSize: params.pageSize,
          keyword: params.keyword,
          direction: params.direction,
          organizationId,
        });
        return toTableRequest(response);
      }}
    />
  );
}
