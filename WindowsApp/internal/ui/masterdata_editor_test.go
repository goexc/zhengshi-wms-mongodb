package ui

import (
	"strings"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestPartnerEditorValidationMatchesWebFields(t *testing.T) {
	valid := partnerEditorValues{
		Type: "企业", Name: "测试供应商", Code: "SUP001", LegalRepresentative: "测试负责人",
		UnifiedSocialCreditIdentifier: "TEST1234567890", Level: 3, Manager: "张三",
		Contact: "18800000000", Email: "test@example.com",
	}
	if err := validatePartnerEditorValues("supplier", "add", valid); err != nil {
		t.Fatalf("valid supplier rejected: %v", err)
	}
	invalid := valid
	invalid.Level = 4
	if err := validatePartnerEditorValues("supplier", "add", invalid); err == nil || !strings.Contains(err.Error(), "三级") {
		t.Fatalf("unexpected level validation: %v", err)
	}
	invalid = valid
	invalid.Contact = "010-12345678"
	if err := validatePartnerEditorValues("supplier", "add", invalid); err == nil || !strings.Contains(err.Error(), "手机号") {
		t.Fatalf("unexpected contact validation: %v", err)
	}
}

func TestWarehouseEditorValidationProtectsHierarchyAndRequiredImage(t *testing.T) {
	zone := warehouseKind{Key: "zone", Label: "库区"}
	valid := warehouseEditorValues{ParentID: "warehouse", ParentStatus: "激活", Name: "原料区", Code: "ZONE-01", Image: "zone.png"}
	if err := validateWarehouseEditorValues(zone, valid); err != nil {
		t.Fatalf("valid zone rejected: %v", err)
	}
	invalid := valid
	invalid.ParentStatus = "禁用"
	if err := validateWarehouseEditorValues(zone, invalid); err == nil || !strings.Contains(err.Error(), "激活父级") {
		t.Fatalf("unexpected parent status validation: %v", err)
	}
	invalid = valid
	invalid.Image = ""
	if err := validateWarehouseEditorValues(zone, invalid); err == nil || !strings.Contains(err.Error(), "图片") {
		t.Fatalf("unexpected image validation: %v", err)
	}
}

func TestMasterDataStatusChoicesExcludeDelete(t *testing.T) {
	for _, kind := range []string{"supplier", "customer", "carrier"} {
		if containsString(partnerStatuses(kind), "删除") {
			t.Fatalf("%s exposes delete status: %#v", kind, partnerStatuses(kind))
		}
	}
	if containsString(warehouseStatuses(), "删除") {
		t.Fatalf("warehouse statuses expose delete: %#v", warehouseStatuses())
	}
}

func TestMasterDataKindsCarryExactWritePermissions(t *testing.T) {
	partnerPerms := api.Perms{
		Menus:   []api.Menu{{Path: "/business_partner/supplier"}},
		Buttons: []api.Button{{Perms: "business_partner:supplier:list"}},
	}
	partners := availablePartnerKinds(partnerPerms)
	if len(partners) != 1 || partners[0].AddPermission != "business_partner:supplier:add" || partners[0].EditPermission != "business_partner:supplier:edit" || partners[0].StatusPermission != "business_partner:supplier:status" {
		t.Fatalf("partner permissions = %#v", partners)
	}

	warehousePerms := api.Perms{
		Menus:   []api.Menu{{Path: "/warehouse/bin"}},
		Buttons: []api.Button{{Perms: "warehouse:bin:list"}},
	}
	warehouses := availableWarehouseKinds(warehousePerms)
	if len(warehouses) != 1 || warehouses[0].AddPermission != "warehouse:bin:add" || warehouses[0].EditPermission != "warehouse:bin:edit" || warehouses[0].StatusPermission != "warehouse:bin:status" {
		t.Fatalf("warehouse permissions = %#v", warehouses)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
