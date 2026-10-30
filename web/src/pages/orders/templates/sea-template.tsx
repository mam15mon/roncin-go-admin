import { Form, Radio } from 'antd';
import { FormRow } from '@/components/ui';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  TemplateControlGroup,
  TemplateFieldGroup,
} from './components/FieldGroup';
import {
  extractPersonnelFromPartnerAssignments,
  getSeaBaseInfoFields,
  SeaCustomerField,
  SeaDangerousGoodsFields,
  TooltipInput,
} from './components/sea/SeaBasicInfoSection';
import {
  SeaCargoDescriptionFields,
  SeaConsignedMeasurementFields,
} from './components/sea/SeaCargoSection';
import { SeaCollapsibleNotesField } from './components/sea/SeaCollapsibleNotesField';
import {
  buildSeaDocumentSection,
  HouseBillIdentityFields,
  SeaBillContentFormFields,
  SeaCreateDocumentModeField,
} from './components/sea/SeaDocumentSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import {
  getSeaTransportFields,
  SeaContainerPlanFields,
  SeaScheduleDateFields,
} from './components/sea/SeaTransportSection';
import type { TemplateProps, TemplateSection } from './types';

export {
  buildSeaDocumentSection,
  extractPersonnelFromPartnerAssignments,
  SeaConsignedMeasurementFields,
  SeaContainerPlanFields,
  SeaCustomerField,
  SeaDangerousGoodsFields,
  SeaScheduleDateFields,
  TooltipInput,
};

/**
 * 五卡片信息架构的唯一构建器：新建与详情共用相同的区块 key、标题、顺序、
 * 字段归属和栅格布局；`isDetail` 只影响字段能力（订单编号时间仅新建可改、
 * 详情按服务端动作只读、提单模式详情只展示），不产生第二套分节组合。
 */
export function getSeaTemplateSections(
  props: TemplateProps,
): TemplateSection[] {
  const basic = getSeaBaseInfoFields(props);
  const transport = getSeaTransportFields(props);
  return [
    {
      key: 'businessCustomer',
      title: '业务与客户',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          {basic.customer}
          <FormRow cols={4}>
            {basic.orderIdentity}
            {basic.references}
          </FormRow>
        </div>
      ),
    },
    {
      key: 'bookingTransport',
      title: '订舱与运输',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          <TemplateFieldGroup title="订舱与主单">
            <FormRow cols={4}>
              {basic.booking}
              {basic.carrier}
              {transport.master}
            </FormRow>
            <FormRow cols={3}>
              {basic.agents}
              {basic.shippingAgent}
            </FormRow>
            <FormRow cols={2}>
              <SeaDocumentModeDisplayField props={props} />
              {transport.ownership}
            </FormRow>
            <SeaCollapsibleNotesField
              name="bookingNotes"
              label="订舱备注"
              placeholder="请输入订舱备注"
              disabled={props.readonly}
            />
          </TemplateFieldGroup>
          <TemplateFieldGroup title="航线与船期">
            <FormRow cols={4}>
              {transport.vessel}
              {transport.schedule}
            </FormRow>
            {transport.ports}
          </TemplateFieldGroup>
          <TemplateFieldGroup title="箱量与截关">
            {transport.containers}
            <FormRow cols={4}>{transport.cutoffs}</FormRow>
            <SeaCollapsibleNotesField
              name="allocationNotes"
              label="配舱备注"
              placeholder="请输入配舱备注"
              disabled={props.readonly}
            />
          </TemplateFieldGroup>
          <SeaCollapsibleNotesField
            name="operationNotes"
            label="操作备注"
            placeholder="请输入操作备注"
            disabled={props.readonly}
            sectionHeading
          />
        </div>
      ),
    },
    {
      key: 'cargoCommercial',
      title: '货物与商业',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          <div>
            {basic.categories}
            {basic.dangerous}
          </div>
          <SeaCargoDescriptionFields />
          <FormRow cols={4}>
            {basic.trade}
            {basic.contract}
            {basic.cargoValue}
            {basic.insurance}
          </FormRow>
          {basic.commercial}
          <SeaConsignedMeasurementFields disabled={props.readonly} />
        </div>
      ),
    },
    buildSeaDocumentSection(props),
    buildSeaPersonnelSection(props),
  ];
}

/**
 * 提单模式字段：新建时可选择 HOUSE/DIRECT；详情只读展示当前模式，
 * 模式变更动作保留在提单信息卡片页签工具栏（走既有预览与确认流程）。
 */
function SeaDocumentModeDisplayField({ props }: { props: TemplateProps }) {
  if (props.isDetail) {
    return (
      <TemplateControlGroup title="提单模式">
        <Form.Item name="seaDocumentStructure" style={{ marginBottom: 0 }}>
          <Radio.Group aria-label="提单模式" disabled>
            <Radio.Button
              value={SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE}
            >
              有货代分单（HOUSE）
            </Radio.Button>
            <Radio.Button
              value={SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT}
            >
              仅船公司主单（DIRECT）
            </Radio.Button>
          </Radio.Group>
        </Form.Item>
      </TemplateControlGroup>
    );
  }
  return <SeaCreateDocumentModeField disabled={props.readonly} />;
}

export { HouseBillIdentityFields, SeaBillContentFormFields };
