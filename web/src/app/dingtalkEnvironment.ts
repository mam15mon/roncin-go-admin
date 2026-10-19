/** UA 仅用于选择登录体验，身份与企业归属由服务端换码校验。 */
export function isDingTalkClient(userAgent: string): boolean {
  return /dingtalk/i.test(userAgent);
}
