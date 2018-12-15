package messagequeue

import (
	fmt "fmt"
	"github.com/streadway/amqp"
	"flag"
	"log"
)

var (
	uri          = flag.String("uri", "amqp://chen:chen@10.10.10.42:5672/", "AMQP URI")
	exchangeName = flag.String("exchange", "test-exchange", "Durable AMQP exchange name")
	exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	reliable     = flag.Bool("reliable", false, "Wait for the publisher confirmation before exiting")
)

func init() {
	flag.Parse()
}

type MessageQueue struct  {
	connection *amqp.Connection
	channel *amqp.Channel
}


func CreateMQ() (*MessageQueue , error){
	needClose := true
	connection, err := amqp.Dial(*uri)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}
	defer func() {
		if needClose {
			connection.Close()
		}
	}()

	log.Printf("got Connection, getting Channel")
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	log.Printf("got Channel, declaring %q Exchange (%q)", *exchangeType, *exchangeName)

	if err := channel.ExchangeDeclare(
		*exchangeName,     // name
		*exchangeType, // type
		true,         // durable  防止队列丢失
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	// Reliable publisher confirms require confirm.select support from the
	// connection.
	if *reliable {
		log.Printf("enabling publishing confirms.")
		if err := channel.Confirm(false); err != nil {
			return nil, fmt.Errorf("Channel could not be put into confirm mode: %s", err)
		}

		confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

		defer confirmOne(confirms)
	}

	needClose = false

	return &MessageQueue{
		connection,
		channel,
	}, nil
}

func (mq *MessageQueue) Close() {
	mq.connection.Close()
}

func (mq *MessageQueue)Publish(routingKey, body string) error {

	// This function dials, connects, declares, publishes, and tears down,
	// all in one go. In a real service, you probably want to maintain a
	// long-lived connection as state, and publish against that.

	log.Printf("declared Exchange, publishing %dB body (%q)", len(body), body)

	if err := mq.channel.Publish(
		*exchangeName,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Body:            []byte(body),
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

// One would typically keep a channel of publishings, a sequence number, and a
// set of unacknowledged sequence numbers and loop until the publishing channel
// is closed.
func confirmOne(confirms <-chan amqp.Confirmation) {
	log.Printf("waiting for confirmation of one publishing")

	if confirmed := <-confirms; confirmed.Ack {
		log.Printf("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
	} else {
		log.Printf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
	}
}
