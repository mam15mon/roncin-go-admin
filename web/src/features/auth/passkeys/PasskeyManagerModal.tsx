import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Empty,
  Flex,
  Modal,
  Space,
  Spin,
  Tooltip,
} from 'antd';
import { useEffect, useRef, useState } from 'react';
import {
  passkeyServiceDeletePasskey,
  passkeyServiceListPasskeys,
} from '@/services/roncin/passkeyService';
import { isWriteResponse } from '@/utils/api';
import { formatDate } from '@/utils/format';
import {
  PasskeyFlowError,
  passkeyConfigOptions,
  passkeyErrorMessage,
  passkeyKeys,
  passkeyUnavailableReason,
} from './api';
import { PasskeyRegistrationModal } from './PasskeyRegistrationModal';

type Props = { open: boolean; userId: string; onClose: () => void };

export function PasskeyManagerModal({ open, userId, onClose }: Props) {
  const { modal, message } = App.useApp();
  const mounted = useRef(true);
  const confirmation = useRef<ReturnType<typeof modal.confirm> | null>(null);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      confirmation.current?.destroy();
      confirmation.current = null;
    };
  }, []);
  const queryClient = useQueryClient();
  const [adding, setAdding] = useState(false);
  const config = useQuery({ ...passkeyConfigOptions, enabled: open });
  const reason = config.isError
    ? '无法获取通行密钥配置，请稍后重试'
    : passkeyUnavailableReason(config.data);
  const canManage =
    config.data?.enabled === true &&
    config.data.origins.includes(window.location.origin);
  const list = useQuery({
    queryKey: passkeyKeys.credentials(userId),
    enabled: open && canManage,
    queryFn: async () => {
      const response = await passkeyServiceListPasskeys({
        skipErrorHandler: true,
      });
      if (!response?.data || response.success === false)
        throw new PasskeyFlowError('通行密钥列表获取失败');
      const items = response.data.items ?? [];
      if (items.some((item) => !item.id))
        throw new PasskeyFlowError('通行密钥列表不完整');
      return items;
    },
    meta: { silent: true },
  });
  const deletion = useMutation({
    mutationFn: async (id: string) => {
      const response = await passkeyServiceDeletePasskey(
        { id },
        { skipErrorHandler: true },
      );
      if (!isWriteResponse(response))
        throw new PasskeyFlowError('通行密钥删除失败');
    },
    onSuccess: () => {
      if (!mounted.current) return;
      void queryClient.invalidateQueries({
        queryKey: passkeyKeys.credentials(userId),
      });
      message.success('通行密钥已删除');
    },
    onError: (error) => {
      if (mounted.current)
        message.error(passkeyErrorMessage(error, '通行密钥删除失败，请重试'));
    },
    meta: { silent: true },
  });
  return (
    <>
      <Modal
        title="通行密钥"
        open={open}
        onCancel={onClose}
        footer={null}
        destroyOnHidden
        width={520}
      >
        <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <div>
            使用指纹、面容、屏锁或安全密钥登录，只有你能管理自己的通行密钥。
          </div>
          {reason && <Alert type="info" showIcon title={reason} />}
          {list.isError && (
            <Alert
              type="error"
              showIcon
              title={passkeyErrorMessage(list.error, '通行密钥列表获取失败')}
            />
          )}
          <Tooltip title={reason || undefined}>
            <Button
              icon={<PlusOutlined />}
              disabled={Boolean(reason)}
              onClick={() => setAdding(true)}
            >
              添加通行密钥
            </Button>
          </Tooltip>
          {canManage && (
            <Spin spinning={list.isLoading}>
              {!list.isLoading && !list.data?.length ? (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="尚未添加通行密钥"
                />
              ) : (
                <ul
                  aria-label="我的通行密钥"
                  style={{ listStyle: 'none', margin: 0, padding: 0 }}
                >
                  {(list.data ?? []).map((credential) => (
                    <li
                      key={credential.id}
                      style={{
                        padding: '12px 0',
                        borderBottom: '1px solid #f0f0f0',
                      }}
                    >
                      <Flex justify="space-between" align="center" gap={16}>
                        <div>
                          <div style={{ marginBottom: 4, fontWeight: 500 }}>
                            {credential.name}
                          </div>
                          <div style={{ color: '#64748b', fontSize: 12 }}>
                            添加时间：
                            {formatDate(credential.createdAt, 'datetime')}
                            <br />
                            上次使用：
                            {formatDate(credential.lastUsedAt, 'datetime')}
                          </div>
                        </div>
                        <Button
                          type="text"
                          danger
                          icon={<DeleteOutlined />}
                          aria-label={`删除${credential.name ?? '通行密钥'}`}
                          disabled={deletion.isPending}
                          onClick={() => {
                            confirmation.current = modal.confirm({
                              title: '删除通行密钥',
                              content: `删除“${credential.name ?? '通行密钥'}”后，将无法再使用它登录。`,
                              okText: '删除',
                              cancelText: '取消',
                              onCancel: () => {
                                confirmation.current = null;
                              },
                              okButtonProps: { danger: true },
                              onOk: () =>
                                deletion.mutateAsync(credential.id ?? ''),
                            });
                            // 使用 App modal 的 await 模式，由 mutation 反馈错误并保留确认框。
                            void confirmation.current.then(
                              () => undefined,
                              () => undefined,
                            );
                          }}
                        >
                          删除
                        </Button>
                      </Flex>
                    </li>
                  ))}
                </ul>
              )}
            </Spin>
          )}
        </Space>
      </Modal>
      {adding && (
        <PasskeyRegistrationModal
          open
          userId={userId}
          onRegistered={() => setAdding(false)}
          onSkip={() => setAdding(false)}
          skipText="取消"
        />
      )}
    </>
  );
}
