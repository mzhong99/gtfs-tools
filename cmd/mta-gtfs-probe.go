package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// exported to outside world
const NycMtaUrl = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace"

func printProtobuf(message proto.Message) {
	options := protojson.MarshalOptions{Multiline: true}
	jsonBytes, err := options.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonBytes))
}

func main() {
	fmt.Println("Hello, world!")

	client := &http.Client{}
	req, _ := http.NewRequest("GET", NycMtaUrl, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	feed := gtfs.FeedMessage{}
	err = proto.Unmarshal(body, &feed)
	if err != nil {
		log.Fatal(err)
	}

	for _, entity := range feed.Entity {
		// tripUpdate := entity.GetTripUpdate()
		// trip := tripUpdate.GetTrip()
		// tripId := trip.GetTripId()
		// fmt.Printf("Trip ID: %s\n", tripId)

		// fmt.Println(entity)
		printProtobuf(entity)
	}
}
