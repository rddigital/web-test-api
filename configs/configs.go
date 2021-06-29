package configs

type MqttBrokerConfig struct {
	Url            string
	Username       string
	Password       string
	QoS            int8
	TimePubSub     int64
	PublishTopic   string
	SubscribeTopic string
}
