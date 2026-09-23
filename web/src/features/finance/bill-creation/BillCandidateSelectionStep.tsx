import { ProTable } from '@ant-design/pro-components';
import { Alert, Card, Select, Typography } from 'antd';
import React from 'react';
import { OrderFeeStatus } from '@/enums.generated';
import { settlementServiceListBillCreationCandidates } from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { selectionFeeColumns } from './billWorkbenchFeeColumns';

const { Text } = Typography;

type BillCandidateSelectionStepProps = {
  fixedSelection: boolean;
  sourceLabel?: string;
  selectedIds: string[];
  selectedFeeIds: React.Key[];
  onSelectedFeeIdsChange: React.Dispatch<React.SetStateAction<React.Key[]>>;
  initialOrganizationId?: string;
  initialOrganizationName?: string;
  organizationId?: string;
  organizationOptions: API.FinanceOrganizationOption[];
  onOrganizationChange: (value: string | undefined) => void;
};

/** 建账工作台第 1 步：固定费用带入提示或候选费用列表 */
export default function BillCandidateSelectionStep({
  fixedSelection,
  sourceLabel,
  selectedIds,
  selectedFeeIds,
  onSelectedFeeIdsChange,
  initialOrganizationId,
  initialOrganizationName,
  organizationId,
  organizationOptions,
  onOrganizationChange,
}: BillCandidateSelectionStepProps) {
  return fixedSelection ? (
    <Card>
      <Alert
        type="info"
        showIcon
        title={`已从${sourceLabel || '业务页面'}带入 ${selectedIds.length} 笔未建账费用`}
        description="费用状态、结算维度和金额快照将在预览及最终建单事务中由服务端再次校验。"
      />
      <div style={{ marginTop: 12 }}>
        来源公司：
        {initialOrganizationName ||
          organizationOptions.find((item) => item.id === initialOrganizationId)
            ?.name ||
          initialOrganizationId ||
          '-'}
      </div>
    </Card>
  ) : (
    <>
      <Select
        aria-label="所属公司"
        allowClear
        placeholder="请先选择可建账所属公司"
        style={{ width: 260, marginBottom: 12 }}
        value={organizationId}
        options={organizationOptions.map((item) => ({
          value: item.id,
          label: item.name || item.code || item.id,
        }))}
        onChange={(value) => onOrganizationChange(value)}
      />
      <ProTable<API.FeeLedgerItem>
        key={organizationId || 'no-organization'}
        rowKey="id"
        headerTitle="选择待结算费用"
        columns={selectionFeeColumns}
        size="small"
        bordered
        options={false}
        pagination={{ defaultPageSize: 15, showSizeChanger: true }}
        rowSelection={{
          selectedRowKeys: selectedFeeIds,
          preserveSelectedRowKeys: true,
          onChange: onSelectedFeeIdsChange,
          getCheckboxProps: (record) => {
            const isSelectable =
              record.status === OrderFeeStatus.ORDER_FEE_STATUS_UNBILLED &&
              !record.billNo;
            return {
              disabled: !isSelectable,
              title: !isSelectable
                ? record.billNo
                  ? `已进入账单 ${record.billNo}`
                  : '只有未建账且未入账单的费用方可创建账单'
                : undefined,
            };
          },
        }}
        tableAlertRender={({ selectedRowKeys }) => (
          <Text>已选择 {selectedRowKeys.length} 笔未建账费用</Text>
        )}
        request={async (params) => {
          if (!organizationId) return { data: [], success: true, total: 0 };
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
    </>
  );
}
