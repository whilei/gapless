package gapless

import (
	"encoding/hex"
	"encoding/json"
	"github.com/cojac/assert"
	"testing"
	"time"
)

func TestServiceFull(t *testing.T) {
	strData := `{"token": "71c12814d8f7095df0bc4881fcd9163c81aede02c1ebc176a548e03a3943cb14", "identifier": 9, "data": {"aps": {"sound": "default", "badge": 9, "alert": "You got your emails."}, "acme1": "bar", "acme2": 42}, "expiry": 3600}`
	apsData := `{"acme1":"bar","acme2":42,"aps":{"alert":"You got your emails.","badge":9,"sound":"default"}}`
	jParsed := make(map[string]interface{})
	_ = json.Unmarshal([]byte(strData), &jParsed)

	hexTok, _ := hex.DecodeString("71c12814d8f7095df0bc4881fcd9163c81aede02c1ebc176a548e03a3943cb14")
	dur := time.Duration(3600) * time.Second
	result, err := parseApnsJson(jParsed)

	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(9), result.identifier)
	assert.Equal(t, hexTok, result.token)
	assert.Equal(t, dur, result.expiry)
	assert.Equal(t, []byte(apsData), result.jData)
}

func TestServiceMissingData(t *testing.T) {
	strData := `{"token": "71c12814d8f7095df0bc4881fcd9163c81aede02c1ebc176a548e03a3943cb14", "identifier": 9, "expiry": 3600}`
	jParsed := make(map[string]interface{})
	_ = json.Unmarshal([]byte(strData), &jParsed)

	result, err := parseApnsJson(jParsed)

	assert.NotEqual(t, nil, err)
	assert.NotEqual(t, nil, result)
}

func TestServiceBadJson(t *testing.T) {
	strData := `"token": "71c12814d8f7095df0bc4881fcd9163c81aede02c1ebc176a548e03a3943cb14", "identifier": 9, "expiry": 3600}`
	jParsed := make(map[string]interface{})
	_ = json.Unmarshal([]byte(strData), &jParsed)

	_, err := parseApnsJson(jParsed)
	assert.NotEqual(t, nil, err)
}
