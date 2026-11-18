import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Descriptions, Space, Spin } from 'antd';
import { useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { GuardedModalForm } from '@/components/ui';
import { PartnerRoleType } from '@/enums.generated';
import {
  partnerServiceCreatePartnerSettlementRule,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartnerSettlementRule,
} from '@/services/roncin/partnerService';
import { ensureListResponse, isWriteResponse, unwrapList } from '@/utils/api';
import { formatAmount } from '@/utils/format';
import { hasInvalidCreditCurrency } from '../../creditLimit';
import PaymentTermsFields, {
  type PaymentTermsFormValues,
  paymentTermsFromForm,
  paymentTermsToForm,
} from '../PaymentTermsFields';

type SettlementRulesPanelProps = {
  partner?: API.Partner;
  canRead: boolean;
  canCreate: boolean;
  canUpdate: boolean;
};

export default function SettlementRulesPanel({
  partner,
  canRead,
  canCreate,
  canUpdate,
}: SettlementRulesPanelProps) {
  const { message } = App.useApp();
  const { initialState } = useInitialState();
  const baseCurrency =
    initialState?.currentUser?.currentOrganization?.baseCurrency;
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRule, setEditingRule] =
    useState<API.PartnerSettlementRule | null>(null);
  const roleType = PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER;
  const hasCustomer = partner?.roles?.some((role) => role.type === roleType);
  const partnerId = partner?.id;
  const query = useQuery({
    queryKey: ['partner-detail', 'settlement-rule', { partnerId, roleType }],
    enabled: Boolean(partnerId && hasCustomer && canRead),
    queryFn: async () => {
      if (!partnerId) throw new Error('缺少档案标识');
      const response = await partnerServiceListPartnerSettlementRules({
        partnerId,
        roleType,
      });
      ensureListResponse(response, '收款约定加载失败，请重试');
      return unwrapList(response)[0] ?? null;
    },
    meta: { silent: true },
  });

  if (!partnerId || !hasCustomer || !canRead) return null;
  if (query.isPending) return <Spin />;
  const rule = query.data ?? null;
  const canWrite = query.isSuccess && (rule ? canUpdate : canCreate);
  const invalidCurrency = hasInvalidCreditCurrency(editingRule, baseCurrency);
  const canSubmit =
    query.isSuccess &&
    !query.isFetching &&
    (editingRule ? canUpdate : canCreate);
  const loadError = query.isError && (
    <Alert
      type="error"
      showIcon
      title="收款约定加载失败，请重试"
      action={
        <Button loading={query.isFetching} onClick={() => void query.refetch()}>
          重新加载收款约定
        </Button>
      }
    />
  );

  return (
    <>
      {!modalOpen && loadError}
      {query.isSuccess && (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Descriptions
            title="收款约定"
            column={1}
            items={[
              {
                key: 'terms',
                label: '收款期限',
                children:
                  rule?.paymentTermsDays == null
                    ? '未约定'
                    : rule.paymentTermsDays === 0
                      ? '现结'
                      : `建账后 ${rule.paymentTermsDays} 天`,
              },
              {
                key: 'credit',
                label: `信用额度（${baseCurrency || '本币'}）`,
                children:
                  rule?.creditLimitMinor == null
                    ? '未约定'
                    : `${formatAmount(Number(rule.creditLimitMinor) / 100)} ${rule.creditCurrency || '币种缺失'}`,
              },
            ]}
          />
          {canWrite && (
            <Button
              type="primary"
              disabled={query.isFetching}
              onClick={() => {
                setEditingRule(rule);
                setModalOpen(true);
              }}
            >
              {rule ? '编辑收款约定' : '设置收款约定'}
            </Button>
          )}
        </Space>
      )}
      <GuardedModalForm<PaymentTermsFormValues>
        key={`${partnerId}:${editingRule?.id || 'new'}`}
        guardIdentity={`${partnerId}:${editingRule?.id || 'new'}`}
        title={editingRule ? '编辑收款约定' : '设置收款约定'}
        open={modalOpen}
        onOpenChange={setModalOpen}
        initialValues={paymentTermsToForm(editingRule, baseCurrency)}
        modalProps={{ destroyOnHidden: true, width: 640 }}
        submitter={{ submitButtonProps: { disabled: !canSubmit } }}
        onFinish={async (values) => {
          if (!canSubmit) return false;
          const input = paymentTermsFromForm(values, baseCurrency);
          const response = editingRule?.id
            ? await partnerServiceUpdatePartnerSettlementRule(
                { partnerId, roleType, id: editingRule.id },
                { partnerId, roleType, id: editingRule.id, rule: input },
              )
            : await partnerServiceCreatePartnerSettlementRule(
                { partnerId, roleType },
                { partnerId, roleType, rule: input },
              );
          if (!isWriteResponse(response, true)) return false;
          message.success('收款约定已保存');
          await queryClient.invalidateQueries({ queryKey: ['partner-detail'] });
          return true;
        }}
      >
        {modalOpen && loadError}
        {invalidCurrency && (
          <Alert
            type="warning"
            showIcon
            title={`原信用额度为 ${formatAmount(Number(editingRule?.creditLimitMinor) / 100)} ${editingRule?.creditCurrency || '币种缺失'}，请重新填写 ${baseCurrency || '公司本币'} 额度`}
            style={{ marginBottom: 16 }}
          />
        )}
        <PaymentTermsFields
          baseCurrency={baseCurrency}
          requireCreditLimit={invalidCurrency}
        />
      </GuardedModalForm>
    </>
  );
}
