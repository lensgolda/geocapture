package algolia

import (
	"bytes"
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
	backupHost1        = "https://places-1.algolianet.com"
	backupHost2        = "https://places-2.algolianet.com"
	backupHost3        = "https://places-3.algolianet.com"
	providerFailedFile = "algolia.failed"
	providerName       = "algolia"
)

var (
	allowedLocales = map[string]string{
		"ru": "ru",
		"en": "en",
		"kk": "kk",
		"uk": "uk",
	}
	query = map[string]string{
		"type": "city",
	}
	data map[string]interface{}
	city models.City
)

type Algolia struct {
	Name           string
	FailedFileName string
	RequestTimeout time.Duration
	Client         *http.Client
}

func NewProvider() *Algolia {
	return &Algolia{
		Name:           providerName,
		FailedFileName: providerFailedFile,
		RequestTimeout: time.Millisecond * 50,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (alg *Algolia) CreateCitiesRequest(city models.City) (*http.Request, error) {

	if city.Name != nil {
		query["query"] = *city.Name
	} else {
		if city.NameNational != nil {
			query["query"] = *city.NameNational
		} else {
			return nil, errors.New("both names from cities table are NULL")
		}
	}

	requestBody, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", settings.Config.Algolia.URL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "MSIE/15.0")
	req.Header.Add("X-Algolia-Application-Id", settings.Config.Algolia.AppId)
	req.Header.Add("X-Algolia-API-Key", settings.Config.Algolia.ApiKey)

	return req, nil
}

func (alg *Algolia) ParseCitiesResponse(resp *http.Response) (map[string]interface{}, error) {
	bytesBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytesBody, &data); err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errors.New("response data have zero length")
	}

	if records, ok := data["hits"]; ok {
		if hits, ok := records.([]interface{}); ok {
			if len(hits) != 0 {
				if firstHit, ok := hits[0].(map[string]interface{}); ok {
					if localeNames, ok := firstHit["locale_names"].(map[string]interface{}); ok {
						return localeNames, nil
					}
				}
			}
		}
	}

	return nil, errors.New("nested data error, see data nested types and values")
}

func (alg Algolia) ProcessCitiesData(db *sql.DB, data interface{}, city models.City) error {
	if data, ok := data.(map[string]interface{}); ok {
		var (
			stmt *sql.Stmt
			err  error
		)
		defer func() {
			_ = stmt.Close()
		}()

		for k, v := range data {
			if locale, ok := allowedLocales[k]; ok {
				stmt, err = db.Prepare("INSERT INTO cities_translations(city_id, locale, name, int_name) VALUES ($1, $2, $3, $4)")
				if err != nil {
					return err
				}

				if names, ok := v.([]interface{}); ok {
					if len(names) == 0 {
						continue
					}
					if name, ok := names[0].(string); ok {
						if _, err = stmt.Exec(city.ID, locale, name, city.NameNational); err != nil {
							return err
						}
						log.Printf("Insert OK: CityID = %d, locale = %s, name = %s, int_name = %v\n", city.ID, locale, name, city.NameNational)
					}
				}
			}
		}
		return nil
	}
	return errors.New("data is not of type map[string]interface{}")
}

func (alg *Algolia) RetryBackupHosts(client http.Client, r *http.Request) (*http.Response, error) {

	r.URL.Host = backupHost1
	respB1, errB1 := client.Do(r)
	if errB1 != nil {
		log.Println("backup host1 unreachable: ", errB1.Error())
		r.URL.Host = backupHost2
		respB2, errB2 := client.Do(r)
		if errB2 != nil {
			log.Println("backup host2 unreachable: ", errB2.Error())
			r.URL.Host = backupHost3
			respB3, errB3 := client.Do(r)
			if errB3 != nil {
				log.Println("backup host3 unreachable: ", errB3.Error())
				return nil, errB3
			}
			return respB3, nil
		}
		return respB2, nil
	}
	return respB1, nil
}

func (alg *Algolia) FailedFile() string {
	return alg.FailedFileName
}

func (alg *Algolia) ProcessCities(db *sql.DB) {
	f, err := os.OpenFile(alg.FailedFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

		req, err := alg.CreateCitiesRequest(city)
		if err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		resp, err := alg.Client.Do(req)
		if err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		data, err := alg.ParseCitiesResponse(resp)
		if err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		if err := alg.ProcessCitiesData(db, data, city); err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}

		time.Sleep(alg.RequestTimeout)
	}
}
