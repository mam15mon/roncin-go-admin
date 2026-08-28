import {
  ModalForm,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { Col, type FormInstance } from 'antd';
import React from 'react';
import { DOC_TYPES, docTypeMap } from './numberRulesConstants';

interface NumberRuleEditModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingItem?: API.NumberRule;
  data: API.NumberRule[];
  form: FormInstance;
  onFinish: (values: any) => Promise<boolean>;
}

export default function NumberRuleEditModal({
  open,
  onOpenChange,
  editingItem,
  data,
  form,
  onFinish,
}: NumberRuleEditModalProps) {
  return (
    <ModalForm
      title={
        editingItem
          ? `编辑【${docTypeMap.get(editingItem.documentType as any)?.label || '单据'}】规则`
          : '新建单据编号规则'
      }
      open={open}
      form={form}
      onOpenChange={onOpenChange}
      onFinish={onFinish}
      modalProps={{
        destroyOnClose: true,
        maskClosable: false,
        width: 520,
      }}
      layout="horizontal"
      labelAlign="right"
      labelCol={{ flex: '96px' }}
      wrapperCol={{ flex: 'auto' }}
      grid
    >
      <Col span={24}>
        <ProFormSelect
          name="documentType"
          label="单据类型"
          options={DOC_TYPES.map((t) => {
            const alreadyExists =
              !editingItem &&
              data.some(
                (r) =>
                  docTypeMap.get(r.documentType as any)?.numValue ===
                  t.numValue,
              );
            return {
              label: alreadyExists ? `${t.label} (已配置)` : t.label,
              value: t.numValue,
              disabled: alreadyExists,
            };
          })}
          placeholder="请选择单据类型"
          rules={[{ required: true, message: '请选择单据类型' }]}
          disabled={Boolean(editingItem)}
        />
      </Col>
      <Col span={24}>
        <ProFormText
          name="prefix"
          label="前缀代码"
          placeholder="可选，例如：OR"
          rules={[
            {
              pattern: /^[A-Za-z0-9_-]*$/,
              message: '仅支持字母、数字与下划线',
            },
          ]}
        />
      </Col>
      <Col span={24}>
        <ProFormSelect
          name="dateFormat"
          label="日期格式"
          options={[
            { label: '年月日 (yyyyMMdd 示例: 20260823)', value: 1 },
            { label: '年月 (yyyyMM 示例: 202608)', value: 2 },
            { label: '年 (yyyy 示例: 2026)', value: 3 },
            { label: '无日期 (仅前缀+流水号)', value: 4 },
          ]}
          rules={[{ required: true, message: '请选择日期格式' }]}
        />
      </Col>
      <Col span={24}>
        <ProFormDigit
          name="sequenceLength"
          label="流水号位数"
          min={1}
          max={12}
          placeholder="例如：4 (生成 0001)"
          rules={[{ required: true, message: '请输入流水号位数 (1-12)' }]}
        />
      </Col>
      <Col span={24}>
        <ProFormSelect
          name="resetPolicy"
          label="重置周期"
          options={[
            { label: '每日重置 (推荐配合年月日)', value: 1 },
            { label: '每月重置 (推荐配合年月)', value: 2 },
            { label: '每年重置 (推荐配合年)', value: 3 },
            { label: '永不重置 (递增累加)', value: 4 },
          ]}
          rules={[{ required: true, message: '请选择重置周期' }]}
        />
      </Col>
      <Col span={24}>
        <ProFormSwitch
          name="enabled"
          label="启用状态"
          checkedChildren="启用"
          unCheckedChildren="停用"
        />
      </Col>
    </ModalForm>
  );
}
