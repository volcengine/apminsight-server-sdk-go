package runtime

// metric names
const (
	metricGoRuntimeGoRoutineNum = "apminsight.runtime.go.routine.num"

	metricGoRuntimeCgoCallCount = "apminsight.runtime.go.cgo.call_count"

	metricGoRuntimeGcCount          = "apminsight.runtime.go.gc.count"
	metricGoRuntimeGcCostTotal      = "apminsight.runtime.go.gc.cost_total.us"
	metricGoRuntimeGcCostDistribute = "apminsight.runtime.go.gc.cost_distribute.us"

	metricGoRuntimeMemStatsHeapAlloc    = "apminsight.runtime.go.mem_stats.heap_alloc"
	metricGoRuntimeMemStatsHeapSys      = "apminsight.runtime.go.mem_stats.heap_sys"
	metricGoRuntimeMemStatsHeapIdle     = "apminsight.runtime.go.mem_stats.heap_idle"
	metricGoRuntimeMemStatsHeapInuse    = "apminsight.runtime.go.mem_stats.heap_inuse"
	metricGoRuntimeMemStatsHeapReleased = "apminsight.runtime.go.mem_stats.heap_released"
	metricGoRuntimeMemStatsHeapObjets   = "apminsight.runtime.go.mem_stats.heap_objects"

	metricGoRuntimeMemStatsLookups = "apminsight.runtime.go.mem_stats.lookups.count"

	metricGoRuntimeMemStatsStackInuse    = "apminsight.runtime.go.mem_stats.stack_inuse"
	metricGoRuntimeMemStatsStackSys      = "apminsight.runtime.go.mem_stats.stack_sys"
	metricGoRuntimeMemStatsMSpanInuse    = "apminsight.runtime.go.mem_stats.m_span_inuse"
	metricGoRuntimeMemStatsMSpanSys      = "apminsight.runtime.go.mem_stats.m_span_sys"
	metricGoRuntimeMemStatsMCacheInuse   = "apminsight.runtime.go.mem_stats.m_cache_inuse"
	metricGoRuntimeMemStatsMCacheSys     = "apminsight.runtime.go.mem_stats.m_cache_sys"
	metricGoRuntimeMemStatsBuckHashSys   = "apminsight.runtime.go.mem_stats.buck_hash_sys"
	metricGoRuntimeMemStatsGcSys         = "apminsight.runtime.go.mem_stats.gc_sys"
	metricGoRuntimeMemStatsOtherSys      = "apminsight.runtime.go.mem_stats.other_sys"
	metricGoRuntimeMemStatsNextGc        = "apminsight.runtime.go.mem_stats.next_gc"
	metricGoRuntimeMemStatsNumForcedGc   = "apminsight.runtime.go.mem_stats.forced_gc.num"
	metricGoRuntimeMemStatsGCCPUFraction = "apminsight.runtime.go.mem_stats.gc_cpu_fraction"

	// derived metric
	metricGoRuntimeMemStatsLiveObjects  = "apminsight.runtime.go.mem_stats.live_objects"  // Mallocs - Frees
	metricGoRuntimeMemStatsHeapRetained = "apminsight.runtime.go.mem_stats.heap_retained" //HeapIdle - HeapReleased
	metricGoRuntimeMemStatsHeapFragment = "apminsight.runtime.go.mem_stats.heap_fragment" //HeapInuse - HeapAlloc
)
