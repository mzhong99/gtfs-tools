package gtfs_rt

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"tarediiran-industries.com/gtfs-services/internal/platform"
)

type PollResult struct {
	FeedID     string
	URL        string
	FetchedAt  time.Time
	StatusCode int
	Payload    []byte
}

func (result *PollResult) ToFeedFrame() platform.FeedFrame {
	return platform.FeedFrame{
		FeedID:     result.FeedID,
		CapturedAt: time.Now(),
		Status:     200,
		Body:       result.Payload,
		SHA256:     sha256.Sum256(result.Payload),
		Source:     "http",
	}
}

type Poller struct {
	Config         platform.RealTimeConfig
	LastHash       []byte
	URL            string
	Ticker         *time.Ticker
	PayloadHandler PollCallback
}

type PollCallback func(ctx context.Context, result PollResult) error

type PollerSet struct {
	Config  platform.FeedConfig
	Client  *http.Client
	Pollers []Poller
}

func NewPollerSet(ctx context.Context, config platform.SingleConfig) (*PollerSet, error) {
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
	for i, _ := range pollerSet.Pollers {
		poller := &pollerSet.Pollers[i]
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
