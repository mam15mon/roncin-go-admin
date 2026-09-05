import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDependency,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { Alert, App, Button, Descriptions, Space, Table, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { FinanceVerificationStatus } from '@/enums.generated';
import {
  settlementServiceCreateCommission,
  settlementServiceListCommissionCandidates,
  settlementServiceListCommissionRules,
  settlementServiceListVerifications,
  settlementServicePreviewCommission,
} from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';
import { generateUUID } from '@/utils/uuid';
import {
  calculationBasisText,
  calculationSignature,
  cnyExchangeRateSourceText,
  decimalText,
  personnelRoleText,
  type CreateValues,
} from '../types';
import { previewColumns, renderExpandedFees } from './CommissionLineTable';

type CommissionCreateModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
};

export default function CommissionCreateModal({
  open,
  onOpenChange,
  onSuccess,
}: CommissionCreateModalProps) {
  const { message } = App.useApp();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [preview, setPreview] = useState<API.CommissionCalculation>();
  const [previewSignature, setPreviewSignature] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);
  const [createIdempotencyKey, setCreateIdempotencyKey] = useState(() =>
    generateUUID(),
  );

  return (
    <ModalForm<CreateValues>
      formRef={formRef}
      title="生成提成"
      open={open}
      width={980}
      submitter={{
        searchConfig: { submitText: '生成草稿' },
        submitButtonProps: { disabled: !preview },
      }}
      modalProps={{
        destroyOnHidden: true,
        onCancel: () => {
          onOpenChange(false);
          setPreview(undefined);
          setPreviewSignature('');
        },
      }}
      onValuesChange={(changedValues) => {
        if (
          'verificationId' in changedValues ||
          'ruleId' in changedValues ||
          'employeeId' in changedValues
        ) {
          setPreview(undefined);
          setPreviewSignature('');
          setCreateIdempotencyKey(generateUUID());
        }
      }}
      onFinish={async (values) => {
        if (!preview || previewSignature !== calculationSignature(values)) {
          message.warning('请先计算并核对当前选择的提成预览');
          return false;
        }
        try {
          await settlementServiceCreateCommission({
            verificationId: values.verificationId,
            employeeId: values.employeeId,
            ruleId: values.ruleId,
            note: values.note,
            idempotencyKey: createIdempotencyKey,
          });
          message.success('提成草稿已生成，列表已按创建结果刷新');
          onOpenChange(false);
          setPreview(undefined);
          setPreviewSignature('');
          onSuccess();
          return true;
        } catch (error: any) {
          message.error(error.message || '提成生成失败');
          return false;
        }
      }}
    >
      <ProFormSearchableSelect
        name="verificationId"
        label="有效应收核销"
        rules={[{ required: true, message: '请选择有效应收核销单' }]}
        request={async () => {
          const response = await settlementServiceListVerifications({
            page: 1,
            pageSize: 200,
            status:
              FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE,
          });
          return unwrapList(response)
            .filter((item) => item.direction === 'RECEIVABLE')
            .map((item) => ({
              label: `${item.verificationNo}｜${item.settlementPartyName}｜${item.amount} ${item.currency}`,
              value: item.id,
            }));
        }}
      />
      <ProFormSearchableSelect
        name="ruleId"
        label="考核规则"
        rules={[{ required: true, message: '请选择考核规则' }]}
        request={async () => {
          const response = await settlementServiceListCommissionRules({
            page: 1,
            pageSize: 200,
            enabled: true,
          });
          return unwrapList(response).map((item) => ({
            label: `${item.name}｜${personnelRoleText(item.personnelRole)}｜${calculationBasisText(item.calculationBasis)} × ${decimalText(item.ratePercent)}%`,
            value: item.id,
          }));
        }}
      />
      <ProFormDependency name={['verificationId', 'ruleId']}>
        {({ verificationId, ruleId }) => (
          <ProFormSearchableSelect
            key={`${verificationId || ''}-${ruleId || ''}`}
            name="employeeId"
            label="符合规则的候选人员"
            rules={[{ required: true, message: '请选择符合角色的候选人员' }]}
            disabled={!verificationId || !ruleId}
            request={async () => {
              if (!verificationId || !ruleId) return [];
              const response = await settlementServiceListCommissionCandidates({
                verificationId,
                ruleId,
                page: 1,
                pageSize: 200,
              });
              return unwrapList(response).map((item) => ({
                label: `${item.employeeName}｜${item.customerCount ?? 0}个客户｜${item.orderCount ?? 0}票订单｜预计 ${decimalText(item.commissionAmount)} ${item.baseCurrency}`,
                value: item.employeeId,
              }));
            }}
            extra="人员来自本次核销涉及订单在创建时固化的业务、操作或客服归属；客户后续换人不会改变历史订单归属。"
          />
        )}
      </ProFormDependency>
      <ProFormTextArea
        name="note"
        label="备注"
        fieldProps={{ maxLength: 500 }}
      />
      <ProFormDependency name={['verificationId', 'ruleId', 'employeeId']}>
        {(values: Partial<CreateValues>) => (
          <Space vertical size={12} style={{ width: '100%' }}>
            <Button
              type="primary"
              ghost
              loading={previewLoading}
              disabled={
                !values.verificationId || !values.ruleId || !values.employeeId
              }
              onClick={async () => {
                const { verificationId, employeeId, ruleId } = values;
                if (!verificationId || !employeeId || !ruleId) return;
                try {
                  setPreviewLoading(true);
                  const response = await settlementServicePreviewCommission({
                    verificationId,
                    employeeId,
                    ruleId,
                  });
                  setPreview(response.data);
                  setPreviewSignature(calculationSignature(values));
                } catch (error: any) {
                  setPreview(undefined);
                  setPreviewSignature('');
                  message.error(error.message || '提成预览计算失败');
                } finally {
                  setPreviewLoading(false);
                }
              }}
            >
              计算并核对预览
            </Button>
            {!preview ? (
              <Alert
                type="info"
                showIcon
                title="生成前必须计算预览"
                description="系统会按本次核销涉及的订单，分别展示已实现收入、分摊成本、毛利和提成金额，并支持下钻展开费用明细。"
              />
            ) : (
              <>
                <Descriptions
                  size="small"
                  bordered
                  column={4}
                  items={[
                    {
                      key: 'employee',
                      label: '提成员工',
                      children: preview.employeeName,
                    },
                    {
                      key: 'rule',
                      label: '规则',
                      children: `${preview.ruleName}（v${preview.ruleVersion}）`,
                    },
                    {
                      key: 'basis',
                      label: '角色/口径',
                      children: `${personnelRoleText(preview.personnelRole)} · ${calculationBasisText(preview.calculationBasis)}`,
                    },
                    {
                      key: 'rate',
                      label: '比例',
                      children: `${decimalText(preview.ratePercent)}%`,
                    },
                    {
                      key: 'coverage',
                      label: '业务覆盖',
                      children: `${preview.customerCount || 1} 个客户 · ${preview.orderCount || 1} 票订单 · ${preview.feeCount || 0} 笔费用`,
                    },
                    {
                      key: 'revenue',
                      label: '已实现收入',
                      children: `${decimalText(preview.realizedRevenue)} ${preview.baseCurrency}`,
                    },
                    {
                      key: 'cost',
                      label: '分摊成本',
                      children: `${decimalText(preview.allocatedCost)} ${preview.baseCurrency}`,
                    },
                    {
                      key: 'profit',
                      label: '已实现毛利',
                      children: `${decimalText(preview.realizedProfit)} ${preview.baseCurrency}`,
                    },
                    {
                      key: 'amount',
                      label: '提成金额',
                      children: (
                        <Typography.Text strong type="success">
                          {`${decimalText(preview.commissionAmount)} ${preview.baseCurrency}`}
                        </Typography.Text>
                      ),
                    },
                    {
                      key: 'cnyAmount',
                      label: '提成金额（CNY）',
                      children: (
                        <Typography.Text strong type="success">
                          {`${decimalText(preview.cnyCommissionAmount)} CNY`}
                        </Typography.Text>
                      ),
                    },
                    {
                      key: 'cnyRate',
                      label: 'CNY 折算率',
                      children: decimalText(preview.cnyExchangeRate),
                    },
                    {
                      key: 'cnyRateDate',
                      label: 'CNY 汇率日期',
                      children: preview.cnyExchangeRateDate || '-',
                    },
                    {
                      key: 'cnyRateSource',
                      label: 'CNY 汇率来源',
                      children: cnyExchangeRateSourceText(
                        preview.cnyExchangeRateSource,
                      ),
                    },
                  ]}
                />
                <Alert
                  type="warning"
                  showIcon
                  title="预览汇率仅供生成前核对"
                  description="创建草稿时会在事务内重新解析 CNY 汇率；最终折算依据和金额以创建结果及刷新后的列表为准。"
                />
                <Table<API.FinanceCommissionLine>
                  size="small"
                  bordered
                  pagination={false}
                  rowKey={(line) => line.orderId || line.orderNo || ''}
                  columns={previewColumns}
                  dataSource={preview.lines || []}
                  scroll={{ x: 1080 }}
                  expandable={{
                    expandedRowRender: renderExpandedFees,
                    rowExpandable: (record) =>
                      Boolean(record.fees && record.fees.length > 0),
                  }}
                />
              </>
            )}
          </Space>
        )}
      </ProFormDependency>
      <Space vertical size={2} style={{ color: '#666', marginTop: 8 }}>
        <span>
          计算比例、角色与口径均取自已启用且在核销日期生效的考核规则。
        </span>
        <span>亏损订单逐票按 0 计提，但仍保留真实负毛利快照。</span>
        <span>草稿确认时会重新校验客户人员与费用来源；来源变化后必须取消并重新生成。</span>
      </Space>
    </ModalForm>
  );
}
