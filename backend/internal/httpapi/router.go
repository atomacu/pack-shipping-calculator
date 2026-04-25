package httpapi

import "net/http"

func NewRouter(packService packService) http.Handler {
	handler := handler{packService: packService}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handler.health)
	mux.HandleFunc("GET /api/v1/packs", handler.getPacks)
	mux.HandleFunc("PUT /api/v1/packs", handler.replacePacks)
	mux.HandleFunc("POST /api/v1/orders/calculate", handler.calculateOrder)

	return withCORS(withJSONHeaders(mux))
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
