package main

import (
	"context"
	"fmt"
	"gibraltar/config"
	"gibraltar/internal/handlers"
	"gibraltar/internal/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	gitService := services.NewGitService(config.ConfigSourceDirectoryPath, config.RemoteRepository, config.RemoteBranch)
	newDataExist := make(chan struct{}, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var tick = time.NewTicker(config.PullCooldown)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				close(newDataExist)
				return
			case <-tick.C:
				if newData, err := gitService.ExistNewCommit(); newData {
					select {
					case newDataExist <- struct{}{}:
					default:
					}
				} else {
					log.Println(err)
				}

			}
		}
	}()
	tester := services.NewVlessTestService(config.TestURL)
	cache := services.NewCache()
	dataSource := services.NewUrlParser(config.VlessSecureConfigsURL, config.CIDRWhitelistURL, config.URLsWhitelistURL)
	CIDRlist, err := dataSource.ParseSubnets()
	if err != nil {
		panic(fmt.Errorf("Can't get CIDR whitelist: %s", err))
	}
	allowedSNIs, err := dataSource.ParseSNIs()
	if err != nil {
		panic(fmt.Errorf("Can't get SNI whitelist: %s", err))
	}
	filter := services.NewConfigFilter(CIDRlist, allowedSNIs)
	updater := services.NewConfigUpdater(cache, filter, tester, dataSource)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-newDataExist:
				if err := updater.AddConfigsToCacheFromSource(); err != nil {
					log.Println(err)
				}
			case <-tick.C:
				if err := updater.AddAvailableConfigsToCache(); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	cfgHandler := handlers.NewConfigsHandler(cache)
	router := gin.Default()
	router.GET("/configs", cfgHandler.CurrentAvailableConfigs)
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router.Handler(),
	}
	go func() {
		err = srv.ListenAndServe()
		if err != nil {
			log.Fatalln(err)
		}

	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	cancel()
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	_ = srv.Shutdown(ctxShutdown)
	wg.Wait()

}
