package ui

import (
	"testing"
	"time"
)

func TestBackgroundTaskRegistryLifecycle(t *testing.T) {
	registry := newBackgroundTaskRegistry()
	canceled := false
	first := registry.start("库存查询导出", func() { canceled = true })
	second := registry.start("入库查询导出", nil)

	tasks := registry.snapshot()
	if len(tasks) != 2 || tasks[0].ID != first || tasks[1].ID != second {
		t.Fatalf("unexpected snapshot: %#v", tasks)
	}
	if !registry.cancel(first) || !canceled {
		t.Fatal("cancellable task was not canceled")
	}
	if registry.cancel(second) {
		t.Fatal("non-cancellable task reported cancellation")
	}
	registry.finish(first)
	registry.finish(second)
	if got := len(registry.snapshot()); got != 0 {
		t.Fatalf("task registry not empty: %d", got)
	}
}

func TestBackgroundTaskRowsNeverReportNegativeElapsed(t *testing.T) {
	now := time.Now()
	rows := backgroundTaskRows([]backgroundTask{{Label: "测试", StartedAt: now.Add(time.Minute)}}, now)
	if len(rows) != 1 || rows[0].Elapsed != "0s" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}
