package service

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/rabbitmq/amqp091-go"
	"log/slog"
	"rb1-downloader/entity"
	"time"
)

type Saver struct {
	minC      *minio.Client
	queue     <-chan amqp091.Delivery
	logger    *slog.Logger
	ch        *amqp091.Channel
	queueName string
	exchange  string
}

func NewSaver(minC *minio.Client, logger *slog.Logger, ch *amqp091.Channel, queueName string, exchange string) *Saver {
	s := &Saver{
		minC:      minC,
		logger:    logger,
		ch:        ch,
		queueName: queueName,
		exchange:  exchange,
	}
	return s
}

func (s *Saver) Setup() error {
	deliveries, err := s.ch.Consume(s.queueName, "", false, false, false, false, nil)
	s.queue = deliveries
	for range 50 {
		go s.Worker()
	}
	return err
}

func (s *Saver) Worker() {
	for msg := range s.queue {
		image, err := entity.ImageFromJSON(msg.Body)
		if err != nil {
			s.logger.Error("Error:%s", err.Error())
			return
		}
		obj, err := s.minC.GetObject(context.Background(), image.BucketName, image.Name, minio.GetObjectOptions{})
		if err != nil {
			s.logger.Error("Error getting object from MinIO:%s", err.Error())
			return
		}
		defer obj.Close()
		imageByte := make([]byte, 0)
		if _, err := obj.Read(imageByte); err != nil {
			s.logger.Error("Error reading object from MinIO:%s", err.Error())
		}

		s.logger.Info("saved successfully", time.Now().UnixNano())
	}
}
