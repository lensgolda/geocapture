package models

type City struct {
	ID           int
	Name         *string
	NameNational *string
}

type Country struct {
	ID     int
	Name   *string
	NameEN *string
}

type AltName struct {
	NameRu  *string `json:"name:ru"`
	NameEn  *string `json:"name:en"`
	NameKk  *string `json:"name:kk"`
	NameUk  *string `json:"name:uk"`
	IntName *string `json:"int_name"`
}

type Location struct {
	Namedetail AltName `json:"namedetails"`
}

type NomResult []Location

type Hit struct {
	IsCity      bool `json:"is_city"`
	IsCountry   bool `json:"is_country"`
	LocaleNames map[string][]string
}
type AlgResult struct {
	Hit []Hit `json:"hits"`
}

type Model interface {
	Type() string
	Id() int
}

func (m Country) Id() int {
	return m.ID
}

func (m Country) Type() string {
	return "country"
}

func (m City) Id() int {
	return m.ID
}

func (m City) Type() string {
	return "city"
}
