import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Form } from 'antd';
import {
  FormRow,
  PackageCountInput,
  ProFormSearchableSelect,
} from '@/components/ui';
import { AirMasterTransportSection } from './components/air/AirMasterFields';
import {
  SeaCustomerField,
  SeaDangerousGoodsFields,
  SeaOrderReferenceField,
  SeaServiceTypeFields,
} from './components/sea/SeaBasicInfoSection';
import { SeaCargoDescriptionFields } from './components/sea/SeaCargoSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import type { TemplateProps, TemplateSection } from './types';

/**
 * 五卡片信息架构与海运出口对齐：业务与客户 / 订舱与运输 / 本票货物 /
 * 分单信息 / 内部人员；新建与详情共用同一分节组合，`isDetail` 只影响
 * 字段能力（编号时间仅新建可改、主单资料详情只读展示摘要、已签发分单
 * 修改原因仅详情展示）。
 */
export function getAirTemplateSections(
  props: TemplateProps,
): TemplateSection[] {
  return [
    {
      key: 'businessCustomer',
      title: '业务与客户',
      content: (
        <>
          <FormRow cols={4}>
            <ProFormText
              name="orderNo"
              label="订单编号"
              disabled
              placeholder="保存后自动生成"
            />
            {!props.isDetail && (
              <ProFormDatePicker name="orderDate" label="订单编号时间" />
            )}
            <SeaOrderReferenceField
              name="customerReferenceNo"
              isDetail={props.isDetail}
            />
            <SeaOrderReferenceField
              name="internalReferenceNo"
              isDetail={props.isDetail}
            />
          </FormRow>
          <FormRow cols={2}>
            <SeaCustomerField
              searchCustomers={props.searchCustomers}
              readonly={props.readonly}
              setCustomerCode={props.setCustomerCode}
              personnelOptions={props.personnelOptions}
            />
            <ProFormText name="customerCode" label="委托单位代码" disabled />
          </FormRow>
          <SeaServiceTypeFields options={props.serviceTypeOptions} />
        </>
      ),
    },
    {
      key: 'bookingTransport',
      title: '订舱与运输',
      content: <AirMasterTransportSection props={props} />,
    },
    {
      key: 'cargoCommercial',
      title: '本票货物',
      content: (
        <>
          <ProFormCheckbox.Group
            name="cargoCategoryIds"
            label="货物品类"
            options={props.cargoCategoryOptions}
          />
          <SeaDangerousGoodsFields
            options={props.cargoCategoryOptions}
            disabled={props.readonly}
            showPackingGroup
          />
          <SeaCargoDescriptionFields />
          <FormRow cols={3}>
            <Form.Item label="本票件数">
              <PackageCountInput
                countName="totalPackages"
                unitName="totalPackageUnit"
                disabled={props.readonly}
              />
            </Form.Item>
            <ProFormDigit
              name="totalGrossWeightKg"
              label="本票毛重（KG）"
              min={0}
              disabled={props.readonly}
              fieldProps={{ precision: 3 }}
            />
            <ProFormDigit
              name="totalVolumeCbm"
              label="本票体积（m³）"
              min={0}
              disabled={props.readonly}
              fieldProps={{ precision: 3 }}
            />
          </FormRow>
        </>
      ),
    },
    {
      key: 'airDocuments',
      title: '分单信息',
      content: <AirHouseDocumentFields props={props} />,
    },
    buildSeaPersonnelSection(props),
  ];
}

/** 分单信息卡：本票分运单号 / 状态 / 单证资料，详情已签发时展示修改原因。 */
function AirHouseDocumentFields({ props }: { props: TemplateProps }) {
  const form = Form.useFormInstance();
  const air = Form.useWatch('airExport', { form, preserve: true }) as
    | API.AirOrderInfo
    | undefined;
  return (
    <>
      <FormRow cols={4}>
        <ProFormText
          name="airHouseNo"
          label="本票分运单号（HAWB）"
          rules={[
            { required: true, whitespace: true, message: '请输入本票分运单号' },
          ]}
          disabled={props.readonly}
          fieldProps={{ maxLength: 64 }}
          tooltip="同一主单内不允许重号，不同主单可重复；号码会统一大写。"
        />
        <ProFormSearchableSelect
          name="airHouseStatus"
          label="本票分单状态"
          initialValue="DRAFT"
          options={[
            { label: '资料待完善', value: 'DRAFT' },
            { label: '已确认', value: 'CONFIRMED' },
            { label: '已签发', value: 'RELEASED' },
          ]}
          disabled={props.readonly}
        />
        <ProFormTextArea
          name="airHouseContent"
          label="本票单证资料"
          disabled={props.readonly}
          fieldProps={{ maxLength: 4000, autoSize: { minRows: 1, maxRows: 4 } }}
        />
      </FormRow>
      {props.isDetail && air?.houseStatus === 'RELEASED' && (
        <ProFormTextArea
          name="airHouseChangeReason"
          label="已签发分单修改原因"
          tooltip="修改已签发分单号、内容或状态时必填；仅保存本票客户、货量等字段时可空。"
          fieldProps={{ maxLength: 500 }}
          disabled={props.readonly}
        />
      )}
    </>
  );
}
