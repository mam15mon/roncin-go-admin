import { Col, Form, Input, Row, Select } from 'antd';
import React from 'react';
import { QuickCreateModal } from '@/components/ui/quick-create-modal';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';

type QuickAddPartnerModalProps = {
  open: boolean;
  onCancel: () => void;
  onSuccess: (newPartner: {
    id: string;
    name: string;
    code?: string;
  }) => void;
};

export default function QuickAddPartnerModal({
  open,
  onCancel,
  onSuccess,
}: QuickAddPartnerModalProps) {
  return (
    <QuickCreateModal<
      any,
      {
        id: string;
        name: string;
        code?: string;
      }
    >
      title="快捷新建往来单位"
      open={open}
      width={540}
      onCancel={onCancel}
      onSuccess={onSuccess}
      onSubmit={async (values) => {
        const roles = (values.roles as string[]).map((r) => ({
          type: r === 'CUSTOMER' ? 1 : 2,
        }));
        const res = await partnerServiceCreatePartner({
          legalName: values.legalName,
          code: values.code || undefined,
          unifiedSocialCreditCode: values.unifiedSocialCreditCode || undefined,
          roles,
        });
        if (res.data?.id) {
          return {
            id: res.data.id,
            name: res.data.legalName ?? values.legalName,
            code: res.data.code,
          };
        }
        return undefined;
      }}
    >
      <Row gutter={16}>
        <Col span={24}>
          <Form.Item
            name="legalName"
            label="单位全称"
            rules={[
              { required: true, whitespace: true, message: '请输入单位全称' },
              { max: 200, message: '不能超过 200 字符' },
            ]}
          >
            <Input placeholder="工商登记全称或客商名称" maxLength={200} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="code" label="客商代码（选填）">
            <Input
              placeholder="例如：COSCO、SITC（留空自动生成）"
              maxLength={50}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="unifiedSocialCreditCode"
            label="统一社会信用代码（选填）"
          >
            <Input placeholder="18 位税号" maxLength={50} />
          </Form.Item>
        </Col>
        <Col span={24}>
          <Form.Item
            name="roles"
            label="客商类型"
            rules={[{ required: true, message: '请选择至少一种类型' }]}
          >
            <Select
              mode="multiple"
              options={[
                { label: '客户 (委托单位/收发通)', value: 'CUSTOMER' },
                {
                  label: '供应商 (船东/车队/报关行/码头)',
                  value: 'SUPPLIER',
                },
              ]}
            />
          </Form.Item>
        </Col>
      </Row>
    </QuickCreateModal>
  );
}
