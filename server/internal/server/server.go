package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewSessionPolicy, NewGRPCServer, NewHTTPServer, NewNotificationWorker, NewObjectDeletionWorker)
