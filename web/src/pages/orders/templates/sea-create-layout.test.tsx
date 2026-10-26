import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import type { FormInstance } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { SeaHouseBillIssuerSource } from '@/enums.generated';
import { buildSeaExportCreatePayload } from '../order-kinds/sea-export/form-adapter';
import {
  activeBillPane,
  HOUSE,
  props,
  SectionCards,
} from './sea-create-layout.fixtures';
import { getSeaTemplateSections } from './sea-template';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
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

describe('SE 新建与详情五卡片合同', () => {
  it('新建与详情返回同一套五卡片 key、标题与顺序', () => {
    const expected = [
      { key: 'businessCustomer', title: '业务与客户' },
      { key: 'bookingTransport', title: '订舱与运输' },
      { key: 'cargoCommercial', title: '货物与商业' },
      { key: 'seaDocument', title: '提单信息' },
      { key: 'internalPersonnel', title: '内部人员' },
    ];
    expect(
      getSeaTemplateSections(props).map(({ key, title }) => ({ key, title })),
    ).toEqual(expected);
    expect(
      getSeaTemplateSections({ ...props, isDetail: true }).map(
        ({ key, title }) => ({ key, title }),
      ),
    ).toEqual(expected);
  });

  it('第二卡按业务分组编排，操作备注合并为一行', async () => {
    let _form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          _form = value;
        }}
        sectionKeys={['bookingTransport']}
      />,
    );
    const bookingSection = screen.getByRole('region', {
      name: '订舱与运输',
    });
    expect(bookingSection).toHaveTextContent('订舱与主单');
    expect(bookingSection).toHaveTextContent('航线与船期');
    expect(bookingSection).toHaveTextContent('箱量与截关');
    expect(bookingSection).not.toHaveTextContent('操作补充');
    expect(within(bookingSection).getAllByText('操作备注')).toHaveLength(1);
    // 订舱号、船公司、MBL 主单号、提单模式同在第一小节。
    expect(bookingSection).toHaveTextContent('订舱号');
    expect(bookingSection).toHaveTextContent('船公司');
    expect(bookingSection).toHaveTextContent('MBL 主单号');
    expect(bookingSection).toHaveTextContent('提单模式');
    await act(async () => {});
  });

  it('委托件重尺仅在货物与商业卡编辑，提单页签提供只读对照与实际录入', async () => {
    let _form!: FormInstance;
    const { container } = render(
      <SectionCards
        exposeForm={(value) => {
          _form = value;
        }}
        sectionKeys={['cargoCommercial', 'seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );
    for (const id of [
      'totalPackages',
      'totalPackageUnit',
      'totalGrossWeightKg',
      'totalVolumeCbm',
    ]) {
      expect(container.querySelectorAll(`[id="${id}"]`)).toHaveLength(1);
    }
    const cargoSection = screen.getByRole('region', { name: '货物与商业' });
    expect(cargoSection).toHaveTextContent('委托件数');
    expect(cargoSection).toHaveTextContent('委托毛重');
    expect(cargoSection).toHaveTextContent('委托体积');
    // 提单卡保留实际件重尺标题；委托摘要不重复注册编辑字段。
    expect(screen.getByText('MBL 实际件重尺')).toBeVisible();
    expect(screen.queryByText('委托 vs 实际件重尺对照')).toBeNull();
    expect(screen.queryByText('关联放货记录')).toBeNull();
    expect(screen.queryByText('暂无关联放货记录')).toBeNull();
    await act(async () => {});
  });

  it('散杂托运在货物与商业卡展示散杂计费吨提示', async () => {
    render(
      <SectionCards
        exposeForm={() => {}}
        sectionKeys={[
          'businessCustomer',
          'cargoCommercial',
          'bookingTransport',
        ]}
        initialValues={{
          shipmentType: 3,
          totalGrossWeightKg: 2500,
          totalVolumeCbm: 1.8,
          containerRequests: [],
        }}
      />,
    );
    expect(screen.getByText('散杂计费吨 (RT)：2.500')).toBeVisible();
    expect(
      screen.getByText('散杂货不使用箱型箱量、箱号或封号配置'),
    ).toBeVisible();
    await act(async () => {});
  });
});

describe('SE 提单卡 MBL/HBL 页签', () => {
  it('HOUSE 一次只展示一个正文，切换页签不丢已填数据', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );

    // 默认展示 MBL 正文，HBL 正文隐藏但页签存在（HBL 自身也有发货人字段，
    // 以字段 id 断言正文归属）。
    expect(
      activeBillPane().querySelector('#seaMasterBillContent_shipperText'),
    ).not.toBeNull();
    expect(activeBillPane().querySelector('#seaHouseBill_houseNo')).toBeNull();

    fireEvent.change(
      within(activeBillPane()).getByPlaceholderText(
        '请输入发货人英文名称与详细地址',
      ),
      { target: { value: 'SHIPPER TEXT' } },
    );
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(
        activeBillPane().querySelector('#seaHouseBill_houseNo'),
      ).not.toBeNull(),
    );
    expect(
      activeBillPane().querySelector('#seaMasterBillContent_shipperText'),
    ).toBeNull();
    fireEvent.change(
      within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      {
        target: { value: 'HBL001' },
      },
    );

    // 切回 MBL：两侧草稿均保留。
    fireEvent.click(screen.getByRole('tab', { name: /主单 \(MBL\)/ }));
    await waitFor(() =>
      expect(
        activeBillPane().querySelector('#seaMasterBillContent_shipperText'),
      ).not.toBeNull(),
    );
    expect(form.getFieldValue(['seaMasterBillContent', 'shipperText'])).toBe(
      'SHIPPER TEXT',
    );
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL001');
  });

  it('HOUSE 必填在从未打开 HBL 页签时同样拦截提交', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );
    expect(screen.queryByPlaceholderText('请输入分单号')).not.toBeVisible();

    await act(async () => {
      await expect(
        form.validateFields([
          ['seaHouseBill', 'houseNo'],
          ['seaHouseBill', 'issuerSource'],
        ]),
      ).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
          expect.objectContaining({ name: ['seaHouseBill', 'issuerSource'] }),
        ]),
      });
    });
  });

  it('DIRECT 不出现 HBL 页签、不校验不提交隐藏草稿，切回 HOUSE 恢复已填值', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );

    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    const houseNoInput = await waitFor(() =>
      within(activeBillPane()).getByPlaceholderText('请输入分单号'),
    );
    fireEvent.change(houseNoInput, { target: { value: 'HBL-KEEP' } });
    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByRole('tab', { name: /分单 \(HBL\)/ }),
      ).not.toBeInTheDocument(),
    );
    // 隐藏草稿保留在 Form store，但提交口径不含 HBL。
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-KEEP');
    await act(async () => {
      await expect(
        form.validateFields([
          ['seaHouseBill', 'houseNo'],
          ['seaHouseBill', 'issuerSource'],
        ]),
      ).resolves.toBeDefined();
    });
    const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(payload.seaDocument?.houseBill).toBeUndefined();

    // 切回 HOUSE：HBL 页签恢复，草稿值原样保留。
    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    fireEvent.click(await screen.findByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      ).toBeVisible(),
    );
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-KEEP');
    const housePayload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(housePayload.seaDocument?.houseBill?.houseNo).toBe('HBL-KEEP');
  });

  it('带入委托件重尺只改当前提单实际值并保持提交口径', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['cargoCommercial', 'seaDocument']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          totalPackages: 10,
          totalPackageUnit: 'CTNS',
          totalGrossWeightKg: 100,
          totalVolumeCbm: 2,
          seaMasterBillContent: {
            packageCount: 11,
            grossWeightKg: 110,
            volumeCbm: 3,
          },
          seaHouseBill: {
            houseNo: 'HBL001',
            issuerSource:
              SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
            content: { packageCount: 12, grossWeightKg: 120, volumeCbm: 4 },
          },
        }}
      />,
    );
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toBeVisible(),
    );
    const house = screen.getByRole('region', { name: '提单信息' });
    fireEvent.click(
      within(house).getByRole('button', { name: /带入委托件重尺/ }),
    );
    await waitFor(() =>
      expect(
        form.getFieldValue(['seaHouseBill', 'content', 'packageCount']),
      ).toBe(10),
    );
    expect(form.getFieldValue(['seaMasterBillContent', 'packageCount'])).toBe(
      11,
    );
    expect(form.getFieldValue('totalPackages')).toBe(10);
    const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(payload.totalPackages).toBe(10);
    expect(payload.seaDocument?.masterBillContent?.packageCount).toBe(11);
    expect(payload.seaDocument?.houseBill?.content?.packageCount).toBe(10);
    expect(payload.seaDocument?.houseBill?.houseNo).toBe('HBL001');
  });
});
