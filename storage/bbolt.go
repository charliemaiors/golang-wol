package storage

import (
	storage "github.com/coreos/bbolt"
)

type BboltDeviceAccessor struct {
	db *storage.DB
}

type BboltPasswordAccessor struct {
	db *storage.DB
}

type BboltSchedulerAccessor struct {
	db *storage.DB
}

func newBboltDeviceAccessor(db *storage.DB) BboltDeviceAccessor {
	return BboltDeviceAccessor{db: db}
}

func newBboltPasswordAccessor(db *storage.DB) BboltPasswordAccessor {
	return BboltPasswordAccessor{db: db}
}

func newBboltSchedulerAccessor(db *storage.DB) BboltSchedulerAccessor {
	return BboltSchedulerAccessor{db: db}
}
