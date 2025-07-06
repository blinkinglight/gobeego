package appctx

import (
	"context"

	"github.com/blinkinglight/gobeego/pkg/rwdb"
	"github.com/ituoga/appcontext"
)

var dbKey = appcontext.Key[*rwdb.DB]("rwdb")

func WithDB(ctx context.Context, db *rwdb.DB) context.Context {
	return appcontext.With(ctx, dbKey, db)
}

func DB(ctx context.Context) *rwdb.DB {
	if db, ok := appcontext.Get(ctx, dbKey); ok {
		return db
	}
	panic("rwdb not found in context")
}
