package main

type Event struct {
	Seq    int                    `json:"seq"`
	Clock  int                    `json:"clock"`
	Kind   string                 `json:"kind"`
	Ref    string                 `json:"ref,omitempty"`
	Asset  AssetID                `json:"asset,omitempty"`
	Amount Amount                 `json:"amount,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

type EventLog struct {
	events []Event
}

func NewEventLog() *EventLog {
	return &EventLog{events: []Event{}}
}

func (l *EventLog) Add(clock int, kind string, ref string, asset AssetID, amount Amount, data map[string]interface{}) Event {
	event := Event{
		Seq:    len(l.events) + 1,
		Clock:  clock,
		Kind:   kind,
		Ref:    ref,
		Asset:  asset,
		Amount: amount,
		Data:   data,
	}
	l.events = append(l.events, event)
	return event
}

func (l *EventLog) All() []Event {
	out := make([]Event, 0, len(l.events))
	out = append(out, l.events...)
	return out
}

func (l *EventLog) Count(kind string) int {
	count := 0
	for _, event := range l.events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}

func (l *EventLog) Last(kind string) (Event, bool) {
	for i := len(l.events) - 1; i >= 0; i-- {
		if l.events[i].Kind == kind {
			return l.events[i], true
		}
	}
	return Event{}, false
}
