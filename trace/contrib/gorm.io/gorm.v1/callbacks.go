package gorm_v1

import (
	"context"

	"time"

	"strings"

	"fmt"

	"strconv"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"gorm.io/gorm"
)

type contextKey string

var contextKeySpan = contextKey("ai_tracer_span")

func WrapDB(dbType, endpoint, dbName string, db *gorm.DB, tracer aitracer.Tracer) (*gorm.DB, error) {
	cb := db.Callback()
	err := cb.Create().Before("gorm:create").Register("ai-tracer:before_create", newBefore(dbType, endpoint, dbName, "gorm:create", tracer))
	if err != nil {
		return nil, err
	}
	err = cb.Create().After("gorm:create").Register("ai-tracer:after_create", newAfterFunc())
	if err != nil {
		return nil, err
	}
	err = cb.Update().Before("gorm:update").Register("ai-tracer:before_update", newBefore(dbType, endpoint, dbName, "gorm:update", tracer))
	if err != nil {
		return nil, err
	}
	err = cb.Update().After("gorm:update").Register("ai-tracer:after_update", newAfterFunc())
	if err != nil {
		return nil, err
	}
	err = cb.Delete().Before("gorm:delete").Register("ai-tracer:before_delete", newBefore(dbType, endpoint, dbName, "gorm:delete", tracer))
	if err != nil {
		return nil, err
	}
	err = cb.Delete().After("gorm:delete").Register("ai-tracer:after_delete", newAfterFunc())
	if err != nil {
		return nil, err
	}
	err = cb.Query().Before("gorm:query").Register("ai-tracer:before_query", newBefore(dbType, endpoint, dbName, "gorm:query", tracer))
	if err != nil {
		return nil, err
	}
	err = cb.Query().After("gorm:query").Register("ai-tracer:after_query", newAfterFunc())
	if err != nil {
		return nil, err
	}
	err = cb.Row().Before("gorm:row").Register("ai-tracer:before_row", newBefore(dbType, endpoint, dbName, "gorm:row", tracer))
	if err != nil {
		return nil, err
	}
	err = cb.Row().After("gorm:row").Register("ai-tracer:after_row", newAfterFunc())
	if err != nil {
		return nil, err
	}
	err = cb.Raw().Before("gorm:raw").Register("ai-tracer:before_raw", newBefore(dbType, endpoint, dbName, "gorm:raw", tracer))
	if err != nil {
		return nil, err
	}
	err = cb.Raw().After("gorm:raw").Register("ai-tracer:after_raw", newAfterFunc())
	if err != nil {
		return nil, err
	}
	return db, nil
}

func newBefore(dbType, endpoint, dbName string, action string, tracer aitracer.Tracer) func(db *gorm.DB) {
	callService := endpoint + "/" + dbName
	return func(db *gorm.DB) {
		if db == nil {
			return
		}
		if db.Statement == nil {
			return
		}
		if db.Statement.Context == nil {
			return
		}
		span, ctx := tracer.StartClientSpanFromContext(db.Statement.Context,
			action, aitracer.ClientResourceAs(dbType, callService, action))
		span.SetTagString("peer.type", dbType)
		span.SetTagString("peer.address", endpoint)
		span.SetTagString("db.instance", dbName)
		span.SetTagString("call.resource", action)

		db.Statement.Context = context.WithValue(ctx, contextKeySpan, span)
	}
}

func newAfterFunc() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db == nil {
			return
		}
		if db.Statement == nil {
			return
		}
		if db.Statement.Context == nil {
			return
		}
		span, _ := db.Statement.Context.Value(contextKeySpan).(aitracer.Span)
		if span == nil {
			return
		}
		span.SetTagString("db.statement", db.Statement.SQL.String())

		// format vars
		{
			sb := strings.Builder{}
			sb.WriteString("[")
			first := true
			for _, v := range db.Statement.Vars {
				if !first {
					sb.WriteString(",")
				}
				switch vv := v.(type) {
				case string:
					sb.WriteString("'")
					sb.WriteString(vv)
					sb.WriteString("'")
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					sb.WriteString(fmt.Sprintf("%v", vv))
				case time.Time:
					sb.WriteString(strconv.FormatInt(vv.Unix(), 10))
				case gorm.DeletedAt:
					if vv.Valid {
						sb.WriteString(strconv.FormatInt(vv.Time.Unix(), 10))
					} else {
						sb.WriteString("null")
					}
				default:
					sb.WriteString("'?'")
				}
				first = false

			}
			sb.WriteString("]")
			span.SetTagString("db.sql.parameters", sb.String())
		}
		if db.Error != nil {
			span.FinishWithOption(aitracer.FinishSpanOption{
				FinishTime: time.Now(),
				Status:     1,
			})
		} else {
			span.Finish()
		}
	}
}
