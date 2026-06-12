package auth

import (
	"log"
	"net/http"
	"slices"
)

func AuthMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var roles []string = r.Context().Value(RolesKey).([]string)

		if slices.Contains(roles, "sysdev") {
			log.Printf("Contains roles: %v", roles)
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}
	return http.HandlerFunc(fn)
}
