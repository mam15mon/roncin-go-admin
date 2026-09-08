import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { App } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { TradeTerm } from '@/enums.generated';
import NewOrderPage from './new';
import { useOrderCreateOptions } from './use-order-create-options';

vi.mock('@umijs/max', () => ({
  useParams: () => ({ kind: 'sea-export' }),
  useAccess: () => ({ canOrder: () => true }),
  useModel: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        displayName: '张三',
        currentOrganization: { id: 'org-1', name: '总公司' },
      },
    },
  }),
  history: {
    push: vi.fn(),
    replace: vi.fn(),
  },
  Link: ({ to, children, ...rest }: any) => (
    <a href={to} {...rest}>
      {children}
    </a>
  ),
}));

vi.mock('./use-order-create-options', () => ({
  useOrderCreateOptions: vi.fn(),
}));

let lastTemplateProps: any = null;

vi.mock('@/components/ui/order-template/OrderFormTemplate', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/components/ui/order-template/OrderFormTemplate')>();
  return {
    ...actual,
    OrderFormTemplate: (props: any) => {
      lastTemplateProps = props;
      return actual.OrderFormTemplate(props);
    },
  };
});

const mockUseOptions = vi.mocked(useOrderCreateOptions);

function renderWithApp(ui: React.ReactElement) {
  return render(<App>{ui}</App>);
}

describe('NewOrderPage', () => {
  const defaultRetry = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    lastTemplateProps = null;
    mockUseOptions.mockReturnValue({
      loading: false,
      error: null,
      retry: defaultRetry,
      serviceTypeOptions: [],
      cargoCategoryOptions: [
        { label: '化工品', value: 'cat-chem', code: 'CHEM' },
        { label: '普通货物', value: 'cat-gen-id', code: 'GENERAL' },
      ],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      personnelOptions: [],
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('loading 为 true 时展示分节骨架屏与 loadingTip，不挂载表单', () => {
    mockUseOptions.mockReturnValue({
      loading: true,
      error: null,
      retry: defaultRetry,
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn(),
      currencyOptions: [],
      containerSpecOptions: [],
      personnelOptions: [],
    });

    renderWithApp(<NewOrderPage />);

    expect(screen.getByText('正在加载业务模板与主数据...')).toBeInTheDocument();
    expect(screen.getByText('业务基本信息')).toBeInTheDocument();
    expect(screen.getByText('运输与订舱信息')).toBeInTheDocument();
    expect(screen.queryByText('创建订单')).not.toBeInTheDocument();
  });

  it('主数据加载失败时渲染错误提示卡片与重新加载按钮，严防展示空表单', () => {
    mockUseOptions.mockReturnValue({
      loading: false,
      error: new Error('后端主数据服务不可用'),
      retry: defaultRetry,
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn(),
      currencyOptions: [],
      containerSpecOptions: [],
      personnelOptions: [],
    });

    renderWithApp(<NewOrderPage />);

    expect(screen.getByText('主数据加载失败')).toBeInTheDocument();
    expect(screen.getByText('后端主数据服务不可用')).toBeInTheDocument();
    expect(screen.getByText('重新加载')).toBeInTheDocument();
    expect(screen.queryByText('创建订单')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('重新加载'));
    expect(defaultRetry).toHaveBeenCalledTimes(1);
  });

  it('优先按 GENERAL 业务码识别默认货物类别，无论名称是否为普货', () => {
    const { container } = renderWithApp(<NewOrderPage />);

    expect(screen.getByText('创建订单')).toBeInTheDocument();
    expect(container.querySelector('form')).toBeInTheDocument();
    expect(lastTemplateProps?.initialValues?.cargoCategoryIds).toEqual([
      'cat-gen-id',
    ]);
  });

  it('当 code=GENERAL 与 label=普货 分属不同选项冲突时，优先采用 code=GENERAL 的选项', () => {
    mockUseOptions.mockReturnValue({
      loading: false,
      error: null,
      retry: defaultRetry,
      serviceTypeOptions: [],
      cargoCategoryOptions: [
        { label: '普货', value: 'cat-pu-wrong', code: 'OTHER' },
        { label: '集装箱普通货', value: 'cat-gen-correct', code: 'GENERAL' },
      ],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      personnelOptions: [],
    });

    renderWithApp(<NewOrderPage />);

    expect(lastTemplateProps?.initialValues?.cargoCategoryIds).toEqual([
      'cat-gen-correct',
    ]);
  });

  it('当无 code=GENERAL 时，优雅回退采用 label=普货 的选项', () => {
    mockUseOptions.mockReturnValue({
      loading: false,
      error: null,
      retry: defaultRetry,
      serviceTypeOptions: [],
      cargoCategoryOptions: [
        { label: '危险品', value: 'cat-dg', code: 'DG' },
        { label: '普货', value: 'cat-pu-fallback' },
      ],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      personnelOptions: [],
    });

    renderWithApp(<NewOrderPage />);

    expect(lastTemplateProps?.initialValues?.cargoCategoryIds).toEqual([
      'cat-pu-fallback',
    ]);
  });

  it('当既无 code=GENERAL 也无 label=普货 时，cargoCategoryIds 为 undefined', () => {
    mockUseOptions.mockReturnValue({
      loading: false,
      error: null,
      retry: defaultRetry,
      serviceTypeOptions: [],
      cargoCategoryOptions: [
        { label: '危险品', value: 'cat-dg', code: 'DG' },
        { label: '化工品', value: 'cat-chem', code: 'CHEM' },
      ],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      personnelOptions: [],
    });

    renderWithApp(<NewOrderPage />);

    expect(lastTemplateProps?.initialValues?.cargoCategoryIds).toBeUndefined();
  });

  it('海运出口订单默认贸易条款为 CIF', () => {
    renderWithApp(<NewOrderPage />);

    expect(lastTemplateProps?.initialValues?.tradeTerm).toBe(
      TradeTerm.TRADE_TERM_CIF,
    );
  });
});
