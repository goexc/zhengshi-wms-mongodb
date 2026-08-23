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
	case "dashboard":
		ui.loadDashboard()
	case "global_lookup":
		ui.runGlobalLookup()
	case "operations":
		ui.refreshOperationCenter()
	case "documents":
		ui.loadSelectedDocument(false)
	case "material":
		ui.loadMaterials()
	case "material_category":
		ui.loadMaterialCategories()
	case "material_quote":
		ui.loadMaterialQuotes()
	case "image_assets":
		ui.loadImageAssets()
	case "material_editor":
		if ui.materialEditor != nil && ui.materialEditor.info != nil {
			ui.materialEditor.info.SetText("当前编辑内容尚未提交；保存时会重新读取线上状态。")
		}
	case "material_quote_editor":
		if ui.materialQuoteEditor != nil && ui.materialQuoteEditor.info != nil {
			ui.materialQuoteEditor.info.SetText("当前报价内容尚未提交；写操作前会重新读取线上状态。")
		}
	case "inventory":
		ui.loadInventory()
	case "inbound":
		ui.loadInbound()
	case "inbound_editor":
		if ui.inboundEditor != nil && ui.inboundEditor.info != nil {
			ui.inboundEditor.info.SetText("当前编辑内容尚未提交；保存时会重新读取线上状态。")
		}
	case "partner_editor":
		if ui.partnerEditor != nil && ui.partnerEditor.info != nil {
			ui.partnerEditor.info.SetText("当前编辑内容尚未提交；保存时会重新读取线上状态。")
		}
	case "warehouse_editor":
		if ui.warehouseEditor != nil && ui.warehouseEditor.info != nil {
			ui.warehouseEditor.info.SetText("当前编辑内容尚未提交；保存时会重新读取线上状态。")
		}
	case "outbound":
		ui.loadOutbound()
	case "outbound_editor":
		if ui.outboundEditor != nil && ui.outboundEditor.info != nil {
			ui.outboundEditor.info.SetText("当前编辑内容尚未提交；创建时会在线复核结果。")
		}
	case "fast_outbound":
		if ui.fastOutbound != nil && ui.fastOutbound.info != nil {
			ui.fastOutbound.info.SetText("当前极速出库内容尚未提交；提交后会按单号在线回读复核。")
		}
	case "outbound_report":
		ui.refreshOutboundReportPage()
	case "partner":
		ui.loadPartners()
	case "customer_finance":
		ui.loadCustomerFinanceTransactions()
	case "warehouse":
		ui.loadWarehouseTree()
		ui.loadWarehouseDirectory()
	case "admin":
		ui.refreshAdminPage()
	case "profile":
		if ui.status != nil {
			ui.status.SetText("个人资料已是最新登录会话数据。")
		}
	case "system":
		if ui.status != nil {
			ui.status.SetText("系统信息已是最新登录会话数据。")
		}
	}
	ui.loadFilterOptions()
}

func (ui *mainUI) focusCurrentPageSearch() {
	switch ui.currentPageKey() {
	case "dashboard":
		if ui.dashboard != nil && ui.dashboard.refresh != nil {
			ui.dashboard.refresh.SetFocus()
		}
	case "global_lookup":
		if ui.globalLookup != nil && ui.globalLookup.query != nil {
			ui.globalLookup.query.SetFocus()
		}
	case "operations":
		if ui.operations != nil && ui.operations.table != nil {
			ui.operations.table.SetFocus()
		}
	case "documents":
		if ui.documents != nil && ui.documents.key != nil {
			ui.documents.key.SetFocus()
		}
	case "material":
		if ui.materialName != nil {
			ui.materialName.SetFocus()
		}
	case "material_category":
		if ui.materialCategories != nil && ui.materialCategories.table != nil {
			ui.materialCategories.table.SetFocus()
		}
	case "material_quote":
		if ui.materialQuote != nil && ui.materialQuote.customer != nil {
			ui.materialQuote.customer.SetFocus()
		}
	case "image_assets":
		if ui.imageAssets != nil && ui.imageAssets.name != nil {
			ui.imageAssets.name.SetFocus()
		}
	case "material_editor":
		if ui.materialEditor != nil && ui.materialEditor.name != nil {
			ui.materialEditor.name.SetFocus()
		}
	case "material_quote_editor":
		if ui.materialQuoteEditor != nil && ui.materialQuoteEditor.quoteMode != nil {
			ui.materialQuoteEditor.quoteMode.SetFocus()
		}
	case "inventory":
		if ui.inventoryName != nil {
			ui.inventoryName.SetFocus()
		}
	case "inbound":
		if ui.inboundSearch != nil {
			ui.inboundSearch.SetFocus()
		}
	case "inbound_editor":
		if ui.inboundEditor != nil && ui.inboundEditor.code != nil {
			ui.inboundEditor.code.SetFocus()
		}
	case "partner_editor":
		if ui.partnerEditor != nil && ui.partnerEditor.name != nil {
			ui.partnerEditor.name.SetFocus()
		}
	case "warehouse_editor":
		if ui.warehouseEditor != nil && ui.warehouseEditor.name != nil {
			ui.warehouseEditor.name.SetFocus()
		}
	case "outbound":
		if ui.outbound != nil && ui.outbound.search != nil {
			ui.outbound.search.SetFocus()
		}
	case "outbound_editor":
		if ui.outboundEditor != nil && ui.outboundEditor.code != nil {
			ui.outboundEditor.code.SetFocus()
		}
	case "fast_outbound":
		if ui.fastOutbound != nil && ui.fastOutbound.code != nil {
			ui.fastOutbound.code.SetFocus()
		}
	case "outbound_report":
		if ui.outboundReport != nil && ui.outboundReport.customer != nil {
			ui.outboundReport.customer.SetFocus()
		}
	case "partner":
		if ui.partner != nil && ui.partner.name != nil {
			ui.partner.name.SetFocus()
		}
	case "customer_finance":
		if ui.customerFinance != nil && ui.customerFinance.table != nil {
			ui.customerFinance.table.SetFocus()
		}
	case "warehouse":
		if ui.warehouse != nil && ui.warehouse.name != nil {
			ui.warehouse.name.SetFocus()
		}
	case "admin":
		if ui.admin != nil && ui.admin.innerTabs != nil {
			page := ui.admin.innerTabs.CurrentIndex()
			if ui.admin.usersPage != nil && page == ui.admin.innerTabs.Pages().Index(ui.admin.usersPage) && ui.admin.userName != nil {
				ui.admin.userName.SetFocus()
			} else if ui.admin.rolesPage != nil && page == ui.admin.innerTabs.Pages().Index(ui.admin.rolesPage) && ui.admin.roleName != nil {
				ui.admin.roleName.SetFocus()
			} else if ui.admin.departmentTable != nil {
				ui.admin.departmentTable.SetFocus()
			}
		}
	case "profile":
		if ui.profileTab != nil {
			ui.profileTab.SetFocus()
		}
	}
}

func (ui *mainUI) saveCurrentEditor() {
	switch ui.currentPageKey() {
	case "material_editor":
		ui.submitMaterialEditor()
	case "material_quote_editor":
		ui.saveCurrentMaterialQuote()
	case "inbound_editor":
		ui.submitInboundEditor()
	case "outbound_editor":
		ui.submitOutboundEditor()
	case "fast_outbound":
		ui.submitFastOutbound()
	case "partner_editor":
		ui.submitPartnerEditor()
	case "warehouse_editor":
		ui.submitWarehouseEditor()
	default:
		if ui.status != nil {
			ui.status.SetText("当前页面不是可保存的编辑页。")
		}
	}
}

func (ui *mainUI) newCurrentRecord() {
	switch ui.currentPageKey() {
	case "material", "material_editor":
		ui.newMaterial()
	case "inbound", "inbound_editor":
		ui.newInboundReceipt()
	case "outbound":
		ui.newOutboundOrder()
	case "outbound_editor":
		ui.newOutboundOrder()
	case "fast_outbound":
		ui.openFastOutbound()
	case "partner", "partner_editor":
		ui.newPartner()
	case "customer_finance":
		ui.addCustomerFinanceTransaction()
	case "warehouse", "warehouse_editor":
		ui.newWarehouseEntity()
	default:
		if ui.status != nil {
			ui.status.SetText("当前模块没有可新建的业务单据。")
		}
	}
}
