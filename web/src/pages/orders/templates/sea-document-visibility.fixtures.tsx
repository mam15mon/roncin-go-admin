import { type ProFormInstance, ProFormText } from '@ant-design/pro-components';
import { render, screen } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { expect, vi } from 'vitest';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type { OrderFormTemplateActions } from '@/components/ui/order-template/types';
import { SeaDocumentStructure } from '@/enums.generated';
import { SeaCreateDocumentModeField } from './components/sea/SeaCreateDocumentModeField';
import { revealSeaFormErrors } from './sea-form-error-reveal';
import { getSeaTemplateSections } from './sea-template';
import type { TemplateProps } from './types';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => false,
  }),
}));
vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
  seaDocumentServicePreviewChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceExecuteChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceUpdateSeaHouseBill: vi.fn(),
  seaDocumentServiceUpdateSeaMasterBillContent: vi.fn(),
}));
vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

export const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
export const DIRECT = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;
const search = async () => [];
const props: TemplateProps = {
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

/** 当前前台提单页签正文区域（页签强制渲染，仅激活正文带 active 类）。 */
export function activeBillPane() {
  const pane = document.querySelector<HTMLElement>('.ant-tabs-content-active');
  if (!pane) throw new Error('提单页签正文未渲染');
  return pane;
}

export function setup(
  mode: SeaDocumentStructure,
  onFinish?: () => Promise<boolean>,
) {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: 1600,
  });
  const formRef: { current: ProFormInstance | undefined } = {
    current: undefined,
  };
  const actionsRef: {
    current: OrderFormTemplateActions<Record<string, unknown>> | undefined;
  } = { current: undefined };
  const sections = getSeaTemplateSections(props).filter(
    (section) => section.key === 'seaDocument',
  );
  const result = render(
    // CSS 零动画不会停止 rc-motion 帧队列；业务断言使用无动画上下文。
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <OrderFormTemplate
          formRef={formRef}
          actionsRef={actionsRef}
          initialValues={{ seaDocumentStructure: mode }}
          enableCloseGuard={false}
          submitter={false}
          sections={[
            {
              key: 'mode',
              title: '提单模式',
              content: (
                <>
                  <ProFormText name="masterNo" />
                  <SeaCreateDocumentModeField />
                </>
              ),
            },
            ...sections,
          ]}
          onFinish={onFinish}
          onRevealError={({ errorFields }) => revealSeaFormErrors(errorFields)}
        />
      </App>
    </ConfigProvider>,
  );
  return { ...result, formRef, actionsRef };
}

export function expectHblTabPresent(present: boolean) {
  if (present) {
    expect(screen.getByRole('tab', { name: /分单 \(HBL\)/ })).toBeVisible();
  } else {
    expect(
      screen.queryByRole('tab', { name: /分单 \(HBL\)/ }),
    ).not.toBeInTheDocument();
  }
  expect(screen.getByRole('tab', { name: /主单 \(MBL\)/ })).toBeVisible();
}
