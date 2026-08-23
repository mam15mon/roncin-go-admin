package server

import (
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

//go:generate go run ../../cmd/generate-access-rules -proto-root ../.. -output access_rules_gen.go

type accessMode uint8

const (
	accessModePublic accessMode = iota + 1
	accessModeAuthenticated
	accessModePermission
	accessModeOrderPermission
)

type accessRule struct {
	mode           accessMode
	permission     string
	scope          biz.DataScope
	orderOperation access.OrderOperation
}
