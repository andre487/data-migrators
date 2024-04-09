package fatsecret

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strconv"
	"strings"
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
		log.Fatalf("FatSecret: %v\n", err)
	}
	err = oauth.GetAuthCode()
	if err != nil {
		log.Fatalf("FatSecret: %v\n", err)
	}

	p := &FatSecret{oauth: oauth}
	for _, opt := range options {
		opt(p)
	}
	return p, nil
}

type ApiRequestRetryConfig struct {
	Retries     int
	Backoff     time.Duration
	MaxTimeout  time.Duration
	RetryNumber int
}

func (s *FatSecret) makeApiRequest(method string, reqData map[string]string, retryConfig *ApiRequestRetryConfig) (map[string]interface{}, error) {
	if retryConfig == nil {
		retryConfig = &ApiRequestRetryConfig{
			Retries:    5,
			Backoff:    time.Second * 10,
			MaxTimeout: time.Second * 120,
		}
	}

	reqBodyParams := url.Values{"method": []string{method}, "format": []string{"json"}}
	for key, value := range reqData {
		reqBodyParams.Set(key, value)
	}
	resp, respBody, err := s.oauth.MakeHttpRequest("POST", apiUrl, reqBodyParams)
	if err != nil {
		return nil, fmt.Errorf("FatSecret: error when making request for method %s: %v", method, err)
	}

	if resp.StatusCode > 201 {
		return nil, fmt.Errorf("FatSecret: error when making API request, HTTP %d: method=%s, %v", resp.StatusCode, method, string(respBody))
	}

	var bodyData map[string]interface{}
	if err := json.Unmarshal(respBody, &bodyData); err != nil {
		return nil, fmt.Errorf("FatSecret: error when requesting token, invalid body: %v", err)
	}

	if bodyData["error"] != nil {
		apiErr := bodyData["error"].(map[string]interface{})
		errCode := uint64(apiErr["code"].(float64))
		errMsg := apiErr["message"].(string)

		if retryConfig.Retries > 0 && retryConfig.RetryNumber < retryConfig.Retries && isRetryableApiError(errMsg) {
			waitTime := retryConfig.Backoff * time.Duration(math.Pow(2, float64(retryConfig.RetryNumber)))
			retryConfig.RetryNumber++
			log.Printf("WARN: FatSecret: Retriable API error: %s", errMsg)
			log.Printf("WARN: FatSecret: Retry API request because of error, retryNumber=%d, waitTime=%s", retryConfig.RetryNumber, waitTime.String())
			time.Sleep(waitTime)
			return s.makeApiRequest(method, reqData, retryConfig)
		}

		return nil, fmt.Errorf("FatSecret: API error: method=%s, code=%d: %s", method, errCode, errMsg)
	}

	return bodyData, nil
}

func isRetryableApiError(msg string) bool {
	if strings.Contains(msg, "User is performing too many actions") {
		return true
	}
	return false
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
	Date                 time.Time
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
		log.Printf("WARN: FatSecret: FoodEntriesDataRaw.Other is not empty: %v\n", rawData.Other)
	}

	if len(rawData.FoodEntries.Other) > 0 {
		log.Printf("WARN: FatSecret: FoodEntriesDataRaw.FoodEntries.Other is not empty: %v\n", rawData.FoodEntries.Other)
	}

	res := FoodEntriesData{}
	for _, item := range rawData.FoodEntries.FoodEntry {
		if len(item.Other) > 0 {
			log.Printf("WARN: FatSecret: FoodEntryData.Other is not empty: %v\n", item.Other)
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
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw DateInt: %v", err)
		}
		if foodId, err = parsing.ParseInt64(item.FoodId); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw FoodId: %v", err)
		}
		if foodEntryId, err = parsing.ParseInt64(item.FoodEntryId); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw FoodEntryId: %v", err)
		}
		if servingId, err = parsing.ParseInt64(item.ServingId); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw ServingId: %v", err)
		}
		if numberOfUnits, err = parsing.ParseFloat64(item.NumberOfUnits); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw NumberOfUnits: %v", err)
		}
		if protein, err = parsing.ParseFloat64(item.Protein); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Protein: %v", err)
		}
		if calories, err = parsing.ParseFloat64(item.Calories); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Calories: %v", err)
		}
		if carbohydrate, err = parsing.ParseFloat64(item.Carbohydrate); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Carbohydrate: %v", err)
		}
		if fat, err = parsing.ParseFloat64(item.Fat); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Fat: %v", err)
		}
		if fiber, err = parsing.ParseFloat64(item.Fiber); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Fiber: %v", err)
		}
		if sugar, err = parsing.ParseFloat64(item.Sugar); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Sugar: %v", err)
		}
		if calcium, err = parsing.ParseFloat64(item.Calcium); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Calcium: %v", err)
		}
		if cholesterol, err = parsing.ParseFloat64(item.Cholesterol); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Cholesterol: %v", err)
		}
		if iron, err = parsing.ParseFloat64(item.Iron); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Iron: %v", err)
		}
		if monounsaturatedFat, err = parsing.ParseFloat64(item.MonounsaturatedFat); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw MonounsaturatedFat: %v", err)
		}
		if polyunsaturatedFat, err = parsing.ParseFloat64(item.PolyunsaturatedFat); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw PolyunsaturatedFat: %v", err)
		}
		if saturatedFat, err = parsing.ParseFloat64(item.SaturatedFat); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw SaturatedFat: %v", err)
		}
		if transFat, err = parsing.ParseFloat64(item.TransFat); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw TransFat: %v", err)
		}
		if vitaminA, err = parsing.ParseFloat64(item.VitaminA); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw VitaminA: %v", err)
		}
		if vitaminC, err = parsing.ParseFloat64(item.VitaminC); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw VitaminC: %v", err)
		}
		if sodium, err = parsing.ParseFloat64(item.Sodium); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Sodium: %v", err)
		}
		if potassium, err = parsing.ParseFloat64(item.Potassium); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesDataRaw Potassium: %v", err)
		}

		res.FoodEntries.FoodEntry = append(res.FoodEntries.FoodEntry, FoodEntryData{
			DateInt:              dateInt,
			Date:                 misc.DaysFromEpochToDate(dateInt),
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
		return nil, fmt.Errorf("FatSecret: auth error: %v", err)
	}

	days := misc.DateToDaysFromEpoch(date)
	reqData := map[string]string{"date": strconv.FormatInt(days, 10)}
	rawData, err := s.makeApiRequest("food_entries.get.v2", reqData, nil)
	if err != nil {
		return nil, fmt.Errorf("FatSecret: error when requesting food entries: %v", err)
	}

	rawRes := FoodEntriesDataRaw{}
	if err := mapstructure.Decode(rawData, &rawRes); err != nil {
		return nil, fmt.Errorf("FatSecret: error when parsing response of food_entries.get.v2: %v", err)
	}

	return FoodEntriesDataFromRaw(rawRes)
}

type FoodEntriesMonthDataRaw struct {
	Month struct {
		Day         []FoodEntryDayDataRaw
		FromDateInt string                 `mapstructure:"from_date_int"`
		ToDateInt   string                 `mapstructure:"to_date_int"`
		Other       map[string]interface{} `mapstructure:",remain"`
	}
	Other map[string]interface{} `mapstructure:",remain"`
}

type FoodEntriesMonthData struct {
	Month struct {
		Day         []FoodEntryDayData
		FromDateInt int64
		ToDateInt   int64
		FromDate    time.Time
		ToDate      time.Time
	}
}

type FoodEntryDayDataRaw struct {
	DateInt      string `mapstructure:"date_int"`
	Calories     string
	Carbohydrate string
	Fat          string
	Protein      string
	Other        map[string]interface{} `mapstructure:",remain"`
}

type FoodEntryDayData struct {
	DateInt      int64
	Date         time.Time
	Calories     float64
	Carbohydrate float64
	Fat          float64
	Protein      float64
}

func FoodEntriesMonthDataFromRaw(rawData *FoodEntriesMonthDataRaw) (*FoodEntriesMonthData, error) {
	if len(rawData.Other) > 0 {
		log.Printf("WARN: FatSecret: FoodEntriesMonthDataRaw.Other is not empty: %v\n", rawData.Other)
	}

	if len(rawData.Month.Other) > 0 {
		log.Printf("WARN: FatSecret: FoodEntriesMonthDataRaw.Month.Other is not empty: %v\n", rawData.Month.Other)
	}

	var err error
	var fromDateInt int64
	var toDateInt int64
	if fromDateInt, err = parsing.ParseInt64(rawData.Month.FromDateInt); err != nil {
		return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesMonthDataRaw FromDateInt: %v", err)
	}
	if toDateInt, err = parsing.ParseInt64(rawData.Month.ToDateInt); err != nil {
		return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesMonthDataRaw ToDateInt: %v", err)
	}

	res := FoodEntriesMonthData{}

	res.Month.FromDateInt = fromDateInt
	res.Month.ToDateInt = toDateInt
	res.Month.FromDate = misc.DaysFromEpochToDate(fromDateInt)
	res.Month.ToDate = misc.DaysFromEpochToDate(toDateInt)

	for _, item := range rawData.Month.Day {
		if len(item.Other) > 0 {
			log.Printf("WARN: FatSecret: FoodEntryDayData.Other is not empty: %v\n", item.Other)
		}

		var err error
		var dateInt int64
		var calories float64
		var carbohydrate float64
		var fat float64
		var protein float64

		if dateInt, err = parsing.ParseInt64(item.DateInt); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesMonthDataRaw DateInt: %v", err)
		}
		if calories, err = parsing.ParseFloat64(item.Calories); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesMonthDataRaw Calories: %v", err)
		}
		if fat, err = parsing.ParseFloat64(item.Fat); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesMonthDataRaw Fat: %v", err)
		}
		if protein, err = parsing.ParseFloat64(item.Protein); err != nil {
			return nil, fmt.Errorf("FatSecret: error when parsing FoodEntriesMonthDataRaw Protein: %v", err)
		}

		res.Month.Day = append(res.Month.Day, FoodEntryDayData{
			DateInt:      dateInt,
			Date:         misc.DaysFromEpochToDate(dateInt),
			Calories:     calories,
			Carbohydrate: carbohydrate,
			Fat:          fat,
			Protein:      protein,
		})
	}

	return &res, nil
}

func (s *FatSecret) FoodEntriesGetMonth(fromDate time.Time) (*FoodEntriesMonthData, error) {
	if err := s.oauth.Authorize(); err != nil {
		return nil, fmt.Errorf("FatSecret: auth error: %v", err)
	}

	days := misc.DateToDaysFromEpoch(fromDate)
	reqData := map[string]string{"date": strconv.FormatInt(days, 10)}
	rawData, err := s.makeApiRequest("food_entries.get_month.v2", reqData, nil)
	if err != nil {
		return nil, fmt.Errorf("FatSecret: error when requesting food entries for month: %v", err)
	}

	rawRes := FoodEntriesMonthDataRaw{}
	if err := mapstructure.Decode(rawData, &rawRes); err != nil {
		return nil, fmt.Errorf("FatSecret: error when parsing response of food_entries.get.v2: %v", err)
	}

	return FoodEntriesMonthDataFromRaw(&rawRes)
}

type DiaryData struct {
	FromDate          time.Time
	ToDate            time.Time
	AggregatedDayData []FoodEntryDayData
	DiaryData         []FoodEntryData
}

func (s *FatSecret) GetDiary(fromDate time.Time, toDate time.Time) (*DiaryData, error) {
	delta := toDate.Sub(fromDate)
	if delta < 0 {
		return nil, errors.New("FatSecret: GetDiary: fromDate > toDate")
	}

	res := DiaryData{}

	var dates []time.Time
	curDate := fromDate
	for {
		log.Printf("FatSecret: Get diary data for month from %v\n", curDate)
		monthData, err := s.FoodEntriesGetMonth(curDate)
		if err != nil {
			return nil, err
		}

		for _, item := range monthData.Month.Day {
			dates = append(dates, item.Date)
			res.AggregatedDayData = append(res.AggregatedDayData, item)
		}

		curDate = monthData.Month.ToDate.Add(time.Hour * 24)
		if toDate.Sub(curDate) < 0 {
			break
		}
		time.Sleep(time.Second)
	}

	if len(dates) > 0 {
		res.FromDate = dates[0]
		res.ToDate = dates[len(dates)-1]
	}

	for _, date := range dates {
		log.Printf("FatSecret: Get diary food entries for date %v\n", date)
		data, err := s.FoodEntriesGet(date)
		if err != nil {
			return nil, err
		}

		for _, foodEntry := range data.FoodEntries.FoodEntry {
			res.DiaryData = append(res.DiaryData, foodEntry)
		}
		time.Sleep(time.Second)
	}

	return &res, nil
}

func (s *FatSecret) GetTestData() (interface{}, error) {
	return s.GetDiary(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC))
}
