package rwdb

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	R *gorm.DB
	W *gorm.DB
}

type Tx struct {
	*gorm.DB
}

type cbfn func(tx *Tx) error

func (db *DB) ReadTX(ctx context.Context, fn cbfn) error {
	return db.R.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Tx{tx})
	})
}

func (db *DB) WriteTX(ctx context.Context, fn cbfn) error {
	return db.W.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Tx{tx})
	})
}

func Open(file string) *DB {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  true,          // Disable color
		},
	)

	db1, err := gorm.Open(sqlite.Open(file), &gorm.Config{
		PrepareStmt: true,
		Logger:      newLogger,
	})
	if err != nil {
		panic(err)
	}

	rdb, _ := db1.DB()
	rdb.SetMaxOpenConns(runtime.NumCPU())
	rdb.SetConnMaxLifetime(-1)

	db2, err := gorm.Open(sqlite.Open(file), &gorm.Config{
		PrepareStmt: true,
		Logger:      newLogger,
	})
	if err != nil {
		panic(err)
	}

	wdb, _ := db2.DB()
	wdb.SetMaxOpenConns(1)
	wdb.SetConnMaxLifetime(-1)

	d := &DB{
		R: db1,
		W: db2,
	}
	return d

}
