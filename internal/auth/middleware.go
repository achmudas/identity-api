package auth

import (
	"net/http"
	"slices"
)

func Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		roles, _ := r.Context().Value(RolesKey).([]string)

		if slices.Contains(roles, "sysdev") {
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}
	return http.HandlerFunc(fn)
}
