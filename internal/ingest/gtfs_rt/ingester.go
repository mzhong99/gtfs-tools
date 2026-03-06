package gtfs_rt

import (
	"bytes"
	"context"
	"fmt"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
	database "tarediiran-industries.com/gtfs-services/internal/db"
	"tarediiran-industries.com/gtfs-services/internal/platform"
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

type FeedIngester struct {
	cfg platform.RealTimeConfig

	snapshotId  int64
	lastHashSum []byte
	tuBuf       []TripUpdateRecord
	stuBuf      []StopTimeUpdateRecord
	db          *database.Database
}

type FeedIngesterSet struct {
	cfg platform.SingleConfig

	ingesters []FeedIngester
	db        *database.Database
}

func NewFeedIngesterSet(ctx context.Context, cfg platform.SingleConfig) (*FeedIngesterSet, error) {
	db, err := cfg.NewDatabase(ctx)
	if err != nil {
		return nil, err
	}

	ingesters := make([]FeedIngester, 0)
	for _, rtcfg := range cfg.Feed.RealTime {
		ingester := FeedIngester{
			cfg:    rtcfg,
			db:     db,
			tuBuf:  make([]TripUpdateRecord, 0, 2048),
			stuBuf: make([]StopTimeUpdateRecord, 0, 2048),
		}
		ingesters = append(ingesters, ingester)
	}

	return &FeedIngesterSet{cfg: cfg, ingesters: ingesters, db: db}, nil
}

func (ingester *FeedIngesterSet) Stop() {
	ingester.db.Close()
}

func (ingester *FeedIngester) insertFeedSnapshot(ctx context.Context) error {
	row := ingester.db.QueryRowContext(
		ctx,
		"INSERT INTO feed_snapshots DEFAULT VALUES RETURNING snapshot_id",
	)

	if err := row.Scan(&ingester.snapshotId); err != nil {
		return err
	}
	return nil
}

func (ingester *FeedIngester) bufferTripUpdate(
	ctx context.Context, tripUpdate *gtfs.TripUpdate,
) error {
	trip := tripUpdate.GetTrip()
	if trip == nil {
		return fmt.Errorf("TripUpdate message does not contain a TripDescriptor")
	}

	tuRecord := TripUpdateRecord{
		TripId:      trip.GetTripId(),
		StartDate:   trip.GetStartDate(),
		StartTime:   trip.GetStartTime(),
		DirectionId: trip.GetDirectionId(),
		SnapshotId:  ingester.snapshotId,
	}

	ingester.tuBuf = append(ingester.tuBuf, tuRecord)

	for _, stopTimeUpdate := range tripUpdate.GetStopTimeUpdate() {
		stuRecord := StopTimeUpdateRecord{
			StopId:       *stopTimeUpdate.StopId,
			ArrivalUTC:   0,
			DepartureUTC: 0,
			SnapshotId:   ingester.snapshotId,
		}

		if arrival := stopTimeUpdate.GetArrival(); arrival != nil {
			stuRecord.ArrivalUTC = arrival.GetTime()
		}
		if departure := stopTimeUpdate.GetDeparture(); departure != nil {
			stuRecord.DepartureUTC = departure.GetTime()
		}

		ingester.stuBuf = append(ingester.stuBuf, stuRecord)
	}

	return nil
}

func (ingester *FeedIngester) flushTripUpdates(ctx context.Context) error {
	_, err := ingester.db.CopyFromSlice(
		ctx,
		"trip_update_events",
		TripUpdateColumns(),
		len(ingester.tuBuf),
		func(i int) ([]any, error) { return ingester.tuBuf[i].ToAnyArray(), nil },
	)

	if err != nil {
		return err
	}
	ingester.tuBuf = make([]TripUpdateRecord, 0, 2048)

	_, err = ingester.db.CopyFromSlice(
		ctx,
		"trip_update_stop_time_events",
		StopTimeUpdateColumns(),
		len(ingester.stuBuf),
		func(i int) ([]any, error) { return ingester.stuBuf[i].ToAnyArray(), nil },
	)

	if err != nil {
		return err
	}
	ingester.stuBuf = make([]StopTimeUpdateRecord, 0, 2048)

	return nil
}

func (ingester *FeedIngester) IngestGtfsMessage(ctx context.Context, gtfsMsg *gtfs.FeedMessage) error {
	if err := ingester.insertFeedSnapshot(ctx); err != nil {
		return err
	}

	for _, entity := range gtfsMsg.GetEntity() {
		tripUpdate := entity.GetTripUpdate()
		if tripUpdate == nil {
			continue
		}

		if err := ingester.bufferTripUpdate(ctx, tripUpdate); err != nil {
			return err
		}
	}

	if err := ingester.flushTripUpdates(ctx); err != nil {
		return err
	}

	return nil
}

func (ingester *FeedIngester) Ingest(ctx context.Context, frame platform.FeedFrame) error {
	if bytes.Equal(ingester.lastHashSum, frame.SHA256[:]) {
		return nil
	}

	ingester.lastHashSum = frame.SHA256[:]

	gtfsMsg := &gtfs.FeedMessage{}
	if err := proto.Unmarshal(frame.Body, gtfsMsg); err != nil {
		return err
	}

	return ingester.IngestGtfsMessage(ctx, gtfsMsg)
}
