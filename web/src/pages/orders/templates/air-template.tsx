import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDigit,
  ProFormText,
} from '@ant-design/pro-components';
import { Form } from 'antd';
import { FormCell, FormRow, PackageCountInput } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import {
  AirBusinessFields,
  AirCargoCommercialFields,
} from './components/air/AirCommercialFields';
import { AirDocumentSection } from './components/air/AirDocumentSection';
import { AirMasterTransportSection } from './components/air/AirMasterFields';
import {
  SeaCustomerField,
  SeaDangerousGoodsFields,
  SeaOrderReferenceField,
} from './components/sea/SeaBasicInfoSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import type { TemplateProps, TemplateSection } from './types';

/**
 * 海空订单统一五卡与六列紧凑栅格；航空重量、运输维护及单证动作由
 * 详情扩展贡献到对应业务区域，新建与详情保持同一字段归属。
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
          <FormRow cols={6} style={{ rowGap: 4 }}>
            <FormCell span={2}>
              <SeaCustomerField
                searchCustomers={props.searchCustomers}
                readonly={props.readonly}
                setCustomerCode={props.setCustomerCode}
                personnelOptions={props.personnelOptions}
              />
            </FormCell>
            <ProFormText name="customerCode" label="委托单位代码" disabled />
            <ProFormText
              name="orderNo"
              label="订单编号"
              disabled
              placeholder="保存后自动生成"
            />
            <SeaOrderReferenceField
              name="customerReferenceNo"
              isDetail={props.isDetail}
            />
            <SeaOrderReferenceField
              name="internalReferenceNo"
              isDetail={props.isDetail}
            />
            {!props.isDetail && (
              <ProFormDatePicker name="orderDate" label="订单编号时间" />
            )}
          </FormRow>
          <AirBusinessFields props={props} />
        </>
      ),
    },
    {
      key: 'bookingTransport',
      title:
        props.businessType === OrderBusinessType.BUSINESS_TYPE_AI
          ? '承运与运输'
          : '订舱与运输',
      content: <AirMasterTransportSection props={props} />,
    },
    {
      key: 'cargoCommercial',
      title: '货物与商业',
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
          <FormRow cols={6} style={{ rowGap: 4 }}>
            <FormCell span={2}>
              <Form.Item label="委托件数">
                <PackageCountInput
                  countName="totalPackages"
                  unitName="totalPackageUnit"
                  disabled={props.readonly}
                />
              </Form.Item>
            </FormCell>
            <FormCell span={2}>
              <ProFormDigit
                name="totalGrossWeightKg"
                label="委托毛重（KG）"
                min={0}
                disabled={props.readonly}
                fieldProps={{ precision: 3 }}
              />
            </FormCell>
            <FormCell span={2}>
              <ProFormDigit
                name="totalVolumeCbm"
                label="委托体积（m³）"
                min={0}
                disabled={props.readonly}
                fieldProps={{ precision: 3 }}
              />
            </FormCell>
          </FormRow>
          {props.airCargoWorkspace}
          <AirCargoCommercialFields props={props} />
        </>
      ),
    },
    {
      key: 'airDocuments',
      title: '单证信息',
      content: <AirDocumentSection props={props} />,
    },
    buildSeaPersonnelSection(props),
  ];
}
