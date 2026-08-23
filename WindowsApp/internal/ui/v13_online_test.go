package ui

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestOnlineV13ReadOnlyLookupAndDocuments(t *testing.T) {
	if os.Getenv("WMS_ONLINE_TEST") != "1" {
		t.Skip("set WMS_ONLINE_TEST=1 to run read-only Windows client online checks")
	}
	baseURL := strings.TrimSpace(os.Getenv("WMS_ONLINE_BASE_URL"))
	mobile := strings.TrimSpace(os.Getenv("WMS_ONLINE_MOBILE"))
	password := os.Getenv("WMS_ONLINE_PASSWORD")
	imageBaseURL := strings.TrimSpace(os.Getenv("WMS_ONLINE_IMAGE_BASE_URL"))
	if baseURL == "" || mobile == "" || password == "" || imageBaseURL == "" {
		t.Fatal("online read-only environment is incomplete")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := api.NewClient(baseURL)
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

	materials, err := client.Materials(ctx, 1, 100, api.MaterialFilters{})
	if err != nil {
		t.Fatal(err)
	}
	for _, material := range materials.List {
		if strings.TrimSpace(material.Image) == "" {
			continue
		}
		document, buildErr := buildOnlineDocument(ctx, client, imageBaseURL, documentMaterialDrawing, material.ID)
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		if len(document.ImageData) == 0 {
			t.Fatal("material drawing document has no image data")
		}
		results, searchErr := searchGlobalLookupKind(ctx, client, "material", material.Model)
		if searchErr != nil || len(results) == 0 {
			t.Fatalf("material lookup: results=%d err=%v", len(results), searchErr)
		}
		break
	}

	inbound, err := client.InboundReceipts(ctx, 1, 5, api.InboundFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(inbound.List) > 0 {
		code := inbound.List[0].Code
		if _, buildErr := buildOnlineDocument(ctx, client, imageBaseURL, documentInboundChecklist, code); buildErr != nil {
			t.Fatal(buildErr)
		}
		if results, searchErr := searchGlobalLookupKind(ctx, client, "inbound", code); searchErr != nil || len(results) != 1 {
			t.Fatalf("inbound lookup: results=%d err=%v", len(results), searchErr)
		}
	}

	outbound, err := client.OutboundOrders(ctx, 1, 5, api.OutboundFilters{IsPack: -1, IsWeigh: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(outbound.List) > 0 {
		code := outbound.List[0].Code
		if _, buildErr := buildOnlineDocument(ctx, client, imageBaseURL, documentOutboundList, code); buildErr != nil {
			t.Fatal(buildErr)
		}
		if results, searchErr := searchGlobalLookupKind(ctx, client, "outbound", code); searchErr != nil || len(results) != 1 {
			t.Fatalf("outbound lookup: results=%d err=%v", len(results), searchErr)
		}
	}
}
