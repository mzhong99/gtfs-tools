package gtfs_rt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (watcher *GtfsRtWatcher) InsertStopTimeUpdate(ctx context.Context, eventId int64, updates []*gtfs.TripUpdate_StopTimeUpdate) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	baseQuery := `
		INSERT INTO trip_update_stop_time_events (
			trip_update_event_id, stop_id, arrival_time, departure_time
		)
		VALUES`

	var builder strings.Builder
	builder.WriteString(baseQuery)

	args := make([]any, 0, 4*len(updates))
	sep := ""
	for i, update := range updates {
		builder.WriteString(sep)
		pb := 4 * i
		fmt.Fprintf(&builder, "($%d,$%d,$%d,$%d)", pb+1, pb+2, pb+3, pb+4)

		args = append(args, eventId, update.StopId, update.Arrival, update.Departure)
		sep = ","
	}

	_, err := watcher.Db.ExecContext(ctx, builder.String(), args...)
	return len(updates), err
}

func (watcher *GtfsRtWatcher) InsertTripUpdateEvent(ctx context.Context, tripUpdate *gtfs.TripUpdate) (int, error) {
	trip := tripUpdate.GetTrip()
	if trip == nil {
		return 0, fmt.Errorf("TripUpdate message does not contain a TripDescriptor")
	}

	row := watcher.Db.QueryRowContext(
		ctx,
		`INSERT INTO trip_update_events (
			trip_id,
			start_date,
			start_time,
			direction_id
		)
		VALUES ($1, $2, $3, $4)
		RETURNING trip_update_event_id`,
		trip.GetTripId(), trip.GetStartDate(), trip.GetStartTime(), trip.GetDirectionId(),
	)

	var tripUpdateEventId int64
	if err := row.Scan(&tripUpdateEventId); err != nil {
		return 0, err
	}

	// for _, stopTimeUpdate := range tripUpdate.GetStopTimeUpdate() {
	// 	_, err := watcher.Db.ExecContext(
	// 		ctx,
	// 		`INSERT INTO trip_update_stop_time_events (
	// 			trip_update_event_id,
	// 			stop_id,
	// 			arrival_time,
	// 			departure_time
	// 		)
	// 		VALUES ($1, $2, $3, $4)`,
	// 		tripUpdateEventId,
	// 		stopTimeUpdate.StopId,
	// 		stopTimeUpdate.Arrival,
	// 		stopTimeUpdate.Departure,
	// 	)

	// 	if err != nil {
	// 		return nil
	// 	}
	// }

	batchInsert := func() (int, error) {
		return watcher.InsertStopTimeUpdate(ctx, tripUpdateEventId, tripUpdate.GetStopTimeUpdate())
	}

	return batchInsert()
}

func (watcher *GtfsRtWatcher) IngestFeedMessage(ctx context.Context, feedMessage *gtfs.FeedMessage) (int, error) {
	totalStopUpdates := 0
	for _, entity := range feedMessage.GetEntity() {
		tripUpdate := entity.GetTripUpdate()
		if tripUpdate == nil {
			continue
		}

		if stopUpdates, err := watcher.InsertTripUpdateEvent(ctx, tripUpdate); err != nil {
			return totalStopUpdates, err
		} else {
			totalStopUpdates += stopUpdates
		}
	}

	return totalStopUpdates, nil
}

func (watcher *GtfsRtWatcher) SampleEndpoints(ctx context.Context) error {
	for _, url := range watcher.Urls {
		feedMessage, err := watcher.SampleEndpoint(ctx, url)
		if err != nil {
			return fmt.Errorf("Failed to sample GTFS-RT feed from URL %s: %w", url, err)
		}

		if updates, err := watcher.IngestFeedMessage(ctx, feedMessage); err != nil {
			fmt.Printf("Failed to ingest GTFS-RT feed from URL %s: %v\n", url, err)
		} else {
			fmt.Printf("Sampled GTFS-RT feed from URL %s: %d entities, %d updates\n", url, len(feedMessage.Entity), updates)
		}
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
