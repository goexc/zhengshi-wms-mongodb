package permissionx

import "net/http"

// Target makes a destructive-operation preview require the same permission as
// its commit endpoint. Existing roles do not need a second, broader read grant.
func Target(path, method string) (string, string) {
	if path == "/outbound/remove-preview" && method == http.MethodGet {
		return "/outbound", http.MethodDelete
	}
	return path, method
}
