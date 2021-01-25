package metrics

type t struct {
	key   string
	value string
}

type metricItem struct {
	mt    uint8
	name  string
	value float64
	tags  []t
}
