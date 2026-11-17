import { ReloadOutlined, SettingOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd';
import {
  Button,
  Card,
  Col,
  Form,
  Row,
  Space,
  Switch,
  Tag,
  Typography,
} from 'antd';
import React from 'react';
import BillBatchSummary from './BillBatchSummary';
import BillGroupCard from './BillGroupCard';
import BillGroupNavigator from './BillGroupNavigator';
import { getPreviewFeeColumns } from './billWorkbenchFeeColumns';
import type {
  BillCreationMode,
  GroupFormValue,
  WorkbenchFormValue,
} from './billWorkbenchHelpers';
import { directionText } from './billWorkbenchHelpers';
import NettingPairsCard from './NettingPairsCard';

const { Text } = Typography;

type BillGroupConfigurationStepProps = {
  mode: BillCreationMode;
  form: FormInstance<WorkbenchFormValue>;
  preview: API.PreviewBillBatchResponse;
  groupingMode: API.BillGroupingPolicy['mode'];
  organizationId?: string;
  sessionIdentity: string;
  loading: boolean;
  frozen?: boolean;
  splitByOrder: boolean;
  splitByTaxRate: boolean;
  onSplitByOrderChange: React.Dispatch<React.SetStateAction<boolean>>;
  onSplitByTaxRateChange: React.Dispatch<React.SetStateAction<boolean>>;
  activeGroupKey?: string;
  onActiveGroupKeyChange: React.Dispatch<
    React.SetStateAction<string | undefined>
  >;
  invalidGroupKeys: Set<string>;
  invalidatePreview: () => void;
  loadPreview: (
    overrideIds?: string[],
    policyOverride?: API.BillGroupingPolicy,
    organizationIdOverride?: string,
  ) => Promise<boolean>;
  onRemoveFee: (feeId?: string) => void;
  onConfigurationChange: () => void;
  onUserEdit?: () => void;
};

/** 建账工作台第 3 步：拆分方式、汇总、对冲配对，左侧账单列表、右侧当前账单资料 */
export default function BillGroupConfigurationStep({
  mode,
  form,
  preview,
  groupingMode,
  organizationId,
  sessionIdentity,
  loading,
  frozen,
  splitByOrder,
  splitByTaxRate,
  onSplitByOrderChange,
  onSplitByTaxRateChange,
  activeGroupKey,
  onActiveGroupKeyChange,
  invalidGroupKeys,
  invalidatePreview,
  loadPreview,
  onRemoveFee,
  onConfigurationChange,
  onUserEdit,
}: BillGroupConfigurationStepProps) {
  const groups = preview.data ?? [];

  return (
    <Form
      form={form}
      disabled={frozen}
      layout="vertical"
      onValuesChange={(changedValues) => {
        onUserEdit?.();
        if (!changedValues?.groups) return;
        const shouldRefresh = Object.values(changedValues.groups).some(
          (groupValue: Partial<GroupFormValue>) => {
            if (!groupValue || typeof groupValue !== 'object') return false;
            return (
              'settlementAccountId' in groupValue ||
              'estimatedInvoiceCurrency' in groupValue ||
              'estimatedInvoiceRate' in groupValue
            );
          },
        );
        if (shouldRefresh) {
          onConfigurationChange();
        }
      }}
    >
      <Card
        size="small"
        style={{
          marginBottom: 16,
          background: '#fafafa',
          border: '1px solid #f0f0f0',
        }}
      >
        <Row justify="space-between" align="middle" gutter={[16, 8]}>
          <Col xs={24} md={16}>
            <Space size="large" wrap>
              <Space>
                <SettingOutlined style={{ color: '#1677ff' }} />
                <Text strong>拆分方式：</Text>
              </Space>
              <Space>
                <Text type="secondary">按币种：</Text>
                <Tag color="blue">固定启用</Tag>
              </Space>
              <Space>
                <Text type="secondary">按订单：</Text>
                <Switch
                  size="small"
                  checked={splitByOrder}
                  onChange={async (checked) => {
                    onSplitByOrderChange(checked);
                    invalidatePreview();
                    await loadPreview(undefined, {
                      mode: groupingMode,
                      splitByOrder: checked,
                      splitByTaxRate,
                    });
                  }}
                />
              </Space>
              <Space>
                <Text type="secondary">按税率：</Text>
                <Switch
                  size="small"
                  checked={splitByTaxRate}
                  onChange={async (checked) => {
                    onSplitByTaxRateChange(checked);
                    invalidatePreview();
                    await loadPreview(undefined, {
                      mode: groupingMode,
                      splitByOrder,
                      splitByTaxRate: checked,
                    });
                  }}
                />
              </Space>
            </Space>
          </Col>
          <Col xs={24} md={8} style={{ textAlign: 'right' }}>
            <Space>
              <Tag color="blue">共 {groups.length} 张账单</Tag>
              <Button
                size="small"
                icon={<ReloadOutlined />}
                loading={loading}
                disabled={frozen}
                onClick={() => void loadPreview()}
              >
                刷新账单
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>
      <BillBatchSummary
        groups={groups}
        incompleteCount={invalidGroupKeys.size}
      />
      {mode === 'NETTING' && (
        <NettingPairsCard pairs={preview.nettingPairs || []} />
      )}
      <Row gutter={16} align="top">
        <Col xs={24} lg={8} xl={7}>
          <div style={{ position: 'sticky', top: 0 }}>
            <BillGroupNavigator
              groups={groups}
              splitByTaxRate={splitByTaxRate}
              splitByOrder={splitByOrder}
              activeGroupKey={activeGroupKey}
              invalidGroupKeys={invalidGroupKeys}
              onSelect={onActiveGroupKeyChange}
            />
          </div>
        </Col>
        <Col xs={24} lg={16} xl={17}>
          {groups.map((group) => {
            const isCurrent = group.groupKey === activeGroupKey;
            return (
              <div
                key={group.groupKey}
                style={{ display: isCurrent ? 'block' : 'none' }}
              >
                <BillGroupCard
                  group={group}
                  organizationId={organizationId || ''}
                  sessionIdentity={sessionIdentity}
                  feeColumns={getPreviewFeeColumns(
                    frozen ? undefined : onRemoveFee,
                  )}
                  directionText={directionText}
                  onConfigurationChange={onConfigurationChange}
                />
              </div>
            );
          })}
        </Col>
      </Row>
    </Form>
  );
}
