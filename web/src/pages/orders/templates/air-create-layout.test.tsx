import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { getAirTemplateSections } from './air-template';
import type { TemplateProps } from './types';

vi.mock('@/services/roncin/masterDataService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/masterDataService')
  >()),
  masterDataServiceListAirlines: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/airExportService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/airExportService')
  >()),
  airExportServiceGetAirMasterWaybill: vi.fn().mockResolvedValue(undefined),
  airExportServiceListAirMasterWaybills: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

const search = async () => [];
const props: TemplateProps = {
  serviceTypeOptions: [],
  cargoCategoryOptions: [],
  locationOptions: [],
  searchLocations: search,
  currencyOptions: [],
  containerSpecOptions: [],
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
  personnelOptions: [],
};

const user = {
  id: 'user-1',
  currentOrganization: { id: 'org-1', name: '测试组织' },
};

function SectionCards({
  initialValues,
  isDetail = false,
}: {
  initialValues?: Record<string, unknown>;
  isDetail?: boolean;
}) {
  const [form] = Form.useForm();
  const sections = getAirTemplateSections({ ...props, isDetail });
  return (
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
  );
}

// 挂载期 request 型下拉会发起异步选项查询；在 act 内冲刷微任务，
// 确保用例结束前已触发的异步流全部落地，避免迟到 setState 触发 act 警告。
async function flushMountRequests() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe('AE 新建与详情五卡片合同', () => {
  it('新建与详情返回同一套五卡片 key、标题与顺序（对齐海运出口节奏）', () => {
    const expected = [
      { key: 'businessCustomer', title: '业务与客户' },
      { key: 'bookingTransport', title: '订舱与运输' },
      { key: 'cargoCommercial', title: '本票货物' },
      { key: 'airDocuments', title: '分单信息' },
      { key: 'internalPersonnel', title: '内部人员' },
    ];
    expect(
      getAirTemplateSections(props).map(({ key, title }) => ({ key, title })),
    ).toEqual(expected);
    expect(
      getAirTemplateSections({ ...props, isDetail: true }).map(
        ({ key, title }) => ({ key, title }),
      ),
    ).toEqual(expected);
  });

  it('订舱与运输卡按小节分组编排：主单与订舱 / 航线与班期 / 时刻与申报', async () => {
    renderWithApp(<SectionCards />, { user });
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    // textContent 顺序即 DOM 顺序：三个小节标题按固定顺序出现。
    const groupIndexes = ['主单与订舱', '航线与班期', '时刻与申报'].map(
      (title) => booking.textContent?.indexOf(title) ?? -1,
    );
    expect(groupIndexes.every((index) => index >= 0)).toBe(true);
    expect([...groupIndexes].sort((a, b) => a - b)).toEqual(groupIndexes);
    // 主单备注与操作备注为可折叠备注（NEW 模式两者都挂载，主单备注在前）。
    const masterNotes = within(booking).getByTestId(
      'sea-notes-airMaster.notes',
    );
    const operationNotes = within(booking).getByTestId(
      'sea-notes-operationNotes',
    );
    expect(masterNotes).toBeInTheDocument();
    expect(operationNotes).toBeInTheDocument();
    expect(
      masterNotes.compareDocumentPosition(operationNotes) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    // 订舱号归入订舱与运输卡的主单小节；不渲染海运专属备注字段。
    expect(booking).toHaveTextContent('订舱号');
    expect(booking).toHaveTextContent('主运单号（MAWB）');
    expect(booking).toHaveTextContent('航班号');
    expect(booking).not.toHaveTextContent('订舱备注');
    expect(booking).not.toHaveTextContent('配舱备注');
    await flushMountRequests();
  });

  it('业务与客户卡不再包含订舱号，分单信息卡承接 HAWB 与单证资料', async () => {
    renderWithApp(<SectionCards />, { user });
    const business = screen.getByRole('region', { name: '业务与客户' });
    expect(business).toHaveTextContent('订单编号');
    expect(business).toHaveTextContent('委托单位');
    expect(business).not.toHaveTextContent('订舱号');
    expect(business).not.toHaveTextContent('本票分运单号');

    const documents = screen.getByRole('region', { name: '分单信息' });
    expect(documents).toHaveTextContent('本票分运单号（HAWB）');
    expect(documents).toHaveTextContent('本票分单状态');
    expect(documents).toHaveTextContent('本票单证资料');
    // 详情已签发才展示的修改原因在新建模式不出现。
    expect(documents).not.toHaveTextContent('已签发分单修改原因');
    await flushMountRequests();
  });

  it('详情模式展示共享主单摘要与订舱号，主单编辑字段收起', async () => {
    renderWithApp(
      <SectionCards
        isDetail
        initialValues={{
          airExport: {
            houseStatus: 'RELEASED',
            master: {
              id: 'master-1',
              airlineName: '测试航空',
              content: { masterNo: '78112345675' },
            },
          },
        }}
      />,
      { user },
    );
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    expect(booking).toHaveTextContent('共享主单 78112345675');
    expect(booking).toHaveTextContent('订舱号');
    expect(booking).not.toHaveTextContent('主单关联方式');
    expect(booking).not.toHaveTextContent('主运单号（MAWB）');
    expect(booking).not.toHaveTextContent('航线与班期');

    const documents = screen.getByRole('region', { name: '分单信息' });
    expect(documents).toHaveTextContent('已签发分单修改原因');
    await flushMountRequests();
  });

  it('切到关联已有主单时收起航线/申报小节与主单备注，保留操作备注', async () => {
    renderWithApp(<SectionCards />, { user });
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    expect(booking).toHaveTextContent('航线与班期');

    fireEvent.click(screen.getByRole('radio', { name: '明确关联已有主单' }));
    await waitFor(() =>
      expect(
        screen.getByRole('region', { name: '订舱与运输' }),
      ).not.toHaveTextContent('航线与班期'),
    );
    const existing = screen.getByRole('region', { name: '订舱与运输' });
    expect(existing).toHaveTextContent('选择共享主单');
    expect(existing).not.toHaveTextContent('时刻与申报');
    expect(
      within(existing).queryByTestId('sea-notes-airMaster.notes'),
    ).toBeNull();
    expect(
      within(existing).getByTestId('sea-notes-operationNotes'),
    ).toBeInTheDocument();
    await flushMountRequests();
  });
});
