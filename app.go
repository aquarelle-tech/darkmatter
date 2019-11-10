/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package main

import (
	"log"
	"net/http"

	"cratos.network/darkmatter/crawlers"
	"cratos.network/darkmatter/mapreduce"
	"cratos.network/darkmatter/service"
	"cratos.network/darkmatter/types"
)

// List of available crawlers
var directory = []types.PriceSourceCrawler{
	crawlers.NewBinanceCrawler(),
	crawlers.NewLiquidCrawler(),
	crawlers.NewBitfinexCrawler(),
}

var publishedPrices = make(chan types.PriceMessage)

func main() {

	quotedCurrency := "USD"

	// Prepare and run the subroutines for the oracle service
	server := service.NewOracleServer(publishedPrices)
	server.Initialize()

	// Prepare and start the subroutines to manage the request of sources
	processor := mapreduce.NewMapReduceProcessor(directory, quotedCurrency, publishedPrices)
	processor.Initialize()

	// Start the server locally
	log.Println("http server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
