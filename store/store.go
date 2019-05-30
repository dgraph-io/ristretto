package store

import "sync"

type Map interface {
	Get(string) interface{}
	Set(string, interface{})
	Del(string)
	Filter(func(interface{}, interface{}) bool)
}

type Default struct {
	*sync.Map
}

func NewDefault() Map {
	return &Default{&sync.Map{}}
}

func (m *Default) Get(key string) interface{} {
	value, _ := m.Load(key)
	return value
}

func (m *Default) Set(key string, value interface{}) {
	m.Store(key, value)
}

func (m *Default) Del(key string) {
	m.Delete(key)
}

func (m *Default) Filter(f func(key, value interface{}) bool) {
	m.Range(f)
}
