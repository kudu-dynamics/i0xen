package natty

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

// The Natty struct represents the data needed to implement a supervised
// service that listens for messages on a NATS subject.
//
// Configuration information is stored to recreate a connection to the NATS
// server when the service crashes.
//
// A buffered channel holds messages drawn from NATS. The service fills this
// channel and discards any others to avoid becoming a slow consumer.
type Natty struct {
	id           string
	conn         *nats.EncodedConn
	subscription *nats.Subscription
	url          string
	subject      string

	// XXX: Handle prioritized channels.
	//      e.g. messages with a request should be prioritized over others
	next chan *nats.Msg
	stop chan bool
}

func New(id, natsUrl, natsSubject string) *Natty {
	return &Natty{
		id,
		nil,
		nil,
		natsUrl,
		natsSubject,
		make(chan *nats.Msg, 1),
		make(chan bool),
	}
}

func (n *Natty) Stop() {
	log.Debug(fmt.Sprintf("natty service [%s] stopping", n.id))
	n.stop <- true
}

func (n *Natty) Serve() {
	// Connect to a NATS server.
	nc, err := nats.Options{
		AllowReconnect: true,
		MaxReconnect:   -1,
		ReconnectWait:  5 * time.Second,
		SubChanLen:     0,
		Timeout:        1 * time.Second,
		Url:            n.url,
	}.Connect()
	if err != nil {
		// XXX: Let the supervisor restart until a connection is made.
		log.Error(err)
		return
	}
	nec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		// XXX: Let the supervisor restart until a connection is made.
		log.Error(err)
		return
	}

	n.conn = nec
	defer n.conn.Close()
	defer n.conn.Drain()

	// Open a subscription.
	sub, err := n.conn.Subscribe(n.subject, func(m *nats.Msg) {
		// XXX: By discarding messages coming in from NATS, we fake the
		//      semblance of a healthy (not slow) consumer.
		// XXX: This actually probably needs to be done such that Natty acquires
		//      1 message at a time.
		select {
		case n.next <- m:
			log.Print("queued up a message")
		default:
			log.Print("discarded message as channel is full")
		}
	})
	if err != nil {
		// XXX: Let the supervisor restart until a subscription is formed.
		log.Print(err)
		return
	}
	n.subscription = sub
	defer n.subscription.Unsubscribe()
	defer n.subscription.Drain()

	for {
		select {
		case <-n.stop:
			return
		}
	}
}
