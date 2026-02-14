package gtfs_rt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type GtfsRtWatcher struct {
	Urls   []string
	Client *http.Client
	Db     *database.Database
	ticker *time.Ticker
}

func printProtobuf(message proto.Message) {
	options := protojson.MarshalOptions{Multiline: true}
	jsonBytes, _ := options.Marshal(message)
	fmt.Println(string(jsonBytes))
}

func NewGtfsRtWatcher(ctx context.Context, urls []string, domainStringName string, intervalSec float64) (*GtfsRtWatcher, error) {
	db, err := database.NewDatabaseConnection(ctx, domainStringName)
	if err != nil {
		return nil, err
	}

	return &GtfsRtWatcher{
		Urls:   urls,
		Db:     db,
		Client: &http.Client{},
		ticker: time.NewTicker(time.Duration(intervalSec) * time.Second),
	}, nil
}

func (watcher *GtfsRtWatcher) SampleEndpoint(ctx context.Context, url string) (*gtfs.FeedMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := watcher.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feedMessage := &gtfs.FeedMessage{}
	err = proto.Unmarshal(body, feedMessage)
	if err != nil {
		return nil, err
	}

	return feedMessage, nil
}

func (watcher *GtfsRtWatcher) SampleEndpoints(ctx context.Context) error {
	for _, url := range watcher.Urls {
		feedMessage, err := watcher.SampleEndpoint(ctx, url)
		if err != nil {
			return fmt.Errorf("Failed to sample GTFS-RT feed from URL %s: %w", url, err)
		}

		fmt.Printf("Sampled GTFS-RT feed from URL %s: %d entities\n", url, len(feedMessage.Entity))
		printProtobuf(feedMessage.GetEntity()[0])
	}
	return nil
}

func (watcher *GtfsRtWatcher) Watch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Poll error:", ctx.Err())
			return ctx.Err()

		case <-watcher.ticker.C:
			fmt.Println("Polling GTFS-RT feeds...")
			if err := watcher.SampleEndpoints(ctx); err != nil {
				return err
			}
		}
	}
}

func (watcher *GtfsRtWatcher) Close() error {
	watcher.ticker.Stop()
	return watcher.Db.Close()
}
