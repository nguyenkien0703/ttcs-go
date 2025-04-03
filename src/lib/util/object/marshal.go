package lib_util_object

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func Marshal(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	switch string(bytes[:1]) {
	case "[":
		fallthrough
	case "{":
		s := base64.StdEncoding.EncodeToString(bytes)
		return fmt.Sprintf("encoded:%s", s), nil
	default:
		return string(bytes), nil
	}
}

func Unmarshal(src string, dest interface{}) error {
	if len(src) == 0 {
		return nil
	}
	var err error
	var jsonData []byte
	if strings.Index(src, "encoded:") == 0 {
		jsonData, err = base64.StdEncoding.DecodeString(src[8:])
		if err != nil {
			return err
		}
	} else {
		jsonData = []byte(src)
	}
	err = json.Unmarshal(jsonData, dest)
	return err
}
