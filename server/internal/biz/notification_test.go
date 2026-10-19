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
	message DingTalkMessage
	err     error
}

func (s *dingTalkNotificationSenderStub) Enabled() bool { return s.enabled }

func (s *dingTalkNotificationSenderStub) SendMessage(_ context.Context, userID string, message DingTalkMessage) error {
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
	if sender.userID != "ding-user-id" || !strings.Contains(sender.content, "张冠楠") || !strings.Contains(sender.content, "账号授权完成") || sender.message.ActionTitle != "进入工作台" || sender.message.ActionPath != "/welcome" {
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
	if taskRepo.failID != taskID || taskRepo.failLeaseToken != leaseToken || !strings.Contains(taskRepo.failErrorMessage, "context deadline exceeded") || taskRepo.completeID != uuid.Nil {
		t.Fatalf("任务失败参数错误: %#v", taskRepo)
	}
	if want := now.Add(time.Minute); !taskRepo.failNextRunAt.Equal(want) {
		t.Fatalf("next run = %s, want %s", taskRepo.failNextRunAt, want)
	}
}

var _ NotificationRepo = (*notificationRepoStub)(nil)
var _ DingTalkNotificationSender = (*dingTalkNotificationSenderStub)(nil)

func TestRenderAllBusinessNotificationsAsCompactMessages(t *testing.T) {
	id := uuid.New()
	feePath := "/orders/sea-export/" + id.String() + "/fees"
	for _, scenario := range []struct {
		template, resourceType, reference, parameter, recipient, title, status, actionTitle, actionPath string
		fields                                                                                          []string
	}{
		{NotificationTemplateOrderPersonnelAssign, "ORDER", "SE202610190001", string(OrderPersonnelRoleDocument), "", "订单协作提醒", "🔵 新分配", "查看订单列表", "/orders", []string{"**SE202610190001**", "你已被分配为该订单的单证人员。"}},
		{NotificationTemplateUserAuthorized, "USER", "", "", "张三", "账号授权完成", "🟢 已开通", "进入工作台", "/welcome", []string{"**张三**", "你的组织和角色已完成授权", "进入 Roncin 工作台"}},
		{NotificationTemplateExchangeRateWeeklyReminder, "ORGANIZATION", "深圳公司", "CNY 2026-10-19~2026-10-25", "", "周汇率待维护", "🟠 待处理", "维护汇率", "/finance/exchange-rates", []string{"**深圳公司**", `CNY 2026\-10\-19\~2026\-10\-25`, "缺少配置将阻止业务折算"}},
		{"FEE_REVIEW_PENDING", "ORDER", "SE202610190001", "李四\n订单入口：" + feePath, "", "订单费用待审核", "🟠 待审核", "查看订单费用", feePath, []string{"**SE202610190001**", "李四已提交整票费用", "核对费用及利润"}},
		{"FEE_REVIEW_RESULT", "ORDER", "SE202610190001", "王五：已退回\n订单入口：" + feePath, "", "订单费用审核结果", "🔴 已退回", "查看订单费用", feePath, []string{"**SE202610190001**", "王五：已退回", "请查看审批意见"}},
		{NotificationTemplateFeeSupplementApprovalPending, "FEE_SUPPLEMENT_REQUEST", "SE202610190001", "应付 100.00000000 CNY", "", "费用补录待审批", "🟠 待审批", "查看订单列表", "/orders", []string{"**SE202610190001**", `应付 100\.00000000 CNY`, "进入费用页审核补录申请"}},
		{NotificationTemplateCommissionDecreaseSuggested, "COMMISSION_ADJUSTMENT", "SE202610190001", "冲减 10.00000000 CNY", "", "提成冲减通知", "🟠 冲减已确认", "查看本人冲减详情", "/commission-adjustments/" + id.String() + "/my-supplement-source", []string{"**SE202610190001**", `冲减 10\.00000000 CNY`, "本消息供你知情，实际少发或扣回由财务登记"}},
	} {
		t.Run(scenario.template, func(t *testing.T) {
			message, err := renderNotification(&NotificationDelivery{DingTalkUserID: "bound-user", Channel: NotificationChannelDingTalk, Template: scenario.template, ResourceType: scenario.resourceType, ResourceID: id, ReferenceCode: scenario.reference, Parameter: scenario.parameter, RecipientDisplayName: scenario.recipient})
			if err != nil {
				t.Fatal(err)
			}
			if message.Title != scenario.title || !strings.HasPrefix(message.Text, "### "+scenario.title+"\n"+scenario.status+"\n\n") || message.ActionTitle != scenario.actionTitle || message.ActionPath != scenario.actionPath || message.ActionURL != "" {
				t.Fatalf("标题、状态或受控动作错误:%+v", message)
			}
			for _, field := range scenario.fields {
				if !strings.Contains(message.Text, field) {
					t.Fatalf("消息缺少%q: %s", field, message.Text)
				}
			}
			if strings.Count(message.Text, "**") != 2 || strings.Contains(message.Text, "订单入口：") {
				t.Fatalf("消息过度加粗或保留入口文本:%s", message.Text)
			}
		})
	}
}

func TestFeeReviewResultUsesOnlyKnownSuffix(t *testing.T) {
	for _, tc := range []struct{ parameter, status string }{
		{"张三：已批准", "🟢 已批准"},
		{"张三：费用补录已批准", "🟢 已批准"},
		{"张三：已退回", "🔴 已退回"},
		{"张三：费用补录已退回", "🔴 已退回"},
		{"张三：费用更正已处理", "🔵 处理结果"},
		{"姓名含已批准：已退回", "🔴 已退回"},
		{"姓名含已退回：已批准", "🟢 已批准"},
		{"已批准同学：未知结果", "🔵 处理结果"},
		{"张三：已批准但未处理", "🔵 处理结果"},
	} {
		t.Run(tc.parameter, func(t *testing.T) {
			message, err := renderNotification(&NotificationDelivery{DingTalkUserID: "user", Channel: NotificationChannelDingTalk, Template: "FEE_REVIEW_RESULT", ResourceType: "ORDER", ResourceID: uuid.New(), ReferenceCode: "SE100", Parameter: tc.parameter})
			if err != nil || !strings.HasPrefix(message.Text, "### 订单费用审核结果\n"+tc.status+"\n") {
				t.Fatalf("状态被动态值误导:%+v %v", message, err)
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
			if err != nil || !strings.Contains(message.Text, want) || strings.Count(message.Text, "\n### ") > 0 || strings.Contains(message.Text, "\n# 标题") || strings.Contains(message.Text, "<b>") || strings.Contains(message.ActionPath, "example.com") || message.ActionURL != "" {
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
					if err != nil || message.ActionPath != scenario.wantPath || (message.ActionTitle != "") != (scenario.wantPath != "") || !strings.Contains(message.Text, "张三：已批准") {
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
