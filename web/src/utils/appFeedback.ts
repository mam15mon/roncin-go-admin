import { App } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import type { NotificationInstance } from 'antd/es/notification/interface';
import type { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useEffect } from 'react';

let messageInstance: MessageInstance | null = null;
let notificationInstance: NotificationInstance | null = null;
let modalInstance: Omit<ModalStaticFunctions, 'warn'> | null = null;

export const setAppFeedback = (app: {
  message: MessageInstance;
  notification: NotificationInstance;
  modal: Omit<ModalStaticFunctions, 'warn'>;
}) => {
  messageInstance = app.message;
  notificationInstance = app.notification;
  modalInstance = app.modal;
};

export const AppFeedbackBridge: React.FC = () => {
  const app = App.useApp();

  useEffect(() => {
    setAppFeedback(app);
  }, [app]);

  return null;
};

export const showErrorMessage = (content: string) => {
  if (messageInstance) {
    messageInstance.error(content);
  }
};

export const showErrorNotification = (args: {
  title: string;
  description?: string;
}) => {
  if (notificationInstance) {
    notificationInstance.error({
      title: args.title,
      description: args.description,
    });
  }
};

export const getAppFeedback = () => ({
  message: messageInstance,
  notification: notificationInstance,
  modal: modalInstance,
});
