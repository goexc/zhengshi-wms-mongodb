package ui

import (
	"errors"
	"testing"
	"time"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestValidateFastOutboundRequestAcceptsOptionalStages(t *testing.T) {
	now := time.Date(2026, 8, 22, 16, 0, 0, 0, time.Local)
	day := time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local).Unix()
	request := api.FastOutboundRequest{
		PickingTime: day, DepartureTime: day, ReceiptTime: day,
	}
	if err := validateFastOutboundRequest(request, now); err != nil {
		t.Fatal(err)
	}
}

func TestValidateFastOutboundRequestRejectsBrokenTimelineAndFuture(t *testing.T) {
	now := time.Date(2026, 8, 22, 16, 0, 0, 0, time.Local)
	request := api.FastOutboundRequest{
		PickingTime: time.Date(2026, 8, 21, 0, 0, 0, 0, time.Local).Unix(),
		PackingTime: time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local).Unix(),
	}
	err := validateFastOutboundRequest(request, now)
	var validationErr *fastOutboundValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "packing" {
		t.Fatalf("error = %#v", err)
	}

	request = api.FastOutboundRequest{DepartureTime: time.Date(2026, 8, 23, 0, 0, 0, 0, time.Local).Unix()}
	err = validateFastOutboundRequest(request, now)
	if !errors.As(err, &validationErr) || validationErr.Field != "departure" {
		t.Fatalf("future error = %#v", err)
	}
}

func TestFastOutboundDayUnixUsesLocalNaturalDay(t *testing.T) {
	value := time.Date(2026, 8, 22, 18, 35, 17, 0, time.Local)
	want := time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local).Unix()
	if got := fastOutboundDayUnix(value); got != want {
		t.Fatalf("day = %d, want %d", got, want)
	}
	if got := fastOutboundDayUnix(time.Time{}); got != 0 {
		t.Fatalf("zero day = %d", got)
	}
}

func TestAdminMenuAvailableFindsNestedSupportedModule(t *testing.T) {
	perms := api.Perms{Menus: []api.Menu{{Path: "/acl", Children: []api.Menu{{Path: "/acl/role"}}}}}
	if !adminMenuAvailable(perms) {
		t.Fatal("expected admin menu to be available")
	}
	if adminMenuAvailable(api.Perms{Menus: []api.Menu{{Path: "/inventory/index"}}}) {
		t.Fatal("unexpected admin menu")
	}
}

func TestFlattenAdminCatalogsPreservesHierarchy(t *testing.T) {
	menus := []api.AdminMenu{{ID: "root", Name: "Root", Type: 1, Children: []api.AdminMenu{{ID: "child", Name: "Child", Type: 2, Meta: api.AdminMenuMeta{Perms: "x:y"}}}}}
	menuRows := flattenAdminMenuRows(menus, "")
	menuOptions := flattenAdminMenuOptions(menus, "")
	if len(menuRows) != 2 || menuRows[1].Title != "    Child" || menuRows[1].Perms != "x:y" {
		t.Fatalf("menu rows = %#v", menuRows)
	}
	if len(menuOptions) != 2 || menuOptions[1].ID != "child" {
		t.Fatalf("menu options = %#v", menuOptions)
	}

	apis := []api.AdminAPI{{ID: "module", Name: "模块", Type: 1, Children: []api.AdminAPI{{ID: "route", Name: "查询", Type: 2, Method: "GET", URI: "/user"}}}}
	apiRows := flattenAdminAPIRows(apis, "")
	if len(apiRows) != 2 || apiRows[1].Name != "    查询" || apiRows[1].Method != "GET" {
		t.Fatalf("api rows = %#v", apiRows)
	}
}

func TestAdminIDSetComparisonAndDiffIgnoreOrder(t *testing.T) {
	if !equalAdminIDSet([]string{"a", "b", "a"}, []string{"b", "a"}) {
		t.Fatal("sets should match")
	}
	if equalAdminIDSet([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("sets should differ")
	}
	added, removed := adminIDDiffCounts([]string{"a", "b"}, []string{"b", "c"})
	if added != 1 || removed != 1 {
		t.Fatalf("diff = %d/%d", added, removed)
	}
}
