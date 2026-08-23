package api

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var routeLinePattern = regexp.MustCompile(`^(get|post|put|delete|patch)\s+([^\s(]+)`)

func TestWindowsClientCriticalRoutesRemainRegistered(t *testing.T) {
	routes := backendRoutes(t)

	required := []string{
		"POST /auth/login", "POST /auth/logout",
		"GET /account/profile", "PATCH /account/avatar", "PATCH /account/password", "GET /account/menu",
		"GET /material", "GET /material/info", "GET /material/category", "POST /material", "PUT /material", "POST /material/category", "PUT /material/category", "GET /material/price",
		"GET /material/new_delivery", "POST /material/new_delivery/quote/export", "POST /material/new_delivery/rebuild", "GET /material/new_delivery/rebuild/latest", "GET /material/new_delivery/rebuild/tasks",
		"GET /material/quote", "GET /material/quote/info", "POST /material/quote", "PUT /material/quote", "POST /material/quote/submit", "POST /material/quote/price", "PATCH /material/quote/void", "POST /material/quote/export",
		"GET /inventory", "GET /inventory/record", "GET /inventory/list",
		"GET /inbound/receipt", "POST /inbound/receipt", "PUT /inbound/receipt", "DELETE /inbound/receipt", "PATCH /inbound/receipt/check", "GET /inbound/receipt/receive", "POST /inbound/receipt/receive",
		"GET /outbound/page", "GET /outbound/materials", "POST /outbound", "DELETE /outbound", "PATCH /outbound/confirm", "PATCH /outbound/pick", "PATCH /outbound/pack", "PATCH /outbound/weigh", "PATCH /outbound/departure", "PATCH /outbound/receipt", "PATCH /outbound/revise", "POST /outbound/fast_departure",
		"GET /outbound/summary",
		"GET /supplier", "GET /supplier/list", "POST /supplier", "PUT /supplier", "PATCH /supplier/status",
		"GET /customer", "GET /customer/list", "POST /customer", "PUT /customer", "PATCH /customer/status",
		"GET /carrier", "POST /carrier", "PUT /carrier", "PATCH /carrier/status",
		"GET /warehouse/tree", "GET /warehouse", "POST /warehouse", "PUT /warehouse", "PATCH /warehouse/status",
		"GET /warehouse_zone", "POST /warehouse_zone", "PUT /warehouse_zone", "PATCH /warehouse_zone/status",
		"GET /warehouse_rack", "POST /warehouse_rack", "PUT /warehouse_rack", "PATCH /warehouse_rack/status",
		"GET /warehouse_bin", "POST /warehouse_bin", "PUT /warehouse_bin", "PATCH /warehouse_bin/status",
		"GET /customer/transaction", "POST /customer/transaction", "GET /images", "POST /images",
		"GET /user", "POST /user", "PUT /user", "PATCH /user/status", "PATCH /user/password", "PATCH /user/roles",
		"GET /department", "POST /department", "PUT /department",
		"GET /role", "GET /role/list", "POST /role", "PUT /role", "PATCH /role/status",
		"GET /menu/list", "GET /api", "GET /role/menus", "POST /role/menus", "GET /role/apis", "POST /role/apis",
	}
	for _, contract := range required {
		if !routes[contract] {
			t.Errorf("critical backend route is no longer registered: %s", contract)
		}
	}
	if routes["PUT /outbound"] {
		t.Fatal("outbound update route appeared; review the deliberate Windows client edit exclusion before enabling it")
	}
}

func TestEveryLiteralClientContractExistsInBackendRegistry(t *testing.T) {
	routes := backendRoutes(t)
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contract test source")
	}
	apiDir := filepath.Dir(source)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		contracts, parseErr := literalClientContracts(filepath.Join(apiDir, entry.Name()))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		for contract := range contracts {
			seen[contract] = true
			if !routes[contract] {
				t.Errorf("%s uses an unregistered backend contract: %s", entry.Name(), contract)
			}
		}
	}
	if len(seen) < 50 {
		t.Fatalf("literal contract audit found only %d contracts; scanner likely regressed", len(seen))
	}
}

func literalClientContracts(name string) (map[string]bool, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
	if err != nil {
		return nil, err
	}
	contracts := make(map[string]bool)
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 3 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (selector.Sel.Name != "do" && selector.Sel.Name != "doWithContentType" && selector.Sel.Name != "download") {
			return true
		}
		methodSelector, ok := call.Args[1].(*ast.SelectorExpr)
		if !ok {
			return true
		}
		httpPackage, ok := methodSelector.X.(*ast.Ident)
		if !ok || httpPackage.Name != "http" || !strings.HasPrefix(methodSelector.Sel.Name, "Method") {
			return true
		}
		pathLiteral, ok := call.Args[2].(*ast.BasicLit)
		if !ok || pathLiteral.Kind != token.STRING {
			return true
		}
		path, unquoteErr := strconv.Unquote(pathLiteral.Value)
		if unquoteErr != nil || !strings.HasPrefix(path, "/") {
			return true
		}
		method := strings.ToUpper(strings.TrimPrefix(methodSelector.Sel.Name, "Method"))
		contracts[method+" "+strings.TrimRight(path, "/")] = true
		return true
	})
	return contracts, nil
}

func TestDynamicClientContractsRemainExplicitlyCovered(t *testing.T) {
	routes := backendRoutes(t)
	// These endpoints deliberately choose a method or path at runtime. Keeping
	// the small list explicit prevents an AST literal audit from silently
	// treating dynamic dispatch as covered.
	for _, contract := range []string{
		"GET /inventory", "GET /inventory/record",
		"POST /material/quote", "PUT /material/quote",
	} {
		if !routes[contract] {
			t.Errorf("dynamic client contract is no longer registered: %s", contract)
		}
	}
}

func backendRoutes(t *testing.T) map[string]bool {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contract test source")
	}
	mainAPI := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", "api", "main.api"))
	file, err := os.Open(mainAPI)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("api/main.api is not available outside the monorepo")
		}
		t.Fatal(err)
	}
	defer file.Close()

	routes := make(map[string]bool)
	prefix := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "prefix:") {
			prefix = strings.TrimSpace(strings.TrimPrefix(line, "prefix:"))
			continue
		}
		matches := routeLinePattern.FindStringSubmatch(strings.ToLower(line))
		if len(matches) != 3 {
			continue
		}
		path := "/" + strings.Trim(strings.TrimSpace(prefix), "/") + "/" + strings.Trim(matches[2], "/")
		path = strings.TrimRight(strings.ReplaceAll(path, "//", "/"), "/")
		routes[strings.ToUpper(matches[1])+" "+path] = true
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return routes
}

func TestWindowsClientKeepsDeliberatelyExcludedContractsOut(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contract test source")
	}
	apiDir := filepath.Dir(source)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		t.Fatal(err)
	}
	var clientSource strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(apiDir, entry.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		clientSource.Write(data)
	}
	for _, excluded := range []string{`"/plan`, `"/customer/recount`, `http.MethodPut, "/account/profile"`, `http.MethodDelete, "/images"`} {
		if strings.Contains(clientSource.String(), excluded) {
			t.Errorf("deliberately excluded backend capability appeared in Windows client source: %s", excluded)
		}
	}
}
