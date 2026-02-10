package model

import "time"

type Job struct {
	ID        string
	CreatedAt time.Time
	Priority  int
	Payload   any

	Attempts    int
	MaxRetries int
}
