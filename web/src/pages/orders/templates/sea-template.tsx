import { ProFormTextArea } from '@ant-design/pro-components';
import { Form, Typography } from 'antd';
import React from 'react';
import { FormRow } from '@/components/ui';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  buildSeaBaseInfoSection,
  extractPersonnelFromPartnerAssignments,
  getSeaBaseInfoFields,
  SeaCustomerField,
  SeaDangerousGoodsFields,
  SeaServiceTypeFields,
  TooltipInput,
} from './components/sea/SeaBasicInfoSection';
import {
  buildSeaCargoSection,
  SeaCargoMeasurementFields,
} from './components/sea/SeaCargoSection';
import {
  buildSeaDocumentSection,
  HouseBillIdentityFields,
  SeaBillContentFormFields,
  SeaCreateDocumentModeField,
  SeaDocumentSectionComponent,
} from './components/sea/SeaDocumentSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import {
  buildSeaTransportSection,
  getSeaTransportFields,
  SeaContainerPlanFields,
  SeaScheduleDateFields,
} from './components/sea/SeaTransportSection';
import type { TemplateProps, TemplateSection } from './types';

export {
  buildSeaCargoSection,
  buildSeaDocumentSection,
  extractPersonnelFromPartnerAssignments,
  SeaCargoMeasurementFields,
  SeaContainerPlanFields,
  SeaCustomerField,
  SeaDangerousGoodsFields,
  SeaScheduleDateFields,
  SeaServiceTypeFields,
  TooltipInput,
};

export function buildSeaCargoAndDocumentSection(
  props: TemplateProps,
): TemplateSection {
  const cargoSection = buildSeaCargoSection();
  return {
    key: 'cargoAndDocumentInfo',
    title: '货物与提单信息',
    content: (
      <>
        {cargoSection.content}
        <SeaDocumentSectionComponent
          disabled={props.readonly}
          isDetail={props.isDetail}
          onOrderDataChanged={props.onOrderDataChanged}
        />
      </>
    ),
  };
}

export function getSeaTemplateSections(
  props: TemplateProps,
): TemplateSection[] {
  if (!props.isDetail) return getSeaCreateTemplateSections(props);
  return [
    buildSeaBaseInfoSection(props),
    buildSeaTransportSection(props),
    buildSeaCargoAndDocumentSection(props),
    {
      key: 'remarks',
      title: '备注',
      content: <SeaRemarksFields />,
    },
    buildSeaPersonnelSection(props),
  ];
}

function SeaRemarksFields() {
  return (
    <FormRow cols={3}>
      <ProFormTextArea
        name="bookingNotes"
        label="订舱备注"
        placeholder="请输入订舱备注"
        fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
      />
      <ProFormTextArea
        name="allocationNotes"
        label="配舱备注"
        placeholder="请输入配舱备注"
        fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
      />
      <ProFormTextArea
        name="operationNotes"
        label="操作备注"
        placeholder="请输入操作备注"
        fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
      />
    </FormRow>
  );
}

/** 新建页只组合字段；草稿、提交与校验仍由 OrderFormTemplate 持有。 */
export function SeaCreateHouseBillFields({ disabled }: { disabled?: boolean }) {
  const form = Form.useFormInstance();
  const structure =
    Form.useWatch('seaDocumentStructure', form) ??
    form.getFieldValue('seaDocumentStructure');
  if (structure !== SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE) {
    return (
      <Typography.Text type="secondary">
        DIRECT 不签发 HBL；选择 HOUSE 后录入分单内容。
      </Typography.Text>
    );
  }
  return (
    <div style={{ width: '100%' }}>
      <HouseBillIdentityFields fieldKey="seaHouseBill" disabled={disabled} />
      <SeaBillContentFormFields
        namePathPrefix={['seaHouseBill', 'content']}
        disabled={disabled}
        showCargoMeasurements={false}
        createLayout
      />
    </div>
  );
}

function getSeaCreateTemplateSections(props: TemplateProps): TemplateSection[] {
  const basic = getSeaBaseInfoFields(props, true);
  const transport = getSeaTransportFields(props, true);
  return [
    {
      key: 'basicInfo',
      title: '业务归属',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          {basic.customer}
          {basic.orderIdentity}
          {basic.services}
          {basic.categories}
          {basic.dangerous}
          <FormRow cols={4}>{basic.references}</FormRow>
        </div>
      ),
    },
    {
      key: 'bookingInfo',
      title: '订舱与主单识别',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          <FormRow cols={6}>
            <div style={{ gridColumn: 'span 2' }}>{basic.carrier}</div>
            {transport.master}
            {basic.booking}
          </FormRow>
          <FormRow cols={6}>
            {basic.agents}
            <div style={{ gridColumn: 'span 2' }}>{basic.shippingAgent}</div>
          </FormRow>
          <FormRow cols={4}>
            {transport.vessel}
            <div style={{ gridColumn: 'span 2' }}>
              <SeaCreateDocumentModeField disabled={props.readonly} />
            </div>
          </FormRow>
        </div>
      ),
    },
    {
      key: 'transportInfo',
      title: '航线与船期',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          {transport.ports}
          {transport.schedule}
          {transport.containers}
          <FormRow cols={4}>{transport.ownership}</FormRow>
          {transport.cutoffs}
        </div>
      ),
    },
    {
      key: 'masterBillContent',
      title: 'MBL 主单内容',
      content: (
        <div style={{ width: '100%' }}>
          <SeaBillContentFormFields
            namePathPrefix={['seaMasterBillContent']}
            disabled={props.readonly}
            createLayout
          />
          <SeaCargoMeasurementFields />
        </div>
      ),
    },
    {
      key: 'houseBillContent',
      title: 'HBL 分单内容（HOUSE）',
      content: <SeaCreateHouseBillFields disabled={props.readonly} />,
    },
    {
      key: 'supplementaryInfo',
      title: '补充与内部信息',
      content: (
        <div style={{ display: 'grid', gap: 12, width: '100%' }}>
          <FormRow cols={4}>{basic.trade}</FormRow>
          {basic.commercial}
          {buildSeaCargoSection().content}
          <SeaRemarksFields />
          {buildSeaPersonnelSection(props).content}
        </div>
      ),
    },
  ];
}
