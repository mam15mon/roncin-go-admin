import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { App, Typography } from 'antd';
import React, { useEffect, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
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
  mobile: string;
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
}

/**
 * 新建扫码邀请模态：预存管理员决策字段（手机号 + 目标组织 + 初始角色 +
 * 备注姓名 + 有效期）；账号身份以钉钉扫码返回为准，不在此登记。
 */
export default function InvitationFormModal({
  open,
  onOpenChange,
  formRef,
  organizations,
  currentOrganizationId,
  canReadRoles,
  onReload,
}: InvitationFormModalProps) {
  const { message } = App.useApp();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);

  useEffect(() => {
    if (!open) return;
    formRef.current?.resetFields();
    setRoles([]);
    // 打开即按默认目标组织（当前组织）加载初始角色。
    if (canReadRoles && currentOrganizationId) {
      fetchRolesForOrganization(currentOrganizationId, currentOrganizationId)
        .then(setRoles)
        .catch(() => setRoles([]));
    }
  }, [open, canReadRoles, currentOrganizationId, formRef]);

  return (
    <ModalForm<InvitationFormValues>
      title="新建扫码邀请"
      width={520}
      open={open}
      formRef={formRef}
      initialValues={{
        organizationId: currentOrganizationId,
        expiresInHours: INVITATION_DEFAULT_TTL_HOURS,
      }}
      modalProps={{
        destroyOnClose: true,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        await adminServiceCreateDingTalkInvitation({
          mobile: values.mobile?.trim() ?? '',
          organizationId: values.organizationId ?? '',
          roleId: values.roleId ?? '',
          displayName: values.displayName?.trim() || undefined,
          expiresInHours: values.expiresInHours ?? INVITATION_DEFAULT_TTL_HOURS,
        });
        message.success('邀请已创建，员工用该手机号钉钉扫码后将自动激活');
        onOpenChange(false);
        onReload();
        return true;
      }}
    >
      <Text
        type="secondary"
        style={{ display: 'block', marginBottom: 16, fontSize: 12 }}
      >
        为目标手机号预建邀请：员工钉钉扫码且企业通讯录手机号匹配后，自动启用账号、
        加入目标组织并授予初始角色，无需人工审批。
      </Text>
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
        label="初始角色"
        placeholder="请选择激活后授予的角色"
        options={roleSelectOptions(roles)}
        rules={[{ required: true, message: '请选择初始角色' }]}
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
