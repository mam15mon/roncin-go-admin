import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import * as orderService from '@/services/roncin/orderService';
import {
  candidateResponse,
  renderVoyageForm,
  voyage,
} from './SeaMasterBillFields.test-utils';

afterEach(() => vi.restoreAllMocks());

describe('共同航程缺项与维护入口权限', () => {
  it.each([
    { read: true, maintain: true },
    { read: true, maintain: false },
    { read: false, maintain: true },
  ])(
    '缺ETA时列明保存阻断，按读写权限给出入口（%j）',
    async ({ read, maintain }) => {
      vi.spyOn(
        orderService,
        'orderServiceMatchSeaMasterBillCandidate',
      ).mockResolvedValue(
        candidateResponse({
          transportExecutions: [{ ...voyage, eta: undefined }],
        }),
      );
      renderVoyageForm(
        {},
        {
          user: {
            id: 'user-1',
            currentOrganization: {
              id: 'org-1',
              kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
            },
            permissionCapabilities: [
              ...(read
                ? [{ key: 'business.order.se.read', dataScope: 'organization' }]
                : []),
              ...(maintain
                ? [
                    {
                      key: 'business.order.se.reassign',
                      dataScope: 'organization',
                    },
                  ]
                : []),
            ],
          },
        },
      );
      expect(await screen.findByText('共同航程尚缺：ETA')).toBeVisible();
      expect(
        screen.getByText(
          'ETD、ETA 为开单必填，须先补齐共同航程后才能保存新票。',
        ),
      ).toBeVisible();
      if (read && maintain) {
        expect(
          screen.getByRole('link', { name: '前往已有订单维护共同航程' }),
        ).toHaveAttribute('href', '/orders/sea-export/order-a');
      } else {
        expect(
          screen.queryByRole('link', { name: '前往已有订单维护共同航程' }),
        ).toBeNull();
        expect(
          screen.getByText(
            maintain
              ? /当前账号无已有订单查看权限/
              : /当前账号无共同航程维护权限/,
          ),
        ).toBeVisible();
      }
      expect(Boolean(screen.queryByRole('link', { name: 'SE-A' }))).toBe(read);
    },
  );

  it('缺船名航次仅提示资料不齐，不显示日期缺失的保存阻断', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(
      candidateResponse({
        transportExecutions: [
          { ...voyage, vesselName: undefined, voyageNo: undefined },
        ],
      }),
    );
    const { form } = renderVoyageForm();
    await screen.findByText('共同航程尚缺：船名、航次');
    expect(
      screen.getByText('以上为资料提示，不单独阻止保存；ETD、ETA 仍为必填。'),
    ).toBeVisible();
    await expect(form.validateFields(['etd', 'eta'])).resolves.toBeDefined();
  });
});
