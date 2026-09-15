import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDependency,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Form,
  Input,
  Space,
  Table,
  Tabs,
  Typography,
} from 'antd';
import type { TableProps } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import {
  settlementServiceCreateCommission,
  settlementServiceListCommissionCandidates,
  settlementServiceListCommissionEmployees,
  settlementServiceListCommissionNettingCandidates,
  settlementServiceListCommissionRuleCandidates,
  settlementServiceListCommissionVerificationCandidates,
  settlementServiceListFinanceOrganizationOptions,
  settlementServicePreviewCommission,
} from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import {
  type CreateValues,
  calculationBasisText,
  calculationSignature,
  cnyExchangeRateSourceText,
  commissionSourceNo,
  decimalText,
  personnelRoleText,
} from '../types';
import { previewColumns, renderExpandedFees } from './CommissionLineTable';

type CommissionCreateModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
};

/** 提成来源 Tab：核销提成走核销单，对冲提成走已确认对冲单。 */
type CommissionSourceTab = 'verification' | 'netting';

const nettingCandidateColumns: TableProps<API.FinanceNetting>['columns'] = [
  { title: '对冲单号', dataIndex: 'nettingNo', width: 170 },
  { title: '结算单位', dataIndex: 'settlementPartyName', ellipsis: true },
  { title: '币种', dataIndex: 'currency', width: 70 },
  {
    title: '抵销金额',
    dataIndex: 'amount',
    width: 130,
    align: 'right',
    render: (value: string, record) =>
      `${value ?? '0'} ${record.currency ?? ''}`,
  },
  {
    title: '确认时间',
    dataIndex: 'confirmedAt',
    width: 160,
    render: (value: string) => formatDate(value),
  },
];

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
  const [organizationId, setOrganizationId] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [sourceType, setSourceType] = useState<CommissionSourceTab>(
    'verification',
  );
  const [nettingCandidates, setNettingCandidates] = useState<
    API.FinanceNetting[]
  >([]);
  const [nettingLoading, setNettingLoading] = useState(false);
  const [nettingKeyword, setNettingKeyword] = useState('');
  const [selectedNetting, setSelectedNetting] = useState<API.FinanceNetting>();
  const [verificationOptions, setVerificationOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [verificationLoading, setVerificationLoading] = useState(false);
  const [verificationKeyword, setVerificationKeyword] = useState('');
  // 候选请求序号：只允许最新一次请求写入候选，防止组织/关键字切换后的迟到响应污染。
  const verificationRequestRef = useRef(0);
  const nettingRequestRef = useRef(0);

  const resetPreview = () => {
    setPreview(undefined);
    setPreviewSignature('');
    setCreateIdempotencyKey(generateUUID());
  };

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_COMMISSION_MANAGE,
    })
      .then((response) => {
        if (!cancelled) setOrganizationOptions(response.data ?? []);
      })
      .catch(() => {
        if (!cancelled) message.warning('提成创建公司候选加载失败');
      });
    return () => {
      cancelled = true;
    };
  }, [message, open]);

  // 核销单候选：打开弹窗或切换组织立即拉取首批；输入关键字后 300ms 防抖走服务端搜索。
  const loadVerificationOptions = useCallback(
    async (targetOrganizationId: string, keyword?: string) => {
      const sequence = ++verificationRequestRef.current;
      setVerificationLoading(true);
      try {
        const response =
          await settlementServiceListCommissionVerificationCandidates({
            page: 1,
            pageSize: 200,
            organizationId: targetOrganizationId,
            ...(keyword ? { keyword } : {}),
          });
        if (sequence !== verificationRequestRef.current) return;
        setVerificationOptions(
          unwrapList(response).map((item) => ({
            label: `${item.verificationNo}｜${item.settlementPartyName}｜${item.amount} ${item.currency}`,
            value: item.id as string,
          })),
        );
      } catch {
        if (sequence !== verificationRequestRef.current) return;
        setVerificationOptions([]);
        message.warning('有效应收核销候选加载失败');
      } finally {
        if (sequence === verificationRequestRef.current) {
          setVerificationLoading(false);
        }
      }
    },
    [message],
  );

  useEffect(() => {
    if (!open || !organizationId) return;
    const keyword = verificationKeyword.trim();
    if (!keyword) {
      void loadVerificationOptions(organizationId);
      return;
    }
    const timer = window.setTimeout(() => {
      void loadVerificationOptions(organizationId, keyword);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [loadVerificationOptions, open, organizationId, verificationKeyword]);

  // 对冲提成候选：进入对冲 Tab 且已选公司时加载已确认且有应收分摊的对冲单；关键字搜索走防抖。
  const loadNettingCandidates = useCallback(
    async (targetOrganizationId: string, keyword?: string) => {
      const sequence = ++nettingRequestRef.current;
      setNettingLoading(true);
      try {
        const response =
          await settlementServiceListCommissionNettingCandidates(
            {
              page: 1,
              pageSize: 200,
              organizationId: targetOrganizationId,
              ...(keyword ? { keyword } : {}),
            },
          );
        if (sequence !== nettingRequestRef.current) return;
        setNettingCandidates(unwrapList(response));
      } catch {
        if (sequence !== nettingRequestRef.current) return;
        setNettingCandidates([]);
        message.warning('对冲提成候选加载失败');
      } finally {
        if (sequence === nettingRequestRef.current) {
          setNettingLoading(false);
        }
      }
    },
    [message],
  );

  useEffect(() => {
    if (!open || sourceType !== 'netting' || !organizationId) return;
    const keyword = nettingKeyword.trim();
    if (!keyword) {
      void loadNettingCandidates(organizationId);
      return;
    }
    const timer = window.setTimeout(() => {
      void loadNettingCandidates(organizationId, keyword);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [loadNettingCandidates, nettingKeyword, open, sourceType, organizationId]);

  const resetSourceSelection = () => {
    setSelectedNetting(undefined);
    formRef.current?.setFieldsValue({
      verificationId: undefined,
      nettingId: undefined,
      employeeId: undefined,
    });
  };

  const resetCandidateState = () => {
    setVerificationOptions([]);
    setVerificationKeyword('');
    setNettingKeyword('');
    setNettingCandidates([]);
  };

  const handleSourceTabChange = (key: string) => {
    const next = key as CommissionSourceTab;
    if (next === sourceType) return;
    setSourceType(next);
    setNettingKeyword('');
    resetPreview();
    resetSourceSelection();
  };

  const handleSelectNetting = (record: API.FinanceNetting) => {
    setSelectedNetting(record);
    resetPreview();
    formRef.current?.setFieldsValue({
      nettingId: record.id,
      employeeId: undefined,
    });
  };

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
          setSourceType('verification');
          setSelectedNetting(undefined);
          resetCandidateState();
          resetPreview();
        },
      }}
      onValuesChange={(changedValues) => {
        if (
          'verificationId' in changedValues ||
          'nettingId' in changedValues ||
          'ruleId' in changedValues ||
          'employeeId' in changedValues
        ) {
          resetPreview();
        }
      }}
      onFinish={async (values) => {
        if (!preview || previewSignature !== calculationSignature(values)) {
          message.warning('请先计算并核对当前选择的提成预览');
          return false;
        }
        try {
          await settlementServiceCreateCommission({
            ...(values.verificationId
              ? { verificationId: values.verificationId }
              : { nettingId: values.nettingId }),
            employeeId: values.employeeId,
            ruleId: values.ruleId,
            note: values.note,
            idempotencyKey: createIdempotencyKey,
          });
          message.success('提成草稿已生成，列表已按创建结果刷新');
          onOpenChange(false);
          setSourceType('verification');
          setSelectedNetting(undefined);
          resetCandidateState();
          resetPreview();
          onSuccess();
          return true;
        } catch (error: any) {
          message.error(error.message || '提成生成失败');
          return false;
        }
      }}
    >
      <ProFormSearchableSelect
        name="organizationId"
        label="所属公司"
        rules={[{ required: true, message: '请选择所属公司' }]}
        options={organizationOptions.map((item) => ({
          value: item.id ?? '',
          label: item.name ?? item.code ?? item.id ?? '',
        }))}
        fieldProps={{
          onChange: (value) => {
            setOrganizationId(value);
            // 切换组织后候选与关键字全部重置，立即拉取新组织首批。
            setVerificationOptions([]);
            setVerificationKeyword('');
            setNettingKeyword('');
            resetPreview();
            resetSourceSelection();
            formRef.current?.setFieldsValue({ ruleId: undefined });
          },
        }}
      />
      <Tabs
        size="small"
        activeKey={sourceType}
        onChange={handleSourceTabChange}
        items={[
          { key: 'verification', label: '核销提成' },
          { key: 'netting', label: '对冲提成' },
        ]}
        style={{ marginBottom: 8 }}
      />
      {sourceType === 'verification' ? (
        <ProFormSearchableSelect
          key={organizationId || 'no-organization'}
          name="verificationId"
          label="有效应收核销"
          rules={[{ required: true, message: '请选择有效应收核销单' }]}
          disabled={!organizationId}
          options={verificationOptions}
          fieldProps={{
            filterOption: false,
            loading: verificationLoading,
            onSearch: (value: string) => setVerificationKeyword(value),
          }}
        />
      ) : (
        <Form.Item
          label="已确认对冲单"
          required
          extra="仅列出已确认且存在应收分摊的对冲单；已冲销完成的来源请勿重复计提。"
        >
          <Form.Item
            name="nettingId"
            hidden
            rules={[{ required: true, message: '请选择对冲单' }]}
          >
            <Input />
          </Form.Item>
          <Input
            allowClear
            disabled={!organizationId}
            placeholder="搜索对冲单号 / 结算单位"
            value={nettingKeyword}
            onChange={(event) => setNettingKeyword(event.target.value)}
            style={{ marginBottom: 8 }}
          />
          <Table<API.FinanceNetting>
            size="small"
            bordered
            rowKey="id"
            loading={nettingLoading}
            pagination={false}
            dataSource={nettingCandidates}
            columns={nettingCandidateColumns}
            scroll={{ y: 260 }}
            rowSelection={{
              type: 'radio',
              selectedRowKeys: selectedNetting?.id
                ? [selectedNetting.id]
                : [],
              onSelect: handleSelectNetting,
            }}
            onRow={(record) => ({
              onClick: () => handleSelectNetting(record),
              style: { cursor: 'pointer' },
            })}
            locale={{
              emptyText: organizationId ? '暂无符合条件的对冲单' : '请先选择所属公司',
            }}
          />
        </Form.Item>
      )}
      <ProFormSearchableSelect
        key={`rule-${organizationId || 'no-organization'}`}
        name="ruleId"
        label="考核规则"
        rules={[{ required: true, message: '请选择考核规则' }]}
        disabled={!organizationId}
        request={async () => {
          if (!organizationId) return [];
          const response = await settlementServiceListCommissionRuleCandidates({
            page: 1,
            pageSize: 200,
            organizationId,
          });
          return unwrapList(response).map((item) => ({
            label: `${item.name}｜${personnelRoleText(item.personnelRole)}｜${calculationBasisText(item.calculationBasis)} × ${decimalText(item.ratePercent)}%`,
            value: item.id,
          }));
        }}
      />
      <ProFormDependency
        name={['verificationId', 'nettingId', 'ruleId', 'organizationId']}
      >
        {({
          verificationId,
          nettingId,
          ruleId,
          organizationId: selectedOrganizationID,
        }) => {
          const isVerificationSource = sourceType === 'verification';
          const sourceId = isVerificationSource
            ? verificationId
            : nettingId;
          return (
            <ProFormSearchableSelect
              key={`${sourceType}-${sourceId || ''}-${ruleId || ''}`}
              name="employeeId"
              label={isVerificationSource ? '符合规则的候选人员' : '提成员工'}
              rules={[{ required: true, message: '请选择提成员工' }]}
              disabled={
                !sourceId ||
                !selectedOrganizationID ||
                (isVerificationSource && !ruleId)
              }
              request={async () => {
                if (!sourceId || !selectedOrganizationID) return [];
                if (isVerificationSource) {
                  if (!ruleId || !verificationId) return [];
                  const response =
                    await settlementServiceListCommissionCandidates({
                      verificationId,
                      ruleId,
                      page: 1,
                      pageSize: 200,
                      organizationId: selectedOrganizationID,
                    });
                  return unwrapList(response).map((item) => ({
                    label: `${item.employeeName}｜${item.customerCount ?? 0}个客户｜${item.orderCount ?? 0}票订单｜预计 ${decimalText(item.commissionAmount)} ${item.baseCurrency}`,
                    value: item.employeeId,
                  }));
                }
                const response =
                  await settlementServiceListCommissionEmployees({
                    page: 1,
                    pageSize: 200,
                    organizationId: selectedOrganizationID,
                  });
                return unwrapList(response).map((item) => ({
                  label: item.displayName,
                  value: item.id,
                }));
              }}
              extra={
                isVerificationSource
                  ? '人员来自本次核销涉及订单在创建时固化的业务、操作或客服归属；客户后续换人不会改变历史订单归属。'
                  : '人员按所选公司列出，预览时会按对冲单涉及订单固化的人员归属校验员工与规则角色是否匹配。'
              }
            />
          );
        }}
      </ProFormDependency>
      <ProFormTextArea
        name="note"
        label="备注"
        fieldProps={{ maxLength: 500 }}
      />
      <ProFormDependency
        name={['verificationId', 'nettingId', 'ruleId', 'employeeId']}
      >
        {(values: Partial<CreateValues>) => {
          const sourceId = values.verificationId || values.nettingId;
          return (
            <Space vertical size={12} style={{ width: '100%' }}>
              <Button
                type="primary"
                ghost
                loading={previewLoading}
                disabled={!sourceId || !values.ruleId || !values.employeeId}
                onClick={async () => {
                  const { verificationId, nettingId, employeeId, ruleId } =
                    values;
                  if (!sourceId || !employeeId || !ruleId) return;
                  try {
                    setPreviewLoading(true);
                    const response = await settlementServicePreviewCommission({
                      ...(verificationId
                        ? { verificationId }
                        : { nettingId }),
                      employeeId,
                      ruleId,
                    });
                    setPreview(response.data);
                    setPreviewSignature(calculationSignature(values));
                  } catch (error: any) {
                    resetPreview();
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
                  description="系统会按本次来源单（核销或对冲）涉及的订单，分别展示已实现收入、分摊成本、毛利和提成金额，并支持下钻展开费用明细。"
                />
              ) : (
                <>
                  <Descriptions
                    size="small"
                    bordered
                    column={4}
                    items={[
                      {
                        key: 'source',
                        label: '来源单号',
                        children: commissionSourceNo(preview),
                      },
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
          );
        }}
      </ProFormDependency>
      <Space vertical size={2} style={{ color: '#666', marginTop: 8 }}>
        <span>
          计算比例、角色与口径均取自已启用且在来源单归属日期生效的考核规则。
        </span>
        <span>亏损订单逐票按 0 计提，但仍保留真实负毛利快照。</span>
        <span>
          草稿确认时会重新校验客户人员与费用来源；来源变化后必须取消并重新生成。
        </span>
      </Space>
    </ModalForm>
  );
}
