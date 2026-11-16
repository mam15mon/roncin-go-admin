import { PlusOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Checkbox, Form, Input, Select, Tag } from 'antd';
import React, {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { ProFormSearchableSelect, QuickCreateModal } from '@/components/ui';
import { defaultSelectFilterOption } from '@/components/ui/searchable-select/utils';
import { PartnerAssignmentRole, PartnerRoleType } from '@/enums.generated';
import { formatPersonnelLabel } from '@/features/personnel';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { history } from '@/router/history';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import { isWriteResponse } from '@/utils/api';
import type { RequestOptions } from '@/utils/requestClient';

export type PartnerSelectOption = {
  label: string;
  value: string | number;
  code?: string;
  isCasual?: boolean;
};

type PartnerQuickAddSelectProps = {
  /** 订单表单字段名，如 customerId / bookingAgentId / foreignAgentId。 */
  name: string;
  /** 显示名称，用于标签、入口与弹窗文案。 */
  displayName: string;
  /** 快捷新增时写入的单一伙伴角色。 */
  role: PartnerRoleType;
  /** 「添加公司详情」跳转的完整档案新建路由。 */
  createRoute: string;
  /** 按角色过滤的服务端关键字检索函数。 */
  searchPartners: (
    keyword?: string,
    requestOptions?: RequestOptions,
  ) => Promise<PartnerSelectOption[]>;
  /** 客户快建时提成责任岗位（业务/操作/客服）的人员候选项。 */
  staffOptions?: API.OrderPersonnelOption[];
  required?: boolean;
  /** 有效只读（无编辑动作权限或业务写入关闭）时隐藏快捷新增入口并禁用字段。 */
  disabled?: boolean;
  onPartnerChange?: (option: PartnerSelectOption | undefined) => void;
};

/**
 * 订单表单伙伴选择字段：远程关键字检索 + 下拉底部快捷新增。
 *
 * - 快捷新增仅录入公司抬头；客户角色额外展示提成责任岗位（业务/操作/客服）
 *   选择作为开单带入默认值，提成归属以订单人员为唯一真相，三岗选填；
 *   成功后回填选中；本地新选项与远程结果按伙伴
 *   ID 去重合并，查询缓存保留已选名称，避免当前值退化为只显示 ID。
 * - 组织切换时清理本地新选项并使在途响应失效，旧组织数据不写入新组织表单。
 */
export default function PartnerQuickAddSelect({
  name,
  displayName,
  role,
  createRoute,
  searchPartners,
  staffOptions,
  required = false,
  disabled = false,
  onPartnerChange,
}: PartnerQuickAddSelectProps) {
  const access = useAccess();
  const { initialState } = useInitialState();
  const orderForm = Form.useFormInstance();

  const [modalOpen, setModalOpen] = useState(false);
  const [selectOpen, setSelectOpen] = useState(false);
  const client = useQueryClient();
  const [keyword, setKeyword] = useState('');
  const [debouncedKeyword, setDebouncedKeyword] = useState('');
  const currentOrganizationId =
    initialState?.currentUser?.currentOrganization?.id;
  const resourceIdentity = `${currentOrganizationId ?? ''}:${name}:${role}`;
  const activeResourceRef = useRef(resourceIdentity);
  activeResourceRef.current = resourceIdentity;
  const organizationIdRef = useRef(currentOrganizationId);
  organizationIdRef.current = currentOrganizationId;
  const previousResourceRef = useRef(resourceIdentity);
  const createGuard = useAsyncGuard();
  const searchRef = useRef(searchPartners);
  searchRef.current = searchPartners;
  const selectedValue = Form.useWatch(name, {
    form: orderForm,
    preserve: true,
  });
  const scopeKey = useMemo(
    () =>
      [
        'orders',
        'partner-options',
        { organizationId: currentOrganizationId, name, role },
      ] as const,
    [currentOrganizationId, name, role],
  );
  const createdKey = useMemo(
    () => [...scopeKey, 'created'] as const,
    [scopeKey],
  );

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(keyword), 300);
    return () => clearTimeout(timer);
  }, [keyword]);

  const candidates = useQuery({
    queryKey: [...scopeKey, { keyword: debouncedKeyword }],
    enabled: Boolean(currentOrganizationId),
    queryFn: async ({ signal }) => {
      const items = await searchRef.current(debouncedKeyword || undefined, {
        signal,
        skipErrorHandler: true,
      });
      signal.throwIfAborted();
      if (!Array.isArray(items)) throw new Error('往来单位候选加载失败');
      return items;
    },
    meta: { errorMessage: '往来单位候选加载失败' },
  });
  const options = useMemo(() => {
    const created =
      client.getQueryData<PartnerSelectOption[]>(createdKey) ?? [];
    const selected = client
      .getQueriesData<PartnerSelectOption[]>({ queryKey: scopeKey })
      .flatMap(([, items]) => items ?? [])
      .filter((option) => option.value === selectedValue);
    return Array.from(
      new Map(
        [...selected, ...(candidates.data ?? []), ...created].map((option) => [
          option.value,
          option,
        ]),
      ).values(),
    );
  }, [candidates.data, client, createdKey, modalOpen, scopeKey, selectedValue]);
  const availableOptions = useMemo(
    () => new Map(options.map((option) => [option.value, option])),
    [options],
  );
  const canQuickAdd = Boolean(access.canCreatePartners) && !disabled;
  // 客户快建展示提成责任岗位（业务/操作/客服）供录入开单带入默认值；
  // 提成归属以订单人员为唯一真相，三岗选填。其他角色保持仅录抬头。
  const isCustomerRole = role === PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER;
  const staffSelectOptions = useMemo(
    () =>
      (staffOptions ?? []).flatMap((item) =>
        item.userId
          ? [{ label: formatPersonnelLabel(item), value: item.userId }]
          : [],
      ),
    [staffOptions],
  );

  useEffect(() => {
    if (canQuickAdd) return;
    createGuard.invalidate();
    setSelectOpen(false);
    setModalOpen(false);
  }, [canQuickAdd, createGuard]);

  // 组织/角色/字段变化只清本地交互与创建会话，查询结果由资源键隔离。
  useEffect(() => {
    if (previousResourceRef.current === resourceIdentity) return;
    previousResourceRef.current = resourceIdentity;
    createGuard.invalidate();
    orderForm?.setFieldValue(name, undefined);
    setKeyword('');
    setDebouncedKeyword('');
    setSelectOpen(false);
    setModalOpen(false);
  }, [createGuard, name, orderForm, resourceIdentity]);

  const quickAddFooter = useCallback(
    (menu: ReactNode) => (
      <>
        {menu}
        <Button
          type="text"
          htmlType="button"
          block
          icon={<PlusOutlined />}
          aria-label={`新增 ${displayName}`}
          style={{
            justifyContent: 'flex-start',
            height: 32,
            padding: '6px 12px',
            color: '#1677ff',
            fontSize: 12,
            background: '#f6faff',
            borderTop: '1px solid #f0f0f0',
            borderRadius: 0,
          }}
          onMouseDown={(event) => {
            event.preventDefault();
            event.stopPropagation();
          }}
          onClick={(event) => {
            event.stopPropagation();
            setSelectOpen(false);
            setModalOpen(true);
          }}
        >
          新增 {displayName}
        </Button>
      </>
    ),
    [displayName],
  );

  const fieldProps = useMemo(
    () => ({
      popupRender: canQuickAdd ? quickAddFooter : undefined,
      open: selectOpen,
      onOpenChange: setSelectOpen,
      onSearch: (value: string) => setKeyword(value.trim()),
      loading: candidates.isFetching,
      filterOption: false,
      optionRender: (option: {
        label?: ReactNode;
        value?: unknown;
        data?: unknown;
      }) => {
        const item = option.data as PartnerSelectOption | undefined;
        return (
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              width: '100%',
            }}
          >
            <span>{option.label}</span>
            {item?.isCasual && (
              <Tag color="orange" style={{ marginInlineEnd: 0 }}>
                散客
              </Tag>
            )}
          </div>
        );
      },
      onChange: (value: unknown, option: unknown) => {
        if (value === undefined || value === null) {
          onPartnerChange?.(undefined);
          return;
        }
        onPartnerChange?.(
          availableOptions.get(value as string | number) ??
            (option as PartnerSelectOption),
        );
      },
    }),
    [
      availableOptions,
      candidates.isFetching,
      canQuickAdd,
      onPartnerChange,
      quickAddFooter,
      selectOpen,
    ],
  );

  return (
    <>
      <ProFormSearchableSelect
        name={name}
        label={displayName}
        disabled={disabled || undefined}
        rules={
          required ? [{ required: true, message: `请选择${displayName}` }] : []
        }
        placeholder="请选择"
        options={options}
        fieldProps={fieldProps}
      />

      <QuickCreateModal<
        {
          legalName: string;
          isCasual?: boolean;
          assignSalesUser?: string;
          assignOperatorUser?: string;
          assignServiceUser?: string;
        },
        PartnerSelectOption
      >
        key={`${resourceIdentity}:${canQuickAdd ? 'enabled' : 'disabled'}`}
        centered
        title={`新增 ${displayName}`}
        open={modalOpen}
        width={520}
        okText="保存"
        initialValues={{
          isCasual: true,
        }}
        onCancel={() => setModalOpen(false)}
        extraAction={{
          text: '添加公司详情',
          onClick: (quickForm) => {
            const legalName = String(
              quickForm.getFieldValue('legalName') ?? '',
            ).trim();
            setModalOpen(false);
            history.push(
              legalName
                ? `${createRoute}?legalName=${encodeURIComponent(legalName)}`
                : createRoute,
            );
          },
        }}
        onSubmit={async (values) => {
          const organizationAtSubmit = organizationIdRef.current;
          const resourceAtSubmit = activeResourceRef.current;
          if (!organizationAtSubmit) {
            throw new Error('当前组织不可用，请刷新后重试');
          }
          // 提成责任岗位选填：仅提交已选择的岗位作为开单带入默认值。
          const selectedStaff: Array<
            [PartnerAssignmentRole, string | undefined]
          > = [
            [
              PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
              values.assignSalesUser,
            ],
            [
              PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
              values.assignOperatorUser,
            ],
            [
              PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
              values.assignServiceUser,
            ],
          ];
          const assignments: API.PartnerAssignmentInput[] | undefined =
            isCustomerRole
              ? selectedStaff.flatMap(([role, userId]) =>
                  userId ? [{ role, userId }] : [],
                )
              : undefined;
          await createGuard.run(
            ({ signal }) =>
              partnerServiceCreatePartner(
                {
                  // 客商代码留空表示未设置，仅作为操作人员搜索辅助，不做自动生成。
                  legalName: values.legalName.trim(),
                  roles: [{ type: role, enabled: true }],
                  isCasual: values.isCasual ?? true,
                  assignments,
                },
                { signal },
              ),
            (response) => {
              if (
                organizationAtSubmit !== organizationIdRef.current ||
                resourceAtSubmit !== activeResourceRef.current
              )
                return;
              // 请求层已提示的失败返回 undefined，保留表单以便重试。
              if (!response || response.success === false) return;
              const partner = response.data;
              if (!isWriteResponse(response, true) || !partner?.id) {
                throw new Error('创建结果缺少伙伴 ID，请重试');
              }
              const option = {
                label: partner.legalName ?? values.legalName.trim(),
                value: partner.id,
                code: partner.code,
                isCasual: partner.isCasual ?? values.isCasual ?? true,
              };
              orderForm?.setFieldValue(name, option.value);
              client.setQueryData<PartnerSelectOption[]>(
                createdKey,
                (current) => [
                  option,
                  ...(current ?? []).filter(
                    (existing) => existing.value !== option.value,
                  ),
                ],
              );
              onPartnerChange?.(option);
              setModalOpen(false);
            },
          );
          // 回填已在 guard 的同步 apply 内完成，避免 QuickCreateModal 在失效后再执行 onSuccess。
          return undefined;
        }}
      >
        <Form.Item
          name="legalName"
          label="公司抬头"
          rules={[
            { required: true, whitespace: true, message: '请输入公司抬头' },
            { max: 200, message: '不能超过 200 字符' },
          ]}
        >
          <Input placeholder="请输入公司抬头" maxLength={200} />
        </Form.Item>
        <Form.Item name="isCasual" valuePropName="checked">
          <Checkbox>单次合作往来单位（散客）</Checkbox>
        </Form.Item>
        {isCustomerRole && (
          <>
            <Form.Item label="业务人员" name="assignSalesUser">
              <Select
                showSearch={{ filterOption: defaultSelectFilterOption }}
                allowClear
                placeholder="选择人员"
                options={staffSelectOptions}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Form.Item label="操作人员" name="assignOperatorUser">
              <Select
                showSearch={{ filterOption: defaultSelectFilterOption }}
                allowClear
                placeholder="选择人员"
                options={staffSelectOptions}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Form.Item label="客服人员" name="assignServiceUser">
              <Select
                showSearch={{ filterOption: defaultSelectFilterOption }}
                allowClear
                placeholder="选择人员"
                options={staffSelectOptions}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </>
        )}
      </QuickCreateModal>
    </>
  );
}
