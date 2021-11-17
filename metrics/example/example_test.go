package example

import (
	"testing"

	"github.com/volcengine/apminsight-server-sdk-go/metrics"
)

func TestMetrics_WithNewClient(t *testing.T) {
	client := metrics.NewMetricClient(metrics.WithPrefix("your metric common prefix"))
	client.Start()

	//without tags
	_ = client.EmitCounter("example_counter_metric", 1, nil)
	_ = client.EmitTimer("example_timer_metric", 1000, nil)
	_ = client.EmitGauge("example_gauge_metric", 100, nil)

	//with tags
	tags := map[string]string{
		"tagKey": "tagValue",
	}
	_ = client.EmitCounter("example_counter_metric", 1, tags)
	_ = client.EmitTimer("example_timer_metric", 1000, tags)
	_ = client.EmitGauge("example_gauge_metric", 100, tags)

	client.Close()
}

func TestMetrics_WithDefaultClient(t *testing.T) {

	metrics.Init(metrics.WithPrefix("your metric common prefix"))

	//without tags
	_ = metrics.EmitCounter("example_counter_metric", 1, nil)
	_ = metrics.EmitTimer("example_timer_metric", 1000, nil)
	_ = metrics.EmitGauge("example_gauge_metric", 100, nil)

	//with tags
	tags := map[string]string{
		"tagKey": "tagValue",
	}
	_ = metrics.EmitCounter("example_counter_metric", 1, tags)
	_ = metrics.EmitTimer("example_timer_metric", 1000, tags)
	_ = metrics.EmitGauge("example_gauge_metric", 100, tags)

	metrics.Close()
}
