package main

import (
	"log"
	"net/http"

	"example/internal/api"
	"example/internal/dto"
	"example/internal/repositories"
	"example/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const listenAddress = ":8080"

func main() {
	ordersService := services.NewOrdersService(
		repositories.NewOrdersMemoryRepository([]dto.Room{
			{"reddison", "lux"},
			{"reddison", "premium"},
		}),
	)
	createOrdersHandler := api.NewCreateOrderHandler(ordersService)

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Post("/orders", createOrdersHandler.Handle)

	log.Default().Printf("api server started on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, router); err != nil {
		panic(err)
	}
}
