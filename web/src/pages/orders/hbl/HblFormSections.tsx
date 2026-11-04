import {
  ProFormDatePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { FormRow } from '@/components/ui';
import type { OrderFormTemplateSection } from '@/components/ui/order-template/types';
import {
  SEA_BILL_FORM_OPTIONS,
  SEA_FREIGHT_TERM_OPTIONS,
  SEA_RELEASE_TYPE_OPTIONS,
  SEA_TRANSPORT_TERM_OPTIONS,
} from '../templates/components/sea/seaDocumentSectionConstants';

type PrintTextKey = Exclude<
  keyof API.SeaHouseBillPrintDetails,
  'originalCount' | 'issueDate' | 'shippedOnBoardDate'
>;

function PrintText({
  field,
  label,
  multiline = false,
  max = 500,
}: {
  field: PrintTextKey;
  label: string;
  multiline?: boolean;
  max?: number;
}) {
  const shared = {
    name: ['printDetails', field],
    label,
    rules: [{ max, message: `最多 ${max} 个字符` }],
  };
  return multiline ? (
    <ProFormTextArea
      {...shared}
      fieldProps={{ autoSize: { minRows: 3, maxRows: 8 } }}
    />
  ) : (
    <ProFormText {...shared} />
  );
}

export function buildHblFormSections({
  containerAction,
  sourceAction,
}: {
  containerAction: React.ReactNode;
  sourceAction: React.ReactNode;
}): OrderFormTemplateSection[] {
  return [
    {
      key: 'parties',
      title: '收发货人与通知人',
      content: (
        <FormRow cols={2}>
          {(
            [
              ['shipperText', '发货人 (Shipper)'],
              ['consigneeText', '收货人 (Consignee)'],
              ['notifyPartyText', '通知人 (Notify Party)'],
              ['secondNotifyPartyText', '第二通知人'],
              ['foreignAgentText', '国外代理 (Forwarding Agent)'],
            ] as const
          ).map(([field, label]) => (
            <ProFormTextArea
              key={field}
              name={['content', field]}
              label={label}
              rules={[{ max: 5000, message: '最多 5000 个字符' }]}
              fieldProps={{ autoSize: { minRows: 4, maxRows: 10 } }}
            />
          ))}
        </FormRow>
      ),
    },
    {
      key: 'transport',
      title: '运输与印字信息',
      extra: sourceAction,
      content: (
        <>
          <FormRow cols={3}>
            <PrintText field="bookingNoText" label="订舱号 (Booking No.)" />
            <PrintText
              field="vesselVoyageText"
              label="船名航次 (Vessel / Voyage)"
            />
            <PrintText
              field="portOfLoadingText"
              label="装货港 (Port of Loading)"
            />
            <PrintText
              field="portOfDischargeText"
              label="卸货港 (Port of Discharge)"
            />
            <PrintText
              field="placeOfDeliveryText"
              label="交货地 (Place of Delivery)"
            />
            <PrintText
              field="placeOfReceiptText"
              label="收货地 (Place of Receipt)"
            />
            <PrintText
              field="preCarriageText"
              label="前程运输 (Pre-carriage)"
            />
            <PrintText
              field="pointOfOriginText"
              label="原产地 (Point of Origin)"
            />
            <PrintText
              field="onwardRoutingText"
              label="后续内陆运输"
              multiline
              max={3000}
            />
          </FormRow>
          <FormRow cols={2}>
            <PrintText
              field="exportReferencesText"
              label="出口参考号"
              multiline
              max={3000}
            />
            <PrintText
              field="domesticRoutingText"
              label="国内运输／出口指示"
              multiline
              max={3000}
            />
          </FormRow>
        </>
      ),
    },
    {
      key: 'cargo',
      title: '货物与箱号',
      extra: containerAction,
      content: (
        <>
          <FormRow cols={2}>
            <ProFormTextArea
              name={['content', 'marksText']}
              label="唛头 (Marks and Numbers)"
              rules={[{ max: 5000, message: '最多 5000 个字符' }]}
              fieldProps={{ autoSize: { minRows: 5, maxRows: 12 } }}
            />
            <ProFormTextArea
              name={['content', 'goodsDescriptionText']}
              label="货物描述 (Description of Goods)"
              rules={[{ max: 10000, message: '最多 10000 个字符' }]}
              fieldProps={{ autoSize: { minRows: 5, maxRows: 16 } }}
            />
          </FormRow>
          <FormRow cols={4}>
            <ProFormDigit
              name={['content', 'packageCount']}
              label="件数"
              min={0}
              fieldProps={{ precision: 0 }}
            />
            <ProFormText
              name={['content', 'packageUnit']}
              label="包装单位"
              rules={[{ max: 64 }]}
            />
            <ProFormDigit
              name={['content', 'grossWeightKg']}
              label="毛重 (KGS)"
              min={0}
            />
            <ProFormDigit
              name={['content', 'volumeCbm']}
              label="体积 (CBM)"
              min={0}
            />
          </FormRow>
          <PrintText
            field="containersText"
            label="箱号／封号／箱型"
            multiline
            max={12000}
          />
          <FormRow cols={4}>
            <ProFormSelect
              name={['content', 'freightTerms']}
              label="运费条款"
              options={SEA_FREIGHT_TERM_OPTIONS}
            />
            <ProFormSelect
              name={['content', 'transportTerms']}
              label="运输条款"
              options={SEA_TRANSPORT_TERM_OPTIONS}
            />
            <ProFormSelect
              name={['content', 'billForm']}
              label="提单形式"
              options={SEA_BILL_FORM_OPTIONS}
            />
            <ProFormSelect
              name={['content', 'releaseType']}
              label="放单方式"
              options={SEA_RELEASE_TYPE_OPTIONS}
            />
          </FormRow>
          <ProFormTextArea
            name={['content', 'clauses']}
            label="特别条款"
            rules={[{ max: 10000 }]}
            fieldProps={{ autoSize: { minRows: 3, maxRows: 8 } }}
          />
        </>
      ),
    },
    {
      key: 'issuance',
      title: '签发与补充信息',
      content: (
        <>
          <FormRow cols={4}>
            <ProFormDigit
              name={['printDetails', 'originalCount']}
              label="正本份数"
              min={0}
              max={99}
              fieldProps={{ precision: 0 }}
            />
            <ProFormDatePicker
              name={['printDetails', 'shippedOnBoardDate']}
              label="实际装船日期"
              fieldProps={{ format: 'YYYY-MM-DD', style: { width: '100%' } }}
            />
            <PrintText field="issuePlaceText" label="签发地点" />
            <ProFormDatePicker
              name={['printDetails', 'issueDate']}
              label="签发日期"
              fieldProps={{ format: 'YYYY-MM-DD', style: { width: '100%' } }}
            />
          </FormRow>
          <FormRow cols={2}>
            <PrintText
              field="headerContactText"
              label="页眉联系信息"
              multiline
              max={3000}
            />
            <PrintText
              field="deliveryContactText"
              label="提货联系信息"
              multiline
              max={3000}
            />
          </FormRow>
          <PrintText
            field="footerReferenceText"
            label="页脚参考信息"
            multiline
            max={500}
          />
        </>
      ),
    },
  ];
}
