import { useQuery } from '@tanstack/react-query';
import { ConfigProvider, Form, Select } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { QuickCreateSelectFooter } from '@/components/ui';
import {
  SettlementAccountQuickCreateModal,
  settlementAccountLabel,
} from '@/features/partners/settlement-account';
import { settlementServiceListBillSettlementAccountCandidates } from '@/services/roncin/settlementService';
import { ensureListResponse, unwrapList } from '@/utils/api';

type Props = {
  group: API.BillBatchPreviewGroup;
  organizationId: string;
  sessionIdentity: string;
  onConfigurationChange: () => void;
};

/** 身份变化直接重建叶子，旧创建结果不能写入新组或新会话。 */
export default function SettlementAccountSelect(props: Props) {
  const { group, organizationId, sessionIdentity } = props;
  const identity = [
    sessionIdentity,
    organizationId,
    group.groupKey,
    group.settlementPartyId,
    group.direction,
    group.currency,
  ].join(':');
  return <AccountLeaf key={identity} {...props} identity={identity} />;
}

function AccountLeaf({
  group,
  organizationId,
  identity,
  onConfigurationChange,
}: Props & { identity: string }) {
  const form = Form.useFormInstance();
  const access = useAccess();
  const { componentDisabled } = ConfigProvider.useConfig();
  const [modalOpen, setModalOpen] = useState(false);
  const [selectOpen, setSelectOpen] = useState(false);
  const [revision, setRevision] = useState(0);
  const [created, setCreated] = useState<API.FinanceSettlementAccountOption>();
  const onConfigurationChangeRef = useRef(onConfigurationChange);
  onConfigurationChangeRef.current = onConfigurationChange;
  const groupKey = group.groupKey || '';
  const partnerId = group.settlementPartyId || '';
  const currency = group.currency || '';
  const isReceivable = group.direction === 'RECEIVABLE';
  const validDirection = isReceivable || group.direction === 'PAYABLE';
  const canOperate = access.canOperateOrganization(organizationId);
  const canQuickCreate = Boolean(
    canOperate &&
      !componentDisabled &&
      access.canCreatePartnerAccounts &&
      groupKey &&
      partnerId &&
      currency &&
      validDirection,
  );
  const accountField = ['groups', groupKey, 'settlementAccountId'];
  const candidates = useQuery({
    queryKey: [
      'finance-bills',
      'settlement-account-candidates',
      identity,
      revision,
    ],
    enabled: Boolean(
      canOperate && groupKey && partnerId && currency && validDirection,
    ),
    queryFn: async ({ signal }) =>
      unwrapList(
        ensureListResponse(
          await settlementServiceListBillSettlementAccountCandidates(
            {
              organizationId,
              settlementPartyId: partnerId,
              direction: group.direction,
              currency,
            },
            { signal },
          ),
          '获取结算账户失败',
        ),
      ),
    meta: { errorMessage: '获取结算账户失败' },
  });
  const accounts = candidates.data ?? [];
  const options = created?.id
    ? [created, ...accounts.filter((account) => account.id !== created.id)]
    : accounts;

  useEffect(() => {
    if (!canQuickCreate) {
      setModalOpen(false);
    }
  }, [canQuickCreate]);

  useEffect(() => {
    if (!candidates.isSuccess || !canOperate || componentDisabled) return;
    const selected = form.getFieldValue([
      'groups',
      groupKey,
      'settlementAccountId',
    ]);
    if (
      selected &&
      (selected === created?.id ||
        candidates.data.some((account) => account.id === selected))
    )
      return;
    const defaultAccount = candidates.data.find((account) => account.isDefault);
    if (!defaultAccount?.id) return;
    form.setFieldValue(
      ['groups', groupKey, 'settlementAccountId'],
      defaultAccount.id,
    );
    onConfigurationChangeRef.current();
  }, [
    canOperate,
    componentDisabled,
    candidates.data,
    candidates.isSuccess,
    created?.id,
    form,
    groupKey,
  ]);

  return (
    <>
      <Form.Item
        name={accountField}
        label="结算账户"
        rules={[{ required: true, message: '请选择结算账户' }]}
      >
        <Select
          allowClear
          disabled={!canOperate || undefined}
          loading={candidates.isFetching}
          placeholder="请选择与账单方向、最终币种一致的启用账户"
          open={selectOpen}
          onOpenChange={setSelectOpen}
          options={options.map((account) => ({
            value: account.id,
            label: settlementAccountLabel(account),
          }))}
          popupRender={
            canQuickCreate
              ? (menu) => (
                  <QuickCreateSelectFooter
                    label="新增结算账户"
                    onCreate={() => {
                      setSelectOpen(false);
                      setModalOpen(true);
                    }}
                  >
                    {menu}
                  </QuickCreateSelectFooter>
                )
              : undefined
          }
        />
      </Form.Item>
      <SettlementAccountQuickCreateModal
        open={modalOpen && canQuickCreate}
        organizationId={organizationId}
        partnerId={partnerId}
        partnerName={group.settlementPartyName}
        currency={currency}
        direction={group.direction || ''}
        sessionIdentity={identity}
        onCancel={() => setModalOpen(false)}
        onCreated={(account) => {
          setCreated(account);
          form.setFieldValue(accountField, account.id);
          setRevision((current) => current + 1);
          onConfigurationChangeRef.current();
          setModalOpen(false);
        }}
      />
    </>
  );
}
