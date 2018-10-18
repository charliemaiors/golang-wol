package storage

import (
	storage "github.com/coreos/bbolt"
)

const (
	devicesBucket  = "DevBucket"
	passwordBucket = "PassBucket"
	scheduleBucket = "SchedBucket"
	passworkdKey   = "AdminPassword"
	dbName         = "rwol.db"
	defaultDbLoc   = "storage"
)

type BboltDeviceAccessor struct {
	reqdb chan func(*storage.DB)
}

type BboltPasswordAccessor struct {
	reqdb chan func(*storage.DB)
}

type BboltSchedulerAccessor struct {
	reqdb chan func(*storage.DB)
}

func newBboltDeviceAccessor(db chan func(*storage.DB)) BboltDeviceAccessor {
	return BboltDeviceAccessor{reqdb: db}
}

func newBboltPasswordAccessor(db chan func(*storage.DB)) BboltPasswordAccessor {
	return BboltPasswordAccessor{reqdb: db}
}

func newBboltSchedulerAccessor(db chan func(*storage.DB)) BboltSchedulerAccessor {
	return BboltSchedulerAccessor{reqdb: db}
}
