package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type backgroundTask struct {
	ID          uint64
	Label       string
	StartedAt   time.Time
	Cancellable bool
	cancel      func()
}

type backgroundTaskRegistry struct {
	mu     sync.Mutex
	nextID uint64
	tasks  map[uint64]backgroundTask
}

func newBackgroundTaskRegistry() *backgroundTaskRegistry {
	return &backgroundTaskRegistry{tasks: make(map[uint64]backgroundTask)}
}

func (registry *backgroundTaskRegistry) start(label string, cancel func()) uint64 {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.nextID++
	registry.tasks[registry.nextID] = backgroundTask{
		ID: registry.nextID, Label: label, StartedAt: time.Now(), Cancellable: cancel != nil, cancel: cancel,
	}
	return registry.nextID
}

func (registry *backgroundTaskRegistry) finish(id uint64) {
	registry.mu.Lock()
	delete(registry.tasks, id)
	registry.mu.Unlock()
}

func (registry *backgroundTaskRegistry) snapshot() []backgroundTask {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	result := make([]backgroundTask, 0, len(registry.tasks))
	for id := uint64(1); id <= registry.nextID; id++ {
		if task, ok := registry.tasks[id]; ok {
			result = append(result, task)
		}
	}
	return result
}

func (registry *backgroundTaskRegistry) cancel(id uint64) bool {
	registry.mu.Lock()
	task, ok := registry.tasks[id]
	registry.mu.Unlock()
	if !ok || task.cancel == nil {
		return false
	}
	task.cancel()
	return true
}

type backgroundTaskRow struct {
	Name      string
	StartedAt string
	Elapsed   string
	Action    string
}

func backgroundTaskRows(tasks []backgroundTask, now time.Time) []backgroundTaskRow {
	rows := make([]backgroundTaskRow, 0, len(tasks))
	for _, task := range tasks {
		elapsed := now.Sub(task.StartedAt).Round(time.Second)
		if elapsed < 0 {
			elapsed = 0
		}
		action := "等待完成"
		if task.Cancellable {
			action = "可取消"
		}
		rows = append(rows, backgroundTaskRow{
			Name: task.Label, StartedAt: task.StartedAt.Format("15:04:05"), Elapsed: elapsed.String(), Action: action,
		})
	}
	return rows
}

func (ui *mainUI) startBackgroundTask(label string, cancel func()) func() {
	if ui == nil || ui.backgroundTasks == nil {
		return func() {}
	}
	id := ui.backgroundTasks.start(label, cancel)
	ui.updateBackgroundTaskIndicator()
	return func() {
		defer func() { _ = recover() }()
		ui.backgroundTasks.finish(id)
		if ui.window == nil || ui.window.IsDisposed() {
			return
		}
		ui.window.Synchronize(ui.updateBackgroundTaskIndicator)
	}
}

func (ui *mainUI) updateBackgroundTaskIndicator() {
	if ui == nil || ui.backgroundTasks == nil || ui.taskButton == nil || ui.taskButton.IsDisposed() {
		return
	}
	count := len(ui.backgroundTasks.snapshot())
	ui.taskButton.SetText(fmt.Sprintf("后台任务（%d）", count))
	ui.taskButton.SetEnabled(count > 0)
	_ = ui.taskButton.Accessibility().SetName(fmt.Sprintf("查看后台任务，当前 %d 项", count))
}

func (ui *mainUI) showBackgroundTasks() {
	if ui == nil || ui.backgroundTasks == nil {
		return
	}
	var dlg *walk.Dialog
	var table *walk.TableView
	var info *walk.Label
	var cancelButton *walk.PushButton
	var closeButton *walk.PushButton
	tasks := ui.backgroundTasks.snapshot()
	refresh := func() {
		tasks = ui.backgroundTasks.snapshot()
		if table != nil {
			_ = table.SetModel(backgroundTaskRows(tasks, time.Now()))
		}
		if info != nil {
			info.SetText(fmt.Sprintf("当前有 %d 项后台任务。这里只允许取消查询导出等可安全中止的任务。", len(tasks)))
		}
		if cancelButton != nil && table != nil {
			index := table.CurrentIndex()
			cancelButton.SetEnabled(index >= 0 && index < len(tasks) && tasks[index].Cancellable)
		}
	}
	err := Dialog{
		AssignTo: &dlg, Title: "后台任务", DefaultButton: &closeButton,
		MinSize: Size{Width: 680, Height: 360}, Size: Size{Width: 760, Height: 430},
		Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "后台任务", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{AssignTo: &info, TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "后台任务状态"}},
			TableView{
				AssignTo: &table, Model: backgroundTaskRows(tasks, time.Now()), AlternatingRowBG: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "正在执行的后台任务"},
				Columns: []TableViewColumn{
					{Title: "任务", DataMember: "Name", Width: 300},
					{Title: "开始时间", DataMember: "StartedAt", Width: 100},
					{Title: "已用时", DataMember: "Elapsed", Width: 100},
					{Title: "状态", DataMember: "Action", Width: 100},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{Text: "刷新", MinSize: Size{Width: 84, Height: 30}, OnClicked: refresh},
				HSpacer{},
				PushButton{AssignTo: &cancelButton, Text: "取消所选任务", Enabled: false, MinSize: Size{Width: 112, Height: 30}, Accessibility: Accessibility{Name: "取消选中的安全后台任务"}},
				PushButton{AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 84, Height: 30}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法打开后台任务", err.Error(), walk.MsgBoxIconError)
		return
	}
	table.CurrentIndexChanged().Attach(func() {
		index := table.CurrentIndex()
		cancelButton.SetEnabled(index >= 0 && index < len(tasks) && tasks[index].Cancellable)
	})
	cancelButton.Clicked().Attach(func() {
		index := table.CurrentIndex()
		if index < 0 || index >= len(tasks) || !tasks[index].Cancellable {
			return
		}
		if ui.backgroundTasks.cancel(tasks[index].ID) {
			info.SetText("已发送取消请求；任务清理完成后会从列表移除。")
			cancelButton.SetEnabled(false)
		}
	})
	refresh()
	dlg.Run()
}
