package data

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
	supplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
)

// 批准和退回共用 decision_reason，须同时贯通 Unicode 校验和正式数据库列长度。
func TestFeeSupplementUnicodeDecisionReasonPostgres(t *testing.T) {
	for _, action := range []string{"approve", "reject"} {
		t.Run(action, func(t *testing.T) {
			f := newFeeSupplementFixture(t)
			o := f.createOrder("COMMENT-" + f.suffix)
			f.approveInitialFees(o)
			r := f.createSupplement(o, "unicode-"+action)
			p := f.approverPrincipal()
			preview, err := f.usecase.PreviewApproval(f.ctx, p, f.organizationID, o.ID, r.ID, r.Version)
			if err != nil {
				t.Fatal(err)
			}
			decide := func(reason string) error {
				if action == "approve" {
					_, err := f.usecase.Approve(f.ctx, p, f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, reason)
					return err
				}
				_, err := f.usecase.Reject(f.ctx, p, f.organizationID, o.ID, r.ID, r.Version, &reason)
				return err
			}
			if err := decide(strings.Repeat("审", 1001)); !errors.Is(err, biz.ErrFeeSupplementInvalidArgument) {
				t.Fatalf("1001 字决策意见须在领域边界拒绝: %v", err)
			}
			unchanged := f.requestByID(r.ID)
			if unchanged.Status != supplementent.StatusPENDING || unchanged.Version != r.Version {
				t.Fatalf("超长意见不得改变申请: %+v", unchanged)
			}
			comment := strings.Repeat("审", 999) + "😀"
			if err := decide(comment); err != nil {
				t.Fatalf("1000 个中文与补充平面字符应可批准或退回: %v", err)
			}
			stored := f.requestByID(r.ID)
			if stored.DecisionReason == nil || *stored.DecisionReason != comment {
				t.Fatalf("意见须原样保存: %+v", stored.DecisionReason)
			}
			if action == "approve" {
				event, err := f.data.db.OrderFeeReviewEvent.Query().Where(eventent.SupplementRequestIDEQ(r.ID)).Only(f.ctx)
				if err != nil || event.Comment != comment {
					t.Fatalf("整票批准事件须保存相同意见: %+v %v", event, err)
				}
			}
			if err := supplementent.DecisionReasonValidator(comment); err != nil {
				t.Fatalf("Ent 应按 Unicode 字符计数: %v", err)
			}
			if err := supplementent.DecisionReasonValidator(strings.Repeat("😀", 1001)); err == nil {
				t.Fatal("Ent 必须拒绝直接写入超过 1000 个字符")
			}
			// 正式执行器按迁移文件 SHA-256 记录校验和，须确认新迁移已真实应用。
			const migrationVersion = "20261012110000_fee_supplement_decision_reason"
			migrationSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", migrationVersion+".sql"))
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(migrationSQL)
			var checksum string
			if err := f.data.sqlDB.QueryRowContext(f.ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, migrationVersion).Scan(&checksum); err != nil || checksum != hex.EncodeToString(hash[:]) {
				t.Fatalf("正式迁移须记录当前 SQL 校验和: checksum=%s err=%v", checksum, err)
			}
			// Ent 对此字段生成不限制长度的 varchar，1000 字上限由领域和 Ent 共同校验。
			var limit sql.NullInt64
			if err := f.data.sqlDB.QueryRowContext(f.ctx, `SELECT character_maximum_length FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'order_fee_supplement_requests' AND column_name = 'decision_reason'`).Scan(&limit); err != nil || limit.Valid {
				t.Fatalf("正式迁移须移除旧的 500 字列上限，与 Ent Diff 一致: limit=%+v err=%v", limit, err)
			}
		})
	}
}
