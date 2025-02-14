package config

import (
	"bpl/repository"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func CreateTopic(event *repository.Event) error {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return fmt.Errorf("KAFKA_BROKER environment variable not set")
	}
	topic := fmt.Sprintf("stash-changes-%d", event.ID)

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
		},
	}

	return controllerConn.CreateTopics(topicConfig)
}

func GetWriter(event *repository.Event) (*kafka.Writer, error) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return nil, fmt.Errorf("KAFKA_BROKER environment variable not set")
	}
	topic := fmt.Sprintf("stash-changes-%d", event.ID)
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:          []string{broker},
		Topic:            topic,
		CompressionCodec: kafka.Zstd.Codec(),
		BatchBytes:       1e8,
	}), nil
}

func GetReader(event *repository.Event, consumerId int) (*kafka.Reader, error) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return nil, fmt.Errorf("KAFKA_BROKER environment variable not set")
	}
	topic := fmt.Sprintf("stash-changes-%d", event.ID)

	err := CreateTopic(event)
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
