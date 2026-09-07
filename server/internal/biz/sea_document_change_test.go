package biz

import (
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

func validAmendmentCommand() *SeaDocumentAmendmentCommand {
	value := "更新后的发货人"
	return &SeaDocumentAmendmentCommand{
		OrderID:                  uuid.New(),
		DocumentType:             SeaDocumentTypeMasterBill,
		DocumentID:               uuid.New(),
		ExpectedOrderVersion:     3,
		ExpectedDocumentVersion:  2,
		ExpectedCurrentVersionID: uuid.New(),
		Reason:                   " 客户书面更正 ",
		IdempotencyKey:           "amend-001",
		Input:                    &SeaDocumentAmendmentInput{MasterBillContent: &SeaBillContent{ShipperText: &value}},
		Confirmation:             &SeaExternalConfirmation{ConfirmedByParty: "船代", ConfirmedAt: time.Now(), ConfirmationNote: "已确认"},
	}
}

func TestValidateSeaDocumentChangeCommands(t *testing.T) {
	t.Run("Preview 允许无幂等键但规范化原因", func(t *testing.T) {
		cmd := validAmendmentCommand()
		cmd.IdempotencyKey = ""
		got, err := validateAmendmentCommand(cmd, false)
		if err != nil || got.Reason != "客户书面更正" || got.IdempotencyKey != "" {
			t.Fatalf("Preview 命令校验失败: got=%+v err=%v", got, err)
		}
	})

	t.Run("Execute 强制原因和幂等键", func(t *testing.T) {
		for name, mutate := range map[string]func(*SeaDocumentAmendmentCommand){
			"空原因":  func(cmd *SeaDocumentAmendmentCommand) { cmd.Reason = "  " },
			"空幂等键": func(cmd *SeaDocumentAmendmentCommand) { cmd.IdempotencyKey = "" },
		} {
			t.Run(name, func(t *testing.T) {
				cmd := validAmendmentCommand()
				mutate(cmd)
				_, err := validateAmendmentCommand(cmd, true)
				if !kratoserrors.IsBadRequest(err) {
					t.Fatalf("应返回参数错误，实际 %v", err)
				}
			})
		}
	})

	t.Run("Execute 强制 expected version 和当前不可变版本", func(t *testing.T) {
		cmd := validAmendmentCommand()
		cmd.ExpectedOrderVersion = 0
		if _, err := validateAmendmentCommand(cmd, true); !kratoserrors.IsBadRequest(err) {
			t.Fatalf("expected_order_version=0 应被拒绝: %v", err)
		}
		cmd = validAmendmentCommand()
		cmd.ExpectedCurrentVersionID = uuid.Nil
		if _, err := validateAmendmentCommand(cmd, true); !kratoserrors.IsBadRequest(err) {
			t.Fatalf("空当前不可变版本应被拒绝: %v", err)
		}
	})

	t.Run("模式切换强制外部确认", func(t *testing.T) {
		cmd := &SeaDocumentModeChangeCommand{OrderID: uuid.New(), ExpectedOrderVersion: 1, ExpectedLinkVersion: 1, TargetMode: SeaDocumentStructureHouse, Reason: "改为 HOUSE", IdempotencyKey: "mode-001", NewHouseBill: &SeaHouseBillInput{HouseNo: "HBL-001", IssuerSource: SeaHouseBillIssuerSourceCustomerPartner}}
		if _, err := validateModeChangeCommand(cmd, true); !kratoserrors.IsBadRequest(err) {
			t.Fatalf("缺少外部确认应被拒绝: %v", err)
		}
		cmd.Confirmation = &SeaExternalConfirmation{ConfirmedByParty: "船代", ConfirmedAt: time.Now(), ConfirmationNote: "已确认"}
		if _, err := validateModeChangeCommand(cmd, true); err != nil {
			t.Fatalf("合法模式切换被拒绝: %v", err)
		}
	})
}
