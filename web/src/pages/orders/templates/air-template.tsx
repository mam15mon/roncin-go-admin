import { ProFormCheckbox, ProFormDigit } from '@ant-design/pro-components';
import { Form } from 'antd';
import { FormCell, FormRow, PackageCountInput } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import {
  AirCargoDescriptionFields,
  getAirCommercialFields,
} from './components/air/AirCommercialFields';
import { AirDocumentSection } from './components/air/AirDocumentSection';
import { AirMasterTransportSection } from './components/air/AirMasterFields';
import { TemplateControlGroup } from './components/FieldGroup';
import {
  getSeaBaseInfoFields,
  SeaDangerousGoodsFields,
} from './components/sea/SeaBasicInfoSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import type { TemplateProps, TemplateSection } from './types';

/** 与海运录入页同一行距：字段自带下边距，行间只再留 4px。 */
const DENSE_ROW = { rowGap: 4 } as const;

/**
 * 海空订单统一五卡与六列紧凑栅格，卡内分组、字段顺序与海运对齐：
 * 业务卡只放客户与单号，经营条款随货物卡首行，代理随主单与订舱，
 * 四类备注集中在运输卡「备注」小节。航空重量、运输维护及单证动作由
 * 详情扩展贡献到对应业务区域，新建与详情保持同一字段归属。
 */
export function getAirTemplateSections(
  props: TemplateProps,
): TemplateSection[] {
  // 客户、单号与参考号片段与海运共用，避免两处维护同一字段。
  const basic = getSeaBaseInfoFields(props);
  const commercial = getAirCommercialFields(props);
  return [
    {
      key: 'businessCustomer',
      title: '业务与客户',
      content: (
        <FormRow cols={6} style={DENSE_ROW}>
          {basic.customer}
          {basic.orderIdentity}
          {basic.references}
          {basic.orderDate}
        </FormRow>
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
        // 与海运同序：品类与条款同行，描述对半，件重尺与时间同行，金额与工厂同行。
        <div style={{ display: 'grid', gap: 4, width: '100%' }}>
          <FormRow cols={6} style={DENSE_ROW}>
            <FormCell span={2}>
              <TemplateControlGroup title="货物品类">
                <ProFormCheckbox.Group
                  name="cargoCategoryIds"
                  fieldProps={{ 'aria-label': '货物品类' }}
                  options={props.cargoCategoryOptions}
                  disabled={props.readonly}
                />
              </TemplateControlGroup>
            </FormCell>
            {commercial.trade}
            {commercial.payment}
            {commercial.contract}
          </FormRow>
          <SeaDangerousGoodsFields
            options={props.cargoCategoryOptions}
            disabled={props.readonly}
            showPackingGroup
          />
          <AirCargoDescriptionFields disabled={props.readonly} />
          <FormRow cols={6} style={DENSE_ROW}>
            <FormCell span={2}>
              <Form.Item label="委托件数">
                <PackageCountInput
                  countName="totalPackages"
                  unitName="totalPackageUnit"
                  disabled={props.readonly}
                />
              </Form.Item>
            </FormCell>
            <ProFormDigit
              name="totalGrossWeightKg"
              label="委托毛重（KG）"
              min={0}
              disabled={props.readonly}
              fieldProps={{ precision: 3, style: { width: '100%' } }}
            />
            <ProFormDigit
              name="totalVolumeCbm"
              label="委托体积（CBM）"
              min={0}
              disabled={props.readonly}
              fieldProps={{ precision: 3, style: { width: '100%' } }}
            />
            {commercial.times}
            {commercial.cargoValue}
            {commercial.insurance}
            {commercial.factory}
          </FormRow>
          {props.airCargoWorkspace}
        </div>
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
