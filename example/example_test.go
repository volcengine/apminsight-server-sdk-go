package example

import (
	"testing"

	"github.com/volcengine/apminsight-server-sdk-go/metrics"
)

func TestMetrics_WithNewClient(t *testing.T) {
	client := metrics.NewMetricClient(metrics.WithPrefix("your metric common prefix"))
	client.Start()

	//without tags
	client.EmitCounter("example_counter_metric", 1, nil)
	client.EmitTimer("example_timer_metric", 1000, nil)
	client.EmitGauge("example_gauge_metric", 100, nil)

	//with tags
	tags := map[string]string{
		"tagKey": "tagValue",
	}
	client.EmitCounter("example_counter_metric", 1, tags)
	client.EmitTimer("example_timer_metric", 1000, tags)
	client.EmitGauge("example_gauge_metric", 100, tags)

	client.Close()
}

func TestMetrics_WithDefaultClient(t *testing.T) {

	metrics.Init(metrics.WithPrefix("your metric common prefix"))

	//without tags
	metrics.EmitCounter("example_counter_metric", 1, nil)
	metrics.EmitTimer("example_timer_metric", 1000, nil)
	metrics.EmitGauge("example_gauge_metric", 100, nil)

	//with tags
	tags := map[string]string{
		"tagKey": "tagValue",
	}
	metrics.EmitCounter("example_counter_metric", 1, tags)
	metrics.EmitTimer("example_timer_metric", 1000, tags)
	metrics.EmitGauge("example_gauge_metric", 100, tags)

	metrics.Close()
}
