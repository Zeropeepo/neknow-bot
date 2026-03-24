package queue

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Zeropeepo/neknow-bot/pkg/config"
)

const QueueIndexFile = "index_file"

type IndexFileMessage struct {
	FileID    string `json:"file_id"`
	BotID     string `json:"bot_id"`
	ObjectKey string `json:"object_key"`
	MimeType  string `json:"mime_type"`
	FileName  string `json:"file_name"`
}

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(cfg *config.Config) (*RabbitMQ, error) {
	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare queue 
	_, err = ch.QueueDeclare(QueueIndexFile, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{conn: conn, channel: ch}, nil
}

func (r *RabbitMQ) PublishIndexFile(ctx context.Context, msg IndexFileMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return r.channel.PublishWithContext(ctx, "", QueueIndexFile, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // pesan tidak hilang kalau RabbitMQ restart
		},
	)
}

func (r *RabbitMQ) Close() {
	r.channel.Close()
	r.conn.Close()
}
