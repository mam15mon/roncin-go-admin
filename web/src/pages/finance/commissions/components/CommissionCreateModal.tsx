import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDependency,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import type { TableProps } from 'antd';
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
import React, { useEffect, useRef, useState } from 'react';
import { MODAL_SIZE, ProFormSearchableSelect } from '@/components/ui';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import {
  settlementServiceCreateCommission,
  settlementServiceListCommissionCandidates,
  settlementServiceListCommissionNettingCandidates,
  settlementServiceListCommissionVerificationCandidates,
  settlementServiceListFinanceOrganizationOptions,
  settlementServicePreviewCommission,
} from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatDate } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import {
  type CreateValues,
  calculationBasisText,
  calculationSignature,
  cnyExchangeRateSourceText,
  commissionSourceNo,
  decimalText,
  parseCandidateKey,
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

/** 计提候选的来源单：核销单或对冲单二选一。 */
type CandidateSource = {
  verificationId?: string;
  nettingId?: string;
};

/** 候选选中键：`${employeeId}|${personnelRole}`，由候选列表派生。 */
const candidateKeyOf = (employeeId?: string, personnelRole?: string) =>
  employeeId && personnelRole ? `${employeeId}|${personnelRole}` : undefined;

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
  const [sourceType, setSourceType] =
    useState<CommissionSourceTab>('verification');
  const [selectedNetting, setSelectedNetting] = useState<API.FinanceNetting>();
  const [verificationKeyword, setVerificationKeyword] = useState('');
  const [nettingKeyword, setNettingKeyword] = useState('');
  // 关键字防抖态：输入停顿 300ms 后收敛进 queryKey，防抖期内不触发搜索。
  const [verificationSearchKeyword, setVerificationSearchKeyword] =
    useState('');
  const [nettingSearchKeyword, setNettingSearchKeyword] = useState('');
  // 来源选中后由服务端解析的「员工 + 身份 + 已解析方案」候选来源单。
  const [candidateSource, setCandidateSource] = useState<CandidateSource>();

  const resetPreview = () => {
    setPreview(undefined);
    setPreviewSignature('');
    setCreateIdempotencyKey(generateUUID());
  };

  // 公司候选：弹窗打开即拉取；失败经全局 cache onError 以 meta 文案提示。
  const organizationQuery = useQuery({
    queryKey: ['commission-create', 'organization-options'],
    enabled: open,
    meta: { errorMessage: '提成创建公司候选加载失败' },
    queryFn: async () => {
      const response = await settlementServiceListFinanceOrganizationOptions({
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_COMMISSION_MANAGE,
      });
      return response.data ?? [];
    },
  });

  // 核销单候选：已选公司即拉取首批；关键字搜索经 300ms 防抖进 queryKey，
  // 输入期间 keepPreviousData 保持列表不闪空，竞态由库按 queryKey 收敛。
  const verificationQuery = useQuery({
    queryKey: [
      'commission-create',
      'verification-candidates',
      { organizationId, keyword: verificationSearchKeyword || undefined },
    ],
    enabled: open && Boolean(organizationId),
    placeholderData: keepPreviousData,
    meta: { errorMessage: '有效应收核销候选加载失败' },
    queryFn: async () => {
      if (!organizationId) return [];
      const response =
        await settlementServiceListCommissionVerificationCandidates({
          page: 1,
          pageSize: 200,
          organizationId,
          ...(verificationSearchKeyword
            ? { keyword: verificationSearchKeyword }
            : {}),
        });
      return unwrapList(response).map((item) => ({
        label: `${item.verificationNo}｜${item.settlementPartyName}｜${item.amount} ${item.currency}`,
        value: item.id as string,
      }));
    },
  });

  // 对冲提成候选：进入对冲 Tab 且已选公司时加载已确认且有应收分摊的对冲单。
  const nettingQuery = useQuery({
    queryKey: [
      'commission-create',
      'netting-candidates',
      { organizationId, keyword: nettingSearchKeyword || undefined },
    ],
    enabled: open && sourceType === 'netting' && Boolean(organizationId),
    placeholderData: keepPreviousData,
    meta: { errorMessage: '对冲提成候选加载失败' },
    queryFn: async () => {
      if (!organizationId) return [];
      const response = await settlementServiceListCommissionNettingCandidates({
        page: 1,
        pageSize: 200,
        organizationId,
        ...(nettingSearchKeyword ? { keyword: nettingSearchKeyword } : {}),
      });
      return unwrapList(response);
    },
  });

  // 计提候选：来源单选中后由服务端按来源归属解析。
  const candidateQuery = useQuery({
    queryKey: [
      'commission-create',
      'commission-candidates',
      {
        organizationId,
        ...(candidateSource?.verificationId
          ? { verificationId: candidateSource.verificationId }
          : { nettingId: candidateSource?.nettingId }),
      },
    ],
    enabled:
      open &&
      Boolean(organizationId) &&
      Boolean(candidateSource?.verificationId || candidateSource?.nettingId),
    meta: { errorMessage: '计提候选加载失败' },
    queryFn: async () => {
      if (!organizationId || !candidateSource) return [];
      const response = await settlementServiceListCommissionCandidates({
        page: 1,
        pageSize: 200,
        organizationId,
        ...(candidateSource.verificationId
          ? { verificationId: candidateSource.verificationId }
          : { nettingId: candidateSource.nettingId }),
      });
      return unwrapList(response).flatMap((item) => {
        const value = candidateKeyOf(item.employeeId, item.personnelRole);
        return value
          ? [
              {
                label: `${item.employeeName}｜${personnelRoleText(item.personnelRole)}｜${item.ruleName}（v${item.ruleVersion ?? ''}）｜预计 ${decimalText(item.commissionAmount)} ${item.baseCurrency}`,
                value,
              },
            ]
          : [];
      });
    },
  });

  // 核销关键字防抖：清空立即收敛为空，输入停顿 300ms 后收敛进 queryKey。
  useEffect(() => {
    const keyword = verificationKeyword.trim();
    if (!keyword) {
      setVerificationSearchKeyword('');
      return;
    }
    const timer = window.setTimeout(() => {
      setVerificationSearchKeyword(keyword);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [verificationKeyword]);

  // 对冲关键字防抖：语义同核销关键字。
  useEffect(() => {
    const keyword = nettingKeyword.trim();
    if (!keyword) {
      setNettingSearchKeyword('');
      return;
    }
    const timer = window.setTimeout(() => {
      setNettingSearchKeyword(keyword);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [nettingKeyword]);

  const resetSourceSelection = () => {
    setSelectedNetting(undefined);
    formRef.current?.setFieldsValue({
      verificationId: undefined,
      nettingId: undefined,
      candidateKey: undefined,
    });
  };

  const resetCandidateState = () => {
    setVerificationKeyword('');
    setVerificationSearchKeyword('');
    setNettingKeyword('');
    setNettingSearchKeyword('');
    setCandidateSource(undefined);
  };

  const handleSourceTabChange = (key: string) => {
    const next = key as CommissionSourceTab;
    if (next === sourceType) return;
    setSourceType(next);
    setNettingKeyword('');
    setNettingSearchKeyword('');
    setCandidateSource(undefined);
    resetPreview();
    resetSourceSelection();
  };

  const handleSelectNetting = (record: API.FinanceNetting) => {
    setSelectedNetting(record);
    resetPreview();
    formRef.current?.setFieldsValue({
      nettingId: record.id,
      candidateKey: undefined,
    });
  };

  return (
    <ModalForm<CreateValues>
      formRef={formRef}
      title="生成提成"
      open={open}
      width={MODAL_SIZE.LG}
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
        if ('candidateKey' in changedValues) {
          resetPreview();
        }
      }}
      onFinish={async (values) => {
        if (!preview || previewSignature !== calculationSignature(values)) {
          message.warning('请先计算并核对当前选择的提成预览');
          return false;
        }
        const { employeeId, personnelRole } = parseCandidateKey(
          values.candidateKey,
        );
        if (!employeeId || !personnelRole) {
          message.warning('请选择计提候选');
          return false;
        }
        try {
          await settlementServiceCreateCommission({
            ...(values.verificationId
              ? { verificationId: values.verificationId }
              : { nettingId: values.nettingId }),
            employeeId,
            personnelRole,
            note: values.note,
            idempotencyKey: createIdempotencyKey,
            organizationId: organizationId ?? '',
          });
          message.success('提成草稿已生成，列表已按创建结果刷新');
          onOpenChange(false);
          setSourceType('verification');
          setSelectedNetting(undefined);
          resetCandidateState();
          resetPreview();
          onSuccess();
          return true;
        } catch (error) {
          message.error(getErrorMessage(error, '提成生成失败'));
          return false;
        }
      }}
    >
      <ProFormSearchableSelect
        name="organizationId"
        label="所属公司"
        rules={[{ required: true, message: '请选择所属公司' }]}
        options={(organizationQuery.data ?? []).map((item) => ({
          value: item.id ?? '',
          label: item.name ?? item.code ?? item.id ?? '',
        }))}
        fieldProps={{
          onChange: (value) => {
            setOrganizationId(value);
            // 切换组织后候选与关键字全部重置，立即拉取新组织首批；
            // 防抖态同步清空，避免渲染间隙携带旧关键字发请求。
            setVerificationKeyword('');
            setVerificationSearchKeyword('');
            setNettingKeyword('');
            setNettingSearchKeyword('');
            setCandidateSource(undefined);
            resetPreview();
            resetSourceSelection();
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
          options={verificationQuery.data ?? []}
          fieldProps={{
            filterOption: false,
            loading: verificationQuery.isFetching,
            onSearch: (value: string) => setVerificationKeyword(value),
            onChange: (value?: string) => {
              if (!value) return;
              resetPreview();
              setCandidateSource({ verificationId: value });
            },
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
            loading={nettingQuery.isFetching}
            pagination={false}
            dataSource={nettingQuery.data ?? []}
            columns={nettingCandidateColumns}
            scroll={{ y: 260 }}
            rowSelection={{
              type: 'radio',
              selectedRowKeys: selectedNetting?.id ? [selectedNetting.id] : [],
              onSelect: (record) => {
                handleSelectNetting(record);
                setCandidateSource({ nettingId: record.id });
              },
            }}
            onRow={(record) => ({
              onClick: () => {
                handleSelectNetting(record);
                setCandidateSource({ nettingId: record.id });
              },
              style: { cursor: 'pointer' },
            })}
            locale={{
              emptyText: organizationId
                ? '暂无符合条件的对冲单'
                : '请先选择所属公司',
            }}
          />
        </Form.Item>
      )}
      <ProFormDependency
        name={['verificationId', 'nettingId', 'organizationId']}
      >
        {({
          verificationId,
          nettingId,
          organizationId: selectedOrganizationID,
        }) => {
          const sourceId = verificationId || nettingId;
          return (
            <ProFormSearchableSelect
              key={`${sourceType}-${sourceId || ''}`}
              name="candidateKey"
              label="计提候选（员工 / 身份 / 已解析方案）"
              rules={[{ required: true, message: '请选择计提候选' }]}
              disabled={!sourceId || !selectedOrganizationID}
              options={candidateQuery.data ?? []}
              fieldProps={{
                filterOption: false,
                loading: candidateQuery.isFetching,
              }}
              extra="候选由服务端按来源单涉及订单的固化归属与归属日期自动解析：仅列出该员工身份在来源日期唯一命中的启用方案；无候选表示该员工没有可用方案或身份归属。"
            />
          );
        }}
      </ProFormDependency>
      <ProFormTextArea
        name="note"
        label="备注"
        fieldProps={{ maxLength: 500 }}
      />
      <ProFormDependency name={['verificationId', 'nettingId', 'candidateKey']}>
        {(values: Partial<CreateValues>) => {
          const sourceId = values.verificationId || values.nettingId;
          return (
            <Space vertical size={12} style={{ width: '100%' }}>
              <Button
                type="primary"
                ghost
                loading={previewLoading}
                disabled={!sourceId || !values.candidateKey}
                onClick={async () => {
                  const { employeeId, personnelRole } = parseCandidateKey(
                    values.candidateKey,
                  );
                  if (!sourceId || !employeeId || !personnelRole) return;
                  try {
                    setPreviewLoading(true);
                    const response = await settlementServicePreviewCommission({
                      ...(values.verificationId
                        ? { verificationId: values.verificationId }
                        : { nettingId: values.nettingId }),
                      employeeId,
                      personnelRole,
                      organizationId: organizationId ?? '',
                    });
                    setPreview(response.data);
                    setPreviewSignature(calculationSignature(values));
                  } catch (error) {
                    resetPreview();
                    message.error(getErrorMessage(error, '提成预览计算失败'));
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
                        label: '方案',
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
                    description="创建草稿时会在事务内重新解析方案与 CNY 汇率；最终折算依据和金额以创建结果及刷新后的列表为准。"
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
          比例、口径与身份均取自服务端按来源归属日期解析的唯一有效方案与员工分配。
        </span>
        <span>亏损订单逐票按 0 计提，但仍保留真实负毛利快照。</span>
        <span>
          草稿确认时会重新校验人员归属与方案；来源或方案变化后必须取消并重新生成。
        </span>
      </Space>
    </ModalForm>
  );
}
