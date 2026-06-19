package middleware

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"oil/config"
	"oil/infras/otel"
	"oil/shared/cache"
)

const (
	otelHTTPScopeName = "http"
)

type AppMiddleware interface {
	Tracing(http.Handler) http.Handler
	RateLimit() func(http.Handler) http.Handler
}

type appMiddleware struct {
	otel   otel.Otel
	config *config.Config
	cache  cache.RedisCache
}

func NewAppMiddleware(otel otel.Otel, config *config.Config, cache cache.RedisCache) AppMiddleware {
	return &appMiddleware{
		otel:   otel,
		config: config,
		cache:  cache,
	}
}

func (a *appMiddleware) Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()

		rctx := chi.RouteContext(ctx)
		method := request.Method
		path := rctx.Routes.Find(chi.NewRouteContext(), method, request.URL.Path)
		userAgent := a.getUA(request)
		spanName := fmt.Sprintf("%s %s", method, path)

		ctx, scope := a.otel.NewScope(ctx, otelHTTPScopeName, spanName)
		defer scope.End()

		scope.SetAttributes(map[string]any{
			"app.name":        a.config.App.Name,
			"http.path":       path,
			"http.route":      path,
			"http.method":     method,
			"http.user_agent": userAgent,
			"http.host":       request.Host,
			"http.source":     request.RemoteAddr,
		})

		ww := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)
		next.ServeHTTP(ww, request.WithContext(ctx))

		scope.SetAttributes(map[string]any{
			"http.status_code": ww.Status(),
		})
	})
}
