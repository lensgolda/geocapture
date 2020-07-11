package logfile

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lensgolda/geocapture/interfaces"
	"github.com/lensgolda/geocapture/models"
)

var (
	city    models.City
	country models.Country
)

type Failed struct {
	File *os.File
}

func LogFailed(f *os.File, recordID int) {
	_, err := f.WriteString(fmt.Sprintf("%d\n", recordID))
	if err != nil {
		log.Printf("Error writing to file. Error: %s\n", err.Error())
	}
}

func (failed *Failed) ProcessFailedCities(db *sql.DB, fileName string, provider interfaces.Provider) {
	f, err := os.OpenFile(provider.FailedFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open file: ", err)
	}
	defer func() {
		_ = f.Close()
	}()

	failedDataFile, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func() {
		_ = failedDataFile.Close()
	}()

	reader := bufio.NewReader(failedDataFile)
	var line string
	var counter uint = 0

	for {
		counter += 1

		//limiter for tests
		/*if counter >= 2 {
			break
		}*/

		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}

		row := db.QueryRow("SELECT id, name, name_national FROM cities WHERE id = $1", line)
		if err := row.Scan(&city.ID, &city.Name, &city.NameNational); err != nil {
			log.Println(err.Error())
			LogFailed(f, city.ID)
			continue
		}
		fmt.Printf("%d >>> CityID: %d, Name: %v\n", counter, city.ID, city.Name)

		// TODO: process through provider

		time.Sleep(time.Millisecond * 1100)

	}

	if err != io.EOF {
		fmt.Printf(" > FAILED: %v\n", err)
	}

	return
}

func (failed *Failed) ProcessFailedCountries(db *sql.DB, fileName string, provider interfaces.Provider) {
	f, err := os.OpenFile(provider.FailedFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open file: ", err)
	}
	defer func() {
		_ = f.Close()
	}()

	failedDataFile, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func() {
		_ = failedDataFile.Close()
	}()

	reader := bufio.NewReader(failedDataFile)
	var (
		line    string
		counter uint = 0
		resp    *http.Response
	)
	defer func() {
		_ = resp.Body.Close()
	}()

	for {
		counter += 1
		// limiter for tests
		/*if counter >= 2 {
			break
		}*/

		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}

		row := db.QueryRow("SELECT id, name, name_en FROM countries WHERE id = $1", line)
		if err := row.Scan(&country.ID, &country.Name, &country.NameEN); err != nil {
			log.Println(err.Error())
			LogFailed(f, country.ID)
			continue
		}
		fmt.Printf("%d >>> CountryID: %d, Name: %v\n", counter, country.ID, country.Name)

		// TODO: process through provider

		time.Sleep(time.Millisecond * 2500)
	}

	if err != io.EOF {
		fmt.Printf(" > Failed!: %v\n", err)
	}

	return
}
