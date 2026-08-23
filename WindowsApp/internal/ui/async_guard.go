package ui

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"zhengshi-wms-windowsapp/internal/diagnostics"
)

var asyncIncidentState struct {
	sync.RWMutex
	handler func(string)
}

func guardedGo(task func()) {
	_, source, line, ok := runtime.Caller(1)
	origin := "background"
	if ok {
		origin = fmt.Sprintf("%s:%d", filepath.Base(source), line)
	}
	go executeGuarded(origin, task)
}

func guardedGo1[T any](value T, task func(T)) {
	guardedGo(func() { task(value) })
}

func guardedGo2[A, B any](first A, second B, task func(A, B)) {
	guardedGo(func() { task(first, second) })
}

func executeGuarded(origin string, task func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			incidentID := diagnostics.RecordBackgroundPanic(origin, recovered)
			notifyAsyncIncident(incidentID)
		}
	}()
	task()
}

func setAsyncIncidentHandler(handler func(string)) {
	asyncIncidentState.Lock()
	asyncIncidentState.handler = handler
	asyncIncidentState.Unlock()
}

func notifyAsyncIncident(incidentID string) {
	asyncIncidentState.RLock()
	handler := asyncIncidentState.handler
	asyncIncidentState.RUnlock()
	if handler != nil {
		handler(incidentID)
	}
}
