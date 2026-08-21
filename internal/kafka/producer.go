package kafka

import (
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

// SendEvent отправляет событие в Kafka
func (p *Producer) SendEvent(key string, value interface{}) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(bytes),
	}
	partition, offset, err := p.producer.SendMessage(msg)
	if err == nil {
		log.Printf("Kafka: sent event to topic %s, partition %d, offset %d", p.topic, partition, offset)
	}
	return err
}

func (p *Producer) Close() error {
	return p.producer.Close()
}
