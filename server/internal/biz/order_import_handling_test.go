package biz

import (
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

func TestImportHandlingFixedParallelContract(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	h := EmptyImportHandling(uuid.New(), 1)
	if ImportHandlingFlow(h) != OrderFlowImportNotApplicable {
		t.Fatal("无承接应不适用")
	}
	h.Steps[4].Undertaken = true
	if ImportHandlingFlow(h) != OrderFlowImportNotStarted {
		t.Fatal("有承接无完成应未开始")
	}
	h.Steps[4].OccurredAt = &now
	if ImportHandlingFlow(h) != OrderFlowImportCompleted {
		t.Fatal("任意子集全部完成应已完成")
	}
	h.Steps[0].Undertaken = true
	if ImportHandlingFlow(h) != OrderFlowImportInProgress {
		t.Fatal("并行部分完成应进行中")
	}
	if e := ValidateImportHandling(h, nil, ""); e != nil {
		t.Fatal(e)
	}
	for _, mutation := range []func(*OrderImportHandling){
		func(h *OrderImportHandling) { h.Steps = h.Steps[:4] },
		func(h *OrderImportHandling) { h.Steps[1].Kind = "ARRIVAL" },
		func(h *OrderImportHandling) { h.Steps[2].Kind = "UNKNOWN" },
		func(h *OrderImportHandling) { h.Steps[1].OccurredAt = &now },
		func(h *OrderImportHandling) { h.Steps[0].ReferenceNo = strings.Repeat("文", 201) },
		func(h *OrderImportHandling) { id := uuid.New(); h.Steps[0].AttachmentIDs = []uuid.UUID{id, id} },
	} {
		bad := EmptyImportHandling(uuid.New(), 1)
		mutation(bad)
		if e := ValidateImportHandling(bad, nil, ""); e == nil {
			t.Fatal("非法完整快照未拒绝")
		}
	}
	before := EmptyImportHandling(uuid.New(), 1)
	before.Steps[4].Undertaken = true
	before.Steps[4].OccurredAt = &now
	after := EmptyImportHandling(before.OrderID, 1)
	after.Steps[4].Undertaken = true
	if e := ValidateImportHandling(after, before, ""); e == nil {
		t.Fatal("撤回必须原因")
	}
	if e := ValidateImportHandling(after, before, "录入时间有误"); e != nil {
		t.Fatal(e)
	}
}
