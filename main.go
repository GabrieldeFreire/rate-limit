package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/GabrieldeFreire/rate-limit/cache"
	"github.com/GabrieldeFreire/rate-limit/log"
	"github.com/GabrieldeFreire/rate-limit/rate"
	"github.com/go-redis/redis/v8"
)

var logger = log.GetLogger()

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Welcome home!")
}

func main() {
	option := &redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	}
	cacheStrategy := cache.NewRedisStrategy(option)
	defer cacheStrategy.Close()

	rateMiddleware := rate.CreateLimitRateMiddleware(cacheStrategy)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", rateMiddleware(http.HandlerFunc(homeHandler)))

	port := ":8080"
	logger.Info("Server listening on port", "port", port)
	err := http.ListenAndServe(port, mux)
	if err != nil {
		logger.Error("ListenAndServe", err)
		os.Exit(1)
	}
}
