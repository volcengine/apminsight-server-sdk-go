package mongo_go_driver

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
)

type spanKey struct {
	ConnectionID string
	RequestID    int64
}

type monitor struct {
	tracer aitracer.Tracer
	spans  map[spanKey]aitracer.Span
	sync.Mutex
}

func NewMonitor(tracer aitracer.Tracer) *event.CommandMonitor {
	if tracer == nil {
		panic("tracer is nil")
	}
	m := &monitor{
		tracer: tracer,
		spans:  make(map[spanKey]aitracer.Span),
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	if evt == nil {
		return
	}
	callService := fmt.Sprintf("mongodb:%s/%s", getAddr(evt), evt.DatabaseName)
	span, _ := m.tracer.StartClientSpanFromContext(ctx, "mongodb.command",
		aitracer.ClientResourceAs(aitracer.Mongodb, callService, evt.CommandName))
	span.SetTagString("peer.type", "mongodb")
	span.SetTagString(aitracer.DbStatement, toJSONString(evt.Command))
	span.SetTagString("mongodb.database", evt.DatabaseName)

	collection := tryGetCollection(evt)
	if collection != "" {
		span.SetTagString("mongodb.collection", collection)
	}

	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Mutex.Lock()
	m.spans[key] = span
	m.Mutex.Unlock()
}

func (m *monitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	if evt == nil {
		return
	}
	span, ok := m.getSpan(&evt.CommandFinishedEvent)
	if !ok {
		return
	}
	span.Finish()
}

func (m *monitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	if evt == nil {
		return
	}
	span, ok := m.getSpan(&evt.CommandFinishedEvent)
	if !ok {
		return
	}
	span.RecordError(fmt.Errorf(evt.Failure), aitracer.WithErrorKind(aitracer.ErrorKindDbError))
	span.SetStatus(aitracer.StatusCodeError)
	span.Finish()
}

func getAddr(evt *event.CommandStartedEvent) string {
	addr := evt.ConnectionID
	if idx := strings.IndexByte(addr, '['); idx >= 0 {
		addr = addr[:idx]
	}
	port := "27017"
	if idx := strings.IndexByte(addr, ':'); idx >= 0 {
		port = addr[idx+1:]
		addr = addr[:idx]
	}
	return addr + ":" + port
}

func tryGetCollection(evt *event.CommandStartedEvent) string {
	kv, err := evt.Command.IndexErr(0)
	if err != nil {
		return ""
	}
	if k := kv.Key(); k == evt.CommandName {
		if v := kv.Value(); v.Type == bsontype.String {
			return v.String()
		}
	}
	return ""
}

func toJSONString(command bson.Raw) string {
	b, _ := bson.MarshalExtJSON(command, false, false)
	return string(b)
}

func (m *monitor) getSpan(evt *event.CommandFinishedEvent) (aitracer.Span, bool) {
	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	span, ok := m.spans[key]
	if ok {
		delete(m.spans, key)
	}
	return span, ok
}
