package data

import (
	"context"
	"encoding/json"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func writeAudit(ctx context.Context, client *ent.AuditLogClient, event *biz.AuditEvent) error {
	requestID, traceID, ipAddress, details, err := resolveAuditValues(ctx, event)
	if err != nil {
		return err
	}
	create := client.Create().
		SetNillableOrganizationID(event.OrganizationID).
		SetNillableUserID(event.UserID).
		SetAction(event.Action).
		SetResult(auditlog.Result(event.Result)).
		SetRequestID(requestID).
		SetTraceID(traceID).
		SetIPAddress(ipAddress)
	if event.ResourceType != "" {
		create.SetResourceType(event.ResourceType)
	}
	if event.ResourceID != "" {
		create.SetResourceID(event.ResourceID)
	}
	if len(details) > 0 {
		create.SetDetails(details)
	}
	_, err = create.Save(ctx)
	return err
}

func resolveAuditValues(ctx context.Context, event *biz.AuditEvent) (string, string, string, json.RawMessage, error) {
	requestID := event.RequestID
	if requestID == "" {
		requestID = requestmeta.FromContext(ctx)
	}
	traceID := event.TraceID
	if traceID == "" {
		traceID = requestmeta.TraceID(ctx)
	}
	ipAddress := event.IPAddress
	if ipAddress == "" {
		ipAddress = requestmeta.IPAddress(ctx)
	}
	if len(event.Details) == 0 {
		return requestID, traceID, ipAddress, nil, nil
	}
	details, err := json.Marshal(event.Details)
	return requestID, traceID, ipAddress, details, err
}
