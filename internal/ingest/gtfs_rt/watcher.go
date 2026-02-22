package gtfs_rt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"
	"tarediiran-industries.com/gtfs-services/internal/common"
	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type GtfsRecord interface {
	ColumnNames() []string
	ToAnyArray() []any
}

type TripUpdateRecord struct {
	SnapshotId  int64
	TripId      string
	StartDate   string
	StartTime   string
	DirectionId uint32
}

func TripUpdateColumns() []string {
	return []string{"trip_id", "start_date", "start_time", "direction_id", "snapshot_id"}
}

func (entry *TripUpdateRecord) ToAnyArray() []any {
	return []any{
		entry.TripId,
		entry.StartDate,
		entry.StartTime,
		entry.DirectionId,
		entry.SnapshotId,
	}
}

type StopTimeUpdateRecord struct {
	SnapshotId   int64
	StopId       string
	ArrivalUTC   int64
	DepartureUTC int64
}

func StopTimeUpdateColumns() []string {
	return []string{"stop_id", "arrival_time", "departure_time", "snapshot_id"}
}

func (entry *StopTimeUpdateRecord) ToAnyArray() []any {
	return []any{
		entry.StopId,
		entry.ArrivalUTC,
		entry.DepartureUTC,
		entry.SnapshotId,
	}
}

type GtfsRtWatcher struct {
	Urls   []string
	Client *http.Client
	Db     *database.Database
	ticker *time.Ticker

	snapshotId int64
	tuBuffer   []TripUpdateRecord
	stuBuffer  []StopTimeUpdateRecord

	metrics   *common.Metrics
	telemetry *common.TelemetryServer
}

func NewGtfsRtWatcher(
	ctx context.Context,
	telemetryAddr string,
	urls []string,
	domainStringName string,
	intervalSec float64,
) (*GtfsRtWatcher, error) {

	db, err := database.NewDatabaseConnection(ctx, domainStringName)
	if err != nil {
		return nil, err
	}

	telemetry := common.NewTelemetryServer(telemetryAddr)
	telemetry.Start()
	metrics := common.NewMetrics(telemetry.GetRegistry())

	return &GtfsRtWatcher{
		Urls:      urls,
		Db:        db,
		Client:    &http.Client{},
		ticker:    time.NewTicker(time.Duration(intervalSec) * time.Second),
		stuBuffer: make([]StopTimeUpdateRecord, 0, 2048),
		telemetry: telemetry,
		metrics:   metrics,
	}, nil
}

func (watcher *GtfsRtWatcher) SampleEndpoint(ctx context.Context, url string) (*gtfs.FeedMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	ttfbTimer := prometheus.NewTimer(watcher.metrics.HttpTTFBSeconds.WithLabelValues(url))
	resp, err := watcher.Client.Do(req)
	ttfbTimer.ObserveDuration()

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyReadTimer := prometheus.NewTimer(watcher.metrics.HttpReadBodySeconds.WithLabelValues(url))
	body, err := io.ReadAll(resp.Body)
	bodyReadTimer.ObserveDuration()
	if err != nil {
		return nil, err
	}

	watcher.metrics.HttpBytesTotal.WithLabelValues(url).Add(float64(len(body)))

	feedMessage := &gtfs.FeedMessage{}
	err = proto.Unmarshal(body, feedMessage)
	if err != nil {
		return nil, err
	}

	return feedMessage, nil
}

func (watcher *GtfsRtWatcher) FlushStopTimeUpdates(ctx context.Context) error {
	_, err := watcher.Db.CopyFromSlice(
		ctx,
		"trip_update_stop_time_events",
		StopTimeUpdateColumns(),
		len(watcher.stuBuffer),
		func(i int) ([]any, error) {
			return watcher.stuBuffer[i].ToAnyArray(), nil
		},
	)

	if err != nil {
		return err
	}

	watcher.stuBuffer = make([]StopTimeUpdateRecord, 0, 2048)
	return nil
}

func (watcher *GtfsRtWatcher) FlushTripUpdates(ctx context.Context) error {
	_, err := watcher.Db.CopyFromSlice(
		ctx,
		"trip_update_events",
		TripUpdateColumns(),
		len(watcher.tuBuffer),
		func(i int) ([]any, error) {
			return watcher.tuBuffer[i].ToAnyArray(), nil
		},
	)

	if err != nil {
		return err
	}

	watcher.tuBuffer = make([]TripUpdateRecord, 0, 2048)
	return nil
}

func (watcher *GtfsRtWatcher) InsertFeedSnapshot(ctx context.Context) error {
	row := watcher.Db.QueryRowContext(
		ctx,
		"INSERT INTO feed_snapshots DEFAULT VALUES RETURNING snapshot_id",
	)

	if err := row.Scan(&watcher.snapshotId); err != nil {
		return err
	}
	return nil
}

func (watcher *GtfsRtWatcher) InsertTripUpdateEvent(ctx context.Context, tripUpdate *gtfs.TripUpdate) error {
	trip := tripUpdate.GetTrip()
	if trip == nil {
		return fmt.Errorf("TripUpdate message does not contain a TripDescriptor")
	}

	tuRecord := TripUpdateRecord{
		TripId:      trip.GetTripId(),
		StartDate:   trip.GetStartDate(),
		StartTime:   trip.GetStartTime(),
		DirectionId: trip.GetDirectionId(),
		SnapshotId:  watcher.snapshotId,
	}

	watcher.tuBuffer = append(watcher.tuBuffer, tuRecord)

	for _, stopTimeUpdate := range tripUpdate.GetStopTimeUpdate() {
		stuRecord := StopTimeUpdateRecord{
			StopId:       *stopTimeUpdate.StopId,
			ArrivalUTC:   0,
			DepartureUTC: 0,
			SnapshotId:   watcher.snapshotId,
		}

		if arrival := stopTimeUpdate.GetArrival(); arrival != nil {
			stuRecord.ArrivalUTC = arrival.GetTime()
		}
		if departure := stopTimeUpdate.GetDeparture(); departure != nil {
			stuRecord.DepartureUTC = departure.GetTime()
		}

		watcher.stuBuffer = append(watcher.stuBuffer, stuRecord)
	}

	return nil
}

func (watcher *GtfsRtWatcher) IngestFeedMessage(ctx context.Context, feedMessage *gtfs.FeedMessage) error {
	if err := watcher.InsertFeedSnapshot(ctx); err != nil {
		return err
	}

	for _, entity := range feedMessage.GetEntity() {
		tripUpdate := entity.GetTripUpdate()
		if tripUpdate == nil {
			continue
		}

		if err := watcher.InsertTripUpdateEvent(ctx, tripUpdate); err != nil {
			return err
		}
	}

	if err := watcher.FlushTripUpdates(ctx); err != nil {
		return err
	}
	if err := watcher.FlushStopTimeUpdates(ctx); err != nil {
		return err
	}

	return nil
}

func (watcher *GtfsRtWatcher) SampleEndpoints(ctx context.Context) error {
	benchmarker := common.NewBenchmarker("sample-endpoints")
	defer benchmarker.Close()

	for _, url := range watcher.Urls {
		feedMessage, err := watcher.SampleEndpoint(ctx, url)
		if err != nil {
			watcher.metrics.HttpErrorsTotal.WithLabelValues(url).Add(1)
			return fmt.Errorf("Failed to sample GTFS-RT feed from URL %s: %w", url, err)
		}

		if err := watcher.IngestFeedMessage(ctx, feedMessage); err != nil {
			fmt.Printf("Failed to ingest GTFS-RT feed from URL %s: %v\n", url, err)
		} else {
			fmt.Printf("Sampled GTFS-RT feed from URL %s: %d entities\n", url, len(feedMessage.Entity))
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
	watcher.telemetry.Stop()
	return watcher.Db.Close()
}
