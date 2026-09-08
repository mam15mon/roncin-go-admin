import { PlusOutlined } from '@ant-design/icons';
import { history, useAccess, useModel } from '@umijs/max';
import { Form, Input } from 'antd';
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
  // params 版本号：新建选项或组织变化后驱动 ProFormSelect 重新请求。
  const [optionsRevision, setOptionsRevision] = useState(0);
  const createdOptionsRef = useRef<PartnerSelectOption[]>([]);
  const requestSequenceRef = useRef(0);
  const organizationIdRef = useRef(
    initialState?.currentUser?.currentOrganization?.id,
  );
  const currentOrganizationId =
    initialState?.currentUser?.currentOrganization?.id;

  // 组织切换：关闭弹窗、清空本地新选项，并让在途请求/创建响应全部失效。
  useEffect(() => {
    if (organizationIdRef.current === currentOrganizationId) return;
    organizationIdRef.current = currentOrganizationId;
    requestSequenceRef.current += 1;
    createdOptionsRef.current = [];
    setOptionsRevision((revision) => revision + 1);
    setModalOpen(false);
  }, [currentOrganizationId]);

  const canQuickAdd = Boolean(access.canCreatePartners) && !disabled;

  const request = useCallback(
    async ({ keyWords }: { keyWords?: string }) => {
      const sequence = ++requestSequenceRef.current;
      const organizationAtRequest = organizationIdRef.current;
      const remote = await searchPartners(keyWords);
      if (
        sequence !== requestSequenceRef.current ||
        organizationAtRequest !== organizationIdRef.current
      ) {
        return [];
      }
      const local = createdOptionsRef.current;
      const remoteIds = new Set(remote.map((option) => option.value));
      return [...local.filter((o) => !remoteIds.has(o.value)), ...remote];
    },
    [searchPartners],
  );

  const quickAddFooter = useCallback(
    (menu: ReactNode) => (
      <>
        {menu}
        <div
          style={{
            padding: '6px 12px',
            cursor: 'pointer',
            color: '#1677ff',
            fontSize: 12,
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            background: '#f6faff',
            borderTop: '1px solid #f0f0f0',
          }}
          onMouseDown={(event) => {
            event.preventDefault();
            event.stopPropagation();
          }}
          onClick={(event) => {
            event.stopPropagation();
            setModalOpen(true);
          }}
        >
          <PlusOutlined /> 新增 {displayName}
        </div>
      </>
    ),
    [displayName],
  );

  const fieldProps = useMemo(
    () => ({
      popupRender: canQuickAdd ? quickAddFooter : undefined,
      onChange: (_value: unknown, option: unknown) => {
        onPartnerChange?.(option as PartnerSelectOption | undefined);
      },
    }),
    [canQuickAdd, quickAddFooter, onPartnerChange],
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
        params={{ revision: optionsRevision }}
        request={request}
        fieldProps={fieldProps}
      />

      <QuickCreateModal<
        { legalName: string; unifiedSocialCreditCode?: string },
        PartnerSelectOption
      >
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
          const sequence = requestSequenceRef.current;
          const organizationAtSubmit = organizationIdRef.current;
          const response = await partnerServiceCreatePartner({
            // 客商代码留空由服务端按组织内唯一规则自动生成。
            legalName: values.legalName.trim(),
            unifiedSocialCreditCode: values.unifiedSocialCreditCode
              ?.trim()
              .toUpperCase(),
            roles: [{ type: role, enabled: true }],
          });
          const partner = response.data;
          if (!partner?.id) {
            throw new Error('创建结果缺少伙伴 ID，请重试');
          }
          // 组织已切换：丢弃迟到结果，不回填新组织表单。
          if (
            sequence !== requestSequenceRef.current ||
            organizationAtSubmit !== organizationIdRef.current
          ) {
            return undefined;
          }
          return {
            label: partner.legalName ?? values.legalName.trim(),
            value: partner.id,
            code: partner.code,
          };
        }}
        onSuccess={(option) => {
          orderForm?.setFieldValue(name, option.value);
          createdOptionsRef.current = [
            option,
            ...createdOptionsRef.current.filter(
              (existing) => existing.value !== option.value,
            ),
          ];
          setOptionsRevision((revision) => revision + 1);
          onPartnerChange?.(option);
          setModalOpen(false);
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
              typeof value === 'string' ? value.toUpperCase() : value
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
