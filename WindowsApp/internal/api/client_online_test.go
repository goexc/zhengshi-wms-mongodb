package api

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOnlineReadOnlyWindowsContracts(t *testing.T) {
	if os.Getenv("WMS_ONLINE_TEST") != "1" {
		t.Skip("set WMS_ONLINE_TEST=1 to run read-only online contract checks")
	}
	baseURL := strings.TrimSpace(os.Getenv("WMS_ONLINE_BASE_URL"))
	mobile := strings.TrimSpace(os.Getenv("WMS_ONLINE_MOBILE"))
	password := os.Getenv("WMS_ONLINE_PASSWORD")
	if baseURL == "" || mobile == "" || password == "" {
		t.Fatal("WMS_ONLINE_BASE_URL, WMS_ONLINE_MOBILE and WMS_ONLINE_PASSWORD are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := NewClient(baseURL)
	login, err := client.Login(ctx, mobile, password)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(login.Token)
	t.Cleanup(func() {
		logoutCtx, logoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer logoutCancel()
		_ = client.Logout(logoutCtx)
	})

	if _, err = client.Profile(ctx); err != nil {
		t.Fatal(err)
	}
	perms, err := client.Permissions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms.Menus) == 0 {
		t.Fatal("online menu is empty")
	}
	if onlineHasMenuPath(perms.Menus, "/acl/user") {
		if _, err = client.AdminUsers(ctx, 1, 20, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if onlineHasMenuPath(perms.Menus, "/acl/department") {
		if _, err = client.AdminDepartments(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if onlineHasMenuPath(perms.Menus, "/acl/role") {
		if _, err = client.AdminRoles(ctx, 1, 20, ""); err != nil {
			t.Fatal(err)
		}
		if _, err = client.AdminRoleList(ctx, ""); err != nil {
			t.Fatal(err)
		}
	}
	if onlineHasMenuPath(perms.Menus, "/acl/menu") {
		if _, err = client.AdminMenus(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if onlineHasMenuPath(perms.Menus, "/acl/api") {
		if _, err = client.AdminAPIs(ctx); err != nil {
			t.Fatal(err)
		}
	}
	materials, err := client.Materials(ctx, 1, 100, MaterialFilters{})
	if err != nil {
		t.Fatal(err)
	}
	var drawingReference string
	var firstMaterialID string
	var withDrawing, withoutDrawing int
	for _, material := range materials.List {
		if firstMaterialID == "" {
			firstMaterialID = material.ID
		}
		if strings.TrimSpace(material.Image) == "" {
			withoutDrawing++
			continue
		}
		withDrawing++
		if drawingReference == "" {
			drawingReference = material.Image
		}
	}
	if withDrawing == 0 || withoutDrawing == 0 {
		t.Fatalf("online material drawing sample is incomplete: with=%d without=%d", withDrawing, withoutDrawing)
	}
	if _, err = client.MaterialCategories(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = client.ImageAssets(ctx, 1, 20, ""); err != nil {
		t.Fatal(err)
	}
	imageURL, err := ResolveImageURL(os.Getenv("WMS_ONLINE_IMAGE_BASE_URL"), drawingReference)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.DownloadImage(ctx, imageURL); err != nil {
		t.Fatal(err)
	}
	if firstMaterialID != "" {
		if _, err = client.MaterialPrices(ctx, firstMaterialID, ""); err != nil {
			t.Fatal(err)
		}
	}
	inbound, err := client.InboundReceipts(ctx, 1, 5, InboundFilters{})
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"待审核", "审核不通过", "部分入库"} {
		if _, err = client.InboundReceipts(ctx, 1, 5, InboundFilters{Status: status}); err != nil {
			t.Fatalf("inbound dashboard status %q: %v", status, err)
		}
	}
	if len(inbound.List) > 0 {
		if _, found, findErr := client.FindInboundReceipt(ctx, inbound.List[0].ID, inbound.List[0].Code); findErr != nil || !found {
			t.Fatalf("find inbound receipt: found=%v err=%v", found, findErr)
		}
		if _, err = client.InboundRecords(ctx, inbound.List[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = client.OutboundOrders(ctx, 1, 5, OutboundFilters{IsPack: -1, IsWeigh: -1}); err != nil {
		t.Fatal(err)
	}
	if len(materials.List) > 0 && strings.TrimSpace(materials.List[0].Model) != "" {
		if _, err = client.OutboundOrders(ctx, 1, 5, OutboundFilters{Model: materials.List[0].Model, IsPack: -1, IsWeigh: -1}); err != nil {
			t.Fatal(err)
		}
	}
	for _, status := range []string{"预发货", "待拣货", "待打包", "待称重", "已出库"} {
		if _, err = client.OutboundOrders(ctx, 1, 10, OutboundFilters{Status: status, IsPack: -1, IsWeigh: -1}); err != nil {
			t.Fatalf("outbound dashboard status %q: %v", status, err)
		}
	}
	if _, err = client.InventoryHistory(ctx, 1, 5, InventoryFilters{}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Carriers(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Customers(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Suppliers(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = client.WarehouseTree(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = client.SupplierDirectory(ctx, 1, 5, PartnerFilters{
		Email: "audit@example.com", Level: 1,
	}); err != nil {
		t.Fatal(err)
	}
	customersPage, err := client.CustomerDirectory(ctx, 1, 5, PartnerFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(customersPage.List) > 0 {
		deliveries, deliveryErr := client.NewCustomerMaterials(ctx, 1, 5, NewCustomerMaterialFilters{
			CustomerID: customersPage.List[0].ID,
			StartTime:  time.Now().AddDate(-10, 0, 0).Unix(),
			EndTime:    time.Now().Unix(),
		})
		if deliveryErr != nil {
			t.Fatal(deliveryErr)
		}
		if len(deliveries.List) > 0 {
			if _, err = client.MaterialQuotes(ctx, 1, 5, MaterialQuoteFilters{DeliveryID: deliveries.List[0].ID}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err = client.CustomerDirectory(ctx, 1, 5, PartnerFilters{
		Email: "audit@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.CarrierDirectory(ctx, 1, 5, PartnerFilters{
		Email: "audit@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Warehouses(ctx, 1, 5, WarehouseFilters{Type: "生产仓库"}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.WarehouseZones(ctx, 1, 5, WarehouseFilters{}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.WarehouseRacks(ctx, 1, 5, WarehouseFilters{Type: "标准货架"}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.WarehouseBins(ctx, 1, 5, WarehouseFilters{}); err != nil {
		t.Fatal(err)
	}
	if len(customersPage.List) > 0 {
		if _, err = client.CustomerTransactions(ctx, customersPage.List[0].ID, 1, 20); err != nil {
			t.Fatal(err)
		}
		end := time.Now().Unix()
		start := time.Now().AddDate(-1, 0, 0).Unix()
		if _, err = client.OutboundSummary(ctx, customersPage.List[0].ID, start, end); err != nil {
			t.Fatal(err)
		}
	}
}

func onlineHasMenuPath(menus []Menu, target string) bool {
	target = strings.ToLower(strings.TrimRight(strings.TrimSpace(target), "/"))
	for _, menu := range menus {
		path := strings.ToLower(strings.TrimRight(strings.TrimSpace(menu.Path), "/"))
		if path == target || onlineHasMenuPath(menu.Children, target) {
			return true
		}
	}
	return false
}

func TestOnlineInboundLifecycleMutation(t *testing.T) {
	if os.Getenv("WMS_ONLINE_MUTATION") != "1" {
		t.Skip("set WMS_ONLINE_MUTATION=1 to run the disposable inbound lifecycle mutation")
	}
	baseURL := strings.TrimSpace(os.Getenv("WMS_ONLINE_BASE_URL"))
	mobile := strings.TrimSpace(os.Getenv("WMS_ONLINE_MOBILE"))
	password := os.Getenv("WMS_ONLINE_PASSWORD")
	if baseURL == "" || mobile == "" || password == "" {
		t.Fatal("WMS_ONLINE_BASE_URL, WMS_ONLINE_MOBILE and WMS_ONLINE_PASSWORD are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := NewClient(baseURL)
	login, err := client.Login(ctx, mobile, password)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(login.Token)
	t.Cleanup(func() {
		logoutCtx, logoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer logoutCancel()
		_ = client.Logout(logoutCtx)
	})

	materials, err := client.Materials(ctx, 1, 5, MaterialFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(materials.List) == 0 {
		t.Fatal("online material list is empty")
	}
	code := "WIN-V07-TEST-" + time.Now().Format("20060102150405.000000000")
	request := InboundReceiptRequest{
		Code: code, Type: "生产入库", ReceivingDate: time.Now().AddDate(0, 0, 1).Unix(),
		Materials: []InboundMaterialRequest{{
			Index: 1, ID: materials.List[0].ID, Price: 1.25, EstimatedQuantity: 1,
		}},
		Remark: "WindowsApp v0.7 线上接口联调测试；测试完成后自动删除",
	}
	if err = client.CreateInboundReceipt(ctx, request); err != nil {
		t.Fatal(err)
	}

	page, err := client.InboundReceipts(ctx, 1, 100, InboundFilters{Code: code})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.List) != 1 {
		t.Fatalf("created inbound receipt count = %d", len(page.List))
	}
	created := page.List[0]
	deleted := false
	t.Cleanup(func() {
		if deleted || created.ID == "" {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		_ = client.DeleteInboundReceipt(cleanupCtx, created.ID)
	})
	if created.Status != "待审核" || created.Type != "生产入库" {
		t.Fatalf("created receipt = %#v", created)
	}

	request.ID = created.ID
	request.Materials[0].EstimatedQuantity = 2
	request.Remark = "WindowsApp v0.7 线上接口联调测试；已完成更新，稍后自动删除"
	if err = client.UpdateInboundReceipt(ctx, request); err != nil {
		t.Fatal(err)
	}
	updated, found, err := client.FindInboundReceipt(ctx, created.ID, code)
	if err != nil || !found {
		t.Fatalf("updated receipt: found=%v err=%v", found, err)
	}
	if len(updated.Materials) != 1 || updated.Materials[0].EstimatedQuantity != 2 || updated.Remark != request.Remark {
		t.Fatalf("updated receipt = %#v", updated)
	}

	if err = client.CheckInboundReceipt(ctx, created.ID, "审核不通过"); err != nil {
		t.Fatal(err)
	}
	rejected, found, err := client.FindInboundReceipt(ctx, created.ID, code)
	if err != nil || !found || rejected.Status != "审核不通过" {
		t.Fatalf("rejected receipt: found=%v status=%q err=%v", found, rejected.Status, err)
	}

	if err = client.DeleteInboundReceipt(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	deleted = true
	if _, found, err = client.FindInboundReceipt(ctx, created.ID, code); err != nil || found {
		t.Fatalf("deleted receipt still exists: found=%v err=%v", found, err)
	}
}

func TestOnlineOutboundCreateDeleteMutation(t *testing.T) {
	if os.Getenv("WMS_ONLINE_OUTBOUND_MUTATION") != "1" {
		t.Skip("set WMS_ONLINE_OUTBOUND_MUTATION=1 to run the disposable outbound create/delete mutation")
	}
	baseURL := strings.TrimSpace(os.Getenv("WMS_ONLINE_BASE_URL"))
	mobile := strings.TrimSpace(os.Getenv("WMS_ONLINE_MOBILE"))
	password := os.Getenv("WMS_ONLINE_PASSWORD")
	if baseURL == "" || mobile == "" || password == "" {
		t.Fatal("WMS_ONLINE_BASE_URL, WMS_ONLINE_MOBILE and WMS_ONLINE_PASSWORD are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := NewClient(baseURL)
	login, err := client.Login(ctx, mobile, password)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(login.Token)
	t.Cleanup(func() {
		logoutCtx, logoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer logoutCancel()
		_ = client.Logout(logoutCtx)
	})

	materials, err := client.Materials(ctx, 1, 5, MaterialFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(materials.List) == 0 {
		t.Fatal("online material list is empty")
	}
	code := "WIN-V10-OUT-" + time.Now().Format("20060102150405.000000000")
	request := OutboundOrderRequest{
		Code: code, Type: "报废出库",
		Materials: []OutboundMaterialRequest{{Index: 1, MaterialID: materials.List[0].ID, Price: 0, Quantity: 1}},
		Remark:    "WindowsApp v1.0 线上接口联调测试；零单价且测试完成后自动删除",
	}
	if err = client.CreateOutbound(ctx, request); err != nil {
		t.Fatal(err)
	}
	created, found, err := client.FindOutbound(ctx, "", code)
	if err != nil || !found {
		t.Fatalf("created outbound: found=%v err=%v", found, err)
	}
	deleted := false
	t.Cleanup(func() {
		if deleted || created.ID == "" {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		_ = client.DeleteOutbound(cleanupCtx, created.ID)
	})
	if created.Status != "预发货" || created.Type != "报废出库" || created.TotalAmount != 0 {
		t.Fatalf("created outbound = %#v", created)
	}
	if err = client.DeleteOutbound(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	deleted = true
	if _, found, err = client.FindOutbound(ctx, created.ID, code); err != nil || found {
		t.Fatalf("deleted outbound still exists: found=%v err=%v", found, err)
	}
}

func TestOnlineSupplierLifecycleMutation(t *testing.T) {
	if os.Getenv("WMS_ONLINE_MASTER_MUTATION") != "1" {
		t.Skip("set WMS_ONLINE_MASTER_MUTATION=1 to run the disposable supplier lifecycle mutation")
	}
	baseURL := strings.TrimSpace(os.Getenv("WMS_ONLINE_BASE_URL"))
	mobile := strings.TrimSpace(os.Getenv("WMS_ONLINE_MOBILE"))
	password := os.Getenv("WMS_ONLINE_PASSWORD")
	if baseURL == "" || mobile == "" || password == "" {
		t.Fatal("WMS_ONLINE_BASE_URL, WMS_ONLINE_MOBILE and WMS_ONLINE_PASSWORD are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client := NewClient(baseURL)
	login, err := client.Login(ctx, mobile, password)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(login.Token)
	t.Cleanup(func() {
		logoutCtx, logoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer logoutCancel()
		_ = client.Logout(logoutCtx)
	})

	perms, err := client.Permissions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, permission := range []string{"business_partner:supplier:add", "business_partner:supplier:edit", "business_partner:supplier:status"} {
		if !onlineHasButton(perms.Buttons, permission) {
			t.Skipf("online account lacks %s; no supplier mutation was sent", permission)
		}
	}

	stamp := time.Now().Format("060102150405000")
	code := "WAPP08SUP" + stamp
	name := "WindowsApp联调供应商" + stamp
	request := SupplierRequest{
		Type: "企业", Level: 1, Code: code, Name: name,
		LegalRepresentative: "WindowsApp联调", UnifiedSocialCreditIdentifier: "W" + stamp,
		Contact: mobile, Manager: "WindowsApp联调", Remark: "WindowsApp v0.8 线上接口联调；测试完成后自动删除",
	}
	if err = client.CreateSupplier(ctx, request); err != nil {
		t.Fatal(err)
	}

	var created Supplier
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		if created.ID == "" {
			page, findErr := client.SupplierDirectory(cleanupCtx, 1, 100, PartnerFilters{Code: code})
			if findErr == nil {
				for _, item := range page.List {
					if item.Code == code {
						created = item
						break
					}
				}
			}
		}
		if created.ID != "" {
			if cleanupErr := client.UpdateSupplierStatus(cleanupCtx, created.ID, "删除"); cleanupErr != nil {
				t.Errorf("cleanup disposable supplier %s: %v", code, cleanupErr)
			}
		}
	})

	page, err := client.SupplierDirectory(ctx, 1, 100, PartnerFilters{Code: code})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.List {
		if item.Code == code {
			created = item
			break
		}
	}
	if created.ID == "" {
		t.Fatalf("created supplier %s was not returned by the list endpoint", code)
	}

	request.ID = created.ID
	request.Remark = "WindowsApp v0.8 线上接口联调；已完成更新，稍后自动删除"
	if err = client.UpdateSupplier(ctx, request); err != nil {
		t.Fatal(err)
	}
	updated, found, err := client.FindSupplier(ctx, created.ID, code)
	if err != nil || !found || updated.Remark != request.Remark {
		t.Fatalf("updated supplier: found=%v remark=%q err=%v", found, updated.Remark, err)
	}

	if err = client.UpdateSupplierStatus(ctx, created.ID, "停用"); err != nil {
		t.Fatal(err)
	}
	stopped, found, err := client.FindSupplier(ctx, created.ID, code)
	if err != nil || !found || stopped.Status != "停用" {
		t.Fatalf("stopped supplier: found=%v status=%q err=%v", found, stopped.Status, err)
	}

	if err = client.UpdateSupplierStatus(ctx, created.ID, "删除"); err != nil {
		t.Fatal(err)
	}
	created.ID = ""
	if _, found, err = client.FindSupplier(ctx, stopped.ID, code); err != nil || found {
		t.Fatalf("deleted supplier still exists: found=%v err=%v", found, err)
	}
}

func onlineHasButton(buttons []Button, permission string) bool {
	for _, button := range buttons {
		if strings.TrimSpace(button.Perms) == permission {
			return true
		}
	}
	return false
}
