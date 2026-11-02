import { expect, test } from '@playwright/test';

test('原生通行密钥注册、免密登录、一次性挑战及本人删除形成完整闭环', async ({
  page,
  context,
  baseURL,
}) => {
  test.skip(
    process.env.RONCIN_PASSKEY_ACCEPTANCE !== '1',
    '需要独立验收库、启用 localhost 通行密钥的服务及专用测试账号',
  );
  test.setTimeout(120_000);
  if (
    !process.env.RONCIN_PASSKEY_TEST_DATABASE?.startsWith(
      'roncin_passkey_verify_',
    )
  )
    throw new Error('必须显式绑定独立通行密钥验收数据库');
  const apiURL = process.env.RONCIN_PASSKEY_API_URL;
  const username = process.env.BOOTSTRAP_ADMIN_USERNAME;
  const password = process.env.BOOTSTRAP_ADMIN_PASSWORD;
  if (!apiURL || !baseURL || !username || !password)
    throw new Error('缺少通行密钥验收的 API、页面地址或专用账号配置');
  const origin = new URL(baseURL).origin;
  if (new URL(origin).hostname !== 'localhost')
    throw new Error(
      '此验收仅使用安全上下文 localhost，不能指向日常开发或生产域名',
    );
  let registered = 0;
  let finishCredentialJSON = '';
  let passwordUserID = '';
  let passkeyUserID = '';
  await page.route('**/api/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const response = await route.fetch({
      url: `${apiURL}${url.pathname}${url.search}`,
      headers: { ...request.headers(), host: new URL(origin).host },
    });
    if (url.pathname === '/api/v1/auth/login' && response.ok()) {
      passwordUserID = (await response.json()).data?.id ?? '';
    }
    if (
      url.pathname === '/api/v1/auth/passkey/registration/finish' &&
      response.ok()
    )
      registered += 1;
    if (url.pathname === '/api/v1/auth/passkey/login/finish' && response.ok()) {
      passkeyUserID = (await response.json()).data?.id ?? '';
      finishCredentialJSON = request.postDataJSON().credentialJson;
    }
    await route.fulfill({ response });
  });

  const cdp = await context.newCDPSession(page);
  await cdp.send('WebAuthn.enable', { enableUI: false });
  const { authenticatorId } = await cdp.send(
    'WebAuthn.addVirtualAuthenticator',
    {
      options: {
        protocol: 'ctap2',
        transport: 'internal',
        hasResidentKey: true,
        hasUserVerification: true,
        isUserVerified: true,
        automaticPresenceSimulation: true,
      },
    },
  );

  async function enterWorkspace() {
    const picker = page.getByText('选择进入的组织', { exact: true });
    await expect
      .poll(
        async () =>
          (await picker.isVisible()) || !page.url().includes('/user/login'),
      )
      .toBe(true);
    if (await picker.isVisible())
      await page.getByRole('button', { name: /^进入/ }).first().click();
    await expect(page).not.toHaveURL(/\/user\/login/);
  }

  try {
    await page.goto('/');
    await page.getByText('账号登录', { exact: true }).click();
    await page.getByPlaceholder('钉钉手机号 / 本地账号').fill(username);
    await page.getByRole('button', { name: '下一步', exact: true }).click();
    await page.getByPlaceholder('请输入密码').fill(password);
    await page
      .getByRole('checkbox', { name: '自动登录', exact: true })
      .uncheck();
    const enable = page.getByRole('checkbox', {
      name: '开启免密登录(通行密钥)',
      exact: true,
    });
    await expect(enable).toBeEnabled();
    await enable.check();
    await page.getByRole('button', { name: '登录', exact: true }).click();
    const registration = page.getByRole('dialog', {
      name: '开启免密登录（通行密钥）',
    });
    await expect(registration).toBeVisible();
    const add = registration.getByRole('button', {
      name: '添加通行密钥',
      exact: true,
    });
    await expect(add).toBeEnabled();
    await add.click();
    await expect(registration).not.toBeVisible();
    await enterWorkspace();
    expect(registered).toBe(1);
    expect(passwordUserID).not.toBe('');
    const sessionCookies = (await context.cookies(origin)).filter(
      (cookie) =>
        !cookie.name.includes('_passkey_') &&
        cookie.name.startsWith('roncin_pk_accept_'),
    );
    expect(sessionCookies).toHaveLength(2);
    expect(sessionCookies.every((cookie) => cookie.expires === -1)).toBe(true);

    const logoutStatus = await page.evaluate(async () => {
      const response = await fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      });
      return response.status;
    });
    expect(logoutStatus).toBe(200);
    await page.goto('/');
    const login = page.getByRole('button', { name: '免密登录', exact: true });
    await expect(login).toBeEnabled();
    await login.click();
    await enterWorkspace();
    expect(passkeyUserID).toBe(passwordUserID);
    expect(registered).toBe(1);
    expect(finishCredentialJSON).not.toBe('');

    // 同一浏览器响应再次提交也必须被一次性 challenge 拒绝，不能新建第二个会话。
    const replayStatus = await page.evaluate(async (credentialJson) => {
      const response = await fetch('/api/v1/auth/passkey/login/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ credentialJson, rememberMe: false }),
      });
      return response.status;
    }, finishCredentialJSON);
    expect(replayStatus).toBe(401);

    const managed = await page.evaluate(async () => {
      const before = await (await fetch('/api/v1/auth/passkeys')).json();
      const items = before.data?.items ?? [];
      if (items.length !== 1)
        return { before: items.length, deleted: 0, after: -1 };
      const deletion = await fetch(
        `/api/v1/auth/passkeys/${encodeURIComponent(items[0].id)}`,
        {
          method: 'DELETE',
        },
      );
      const after = await (await fetch('/api/v1/auth/passkeys')).json();
      return {
        before: items.length,
        deleted: deletion.status,
        after: after.data?.items?.length ?? 0,
      };
    });
    expect(managed).toEqual({ before: 1, deleted: 200, after: 0 });
  } finally {
    await cdp.send('WebAuthn.removeVirtualAuthenticator', { authenticatorId });
    await cdp.send('WebAuthn.disable');
  }
});
