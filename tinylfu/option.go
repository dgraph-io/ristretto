package tinylfu

// A Option allows callers to configure a Cache.
type Option func(p *Policy)

// An AdmissionPolicy augments a cache's eviction policy by letting the cache
// drop a new entry in favor of a victim which is more likely to be used again.
type AdmissionPolicy interface {
	Record(key uint64)
	Admit(candidate uint64, victim uint64) bool
}

// WithAdmission configures TinyLFU's admission policy, i.e. whether it rejects
// new entries in favor of evicting existing entries.
//
// By default, TinyLFU assumes admission.
func WithAdmission(admittor AdmissionPolicy) Option {
	return func(p *Policy) {
		p.admittor = admittor
	}
}

// A StatsRecorder allows TinyLFU to report performance metrics.
type StatsRecorder interface {
	RecordMiss()
	RecordHit()
	RecordEviction()
}

// WithRecorder configures TinyLFU to use the given instrumentation.
func WithRecorder(recorder StatsRecorder) Option {
	return func(p *Policy) {
		p.stats = recorder
	}
}

// WithSegmentation configures the relative sizes of TinyLFU's cache segments.
//
// TinyLFU uses two primary caches: an admission window to reduce the impact of
// many one-off accesses and a main cache. The main cache is further split into
// a probation segment and a protected segment for hot (frequent) entries.
//
// While optimal values may vary widely by workload, the default main and
// protected segments of 0.99 and 0.80, respectively, are good starting values.
//
// Example: Given a total capacity = 1000 with default segmentation
//   Window    = 1000 * (1 - 0.99)   = 10
//   Protected = 1000 * (0.99 * 0.8) = 792
//   Probation = 1000 - 792 - 10     = 198
func WithSegmentation(main, protected float64) Option {
	if main < 0 || main > 1 || protected < 0 || protected > 1 {
		panic("tinylfu: cache segment ratios must be within the range [0, 1]")
	}

	return func(p *Policy) {
		maxMain := int(float64(p.capacity) * main)
		if maxMain < 2 {
			// Leave at least one capacity each for probation and protected.
			maxMain = 2
		}
		if maxMain == p.capacity {
			// Leave at least one element for the window.
			maxMain = p.capacity - 1
		}
		p.maxWindow = p.capacity - maxMain

		p.maxProtected = int(float64(maxMain) * protected)
		if p.maxProtected < 1 {
			p.maxProtected = 1
		}
		if p.maxProtected == maxMain {
			// Leave at least one element for probation.
			p.maxProtected = maxMain - 1
		}
	}
}
