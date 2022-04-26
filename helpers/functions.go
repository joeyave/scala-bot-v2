package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func AddCallbackData(message string, url string) string {
	message = fmt.Sprintf("%s\n<a href=\"%s\">&#8203;</a>", message, url)
	return message
}

func ParseCallbackData(data string) (int, int, string) {

	parsedData := strings.Split(data, ":")
	stateStr := parsedData[0]
	indexStr := parsedData[1]
	payload := strings.Join(parsedData[2:], ":")

	state, err := strconv.Atoi(stateStr)
	if err != nil {
		state = 0 // TODO
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		index = 0 // TODO
	}

	return state, index, payload
}

func AggregateCallbackData(state int, index int, payload string) string {
	return fmt.Sprintf("%d:%d:%s", state, index, payload)
}

func JsonEscape(i string) string {

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(i)
	if err != nil {
		panic(err)
	}

	buffer.Bytes()

	b := bytes.Trim(bytes.TrimSpace(buffer.Bytes()), `"`)

	return string(b)
}
