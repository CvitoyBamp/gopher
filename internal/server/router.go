package server

import (
	"github.com/CvitoyBamp/gopher/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

// router for http services
func (bs *BackendServer) router() chi.Router {
	r := chi.NewRouter()

	//r.Use(cors.Handler(cors.Options{
	//	AllowedOrigins:   []string{"http://*"},
	//	AllowedMethods:   []string{"GET", "POST"},
	//	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	//	AllowCredentials: false,
	//}))

	r.Use(middlewares.VerifyMiddleware(bs.DB))

	r.Route("/api/user/", func(r chi.Router) {
		r.Post("/register", bs.registerHandler)
		r.Post("/login", bs.loginHandler)
		r.Get("/withdrawals", bs.withdrawalsHandlers)
		r.Route("/orders", func(r chi.Router) {
			r.Post("/", bs.postOrdersHandler)
			r.Get("/", bs.getOrdersHandler)
		})
		r.Route("/balance", func(r chi.Router) {
			r.Post("/withdraw", bs.withdrawHandler)
			r.Get("/", bs.getBalanceHandler)
		})
	})

	return r
}
