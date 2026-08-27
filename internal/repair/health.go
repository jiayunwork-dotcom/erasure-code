package repair

type HealthReport struct {
	Total         int
	Healthy       int
	Degraded      int
	AtRisk        int
	Lost          int
	AvgRedundancy float64
}

func Assess(stripes []*Stripe) *HealthReport {
	report := &HealthReport{Total: len(stripes)}
	if len(stripes) == 0 {
		return report
	}
	totalRedundancy := 0.0
	for _, s := range stripes {
		if s.Validate() != nil {
			continue
		}
		if s.IsHealthy() {
			report.Healthy++
			totalRedundancy += float64(s.ParityShards)
			continue
		}
		avail := s.AvailableCount()
		if avail < s.DataShards {
			report.Lost++
			totalRedundancy += 0
			continue
		}
		missing := len(s.MissingIndices())
		degraded := len(s.DegradedIndices())
		remaining := avail - s.DataShards
		totalRedundancy += float64(remaining)
		if missing > 0 {
			report.AtRisk++
		} else if degraded > 0 {
			report.Degraded++
		}
	}
	if report.Total > 0 {
		report.AvgRedundancy = totalRedundancy / float64(report.Total)
	}
	return report
}

func (r *HealthReport) NeedsUrgentRepair() bool {
	if r.Lost > 0 {
		return true
	}
	return r.AvgRedundancy < 1.0
}

func (r *HealthReport) HealthScore() int {
	if r.Total == 0 {
		return 100
	}
	score := float64(r.Healthy)*100 + float64(r.Degraded)*80 +
		float64(r.AtRisk)*50 + float64(r.Lost)*0
	avg := score / float64(r.Total)
	if avg > 100 {
		avg = 100
	}
	if avg < 0 {
		avg = 0
	}
	return int(avg)
}

func ClassifyStripe(s *Stripe) string {
	if err := s.Validate(); err != nil {
		return "invalid"
	}
	if s.IsHealthy() {
		return "healthy"
	}
	avail := s.AvailableCount()
	if avail < s.DataShards {
		return "lost"
	}
	missing := len(s.MissingIndices())
	if missing > 0 {
		return "at-risk"
	}
	return "degraded"
}
