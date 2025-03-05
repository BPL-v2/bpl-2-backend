package config

import (
	"bpl/client"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// message object
type StashChangeMessage struct {
	Stashes      []client.PublicStashChange
	ChangeId     string
	NextChangeId string
	Timestamp    time.Time
}

func CreateTopic(eventId int) error {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return fmt.Errorf("KAFKA_BROKER environment variable not set")
	}
	topic := fmt.Sprintf("stash-changes-%d", eventId)

	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
		ConfigEntries: []kafka.ConfigEntry{
			{
				ConfigName:  "max.message.bytes",
				ConfigValue: "1000000000",
			},
			{
				ConfigName:  "compression.type",
				ConfigValue: "zstd",
			},
			// 7 days retention
			{
				ConfigName:  "retention.ms",
				ConfigValue: "604800000",
			},
			{
				ConfigName:  "retention.bytes",
				ConfigValue: "-1",
			},
		},
	}

	return controllerConn.CreateTopics(topicConfig)
}

func GetWriter(eventId int) (*kafka.Writer, error) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return nil, fmt.Errorf("KAFKA_BROKER environment variable not set")
	}
	topic := fmt.Sprintf("stash-changes-%d", eventId)
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{broker},
		Topic:            topic,
		CompressionCodec: kafka.Zstd.Codec(),
		BatchBytes:       1e8,
	}), nil
}

func GetReader(eventId int, consumerId int) (*kafka.Reader, error) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return nil, fmt.Errorf("KAFKA_BROKER environment variable not set")
	}
	topic := fmt.Sprintf("stash-changes-%d", eventId)

	err := CreateTopic(eventId)
	if err != nil {
		return nil, err
	}
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{broker},
		Topic:       topic,
		GroupID:     fmt.Sprintf("%s-%d", topic, consumerId),
		MaxBytes:    1e8,               // 100MB
		StartOffset: kafka.FirstOffset, // Start from the beginning
	}), nil

}
