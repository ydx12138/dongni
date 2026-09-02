package middleware

import (
	"blog/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			for _, allowed := range config.Cfg.CORS.AllowOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
		AllowMethods:     config.Cfg.CORS.AllowMethods,
		AllowHeaders:     config.Cfg.CORS.AllowHeaders,
		AllowCredentials: config.Cfg.CORS.AllowCredentials,
		MaxAge:           config.Cfg.CORS.MaxAge,
	})
}
