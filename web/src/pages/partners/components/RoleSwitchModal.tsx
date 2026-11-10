import { App, Button, Checkbox, Form, Modal, Typography } from 'antd';
import React, { useEffect } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { PartnerRoleType } from '@/enums.generated';
import { partnerServiceUpdatePartner } from '@/services/roncin/partnerService';
import { isWriteResponse } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';

const { Text } = Typography;

const ROLE_OPTIONS = [
  { label: '客户', value: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER },
  { label: '供应商', value: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER },
  { label: '国外代理', value: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT },
];

interface RoleSwitchModalProps {
  open: boolean;
  partner: API.Partner | null;
  onClose: () => void;
  onSuccess: () => void;
}

export default function RoleSwitchModal({
  open,
  partner,
  onClose,
  onSuccess,
}: RoleSwitchModalProps) {
  const { markDirty, resetDirty } = useUnsavedChangesGuard({
    enabled: open,
    identity: partner?.id,
  });
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = React.useState(false);

  useEffect(() => {
    if (open && partner) {
      const activeRoleTypes = (partner.roles ?? [])
        .filter((r) => r.enabled)
        .map((r) => r.type as number);
      form.setFieldsValue({
        roleTypes: activeRoleTypes.length > 0 ? activeRoleTypes : [1],
      });
    }
  }, [open, partner, form]);

  const handleSubmit = async (): Promise<false | undefined> => {
    if (!partner?.id) return;
    try {
      const values = await form.validateFields();
      const selectedTypes: number[] = values.roleTypes || [];
      if (selectedTypes.length === 0) {
        message.warning('请至少选择一个业务身份');
        return;
      }

      setSubmitting(true);

      // 构建角色输入
      const roleInputs: API.PartnerRoleInput[] = selectedTypes.map((type) => ({
        type,
        enabled: true,
      }));

      const writeResponse1 = await partnerServiceUpdatePartner(
        { id: partner.id },
        {
          id: partner.id,
          legalName: partner.legalName ?? '',
          code: partner.code ?? '',
          unifiedSocialCreditCode: partner.unifiedSocialCreditCode,
          registeredAddress: partner.registeredAddress,
          enabled: partner.enabled ?? true,
          roles: roleInputs,
        },
      );
      if (!isWriteResponse(writeResponse1, true)) return false;

      resetDirty();
      message.success('业务身份已更新');
      onSuccess();
      onClose();
    } catch (e) {
      message.error(getErrorMessage(e, '业务身份更新失败'));
    } finally {
      setSubmitting(false);
    }

    return undefined;
  };

  return (
    <Modal
      title="业务身份"
      width={400}
      open={open}
      onCancel={onClose}
      destroyOnHidden
      footer={[
        <Button key="cancel" onClick={onClose} disabled={submitting}>
          取消
        </Button>,
        <Button
          key="submit"
          type="primary"
          onClick={handleSubmit}
          loading={submitting}
        >
          保存
        </Button>,
      ]}
    >
      <Typography.Paragraph style={{ marginTop: 16, marginBottom: 16 }}>
        <Text strong>{partner?.legalName}</Text>
      </Typography.Paragraph>
      <Form form={form} layout="vertical" onValuesChange={markDirty}>
        <fieldset style={{ border: 0, padding: 0, margin: 0 }}>
          <legend style={{ fontSize: 13, marginBottom: 8 }}>
            业务身份（可多选）
          </legend>
          <Form.Item
            name="roleTypes"
            rules={[{ required: true, message: '请至少选择一个业务身份' }]}
            style={{ marginBottom: 0 }}
          >
            <Checkbox.Group options={ROLE_OPTIONS} />
          </Form.Item>
        </fieldset>
      </Form>
    </Modal>
  );
}
