package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type LogTextFormatter struct {
	log.TextFormatter
}

func (f *LogTextFormatter) Format(entry *log.Entry) ([]byte, error) {
	result := entry.Time.Format("15:04:05")
	result = result + " [" + strings.ToUpper(string(entry.Level.String())) + "] (tezpay) " + entry.Message + "\n"
	for k, v := range entry.Data {
		result = result + k + "=" + fmt.Sprint(v) + "\n"
	}
	return []byte(result), nil
}

type LogJsonFormatter struct {
	log.JSONFormatter
}

func (f *LogJsonFormatter) Format(entry *log.Entry) ([]byte, error) {
	//strconv.FormatInt(entry.Time.Unix(), 10)
	l, err := f.JSONFormatter.Format(entry)
	if err != nil {
		return []byte{}, err
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(l, &result)
	if err != nil {
		return []byte{}, err
	}
	delete(result, "time")
	result["timestamp"] = strconv.FormatInt(entry.Time.Unix(), 10)
	result["module"] = "tezpay"
	resultLog, err := json.Marshal(result)
	resultLog = append(resultLog, byte('\n'))
	return resultLog, err
}
