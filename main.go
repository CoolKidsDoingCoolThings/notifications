package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	stan "github.com/nats-io/go-nats-streaming"
	sendgrid "github.com/sendgrid/sendgrid-go"
)

type event struct {
	EventType string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type userRegisteredEvent struct {
	Email string
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sc, err := stan.Connect("test-cluster", "notifications")
	if err != nil {
		panic(err)
	}

	_, err = sc.Subscribe("events", handleMessage, stan.DurableName("notifications"), stan.DeliverAllAvailable(), stan.SetManualAckMode())
	if err != nil {
		panic(err)
	}
	<-sigs
	sc.Close()
}

func handleMessage(m *stan.Msg) {
	fmt.Printf("Received a message: %s\n", string(m.Data))
	var ev event
	err := json.Unmarshal(m.Data, &ev)
	if err != nil {
		log.Println("Unable to unmarshal event:", err)
		return
	}
	switch ev.EventType {
	case "USER_REGISTERED":
		var userRegEv userRegisteredEvent
		err := json.Unmarshal(ev.Payload, &userRegEv)
		if err != nil {
			log.Println("Unable to unmarshal user registered event", err)
			return
		}
		sendNotification(userRegEv.Email)
		err = m.Ack()
		if err != nil {
			log.Println("Unable to ack message", err)
		}
	default:
		return
	}

}

func sendNotification(email string) {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	host := "https://api.sendgrid.com"
	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", host)
	request.Method = "POST"
	request.Body = []byte(fmt.Sprintf(`{
		"personalizations": [{
			"to": [{
				"email": "%s",
			}]
		}],
		"from": {
			"email": "no-reply@coolkids.com",
			"name": "Cool Kids"
		},
		"template_id": "d-e8926f12381d4a7b83625f8503b2af5e"
	}`, email))
	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
