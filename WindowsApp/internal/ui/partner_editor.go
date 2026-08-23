package ui

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

var mobilePattern = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

type partnerEditorUI struct {
	tab             *walk.TabPage
	kind            partnerKind
	mode            string
	baseline        partnerDetail
	entityType      *walk.ComboBox
	name            *walk.LineEdit
	code            *walk.LineEdit
	legal           *walk.LineEdit
	creditID        *walk.LineEdit
	levelLabel      *walk.Label
	level           *walk.ComboBox
	manager         *walk.LineEdit
	contact         *walk.LineEdit
	email           *walk.LineEdit
	address         *walk.LineEdit
	image           *walk.LineEdit
	selectImage     *walk.PushButton
	upload          *walk.PushButton
	preview         *walk.PushButton
	clearImage      *walk.PushButton
	remark          *walk.LineEdit
	receivableLabel *walk.Label
	receivable      *walk.LineEdit
	info            *walk.Label
	save            *walk.PushButton
	cancelButton    *walk.PushButton

	dirty        bool
	initializing bool
	busy         bool
	submitted    bool
	ctx          context.Context
	cancel       context.CancelFunc
	uploadCancel context.CancelFunc
	closed       atomic.Bool
}

type partnerEditorValues struct {
	Type                          string
	Name                          string
	Code                          string
	LegalRepresentative           string
	UnifiedSocialCreditIdentifier string
	Level                         int
	Manager                       string
	Contact                       string
	Email                         string
	Address                       string
	Image                         string
	Remark                        string
	ReceivableBalance             float64
}

func newPartnerEditorUI(kind partnerKind, mode string, baseline partnerDetail) *partnerEditorUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &partnerEditorUI{kind: kind, mode: mode, baseline: baseline, ctx: ctx, cancel: cancel, initializing: true}
}

func (state *partnerEditorUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
}

func (ui *mainUI) partnerEditorPageWidget(state *partnerEditorUI) TabPage {
	title := "新增" + state.kind.Label
	description := "填写内容将直接提交到线上服务；必填项和取值范围与现有接口一致。"
	if state.mode == "edit" {
		title = "编辑" + state.kind.Label + " · " + state.baseline.Name
		description = "保存前会重新读取线上更新时间；发现资料已变化时将停止覆盖。"
	}
	isSupplier := state.kind.Key == "supplier"
	isCustomerAdd := state.kind.Key == "customer" && state.mode == "add"
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle(title),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: title, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: description, TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: state.kind.Label + "编辑说明"}},
			GroupBox{
				Title:  "主体信息",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{Text: "主体类型 *"},
					ComboBox{AssignTo: &state.entityType, Model: []string{"个人", "企业", "组织"}, CurrentIndex: 1, MinSize: Size{Width: 210, Height: 28}},
					Label{Text: state.kind.Label + "名称 *"},
					LineEdit{AssignTo: &state.name, MinSize: Size{Width: 240, Height: 28}, CueBanner: "必填"},
					Label{Text: state.kind.Label + "编号 *"},
					LineEdit{AssignTo: &state.code, MinSize: Size{Width: 210, Height: 28}, CueBanner: "6 至 32 个字符"},
					Label{Text: "法定代表人 *"},
					LineEdit{AssignTo: &state.legal, MinSize: Size{Width: 240, Height: 28}},
					Label{Text: "身份证/信用代码 *"},
					LineEdit{AssignTo: &state.creditID, MinSize: Size{Width: 210, Height: 28}, CueBanner: "最多 18 个字符"},
					Label{AssignTo: &state.levelLabel, Text: "供应商等级 *", Visible: isSupplier},
					ComboBox{AssignTo: &state.level, Model: supplierEditorLevelLabels(), CurrentIndex: 0, Visible: isSupplier, MinSize: Size{Width: 240, Height: 28}},
				},
			},
			GroupBox{
				Title:  "联系信息",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{Text: "负责人 *"},
					LineEdit{AssignTo: &state.manager, MinSize: Size{Width: 210, Height: 28}},
					Label{Text: "联系电话 *"},
					LineEdit{AssignTo: &state.contact, MinSize: Size{Width: 240, Height: 28}, CueBanner: "完整手机号"},
					Label{Text: "Email"},
					LineEdit{AssignTo: &state.email, MinSize: Size{Width: 210, Height: 28}, CueBanner: "可选"},
					Label{Text: "地址"},
					LineEdit{AssignTo: &state.address, MinSize: Size{Width: 240, Height: 28}, CueBanner: "可选"},
				},
			},
			GroupBox{
				Title:  "图片与补充信息",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{Text: "图片"},
					Composite{ColumnSpan: 3, Layout: HBox{Spacing: 8}, Children: []Widget{
						LineEdit{AssignTo: &state.image, ReadOnly: true, StretchFactor: 1, CueBanner: "未选择图片"},
						PushButton{AssignTo: &state.selectImage, Text: "选择素材", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.selectPartnerImage},
						PushButton{AssignTo: &state.upload, Text: "上传图片", MinSize: Size{Width: 92, Height: 30}, ToolTipText: "上传期间再次点击可取消", Accessibility: Accessibility{Name: "上传合作伙伴图片"}, OnClicked: ui.uploadPartnerImage},
						PushButton{AssignTo: &state.preview, Text: "预览", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.previewPartnerImage},
						PushButton{AssignTo: &state.clearImage, Text: "清除", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.clearPartnerImage},
					}},
					Label{Text: "备注"},
					LineEdit{AssignTo: &state.remark, ColumnSpan: 3, CueBanner: "可选"},
					Label{AssignTo: &state.receivableLabel, Text: "期初应收账款", Visible: isCustomerAdd},
					LineEdit{AssignTo: &state.receivable, Visible: isCustomerAdd, CueBanner: "非负数字，默认 0"},
					HSpacer{ColumnSpan: 2},
				},
			},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "请核对必填项后保存。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: state.kind.Label + "编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 82, Height: 30}, OnClicked: func() { ui.closePartnerEditor(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并保存", MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.submitPartnerEditor},
			}},
		},
	}
}

func supplierEditorLevelLabels() []string {
	return []string{"一级", "二级", "三级"}
}

func (ui *mainUI) currentPartnerKind() (partnerKind, bool) {
	state := ui.partner
	if state == nil || state.kind == nil {
		return partnerKind{}, false
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kinds) {
		return partnerKind{}, false
	}
	return state.kinds[index], true
}

func (ui *mainUI) selectedPartner() (partnerDetail, partnerKind, bool) {
	kind, ok := ui.currentPartnerKind()
	if !ok || ui.partner.table == nil {
		return partnerDetail{}, partnerKind{}, false
	}
	index := ui.partner.table.CurrentIndex()
	if index < 0 || index >= len(ui.partner.rows) {
		return partnerDetail{}, kind, false
	}
	return ui.partner.rows[index].Detail, kind, true
}

func (ui *mainUI) updatePartnerActions() {
	state := ui.partner
	if state == nil || state.add == nil {
		return
	}
	kind, kindOK := ui.currentPartnerKind()
	_, _, selected := ui.selectedPartner()
	state.add.SetEnabled(kindOK && !state.busy && hasButton(ui.session.Perms.Buttons, kind.AddPermission))
	state.edit.SetEnabled(kindOK && selected && !state.busy && hasButton(ui.session.Perms.Buttons, kind.EditPermission))
	state.status.SetEnabled(kindOK && selected && !state.busy && hasButton(ui.session.Perms.Buttons, kind.StatusPermission))
	state.detail.SetEnabled(selected && !state.busy)
	if state.finance != nil {
		isCustomer := kindOK && kind.Key == "customer"
		state.finance.SetVisible(isCustomer)
		state.finance.SetEnabled(isCustomer && selected && !state.busy)
	}
	if state.action == nil {
		return
	}
	if state.busy {
		state.action.SetText("正在读取或提交线上资料，请等待完成。")
	} else if !selected {
		state.action.SetText("选择一条资料后可按账号权限执行编辑和状态变更。")
	} else {
		state.action.SetText("写操作会在提交前重新读取，并在成功后回读确认。")
	}
}

func (ui *mainUI) newPartner() {
	kind, ok := ui.currentPartnerKind()
	if !ok || !hasButton(ui.session.Perms.Buttons, kind.AddPermission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有新增该类合作伙伴的明确按钮权限。", walk.MsgBoxIconWarning)
		return
	}
	ui.openPartnerEditor(kind, "add", partnerDetail{})
}

func (ui *mainUI) editSelectedPartner() {
	detail, kind, ok := ui.selectedPartner()
	if !ok {
		walk.MsgBox(ui.window, "请选择合作伙伴", "请先选择需要编辑的资料。", walk.MsgBoxIconInformation)
		return
	}
	if !hasButton(ui.session.Perms.Buttons, kind.EditPermission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有编辑该类合作伙伴的权限。", walk.MsgBoxIconWarning)
		return
	}
	ui.openPartnerEditor(kind, "edit", detail)
}

func (ui *mainUI) openPartnerEditor(kind partnerKind, mode string, baseline partnerDetail) {
	if current := ui.partnerEditor; current != nil {
		if current.busy {
			walk.MsgBox(ui.window, "正在提交", "当前合作伙伴资料正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if current.kind.Key == kind.Key && current.mode == mode && current.baseline.ID == baseline.ID {
			_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(current.tab))
			return
		}
		if current.dirty && walk.MsgBox(ui.window, "替换未提交编辑", "当前合作伙伴资料还有未提交修改，是否放弃并打开另一条资料？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		ui.closePartnerEditor(true)
	}

	state := newPartnerEditorUI(kind, mode, baseline)
	decl := ui.partnerEditorPageWidget(state)
	if err := decl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.partnerEditor = state
	ui.partnerEditorTab = state.tab
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if ui.partnerTab != nil {
		if index := ui.tabs.Pages().Index(ui.partnerTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.tab.Dispose()
		state.dispose()
		ui.partnerEditor = nil
		ui.partnerEditorTab = nil
		walk.MsgBox(ui.window, "无法打开编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializePartnerEditor(state)
}

func (ui *mainUI) initializePartnerEditor(state *partnerEditorUI) {
	if state == nil || state != ui.partnerEditor {
		return
	}
	baseline := state.baseline
	state.entityType.SetCurrentIndex(stringIndex([]string{"个人", "企业", "组织"}, baseline.Type))
	if state.entityType.CurrentIndex() < 0 {
		state.entityType.SetCurrentIndex(1)
	}
	state.name.SetText(baseline.Name)
	state.code.SetText(baseline.Code)
	state.legal.SetText(baseline.LegalPerson)
	state.creditID.SetText(baseline.CreditIdentifier)
	if baseline.LevelValue > 0 {
		state.level.SetCurrentIndex(baseline.LevelValue - 1)
	}
	state.manager.SetText(baseline.Manager)
	state.contact.SetText(baseline.Contact)
	state.email.SetText(baseline.Email)
	state.address.SetText(baseline.Address)
	state.image.SetText(baseline.Image)
	state.remark.SetText(baseline.Remark)
	state.receivable.SetText("0")
	ui.refreshPartnerImageActions()
	markDirty := func() {
		if state.initializing || state.busy || state.submitted {
			return
		}
		state.dirty = true
		state.info.SetText("存在尚未提交的修改。")
	}
	for _, edit := range []*walk.LineEdit{state.name, state.code, state.legal, state.creditID, state.manager, state.contact, state.email, state.address, state.remark, state.receivable} {
		edit.TextChanged().Attach(markDirty)
	}
	state.entityType.CurrentIndexChanged().Attach(markDirty)
	state.level.CurrentIndexChanged().Attach(markDirty)
	state.initializing = false
	state.dirty = false
	state.name.SetFocus()
}

func (ui *mainUI) closePartnerEditor(force bool) {
	state := ui.partnerEditor
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在提交", "资料正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && walk.MsgBox(ui.window, "放弃未提交修改", "当前合作伙伴资料还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
	}
	if index := ui.tabs.Pages().Index(state.tab); index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭编辑页", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.partnerEditor = nil
	ui.partnerEditorTab = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) uploadPartnerImage() {
	state := ui.partnerEditor
	if state == nil {
		return
	}
	if state.uploadCancel != nil {
		state.uploadCancel()
		state.upload.SetText("正在取消")
		state.upload.SetEnabled(false)
		state.info.SetText("正在取消图片上传……")
		return
	}
	if state.busy || state.submitted {
		return
	}
	dialog := new(walk.FileDialog)
	dialog.Title = "选择" + state.kind.Label + "图片"
	dialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg;*.gif;*.svg)|*.png;*.jpg;*.jpeg;*.gif;*.svg|所有文件 (*.*)|*.*"
	if ok, err := dialog.ShowOpen(ui.window); err != nil {
		walk.MsgBox(ui.window, "无法选择图片", err.Error(), walk.MsgBoxIconError)
		return
	} else if !ok {
		return
	}
	previous := state.image.Text()
	state.busy = true
	uploadCtx, uploadCancel := context.WithCancel(state.ctx)
	state.uploadCancel = uploadCancel
	ui.setPartnerEditorEnabled(false)
	state.upload.SetText("取消上传")
	state.upload.SetEnabled(true)
	state.info.SetText("正在上传图片；失败时将保留原图片。")
	guardedGo1(dialog.FilePath, func(filePath string) {
		defer uploadCancel()
		reference, err := ui.session.Client.UploadImage(uploadCtx, filePath)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		canceled := uploadCtx.Err() != nil
		ui.window.Synchronize(func() {
			if state != ui.partnerEditor || state.closed.Load() {
				return
			}
			state.uploadCancel = nil
			state.busy = false
			ui.setPartnerEditorEnabled(true)
			state.upload.SetText("上传图片")
			if canceled {
				state.image.SetText(previous)
				state.info.SetText("图片上传已取消，原图片已保留。")
				return
			}
			if err != nil {
				state.image.SetText(previous)
				state.info.SetText("图片上传失败：" + err.Error() + "。原图片已保留，可重新选择。")
				return
			}
			state.image.SetText(reference)
			state.dirty = true
			ui.refreshPartnerImageActions()
			state.info.SetText("图片已上传并加入当前资料；保存后才会关联到合作伙伴。")
		})
	})
}

func (ui *mainUI) selectPartnerImage() {
	state := ui.partnerEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	references, ok := SelectImageAssets(ui.window, ui.session.Client, config.ImageBaseURL(), 1)
	if !ok || len(references) == 0 {
		return
	}
	state.image.SetText(references[0])
	state.dirty = true
	ui.refreshPartnerImageActions()
	state.info.SetText("已选择线上图片素材，保存资料后生效。")
}

func (ui *mainUI) previewPartnerImage() {
	state := ui.partnerEditor
	if state == nil || strings.TrimSpace(state.image.Text()) == "" {
		return
	}
	ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), state.kind.Label+" "+state.name.Text(), []string{state.image.Text()})
}

func (ui *mainUI) clearPartnerImage() {
	state := ui.partnerEditor
	if state == nil || state.busy || state.submitted || strings.TrimSpace(state.image.Text()) == "" {
		return
	}
	state.image.SetText("")
	state.dirty = true
	ui.refreshPartnerImageActions()
	state.info.SetText("图片已从当前提交内容清除；服务端图片库文件不会被删除。")
}

func (ui *mainUI) refreshPartnerImageActions() {
	state := ui.partnerEditor
	if state == nil || state.image == nil {
		return
	}
	hasImage := strings.TrimSpace(state.image.Text()) != ""
	state.preview.SetEnabled(hasImage && !state.busy)
	state.clearImage.SetEnabled(hasImage && !state.busy && !state.submitted)
}

func (ui *mainUI) setPartnerEditorEnabled(enabled bool) {
	state := ui.partnerEditor
	if state == nil {
		return
	}
	canEdit := enabled && !state.submitted
	for _, edit := range []*walk.LineEdit{state.name, state.code, state.legal, state.creditID, state.manager, state.contact, state.email, state.address, state.remark, state.receivable} {
		edit.SetEnabled(canEdit)
	}
	state.entityType.SetEnabled(canEdit)
	state.level.SetEnabled(canEdit)
	state.upload.SetEnabled(canEdit)
	state.selectImage.SetEnabled(canEdit)
	state.save.SetEnabled(canEdit)
	state.cancelButton.SetEnabled(enabled)
	ui.refreshPartnerImageActions()
}

func collectPartnerEditorValues(state *partnerEditorUI) (partnerEditorValues, error) {
	level := 0
	if state.kind.Key == "supplier" {
		level = state.level.CurrentIndex() + 1
	}
	receivable := 0.0
	if state.kind.Key == "customer" && state.mode == "add" {
		value := strings.TrimSpace(state.receivable.Text())
		if value != "" {
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return partnerEditorValues{}, fmt.Errorf("期初应收账款必须是非负数字")
			}
			receivable = parsed
		}
	}
	values := partnerEditorValues{
		Type: state.entityType.Text(), Name: state.name.Text(), Code: state.code.Text(),
		LegalRepresentative: state.legal.Text(), UnifiedSocialCreditIdentifier: state.creditID.Text(),
		Level: level, Manager: state.manager.Text(), Contact: state.contact.Text(), Email: state.email.Text(),
		Address: state.address.Text(), Image: state.image.Text(), Remark: state.remark.Text(), ReceivableBalance: receivable,
	}
	return values, validatePartnerEditorValues(state.kind.Key, state.mode, values)
}

func validatePartnerEditorValues(kind, mode string, values partnerEditorValues) error {
	values.Type = strings.TrimSpace(values.Type)
	if stringIndex([]string{"个人", "企业", "组织"}, values.Type) < 0 {
		return fmt.Errorf("请选择有效的主体类型")
	}
	if strings.TrimSpace(values.Name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	codeLength := utf8.RuneCountInString(strings.TrimSpace(values.Code))
	if codeLength < 6 || codeLength > 32 {
		return fmt.Errorf("编号长度必须为 6 至 32 个字符")
	}
	if strings.TrimSpace(values.LegalRepresentative) == "" {
		return fmt.Errorf("法定代表人不能为空")
	}
	creditID := strings.TrimSpace(values.UnifiedSocialCreditIdentifier)
	if creditID == "" || utf8.RuneCountInString(creditID) > 18 {
		return fmt.Errorf("身份证或统一社会信用代码不能为空且不能超过 18 个字符")
	}
	if kind == "supplier" && (values.Level < 1 || values.Level > 3) {
		return fmt.Errorf("供应商等级必须为一级、二级或三级")
	}
	if strings.TrimSpace(values.Manager) == "" {
		return fmt.Errorf("负责人不能为空")
	}
	if !mobilePattern.MatchString(strings.TrimSpace(values.Contact)) {
		return fmt.Errorf("联系电话必须是有效手机号")
	}
	if email := strings.TrimSpace(values.Email); email != "" {
		address, err := mail.ParseAddress(email)
		if err != nil || !strings.EqualFold(address.Address, email) {
			return fmt.Errorf("Email 格式不正确")
		}
	}
	if kind == "customer" && mode == "add" && values.ReceivableBalance < 0 {
		return fmt.Errorf("期初应收账款不能小于 0")
	}
	return nil
}

func (ui *mainUI) submitPartnerEditor() {
	state := ui.partnerEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	permission := state.kind.AddPermission
	if state.mode == "edit" {
		permission = state.kind.EditPermission
	}
	if !hasButton(ui.session.Perms.Buttons, permission) {
		state.info.SetText("当前账号缺少本次操作权限，未发送请求。")
		return
	}
	values, err := collectPartnerEditorValues(state)
	if err != nil {
		state.info.SetText("无法提交：" + err.Error())
		return
	}
	action := "新增"
	if state.mode == "edit" {
		action = "编辑"
	}
	message := fmt.Sprintf("操作：%s%s\r\n编号：%s\r\n名称：%s", action, state.kind.Label, strings.TrimSpace(values.Code), strings.TrimSpace(values.Name))
	if state.kind.Key == "customer" && state.mode == "add" && values.ReceivableBalance > 0 {
		message += fmt.Sprintf("\r\n期初应收：%.2f\r\n\r\n服务端将同时生成期初应收交易流水。", values.ReceivableBalance)
	}
	message += "\r\n\r\n是否提交到线上服务？"
	if walk.MsgBox(ui.window, "核对合作伙伴资料", message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}
	state.busy = true
	ui.setPartnerEditorEnabled(false)
	state.info.SetText("正在重新读取线上资料并提交……")
	guardedGo(func() {
		requestErr := ui.writePartner(state.ctx, state.kind, state.mode, state.baseline, values)
		var verifyErr error
		if requestErr == nil {
			verifyErr = ui.verifyPartnerWrite(state.ctx, state.kind, state.baseline.ID, values.Code, values.Name)
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.partnerEditor || state.closed.Load() {
				return
			}
			state.busy = false
			if requestErr != nil {
				ui.setPartnerEditorEnabled(true)
				state.info.SetText("提交失败：" + requestErr.Error() + "。请刷新线上资料后重试。")
				return
			}
			state.dirty = false
			if verifyErr != nil {
				state.submitted = true
				ui.setPartnerEditorEnabled(false)
				state.cancelButton.SetEnabled(true)
				state.info.SetText("服务端已返回成功，但回读失败：" + verifyErr.Error() + "。请勿重复提交，关闭后刷新列表核对。")
				walk.MsgBox(ui.window, "提交后复核失败", state.info.Text(), walk.MsgBoxIconWarning)
				ui.loadPartners()
				return
			}
			ui.notifyStatusSuccess("合作伙伴资料已保存，并完成线上回读复核。")
			ui.loadPartners()
			ui.loadFilterOptions()
			ui.closePartnerEditor(true)
		})
	})
}

func (ui *mainUI) writePartner(ctx context.Context, kind partnerKind, mode string, baseline partnerDetail, values partnerEditorValues) error {
	if mode == "edit" {
		updatedAt, found, err := ui.partnerUpdatedAt(ctx, kind, baseline.ID, baseline.Code)
		if err != nil {
			return fmt.Errorf("提交前重新读取失败：%w", err)
		}
		if !found {
			return fmt.Errorf("资料已不存在")
		}
		if baseline.UpdatedAt > 0 && updatedAt != baseline.UpdatedAt {
			return fmt.Errorf("资料已被其他用户更新，请关闭编辑页并刷新后重试")
		}
	}
	switch kind.Key {
	case "supplier":
		request := api.SupplierRequest{
			ID: baseline.ID, Type: strings.TrimSpace(values.Type), Level: values.Level,
			Code: strings.TrimSpace(values.Code), Image: strings.TrimSpace(values.Image), Name: strings.TrimSpace(values.Name),
			LegalRepresentative: strings.TrimSpace(values.LegalRepresentative), UnifiedSocialCreditIdentifier: strings.TrimSpace(values.UnifiedSocialCreditIdentifier),
			Contact: strings.TrimSpace(values.Contact), Manager: strings.TrimSpace(values.Manager), Email: strings.TrimSpace(values.Email),
			Address: strings.TrimSpace(values.Address), Remark: strings.TrimSpace(values.Remark),
		}
		if mode == "add" {
			return ui.session.Client.CreateSupplier(ctx, request)
		}
		return ui.session.Client.UpdateSupplier(ctx, request)
	case "customer":
		request := api.CustomerRequest{
			ID: baseline.ID, Type: strings.TrimSpace(values.Type), Code: strings.TrimSpace(values.Code), Name: strings.TrimSpace(values.Name),
			Image: strings.TrimSpace(values.Image), LegalRepresentative: strings.TrimSpace(values.LegalRepresentative),
			UnifiedSocialCreditIdentifier: strings.TrimSpace(values.UnifiedSocialCreditIdentifier), Address: strings.TrimSpace(values.Address),
			Contact: strings.TrimSpace(values.Contact), Manager: strings.TrimSpace(values.Manager), Email: strings.TrimSpace(values.Email),
			Remark: strings.TrimSpace(values.Remark), ReceivableBalance: values.ReceivableBalance,
		}
		if mode == "add" {
			return ui.session.Client.CreateCustomer(ctx, request)
		}
		return ui.session.Client.UpdateCustomer(ctx, request)
	case "carrier":
		request := api.CarrierRequest{
			ID: baseline.ID, Type: strings.TrimSpace(values.Type), Code: strings.TrimSpace(values.Code), Name: strings.TrimSpace(values.Name),
			Image: strings.TrimSpace(values.Image), LegalRepresentative: strings.TrimSpace(values.LegalRepresentative),
			UnifiedSocialCreditIdentifier: strings.TrimSpace(values.UnifiedSocialCreditIdentifier), Address: strings.TrimSpace(values.Address),
			Contact: strings.TrimSpace(values.Contact), Manager: strings.TrimSpace(values.Manager), Email: strings.TrimSpace(values.Email), Remark: strings.TrimSpace(values.Remark),
		}
		if mode == "add" {
			return ui.session.Client.CreateCarrier(ctx, request)
		}
		return ui.session.Client.UpdateCarrier(ctx, request)
	default:
		return fmt.Errorf("不支持的合作伙伴类型：%s", kind.Label)
	}
}

func (ui *mainUI) partnerUpdatedAt(ctx context.Context, kind partnerKind, id, code string) (int64, bool, error) {
	switch kind.Key {
	case "supplier":
		item, found, err := ui.session.Client.FindSupplier(ctx, id, code)
		return item.UpdatedAt, found, err
	case "customer":
		item, found, err := ui.session.Client.FindCustomer(ctx, id, code)
		return item.UpdatedAt, found, err
	case "carrier":
		item, found, err := ui.session.Client.FindCarrier(ctx, id, code)
		return item.UpdatedAt, found, err
	default:
		return 0, false, fmt.Errorf("不支持的合作伙伴类型")
	}
}

func (ui *mainUI) verifyPartnerWrite(ctx context.Context, kind partnerKind, id, code, name string) error {
	if strings.TrimSpace(id) != "" {
		_, found, err := ui.partnerUpdatedAt(ctx, kind, id, code)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("未查询到编号 %s 的资料", code)
		}
		return nil
	}
	filters := api.PartnerFilters{Code: strings.TrimSpace(code)}
	result, err := ui.loadPartnerRows(ctx, kind, 1, 100, filters)
	if err != nil {
		return err
	}
	for _, row := range result.rows {
		if strings.EqualFold(strings.TrimSpace(row.Code), strings.TrimSpace(code)) && strings.TrimSpace(row.Name) == strings.TrimSpace(name) {
			return nil
		}
	}
	return fmt.Errorf("未查询到新增后的资料 %s", code)
}

func partnerStatuses(kind string) []string {
	switch kind {
	case "supplier":
		return []string{"审核中", "审核不通过", "活动", "停用", "黑名单", "合同到期"}
	case "customer":
		return []string{"潜在", "活动", "停用", "冻结", "黑名单", "合同到期"}
	case "carrier":
		return []string{"活跃", "停用", "暂停合作", "终止合作", "待审核", "审核不通过", "审核中", "审核通过", "资质过期"}
	default:
		return nil
	}
}

func (ui *mainUI) changeSelectedPartnerStatus() {
	detail, kind, ok := ui.selectedPartner()
	if !ok {
		walk.MsgBox(ui.window, "请选择合作伙伴", "请先选择需要变更状态的资料。", walk.MsgBoxIconInformation)
		return
	}
	if !hasButton(ui.session.Perms.Buttons, kind.StatusPermission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有变更该类合作伙伴状态的权限。", walk.MsgBoxIconWarning)
		return
	}
	statuses := partnerStatuses(kind.Key)
	var dlg *walk.Dialog
	var target *walk.ComboBox
	var okButton *walk.PushButton
	accepted := false
	err := Dialog{
		AssignTo: &dlg, Title: "变更" + kind.Label + "状态", DefaultButton: &okButton,
		MinSize: Size{Width: 480, Height: 260}, Size: Size{Width: 520, Height: 300},
		Layout: VBox{Margins: Margins{Left: 20, Top: 18, Right: 20, Bottom: 18}, Spacing: 12},
		Children: []Widget{
			Label{Text: detail.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 13, Bold: true}},
			Label{Text: "当前状态：" + displayMaterialValue(detail.Status)},
			GroupBox{Title: "目标状态", Layout: VBox{Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}}, Children: []Widget{
				ComboBox{AssignTo: &target, Model: statuses, CurrentIndex: stringIndex(statuses, detail.Status), MinSize: Size{Height: 30}},
			}},
			Label{Text: "本阶段不提供删除状态；提交前会重新读取线上资料。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &okButton, Text: "继续", OnClicked: func() { accepted = true; dlg.Accept() }},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法打开状态窗口", err.Error(), walk.MsgBoxIconError)
		return
	}
	if target.CurrentIndex() < 0 {
		target.SetCurrentIndex(0)
	}
	dlg.Run()
	if !accepted {
		return
	}
	status := target.Text()
	if status == detail.Status {
		walk.MsgBox(ui.window, "状态未变化", "请选择不同于当前状态的目标状态。", walk.MsgBoxIconInformation)
		return
	}
	if stringIndex(statuses, status) < 0 {
		walk.MsgBox(ui.window, "状态无效", "请选择接口允许的非删除状态。", walk.MsgBoxIconWarning)
		return
	}
	if walk.MsgBox(ui.window, "确认变更状态", fmt.Sprintf("%s：%s\r\n当前状态：%s\r\n目标状态：%s\r\n\r\n是否提交到线上服务？", kind.Label, detail.Name, detail.Status, status), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state := ui.partner
	state.busy = true
	ui.updatePartnerActions()
	guardedGo(func() {
		updatedAt, found, requestErr := ui.partnerUpdatedAt(context.Background(), kind, detail.ID, detail.Code)
		if requestErr == nil && !found {
			requestErr = fmt.Errorf("资料已不存在")
		}
		if requestErr == nil && detail.UpdatedAt > 0 && updatedAt != detail.UpdatedAt {
			requestErr = fmt.Errorf("资料已被其他用户更新，请刷新后重试")
		}
		if requestErr == nil {
			switch kind.Key {
			case "supplier":
				requestErr = ui.session.Client.UpdateSupplierStatus(context.Background(), detail.ID, status)
			case "customer":
				requestErr = ui.session.Client.UpdateCustomerStatus(context.Background(), detail.ID, status)
			case "carrier":
				requestErr = ui.session.Client.UpdateCarrierStatus(context.Background(), detail.ID, status)
			}
		}
		if requestErr == nil {
			requestErr = ui.verifyPartnerStatus(context.Background(), kind, detail.ID, detail.Code, status)
		}
		ui.window.Synchronize(func() {
			if state != ui.partner {
				return
			}
			state.busy = false
			ui.updatePartnerActions()
			if requestErr != nil {
				state.info.SetText("状态变更失败或复核失败：" + requestErr.Error() + "。请刷新后核对，勿重复提交。")
				walk.MsgBox(ui.window, "状态变更未确认", state.info.Text(), walk.MsgBoxIconWarning)
				return
			}
			ui.notifyStatusSuccess("合作伙伴状态已变更，并完成线上回读复核。")
			ui.loadPartners()
			ui.loadFilterOptions()
		})
	})
}

func (ui *mainUI) verifyPartnerStatus(ctx context.Context, kind partnerKind, id, code, status string) error {
	switch kind.Key {
	case "supplier":
		item, found, err := ui.session.Client.FindSupplier(ctx, id, code)
		if err != nil || !found {
			if err != nil {
				return err
			}
			return fmt.Errorf("回读时未找到资料")
		}
		if item.Status != status {
			return fmt.Errorf("回读状态为 %s", item.Status)
		}
	case "customer":
		item, found, err := ui.session.Client.FindCustomer(ctx, id, code)
		if err != nil || !found {
			if err != nil {
				return err
			}
			return fmt.Errorf("回读时未找到资料")
		}
		if item.Status != status {
			return fmt.Errorf("回读状态为 %s", item.Status)
		}
	case "carrier":
		item, found, err := ui.session.Client.FindCarrier(ctx, id, code)
		if err != nil || !found {
			if err != nil {
				return err
			}
			return fmt.Errorf("回读时未找到资料")
		}
		if item.Status != status {
			return fmt.Errorf("回读状态为 %s", item.Status)
		}
	}
	return nil
}
