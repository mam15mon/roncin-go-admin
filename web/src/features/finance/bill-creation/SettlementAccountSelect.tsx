import { PlusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Button, Checkbox, ConfigProvider, Form, Input, Select } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { QuickCreateModal } from '@/components/ui';
import { PartnerAccountUsage } from '@/enums.generated';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { partnerServiceCreatePartnerAccount } from '@/services/roncin/partnerService';
import { settlementServiceListBillSettlementAccountCandidates } from '@/services/roncin/settlementService';
import { ensureListResponse, unwrapList } from '@/utils/api';

type Props = {
  group: API.BillBatchPreviewGroup;
  organizationId: string;
  sessionIdentity: string;
  onConfigurationChange: () => void;
};

type AccountFormValues = Pick<
  API.PartnerAccountInput,
  'name' | 'accountHolder' | 'bankName' | 'accountNo' | 'swiftCode'
> & { isDefault?: boolean };

function accountLabel(account: API.FinanceSettlementAccountOption) {
  const name = account.name || account.accountHolder || '未命名账户';
  const suffix = account.accountNo ? ` · ${account.accountNo}` : '';
  return `${name}｜${account.bankName || '-'}${suffix}｜${account.currency || '-'}`;
}

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
  const createGuard = useAsyncGuard();
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
  const canQuickCreateRef = useRef(canQuickCreate);
  canQuickCreateRef.current = canQuickCreate;
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
      createGuard.invalidate();
      setModalOpen(false);
    }
  }, [canQuickCreate, createGuard]);

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
            label: accountLabel(account),
          }))}
          popupRender={
            canQuickCreate
              ? (menu) => (
                  <>
                    {menu}
                    <Button
                      type="text"
                      block
                      icon={<PlusOutlined />}
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => {
                        setSelectOpen(false);
                        setModalOpen(true);
                      }}
                    >
                      新增结算账户
                    </Button>
                  </>
                )
              : undefined
          }
        />
      </Form.Item>
      <QuickCreateModal<AccountFormValues, API.PartnerAccount>
        key={canQuickCreate ? 'enabled' : 'disabled'}
        title="新增结算账户"
        open={modalOpen && canQuickCreate}
        initialValues={{ isDefault: false }}
        onCancel={() => setModalOpen(false)}
        onSubmit={async (values) => {
          if (!canQuickCreateRef.current) return undefined;
          await createGuard.run(
            ({ signal }) =>
              partnerServiceCreatePartnerAccount(
                { partnerId },
                {
                  partnerId,
                  account: {
                    name: values.name.trim(),
                    accountHolder: values.accountHolder.trim(),
                    currency,
                    bankName: values.bankName.trim(),
                    accountNo: values.accountNo.trim(),
                    swiftCode: values.swiftCode?.trim() || undefined,
                    usage: isReceivable
                      ? PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_RECEIVABLE
                      : PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_PAYABLE,
                    isDefaultReceivable:
                      isReceivable && Boolean(values.isDefault),
                    isDefaultPayable:
                      !isReceivable && Boolean(values.isDefault),
                    enabled: true,
                  },
                },
                { signal },
              ),
            (response) => {
              if (
                !canQuickCreateRef.current ||
                !response ||
                response.success === false
              )
                return;
              const account = response.data;
              if (!account?.id)
                throw new Error('创建结果缺少账户 ID，请核对后重试');
              setCreated({
                id: account.id,
                name: account.name || values.name.trim(),
                accountHolder:
                  account.accountHolder || values.accountHolder.trim(),
                bankName: account.bankName || values.bankName.trim(),
                accountNo: account.accountNo || values.accountNo.trim(),
                currency,
              });
              form.setFieldValue(accountField, account.id);
              setRevision((current) => current + 1);
              onConfigurationChangeRef.current();
              setModalOpen(false);
            },
          );
          // 成功回填在同步 guard 内完成；失效结果不能触发弹窗的成功副作用。
          return undefined;
        }}
      >
        <Form.Item label="结算对象">
          <Input
            aria-label="结算对象"
            disabled
            value={group.settlementPartyName || partnerId}
          />
        </Form.Item>
        <Form.Item label="账户币种">
          <Input aria-label="账户币种" disabled value={currency} />
        </Form.Item>
        <Form.Item label="允许用途">
          <Input
            aria-label="允许用途"
            disabled
            value={isReceivable ? '应收' : '应付'}
          />
        </Form.Item>
        {(
          [
            ['name', '账户名称'],
            ['accountHolder', '账户户名'],
            ['bankName', '开户银行名称及支行'],
            ['accountNo', '银行开户账号'],
          ] as const
        ).map(([name, label]) => (
          <Form.Item
            key={name}
            name={name}
            label={label}
            rules={[
              { required: true, whitespace: true, message: `请输入${label}` },
            ]}
          >
            <Input />
          </Form.Item>
        ))}
        <Form.Item name="swiftCode" label="SWIFT Code（外币国际结算）">
          <Input />
        </Form.Item>
        <Form.Item name="isDefault" valuePropName="checked">
          <Checkbox>
            {isReceivable ? '设为应收默认账户' : '设为应付默认账户'}
          </Checkbox>
        </Form.Item>
      </QuickCreateModal>
    </>
  );
}
