package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lensgolda/geocapture/interfaces"
	"github.com/lensgolda/geocapture/providers/nominatim"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("Start...")

	/* Init local db connection */
	connStr := "postgres://postgres:@localhost/wiwin?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	defer func() {
		_ = db.Close()
	}()

	if err != nil {
		log.Fatal(err)
		return
	}

	/*
	 * Create provider
	 * var algoliaProvider models.Provider = algolia.NewProvider()
	 * var mapqProvider models.Provider = mapquest.NewProvider()
	 */
	var nomProvider interfaces.Provider = nominatim.NewProvider()

	/*
	 * Process from local db or from file
	 * logfile.ProcessFailedCountries(db, "nominatim.failed", nomProvider)
	 */
	nomProvider.CountryNameLocalize(db)
}
