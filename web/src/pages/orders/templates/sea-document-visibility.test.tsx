import { type ProFormInstance, ProFormText } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type { OrderFormTemplateActions } from '@/components/ui/order-template/types';
import { SeaDocumentStructure } from '@/enums.generated';
import { SeaCreateDocumentModeField } from './components/sea/SeaCreateDocumentModeField';
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
    canOperateOrganization: () => true,
  }),
}));

const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
const DIRECT = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;
const search = async () => [];
const props: TemplateProps = {
  serviceTypeOptions: [],
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
  checkCustomerReferenceNo: async () => {},
  checkInternalReferenceNo: async () => {},
};

function setup(mode: SeaDocumentStructure) {
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
  const houseSection = getSeaTemplateSections(props).find(
    (section) => section.key === 'houseBillContent',
  );
  if (!houseSection) throw new Error('缺少 HBL 分节配置');
  const result = render(
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
            content: <SeaCreateDocumentModeField />,
          },
          {
            key: 'master',
            title: 'MBL 主单内容',
            content: <ProFormText name="masterNo" />,
          },
          houseSection,
        ]}
      />
    </App>,
  );
  return { ...result, formRef, actionsRef };
}

function expectHouseVisible(visible: boolean) {
  if (visible) {
    expect(
      document.querySelector('#section-houseBillContent'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /HBL 分单内容/ }),
    ).toBeInTheDocument();
  } else {
    expect(
      document.querySelector('#section-houseBillContent'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /HBL 分单内容/ }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/DIRECT 不签发 HBL/)).not.toBeInTheDocument();
  }
  expect(document.querySelector('#section-master')).toBeInTheDocument();
}

describe('海运分单分节与导航可见性', () => {
  it('初始 DIRECT 隐藏整节，切换 HOUSE 再切回同步移除导航和必填校验', async () => {
    const { formRef } = setup(DIRECT);
    expectHouseVisible(false);
    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    await waitFor(() => expectHouseVisible(true));
    await act(async () => {
      await expect(formRef.current?.validateFields()).rejects.toMatchObject({
        errorFields: expect.any(Array),
      });
    });
    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() => expectHouseVisible(false));
    await act(async () => {
      await expect(formRef.current?.validateFields()).resolves.toBeDefined();
    });
    expect(formRef.current?.getFieldValue('seaHouseBill')).toBeUndefined();
  });

  it('程序回填与 resetTo 同步更新分节可见性', async () => {
    const { formRef, actionsRef } = setup(HOUSE);
    await waitFor(() => expectHouseVisible(true));
    act(() =>
      formRef.current?.setFieldsValue({ seaDocumentStructure: DIRECT }),
    );
    await waitFor(() => expectHouseVisible(false));
    act(() => actionsRef.current?.resetTo({ seaDocumentStructure: HOUSE }));
    await waitFor(() => expectHouseVisible(true));
    act(() => actionsRef.current?.resetTo({ seaDocumentStructure: DIRECT }));
    await waitFor(() => expectHouseVisible(false));
  });
});
