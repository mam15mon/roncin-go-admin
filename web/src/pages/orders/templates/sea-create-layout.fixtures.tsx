import { ProForm } from '@ant-design/pro-components';
import { App, ConfigProvider, Form, type FormInstance } from 'antd';
import { SeaDocumentStructure } from '@/enums.generated';
import { getSeaTemplateSections } from './sea-template';
import type { TemplateProps } from './types';

const search = async () => [];
export const props: TemplateProps = {
  cargoCategoryOptions: [],
  locationOptions: [],
  currencyOptions: [],
  containerSpecOptions: [],
  personnelOptions: [],
  searchLocations: search,
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
};

export const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
export const DIRECT = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

/** 当前前台提单页签正文区域（页签强制渲染，仅激活正文带 active 类）。 */
export function activeBillPane() {
  const pane = document.querySelector<HTMLElement>('.ant-tabs-content-active');
  if (!pane) throw new Error('提单页签正文未渲染');
  return pane;
}

export function SectionCards({
  exposeForm,
  initialValues,
  sectionKeys,
  isDetail = false,
  readonly = false,
}: {
  exposeForm: (form: FormInstance) => void;
  initialValues?: Record<string, unknown>;
  sectionKeys?: string[];
  isDetail?: boolean;
  readonly?: boolean;
}) {
  const [form] = Form.useForm();
  exposeForm(form);
  const sections = getSeaTemplateSections({
    ...props,
    isDetail,
    readonly,
  }).filter((section) => !sectionKeys || sectionKeys.includes(section.key));
  return (
    // CSS 零动画不会停止 rc-motion 帧队列；业务断言使用无动画上下文。
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <ProForm
          form={form}
          submitter={false}
          layout="vertical"
          initialValues={initialValues}
        >
          {sections.map((section) => (
            <section key={section.key} aria-label={section.title}>
              {section.content}
            </section>
          ))}
        </ProForm>
      </App>
    </ConfigProvider>
  );
}
