package interfaces

import (
	"database/sql"
)

type Provider interface {
	CountryNameLocalize(db *sql.DB)
	CityNameLocalize(db *sql.DB)
	FailedFile() string
}
