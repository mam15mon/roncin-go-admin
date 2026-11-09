import type { SelectProps } from 'antd';

// 分页只需选择页容量；使用按钮承接焦点，避免文本框被密码填充插件识别。
// 页面显式开启页容量选择时也复用此配置；true 会覆盖 ConfigProvider 的全局配置。
export const paginationSizeChanger: SelectProps = {
  showSearch: false,
  components: { input: <button type="button" /> },
};
