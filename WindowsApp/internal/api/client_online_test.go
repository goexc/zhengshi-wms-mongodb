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
	if len(inbound.List) > 0 {
		if _, err = client.InboundRecords(ctx, inbound.List[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = client.OutboundOrders(ctx, 1, 5, OutboundFilters{IsPack: -1, IsWeigh: -1}); err != nil {
		t.Fatal(err)
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
