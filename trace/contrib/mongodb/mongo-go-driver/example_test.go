package mongo_go_driver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
example of *event.CommandMonitor:
{
	Command:{"find": "col","filter": {"name": "pico"},"limit": {"$numberLong":"1"},"singleBatch": true,"lsid": {"id": {"$binary":{"base64":"keX1d0QOQPissLFyTW1ZHA==","subType":"04"}}},"$db": "example"}
	DatabaseName:example
	CommandName:find
	RequestID:6
	ConnectionID:10.227.96.131:27017[-4]
	ServerConnectionID:0xc0000d625c
	ServiceID:<nil>
}
*/

func TestExample(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithLogger(&logger{}))
	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()

	mongoOpts := options.Client()
	// add monitor
	mongoOpts.Monitor = NewMonitor(tracer)
	mongoOpts.ApplyURI("mongodb://0.0.0.0:27017")
	client, err := mongo.Connect(context.Background(), mongoOpts)
	if err != nil {
		panic(err)
	}

	// root span
	span := tracer.StartServerSpan("root")
	ctx := aitracer.ContextWithSpan(context.Background(), span)

	db := client.Database("example")
	col := db.Collection("col")
	{
		// insert
		_, err = col.InsertOne(ctx, bson.D{
			{Key: "name", Value: "pico"},
			{Key: "price", Value: 1000},
			{Key: "timestamp", Value: time.Now()},
			{Key: "attributes", Value: bson.D{
				{Key: "foo", Value: 30},
				{Key: "bar", Value: "hello"},
				{Key: "pi", Value: 3.14},
			}},
		})
		if err != nil {
			panic(err)
		}
	}
	{
		// query
		res := col.FindOne(ctx, map[string]string{"name": "pico"})
		raw, err := res.DecodeBytes()
		if err != nil {
			fmt.Printf("err=%+v\n", err)
		}
		fmt.Printf("result=%+v\n", raw)
	}
	span.Finish() // must finish

	time.Sleep(1 * time.Second) // wait to print log
}

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug]"+format+"\n", args)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info]"+format+"\n", args)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error]"+format+"\n", args)
}
