import { PlusOutlined } from '@ant-design/icons';
import { history, useAccess, useModel } from '@umijs/max';
import { Button, Form, Input } from 'antd';
import React, {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { ProFormSearchableSelect, QuickCreateModal } from '@/components/ui';
import type { PartnerRoleType } from '@/enums.generated';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { useLatestAsync } from '@/hooks/useLatestAsync';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';

export type PartnerSelectOption = {
  label: string;
  value: string | number;
  code?: string;
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
  required?: boolean;
  /** 客户/供应商角色创建时必须提供纳税人识别号（后端业务规则），国外代理不需要。 */
  taxIdentifierRequired?: boolean;
  /** 有效只读（无编辑动作权限或业务写入关闭）时隐藏快捷新增入口并禁用字段。 */
  disabled?: boolean;
  onPartnerChange?: (option: PartnerSelectOption | undefined) => void;
};

/**
 * 订单表单伙伴选择字段：远程关键字检索 + 下拉底部快捷新增。
 *
 * - 快捷新增仅录入公司抬头，成功后回填选中；本地新选项与远程结果按伙伴 ID
 *   去重合并，并通过 params 版本号触发重载，避免当前值退化为只显示 ID。
 * - 组织切换时清理本地新选项并使在途响应失效，旧组织数据不写入新组织表单。
 */
export default function PartnerQuickAddSelect({
  name,
  displayName,
  role,
  createRoute,
  searchPartners,
  required = false,
  taxIdentifierRequired = false,
  disabled = false,
  onPartnerChange,
}: PartnerQuickAddSelectProps) {
  const access = useAccess();
  const { initialState } = useModel('@@initialState');
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
        { legalName: string; unifiedSocialCreditCode?: string },
        PartnerSelectOption
      >
        key={`${currentOrganizationId ?? 'no-organization'}:${canQuickAdd ? 'enabled' : 'disabled'}`}
        centered
        title={`新增 ${displayName}`}
        open={modalOpen}
        width={520}
        okText="保存"
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
          await createGuard.run(
            ({ signal }) =>
              partnerServiceCreatePartner(
                {
                  // 客商代码留空由服务端按组织内唯一规则自动生成。
                  legalName: values.legalName.trim(),
                  unifiedSocialCreditCode: values.unifiedSocialCreditCode
                    ?.trim()
                    .toUpperCase(),
                  roles: [{ type: role, enabled: true }],
                },
                { signal },
              ),
            (response) => {
              if (organizationAtSubmit !== organizationIdRef.current) return;
              const partner = response.data;
              if (!partner?.id) {
                throw new Error('创建结果缺少伙伴 ID，请重试');
              }
              const option = {
                label: partner.legalName ?? values.legalName.trim(),
                value: partner.id,
                code: partner.code,
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
        {taxIdentifierRequired && (
          <Form.Item
            name="unifiedSocialCreditCode"
            label="纳税人识别号"
            tooltip="客户/供应商往来单位必须有统一社会信用代码（后端业务规则）"
            normalize={(value) =>
              typeof value === 'string' ? value.trim().toUpperCase() : value
            }
            rules={[
              {
                required: true,
                whitespace: true,
                message: '请输入纳税人识别号',
              },
              {
                pattern: /^[0-9ABCDEFGHJKLMNPQRTUWXY]{18}$/,
                message: '请输入正确的18位统一社会信用代码',
              },
            ]}
          >
            <Input placeholder="18 位统一社会信用代码" maxLength={18} />
          </Form.Item>
        )}
      </QuickCreateModal>
    </>
  );
}
