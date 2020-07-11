package nominatim

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/lensgolda/geocapture/logfile"
	"github.com/lensgolda/geocapture/models"
)

const (
	URL                = "https://nominatim.openstreetmap.org/search?"
	providerName       = "nominatim"
	providerFailedFile = "nominatim.failed"
	localeRU           = "ru"
	localeEN           = "en"
	localeKK           = "kk"
	localeUK           = "uk"
)

var (
	city    models.City
	country models.Country
)

type Nominatim struct {
	Name           string
	FailedFileName string
	RequestTimeout time.Duration
}

func NewProvider() *Nominatim {
	return &Nominatim{
		Name:           providerName,
		FailedFileName: providerFailedFile,
		RequestTimeout: time.Millisecond * 1500,
	}
}

func sendSearchRequest(model models.Model) (*http.Response, error) {
	params := url.Values{}
	params.Add("format", "json")
	//params.Add("addressdetails", "1")
	params.Add("namedetails", "1")
	city, ok1 := model.(models.City)
	country, ok2 := model.(models.Country)

	switch true {
	case ok1:
		if city.Name != nil {
			params.Add(city.Type(), *city.Name)
		} else {
			if city.NameNational != nil {
				params.Add(city.Type(), *city.NameNational)
			} else {
				return nil, errors.New("both names from cities table are NULL")
			}
		}
	case ok2:
		if country.Name != nil {
			params.Add(country.Type(), *country.Name)
		} else {
			if country.NameEN != nil {
				params.Add(country.Type(), *country.NameEN)
			} else {
				return nil, errors.New("both names from countries table are NULL")
			}
		}
	default:
		return nil, errors.New("wrong model type")
	}

	reqURL := URL + params.Encode()
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func parseSearchResponse(resp *http.Response) (models.AltName, error) {
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.AltName{}, err
	}
	result := models.NomResult{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return models.AltName{}, err
	}
	if len(result) == 0 {
		return models.AltName{}, errors.New("response data have zero length")
	}

	return result[0].Namedetail, nil
}

func insertLocale(db *sql.DB, locale string, altName models.AltName, model models.Model) error {
	stmt, err := db.Prepare("INSERT INTO countries_translations_temp(country_id, locale, name, int_name) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return err
	}

	var localizedName string
	switch locale {
	case localeRU:
		localizedName = *altName.NameRu
	case localeEN:
		localizedName = *altName.NameEn
	case localeKK:
		localizedName = *altName.NameKk
	case localeUK:
		localizedName = *altName.NameUk
	}
	_, err = stmt.Exec(model.Id(), locale, localizedName, altName.IntName)
	if err != nil {
		return err
	}
	log.Println("INSERT: OK")
	return nil
}

func processResponseData(db *sql.DB, altName models.AltName, model models.Model) error {
	var err error
	if altName.NameRu == nil &&
		altName.NameEn == nil &&
		altName.NameKk == nil &&
		altName.NameUk == nil {
		return errors.New("response data doesn't contain appropriate locale")
	}
	if altName.NameRu != nil {
		fmt.Printf("Locale: %v, AltName: %v\n", localeRU, altName.NameRu)
		err = insertLocale(db, localeRU, altName, model)
	}
	if altName.NameEn != nil {
		fmt.Printf("Locale: %v, value: %v\n", localeEN, altName.NameEn)
		err = insertLocale(db, localeEN, altName, model)
	}
	if altName.NameKk != nil {
		fmt.Printf("Locale: %v, value: %v\n", localeKK, altName.NameKk)
		err = insertLocale(db, localeKK, altName, model)
	}
	if altName.NameUk != nil {
		fmt.Printf("Locale: %v, value: %v\n", localeUK, altName.NameUk)
		err = insertLocale(db, localeUK, altName, model)
	}
	return err
}

func (nom *Nominatim) FailedFile() string {
	return nom.FailedFileName
}

func (nom *Nominatim) CountryNameLocalize(db *sql.DB) {
	f, err := os.OpenFile(nom.FailedFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open file: ", err)
	}
	defer func() {
		_ = f.Close()
	}()

	rowsAll, err := db.Query("SELECT id, name, name_en FROM countries ORDER BY id")
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

		if counter >= 2 {
			break
		}

		if err := rowsAll.Scan(&country.ID, &country.Name, &country.NameEN); err != nil {
			logfile.LogFailed(f, country.ID)
			log.Println(err.Error())
			continue
		}
		fmt.Printf("%d >>> CountryID: %d Name: %s, NameEn: %s\n", counter, country.ID, *country.Name, *country.NameEN)

		resp, err = sendSearchRequest(country)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		nd, err := parseSearchResponse(resp)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if err := processResponseData(db, nd, country); err != nil {
			log.Println(err.Error())
			continue
		}

		time.Sleep(nom.RequestTimeout)
	}
}

func (nom *Nominatim) CityNameLocalize(db *sql.DB) {
	f, err := os.OpenFile(nom.FailedFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

		// limit for testing
		if counter >= 2 {
			break
		}

		if err := rowsAll.Scan(&city.ID, &city.Name, &city.NameNational); err != nil {
			logfile.LogFailed(f, city.ID)
			log.Println(err.Error())
			continue
		}
		fmt.Printf("%d >>> CityID: %d Name: %v\n", counter, city.ID, city.Name)

		resp, err = sendSearchRequest(country)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		nd, err := parseSearchResponse(resp)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if err := processResponseData(db, nd, country); err != nil {
			log.Println(err.Error())
			continue
		}

		time.Sleep(nom.RequestTimeout)
	}
}
