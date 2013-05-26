package gapless

import (
    "encoding/json"
    "github.com/bmizerany/assert"
    "testing"
)

func TestSettingsSetFromEnv(t *testing.T) {
    dictObj := NewSettingsObj()

    dictObj.SetFromEnv("a", "bad_key_here", true)
    assert.Equal(t, true, dictObj.Bool("a"))

    dictObj.SetFromEnv("a", "bad_key_here", false)
    assert.Equal(t, false, dictObj.Bool("a"))

    dictObj.SetFromEnv("debug", "bad_key_here222", true)

    assert.Panic(t, "SetFromEnv received too many args: [2]", func() {
        dictObj.SetFromEnv("z", "bbaadddddd", false, true)
    })
}

func TestDictObjJsonFile(t *testing.T) {
    dictObj := NewSettingsObj()

    assert.Panic(t, "Opening settings file failed: open not_there.golang: no such file or directory", func() {
        dictObj.LoadFromFile("not_there.golang")
    })
}

func TestDictObjJsonData(t *testing.T) {
    tmp := make(map[string]interface{})

    err := json.Unmarshal([]byte(`{"intOne":1,"intTwo":23456789}`), &tmp)
    if err != nil {
        t.Errorf("Json unmarshal error: %q", err)
    }

    dictObj := NewSettingsObj()

    dictObj.Set("a", tmp["intOne"])
    dictObj.Set("b", tmp["intTwo"])

    assert.Equal(t, 1, dictObj.Int("a"))
    assert.Equal(t, 23456789, dictObj.Int("b"))
}

func TestDictObjBool(t *testing.T) {
    dictObj := NewSettingsObj()

    dictObj.Set("a", true)
    dictObj.Set("b", false)

    assert.Equal(t, true, dictObj.Bool("a"))
    assert.Equal(t, false, dictObj.Bool("b"))
    assert.Equal(t, true, dictObj.Bool("c", true))
    assert.Equal(t, false, dictObj.Bool("d", false))

    dictObj.Set("a", false)
    assert.Equal(t, false, dictObj.Bool("a"))

    assert.Panic(t, "Bool received too many args: [3]", func() {
        dictObj.Bool("z", true, false, true)
    })
}

func TestDictObjInt(t *testing.T) {
    dictObj := NewSettingsObj()

    dictObj.Set("a", 1)
    dictObj.Set("b", 12345678901234)

    assert.Equal(t, 1, dictObj.Int("a"))
    assert.Equal(t, 12345678901234, dictObj.Int("b"))

    dictObj.Set("a", 2)

    assert.Equal(t, 2, dictObj.Int("a"))
    assert.Equal(t, 55, dictObj.Int("c", 55))

    assert.Panic(t, "Int received too many args: [2]", func() {
        dictObj.Int("z", 12, 34)
    })
}

func TestDictObjFloat(t *testing.T) {
    dictObj := NewSettingsObj()

    dictObj.Set("a", 1.0)
    dictObj.Set("b", 12345678901234.56789)

    assert.Equal(t, 1.0, dictObj.Float("a"))
    assert.Equal(t, 12345678901234.56789, dictObj.Float("b"))

    dictObj.Set("a", 222.333)

    assert.Equal(t, 222.333, dictObj.Float("a"))
    assert.Equal(t, 55.667788, dictObj.Float("c", 55.667788))

    assert.Panic(t, "Float received too many args: [2]", func() {
        dictObj.Float("z", 12.23, 33.22)
    })
}

func TestDictObjString(t *testing.T) {
    dictObj := NewSettingsObj()

    dictObj.Set("a", "AAA")
    dictObj.Set("b", "BBB")

    assert.Equal(t, "AAA", dictObj.String("a"))
    assert.Equal(t, "BBB", dictObj.String("b"))

    dictObj.Set("a", "not AAA")

    assert.Equal(t, "not AAA", dictObj.String("a"))
    assert.Equal(t, "CCC", dictObj.String("c", "CCC"))

    assert.Panic(t, "String received too many args: [2]", func() {
        dictObj.String("z", "default", "bad")
    })
}
