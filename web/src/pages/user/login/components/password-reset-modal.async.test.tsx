import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  challenge,
  confirmPassword,
  fillPassword,
  readyChallenge,
  renderModal,
  requestCode,
  sendCode,
  submitPassword,
} from './password-reset-modal.test-utils';

beforeEach(() => vi.resetAllMocks());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('密码找回异步与敏感字段隔离', () => {
  it('手机号更换后丢弃迟到成功且清空验证码和密码，新号必须重新申请', async () => {
    let finish!: (value: ReturnType<typeof challenge>) => void;
    requestCode.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    renderModal();
    await sendCode();
    fillPassword();
    fireEvent.change(screen.getByLabelText('钉钉手机号'), {
      target: { value: '13900139000' },
    });
    await act(async () => finish(challenge('old-phone-token')));
    expect(screen.getByLabelText('新密码')).toHaveValue('');
    expect(screen.getByRole('textbox', { name: '钉钉验证码' })).toHaveValue('');
    expect(screen.queryByText(/请前往钉钉查看/)).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
    await readyChallenge();
    fillPassword();
    confirmPassword.mockResolvedValue({ success: true });
    submitPassword();
    await waitFor(() =>
      expect(confirmPassword).toHaveBeenCalledWith(
        expect.objectContaining({ challengeToken: 'challenge-one' }),
        expect.anything(),
      ),
    );
    expect(requestCode).toHaveBeenLastCalledWith(
      { mobile: '13900139000' },
      expect.anything(),
    );
  });

  it('手机号变更后迟到失败不污染新号提示', async () => {
    let reject!: (reason: unknown) => void;
    requestCode.mockReturnValueOnce(
      new Promise((_, no) => {
        reject = no;
      }),
    );
    renderModal();
    await sendCode();
    fireEvent.change(screen.getByLabelText('钉钉手机号'), {
      target: { value: '13900139000' },
    });
    await act(async () =>
      reject({ response: { data: { message: '旧号发送失败' } } }),
    );
    expect(screen.queryByText('旧号发送失败')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送验证码' })).toBeEnabled();
  });

  it('关闭后在途申请结果不回填，重开不保留密码、验证码和挑战', async () => {
    let finish!: (value: ReturnType<typeof challenge>) => void;
    requestCode.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const first = renderModal();
    await sendCode();
    await act(async () => fillPassword());
    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
    expect(first.onClose).toHaveBeenCalled();
    first.unmount();
    renderModal();
    await act(async () => finish(challenge('closed-token')));
    expect(screen.getByLabelText('新密码')).toHaveValue('');
    expect(screen.getByRole('textbox', { name: '钉钉验证码' })).toHaveValue('');
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
    expect(screen.getByRole('button', { name: '发送验证码' })).toBeEnabled();
  });

  it('确认期间由ModalForm显示提交状态并阻止重复提交和关闭', async () => {
    let finish!: (value: { success: boolean }) => void;
    confirmPassword.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const { onSuccess, onClose } = renderModal();
    await readyChallenge();
    fillPassword();
    submitPassword();
    await waitFor(() => expect(confirmPassword).toHaveBeenCalledOnce());
    submitPassword();
    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
    expect(onClose).not.toHaveBeenCalled();
    expect(confirmPassword).toHaveBeenCalledOnce();
    await act(async () => finish({ success: true }));
    expect(onSuccess).toHaveBeenCalledOnce();
  });
  it('确认在途锁定手机号，目标发生变化时丢弃旧确认成功', async () => {
    let finish!: (value: { success: boolean }) => void;
    confirmPassword.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const { onSuccess } = renderModal();
    await readyChallenge();
    fillPassword();
    submitPassword();
    await waitFor(() => expect(confirmPassword).toHaveBeenCalledOnce());
    expect(screen.getByLabelText('钉钉手机号')).toBeDisabled();
    expect(screen.getByLabelText('新密码')).toBeDisabled();
    // 模拟外部表单写入，验证禁用控件之外仍有目标/代次校验。
    fireEvent.change(screen.getByLabelText('钉钉手机号'), {
      target: { value: '13900139000' },
    });
    await act(async () => finish({ success: true }));
    expect(onSuccess).not.toHaveBeenCalled();
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
  });

  it('确认中卸载后迟到失败不污染重新打开的表单', async () => {
    let reject!: (reason: unknown) => void;
    confirmPassword.mockReturnValueOnce(
      new Promise((_, no) => {
        reject = no;
      }),
    );
    const first = renderModal();
    await readyChallenge();
    fillPassword();
    submitPassword();
    await waitFor(() => expect(confirmPassword).toHaveBeenCalledOnce());
    first.unmount();
    renderModal('13900139000');
    await act(async () => reject({ data: { message: '旧确认失败' } }));
    expect(screen.queryByText('旧确认失败')).not.toBeInTheDocument();
    expect(screen.getByLabelText('钉钉手机号')).toHaveValue('13900139000');
    expect(first.onSuccess).not.toHaveBeenCalled();
  });
  it('确认中卸载后迟到成功不关闭新表单或触发旧成功回调', async () => {
    let finish!: (value: { success: boolean }) => void;
    confirmPassword.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const first = renderModal();
    await readyChallenge();
    fillPassword();
    submitPassword();
    await waitFor(() => expect(confirmPassword).toHaveBeenCalledOnce());
    first.unmount();
    renderModal('13900139000');
    await act(async () => finish({ success: true }));
    expect(first.onSuccess).not.toHaveBeenCalled();
    expect(screen.getByLabelText('钉钉手机号')).toHaveValue('13900139000');
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
  });

  it('同一轮连续点击发送只发一次请求，验证期间关闭不再发出申请', async () => {
    requestCode.mockReturnValue(new Promise(() => {}));
    const first = renderModal();
    const button = screen.getByRole('button', { name: '发送验证码' });
    fireEvent.click(button);
    fireEvent.click(button);
    await waitFor(() => expect(requestCode).toHaveBeenCalledOnce());
    first.unmount();
    requestCode.mockClear();
    const second = renderModal();
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
      fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
    });
    expect(second.onClose).toHaveBeenCalled();
    expect(requestCode).not.toHaveBeenCalled();
  });
});
