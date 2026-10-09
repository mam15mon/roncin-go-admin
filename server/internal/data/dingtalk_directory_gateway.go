package data

import (
	"context"
	"encoding/json"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"net/http"
	"net/url"
	"strings"
)

func (p *dingTalkIdentityProvider) EnterpriseID() string { return p.corpID }
func (p *dingTalkIdentityProvider) directoryRequest(ctx context.Context, path string, body any, result any) error {
	token, err := p.enterpriseAccessToken(ctx)
	if err != nil {
		return biz.ErrDingTalkDirectoryFailed
	}
	var response struct {
		ErrorCode int             `json:"errcode"`
		Result    json.RawMessage `json:"result"`
	}
	_, err = p.requestJSON(ctx, http.MethodPost, "https://oapi.dingtalk.com"+path+"?access_token="+url.QueryEscape(token), body, "", &response)
	if err != nil || response.ErrorCode != 0 || len(response.Result) == 0 || string(response.Result) == "null" {
		return biz.ErrDingTalkDirectoryFailed
	}
	if err = json.Unmarshal(response.Result, result); err != nil {
		return biz.ErrDingTalkDirectoryFailed
	}
	return nil
}
func (p *dingTalkIdentityProvider) ReadDirectory(ctx context.Context) (*biz.DingTalkDirectorySnapshot, error) {
	if !p.enabled {
		return nil, biz.ErrDingTalkDisabled
	}
	snapshot := &biz.DingTalkDirectorySnapshot{CorpID: p.corpID, ScopeSummary: "当前应用授权的通讯录范围；全员范围需在钉钉开发者后台核实", AvailableFields: []string{"display_name", "departments"}}
	// 查询授权范围；仅根部门授权才能明确标记全企业范围。
	var scope struct {
		AuthDeptField []string `json:"auth_dept_field"`
		AuthUserField []string `json:"auth_user_field"`
		AuthOrgScopes struct {
			AuthDept []int64  `json:"authed_dept"`
			AuthUser []string `json:"authed_user"`
		} `json:"auth_org_scopes"`
	}
	// scope 接口没有 result 包装，单独处理；失败不可把局部授权宣称全员。
	token, err := p.enterpriseAccessToken(ctx)
	if err != nil {
		return nil, biz.ErrDingTalkDirectoryFailed
	}
	var scopeResponse struct {
		ErrorCode     int      `json:"errcode"`
		AuthDeptField []string `json:"auth_dept_field"`
		AuthUserField []string `json:"auth_user_field"`
		AuthOrgScopes struct {
			AuthDept []int64  `json:"authed_dept"`
			AuthUser []string `json:"authed_user"`
		} `json:"auth_org_scopes"`
	}
	_, err = p.requestJSON(ctx, http.MethodGet, "https://oapi.dingtalk.com/auth/scopes?access_token="+url.QueryEscape(token), nil, "", &scopeResponse)
	if err != nil || scopeResponse.ErrorCode != 0 {
		return nil, biz.ErrDingTalkDirectoryFailed
	}
	scope.AuthUserField = scopeResponse.AuthUserField
	scope.AuthOrgScopes = scopeResponse.AuthOrgScopes
	for _, id := range scope.AuthOrgScopes.AuthDept {
		if id == 1 {
			snapshot.FullScope = true
			snapshot.ScopeSummary = "通讯录授权范围：全企业"
		}
	}
	for _, field := range scope.AuthUserField {
		switch field {
		case "email", "mobile":
			snapshot.AvailableFields = append(snapshot.AvailableFields, field)
		case "avatar":
			snapshot.AvailableFields = append(snapshot.AvailableFields, "avatar_url")
		}
	}
	queue := append([]int64{}, scope.AuthOrgScopes.AuthDept...)
	if len(queue) == 0 && len(scope.AuthOrgScopes.AuthUser) == 0 {
		return nil, biz.ErrDingTalkDirectoryFailed
	}
	names := map[int64]string{1: "根部门"}
	seenDept := map[int64]bool{}
	seenUser := map[string]bool{}
	for len(queue) > 0 {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		id := queue[0]
		queue = queue[1:]
		if names[id] == "" {
			var department struct {
				Name string `json:"name"`
			}
			if err := p.directoryRequest(ctx, "/topapi/v2/department/get", map[string]any{"dept_id": id}, &department); err != nil {
				return nil, err
			}
			names[id] = department.Name
		}
		if seenDept[id] {
			continue
		}
		seenDept[id] = true
		var children []struct {
			ID   int64  `json:"dept_id"`
			Name string `json:"name"`
		}
		if err := p.directoryRequest(ctx, "/topapi/v2/department/listsub", map[string]any{"dept_id": id}, &children); err != nil {
			if id == 1 {
				return nil, biz.ErrDingTalkDirectoryFailed
			}
			snapshot.Failures = append(snapshot.Failures, biz.DingTalkDirectoryFailure{DepartmentID: id, Reason: "DINGTALK_DEPARTMENT_READ_FAILED", Message: "部门读取失败，请检查通讯录授权"})
		} else {
			for _, d := range children {
				queue = append(queue, d.ID)
				names[d.ID] = d.Name
			}
		}
		cursor := int64(0)
		for {
			var page struct {
				HasMore    *bool `json:"has_more"`
				NextCursor int64 `json:"next_cursor"`
				List       []struct {
					UserID        string  `json:"userid"`
					UnionID       string  `json:"unionid"`
					Name          string  `json:"name"`
					Email         *string `json:"email"`
					Mobile        *string `json:"mobile"`
					Avatar        *string `json:"avatar"`
					DepartmentIDs []int64 `json:"dept_id_list"`
				} `json:"list"`
			}
			err := p.directoryRequest(ctx, "/topapi/v2/user/list", map[string]any{"dept_id": id, "cursor": cursor, "size": 100}, &page)
			if err != nil {
				if id == 1 {
					return nil, biz.ErrDingTalkDirectoryFailed
				}
				snapshot.Failures = append(snapshot.Failures, biz.DingTalkDirectoryFailure{DepartmentID: id, Reason: "DINGTALK_MEMBERS_READ_FAILED", Message: "部门人员读取失败，请检查通讯录授权"})
				break
			}
			for _, u := range page.List {
				seenUser[u.UserID] = true
				if missing := directoryMissingOptionalFields(snapshot.AvailableFields, u.Email, u.Mobile, u.Avatar); len(missing) > 0 {
					snapshot.Failures = append(snapshot.Failures, biz.DingTalkDirectoryFailure{UserID: u.UserID, DepartmentID: id, Reason: "DINGTALK_PERSON_INCOMPLETE", Message: "钉钉部门人员接口未返回已授权字段：" + strings.Join(missing, "、") + "（字段缺失或为 null）；本条资料未更新，已有本地资料保留。请核对钉钉人员资料和应用字段权限后重新同步。"})
					continue
				}
				snapshot.People = append(snapshot.People, biz.DingTalkDirectoryPerson{UserID: strings.TrimSpace(u.UserID), UnionID: strings.TrimSpace(u.UnionID), Name: strings.TrimSpace(u.Name), Email: u.Email, Mobile: u.Mobile, AvatarURL: u.Avatar, Departments: []biz.DingTalkDepartment{{ID: id, Name: names[id]}}})
			}
			if page.HasMore == nil || page.List == nil {
				return nil, biz.ErrDingTalkDirectoryFailed
			}
			if !*page.HasMore {
				break
			}
			if page.NextCursor <= cursor {
				return nil, biz.ErrDingTalkDirectoryFailed
			}
			cursor = page.NextCursor
		}
	}
	// 独立授权的人员可能不在部门范围内，逐个读取而不扩大授权范围。
	for _, id := range scope.AuthOrgScopes.AuthUser {
		if seenUser[id] {
			continue
		}
		var person struct {
			UserID        string  `json:"userid"`
			UnionID       string  `json:"unionid"`
			Name          string  `json:"name"`
			Email         *string `json:"email"`
			Mobile        *string `json:"mobile"`
			Avatar        *string `json:"avatar"`
			DepartmentIDs []int64 `json:"dept_id_list"`
		}
		if err := p.directoryRequest(ctx, "/topapi/v2/user/get", map[string]any{"userid": id}, &person); err != nil {
			snapshot.Failures = append(snapshot.Failures, biz.DingTalkDirectoryFailure{UserID: id, Reason: "DINGTALK_PERSON_READ_FAILED", Message: "授权人员读取失败"})
			continue
		}
		if missing := directoryMissingOptionalFields(snapshot.AvailableFields, person.Email, person.Mobile, person.Avatar); len(missing) > 0 {
			snapshot.Failures = append(snapshot.Failures, biz.DingTalkDirectoryFailure{UserID: id, Reason: "DINGTALK_PERSON_INCOMPLETE", Message: "钉钉人员详情接口未返回已授权字段：" + strings.Join(missing, "、") + "（字段缺失或为 null）；本条资料未更新，已有本地资料保留。请核对钉钉人员资料和应用字段权限后重新同步。"})
			continue
		}
		depts := []biz.DingTalkDepartment{}
		for _, d := range person.DepartmentIDs {
			if names[d] == "" {
				var department struct {
					Name string `json:"name"`
				}
				if err := p.directoryRequest(ctx, "/topapi/v2/department/get", map[string]any{"dept_id": d}, &department); err != nil {
					snapshot.Failures = append(snapshot.Failures, biz.DingTalkDirectoryFailure{UserID: id, DepartmentID: d, Reason: "DINGTALK_DEPARTMENT_READ_FAILED", Message: "人员部门资料读取失败，保留本地资料"})
					depts = nil
					break
				}
				names[d] = department.Name
			}
			depts = append(depts, biz.DingTalkDepartment{ID: d, Name: names[d]})
		}
		if depts == nil {
			continue
		}
		snapshot.People = append(snapshot.People, biz.DingTalkDirectoryPerson{UserID: person.UserID, UnionID: person.UnionID, Name: person.Name, Email: person.Email, Mobile: person.Mobile, AvatarURL: person.Avatar, Departments: depts})
	}
	return snapshot, nil
}

var _ biz.DingTalkDirectoryGateway = (*dingTalkIdentityProvider)(nil)

// 已授权字段缺键/null 是不完整响应；明确空字符串才表示源资料为空。
func directoryMissingOptionalFields(fields []string, email, mobile, avatar *string) []string {
	var missing []string
	for _, field := range fields {
		var value *string
		var name string
		switch field {
		case "email":
			value, name = email, "邮箱"
		case "mobile":
			value, name = mobile, "联系方式"
		case "avatar_url":
			value, name = avatar, "头像"
		default:
			continue
		}
		if value == nil {
			missing = append(missing, name)
		}
	}
	return missing
}
