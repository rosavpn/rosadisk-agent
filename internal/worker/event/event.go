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
	ActionBackup              ActionType = "background:backup"
	ActionSnapshotCheck      ActionType = "background:snapshot:check"
	ActionDefrag              ActionType = "background:defrag"
	ActionScrubCheck          ActionType = "background:scrub:check"
	ActionBalanceCheck        ActionType = "background:balance:check"
	ActionScrubDisk           ActionType = "background:scrub:disk"
	ActionBalanceDisk         ActionType = "background:balance:disk"
	ActionSnapshotSubvolume   ActionType = "background:snapshot:subvolume"
	ActionSnapshotCleanup     ActionType = "background:snapshot:cleanup"
	ActionUploadBackup        ActionType = "upload:backup"
	ActionUploadSnapshot      ActionType = "upload:snapshot"
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
