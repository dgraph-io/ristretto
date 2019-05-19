package ring

type (
	Consumer interface {
		Push(uint64)
		Wrap(func())
	}

	Config struct {
		Consumer Consumer

		Size int32
		Rows uint64
	}
)
