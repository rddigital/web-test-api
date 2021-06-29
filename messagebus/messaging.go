//
// Copyright (c) 2020 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package messagebus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rddigital/web-test-api/configs"
	"github.com/rddigital/web-test-api/pubsub"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Constants related to the possible content types supported by the APIs
const (
	ContentType     = "Content-Type"
	ContentTypeCBOR = "application/cbor"
	ContentTypeJSON = "application/json"
	ContentTypeYAML = "application/x-yaml"
	ContentTypeText = "text/plain"
)

const ChanSizeDefault = 10

type BusClient struct {
	client  mqtt.Client
	bus     *pubsub.Publisher
	config  configs.MqttBrokerConfig
	timeout int64
}

var busclient BusClient

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	busclient.bus.Publish(msg.Payload())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("MQTT Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connect lost: %v", err)
}

func Initialize(config configs.MqttBrokerConfig) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Url)
	opts.SetClientID("")
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(time.Duration(time.Duration(5).Seconds()))
	opts.SetKeepAlive(time.Duration(time.Duration(10).Seconds()))

	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	timeout := config.TimePubSub
	p := pubsub.NewPublisher(time.Duration(timeout)*time.Second, ChanSizeDefault)

	var c = BusClient{
		client:  client,
		config:  config,
		bus:     p,
		timeout: timeout,
	}

	busclient = c

	return nil
}

func filter(requestID string) func(v interface{}) bool {
	return func(v interface{}) bool {
		msg := v.([]byte)
		repContent, err := DecodeSenML(msg)
		if err != nil {
			return false
		}
		if repContent.RequestID == requestID {
			return true
		}
		return false
	}
}

func (c *BusClient) request(content HTTPContent, timeout int) ([]byte, error) {
	// encode HTTPContent to SenML
	payload := EncodeSenML(content)

	reper := c.bus.SubscribeTopic(filter(content.RequestID))
	defer c.bus.Evict(reper)

	topic := busclient.config.PublishTopic
	qos := busclient.config.QoS
	token := c.client.Publish(topic, byte(qos), false, payload)
	log.Printf("Publish request:%s", string(payload))

	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	topic = CreateResponseTopic(busclient.config.SubscribeTopic, content.RequestID)
	token = c.client.Subscribe(topic, byte(qos), messagePubHandler)
	token.Wait()
	defer func() {
		token = c.client.Unsubscribe(topic)
		token.Wait()
	}()

	var repEnvelope interface{}
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		return nil, fmt.Errorf("wait response timeout")
	case repEnvelope = <-reper:
		// log.Println("Received respond:", string(repEnvelope.([]byte)))
	}

	// Decode SenML to HTTPContent
	// repContent, err := DecodeSenML(repEnvelope.([]byte))
	// return repContent, err
	return repEnvelope.([]byte), nil
}

func MQTTRequestActionHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	requestID := vars["requestID"]
	method := vars["method"]
	path := vars["path"]
	param := r.URL.Query().Encode()
	path = strings.ReplaceAll(path, ":", "/")

	url := CreateRequestURL(method, path)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// create request content
	requestContent := HTTPContent{
		Body:      string(body),
		Url:       url,
		RequestID: requestID,
	}
	if param != "" {
		requestContent.QueryParameters = param
	}
	rep, err := busclient.request(requestContent, int(busclient.timeout))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error send request: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(rep)
}

func MQTTBodyHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	requestID := vars["requestID"]
	method := vars["method"]
	path := vars["path"]
	param := r.URL.Query().Encode()
	path = strings.ReplaceAll(path, ":", "/")

	url := CreateRequestURL(method, path)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// create request content
	requestContent := HTTPContent{
		Body:      string(body),
		Url:       url,
		RequestID: requestID,
	}
	if param != "" {
		requestContent.QueryParameters = param
	}
	// encode HTTPContent to SenML
	payload := EncodeSenML(requestContent)

	w.Write(payload)
}

func GetMQTTResponseTopicHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	requestID := vars["requestID"]

	topic := CreateResponseTopic(busclient.config.SubscribeTopic, requestID)

	w.Write([]byte(topic))
}

func Disconnect() {
	busclient.bus.Close()
	if busclient.client != nil {
		if busclient.client.IsConnected() {
			busclient.client.Disconnect(0)
		}
	}
}

func ConnectHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	var bodyConfig configs.MqttBrokerConfig
	err = json.Unmarshal(body, &bodyConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error Unmarshal: %s", err.Error()), http.StatusInternalServerError)
		// log.Println(err.Error())
		return
	}

	if bodyConfig.Url == "0" {
		Disconnect()
		log.Println("MQTT Disconnected")
	} else {
		if busclient.client != nil {
			if !busclient.client.IsConnected() {
				err = Initialize(bodyConfig)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error Initialize: %v", err), http.StatusInternalServerError)
					return
				}
			}
		} else {
			err = Initialize(bodyConfig)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error Initialize: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}
	w.Write([]byte("Success"))
}
