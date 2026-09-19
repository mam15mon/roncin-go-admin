import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormRadio,
  ProFormText,
} from '@ant-design/pro-components';
import { App, Typography } from 'antd';
import React, { useEffect, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { DingTalkInvitationKind } from '@/enums.generated';
import { adminServiceCreateDingTalkInvitation } from '@/services/roncin/adminService';
import {
  CHINA_MOBILE_PATTERN,
  fetchRolesForOrganization,
  INVITATION_DEFAULT_TTL_HOURS,
  INVITATION_TTL_OPTIONS,
  organizationSelectOptions,
  roleSelectOptions,
} from './constants';

const { Text } = Typography;

export type InvitationFormValues = {
  kind?: number;
  mobile?: string;
  organizationId: string;
  roleId?: string;
  displayName?: string;
  expiresInHours?: number;
};

interface InvitationFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  formRef: React.RefObject<ProFormInstance | undefined>;
  /** 可选目标组织：全局组织读取权限者拿全量列表，普通管理员只有当前组织。 */
  organizations: API.AdminOrganization[];
  /** 当前组织 ID，作为目标组织默认值。 */
  currentOrganizationId?: string;
  /** 是否具备角色读取权限（角色下拉数据源门控）。 */
  canReadRoles: boolean;
  onReload: () => void;
  /** 创建成功后的回调（例如弹出二维码查看模态）。 */
  onSuccess?: (
    invitation: API.DingTalkInvitation,
    invitationUrl?: string,
  ) => void;
}

/**
 * 新建扫码邀请模态：支持「通用入职码（多人扫码·审批入职）」与「定向邀请码（单人扫码·免审激活）」。
 */
export default function InvitationFormModal({
  open,
  onOpenChange,
  formRef,
  organizations,
  currentOrganizationId,
  canReadRoles,
  onReload,
  onSuccess,
}: InvitationFormModalProps) {
  const { message } = App.useApp();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);
  const [kind, setKind] = useState<number>(
    DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED,
  );

  useEffect(() => {
    if (!open) return;
    setKind(DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED);
    formRef.current?.resetFields();
    setRoles([]);
    // 打开即按默认目标组织（当前组织）加载初始角色。
    if (canReadRoles && currentOrganizationId) {
      fetchRolesForOrganization(currentOrganizationId, currentOrganizationId)
        .then(setRoles)
        .catch(() => setRoles([]));
    }
  }, [open, canReadRoles, currentOrganizationId, formRef]);

  const isTargeted =
    kind === DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED;

  return (
    <ModalForm<InvitationFormValues>
      title="新建扫码邀请"
      width={520}
      open={open}
      formRef={formRef}
      initialValues={{
        kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED,
        organizationId: currentOrganizationId,
        expiresInHours: INVITATION_DEFAULT_TTL_HOURS,
      }}
      modalProps={{
        destroyOnHidden: true,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        const selectedKind =
          values.kind ??
          DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED;
        const targeted =
          selectedKind ===
          DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED;

        const response = await adminServiceCreateDingTalkInvitation({
          kind: selectedKind,
          mobile: targeted ? (values.mobile?.trim() ?? '') : undefined,
          organizationId: values.organizationId ?? '',
          roleId: values.roleId || undefined,
          displayName: values.displayName?.trim() || undefined,
          expiresInHours: values.expiresInHours ?? INVITATION_DEFAULT_TTL_HOURS,
        });

        message.success(
          targeted
            ? '邀请已创建，员工用该手机号钉钉扫码后将自动激活'
            : '通用入职码已生成，新员工扫码后将进入审批队列',
        );
        onOpenChange(false);
        onReload();
        if (onSuccess && response.data) {
          onSuccess(response.data, response.invitationUrl);
        }
        return true;
      }}
    >
      <ProFormRadio.Group
        name="kind"
        label="邀请模式"
        options={[
          {
            label: '定向邀请码（单人免审）',
            value: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED,
          },
          {
            label: '通用入职码（多人审批）',
            value: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC,
          },
        ]}
        fieldProps={{
          onChange: (e) => {
            const nextKind = Number(e.target.value);
            setKind(nextKind);
            if (
              nextKind ===
              DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC
            ) {
              formRef.current?.setFieldValue('mobile', undefined);
            }
          },
        }}
      />

      <Text
        type="secondary"
        style={{ display: 'block', marginBottom: 16, fontSize: 12 }}
      >
        {isTargeted
          ? '为目标手机号预建邀请：员工钉钉扫码且企业通讯录手机号匹配后，自动启用账号、加入目标组织并授予初始角色，无需人工审批。'
          : '生成分公司专属通用入职码：多人可扫码申请加入目标组织，员工扫码后进入审批队列，由管理员审核后入职。'}
      </Text>

      {isTargeted && (
        <ProFormText
          name="mobile"
          label="手机号"
          placeholder="请输入员工钉钉绑定的手机号，例如 13800138000"
          fieldProps={{ maxLength: 13 }}
          rules={[
            { required: true, message: '请输入手机号' },
            {
              pattern: CHINA_MOBILE_PATTERN,
              message: '请输入有效的国内手机号',
            },
          ]}
        />
      )}

      <ProFormSearchableSelect
        name="organizationId"
        label="目标组织"
        placeholder="请选择要加入的组织"
        options={organizationSelectOptions(organizations)}
        rules={[{ required: true, message: '请选择目标组织' }]}
        fieldProps={{
          onChange: (organizationId: string) => {
            formRef.current?.setFieldValue('roleId', undefined);
            if (!canReadRoles) return;
            fetchRolesForOrganization(organizationId, currentOrganizationId)
              .then(setRoles)
              .catch(() => setRoles([]));
          },
        }}
      />

      <ProFormSearchableSelect
        name="roleId"
        label={isTargeted ? '初始角色' : '预设角色（可选）'}
        placeholder={
          isTargeted
            ? '请选择激活后授予的角色'
            : '请选择预设角色（可选，审批时可修改）'
        }
        options={roleSelectOptions(roles)}
        rules={
          isTargeted ? [{ required: true, message: '请选择初始角色' }] : []
        }
      />

      <ProFormText
        name="displayName"
        label="备注姓名（可选）"
        placeholder="例如：张三；仅供邀请列表识别，不影响账号身份"
        fieldProps={{ maxLength: 100 }}
      />

      <ProFormSearchableSelect
        name="expiresInHours"
        label="有效期"
        options={INVITATION_TTL_OPTIONS}
        allowClear={false}
        rules={[{ required: true, message: '请选择有效期' }]}
      />
    </ModalForm>
  );
}
