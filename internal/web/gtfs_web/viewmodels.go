package gtfs_web

type TrainsPageVM struct {
	Routes        []string
	SelectedRoute string
	PollSeconds   int
}

type TrainsTableVM struct {
	SelectedRoute string
	UpdatedAt     string
	Rows          []TrainRowVM
}

type TrainRowVM struct {
	Route     string
	Direction string
	TripID    string
	LastSeen  string
	Status    string
}
