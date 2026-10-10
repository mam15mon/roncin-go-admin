package service

import (
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestDingTalkDirectoryFailureNamesToAPI(t *testing.T) {
	result := dingTalkDirectorySyncToAPI(&biz.DingTalkDirectorySync{TaskID: uuid.New(), FailureTotal: 3, Page: 2, PageSize: 1, Failures: []biz.DingTalkDirectoryFailure{{UserID: "person-id", UserName: "失败人员", DepartmentID: 10, DepartmentName: "来源部门", Reason: "DINGTALK_PERSON_SYNC_FAILED", Message: "写入失败"}}})
	if len(result.Failures) != 1 || result.FailureTotal != 3 || result.Page != 2 || result.PageSize != 1 {
		t.Fatal("失败分页元数据未保留")
	}
	f := result.Failures[0]
	if f.UserId != "person-id" || f.UserName != "失败人员" || f.DepartmentId != "10" || f.DepartmentName != "来源部门" || f.Reason != "DINGTALK_PERSON_SYNC_FAILED" || f.Message != "写入失败" {
		t.Fatal("失败名称未贯穿 DTO 转换")
	}
	encoded, err := protojson.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var decoded v1.DingTalkDirectorySyncFailure
	if err := protojson.Unmarshal(encoded, &decoded); err != nil || decoded.UserName != f.UserName || decoded.DepartmentName != f.DepartmentName {
		t.Fatal("名称未贯穿 Proto JSON 响应")
	}
}
