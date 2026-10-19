package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type notificationRepoStub struct {
	delivery *NotificationDelivery
	err      error
}

func (s *notificationRepoStub) FindByTaskID(context.Context, uuid.UUID) (*NotificationDelivery, error) {
	return s.delivery, s.err
}

type dingTalkNotificationSenderStub struct {
	enabled bool
	userID  string
	content string
	message DingTalkMarkdownMessage
	err     error
}

func (s *dingTalkNotificationSenderStub) Enabled() bool { return s.enabled }

func (s *dingTalkNotificationSenderStub) SendMarkdown(_ context.Context, userID string, message DingTalkMarkdownMessage) error {
	s.userID = userID
	s.content = message.Text
	s.message = message
	return s.err
}

func TestNotificationProcessNextCompletesDingTalkTask(t *testing.T) {
	organizationID := uuid.New()
	taskID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: taskID, OrganizationID: organizationID, Kind: BackgroundTaskKindDingTalkNotice,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken,
	}}
	sender := &dingTalkNotificationSenderStub{enabled: true}
	usecase := NewNotificationUsecase(NewBackgroundTaskUsecase(taskRepo), &notificationRepoStub{delivery: &NotificationDelivery{
		RecipientUserID: uuid.New(), DingTalkUserID: "ding-user-id", Channel: NotificationChannelDingTalk,
		Template: NotificationTemplateOrderPersonnelAssign, ResourceType: "ORDER", ResourceID: uuid.New(),
		ReferenceCode: "SE202608280001", Parameter: string(OrderPersonnelRoleDocument),
	}}, sender)

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if sender.userID != "ding-user-id" || !strings.Contains(sender.content, "SE202608280001") || !strings.Contains(sender.content, "单证人员") {
		t.Fatalf("发送内容错误: userID=%q content=%q", sender.userID, sender.content)
	}
	if taskRepo.completeOrgID != organizationID || taskRepo.completeID != taskID || taskRepo.completeLeaseToken != leaseToken {
		t.Fatalf("任务完成参数错误: %#v", taskRepo)
	}
}

func TestNotificationProcessNextSendsUserAuthorizedMessage(t *testing.T) {
	organizationID := uuid.New()
	taskID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: taskID, OrganizationID: organizationID, Kind: BackgroundTaskKindDingTalkNotice,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken,
	}}
	sender := &dingTalkNotificationSenderStub{enabled: true}
	usecase := NewNotificationUsecase(NewBackgroundTaskUsecase(taskRepo), &notificationRepoStub{delivery: &NotificationDelivery{
		RecipientUserID: uuid.New(), RecipientDisplayName: "张冠楠", DingTalkUserID: "ding-user-id", Channel: NotificationChannelDingTalk,
		Template: NotificationTemplateUserAuthorized, ResourceType: "USER", ResourceID: uuid.New(),
	}}, sender)

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if sender.userID != "ding-user-id" || !strings.Contains(sender.content, "张冠楠") || !strings.Contains(sender.content, "账号授权完成") || !strings.Contains(sender.content, "钉钉扫码登录") {
		t.Fatalf("发送内容错误: userID=%q content=%q", sender.userID, sender.content)
	}
	if taskRepo.completeID != taskID || taskRepo.completeLeaseToken != leaseToken {
		t.Fatalf("任务完成参数错误: %#v", taskRepo)
	}
}

func TestRenderOrderPersonnelNotificationRequiresReferenceCode(t *testing.T) {
	_, err := renderNotification(&NotificationDelivery{
		DingTalkUserID: "ding-user-id",
		Channel:        NotificationChannelDingTalk,
		Template:       NotificationTemplateOrderPersonnelAssign,
		ResourceType:   "ORDER",
		ResourceID:     uuid.New(),
		Parameter:      string(OrderPersonnelRoleOperator),
	})
	if err == nil {
		t.Fatal("订单人员通知缺少订单号时应失败")
	}
}

func TestNotificationProcessNextRecordsRetryOnSendFailure(t *testing.T) {
	organizationID := uuid.New()
	taskID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: taskID, OrganizationID: organizationID, Kind: BackgroundTaskKindDingTalkNotice,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken, Attempts: 1,
	}}
	sender := &dingTalkNotificationSenderStub{enabled: true, err: context.DeadlineExceeded}
	usecase := NewNotificationUsecase(NewBackgroundTaskUsecase(taskRepo), &notificationRepoStub{delivery: &NotificationDelivery{
		RecipientUserID: uuid.New(), DingTalkUserID: "ding-user-id", Channel: NotificationChannelDingTalk,
		Template: NotificationTemplateOrderPersonnelAssign, ResourceType: "ORDER", ResourceID: uuid.New(),
		ReferenceCode: "SE202608280002", Parameter: string(OrderPersonnelRoleOperator),
	}}, sender)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	usecase.now = func() time.Time { return now }

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err == nil {
		t.Fatal("ProcessNext() expected send error")
	}
	if taskRepo.failID != taskID || taskRepo.failLeaseToken != leaseToken || !strings.Contains(taskRepo.failErrorMessage, "context deadline exceeded") {
		t.Fatalf("任务失败参数错误: %#v", taskRepo)
	}
	if want := now.Add(time.Minute); !taskRepo.failNextRunAt.Equal(want) {
		t.Fatalf("next run = %s, want %s", taskRepo.failNextRunAt, want)
	}
}

var _ NotificationRepo = (*notificationRepoStub)(nil)
var _ DingTalkNotificationSender = (*dingTalkNotificationSenderStub)(nil)

func TestRenderAllBusinessNotificationsAsMarkdown(t *testing.T) {
	for _, scenario := range []struct {
		template, resourceType, reference, parameter, recipient, title string
		fields                                                         []string
	}{
		{NotificationTemplateOrderPersonnelAssign, "ORDER", "SE202610190001", string(OrderPersonnelRoleDocument), "", "海运出口订单协作提醒", []string{"**订单：** SE202610190001", "**您已被分配为：** 单证人员", "请登录 Roncin 系统查看并处理。"}},
		{NotificationTemplateUserAuthorized, "USER", "", "", "张三", "Roncin 账号授权完成", []string{"**张三**", "所属组织和角色已完成授权", "**钉钉扫码登录**"}},
		{NotificationTemplateExchangeRateWeeklyReminder, "ORGANIZATION", "深圳公司", "CNY 2026-10-19~2026-10-25", "", "周汇率同步督办", []string{"**组织：** 深圳公司", `CNY 2026\-10\-19\~2026\-10\-25`, "缺少本周配置的业务折算将被阻止", "请先在「财务 · 汇率」页维护汇率。"}},
		{"FEE_REVIEW_PENDING", "ORDER", "SE202610190001", "李四", "", "订单费用待审核", []string{"**订单号：** SE202610190001", "**提交人：** 李四", "整票费用已完成录入并提交审核", "查看费用及利润后处理"}},
		{"FEE_REVIEW_RESULT", "ORDER", "SE202610190001", "王五：已退回", "", "订单费用审核结果", []string{"**订单号：** SE202610190001", "**处理结果：** 王五：已退回", "请进入 Roncin 查看审批意见。"}},
		{NotificationTemplateFeeSupplementApprovalPending, "FEE_SUPPLEMENT_REQUEST", "SE202610190001", "应付 100.00000000 CNY", "", "费用补录待审批", []string{"**订单：** SE202610190001", `**补录摘要：** 应付 100\.00000000 CNY`, "您具备该订单的费用审核权限", "在订单费用页处理补录申请。"}},
		{NotificationTemplateCommissionDecreaseSuggested, "COMMISSION_ADJUSTMENT", "SE202610190001", "冲减 10.00000000 CNY", "", "提成冲减通知", []string{"**订单：** SE202610190001", `**冲减摘要：** 冲减 10\.00000000 CNY`, "已确认的冲减", "本消息供您知情，实际少发或扣回由财务登记", "查看本人专属详情。"}},
	} {
		t.Run(scenario.template, func(t *testing.T) {
			message, err := renderNotification(&NotificationDelivery{
				DingTalkUserID: "bound-user", Channel: NotificationChannelDingTalk, Template: scenario.template,
				ResourceType: scenario.resourceType, ResourceID: uuid.New(), ReferenceCode: scenario.reference,
				Parameter: scenario.parameter, RecipientDisplayName: scenario.recipient,
			})
			if err != nil {
				t.Fatal(err)
			}
			if message.Title != scenario.title || !strings.HasPrefix(message.Text, "### "+scenario.title+"\n\n") || message.OrderEntryPath != "" {
				t.Fatalf("消息标题或入口错误: %+v", message)
			}
			for _, field := range scenario.fields {
				if !strings.Contains(message.Text, field) {
					t.Fatalf("消息缺少 %q: %s", field, message.Text)
				}
			}
		})
	}
}

func TestNotificationDynamicFieldsRemainPlainText(t *testing.T) {
	const input = "张三\\`*_{}[]()#+-.!|~\r\n> 引用\n# 标题\t[链接](https://example.com) <b>内容</b> &lt;"
	const want = "张三\\\\\\`\\*\\_\\{\\}\\[\\]\\(\\)\\#\\+\\-\\.\\!\\|\\~ &gt; 引用 \\# 标题 \\[链接\\]\\(https://example\\.com\\) &lt;b&gt;内容&lt;/b&gt; &amp;lt;"
	if got := escapeNotificationMarkdown(input); got != want {
		t.Fatalf("纯文本转义错误:\ngot  %q\nwant %q", got, want)
	}
	for _, scenario := range []struct{ template, resourceType, parameter string }{
		{NotificationTemplateOrderPersonnelAssign, "ORDER", string(OrderPersonnelRoleOperator)},
		{NotificationTemplateUserAuthorized, "USER", input},
		{NotificationTemplateExchangeRateWeeklyReminder, "ORGANIZATION", input},
		{"FEE_REVIEW_PENDING", "ORDER", input},
		{"FEE_REVIEW_RESULT", "ORDER", input},
		{NotificationTemplateFeeSupplementApprovalPending, "FEE_SUPPLEMENT_REQUEST", input},
		{NotificationTemplateCommissionDecreaseSuggested, "COMMISSION_ADJUSTMENT", input},
	} {
		t.Run(scenario.template, func(t *testing.T) {
			message, err := renderNotification(&NotificationDelivery{DingTalkUserID: "user", Channel: NotificationChannelDingTalk, Template: scenario.template, ResourceType: scenario.resourceType, ResourceID: uuid.New(), ReferenceCode: input, Parameter: scenario.parameter, RecipientDisplayName: input})
			if err != nil || !strings.Contains(message.Text, want) || strings.Count(message.Text, "\n### ") > 0 || strings.Contains(message.Text, "\n# 标题") || strings.Contains(message.Text, "<b>") || message.OrderEntryPath != "" {
				t.Fatalf("动态字段改变消息布局: %+v %v", message, err)
			}
		})
	}
}

func TestFeeReviewMarkdownPreservesTrustedOrderEntry(t *testing.T) {
	id := uuid.New()
	path := "/orders/sea-export/" + id.String() + "/fees"
	for _, template := range []string{"FEE_REVIEW_PENDING", "FEE_REVIEW_RESULT"} {
		t.Run(template, func(t *testing.T) {
			for _, scenario := range []struct {
				name, suffix, wantPath string
			}{
				{"匹配资源", "\n订单入口：" + path, path},
				{"不匹配资源", "\n订单入口：/orders/sea-export/" + uuid.NewString() + "/fees", ""},
				{"伪造外链", "\n订单入口：[点我](https://example.com)", ""},
				{"没有入口", "", ""},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					message, err := renderNotification(&NotificationDelivery{DingTalkUserID: "user", Channel: NotificationChannelDingTalk, Template: template, ResourceType: "ORDER", ResourceID: id, ReferenceCode: "SE100", Parameter: "张三：已批准" + scenario.suffix})
					if err != nil || message.OrderEntryPath != scenario.wantPath || !strings.Contains(message.Text, "张三：已批准") {
						t.Fatalf("审核信息或入口丢失: %+v %v", message, err)
					}
					if scenario.wantPath != "" && strings.Contains(message.Text, "订单入口") {
						t.Fatal("入口不应重复出现在摘要中")
					}
					if scenario.wantPath == "" && scenario.suffix != "" && !strings.Contains(message.Text, escapeNotificationMarkdown(scenario.suffix)) {
						t.Fatal("非可信入口应保留为纯文本")
					}
				})
			}
		})
	}
}
