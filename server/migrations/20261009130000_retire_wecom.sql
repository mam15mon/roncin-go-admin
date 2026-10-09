-- 企业微信功能下线，只移除专属身份字段，保留用户及全部业务关联。
DROP INDEX "users_wecom_userid_key";
ALTER TABLE "users"
    DROP COLUMN "wecom_userid",
    DROP COLUMN "wecom_name";

-- 迁移命令随后按 display_name、dingtalk_name 重建拼音检索键，清除已删除姓名来源。
UPDATE "users" SET "search_keywords" = '';
