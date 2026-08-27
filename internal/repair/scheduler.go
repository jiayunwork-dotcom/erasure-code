package repair

import (
	"sort"
)

type Job struct {
	StripeID string
	Plan     *RepairPlan
}

type Scheduler struct {
	jobs     []*Job
	maxQueue int
}

func NewScheduler(maxQueue int) *Scheduler {
	return &Scheduler{
		maxQueue: maxQueue,
	}
}

func (s *Scheduler) Submit(job *Job) bool {
	if job == nil || job.Plan == nil {
		return false
	}
	if s.maxQueue > 0 && len(s.jobs) >= s.maxQueue {
		minIdx := 0
		for i := 1; i < len(s.jobs); i++ {
			if s.jobs[i].Plan.Priority < s.jobs[minIdx].Plan.Priority {
				minIdx = i
			}
		}
		if job.Plan.Priority <= s.jobs[minIdx].Plan.Priority {
			return false
		}
		s.jobs[minIdx] = job
		s.sortJobs()
		return true
	}
	s.jobs = append(s.jobs, job)
	s.sortJobs()
	return true
}

func (s *Scheduler) Next() *Job {
	if len(s.jobs) == 0 {
		return nil
	}
	job := s.jobs[0]
	s.jobs = s.jobs[1:]
	return job
}

func (s *Scheduler) Peek() *Job {
	if len(s.jobs) == 0 {
		return nil
	}
	return s.jobs[0]
}

func (s *Scheduler) Len() int { return len(s.jobs) }

func (s *Scheduler) Clear() { s.jobs = nil }

func (s *Scheduler) Drain() []*Job {
	out := s.jobs
	s.jobs = nil
	return out
}

func (s *Scheduler) sortJobs() {
	sort.Slice(s.jobs, func(i, j int) bool {
		return s.jobs[i].Plan.Priority > s.jobs[j].Plan.Priority
	})
}

func BatchPlan(stripes map[string]*Stripe) []*Job {
	var jobs []*Job
	for id, st := range stripes {
		plan, err := Plan(st)
		if err != nil {
			continue
		}
		jobs = append(jobs, &Job{StripeID: id, Plan: plan})
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Plan.Priority > jobs[j].Plan.Priority
	})
	return jobs
}

func (s *Scheduler) EstimateBandwidth() int {
	total := 0
	for _, j := range s.jobs {
		total += len(j.Plan.ReadFrom)
	}
	return total
}

func (s *Scheduler) FilterByPriority(minPriority int) []*Job {
	var out []*Job
	for _, j := range s.jobs {
		if j.Plan.Priority >= minPriority {
			out = append(out, j)
		}
	}
	return out
}
