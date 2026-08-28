import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { App, Typography } from 'antd';
import React, { useRef } from 'react';
import { adminServiceUpdateOrganization } from '@/services/roncin/adminService';
import type { EditFormValues } from './types';

const { Text } = Typography;

type OrgEditModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingOrg: API.AdminOrganization | null;
  onSuccess: (updatedId?: string) => Promise<void>;
};

export default function OrgEditModal({
  open,
  onOpenChange,
  editingOrg,
  onSuccess,
}: OrgEditModalProps) {
  const { message } = App.useApp();
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  return (
    <ModalForm<EditFormValues>
      title={`编辑组织：${editingOrg?.name ?? ''}`}
      open={open}
      formRef={formRef}
      initialValues={{
        name: editingOrg?.name,
        enabled: editingOrg?.enabled ?? true,
        baseCurrency: editingOrg?.baseCurrency,
      }}
      modalProps={{
        destroyOnClose: true,
        width: 520,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (!editingOrg?.id) return false;
        try {
          await adminServiceUpdateOrganization(
            { id: editingOrg.id },
            {
              id: editingOrg.id,
              name: values.name?.trim() ?? '',
              enabled: values.enabled ?? true,
              baseCurrency:
                editingOrg.kind === 1 || editingOrg.kind === 2
                  ? values.baseCurrency?.trim().toUpperCase()
                  : undefined,
            },
          );
          message.success('组织已成功更新');
          onOpenChange(false);
          await onSuccess(editingOrg.id);
          return true;
        } catch {
          message.error('更新组织失败，请重试');
          return false;
        }
      }}
    >
      <div style={{ marginBottom: 16 }}>
        <Text
          type="secondary"
          style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
        >
          组织编码（不可变更）
        </Text>
        <Text strong style={{ fontFamily: 'monospace', fontSize: 14 }}>
          {editingOrg?.code}
        </Text>
      </div>
      <ProFormText
        name="name"
        label="组织名称"
        rules={[{ required: true, message: '请输入组织名称' }]}
      />
      <ProFormSwitch
        name="enabled"
        label="启用状态"
        extra="停用后该组织及其关联成员将无法进行业务操作"
      />
      {(editingOrg?.kind === 1 || editingOrg?.kind === 2) && (
        <ProFormText
          name="baseCurrency"
          label="本币"
          placeholder="例如 CNY、USD"
          fieldProps={{ maxLength: 3 }}
          rules={[
            { required: true, message: '请输入组织本币' },
            { pattern: /^[A-Za-z]{3}$/, message: '请输入 3 位币种代码' },
          ]}
          extra="修改本币不会改变已保存费用的汇率快照"
        />
      )}
    </ModalForm>
  );
}
