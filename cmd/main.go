package main

import (
	handlers "gibraltar/internal/handlers"
	services "gibraltar/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

const timerDuration = 6 * time.Hour

func main() {
	preparationService := services.NewPreparationService("/home/mydan/rjsxrd", "https://github.com/whoahaow/rjsxrd.git", "main")
	tester := services.NewVlessTestService("http://cp.cloudflare.com/")
	cache := services.NewCache()
	deps := &services.Dependencies{
		PreparationService: preparationService,
		VlessTestService:   tester,
		Cache:              cache,
	}
	deps.CalculateAvailableServers()
	cfgHandler := handlers.NewConfigHandler(deps)
	router := gin.Default()
	router.GET("/configs", cfgHandler.CurrentAvailableConfigs)
	router.PATCH("/configs", cfgHandler.RequestConfigsUpdate)

	var Timer *time.Timer = time.NewTimer(timerDuration)
	go func() {
		for {
			<-Timer.C
			deps.CalculateAvailableServers()
			Timer.Reset(timerDuration)
		}
	}()
	router.Run("0.0.0.0:8080")

}
