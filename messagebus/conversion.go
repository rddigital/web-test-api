package messagebus

import (
	"encoding/json"
	"fmt"
	"strings"
)

/*
RestAPI.Request: {METHOD} {api}	{Body}
```
MQTT.Request:
	Topic: 		'{prefix_request}'
	Message: 	'[{"bn":"{UUID}", "n": "{Method}/{ServiceName}/{Api}", "vs": "{Body}"}]'

	Example: prefix_request = 'channels/{channelId}/messages/req'
```

```
MQTT.Response:
	Topic:		'{prefix_response}/{UUID}'
	Message:	'[{"bn":"{UUID}", "n":"{StatusCode}", "vs":"{Response/Error}"}]'

	Example: prefix_response = 'channels/{channelId}/messages/res/{THING_KEY}'
```
*/
type HTTPContent struct {
	RequestID       string
	Url             string
	Body            string
	QueryParameters string
	StatusCode      int
}

func DecodeSenML(body []byte) (HTTPContent, error) {
	var httpContent HTTPContent

	type Alias struct {
		BaseName    string `json:"bn"`
		Name        string `json:"n"`
		StringValue string `json:"vs"`
	}

	var records []Alias
	err := json.Unmarshal(body, &records)
	if err != nil {
		return httpContent, err
	}
	if len(records) < 1 {
		return httpContent, fmt.Errorf("Invalid input data")
	}

	var record = records[0]
	_, err = fmt.Sscanf(record.Name, "%d", &httpContent.StatusCode)
	if err != nil {
		return httpContent, err
	}
	httpContent.RequestID = getUUID(record.BaseName)
	httpContent.Body = record.StringValue

	return httpContent, nil
}

func EncodeSenML(content HTTPContent) []byte {
	if content.Body == "" {
		content.Body = "{}"
	}

	type Alias struct {
		BaseName    string `json:"bn"`
		BaseVersion int    `json:"bver"`
		Name        string `json:"n"`
		StringValue string `json:"vs"`
		BaseUnit    string `json:"bu,omitempty"`
	}

	var records []Alias = []Alias{{
		BaseName:    setUUIDSenML(content.RequestID),
		BaseVersion: 2,
		Name:        content.Url,
		StringValue: content.Body,
		BaseUnit:    content.QueryParameters,
	}}

	dataOut, _ := json.Marshal(records)

	return dataOut
}

func CreateRequestURL(method string, api string) string {
	return fmt.Sprintf("%s/%s", strings.ToUpper(method), api)
}

func CreateResponseTopic(prefix string, requestID string) string {
	return fmt.Sprintf("%s/%s", prefix, requestID)
}

// getUUID remove ":" in end of originID
func getUUID(originID string) string {
	arrStr := strings.Split(originID, ":")
	return arrStr[0]
}

// setUUIDSenML add ":" in end of uuid
func setUUIDSenML(uuid string) string {
	return fmt.Sprintf("%s:", uuid)
}
