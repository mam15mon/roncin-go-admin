import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Form } from 'antd';
import { FormRow, PackageCountInput } from '@/components/ui';
import AirMasterIdentityFields from './components/air/AirMasterFields';
import {
  SeaCustomerField,
  SeaOrderReferenceField,
  SeaServiceTypeFields,
} from './components/sea/SeaBasicInfoSection';
import { SeaCargoDescriptionFields } from './components/sea/SeaCargoSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import type { TemplateProps, TemplateSection } from './types';

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
            <ProFormText
              name="bookingNo"
              label="订舱号"
              disabled={props.readonly}
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
      key: 'airDocuments',
      title: '航空运单与共同运输',
      content: <AirMasterIdentityFields props={props} />,
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
          <ProFormTextArea
            name="operationNotes"
            label="操作备注"
            disabled={props.readonly}
            fieldProps={{ maxLength: 1000 }}
          />
        </>
      ),
    },
    buildSeaPersonnelSection(props),
  ];
}
