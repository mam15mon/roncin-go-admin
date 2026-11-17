import { useQueryClient } from '@tanstack/react-query';
import { ConfigProvider, Form, Input, Select, type SelectProps } from 'antd';
import { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { QuickCreateModal, QuickCreateSelectFooter } from '@/components/ui';
import { defaultSelectFilterOption } from '@/components/ui/searchable-select/utils';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { feeCatalogServiceCreateTaxableService } from '@/services/roncin/feeCatalogService';
import { isWriteResponse } from '@/utils/api';

const taxRatePattern =
  /^(100(?:\.0{1,2})?|(?:[0-9]|[1-9][0-9])(?:\.[0-9]{1,2})?)$/;

type TaxableServiceOption = {
  value: string;
  label: string;
  name?: string;
  code?: string;
};
type Props = Omit<
  SelectProps<string, TaxableServiceOption>,
  'options' | 'popupRender'
> & {
  taxableServices: API.TaxableService[];
  sessionIdentity: string;
  active?: boolean;
  onCreated?: (item: API.TaxableService) => void;
};

/** 公司应税劳务目录：同一创建入口供费用设置与科目快建使用。 */
export function TaxableServiceQuickCreateSelect(props: Props) {
  const { initialState } = useInitialState();
  const access = useAccess();
  const { componentDisabled } = ConfigProvider.useConfig();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const disabled = Boolean(props.disabled || componentDisabled);
  const canCreate = Boolean(
    organizationId &&
      access.canOperateBusiness &&
      access.canCreateFeeSettings &&
      props.active !== false &&
      !disabled,
  );
  return (
    <TaxableServiceSelectLeaf
      key={`${organizationId}:${props.sessionIdentity}:${canCreate}`}
      {...props}
      disabled={disabled}
      canCreate={canCreate}
    />
  );
}

function TaxableServiceSelectLeaf({
  taxableServices,
  sessionIdentity: _sessionIdentity,
  active: _active,
  onCreated,
  canCreate,
  onChange,
  onOpenChange,
  ...selectProps
}: Props & { canCreate: boolean }) {
  const [created, setCreated] = useState<API.TaxableService[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [selectOpen, setSelectOpen] = useState(false);
  const createGuard = useAsyncGuard();
  const client = useQueryClient();
  const callbacksRef = useRef({ onChange, onCreated });
  callbacksRef.current = { onChange, onCreated };
  const options = [...created, ...taxableServices]
    .filter(
      (item, index, all) =>
        item.id && all.findIndex((other) => other.id === item.id) === index,
    )
    .map((item) => ({
      value: item.id as string,
      label: item.goodsCode
        ? `${item.name || ''} (${item.goodsCode})`
        : item.name || '',
      name: item.name,
      code: item.goodsCode,
    }));
  return (
    <>
      <Select<string, TaxableServiceOption>
        allowClear
        placeholder="请选择货物或应税劳务"
        showSearch={{ filterOption: defaultSelectFilterOption }}
        {...selectProps}
        style={{ width: '100%', ...selectProps.style }}
        options={options}
        open={selectProps.open ?? selectOpen}
        onOpenChange={(next) => {
          setSelectOpen(next);
          onOpenChange?.(next);
        }}
        onChange={onChange}
        popupRender={
          canCreate
            ? (menu) => (
                <QuickCreateSelectFooter
                  label="新增货物或应税劳务"
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
      {canCreate && modalOpen && (
        <QuickCreateModal<API.CreateTaxableServiceRequest, API.TaxableService>
          title="新增货物或应税劳务"
          open={modalOpen}
          initialValues={{ defaultTaxRate: '0' }}
          onCancel={() => {
            createGuard.invalidate();
            setModalOpen(false);
          }}
          onSubmit={async (values) => {
            await createGuard.run(
              ({ signal }) =>
                feeCatalogServiceCreateTaxableService(
                  {
                    name: values.name.trim(),
                    shortName: values.shortName?.trim() || undefined,
                    goodsCode: values.goodsCode?.trim() || undefined,
                    defaultTaxRate: values.defaultTaxRate,
                  },
                  { signal },
                ),
              (response) => {
                if (!response || response.success === false) return;
                if (!isWriteResponse(response, true))
                  throw new Error('创建结果缺少应税劳务标识，请核对后重试');
                const item = response.data as API.TaxableService & {
                  id: string;
                };
                const resolvedItem = {
                  ...item,
                  name: item.name || values.name.trim(),
                };
                const option = {
                  value: item.id,
                  label: item.goodsCode
                    ? `${item.name || values.name} (${item.goodsCode})`
                    : item.name || values.name,
                  name: item.name,
                  code: item.goodsCode,
                };
                setCreated((current) => [
                  resolvedItem,
                  ...current.filter((entry) => entry.id !== item.id),
                ]);
                callbacksRef.current.onChange?.(item.id, option);
                callbacksRef.current.onCreated?.(resolvedItem);
                void client.invalidateQueries({
                  queryKey: ['finance', 'fee-settings-reference-options'],
                });
                void client.invalidateQueries({
                  queryKey: ['finance', 'fee-catalog', 'taxable-services'],
                });
                void client.invalidateQueries({
                  queryKey: ['orders', 'taxable-services'],
                });
                setModalOpen(false);
              },
            );
            return undefined;
          }}
        >
          <Form.Item
            name="name"
            label="货物或应税劳务名称"
            rules={[
              { required: true, whitespace: true, message: '请输入名称' },
              { max: 128 },
            ]}
          >
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="shortName" label="简称" rules={[{ max: 64 }]}>
            <Input maxLength={64} />
          </Form.Item>
          <Form.Item name="goodsCode" label="商品编码" rules={[{ max: 64 }]}>
            <Input maxLength={64} />
          </Form.Item>
          <Form.Item
            name="defaultTaxRate"
            label="默认税率（%）"
            rules={[
              { required: true, message: '请输入默认税率' },
              { pattern: taxRatePattern, message: '请输入0–100，最多两位小数' },
            ]}
          >
            <Input inputMode="decimal" />
          </Form.Item>
        </QuickCreateModal>
      )}
    </>
  );
}
