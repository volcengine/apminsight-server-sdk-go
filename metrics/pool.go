package metrics

import (
	"sync"
)

var (
	metricItemPool = sync.Pool{
		New: func() interface{} {
			buf := make([]metricItem, 0, batchSize)
			return &buf
		},
	}
)

func getMetricItems() *[]metricItem {
	return metricItemPool.Get().(*[]metricItem)
}

func putMetricItems(items *[]metricItem) {
	*items = (*items)[:0]
	metricItemPool.Put(items)
}
