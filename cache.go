package caffeine

type Cache interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
}
