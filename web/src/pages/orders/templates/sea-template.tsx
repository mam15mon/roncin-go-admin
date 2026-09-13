import { ProFormTextArea } from '@ant-design/pro-components';
import { Col, Tag } from 'antd';
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
import {
  buildSeaDocumentSection,
  SeaDocumentSectionComponent,
} from './components/sea/SeaDocumentSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import {
  buildSeaTransportSection,
  SeaContainerPlanFields,
  SeaScheduleDateFields,
} from './components/sea/SeaTransportSection';
import type { TemplateProps, TemplateSection } from './types';

export {
  buildSeaCargoSection,
  buildSeaDocumentSection,
  SeaCargoMeasurementFields,
  SeaContainerPlanFields,
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
        {/* 1. 委托货物申报 */}
        <Col span={24}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              marginBottom: 16,
            }}
          >
            <div
              style={{
                width: 3,
                height: 14,
                backgroundColor: '#1677ff',
                borderRadius: 2,
                marginRight: 8,
              }}
            />
            <span style={{ fontSize: 13, fontWeight: 600, color: '#1f2329' }}>
              货物委托信息 (Entrusted Cargo)
            </span>
            <Tag color="blue" style={{ marginLeft: 8 }}>
              委托申报数据
            </Tag>
          </div>
        </Col>

        {cargoSection.content}

        {/* 2. 分隔线与海运单证 */}
        <Col span={24}>
          <div
            style={{
              height: 1,
              backgroundColor: '#f0f0f0',
              margin: '12px 0 20px 0',
            }}
          />
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              marginBottom: 16,
            }}
          >
            <div
              style={{
                width: 3,
                height: 14,
                backgroundColor: '#1677ff',
                borderRadius: 2,
                marginRight: 8,
              }}
            />
            <span style={{ fontSize: 13, fontWeight: 600, color: '#1f2329' }}>
              海运提单单证 (Ocean Documents)
            </span>
            <Tag color="cyan" style={{ marginLeft: 8 }}>
              提单实际数据
            </Tag>
          </div>
        </Col>

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
  return [
    buildSeaBaseInfoSection(props),
    buildSeaTransportSection(props),
    buildSeaCargoAndDocumentSection(props),
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
