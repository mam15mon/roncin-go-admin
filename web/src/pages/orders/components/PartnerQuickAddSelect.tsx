import { PlusOutlined } from '@ant-design/icons';
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
import { PartnerAssignmentRole, PartnerRoleType } from '@/enums.generated';
import { formatPersonnelLabel } from '@/features/personnel';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { useLatestAsync } from '@/hooks/useLatestAsync';
import { history } from '@/router/history';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';

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
  searchPartners: (keyword?: string) => Promise<PartnerSelectOption[]>;
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
 *   ID 去重合并，并通过 params 版本号触发重载，避免当前值退化为只显示 ID。
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
  const [options, setOptions] = useState<PartnerSelectOption[]>([]);
  const currentOrganizationId =
    initialState?.currentUser?.currentOrganization?.id;
  const createdOptionsRef = useRef<PartnerSelectOption[]>([]);
  const availableOptionsRef = useRef<Map<string | number, PartnerSelectOption>>(
    new Map(),
  );
  const optionsOrganizationIdRef = useRef(currentOrganizationId);
  const organizationIdRef = useRef(currentOrganizationId);
  const previousOrganizationIdRef = useRef(currentOrganizationId);
  const latestSearch = useLatestAsync();
  const createGuard = useAsyncGuard();
  organizationIdRef.current = currentOrganizationId;
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
    latestSearch.cancel();
    createGuard.invalidate();
    setSelectOpen(false);
    setModalOpen(false);
  }, [canQuickAdd, createGuard, latestSearch]);

  const loadOptions = useCallback(
    (keyWords?: string) => {
      const organizationAtRequest = organizationIdRef.current;
      if (!organizationAtRequest) {
        setOptions([]);
        return Promise.resolve();
      }

      return latestSearch.run(
        async () => {
          try {
            return {
              organizationId: organizationAtRequest,
              remote: await searchPartners(keyWords),
            };
          } catch (error) {
            // 组织已经变化时，旧请求的错误也不能反馈到新工作区。
            if (organizationAtRequest !== organizationIdRef.current) {
              return { organizationId: organizationAtRequest, remote: [] };
            }
            throw error;
          }
        },
        ({ organizationId, remote }) => {
          // Hook 负责生命周期和 latest；组织身份仍是业务数据隔离条件。
          if (organizationId !== organizationIdRef.current) return;
          const local = createdOptionsRef.current;
          const localIds = new Set(local.map((option) => option.value));
          const merged = [
            ...local,
            ...remote.filter((option) => !localIds.has(option.value)),
          ];
          availableOptionsRef.current = new Map(
            merged.map((option) => [option.value, option]),
          );
          optionsOrganizationIdRef.current = organizationId;
          setOptions(merged);
        },
      );
    },
    [latestSearch, searchPartners],
  );

  const triggerLoadOptions = useCallback(
    (keyWords?: string) => {
      // requestErrorConfig 负责当前请求的用户提示；这里仅避免 Select 事件留下未处理 Promise。
      void loadOptions(keyWords).catch(() => undefined);
    },
    [loadOptions],
  );

  // 受控 options 不会像 ProForm 的 request 一样自动在挂载时加载；保留首批
  // 候选项和已选值标签的原有回显能力。
  useEffect(() => {
    triggerLoadOptions();
  }, [triggerLoadOptions]);

  // 组织切换：清除本地候选项、使在途请求/创建响应失效，并加载新组织首批候选项。
  useEffect(() => {
    if (previousOrganizationIdRef.current === currentOrganizationId) return;
    previousOrganizationIdRef.current = currentOrganizationId;
    latestSearch.cancel();
    createGuard.invalidate();
    createdOptionsRef.current = [];
    availableOptionsRef.current.clear();
    optionsOrganizationIdRef.current = currentOrganizationId;
    setOptions([]);
    orderForm?.setFieldValue(name, undefined);
    setSelectOpen(false);
    setModalOpen(false);
    triggerLoadOptions();
  }, [
    createGuard,
    currentOrganizationId,
    latestSearch,
    name,
    orderForm,
    triggerLoadOptions,
  ]);

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
      onSearch: triggerLoadOptions,
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
        if (optionsOrganizationIdRef.current !== organizationIdRef.current) {
          return;
        }
        onPartnerChange?.(
          availableOptionsRef.current.get(value as string | number) ??
            (option as PartnerSelectOption),
        );
      },
    }),
    [
      canQuickAdd,
      onPartnerChange,
      quickAddFooter,
      selectOpen,
      triggerLoadOptions,
    ],
  );

  return (
    <>
      <ProFormSearchableSelect
        name={name}
        label={displayName}
        disabled={disabled}
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
        key={`${currentOrganizationId ?? 'no-organization'}:${canQuickAdd ? 'enabled' : 'disabled'}`}
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
              if (organizationAtSubmit !== organizationIdRef.current) return;
              // 请求层已提示的失败返回 undefined，保留表单以便重试。
              if (!response) return;
              const partner = response.data;
              if (!partner?.id) {
                throw new Error('创建结果缺少伙伴 ID，请重试');
              }
              const option = {
                label: partner.legalName ?? values.legalName.trim(),
                value: partner.id,
                code: partner.code,
                isCasual: partner.isCasual ?? values.isCasual ?? true,
              };
              orderForm?.setFieldValue(name, option.value);
              createdOptionsRef.current = [
                option,
                ...createdOptionsRef.current.filter(
                  (existing) => existing.value !== option.value,
                ),
              ];
              availableOptionsRef.current.set(option.value, option);
              optionsOrganizationIdRef.current = organizationIdRef.current;
              setOptions((current) => [
                option,
                ...current.filter(
                  (existing) => existing.value !== option.value,
                ),
              ]);
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
                showSearch
                allowClear
                placeholder="选择人员"
                options={staffSelectOptions}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Form.Item label="操作人员" name="assignOperatorUser">
              <Select
                showSearch
                allowClear
                placeholder="选择人员"
                options={staffSelectOptions}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Form.Item label="客服人员" name="assignServiceUser">
              <Select
                showSearch
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
