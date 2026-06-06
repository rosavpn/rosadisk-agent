package event

import "time"

type ActionType string

const (
	ActionDiskList ActionType = "disk:list"
)

type Event struct {
	Action    ActionType
	Data      interface{}
	Timestamp time.Time
	Result    chan Result
}

type Result struct {
	Data  interface{}
	Error error
}
