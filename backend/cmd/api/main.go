package main

import (
	"log"
	"net/http"

	"github.com/kappannotavio/ponto-eletronico/handler"
	"github.com/kappannotavio/ponto-eletronico/repository"
	"github.com/kappannotavio/ponto-eletronico/service"
)

func main() {
	timeRecordRepository := repository.NewInMemoryTimeRecordRepository()

	timeRecordService := service.NewTimeRecordService(
		timeRecordRepository,
	)

	timeRecordHandler := handler.NewTimeRecordHandler(
		timeRecordService,
	)

	mux := http.NewServeMux()

	mux.HandleFunc(
		"POST /api/v1/time-records",
		timeRecordHandler.Create,
	)

	log.Println("server running on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
