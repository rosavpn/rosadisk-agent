package event

import "time"

type ActionType string

const (
	ActionDiskList         ActionType = "disk:list"
	ActionFilesystemList   ActionType = "filesystem:list"
	ActionFilesystemCreate ActionType = "filesystem:create"
	ActionMountList        ActionType = "mount:list"
	ActionMountCreate      ActionType = "mount:create"
	ActionSubvolumeList    ActionType = "subvolume:list"
	ActionSubvolumeCreate  ActionType = "subvolume:create"
	ActionSubvolumeGet     ActionType = "subvolume:get"
	ActionSubvolumeDelete  ActionType = "subvolume:delete"
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
