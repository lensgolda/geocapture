package mapquest

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lensgolda/geocapture/logfile"
	"github.com/lensgolda/geocapture/models"
	"github.com/lensgolda/geocapture/settings"
)

const (
	providerFailedFile = "mapquest.failed"
	providerName       = "mapquest"
)

var (
	allowedLocales = map[string]string{
		"name:ru": "ru",
		"name:en": "en",
		"name:kk": "kk",
		"name:uk": "uk",
	}
	data []map[string]interface{}
	city models.City
)

type Provider struct {
	Name           string
	FailedFileName string
	RequestTimeout time.Duration
	Client         *http.Client
}

func NewProvider() *Provider {
	return &Provider{
		Name:           providerName,
		FailedFileName: providerFailedFile,
		RequestTimeout: time.Millisecond * 350,
		Client: &http.Client{
			Timeout: 40 * time.Second,
		},
	}
}

func (mapq *Provider) CreateCitiesRequest(city models.City) (*http.Request, error) {
	req, err := http.NewRequest("GET", settings.Config.Mapquest.URL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("key", settings.Config.Mapquest.ApiKey)
	q.Add("format", "json")
	if city.Name != nil {
		q.Add(city.Type(), *city.Name)
	} else {
		if city.NameNational != nil {
			q.Add(city.Type(), *city.NameNational)
		} else {
			return nil, errors.New("both names from cities table are NULL")
		}
	}
	q.Add("addressdetails", "1")
	q.Add("namedetails", "1")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "MSIE/15.0")

	return req, nil
}

func (mapq *Provider) ParseResponse(resp *http.Response) (map[string]interface{}, error) {
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errors.New("response data have zero length")
	}

	if namedetails, ok := data[0]["namedetails"]; ok {
		if record, ok := namedetails.(map[string]interface{}); ok {
			return record, nil
		}
	}
	return nil, errors.New("nested data error, see data nested types and values")
}

func (mapq *Provider) ProcessData(db *sql.DB, data interface{}, city models.City) error {
	if data, ok := data.(map[string]interface{}); ok {
		for k, v := range data {
			if locale, ok := allowedLocales[k]; ok {
				stmt, err := db.Prepare("INSERT INTO cities_translations(city_id, locale, name, int_name) VALUES ($1, $2, $3, $4)")
				if err != nil {
					return err
				}

				_, err = stmt.Exec(city.ID, locale, v, data["int_name"])
				if err != nil {
					return err
				}
				log.Printf("Insert OK: CityID = %d, locale = %s, name = %s, int_name = %v\n", city.ID, locale, v, data["int_name"])
			}
		}
		return nil
	}
	return errors.New("data is not of type map[string]interface{}")
}

func (mapq *Provider) FailedFile() string {
	return mapq.FailedFileName
}

func (mapq *Provider) ProcessCities(db *sql.DB) {
	f, err := os.OpenFile("mapquest.failed", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open file: ", err)
	}
	defer func() {
		_ = f.Close()
	}()

	rowsAll, err := db.Query("SELECT id, name, name_national FROM cities ORDER BY id")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = rowsAll.Close()
	}()

	var (
		counter uint = 0
		resp    *http.Response
	)
	defer func() {
		_ = resp.Body.Close()
	}()

	for rowsAll.Next() {
		counter += 1
		if err := rowsAll.Scan(&city.ID, &city.Name, &city.NameNational); err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err)
			continue
		}
		fmt.Printf("%d >>> CityID: %d Name: %v\n", counter, city.ID, city.Name)

		req, err := mapq.CreateCitiesRequest(city)
		if err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		resp, err := mapq.Client.Do(req)
		if err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		data, err := mapq.ParseResponse(resp)
		if err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		if err := mapq.ProcessData(db, data, city); err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		time.Sleep(mapq.RequestTimeout)
	}
}
