package ui

import "testing"

func TestExecuteGuardedReportsRecoveredIncident(t *testing.T) {
	reported := make(chan string, 1)
	setAsyncIncidentHandler(func(incidentID string) { reported <- incidentID })
	t.Cleanup(func() { setAsyncIncidentHandler(nil) })

	executeGuarded("test", func() { panic("do not log this text") })
	select {
	case incidentID := <-reported:
		if incidentID == "" {
			t.Fatal("incident id is empty")
		}
	default:
		t.Fatal("recovered panic was not reported")
	}
}

func TestExecuteGuardedRunsTask(t *testing.T) {
	run := false
	executeGuarded("test", func() { run = true })
	if !run {
		t.Fatal("task did not run")
	}
}
