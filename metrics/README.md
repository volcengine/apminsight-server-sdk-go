# APMInsight Server SDK Golang

## Installation

Get the code with:

```shell
go get github.com/volcengine/apminsight-server-sdk-go
```

recommend go >= 1.13.1

## Usage

see [CODE EXAMPLE](./example/example_test.go)

### Build Client

create a new client:

```go
client := metrics.NewMetricClient(metrics.WithPrefix("your metric common prefix"))
client.Close()
```

or init with default client:
```go
metrics.Init(metrics.WithPrefix("your metric common prefix"))
metrics.Close()
```

### Metrics

emit with your client

```go
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
```

or emit with default client

```go
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
```

### Configuration

you can config client with `metrics.WithXXX(value)`