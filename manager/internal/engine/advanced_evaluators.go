package engine

import (
	"time"

	"github.com/google/uuid"
)

type sequenceRuntimeState struct {
	StepIndex     int
	FirstSeen     time.Time
	LastSeen      time.Time
	LastAlertTime time.Time
	LastEventID   uuid.UUID
}

type correlationRuntimeState struct {
	Count         int
	DistinctTypes map[string]struct{}
	FirstSeen     time.Time
	LastSeen      time.Time
	LastAlertTime time.Time
	LastEventID   uuid.UUID
}

type anomalyBucket struct {
	BucketStart time.Time
	Count       int
}

type anomalyRuntimeState struct {
	CurrentBucketStart time.Time
	CurrentCount       int
	History            []anomalyBucket
	LastSeen           time.Time
	LastAlertTime      time.Time
	LastEventID        uuid.UUID
}
