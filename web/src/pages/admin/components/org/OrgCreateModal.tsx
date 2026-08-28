import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { Alert, App } from 'antd';
import React, { useRef } from 'react';
import { adminServiceCreateOrganization } from '@/services/roncin/adminService';
import {
  getChildOrganizationKind,
  getOrganizationKindMeta,
  type CreateFormValues,
} from './types';

type OrgCreateModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  parentOrg: API.AdminOrganization | null;
  onSuccess: (createdId?: string) => Promise<void>;
};

export default function OrgCreateModal({
  open,
  onOpenChange,
  parentOrg,
  onSuccess,
}: OrgCreateModalProps) {
  const { message } = App.useApp();
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  const childKind = getChildOrganizationKind(parentOrg?.kind);
  const childKindMeta = getOrganizationKindMeta(childKind);

  return (
    <ModalForm<CreateFormValues>
      title={`新增${childKindMeta?.label ?? ''}（所属上级：${parentOrg?.name ?? ''}）`}
      open={open}
      formRef={formRef}
      modalProps={{
        destroyOnClose: true,
        width: 520,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (!parentOrg?.id || !childKind) return false;
        try {
          const response = await adminServiceCreateOrganization({
            code: values.code?.trim() ?? '',
            name: values.name?.trim() ?? '',
            parentId: parentOrg.id,
            kind: childKind,
            baseCurrency:
              childKind === 2
                ? values.baseCurrency?.trim().toUpperCase()
                : undefined,
          });
          message.success(`${childKindMeta?.label}已成功创建`);
          onOpenChange(false);
          await onSuccess(response.data?.id);
          return true;
        } catch {
          message.error('创建组织失败，请重试');
          return false;
        }
      }}
    >
      {parentOrg && childKindMeta && (
        <Alert
          showIcon
          type="info"
          title={`当前上级：${parentOrg.name}；本次创建：${childKindMeta.label}`}
          style={{ marginBottom: 16 }}
        />
      )}
      <ProFormText
        name="code"
        label="组织编码"
        placeholder="例如：SH_BRANCH 或 LOGISTICS_HQ"
        rules={[
          { required: true, message: '请输入组织编码' },
          {
            pattern: /^[A-Za-z0-9_-]+$/,
            message: '编码仅支持英文字母、数字、下划线及连字符',
          },
        ]}
      />
      <ProFormText
        name="name"
        label="组织名称"
        placeholder="例如：上海分公司 / 华东海运中心"
        rules={[{ required: true, message: '请输入组织名称' }]}
      />
      {childKind === 2 && (
        <ProFormText
          name="baseCurrency"
          label="本币"
          placeholder="例如 CNY、USD"
          fieldProps={{ maxLength: 3 }}
          rules={[
            { required: true, message: '请输入公司本币' },
            { pattern: /^[A-Za-z]{3}$/, message: '请输入 3 位币种代码' },
          ]}
        />
      )}
    </ModalForm>
  );
}
