package fatsecret

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/andre487/data-migrators/utils/misc"
	"github.com/andre487/data-migrators/utils/parsing"
)

const apiUrl = "https://platform.fatsecret.com/rest/server.api"

type FatSecret struct {
	oauth *FatSOauth1Service
}

func New(keyData []byte, options ...func(s *FatSecret)) (*FatSecret, error) {
	keys := FatSOauth1Keys{}
	if err := json.Unmarshal(keyData, &keys); err != nil {
		return nil, fmt.Errorf("error when parsing FatSecret keys: %v", err)
	}
	oauth := NewFatSOauth1Service(keys)
	err := oauth.GetRequestToken()
	if err != nil {
		log.Fatal(err)
	}
	err = oauth.GetAuthCode()
	if err != nil {
		log.Fatal(err)
	}

	p := &FatSecret{oauth: oauth}
	for _, opt := range options {
		opt(p)
	}
	return p, nil
}

func (s *FatSecret) makeApiRequest(method string, reqData map[string]string) (map[string]interface{}, error) {
	reqBodyParams := url.Values{"method": []string{method}, "format": []string{"json"}}
	for key, value := range reqData {
		reqBodyParams.Set(key, value)
	}
	resp, respBody, err := s.oauth.MakeHttpRequest("POST", apiUrl, reqBodyParams)
	if err != nil {
		return nil, fmt.Errorf("error when making request for method %s: %v", method, err)
	}

	if resp.StatusCode > 201 {
		return nil, fmt.Errorf("error when making API request, HTTP %d: method=%s, %v", resp.StatusCode, method, string(respBody))
	}

	var bodyData map[string]interface{}
	if err := json.Unmarshal(respBody, &bodyData); err != nil {
		return nil, fmt.Errorf("error when requesting token, invalid body: %v", err)
	}

	if bodyData["error"] != nil {
		apiErr := bodyData["error"].(map[string]interface{})
		errCode := uint64(apiErr["code"].(float64))
		errMsg := apiErr["message"].(string)
		return nil, fmt.Errorf("API error: method=%s, code=%d: %s", method, errCode, errMsg)
	}

	return bodyData, nil
}

type FoodEntriesDataRaw struct {
	FoodEntries struct {
		FoodEntry []FoodEntryDataRaw     `mapstructure:"food_entry"`
		Other     map[string]interface{} `mapstructure:",remain"`
	} `mapstructure:"food_entries"`
	Other map[string]interface{} `mapstructure:",remain"`
}

type FoodEntriesData struct {
	FoodEntries struct {
		FoodEntry []FoodEntryData
	}
}

type FoodEntryDataRaw struct {
	DateInt              string `mapstructure:"date_int"`
	FoodId               string `mapstructure:"food_id"`
	FoodEntryId          string `mapstructure:"food_entry_id"`
	ServingId            string `mapstructure:"serving_id"`
	FoodEntryName        string `mapstructure:"food_entry_name"`
	FoodEntryDescription string `mapstructure:"food_entry_description"`
	NumberOfUnits        string `mapstructure:"number_of_units"`
	Meal                 string
	Protein              string
	Calories             string
	Carbohydrate         string
	Fat                  string
	Fiber                string
	Sugar                string
	Calcium              string
	Cholesterol          string
	Iron                 string
	MonounsaturatedFat   string `mapstructure:"monounsaturated_fat"`
	PolyunsaturatedFat   string `mapstructure:"polyunsaturated_fat"`
	SaturatedFat         string `mapstructure:"saturated_fat"`
	TransFat             string `mapstructure:"trans_fat"`
	VitaminA             string `mapstructure:"vitamin_a"`
	VitaminC             string `mapstructure:"vitamin_c"`
	Sodium               string
	Potassium            string
	Other                map[string]interface{} `mapstructure:",remain"`
}

type FoodEntryData struct {
	DateInt              int64
	FoodId               int64
	FoodEntryId          int64
	ServingId            int64
	FoodEntryName        string
	FoodEntryDescription string
	NumberOfUnits        float64
	Meal                 string
	Protein              float64
	Calories             float64
	Carbohydrate         float64
	Fat                  float64
	Fiber                float64
	Sugar                float64
	Calcium              float64
	Cholesterol          float64
	Iron                 float64
	MonounsaturatedFat   float64
	PolyunsaturatedFat   float64
	SaturatedFat         float64
	TransFat             float64
	VitaminA             float64
	VitaminC             float64
	Sodium               float64
	Potassium            float64
}

func FoodEntriesDataFromRaw(rawData FoodEntriesDataRaw) (*FoodEntriesData, error) {
	if len(rawData.Other) > 0 {
		log.Printf("WARN: FoodEntriesDataRaw.Other is not empty: %v\n", rawData.Other)
	}

	if len(rawData.FoodEntries.Other) > 0 {
		log.Printf("WARN: FoodEntriesDataRaw.FoodEntries.Other is not empty: %v\n", rawData.FoodEntries.Other)
	}

	res := FoodEntriesData{}
	for _, item := range rawData.FoodEntries.FoodEntry {
		if len(item.Other) > 0 {
			log.Printf("WARN: FoodEntryData.Other is not empty: %v\n", item.Other)
		}

		var err error
		var dateInt int64
		var foodId int64
		var foodEntryId int64
		var servingId int64
		var numberOfUnits float64
		var protein float64
		var calories float64
		var carbohydrate float64
		var fat float64
		var fiber float64
		var sugar float64
		var calcium float64
		var cholesterol float64
		var iron float64
		var monounsaturatedFat float64
		var polyunsaturatedFat float64
		var saturatedFat float64
		var transFat float64
		var vitaminA float64
		var vitaminC float64
		var sodium float64
		var potassium float64

		if dateInt, err = parsing.ParseInt64(item.DateInt); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw DateInt: %v", err)
		}
		if foodId, err = parsing.ParseInt64(item.FoodId); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw FoodId: %v", err)
		}
		if foodEntryId, err = parsing.ParseInt64(item.FoodEntryId); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw FoodEntryId: %v", err)
		}
		if servingId, err = parsing.ParseInt64(item.ServingId); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw ServingId: %v", err)
		}
		if numberOfUnits, err = parsing.ParseFloat64(item.NumberOfUnits); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw NumberOfUnits: %v", err)
		}
		if protein, err = parsing.ParseFloat64(item.Protein); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Protein: %v", err)
		}
		if calories, err = parsing.ParseFloat64(item.Calories); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Calories: %v", err)
		}
		if carbohydrate, err = parsing.ParseFloat64(item.Carbohydrate); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Carbohydrate: %v", err)
		}
		if fat, err = parsing.ParseFloat64(item.Fat); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Fat: %v", err)
		}
		if fiber, err = parsing.ParseFloat64(item.Fiber); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Fiber: %v", err)
		}
		if sugar, err = parsing.ParseFloat64(item.Sugar); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Sugar: %v", err)
		}
		if calcium, err = parsing.ParseFloat64(item.Calcium); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Calcium: %v", err)
		}
		if cholesterol, err = parsing.ParseFloat64(item.Cholesterol); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Cholesterol: %v", err)
		}
		if iron, err = parsing.ParseFloat64(item.Iron); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Iron: %v", err)
		}
		if monounsaturatedFat, err = parsing.ParseFloat64(item.MonounsaturatedFat); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw MonounsaturatedFat: %v", err)
		}
		if polyunsaturatedFat, err = parsing.ParseFloat64(item.PolyunsaturatedFat); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw PolyunsaturatedFat: %v", err)
		}
		if saturatedFat, err = parsing.ParseFloat64(item.SaturatedFat); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw SaturatedFat: %v", err)
		}
		if transFat, err = parsing.ParseFloat64(item.TransFat); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw TransFat: %v", err)
		}
		if vitaminA, err = parsing.ParseFloat64(item.VitaminA); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw VitaminA: %v", err)
		}
		if vitaminC, err = parsing.ParseFloat64(item.VitaminC); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw VitaminC: %v", err)
		}
		if sodium, err = parsing.ParseFloat64(item.Sodium); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Sodium: %v", err)
		}
		if potassium, err = parsing.ParseFloat64(item.Potassium); err != nil {
			return nil, fmt.Errorf("error when parsing FoodEntriesDataRaw Potassium: %v", err)
		}

		res.FoodEntries.FoodEntry = append(res.FoodEntries.FoodEntry, FoodEntryData{
			DateInt:              dateInt,
			FoodId:               foodId,
			FoodEntryId:          foodEntryId,
			ServingId:            servingId,
			FoodEntryName:        item.FoodEntryName,
			FoodEntryDescription: item.FoodEntryDescription,
			NumberOfUnits:        numberOfUnits,
			Meal:                 item.Meal,
			Calories:             calories,
			Protein:              protein,
			Carbohydrate:         carbohydrate,
			Fat:                  fat,
			Fiber:                fiber,
			Sugar:                sugar,
			Calcium:              calcium,
			Cholesterol:          cholesterol,
			Iron:                 iron,
			MonounsaturatedFat:   monounsaturatedFat,
			PolyunsaturatedFat:   polyunsaturatedFat,
			SaturatedFat:         saturatedFat,
			TransFat:             transFat,
			VitaminA:             vitaminA,
			VitaminC:             vitaminC,
			Sodium:               sodium,
			Potassium:            potassium,
		})
	}

	return &res, nil
}

func (s *FatSecret) FoodEntriesGet(date time.Time) (*FoodEntriesData, error) {
	if err := s.oauth.Authorize(); err != nil {
		return nil, fmt.Errorf("auth error: %v", err)
	}

	days := misc.DatToDaysFromEpoch(date)
	reqData := map[string]string{"date": strconv.FormatInt(days, 10)}
	rawData, err := s.makeApiRequest("food_entries.get.v2", reqData)
	if err != nil {
		return nil, fmt.Errorf("error when requesting food entries: %v", err)
	}

	rawRes := FoodEntriesDataRaw{}
	if err := mapstructure.Decode(rawData, &rawRes); err != nil {
		return nil, fmt.Errorf("error when parsing response of food_entries.get.v2: %v", err)
	}

	return FoodEntriesDataFromRaw(rawRes)
}

func (s *FatSecret) GetTestData() (*FoodEntriesData, error) {
	return s.FoodEntriesGet(time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC))
}
