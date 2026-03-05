package gtfs_rt

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"golang.org/x/sync/errgroup"
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

type PollResult struct {
	FeedID     string
	URL        string
	FetchedAt  time.Time
	StatusCode int
	Payload    []byte
}

func (result *PollResult) ToFeedFrame() common.FeedFrame {
	return common.FeedFrame{
		FeedID:     result.FeedID,
		CapturedAt: time.Now(),
		Status:     200,
		Body:       result.Payload,
		SHA256:     sha256.Sum256(result.Payload),
		Source:     "http",
	}
}

type Poller struct {
	Config         common.RealTimeConfig
	LastHash       []byte
	URL            string
	Ticker         *time.Ticker
	PayloadHandler PollCallback
}

type PollCallback func(ctx context.Context, result PollResult) error

type PollerSet struct {
	Config  common.FeedConfig
	Client  *http.Client
	Pollers []Poller
}

func NewPollerSet(ctx context.Context, config common.SingleConfig) (*PollerSet, error) {
	pollerSet := &PollerSet{
		Config:  config.Feed,
		Client:  &http.Client{},
		Pollers: make([]Poller, 0),
	}

	for _, realTimeConfig := range config.Feed.RealTime {
		poller := Poller{
			Config: realTimeConfig,
			URL:    realTimeConfig.URL,
			Ticker: time.NewTicker(time.Duration(realTimeConfig.PollSeconds) * time.Second),
		}
		pollerSet.Pollers = append(pollerSet.Pollers, poller)
	}

	return pollerSet, nil
}

func (pollerSet *PollerSet) SetHandler(handler PollCallback) {
	for _, poller := range pollerSet.Pollers {
		poller.PayloadHandler = handler
	}
}

func (pollerSet *PollerSet) SetHandlerByID(id string, handler PollCallback) {
	for i, _ := range pollerSet.Pollers {
		poller := &pollerSet.Pollers[i]
		if poller.Config.ID == id {
			poller.PayloadHandler = handler
		}
	}
}

func (pollerSet *PollerSet) String() string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "PollerSet with %d endpoints\n", len(pollerSet.Pollers))
	for _, poller := range pollerSet.Pollers {
		fmt.Fprintf(
			&builder, "%s: (%.2fs) - %s\n",
			poller.Config.ID,
			poller.Config.PollSeconds,
			poller.URL,
		)
	}

	return builder.String()
}

func (poller *Poller) SampleEndpoint(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, "GET", poller.URL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	result := PollResult{
		FeedID:     poller.Config.ID,
		URL:        poller.URL,
		FetchedAt:  time.Now(),
		StatusCode: resp.StatusCode,
		Payload:    body,
	}

	return poller.PayloadHandler(ctx, result)
}

func (pollerSet *PollerSet) PollEndpoint(ctx context.Context, poller Poller) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-poller.Ticker.C:
			err := poller.SampleEndpoint(ctx, pollerSet.Client)
			if err != nil {
				log.Printf("poll %s: %v", poller.Config.ID, err)
			}
		}
	}
}

func (pollerSet *PollerSet) Poll(ctx context.Context) error {
	group, subctx := errgroup.WithContext(ctx)
	for _, poller := range pollerSet.Pollers {
		thisPoller := poller // need to capture the loop var, otherwise they all are the same
		group.Go(func() error {
			return pollerSet.PollEndpoint(subctx, thisPoller)
		})
	}
	return group.Wait()
}

func (pollerSet *PollerSet) Stop() {
	for _, poller := range pollerSet.Pollers {
		poller.Ticker.Stop()
	}
}

type FeedIngester struct {
	cfg common.RealTimeConfig

	snapshotId  int64
	lastHashSum []byte
	tuBuf       []TripUpdateRecord
	stuBuf      []StopTimeUpdateRecord
	db          *database.Database
}

type FeedIngesterSet struct {
	cfg common.SingleConfig

	ingesters []FeedIngester
	db        *database.Database
}

func NewFeedIngesterSet(ctx context.Context, cfg common.SingleConfig) (*FeedIngesterSet, error) {
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

func (ingester *FeedIngester) Ingest(ctx context.Context, frame common.FeedFrame) error {
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

type Watcher struct {
	cfg common.SingleConfig

	pollerSet   *PollerSet
	ingesterSet *FeedIngesterSet

	metrics   *common.Metrics
	telemetry *common.TelemetryServer
}

func NewWatcher(ctx context.Context, cfg common.SingleConfig) (*Watcher, error) {
	telemetry := cfg.NewTelemetryServer()
	telemetry.Start()
	metrics := common.NewMetrics(telemetry.GetRegistry())

	pollerSet, err := NewPollerSet(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ingesterSet, err := NewFeedIngesterSet(ctx, cfg)
	if err != nil {
		return nil, err
	}

	for _, ingester := range ingesterSet.ingesters {
		pollerSet.SetHandlerByID(
			ingester.cfg.ID,
			func(ctx context.Context, result PollResult) error {
				ingester := ingester
				return ingester.Ingest(ctx, result.ToFeedFrame())
			},
		)
	}

	return &Watcher{
		cfg:         cfg,
		pollerSet:   pollerSet,
		ingesterSet: ingesterSet,
		metrics:     metrics,
		telemetry:   telemetry,
	}, nil
}

func (watcher *Watcher) Watch(ctx context.Context) error {
	return watcher.pollerSet.Poll(ctx)
}

func (watcher *Watcher) Close() {
	watcher.telemetry.Stop()
	watcher.pollerSet.Stop()
	watcher.ingesterSet.Stop()
}
