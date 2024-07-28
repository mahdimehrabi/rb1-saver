package service

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/rabbitmq/amqp091-go"
	"io"
	"log/slog"
	"os"
	"path"
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
		s.Save(msg)
	}
}

func (s *Saver) Save(msg amqp091.Delivery) {
	image, err := entity.ImageFromJSON(msg.Body)
	if err != nil {
		s.handleError(msg, "Error image from json:%s", err)
		return
	}

	obj, err := s.minC.GetObject(context.Background(), image.BucketName, image.Name, minio.GetObjectOptions{})
	if err != nil {
		s.handleError(msg, "Error reading object from minIO:%s", err)
		return
	}
	defer obj.Close()
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, obj); err != nil {
		s.handleError(msg, "Error reading object from minIO:%s", err)
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		s.handleError(msg, "Error getting current directory:%s", err)
		return
	}

	dir = path.Join(dir, "images", image.Name+".jpg")
	f, err := os.OpenFile(dir, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		s.handleError(msg, "Error opening directory:%s", err)
		return
	}
	defer f.Close()

	_, err = f.Write(buf.Bytes())
	if err != nil {
		s.handleError(msg, "Error saving image Error:%s", err)
		return
	}

	if err := msg.Ack(false); err != nil {
		s.logger.Error("failed to ack message: %v", err)
		return
	}
	s.logger.Info("saved successfully", time.Now().UnixNano())
}

func (s *Saver) handleError(msg amqp091.Delivery, msgStr string, err error) {
	s.logger.Error(msgStr, err.Error())
	if err := msg.Nack(false, false); err != nil {
		s.logger.Error("failed to nack message: %v", err)
		return
	}
}
