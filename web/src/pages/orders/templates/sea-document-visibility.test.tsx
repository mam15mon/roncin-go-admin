import { type ProFormInstance, ProFormText } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
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

/** 当前前台提单页签正文区域（页签强制渲染，仅激活正文带 active 类）。 */
function activeBillPane() {
  const pane = document.querySelector<HTMLElement>('.ant-tabs-content-active');
  if (!pane) throw new Error('提单页签正文未渲染');
  return pane;
}

function setup(mode: SeaDocumentStructure, onFinish?: () => Promise<boolean>) {
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
    </App>,
  );
  return { ...result, formRef, actionsRef };
}

function expectHblTabPresent(present: boolean) {
  if (present) {
    expect(screen.getByRole('tab', { name: /分单 \(HBL\)/ })).toBeVisible();
  } else {
    expect(
      screen.queryByRole('tab', { name: /分单 \(HBL\)/ }),
    ).not.toBeInTheDocument();
  }
  expect(screen.getByRole('tab', { name: /主单 \(MBL\)/ })).toBeVisible();
}

describe('海运提单卡页签可见性', () => {
  it('初始 DIRECT 无 HBL 页签，切换 HOUSE 同步出现并注册必填校验', async () => {
    const { formRef } = setup(DIRECT);
    expectHblTabPresent(false);

    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    await waitFor(() => expectHblTabPresent(true));
    // 页签强制渲染：HBL 必填在从未点击页签时同样注册。
    await act(async () => {
      await expect(formRef.current?.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
        ]),
      });
    });

    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() => expectHblTabPresent(false));
    await act(async () => {
      await expect(formRef.current?.validateFields()).resolves.toBeDefined();
    });
  });

  it('程序回填与 resetTo 同步更新页签可见性', async () => {
    const { formRef, actionsRef } = setup(HOUSE);
    await waitFor(() => expectHblTabPresent(true));
    act(() =>
      formRef.current?.setFieldsValue({ seaDocumentStructure: DIRECT }),
    );
    await waitFor(() => expectHblTabPresent(false));
    act(() => actionsRef.current?.resetTo({ seaDocumentStructure: HOUSE }));
    await waitFor(() => expectHblTabPresent(true));
  });
});

describe('提单页签校验失败定位链路', () => {
  it('提交校验命中 HBL 首个错误时自动切到 HBL 页签', async () => {
    const { formRef } = setup(HOUSE);
    // 默认 MBL 页签在前台，HBL 输入不可见。
    expect(
      within(activeBillPane()).queryByPlaceholderText('请输入分单号'),
    ).toBeNull();

    await act(async () => {
      formRef.current?.submit();
    });
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      ).toBeVisible(),
    );
  });

  it('DIRECT 无 HBL 页签时提交直接通过，不触发隐藏校验', async () => {
    const onFinish = vi.fn().mockResolvedValue(true);
    const { formRef } = setup(DIRECT, onFinish);
    await act(async () => {
      formRef.current?.submit();
    });
    await waitFor(() => expect(onFinish).toHaveBeenCalled());
  });

  it('revealSeaFormErrors 按首个错误切换前台页签', async () => {
    setup(HOUSE);
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      ).toBeVisible(),
    );
    act(() => {
      revealSeaFormErrors([
        { name: ['seaMasterBillContent', 'shipperText'], errors: ['必填'] },
      ]);
    });
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText(
          '请输入发货人英文名称与详细地址',
        ),
      ).toBeVisible(),
    );
  });
});
