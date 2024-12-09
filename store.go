package main

import (
	"github.com/r3labs/sse/v2"
)

type MachinesStore struct {
	Store  *[]string
	Server *sse.Server
}

func (m *MachinesStore) Add(n *string) {
	exists := func() bool {
		for _, v := range *m.Store {
			if v == *n {
				return true
			}
		}

		return false
	}()
	if !exists {
		*m.Store = append(*m.Store, *n)
		m.Server.Publish("machine", &sse.Event{Data: []byte(*n)})
	}
}

func (m *MachinesStore) Get() *[]string {
	return m.Store
}
