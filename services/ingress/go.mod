module hound-todo/services/ingress

go 1.21

require (
	github.com/rabbitmq/amqp091-go v1.9.0
	github.com/twilio/twilio-go v1.15.0
	hound-todo/shared v0.0.0
)

require (
	github.com/golang/mock v1.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace hound-todo/shared => ../../shared
