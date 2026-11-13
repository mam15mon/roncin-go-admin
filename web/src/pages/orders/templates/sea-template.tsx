import { Form, Tag } from 'antd';
import { FormCell, FormRow } from '@/components/ui';
import { SeaDocumentStructure } from '@/enums.generated';
import { TemplateFieldGroup } from './components/FieldGroup';
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
 * 字段自带 12px 下边距（含校验提示位），行间再叠 12px 会显得松散；
 * 海运录入页把行距收到 4px，整体每行约 74px。
 */
const DENSE_ROW = { rowGap: 4 } as const;

/**
 * 五卡片信息架构的唯一构建器：新建与详情共用相同的区块 key、标题、顺序、
 * 字段归属和 6 列底栅格（长值跨 2 列、短值 1 列）；`isDetail` 只影响字段能力（订单编号时间仅新建可改、
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
      // 6 列底栅格：订单编号时间仅新建存在，置于行尾，详情其余字段位置不变。
      content: (
        <FormRow cols={6} style={DENSE_ROW}>
          {basic.customer}
          {basic.orderIdentity}
          {basic.references}
          {basic.shipmentOptions}
          {basic.orderDate}
        </FormRow>
      ),
    },
    {
      key: 'bookingTransport',
      title: '订舱与运输',
      // 各小节共用 6 列底栅格，列边界上下对齐；长值跨 2 列，编号/日期占 1 列。
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          <TemplateFieldGroup title="订舱与主单">
            <FormRow cols={6} style={DENSE_ROW}>
              <FormCell span={2}>
                <SeaDocumentModeDisplayField props={props} />
              </FormCell>
              {basic.booking}
              <FormCell span={2}>{basic.carrier}</FormCell>
              {basic.shippingAgent}
              {basic.agents}
              {/* 主单字段置于行尾：核对提示、新增分单入口等整行内容紧跟其后，不打断代理行。 */}
              {transport.master}
            </FormRow>
          </TemplateFieldGroup>
          <TemplateFieldGroup
            title="共同航程"
            extra={props.sharedVoyageMaintenance}
          >
            <FormRow cols={6} style={DENSE_ROW}>
              {transport.vessel}
              {transport.schedule}
              {transport.ports}
              <FormCell span={2}>{transport.destination}</FormCell>
            </FormRow>
          </TemplateFieldGroup>
          <TemplateFieldGroup title="箱量与截关">
            {transport.containers}
            <FormRow cols={6} style={DENSE_ROW}>
              {transport.ownership}
              {transport.cutoffs}
            </FormRow>
          </TemplateFieldGroup>
          <TemplateFieldGroup title="备注">
            <FormRow cols={3} style={DENSE_ROW}>
              <SeaCollapsibleNotesField
                name="bookingNotes"
                label="订舱备注"
                placeholder="请输入订舱备注"
                disabled={props.readonly}
              />
              <SeaCollapsibleNotesField
                name="allocationNotes"
                label="配舱备注"
                placeholder="请输入配舱备注"
                disabled={props.readonly}
              />
              <SeaCollapsibleNotesField
                name="operationNotes"
                label="操作备注"
                placeholder="请输入操作备注"
                disabled={props.readonly}
              />
            </FormRow>
          </TemplateFieldGroup>
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
const SEA_DOCUMENT_MODE_OUTPUT_ID = 'seaDocumentStructureDisplay';

function SeaDocumentModeDisplayField({ props }: { props: TemplateProps }) {
  if (props.isDetail) {
    // 使用标准 Form.Item 标签，与同行输入框的标签行与控件行严格对齐。
    return (
      <Form.Item
        name="seaDocumentStructure"
        label="提单模式"
        htmlFor={SEA_DOCUMENT_MODE_OUTPUT_ID}
      >
        <SeaDocumentModeValue />
      </Form.Item>
    );
  }
  return <SeaCreateDocumentModeField disabled={props.readonly} />;
}

function SeaDocumentModeValue({ value }: { value?: SeaDocumentStructure }) {
  const label =
    value === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
      ? '有货代分单（HOUSE）'
      : value === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
        ? '仅船公司主单（DIRECT）'
        : '未设置';
  return (
    // 与同行输入框等高对齐的只读展示；模式变更入口在提单信息卡片。
    <output
      id={SEA_DOCUMENT_MODE_OUTPUT_ID}
      aria-label="当前提单模式"
      style={{
        display: 'flex',
        alignItems: 'center',
        height: 32,
        padding: 0,
      }}
    >
      <Tag color="blue" style={{ margin: 0, paddingInline: 10 }}>
        {label}
      </Tag>
    </output>
  );
}

export { HouseBillIdentityFields, SeaBillContentFormFields };
