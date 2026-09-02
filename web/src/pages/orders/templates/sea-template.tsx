import { ProFormTextArea } from '@ant-design/pro-components';
import React from 'react';
import {
  buildSeaBaseInfoSection,
  SeaServiceTypeFields,
  TooltipInput,
} from './components/sea/SeaBasicInfoSection';
import {
  buildSeaCargoSection,
  SeaCargoMeasurementFields,
} from './components/sea/SeaCargoSection';
import { buildSeaDocumentSection } from './components/sea/SeaDocumentSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import {
  buildSeaTransportSection,
  SeaContainerPlanFields,
  SeaScheduleDateFields,
} from './components/sea/SeaTransportSection';
import type { TemplateProps, TemplateSection } from './types';

export {
  SeaContainerPlanFields,
  SeaScheduleDateFields,
  SeaCargoMeasurementFields,
  SeaServiceTypeFields,
  TooltipInput,
};

export function getSeaTemplateSections(
  props: TemplateProps,
): TemplateSection[] {
  return [
    buildSeaBaseInfoSection(props),
    buildSeaTransportSection(props),
    buildSeaDocumentSection(props),
    buildSeaCargoSection(),
    {
      key: 'remarks',
      title: '备注',
      content: (
        <>
          <ProFormTextArea
            colProps={{ xs: 24, lg: 8 }}
            name="bookingNotes"
            label="订舱备注"
            placeholder="请输入订舱备注"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
          <ProFormTextArea
            colProps={{ xs: 24, lg: 8 }}
            name="allocationNotes"
            label="配舱备注"
            placeholder="请输入配舱备注"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
          <ProFormTextArea
            colProps={{ xs: 24, lg: 8 }}
            name="operationNotes"
            label="操作备注"
            placeholder="请输入操作备注"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
        </>
      ),
    },
    buildSeaPersonnelSection(props),
  ];
}
