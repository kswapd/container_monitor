package storage

import (
	"fmt"
	"sort"
	"time"
)

type StorageDriver interface {
	Send(start, end time.Time) error
	Start() error
}

type StorageDriverFunc func(monitorType string) (StorageDriver, error)

var registeredPlugins = map[string](StorageDriverFunc){}

func RegisterStorageDriver(name string, f StorageDriverFunc) {
	registeredPlugins[name] = f
}

func New(name string, monitorType string) (StorageDriver, error) {
	if name == "" {
		return nil, nil
	}
	f, ok := registeredPlugins[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend storage driver: %s", name)
	}
	return f(monitorType)
}

func ListDrivers() []string {
	drivers := make([]string, 0, len(registeredPlugins))
	for name := range registeredPlugins {
		drivers = append(drivers, name)
	}
	sort.Strings(drivers)
	return drivers
}
