package data

import (
	"errors"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
)

func TestOrderFeeReviewUnicodeCommentPostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "unicode-submit"))
	if err != nil {
		t.Fatal(err)
	}
	input := reviewAction(pending, "unicode-approve")
	input.Comment = strings.Repeat("审", 1001)
	if _, err := u.Act(f.ctx, p, "APPROVE", input); !errors.Is(err, biz.ErrFeeReviewInvalid) {
		t.Fatalf("超长意见应返回参数错误: %v", err)
	}
	input.Comment = strings.Repeat("审", 999) + "😀"
	approved, err := u.Act(f.ctx, p, "APPROVE", input)
	if err != nil || approved == nil || approved.Status != "APPROVED" {
		t.Fatalf("1000 个中英文及补充平面字符应能批准落库: %+v %v", approved, err)
	}
	event, err := f.data.db.OrderFeeReviewEvent.Query().Where(eventent.OrderIDEQ(o.ID), eventent.ActionEQ(eventent.ActionAPPROVE)).Only(f.ctx)
	if err != nil || event.Comment != input.Comment {
		t.Fatalf("意见应原样保存: %+v %v", event, err)
	}
	if err := eventent.CommentValidator(strings.Repeat("😀", 1001)); err == nil {
		t.Fatal("Ent 校验必须拒绝超过 1000 个字符的直接写入")
	}
}
