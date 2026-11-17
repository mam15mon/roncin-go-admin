import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { ConfigProvider, Form } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import { describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import { type EditorValues, resourceTabs } from '../resourceConstants';
import ResourceEditorModal from './ResourceEditorModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

function Harness() {
  const [form] = Form.useForm<EditorValues>();
  const partners = Form.useWatch('partnerIds', form);
  const active = resourceTabs.find((item) => item.type === 3);
  if (!active) throw new Error('缺少图片资源类型');
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <ResourceEditorModal
        active={active}
        open
        setOpen={vi.fn()}
        saving={false}
        form={form}
        onSave={vi.fn()}
        imageFiles={[]}
        setImageFiles={vi.fn()}
        partnerOptions={[{ value: 'existing', label: '已有企业' }]}
        searchPartners={vi.fn()}
        assigneeOptions={[]}
        searchAssignees={vi.fn()}
        provinceOptions={[]}
        cityOptions={[]}
        districtOptions={[]}
        message={{ error: vi.fn() } as unknown as MessageInstance}
      />
      <output data-testid="partners">{JSON.stringify(partners ?? [])}</output>
    </ConfigProvider>
  );
}

describe('企业资源关联企业快捷创建', () => {
  it('新企业追加到原有多选并保留资源名称', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      success: true,
      data: { id: 'created', legalName: '新企业' },
    } as never);
    renderWithApp(<Harness />, {
      user: {
        id: 'user-1',
        username: 'operator',
        currentOrganization: {
          id: 'company-1',
          name: '测试公司',
          kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
        },
        permissionCapabilities: [
          { key: 'business.partner.create', dataScope: 'organization' },
        ],
      },
    });
    fireEvent.change(screen.getByLabelText('简称/名称'), {
      target: { value: '未保存图片资料' },
    });
    fireEvent.mouseDown(screen.getByLabelText('关联企业'));
    fireEvent.click(await screen.findByText('已有企业'));
    fireEvent.mouseDown(screen.getByLabelText('关联企业'));
    const create = (await screen.findByText('快捷新增关联企业')).closest(
      'button',
    );
    if (!create) throw new Error('缺少创建企业入口');
    fireEvent.click(create);
    const input = await screen.findByLabelText('单位全称');
    fireEvent.change(input, { target: { value: '新企业' } });
    const dialog = input.closest('[role="dialog"]') as HTMLElement;
    const save = within(dialog).getByRole('button', { name: '保存并选用' });
    await waitFor(() => expect(save).toBeVisible());
    fireEvent.click(save);
    await waitFor(() =>
      expect(screen.getByTestId('partners')).toHaveTextContent(
        '["existing","created"]',
      ),
    );
    expect(screen.getByLabelText('简称/名称')).toHaveValue('未保存图片资料');
    expect(screen.getByTitle('新企业')).toBeVisible();
  });
});
