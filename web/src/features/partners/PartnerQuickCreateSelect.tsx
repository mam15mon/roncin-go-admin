import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Checkbox,
  ConfigProvider,
  Form,
  Input,
  Select,
  type SelectProps,
} from 'antd';
import { useEffect, useId, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  defaultSelectFilterOption,
  GuardedModalForm,
  QuickCreateSelectFooter,
} from '@/components/ui';
import { PartnerRoleType } from '@/enums.generated';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import { isWriteResponse } from '@/utils/api';
import type { PartnerOption } from './partnerOptions';

export type PartnerQuickCreateSelectProps<
  ValueType extends string | string[] = string,
> = Omit<SelectProps<ValueType, PartnerOption>, 'popupRender' | 'role'> & {
  sessionIdentity: string;
  active?: boolean;
  allowCreate?: boolean;
  role?: PartnerRoleType;
  defaultRole?: PartnerRoleType;
  createLabel?: string;
  isOptionDisabled?: (option: PartnerOption) => boolean;
  onCreated?: (option: PartnerOption, partner: API.Partner) => void;
  /** 财务等限定候选先经过原用途接口确认，不能以刚创建的档案绕过选择资格。 */
  resolveCreatedOption?: (
    partner: API.Partner,
    signal: AbortSignal,
  ) => Promise<PartnerOption | undefined>;
  request?: (
    keyword: string | undefined,
    signal: AbortSignal,
  ) => Promise<PartnerOption[]>;
};

/** 同一公司和编辑会话内保留新增候选；身份改变直接销毁旧的新增表单。 */
export function PartnerQuickCreateSelect<
  ValueType extends string | string[] = string,
>(props: PartnerQuickCreateSelectProps<ValueType>) {
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  return (
    <PartnerQuickCreateSelectLeaf<ValueType>
      key={`${initialState?.currentUser?.id ?? ''}:${organizationId ?? ''}:${props.sessionIdentity}:${props.role ?? ''}:${props.active !== false}`}
      {...props}
    />
  );
}

function PartnerQuickCreateSelectLeaf<ValueType extends string | string[]>({
  sessionIdentity: _sessionIdentity,
  active = true,
  allowCreate = true,
  role,
  defaultRole = PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
  createLabel = '快捷新增往来单位',
  isOptionDisabled,
  onCreated,
  resolveCreatedOption,
  request,
  options = [],
  disabled,
  value,
  onChange,
  mode,
  ...selectProps
}: PartnerQuickCreateSelectProps<ValueType>) {
  const access = useAccess();
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const { componentDisabled } = ConfigProvider.useConfig();
  const isDisabled = Boolean(disabled || componentDisabled);
  const canOperate = access.canOperateOrganization(organizationId);
  const canCreate =
    allowCreate &&
    active &&
    !isDisabled &&
    access.canCreatePartners &&
    canOperate;
  const current = useRef({
    canCreate,
    organizationId,
    onCreated,
    resolveCreatedOption,
    onChange,
    value,
    isOptionDisabled,
  });
  current.current = {
    canCreate,
    organizationId,
    onCreated,
    resolveCreatedOption,
    onChange,
    value,
    isOptionDisabled,
  };
  const [modalOpen, setModalOpen] = useState(false);
  const [selectOpen, setSelectOpen] = useState(false);
  const [createdOptions, setCreatedOptions] = useState<PartnerOption[]>([]);
  const [searchText, setSearchText] = useState('');
  const [keyword, setKeyword] = useState('');
  useEffect(() => {
    if (!canCreate) setModalOpen(false);
  }, [canCreate]);
  useEffect(() => {
    const timer = setTimeout(() => setKeyword(searchText.trim()), 300);
    return () => clearTimeout(timer);
  }, [searchText]);
  const instance = useId();
  const requestRef = useRef(request);
  requestRef.current = request;
  const candidates = useQuery({
    queryKey: [
      'partners',
      'quick-create-options',
      organizationId,
      _sessionIdentity,
      instance,
      role,
      keyword,
    ],
    enabled: active && canOperate && !isDisabled && Boolean(request),
    queryFn: ({ signal }) =>
      requestRef.current?.(keyword || undefined, signal) ?? Promise.resolve([]),
    meta: { errorMessage: '往来单位候选加载失败' },
  });
  const selectedOptions = useRef<PartnerOption[]>([]);
  const selectedValues: string[] = Array.isArray(value)
    ? value
    : value
      ? [value]
      : [];
  const merged = [
    ...createdOptions,
    ...(candidates.data ?? []),
    ...options,
    ...selectedOptions.current,
  ]
    .filter(
      (option, index, entries) =>
        entries.findIndex((entry) => entry.value === option.value) === index,
    )
    .map((option) => ({
      ...option,
      disabled: option.disabled || isOptionDisabled?.(option),
    }));
  selectedOptions.current = merged.filter((option) =>
    selectedValues.includes(option.value),
  );
  return (
    <>
      <Select<ValueType, PartnerOption>
        {...selectProps}
        style={{ width: '100%', ...selectProps.style }}
        showSearch={
          request
            ? {
                ...(typeof selectProps.showSearch === 'object'
                  ? selectProps.showSearch
                  : {}),
                filterOption: false,
                onSearch: (input) => {
                  setSearchText(input);
                  if (typeof selectProps.showSearch === 'object')
                    selectProps.showSearch.onSearch?.(input);
                },
              }
            : (selectProps.showSearch ?? {
                filterOption: defaultSelectFilterOption,
              })
        }
        mode={mode}
        allowClear={selectProps.allowClear ?? true}
        value={value}
        onChange={onChange}
        disabled={isDisabled}
        loading={selectProps.loading ?? candidates.isFetching}
        options={merged}
        open={selectOpen}
        onOpenChange={(open) => {
          setSelectOpen(open);
          selectProps.onOpenChange?.(open);
        }}
        popupRender={
          canCreate
            ? (menu) => (
                <QuickCreateSelectFooter
                  label={createLabel}
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
      {modalOpen && canCreate && (
        <PartnerCreateForm
          role={role}
          defaultRole={defaultRole}
          title={createLabel}
          onCancel={() => setModalOpen(false)}
          onCreate={async (partner, signal) => {
            if (
              !current.current.canCreate ||
              current.current.organizationId !== organizationId
            )
              return;
            const option = current.current.resolveCreatedOption
              ? await current.current.resolveCreatedOption(partner, signal)
              : {
                  value: partner.id || '',
                  label: partner.legalName || '',
                  name: partner.legalName,
                  code: partner.code,
                  isCasual: partner.isCasual ?? false,
                };
            if (
              !current.current.canCreate ||
              current.current.organizationId !== organizationId ||
              signal.aborted
            )
              return;
            if (
              !option?.value ||
              option.disabled ||
              current.current.isOptionDisabled?.(option)
            )
              throw new Error(
                '档案已创建，但不符合当前候选条件，请核对后重新选择',
              );
            setCreatedOptions((items) => [
              option,
              ...items.filter((item) => item.value !== option.value),
            ]);
            current.current.onCreated?.(option, partner);
            const nextValue =
              mode === 'multiple' || mode === 'tags'
                ? [
                    ...new Set([
                      ...(Array.isArray(current.current.value)
                        ? current.current.value
                        : []),
                      option.value,
                    ]),
                  ]
                : option.value;
            current.current.onChange?.(
              nextValue as ValueType,
              Array.isArray(nextValue)
                ? nextValue
                    .map((selected) =>
                      selected === option.value
                        ? option
                        : merged.find((entry) => entry.value === selected),
                    )
                    .filter((entry): entry is PartnerOption => Boolean(entry))
                : option,
            );
            setModalOpen(false);
          }}
        />
      )}
    </>
  );
}

function PartnerCreateForm({
  role,
  defaultRole,
  title,
  onCancel,
  onCreate,
}: {
  role?: PartnerRoleType;
  defaultRole: PartnerRoleType;
  title: string;
  onCancel: () => void;
  onCreate: (partner: API.Partner, signal: AbortSignal) => Promise<void>;
}) {
  const guard = useAsyncGuard();
  const client = useQueryClient();
  const { message } = App.useApp();
  const createdPartner = useRef<API.Partner | undefined>(undefined);
  const [created, setCreated] = useState(false);
  return (
    <GuardedModalForm<{
      legalName: string;
      role: PartnerRoleType;
      isCasual: boolean;
    }>
      title={title}
      open
      width={520}
      initialValues={{ role: role ?? defaultRole, isCasual: true }}
      modalProps={{ destroyOnHidden: true, onCancel }}
      onOpenChange={(open) => {
        if (!open) onCancel();
      }}
      submitter={{ searchConfig: { submitText: '保存并选用' } }}
      onFinish={async (values) => {
        let applied = false;
        try {
          await guard.run(
            async ({ signal }) => {
              if (!createdPartner.current) {
                const response = await partnerServiceCreatePartner(
                  {
                    legalName: values.legalName.trim(),
                    roles: [{ type: role ?? values.role, enabled: true }],
                    isCasual: values.isCasual,
                  },
                  { signal },
                );
                if (signal.aborted) return false;
                if (!isWriteResponse(response)) return false;
                if (!response.data?.id)
                  throw new Error('创建结果缺少伙伴 ID，请核对后重试');
                createdPartner.current = {
                  ...response.data,
                  legalName: response.data.legalName ?? values.legalName.trim(),
                };
                setCreated(true);
                void client.invalidateQueries({ queryKey: ['partners'] });
              }
              if (signal.aborted) return false;
              await onCreate(createdPartner.current, signal);
              return true;
            },
            (success) => {
              applied = success;
            },
          );
        } catch (error) {
          message.error(
            error instanceof Error ? error.message : '新增往来单位失败',
          );
        }
        return applied;
      }}
    >
      <Form.Item
        name="legalName"
        label="单位全称"
        rules={[
          { required: true, whitespace: true, message: '请输入单位全称' },
          { max: 200, message: '不能超过 200 字符' },
        ]}
      >
        <Input
          disabled={created}
          maxLength={200}
          placeholder="工商登记全称或客商名称"
        />
      </Form.Item>
      {!role && (
        <Form.Item
          name="role"
          label="客商类型"
          rules={[{ required: true, message: '请选择客商类型' }]}
        >
          <Select
            disabled={created}
            options={[
              {
                label: '客户',
                value: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              },
              {
                label: '供应商',
                value: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
              },
              {
                label: '国外代理',
                value: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
              },
            ]}
          />
        </Form.Item>
      )}
      <Form.Item name="isCasual" valuePropName="checked">
        <Checkbox disabled={created}>单次合作往来单位（散客）</Checkbox>
      </Form.Item>
    </GuardedModalForm>
  );
}
