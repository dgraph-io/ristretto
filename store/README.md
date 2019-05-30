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

`store` will eventually contain multiple hash map implementations fulfilling
this interface. The benefit of this will be increased performance, as certain
types of hash maps work better with certain types of data distributions.

**NOTE**: The `store.Map` type does not handle admission and eviction. That is
left up to the user. The interface is designed to be as small as possible and
still usable for a concurrent cache backend.

## eviction

Eviction is done with the `Run(...)` and `Del(key)` methods.

The `Run(...)` method randomly applies the parameter function to elements in
the hash map. The user can count iterations and randomly sample entries for
LRU/LFU eviction.  
