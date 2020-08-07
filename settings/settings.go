package settings

import (
	"github.com/caarlos0/env"
)

type DB struct {
	Host string `env:"DB_HOST" envDefault:"localhost"`
	Port string `env:"DB_PORT"`
	Name string `env:"DB_NAME"`
	SSL  string `env:"DB_SSL_MODE" envDefault:"disable"`
}

type Algolia struct {
	AppId  string `env:"ALGOLIA_APP_ID,required"`
	ApiKey string `env:"ALGOLIA_API_KEY,required"`
	URL    string `env:"ALGOLIA_API_URL" envDefault:"https://places-dsn.algolia.net/1/places/query"`
}

type Mapquest struct {
	ApiKey string `env:"MAPQUEST_API_KEY,required"`
	URL    string `env:"MAPQUEST_API_URL" envDefault:"http://open.mapquestapi.com/nominatim/v1/search.php?"`
}

type AppConfig struct {
	Algolia  *Algolia
	Mapquest *Mapquest
	DB       *DB
}

var Config = &AppConfig{
	Algolia:  &Algolia{},
	Mapquest: &Mapquest{},
	DB:       &DB{},
}

func LoadSettings() error {
	var err error

	if err = env.Parse(Config); err != nil {
		return err
	}
	if err = env.Parse(Config.DB); err != nil {
		return err
	}
	if err = env.Parse(Config.Algolia); err != nil {
		return err
	}
	if err = env.Parse(Config.Mapquest); err != nil {
		return err
	}
	return nil
}
