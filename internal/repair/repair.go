package repair

import (
	"errors"
	"sort"
)

var ErrInvalidLayout = errors.New("repair: invalid layout")

var ErrUnrecoverable = errors.New("repair: unrecoverable stripe")

var ErrNoRepairNeeded = errors.New("repair: no repair needed")

type ShardStatus int

const (
	Present ShardStatus = iota
	Missing
	Degraded
)

func (s ShardStatus) String() string {
	switch s {
	case Present:
		return "present"
	case Missing:
		return "missing"
	case Degraded:
		return "degraded"
	default:
		return "unknown"
	}
}

type Stripe struct {
	DataShards   int
	ParityShards int
	Status       []ShardStatus
}

func (s *Stripe) Total() int { return s.DataShards + s.ParityShards }

func (s *Stripe) Validate() error {
	if s.DataShards <= 0 || s.ParityShards <= 0 {
		return ErrInvalidLayout
	}
	if len(s.Status) != s.Total() {
		return ErrInvalidLayout
	}
	return nil
}

func (s *Stripe) AvailableCount() int {
	count := 0
	for _, st := range s.Status {
		if st == Present || st == Degraded {
			count++
		}
	}
	return count
}

func (s *Stripe) MissingIndices() []int {
	var idxs []int
	for i, st := range s.Status {
		if st == Missing {
			idxs = append(idxs, i)
		}
	}
	sort.Ints(idxs)
	return idxs
}

func (s *Stripe) DegradedIndices() []int {
	var idxs []int
	for i, st := range s.Status {
		if st == Degraded {
			idxs = append(idxs, i)
		}
	}
	sort.Ints(idxs)
	return idxs
}

func (s *Stripe) IsHealthy() bool {
	for _, st := range s.Status {
		if st != Present {
			return false
		}
	}
	return true
}

type RepairPlan struct {
	ReadFrom []int
	Rebuild  []int
	Priority int
}

func Plan(stripe *Stripe) (*RepairPlan, error) {
	if err := stripe.Validate(); err != nil {
		return nil, err
	}
	if stripe.IsHealthy() {
		return nil, ErrNoRepairNeeded
	}

	available := stripe.AvailableCount()
	if available < stripe.DataShards {
		return nil, ErrUnrecoverable
	}

	missing := stripe.MissingIndices()
	degraded := stripe.DegradedIndices()

	readFrom := make([]int, 0, stripe.DataShards)
	for i, st := range stripe.Status {
		if st == Present && len(readFrom) < stripe.DataShards {
			readFrom = append(readFrom, i)
		}
	}
	for i, st := range stripe.Status {
		if st == Degraded && len(readFrom) < stripe.DataShards {
			readFrom = append(readFrom, i)
		}
	}
	sort.Ints(readFrom)

	rebuild := make([]int, 0, len(missing)+len(degraded))
	rebuild = append(rebuild, missing...)
	rebuild = append(rebuild, degraded...)
	sort.Ints(rebuild)

	priority := len(missing) + len(degraded)
	if priority > stripe.ParityShards {
		priority = stripe.ParityShards
	}

	return &RepairPlan{
		ReadFrom: readFrom,
		Rebuild:  rebuild,
		Priority: priority,
	}, nil
}
