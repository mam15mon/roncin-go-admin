import {
  ProFormDigit,
  ProFormList,
  ProFormSelect,
  ProFormText,
} from '@ant-design/pro-components';
import { Col } from 'antd';
import React from 'react';

type SelectOption = { label: string; value: string | number };

export function OrderShippingDocumentFields() {
  return (
    <Col span={24}>
      <ProFormList
        name="shippingDocuments"
        label="主单号 / 分单号"
        creatorButtonProps={{ creatorButtonText: '新增主分单' }}
        creatorRecord={{ masterNo: '', houseNo: '' }}
        copyIconProps={false}
        deleteIconProps={{ tooltipText: '删除该主分单' }}
        itemContainerRender={(doms) => <div style={{ width: '100%' }}>{doms}</div>}
      >
        <ProFormText name="id" hidden />
        <ProFormText
          name="masterNo"
          label="主单号"
          placeholder="请输入 MBL / MAWB"
          rules={[
            { required: true, message: '请输入主单号' },
            { max: 64, message: '主单号不能超过 64 个字符' },
          ]}
          width="md"
        />
        <ProFormText
          name="houseNo"
          label="分单号"
          placeholder="请输入 HBL / HAWB"
          rules={[
            { required: true, message: '请输入分单号' },
            { max: 64, message: '分单号不能超过 64 个字符' },
          ]}
          width="md"
        />
      </ProFormList>
    </Col>
  );
}

export function OrderContainerRequestFields({
  options,
}: {
  options: SelectOption[];
}) {
  return (
    <Col span={24}>
      <ProFormList
        name="containerRequests"
        label="箱型箱量"
        creatorButtonProps={{ creatorButtonText: '新增箱型箱量' }}
        creatorRecord={{ containerSpecId: '', quantity: 1 }}
        copyIconProps={false}
        deleteIconProps={{ tooltipText: '删除该箱型箱量' }}
        itemContainerRender={(doms) => <div style={{ width: '100%' }}>{doms}</div>}
      >
        <ProFormText name="id" hidden />
        <ProFormSelect
          name="containerSpecId"
          label="箱型"
          options={options}
          placeholder="请选择箱型"
          rules={[{ required: true, message: '请选择箱型' }]}
          fieldProps={{ showSearch: true, optionFilterProp: 'label' }}
          width="md"
        />
        <ProFormDigit
          name="quantity"
          label="箱量"
          min={1}
          max={999}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入箱量' }]}
          width="sm"
        />
      </ProFormList>
    </Col>
  );
}
