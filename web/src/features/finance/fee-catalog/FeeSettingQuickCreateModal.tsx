import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Col, ConfigProvider, Form, Input, Row, Select } from 'antd';
import { useRef } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  MODAL_SIZE,
  QuickCreateModal,
  SearchableSelect,
} from '@/components/ui';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import {
  feeCatalogServiceCreateFeeSetting,
  feeCatalogServiceListTaxableServices,
} from '@/services/roncin/feeCatalogService';
import { ensureListResponse, isWriteResponse, unwrapList } from '@/utils/api';
import { TaxableServiceQuickCreateSelect } from './TaxableServiceQuickCreateSelect';

type Props = {
  open: boolean;
  onCancel: () => void;
  onSuccess: (newFeeSetting: API.FeeSetting) => void;
  currencies: API.Currency[];
  billingUnits: API.BillingUnit[];
  taxableServices?: API.TaxableService[];
  sessionIdentity?: string;
};

export function FeeSettingQuickCreateModal(props: Props) {
  const { initialState } = useInitialState();
  const access = useAccess();
  const { componentDisabled } = ConfigProvider.useConfig();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const canCreate = Boolean(
    organizationId &&
      access.canOperateBusiness &&
      access.canCreateFeeSettings &&
      !componentDisabled,
  );
  return (
    <FeeSettingCreateLeaf
      key={`${organizationId}:${props.sessionIdentity || ''}:${props.open}:${canCreate}`}
      {...props}
      canCreate={canCreate}
      organizationId={organizationId}
    />
  );
}

function FeeSettingCreateLeaf({
  open,
  onCancel,
  onSuccess,
  currencies,
  billingUnits,
  taxableServices,
  sessionIdentity = 'fee-setting',
  canCreate,
  organizationId,
}: Props & { canCreate: boolean; organizationId?: string }) {
  const createGuard = useAsyncGuard();
  const client = useQueryClient();
  const callbacksRef = useRef({ onSuccess });
  callbacksRef.current = { onSuccess };
  const taxableQuery = useQuery({
    queryKey: ['finance', 'fee-catalog', 'taxable-services', organizationId],
    enabled: open && canCreate && taxableServices === undefined,
    queryFn: async ({ signal }) =>
      unwrapList(
        ensureListResponse(
          await feeCatalogServiceListTaxableServices({
            signal,
            skipErrorHandler: true,
          }),
          '应税劳务加载失败',
        ),
      ).filter((item) => item.enabled),
    meta: { errorMessage: '应税劳务加载失败' },
  });
  return (
    <QuickCreateModal<API.CreateFeeSettingRequest, API.FeeSetting>
      title="快捷新增费用科目"
      open={open && canCreate}
      width={MODAL_SIZE.SM}
      onCancel={() => {
        createGuard.invalidate();
        onCancel();
      }}
      onSubmit={async (values) => {
        await createGuard.run(
          ({ signal }) =>
            feeCatalogServiceCreateFeeSetting(
              {
                ...values,
                feeCode: values.feeCode.trim().toUpperCase(),
                nameZh: values.nameZh.trim(),
                nameEn: values.nameEn?.trim() || undefined,
              },
              { signal },
            ),
          (response) => {
            if (!response || response.success === false) return;
            if (!isWriteResponse(response, true))
              throw new Error('创建结果缺少费用科目标识，请核对后重试');
            callbacksRef.current.onSuccess(response.data as API.FeeSetting);
            void client.invalidateQueries({
              queryKey: ['orders', 'fee-options'],
            });
          },
        );
        return undefined;
      }}
    >
      <Row gutter={16}>
        <Col span={12}>
          <Form.Item
            name="feeCode"
            label="科目代码"
            rules={[
              { required: true, whitespace: true, message: '请输入科目代码' },
              {
                pattern: /^[A-Z0-9_]{2,32}$/i,
                message: '请输入2–32位字母、数字或下划线',
              },
            ]}
          >
            <Input maxLength={32} placeholder="例如：THC、OFRT" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="nameZh"
            label="科目中文名称"
            rules={[
              { required: true, whitespace: true, message: '请输入中文名称' },
              { max: 64 },
            ]}
          >
            <Input maxLength={64} placeholder="例如：码头操作费" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="nameEn"
            label="英文名称（选填）"
            rules={[{ max: 128 }]}
          >
            <Input maxLength={128} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="defaultCurrency"
            label="默认币种"
            rules={[{ required: true, message: '请选择币种' }]}
          >
            <SearchableSelect
              options={currencies.map((item) => ({
                value: item.code || '',
                label: `${item.code} (${item.name})`,
              }))}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="billingUnitId"
            label="默认计费单位"
            rules={[{ required: true, message: '请选择计费单位' }]}
          >
            <SearchableSelect
              options={billingUnits.map((item) => ({
                value: item.id || '',
                label: item.name || '',
              }))}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="taxRate"
            label="默认增值税税率"
            rules={[{ required: true, message: '请选择税率' }]}
          >
            <Select
              options={[
                { label: '0% (零税率/免税)', value: '0' },
                { label: '6% (现代服务业/货运代理)', value: '6' },
                { label: '9% (基础交通运输)', value: '9' },
                { label: '13% (商品贸易/修箱)', value: '13' },
              ]}
            />
          </Form.Item>
        </Col>
        <Col span={24}>
          <Form.Item
            name="taxableServiceId"
            label="应税服务类别"
            rules={[{ required: true, message: '请选择应税服务类别' }]}
          >
            <TaxableServiceQuickCreateSelect
              taxableServices={taxableServices ?? taxableQuery.data ?? []}
              sessionIdentity={sessionIdentity}
              active={open && canCreate}
              loading={taxableQuery.isFetching}
              placeholder="选择税目分类"
            />
          </Form.Item>
        </Col>
      </Row>
    </QuickCreateModal>
  );
}
