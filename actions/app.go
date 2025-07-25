package actions

import (
	"sync"

	"github.com/akingundogdu/production-ready-go-backend-architecture/locales"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/middleware/contenttype"
	"github.com/gobuffalo/middleware/forcessl"
	"github.com/gobuffalo/middleware/i18n"
	"github.com/gobuffalo/middleware/paramlogger"
	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"
	"github.com/unrolled/secure"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")

var (
	app     *buffalo.App
	appOnce sync.Once
	T       *i18n.Translator
)

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
//
// Routing, middleware, groups, etc... are declared TOP -> DOWN.
// This means if you add a middleware to `app` *after* declaring a
// group, that group will NOT have that new middleware. The same
// is true of resource declarations as well.
//
// It also means that routes are checked in the order they are declared.
// `ServeFiles` is a CATCH-ALL route, so it should always be
// placed last in the route declarations, as it will prevent routes
// declared after it to never be called.
func App() *buffalo.App {
	appOnce.Do(func() {
		app = buffalo.New(buffalo.Options{
			Env:          ENV,
			SessionStore: sessions.Null{},
			PreWares: []buffalo.PreWare{
				cors.Default().Handler,
			},
			SessionName: "_production_ready_go_backend_session",
		})

		// Automatically redirect to SSL
		app.Use(forceSSL())

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Set the request content type to JSON
		app.Use(contenttype.Set("application/json"))

		// Health check routes
		// These should be at the top for quick health monitoring
		app.GET("/health", HealthHandler)
		app.GET("/health/live", LivenessHandler)
		app.GET("/health/ready", ReadinessHandler)
		
		// Authentication routes (public)
		authGroup := app.Group("/auth")
		{
			authGroup.POST("/register", RegisterHandler)
			authGroup.POST("/login", LoginHandler)
			
			// Protected auth routes (require valid JWT)
			protectedAuth := authGroup.Group("")
			protectedAuth.Use(AuthMiddleware)
			{
				protectedAuth.GET("/me", MeHandler)
				protectedAuth.POST("/refresh", RefreshTokenHandler)
			}
		}
		
		// API v1 routes
		apiV1 := app.Group("/api/v1")
		{
			// Protected routes (require authentication)
			protected := apiV1.Group("")
			protected.Use(AuthMiddleware)
			{
				// User routes - any authenticated user
				protected.GET("/profile", MeHandler) // Alias for /auth/me
				
				// Admin only routes
				adminOnly := protected.Group("/admin")
				adminOnly.Use(AdminMiddleware)
				{
					// Admin endpoints will be added here
					// adminOnly.GET("/users", AdminUsersListHandler)
					// adminOnly.GET("/stats", AdminStatsHandler)
				}
			}
		}
	})

	return app
}

// translations will load locale files, set up the translator `actions.T`,
// and will return a middleware to use to load the correct locale for each
// request.
// for more information: https://gobuffalo.io/en/docs/localization
func translations() buffalo.MiddlewareFunc {
	var err error
	if T, err = i18n.New(locales.FS(), "en-US"); err != nil {
		app.Stop(err)
	}
	return T.Middleware()
}

// forceSSL will return a middleware that will redirect an incoming request
// if it is not HTTPS. "http://example.com" => "https://example.com".
// This middleware does **not** enable SSL. for your application. To do that
// we recommend using a proxy: https://gobuffalo.io/en/docs/proxy
// for more information: https://github.com/unrolled/secure/
func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}
