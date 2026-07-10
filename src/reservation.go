package main

type ReservationKind string

const (
	ReservationForecast ReservationKind = "forecast"
	ReservationManual   ReservationKind = "manual"
	ReservationRefresh  ReservationKind = "refresh"
)

type ReservationStatus string

const (
	ReservationQueued    ReservationStatus = "queued"
	ReservationBound     ReservationStatus = "bound"
	ReservationSettling  ReservationStatus = "settling"
	ReservationCompleted ReservationStatus = "completed"
	ReservationFailed    ReservationStatus = "failed"
	ReservationCancelled ReservationStatus = "cancelled"
)

type Reservation struct {
	ID            ReservationID     `json:"id"`
	Cycle         CycleID           `json:"cycle"`
	Forecast      ForecastID        `json:"forecast"`
	Route         RouteID           `json:"route"`
	SourceVault   VaultID           `json:"source_vault"`
	TargetVault   VaultID           `json:"target_vault"`
	Asset         AssetID           `json:"asset"`
	Owner         AccountID         `json:"owner"`
	Kind          ReservationKind   `json:"kind"`
	Amount        Amount            `json:"amount"`
	Fee           Amount            `json:"fee"`
	SettlementDue int               `json:"settlement_due"`
	CreatedClock  int               `json:"created_clock"`
	BoundClock    int               `json:"bound_clock,omitempty"`
	ClosedClock   int               `json:"closed_clock,omitempty"`
	Status        ReservationStatus `json:"status"`
	Notes         []string          `json:"notes"`
}

func NewReservation(id ReservationID, cycle CycleID, forecast *ForecastEntry, route *Route, owner AccountID, kind ReservationKind, amount Amount, fee Amount, clock int) *Reservation {
	return &Reservation{
		ID:            id,
		Cycle:         cycle,
		Forecast:      forecast.ID,
		Route:         route.ID,
		SourceVault:   route.SourceVault,
		TargetVault:   route.TargetVault,
		Asset:         route.Asset,
		Owner:         owner,
		Kind:          kind,
		Amount:        amount,
		Fee:           fee,
		SettlementDue: clock + route.Policy.SettlementDelay,
		CreatedClock:  clock,
		Status:        ReservationQueued,
		Notes:         []string{},
	}
}

func (r *Reservation) IsOpen() bool {
	return r.Status == ReservationQueued || r.Status == ReservationBound || r.Status == ReservationSettling
}

func (r *Reservation) Ready(clock int) bool {
	return r.IsOpen() && clock >= r.SettlementDue
}

func (r *Reservation) Bind(clock int) {
	if r.Status == ReservationQueued {
		r.Status = ReservationBound
		r.BoundClock = clock
	}
}

func (r *Reservation) MarkSettling(clock int) {
	if r.Status == ReservationQueued {
		r.Bind(clock)
	}
	if r.Status == ReservationBound {
		r.Status = ReservationSettling
	}
}

func (r *Reservation) Complete(clock int) {
	r.Status = ReservationCompleted
	r.ClosedClock = clock
}

func (r *Reservation) Fail(clock int, note string) {
	r.Status = ReservationFailed
	r.ClosedClock = clock
	if note != "" {
		r.Notes = append(r.Notes, note)
	}
}

func (r *Reservation) Cancel(clock int, note string) {
	r.Status = ReservationCancelled
	r.ClosedClock = clock
	if note != "" {
		r.Notes = append(r.Notes, note)
	}
}

type ReservationRequest struct {
	Route  RouteID         `json:"route"`
	Owner  AccountID       `json:"owner"`
	Amount Amount          `json:"amount"`
	Kind   ReservationKind `json:"kind"`
}

type ReservationBatch struct {
	Cycle     CycleID              `json:"cycle"`
	Requests  []ReservationRequest `json:"requests"`
	Accepted  []ReservationID      `json:"accepted"`
	Rejected  []RouteID            `json:"rejected"`
	Total     AmountBuckets        `json:"total"`
	CreatedAt int                  `json:"created_at"`
}

func NewReservationBatch(cycle CycleID, clock int) *ReservationBatch {
	return &ReservationBatch{
		Cycle:     cycle,
		Requests:  []ReservationRequest{},
		Accepted:  []ReservationID{},
		Rejected:  []RouteID{},
		Total:     AmountBuckets{},
		CreatedAt: clock,
	}
}

func (b *ReservationBatch) AddRequest(request ReservationRequest) {
	b.Requests = append(b.Requests, request)
}

func (b *ReservationBatch) AddAccepted(id ReservationID, asset AssetID, amount Amount) {
	b.Accepted = append(b.Accepted, id)
	b.Total.Add(asset, amount)
}

func (b *ReservationBatch) AddRejected(route RouteID) {
	b.Rejected = append(b.Rejected, route)
}
