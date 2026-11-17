import { useQueryClient } from '@tanstack/react-query';
import {
  Checkbox,
  ConfigProvider,
  Form,
  Input,
  Select,
  type SelectProps,
} from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { QuickCreateModal, QuickCreateSelectFooter } from '@/components/ui';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { partnerServiceCreatePartnerInvoiceProfile } from '@/services/roncin/partnerService';
import { isWriteResponse } from '@/utils/api';
import {
  DEFAULT_INVOICE_TYPE,
  INVOICE_TYPE_OPTIONS,
  isSpecialInvoiceType,
} from './invoiceTypes';

type Props = Omit<SelectProps<string>, 'options' | 'onChange'> & {
  organizationId: string;
  settlementPartyId: string;
  sessionIdentity: string;
  profiles: API.FinanceInvoiceProfileOption[];
  onChange?: (value: string) => void;
  onProfileChange: (
    profile: API.FinanceInvoiceProfileOption | undefined,
  ) => void;
};
type Values = Omit<API.CreatePartnerInvoiceProfileRequest, 'partnerId'>;

/** 每张登记记录持有独立新增会话，不把新增抬头写入另一张发票。 */
export default function InvoiceProfileQuickSelect(props: Props) {
  const identity = [
    props.sessionIdentity,
    props.organizationId,
    props.settlementPartyId,
  ].join(':');
  return <ProfileSelectLeaf key={identity} {...props} />;
}

function ProfileSelectLeaf({
  organizationId,
  settlementPartyId,
  sessionIdentity,
  profiles,
  onChange,
  onProfileChange,
  disabled,
  ...selectProps
}: Props) {
  const access = useAccess();
  const { componentDisabled } = ConfigProvider.useConfig();
  const client = useQueryClient();
  const [open, setOpen] = useState(false);
  const [selectOpen, setSelectOpen] = useState(false);
  const [created, setCreated] = useState<API.FinanceInvoiceProfileOption[]>([]);
  const canCreate = Boolean(
    !disabled &&
      !componentDisabled &&
      access.canCreateFinanceInvoices &&
      access.canUpdatePartners &&
      access.canOperateOrganization(organizationId) &&
      settlementPartyId,
  );
  const current = useRef({ onChange, onProfileChange });
  current.current = { onChange, onProfileChange };
  const options = [
    ...created,
    ...profiles.filter(
      (profile) => !created.some((item) => item.id === profile.id),
    ),
  ];
  useEffect(() => {
    if (!canCreate) setOpen(false);
  }, [canCreate]);
  return (
    <>
      <Select<string>
        {...selectProps}
        disabled={disabled || undefined}
        open={selectOpen}
        onOpenChange={setSelectOpen}
        options={options.map((profile) => ({
          value: profile.id,
          label: `${profile.invoiceTitle ?? ''} · ${profile.taxpayerIdentificationNo ?? ''}`,
        }))}
        onChange={(value) => {
          onChange?.(value);
          onProfileChange(options.find((profile) => profile.id === value));
        }}
        popupRender={
          canCreate
            ? (menu) => (
                <QuickCreateSelectFooter
                  label="新增开票抬头"
                  onCreate={() => {
                    setSelectOpen(false);
                    setOpen(true);
                  }}
                >
                  {menu}
                </QuickCreateSelectFooter>
              )
            : undefined
        }
      />
      <ProfileCreateModal
        key={`${sessionIdentity}:${open}:${canCreate}`}
        open={open && canCreate}
        partnerId={settlementPartyId}
        onCancel={() => setOpen(false)}
        onCreated={(profile) => {
          setCreated((items) => [
            profile,
            ...items.filter((item) => item.id !== profile.id),
          ]);
          current.current.onChange?.(profile.id || '');
          current.current.onProfileChange(profile);
          setOpen(false);
          void client.invalidateQueries({
            queryKey: ['finance', 'invoice-registration-profiles'],
          });
        }}
      />
    </>
  );
}

function ProfileCreateModal({
  open,
  partnerId,
  onCancel,
  onCreated,
}: {
  open: boolean;
  partnerId: string;
  onCancel: () => void;
  onCreated: (profile: API.FinanceInvoiceProfileOption) => void;
}) {
  const guard = useAsyncGuard();
  const active = useRef(open);
  active.current = open;
  useEffect(() => {
    if (!open) guard.invalidate();
  }, [open, guard]);
  return (
    <QuickCreateModal<Values, never>
      title="新增开票抬头"
      open={open}
      width={640}
      initialValues={{
        defaultInvoiceType: DEFAULT_INVOICE_TYPE,
        isDefault: false,
      }}
      onCancel={onCancel}
      onSubmit={async (values) => {
        if (!active.current) return undefined;
        await guard.run(
          ({ signal }) =>
            partnerServiceCreatePartnerInvoiceProfile(
              { partnerId },
              {
                partnerId,
                invoiceTitle: values.invoiceTitle.trim(),
                taxpayerIdentificationNo:
                  values.taxpayerIdentificationNo.trim(),
                defaultInvoiceType: values.defaultInvoiceType,
                registeredAddress: values.registeredAddress?.trim() || '',
                registeredPhone: values.registeredPhone?.trim() || '',
                bankName: values.bankName?.trim() || '',
                bankAccount: values.bankAccount?.trim() || '',
                isDefault: Boolean(values.isDefault),
              },
              { signal },
            ),
          (response) => {
            if (!active.current || !response || response.success === false)
              return;
            if (!response.data?.id)
              throw new Error('创建结果缺少开票抬头 ID，请核对后重试');
            if (!isWriteResponse(response, true)) return;
            onCreated({
              id: response.data.id,
              invoiceTitle:
                response.data.invoiceTitle || values.invoiceTitle.trim(),
              taxpayerIdentificationNo:
                response.data.taxpayerIdentificationNo ||
                values.taxpayerIdentificationNo.trim(),
              defaultInvoiceType:
                response.data.defaultInvoiceType || values.defaultInvoiceType,
              isDefault: Boolean(response.data.isDefault),
            });
          },
        );
        return undefined;
      }}
    >
      <Form.Item
        name="invoiceTitle"
        label="发票抬头"
        rules={[
          { required: true, whitespace: true, message: '请输入发票抬头' },
          { max: 200, message: '发票抬头不能超过200个字符' },
        ]}
      >
        <Input maxLength={200} />
      </Form.Item>
      <Form.Item
        name="taxpayerIdentificationNo"
        label="纳税人识别号"
        rules={[
          { required: true, whitespace: true, message: '请输入纳税人识别号' },
          { max: 64, message: '纳税人识别号不能超过64个字符' },
        ]}
      >
        <Input maxLength={64} />
      </Form.Item>
      <Form.Item
        name="defaultInvoiceType"
        label="默认发票类型"
        rules={[{ required: true, message: '请选择默认发票类型' }]}
      >
        <Select options={[...INVOICE_TYPE_OPTIONS]} />
      </Form.Item>
      <Form.Item
        noStyle
        shouldUpdate={(previous, next) =>
          previous.defaultInvoiceType !== next.defaultInvoiceType
        }
      >
        {({ getFieldValue }) => {
          const required = isSpecialInvoiceType(
            getFieldValue('defaultInvoiceType'),
          );
          return (
            [
              ['registeredAddress', '税务登记地址', 500],
              ['registeredPhone', '税务登记电话', 50],
              ['bankName', '开户银行', 200],
              ['bankAccount', '银行账号', 100],
            ] as const
          ).map(([name, label, max]) => (
            <Form.Item
              key={name}
              name={name}
              label={label}
              rules={[
                ...(required
                  ? [
                      {
                        required: true,
                        whitespace: true,
                        message: `专票必须填写${label}`,
                      },
                    ]
                  : []),
                { max, message: `${label}不能超过${max}个字符` },
              ]}
            >
              <Input maxLength={max} />
            </Form.Item>
          ));
        }}
      </Form.Item>
      <Form.Item name="isDefault" valuePropName="checked">
        <Checkbox>设为默认抬头</Checkbox>
      </Form.Item>
    </QuickCreateModal>
  );
}
