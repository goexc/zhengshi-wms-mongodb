package ui

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

const (
	documentInboundChecklist = "inbound_checklist"
	documentOutboundList     = "outbound_list"
	documentOutboundReceipt  = "outbound_receipt"
	documentMaterialDrawing  = "material_drawing"
	documentInboundLabel     = "inbound_label"
	documentOutboundLabel    = "outbound_label"
)

var documentKindLabels = []string{
	"入库收货核对单",
	"出库物料清单",
	"出库签收摘要",
	"物料图纸",
	"入库单号标签",
	"出库单号标签",
}

var documentKindKeys = []string{
	documentInboundChecklist,
	documentOutboundList,
	documentOutboundReceipt,
	documentMaterialDrawing,
	documentInboundLabel,
	documentOutboundLabel,
}

type printDocument struct {
	Kind      string
	Key       string
	Title     string
	Lines     []string
	ImageData []byte
	LabelCode string
	Material  api.Material
}

type documentCenterUI struct {
	kind       *walk.ComboBox
	key        *walk.LineEdit
	load       *walk.PushButton
	print      *walk.PushButton
	image      *walk.PushButton
	preview    *walk.TextEdit
	info       *walk.Label
	document   *printDocument
	kindKeys   []string
	generation int
	cancel     context.CancelFunc
	busy       bool
}

func newDocumentCenterUI() *documentCenterUI { return &documentCenterUI{} }

func documentMenuAvailable(perms api.Perms) bool {
	return hasAnyMenuPath(perms.Menus, "/material/list", "/inbound/receipt", "/outbound/receipt", "/outbound/receipt2")
}

func availableDocumentKinds(perms api.Perms) ([]string, []string) {
	labels := make([]string, 0, len(documentKindLabels))
	keys := make([]string, 0, len(documentKindKeys))
	hasInbound := hasMenuPath(perms.Menus, "/inbound/receipt")
	hasOutbound := hasAnyMenuPath(perms.Menus, "/outbound/receipt", "/outbound/receipt2")
	if hasInbound {
		labels = append(labels, documentKindLabels[0])
		keys = append(keys, documentInboundChecklist)
	}
	if hasOutbound {
		labels = append(labels, documentKindLabels[1], documentKindLabels[2])
		keys = append(keys, documentOutboundList, documentOutboundReceipt)
	}
	if hasMenuPath(perms.Menus, "/material/list") {
		labels = append(labels, documentKindLabels[3])
		keys = append(keys, documentMaterialDrawing)
	}
	if hasInbound {
		labels = append(labels, documentKindLabels[4])
		keys = append(keys, documentInboundLabel)
	}
	if hasOutbound {
		labels = append(labels, documentKindLabels[5])
		keys = append(keys, documentOutboundLabel)
	}
	return labels, keys
}

func (ui *mainUI) documentCenterPageWidget() TabPage {
	if ui.documents == nil {
		ui.documents = newDocumentCenterUI()
	}
	state := ui.documents
	labels, keys := availableDocumentKinds(ui.session.Perms)
	state.kindKeys = keys
	return TabPage{
		AssignTo: &ui.documentTab,
		Title:    closableTabTitle("单据预览与打印"),
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "单据预览与受控打印", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "预览和打印都从现有线上接口读取；点击打印时会再次回读，失败会清空旧预览并停止打印。", TextColor: secondaryTextColor()},
			GroupBox{Title: "文档条件", Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10}, Children: []Widget{
				Label{Text: "文档类型"},
				ComboBox{AssignTo: &state.kind, Model: labels, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 30}, Accessibility: Accessibility{Name: "需要预览和打印的文档类型"}},
				Label{Text: "业务标识"},
				LineEdit{AssignTo: &state.key, MinSize: Size{Width: 300, Height: 30}, CueBanner: "单号，或物料 ID/型号/名称", Accessibility: Accessibility{Name: "打印文档业务标识"}},
				Composite{ColumnSpan: 4, Layout: HBox{Spacing: 8}, Children: []Widget{
					Label{Text: "单号标签按所选单据类型在线核对；物料图纸优先按 ID，其次按精确型号或名称。", TextColor: secondaryTextColor()},
					HSpacer{},
					PushButton{AssignTo: &state.load, Text: "在线生成预览", MinSize: Size{Width: 116, Height: 32}, OnClicked: func() { ui.loadSelectedDocument(false) }},
				}},
			}},
			GroupBox{Title: "只读预览", StretchFactor: 1, Layout: VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
				TextEdit{AssignTo: &state.preview, ReadOnly: true, StretchFactor: 1, Text: "尚未生成预览。", Accessibility: Accessibility{Name: "当前待打印文档的只读预览"}},
			}},
			Label{AssignTo: &state.info, Text: "请选择文档类型并输入业务标识。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "文档生成与打印状态"}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.image, Text: "查看图纸原图", Enabled: false, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.previewDocumentDrawing},
				HSpacer{},
				PushButton{AssignTo: &state.print, Text: "重新回读并打印", Enabled: false, MinSize: Size{Width: 132, Height: 32}, OnClicked: ui.printSelectedDocument},
			}},
		},
	}
}

func (ui *mainUI) initializeDocumentCenterPage() {
	state := ui.documents
	if state == nil || state.key == nil {
		return
	}
	state.key.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			ui.loadSelectedDocument(false)
		}
	})
	state.kind.CurrentIndexChanged().Attach(func() {
		if state.busy {
			return
		}
		state.document = nil
		state.preview.SetText("文档类型已变化，请重新生成线上预览。")
		state.info.SetText("请核对业务标识后重新生成预览。")
		ui.updateDocumentCenterActions()
	})
	ui.updateDocumentCenterActions()
}

func (ui *mainUI) releaseDocumentCenterPage() {
	if ui.documents != nil && ui.documents.cancel != nil {
		ui.documents.cancel()
	}
	ui.documentTab = nil
	ui.documents = newDocumentCenterUI()
}

func (ui *mainUI) selectedDocumentKind() string {
	state := ui.documents
	if state == nil || state.kind == nil {
		return ""
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kindKeys) {
		return ""
	}
	return state.kindKeys[index]
}

func (ui *mainUI) loadSelectedDocument(printAfterLoad bool) {
	state := ui.documents
	if state == nil || state.busy {
		return
	}
	kind := ui.selectedDocumentKind()
	key := strings.TrimSpace(state.key.Text())
	if kind == "" || key == "" {
		state.info.SetText("请选择文档类型并输入业务标识。")
		state.key.SetFocus()
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	state.busy = true
	state.document = nil
	state.preview.SetText("正在从线上重新读取文档数据……")
	state.info.SetText("正在生成最新线上预览……")
	ui.updateDocumentCenterActions()
	guardedGo(func() {
		document, err := buildOnlineDocument(ctx, ui.session.Client, config.ImageBaseURL(), kind, key)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != state.generation || state != ui.documents || state.preview == nil {
				return
			}
			state.busy = false
			if err != nil {
				state.document = nil
				state.preview.SetText("预览已清空。")
				state.info.SetText("线上读取失败：" + err.Error() + "。未保留旧预览，也不会继续打印。")
				ui.updateDocumentCenterActions()
				return
			}
			state.document = document
			state.preview.SetText(documentPreviewText(document))
			state.info.SetText("已生成最新线上预览。打印时仍会再次回读。")
			ui.updateDocumentCenterActions()
			if printAfterLoad {
				printed, printErr := printNativeDocument(ui.window.Handle(), document)
				if printErr != nil {
					state.info.SetText("打印失败：" + printErr.Error())
				} else if printed {
					state.info.SetText("文档已提交到所选 Windows 打印机。")
				} else {
					state.info.SetText("已取消打印；线上预览保持不变。")
				}
			}
		})
	})
}

func (ui *mainUI) updateDocumentCenterActions() {
	state := ui.documents
	if state == nil || state.print == nil {
		return
	}
	ready := state.document != nil && !state.busy
	state.load.SetEnabled(!state.busy)
	state.kind.SetEnabled(!state.busy)
	state.key.SetEnabled(!state.busy)
	state.print.SetEnabled(ready)
	state.image.SetEnabled(ready && state.document.Kind == documentMaterialDrawing && hasMaterialDrawing(state.document.Material))
}

func (ui *mainUI) printSelectedDocument() {
	state := ui.documents
	if state == nil || state.document == nil || state.busy {
		return
	}
	ui.loadSelectedDocument(true)
}

func (ui *mainUI) previewDocumentDrawing() {
	state := ui.documents
	if state == nil || state.document == nil || state.document.Kind != documentMaterialDrawing {
		return
	}
	ShowMaterialDetail(ui.window, ui.session.Client, config.ImageBaseURL(), state.document.Material)
}

func buildOnlineDocument(ctx context.Context, client *api.Client, imageBaseURL, kind, key string) (*printDocument, error) {
	switch kind {
	case documentInboundChecklist:
		receipt, found, err := client.FindInboundReceipt(ctx, "", key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("未找到入库单 %s", key)
		}
		records, err := client.InboundRecords(ctx, receipt.ID)
		if err != nil {
			return nil, err
		}
		return inboundChecklistDocument(receipt, records), nil
	case documentOutboundList, documentOutboundReceipt:
		order, found, err := client.FindOutbound(ctx, "", key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("未找到出库单 %s", key)
		}
		materials, err := client.OutboundMaterials(ctx, order.Code)
		if err != nil {
			return nil, err
		}
		if kind == documentOutboundReceipt {
			return outboundReceiptDocument(order, materials), nil
		}
		return outboundListDocument(order, materials), nil
	case documentMaterialDrawing:
		material, err := findMaterialForDocument(ctx, client, key)
		if err != nil {
			return nil, err
		}
		if !hasMaterialDrawing(material) {
			return nil, fmt.Errorf("物料 %s / %s 没有图纸", material.Name, material.Model)
		}
		imageURL, err := api.ResolveImageURL(imageBaseURL, material.Image)
		if err != nil {
			return nil, err
		}
		data, err := client.DownloadImage(ctx, imageURL)
		if err != nil {
			return nil, err
		}
		return materialDrawingDocument(material, data), nil
	case documentInboundLabel:
		receipt, found, err := client.FindInboundReceipt(ctx, "", key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("未找到入库单 %s", key)
		}
		return orderLabelDocument("入库单", receipt.Code, receipt.Type, receipt.Status), nil
	case documentOutboundLabel:
		order, found, err := client.FindOutbound(ctx, "", key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("未找到出库单 %s", key)
		}
		return orderLabelDocument("出库单", order.Code, order.Type, order.Status), nil
	default:
		return nil, fmt.Errorf("不支持的文档类型")
	}
}

func inboundChecklistDocument(receipt api.InboundReceipt, records []api.InboundRecord) *printDocument {
	business := receipt.SupplierName
	if strings.TrimSpace(business) == "" {
		business = receipt.CustomerName
	}
	lines := []string{
		"单号：" + receipt.Code,
		"类型：" + receipt.Type + "    状态：" + receipt.Status,
		"供应商/客户：" + displayMaterialValue(business),
		"计划收货日期：" + formatDocumentDate(receipt.ReceivingDate),
		fmt.Sprintf("已记录收货批次：%d", len(records)),
		"",
		"物料核对",
	}
	for _, material := range receipt.Materials {
		remaining := material.EstimatedQuantity - material.ActualQuantity
		if remaining < 0 {
			remaining = 0
		}
		lines = append(lines, fmt.Sprintf("%d. %s / %s    计划 %g %s    已收 %g    剩余 %g    %s", material.Index+1, material.Name, displayMaterialValue(material.Model), material.EstimatedQuantity, material.Unit, material.ActualQuantity, remaining, material.Status))
	}
	lines = append(lines, "", "备注："+displayMaterialValue(receipt.Remark), "核对人：________________    日期：________________")
	return &printDocument{Kind: documentInboundChecklist, Key: receipt.Code, Title: "入库收货核对单", Lines: lines}
}

func outboundListDocument(order api.OutboundOrder, materials []api.OutboundMaterial) *printDocument {
	business := order.SupplierName
	if strings.TrimSpace(business) == "" {
		business = order.CustomerName
	}
	lines := []string{
		"单号：" + order.Code,
		"类型：" + order.Type + "    状态：" + order.Status,
		"供应商/客户：" + displayMaterialValue(business),
		"",
		"出库物料",
	}
	for _, material := range materials {
		lines = append(lines, fmt.Sprintf("%d. %s / %s / %s    %g %s    单价 %.3f    重量 %g", material.Index, material.Name, displayMaterialValue(material.Model), displayMaterialValue(material.Specification), material.Quantity, material.Unit, material.Price, material.Weight))
	}
	lines = append(lines, "", fmt.Sprintf("合计金额：%.2f", order.TotalAmount), "复核人：________________    日期：________________")
	return &printDocument{Kind: documentOutboundList, Key: order.Code, Title: "出库物料清单", Lines: lines}
}

func outboundReceiptDocument(order api.OutboundOrder, materials []api.OutboundMaterial) *printDocument {
	document := outboundListDocument(order, materials)
	document.Kind = documentOutboundReceipt
	document.Title = "出库签收摘要"
	document.Lines = append([]string{
		"单号：" + order.Code,
		"当前状态：" + order.Status,
		"出库时间：" + formatDocumentDateTime(order.DepartureTime),
		"签收时间：" + formatDocumentDateTime(order.ReceiptTime),
		fmt.Sprintf("签收附件：%d 张", len(order.Annex)),
		"承运商：" + displayMaterialValue(order.CarrierName),
		"",
	}, document.Lines[4:]...)
	return document
}

func materialDrawingDocument(material api.Material, data []byte) *printDocument {
	lines := []string{
		"物料名称：" + displayMaterialValue(material.Name),
		"型号：" + displayMaterialValue(material.Model),
		"分类：" + displayMaterialValue(material.CategoryName),
		"规格：" + displayMaterialValue(material.Specification),
		"材质：" + displayMaterialValue(material.Material),
		"表面处理：" + displayMaterialValue(material.SurfaceTreatment),
		"强度等级：" + displayMaterialValue(material.StrengthGrade),
		"备注：" + displayMaterialValue(material.Remark),
	}
	return &printDocument{Kind: documentMaterialDrawing, Key: material.ID, Title: "物料图纸", Lines: lines, ImageData: data, Material: material}
}

func orderLabelDocument(labelType, code, orderType, status string) *printDocument {
	kind := documentInboundLabel
	if labelType == "出库单" {
		kind = documentOutboundLabel
	}
	return &printDocument{
		Kind: kind, Key: code, Title: labelType + "单号标签", LabelCode: code,
		Lines: []string{"单据类型：" + labelType, "业务类型：" + orderType, "当前状态：" + status, "生成时间：" + time.Now().Format("2006-01-02 15:04:05")},
	}
}

func findMaterialForDocument(ctx context.Context, client *api.Client, key string) (api.Material, error) {
	key = strings.TrimSpace(key)
	if len(key) == 24 {
		if _, err := hex.DecodeString(key); err == nil {
			if material, requestErr := client.MaterialInfo(ctx, key); requestErr == nil {
				return material, nil
			}
		}
	}
	byModel, err := client.Materials(ctx, 1, 20, api.MaterialFilters{Model: key})
	if err != nil {
		return api.Material{}, err
	}
	byName, err := client.Materials(ctx, 1, 20, api.MaterialFilters{Name: key})
	if err != nil {
		return api.Material{}, err
	}
	candidates := append(byModel.List, byName.List...)
	seen := map[string]bool{}
	unique := make([]api.Material, 0, len(candidates))
	for _, material := range candidates {
		if seen[material.ID] {
			continue
		}
		seen[material.ID] = true
		if strings.EqualFold(strings.TrimSpace(material.Model), key) || strings.EqualFold(strings.TrimSpace(material.Name), key) {
			unique = append(unique, material)
		}
	}
	if len(unique) == 1 {
		return unique[0], nil
	}
	if len(unique) == 0 {
		return api.Material{}, fmt.Errorf("未找到精确匹配的物料 %s", key)
	}
	return api.Material{}, fmt.Errorf("物料标识 %s 精确匹配到 %d 项，请改用物料 ID", key, len(unique))
}

func documentPreviewText(document *printDocument) string {
	if document == nil {
		return "尚未生成预览。"
	}
	text := document.Title + "\r\n" + strings.Repeat("=", 36) + "\r\n" + strings.Join(document.Lines, "\r\n")
	if len(document.ImageData) > 0 {
		text += "\r\n\r\n[图纸图片已在线加载，点击“查看图纸原图”检查细节；打印时自动适应页面。]"
	}
	if document.LabelCode != "" {
		text = document.Title + "\r\n" + strings.Repeat("=", 36) + "\r\n\r\n" + document.LabelCode + "\r\n\r\n" + strings.Join(document.Lines, "\r\n")
	}
	return text
}

func formatDocumentDate(value int64) string {
	if value <= 0 {
		return "—"
	}
	return time.Unix(value, 0).Format("2006-01-02")
}

func formatDocumentDateTime(value int64) string {
	if value <= 0 {
		return "—"
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04:05")
}
