import { RightOutlined } from '@ant-design/icons';
import { App, Spin, Tag } from 'antd';
import clsx from 'clsx';
import React, { useState } from 'react';
import { authServiceSwitchOrganization } from '@/services/roncin/authService';
import styles from './organization-picker.module.less';

/** 登录后组织选择视图使用的候选组织（与登录响应 OrganizationChoice 同构）。 */
export interface LoginOrganizationOption {
  organizationId: string;
  organizationName: string;
  organizationCode: string;
  isDefault: boolean;
}

/**
 * 解析登录后的候选组织列表。
 *
 * 优先使用登录响应的 organizationChoices（服务端已按默认组织置首并标记）；
 * 钉钉/企微登录响应不携带候选列表，回退到 principal organizations，
 * 并把当前会话组织视为默认组织（未显式选择时的进入组织）。
 */
export function resolveLoginOrganizationOptions(
  organizationChoices: API.OrganizationChoice[] | undefined,
  currentUser?: API.CurrentUser,
): LoginOrganizationOption[] {
  if (organizationChoices && organizationChoices.length > 0) {
    return organizationChoices
      .filter((choice) => choice.organizationId)
      .map((choice) => ({
        organizationId: choice.organizationId,
        organizationName: choice.organizationName,
        organizationCode: choice.organizationCode,
        isDefault: Boolean(choice.isDefault),
      }));
  }
  const currentOrganizationId = currentUser?.currentOrganization?.id;
  return (currentUser?.organizations ?? [])
    .filter((organization) => organization.id)
    .map((organization) => ({
      organizationId: organization.id as string,
      organizationName: organization.name ?? '',
      organizationCode: organization.code ?? '',
      isDefault: organization.id === currentOrganizationId,
    }));
}

interface OrganizationPickerProps {
  /** 候选组织（调用方保证多于 1 个才渲染本组件）。 */
  options: LoginOrganizationOption[];
  /** 进入所选组织后完成登录导航。 */
  onEnter: () => void;
}

/**
 * 登录后的组织选择视图。
 *
 * 登录会话已按默认组织建立：选默认组织直接进入；选其他组织时经
 * switch-organization 换发会话（新凭据由服务端下发），成功后才进入。
 */
export function OrganizationPicker({
  options,
  onEnter,
}: OrganizationPickerProps) {
  const { message } = App.useApp();
  const [switchingOrgId, setSwitchingOrgId] = useState<string>();

  const handleSelect = async (option: LoginOrganizationOption) => {
    if (switchingOrgId) return;
    if (option.isDefault) {
      onEnter();
      return;
    }
    setSwitchingOrgId(option.organizationId);
    try {
      const response = await authServiceSwitchOrganization(
        { organizationId: option.organizationId },
        { skipErrorHandler: true },
      );
      if (!response.data) {
        message.error('进入所选组织失败，请稍后重试');
        return;
      }
      onEnter();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : '进入所选组织失败，请稍后重试',
      );
    } finally {
      setSwitchingOrgId(undefined);
    }
  };

  return (
    <div className={styles.picker}>
      <h1 className={styles.title}>选择进入的组织</h1>
      <p className={styles.subtitle}>
        你的账号属于多个组织，请选择本次进入的组织
      </p>
      <div className={styles.grid}>
        {options.map((option) => {
          const switching = switchingOrgId === option.organizationId;
          return (
            <button
              key={option.organizationId}
              type="button"
              className={clsx(
                styles.orgCard,
                option.isDefault && styles.orgCardDefault,
              )}
              disabled={Boolean(switchingOrgId)}
              aria-label={`进入${option.organizationName}`}
              title={option.organizationName}
              onClick={() => {
                void handleSelect(option);
              }}
            >
              <div className={styles.orgCardBody}>
                <div className={styles.orgCardHeader}>
                  <span className={styles.orgName}>
                    {option.organizationName}
                  </span>
                  {option.isDefault && (
                    <Tag color="blue" className={styles.defaultTag}>
                      默认
                    </Tag>
                  )}
                </div>
                <span className={styles.orgCode}>
                  {option.organizationCode}
                </span>
              </div>
              <div className={styles.orgCardAction}>
                {switching ? (
                  <Spin size="small" className={styles.cardSpin} />
                ) : (
                  <RightOutlined className={styles.enterIcon} />
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
