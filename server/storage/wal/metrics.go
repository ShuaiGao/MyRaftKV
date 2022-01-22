package wal

var (
//walFsyncSec = prometheus.NewHistogram(prometheus.HistogramOpts{
//	Namespace: "etcd",
//	Subsystem: "disk",
//	Name:      "wal_fsync_duration_second",
//	Help:      "The latency distributions of fsync called byWAL.",
//
//	// lowest bucket start of upper bound 0.001 sec (1ms) with factor 2
//	// higest bucket start of 0.001 sec * 2^13 = 8.s92 sec
//	Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14),
//})
//walWriteBytes = prometheus.NewGauge(prometheus.GaugeOpts{
//	Namespace: "etcd",
//	Subsystem: "disk",
//	Name:      "wal_write_bytes_total",
//	Help:      "Total number of bytes written in WAL.",
//})
)

func init() {
	//prometheus.MustRegister(walFsyncSec)
	//prometheus.MustRegister(walWriteBytes)
}
