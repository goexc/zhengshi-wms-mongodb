package ui

func (ui *mainUI) currentPageKey() string {
	if ui.tabs == nil {
		return ""
	}
	index := ui.tabs.CurrentIndex()
	if index < 0 || index >= ui.tabs.Pages().Len() {
		return ""
	}
	return ui.keyForTab(ui.tabs.Pages().At(index))
}

func (ui *mainUI) refreshCurrentPage() {
	switch ui.currentPageKey() {
	case "material":
		ui.loadMaterials()
	case "inventory":
		ui.loadInventory()
	case "inbound":
		ui.loadInbound()
	case "outbound":
		ui.loadOutbound()
	case "outbound_report":
		ui.refreshOutboundReportPage()
	case "partner":
		ui.loadPartners()
	case "warehouse":
		ui.loadWarehouseTree()
		ui.loadWarehouseDirectory()
	case "system":
		if ui.status != nil {
			ui.status.SetText("系统信息已是最新登录会话数据。")
		}
	}
}

func (ui *mainUI) focusCurrentPageSearch() {
	switch ui.currentPageKey() {
	case "material":
		if ui.materialName != nil {
			ui.materialName.SetFocus()
		}
	case "inventory":
		if ui.inventoryName != nil {
			ui.inventoryName.SetFocus()
		}
	case "inbound":
		if ui.inboundSearch != nil {
			ui.inboundSearch.SetFocus()
		}
	case "outbound":
		if ui.outbound != nil && ui.outbound.search != nil {
			ui.outbound.search.SetFocus()
		}
	case "outbound_report":
		if ui.outboundReport != nil && ui.outboundReport.customer != nil {
			ui.outboundReport.customer.SetFocus()
		}
	case "partner":
		if ui.partner != nil && ui.partner.name != nil {
			ui.partner.name.SetFocus()
		}
	case "warehouse":
		if ui.warehouse != nil && ui.warehouse.name != nil {
			ui.warehouse.name.SetFocus()
		}
	}
}
