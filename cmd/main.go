package main

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rabbitmq/amqp091-go"
	"log"
	"log/slog"
	"os"
	"rb1-downloader/infrastructure/godotenv"
	"rb1-downloader/service"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	env := godotenv.NewEnv()
	env.Load()
	ampqConn, err := amqp091.Dial(env.AMQP)
	FatalOnError(err)
	defer ampqConn.Close()
	ch, err := ampqConn.Channel()
	defer ch.Close()
	FatalOnError(err)

	err = ch.ExchangeDeclare(env.ImageExchange, "topic", true, false, false, false, nil)
	FatalOnError(err)

	q, err := ch.QueueDeclare(env.QueueName, true, false, false, false, nil)
	FatalOnError(err)

	for _, t := range env.ScrapTopics {
		err = ch.QueueBind(q.Name, "downloaded."+t, env.ImageExchange, false, nil)
		FatalOnError(err)
	}

	minioClient, err := minio.New(env.MinioHost, &minio.Options{
		Creds:  credentials.NewStaticV4(env.MinioAccessToken, env.MinioSecret, ""),
		Secure: false,
	})
	FatalOnError(err)

	c := make(chan bool)
	saver := service.NewSaver(minioClient, logger, ch, env.QueueName, "images")
	if err := saver.Setup(); err != nil {
		log.Fatal(err)
	}
	<-c
}

func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
