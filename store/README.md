# store

The storage backend of Ristretto. The primary interface exposed is `store.Map`:

```go
type Map interface {
    Get(string) interface{}
    Set(string, interface{})
    Del(string)
    Run(func(interface{}, interface{}) bool)
}
```

This package will eventually contain multiple hash map implementations fulfilling
the `store.Map` interface. The benefit of this will be increased performance and 
flexibility, as certain hash map implementations are better suited for certain 
data distributions.

The `store.Map` interface doesn't handle admission or eviction. It's just a
simple abstraction meant to decouple the storage part of Ristretto from the rest
of the components.
