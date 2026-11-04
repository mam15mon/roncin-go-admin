import { expect, test } from '@playwright/test';

/** 组织接口用独立夹具，验证真实公司表单、详情和重新加载链路。 */
test('公司英文名称可创建、编辑、清空并回显，部门不显示该字段', async ({
  page,
}) => {
  const organizations: API.AdminOrganization[] = [
    {
      id: 'english-company',
      code: 'COMPANY',
      name: '公司英文名验收',
      nameEn: 'INITIAL TEST COMPANY LTD',
      kind: 2,
      enabled: true,
      baseCurrency: 'CNY',
    },
  ];
  const writes: API.UpdateOrganizationRequest[] = [];
  const unexpected: string[] = [];
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url());
    const send = (data: unknown) =>
      route.fulfill({ json: { success: true, code: 0, data } });
    if (url.pathname === '/api/v1/auth/me') {
      const permissions = [
        'system.platform.access',
        'system.organization.read',
        'system.organization.create',
        'system.organization.update',
      ];
      return send({
        id: 'english-admin',
        username: 'english-admin',
        displayName: '公司英文名验收',
        currentOrganization: {
          id: 'english-system',
          code: 'SYS',
          name: '系统管理',
          kind: 1,
        },
        permissions,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
      });
    }
    if (url.pathname === '/api/v1/reference/currencies')
      return send([{ id: 'cny', code: 'CNY', name: '人民币', enabled: true }]);
    if (url.pathname === '/api/v1/admin/organizations') {
      if (route.request().method() === 'POST') {
        const body = route.request().postDataJSON();
        expect(body.nameEn).toBe('CREATED TEST COMPANY LTD');
        const created = { ...body, id: 'created-company', enabled: true };
        organizations.push(created);
        return send(created);
      }
      return send(organizations);
    }
    if (
      url.pathname === '/api/v1/admin/organizations/created-company' &&
      route.request().method() === 'PUT'
    ) {
      const body: API.UpdateOrganizationRequest = route
        .request()
        .postDataJSON();
      writes.push(body);
      const index = organizations.findIndex((item) => item.id === body.id);
      expect(index).toBeGreaterThanOrEqual(0);
      organizations[index] = { ...organizations[index], ...body };
      return send(organizations[index]);
    }
    unexpected.push(url.pathname);
    return route.fulfill({ status: 404, json: { success: false } });
  });
  await page.goto('/admin?tab=organizations');
  await page.getByText('树表列表', { exact: true }).click();
  await page.getByRole('button', { name: '新增公司', exact: true }).click();
  const create = page.getByRole('dialog', { name: '新增公司', exact: true });
  await create.getByLabel('组织编码', { exact: true }).fill('CREATED');
  await create
    .getByLabel('组织名称', { exact: true })
    .fill('新建英文名验收公司');
  await create
    .getByLabel('公司英文名称', { exact: true })
    .fill('  CREATED TEST COMPANY LTD  ');
  await create.getByRole('button', { name: /确.*(认|定)/ }).click();
  await expect(create).not.toBeVisible();
  await expect(
    page.getByText('CREATED TEST COMPANY LTD', { exact: true }),
  ).toBeVisible();

  await page.getByRole('button', { name: /编辑组织/ }).click();
  const edit = page.getByRole('dialog', {
    name: /编辑组织：新建英文名验收公司/,
  });
  await expect(edit.getByLabel('公司英文名称', { exact: true })).toHaveValue(
    'CREATED TEST COMPANY LTD',
  );
  await edit
    .getByLabel('公司英文名称', { exact: true })
    .fill('UPDATED TEST COMPANY LTD');
  await edit.getByRole('button', { name: /确.*(认|定)/ }).click();
  await expect(edit).not.toBeVisible();
  await expect(
    page.getByText('UPDATED TEST COMPANY LTD', { exact: true }),
  ).toBeVisible();
  expect(writes.at(-1)?.nameEn).toBe('UPDATED TEST COMPANY LTD');

  await page.getByRole('button', { name: /编辑组织/ }).click();
  await edit.getByLabel('公司英文名称', { exact: true }).clear();
  await edit.getByRole('button', { name: /确.*(认|定)/ }).click();
  await expect(edit).not.toBeVisible();
  expect(writes.at(-1)?.nameEn).toBe('');
  await page.getByRole('button', { name: /编辑组织/ }).click();
  await expect(edit.getByLabel('公司英文名称', { exact: true })).toHaveValue(
    '',
  );
  await edit.getByRole('button', { name: /取.*消/ }).click();

  await page.getByRole('button', { name: /新增部门/ }).click();
  const department = page.getByRole('dialog', { name: /新增部门/ });
  await expect(department).toBeVisible();
  await expect(
    department.getByLabel('公司英文名称', { exact: true }),
  ).toHaveCount(0);
  await department.getByRole('button', { name: /取.*消/ }).click();
  expect(errors).toEqual([]);
  expect(unexpected).toEqual([]);
});
