import type { FormInstance } from 'antd';
import { Col, Form, Input, InputNumber, Modal, Row, Select } from 'antd';
import React from 'react';

type SeaSharedContainerCreateModalProps = {
  open: boolean;
  form: FormInstance;
  submitting: boolean;
  containerSpecOptions: { label: string; value: string | number }[];
  onCancel: () => void;
  onFinish: (values: any) => void;
};

/** 新建跨订单共享物理箱 Modal */
export default function SeaSharedContainerCreateModal({
  open,
  form,
  submitting,
  containerSpecOptions,
  onCancel,
  onFinish,
}: SeaSharedContainerCreateModalProps) {
  return (
    <Modal
      title="新建跨订单共享物理箱"
      open={open}
      onCancel={onCancel}
      onOk={() => form.submit()}
      confirmLoading={submitting}
      destroyOnClose
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={onFinish}
        initialValues={{
          packageCount: 0,
          grossWeightKg: '0.000',
          volumeCbm: '0.000000',
        }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="containerNo"
              label="箱号 (Container No.)"
              rules={[
                { required: true, message: '请输入箱号' },
                {
                  pattern: /^[A-Z0-9]+$/i,
                  message: '箱号仅允许英文与数字',
                },
              ]}
            >
              <Input placeholder="例如 COSU1234567" maxLength={30} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="containerSpecId"
              label="集装箱规格"
              rules={[{ required: true, message: '请选择箱型规格' }]}
            >
              <Select placeholder="选择规格" options={containerSpecOptions} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="sealNo" label="铅封号 (Seal No.)">
              <Input placeholder="可选填铅封号" maxLength={50} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="packageCount"
              label="总件数 (PCS)"
              rules={[{ required: true, message: '请输入总件数' }]}
            >
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="grossWeightKg"
              label="总毛重 (KG)"
              rules={[{ required: true, message: '请输入总毛重' }]}
            >
              <Input placeholder="例如 20000.000" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="volumeCbm"
              label="总体积 (CBM)"
              rules={[{ required: true, message: '请输入总体积' }]}
            >
              <Input placeholder="例如 65.000000" />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item name="note" label="备注说明">
          <Input.TextArea rows={2} placeholder="拼箱注意事项或客户说明" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
