package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
)

type directoryRouteService struct {
	v1.UnimplementedAdminServiceServer
	latestCalls int
	taskIDs     []string
}

func (s *directoryRouteService) GetLatestDingTalkDirectorySync(context.Context, *v1.GetLatestDingTalkDirectorySyncRequest) (*v1.GetLatestDingTalkDirectorySyncResponse, error) {
	s.latestCalls++
	return &v1.GetLatestDingTalkDirectorySyncResponse{}, nil
}

func (s *directoryRouteService) GetDingTalkDirectorySync(_ context.Context, request *v1.GetDingTalkDirectorySyncRequest) (*v1.GetDingTalkDirectorySyncResponse, error) {
	s.taskIDs = append(s.taskIDs, request.GetTaskId())
	return &v1.GetDingTalkDirectorySyncResponse{}, nil
}

// 实际注册生成的 HTTP 路由，防止 latest 被动态 task_id 路由捕获。
func TestDingTalkDirectoryHTTPRoutesDistinguishLatestAndTask(t *testing.T) {
	service := &directoryRouteService{}
	server := kratoshttp.NewServer()
	v1.RegisterAdminServiceHTTPServer(server, service)
	for _, segment := range []string{"latest", "018f0718-9baf-70fd-97eb-749611f7dbe4"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dingtalk/directory-syncs/"+segment+"?page=1&pageSize=20", nil)
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("同步路由 %s 返回 %d: %s", segment, response.Code, response.Body.String())
		}
	}
	if service.latestCalls != 1 || len(service.taskIDs) != 1 || service.taskIDs[0] != "018f0718-9baf-70fd-97eb-749611f7dbe4" {
		t.Fatalf("最新同步与指定任务路由混淆: latest=%d taskIDs=%v", service.latestCalls, service.taskIDs)
	}
}
