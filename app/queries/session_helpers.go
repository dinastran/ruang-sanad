package queries

import (
	"encoding/json"

	"github.com/maulanashalihin/laju-go/app/models"
)

func sessionDataFromJSON(data string, out *models.SessionData) error {
	return json.Unmarshal([]byte(data), out)
}

func sessionDataToJSON(data *models.SessionData) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
