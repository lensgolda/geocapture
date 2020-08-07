package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lensgolda/geocapture/interfaces"
	"github.com/lensgolda/geocapture/providers/nominatim"
	"github.com/lensgolda/geocapture/settings"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("Start...")
	fmt.Print("Loading configuration...")
	if err := settings.LoadSettings(); err != nil {
		fmt.Printf("ERROR during settings loading: %s", err.Error())
		return
	}
	time.Sleep(time.Second * 1)
	fmt.Printf("OK\n")

	/* Init local db connection */
	connStr := fmt.Sprintf(
		"postgres://postgres:@%s/%s?sslmode=%s",
		settings.Config.DB.Host,
		settings.Config.DB.Name,
		settings.Config.DB.SSL,
	)
	fmt.Printf("Connecting to database: %s...", connStr)

	db, err := sql.Open("postgres", connStr)
	defer func() {
		_ = db.Close()
	}()

	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second * 1)
	fmt.Printf("OK\n")

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
	fmt.Println("Success...OK")
}
