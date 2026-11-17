import { Checkbox, ConfigProvider, Form, Input } from 'antd';
import { useEffect, useRef } from 'react';
import { useAccess } from '@/app/access';
import { QuickCreateModal } from '@/components/ui';
import { PartnerAccountUsage } from '@/enums.generated';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { partnerServiceCreatePartnerAccount } from '@/services/roncin/partnerService';
import { isWriteResponse } from '@/utils/api';

type AccountFormValues = Pick<
  API.PartnerAccountInput,
  'name' | 'accountHolder' | 'bankName' | 'accountNo' | 'swiftCode'
> & { isDefault?: boolean };

type Props = {
  open: boolean;
  organizationId: string;
  partnerId: string;
  partnerName?: string;
  currency: string;
  direction: string;
  sessionIdentity: string;
  disabled?: boolean;
  onCancel: () => void;
  onCreated: (account: API.FinanceSettlementAccountOption) => void;
};

export function settlementAccountLabel(
  account: API.FinanceSettlementAccountOption,
) {
  const name = account.name || account.accountHolder || '未命名账户';
  const suffix = account.accountNo ? ` · ${account.accountNo}` : '';
  return `${name}｜${account.bankName || '-'}${suffix}｜${account.currency || '-'}`;
}

/** 以来源和弹窗会话重建提交守卫，迟到写结果不回填另一张账单。 */
export function SettlementAccountQuickCreateModal(props: Props) {
  const identity = [
    props.sessionIdentity,
    props.organizationId,
    props.partnerId,
    props.currency,
    props.direction,
    props.open,
  ].join(':');
  return <AccountModalLeaf key={identity} {...props} />;
}

function AccountModalLeaf({
  open,
  organizationId,
  partnerId,
  partnerName,
  currency,
  direction,
  disabled,
  onCancel,
  onCreated,
}: Props) {
  const access = useAccess();
  const { componentDisabled } = ConfigProvider.useConfig();
  const guard = useAsyncGuard();
  const isReceivable = direction === 'RECEIVABLE';
  const canCreate = Boolean(
    open &&
      !disabled &&
      !componentDisabled &&
      access.canCreatePartnerAccounts &&
      access.canOperateOrganization(organizationId) &&
      partnerId &&
      currency &&
      (isReceivable || direction === 'PAYABLE'),
  );
  const active = useRef(canCreate);
  active.current = canCreate;
  const onCreatedRef = useRef(onCreated);
  onCreatedRef.current = onCreated;
  useEffect(() => {
    if (!canCreate) guard.invalidate();
  }, [canCreate, guard]);
  return (
    <QuickCreateModal<AccountFormValues, never>
      key={canCreate ? 'editable' : 'readonly'}
      title="新增结算账户"
      open={canCreate}
      initialValues={{ isDefault: false }}
      onCancel={onCancel}
      onSubmit={async (values) => {
        if (!active.current) return undefined;
        await guard.run(
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
                  isDefaultPayable: !isReceivable && Boolean(values.isDefault),
                  enabled: true,
                },
              },
              { signal },
            ),
          (response) => {
            if (!active.current || !response || response.success === false)
              return;
            if (!response.data?.id)
              throw new Error('创建结果缺少账户 ID，请核对后重试');
            if (!isWriteResponse(response, true)) return;
            const account = response.data;
            onCreatedRef.current({
              id: account.id,
              name: account.name || values.name.trim(),
              accountHolder:
                account.accountHolder || values.accountHolder.trim(),
              bankName: account.bankName || values.bankName.trim(),
              accountNo: account.accountNo || values.accountNo.trim(),
              currency,
            });
          },
        );
        return undefined;
      }}
    >
      <Form.Item label="结算对象">
        <Input
          aria-label="结算对象"
          disabled
          value={partnerName || partnerId}
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
          ['name', '账户名称', 200],
          ['accountHolder', '账户户名', 200],
          ['bankName', '开户银行名称及支行', 200],
          ['accountNo', '银行开户账号', 100],
        ] as const
      ).map(([name, label, max]) => (
        <Form.Item
          key={name}
          name={name}
          label={label}
          rules={[
            { required: true, whitespace: true, message: `请输入${label}` },
            { max, message: `${label}不能超过${max}个字符` },
          ]}
        >
          <Input maxLength={max} />
        </Form.Item>
      ))}
      <Form.Item
        name="swiftCode"
        label="SWIFT Code（外币国际结算）"
        rules={[{ max: 32, message: 'SWIFT Code不能超过32个字符' }]}
      >
        <Input maxLength={32} />
      </Form.Item>
      <Form.Item name="isDefault" valuePropName="checked">
        <Checkbox>
          {isReceivable ? '设为应收默认账户' : '设为应付默认账户'}
        </Checkbox>
      </Form.Item>
    </QuickCreateModal>
  );
}
