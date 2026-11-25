// Package sqlite contains the small, shared SQLite conventions used by the
// independently built RealtyRanker services. Keep copies in sync.
package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Open uses a local file shared by all services on the same host. Pragmas in
// the DSN apply to every connection, including connections opened later.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	if !strings.HasPrefix(dsn, "file:") {
		return nil, fmt.Errorf("SQLite DSN must start with file: (got a legacy or empty DSN)")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for _, pragma := range []string{"busy_timeout(10000)", "journal_mode(WAL)", "foreign_keys(ON)", "synchronous(FULL)", "cache_size(-1024)"} {
		q.Add("_pragma", pragma)
	}
	q.Set("_txlock", "immediate")
	u.RawQuery = q.Encode()
	pool, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(4)
	pool.SetMaxIdleConns(4)
	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

type Row interface {
	Scan(...any) error
}

type Strings []string
type Ints []int64
type Floats []float64

func arrayValue[T any](v []T) (driver.Value, error) {
	if v == nil {
		return "[]", nil
	}
	data, err := json.Marshal(v)
	return string(data), err
}

func (v Strings) Value() (driver.Value, error) { return arrayValue([]string(v)) }
func (v Ints) Value() (driver.Value, error)    { return arrayValue([]int64(v)) }
func (v Floats) Value() (driver.Value, error)  { return arrayValue([]float64(v)) }

type scannerFunc func(any) error

func (s scannerFunc) Scan(src any) error { return s(src) }

func JSON[T any](dst *T) sql.Scanner {
	return scannerFunc(func(src any) error {
		var data []byte
		switch v := src.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			return fmt.Errorf("expected JSON text, got %T", src)
		}
		return json.Unmarshal(data, dst)
	})
}

// Time keeps database comparisons and imported PostgreSQL timestamps exact:
// SQLite stores UTC Unix microseconds; API models continue to use time.Time.
func Time(dst *time.Time) sql.Scanner {
	return scannerFunc(func(src any) error {
		v, ok := src.(int64)
		if !ok {
			return fmt.Errorf("expected Unix microseconds, got %T", src)
		}
		*dst = time.UnixMicro(v).UTC()
		return nil
	})
}

func RowsChanged(result sql.Result) (bool, error) {
	n, err := result.RowsAffected()
	return n > 0, err
}
