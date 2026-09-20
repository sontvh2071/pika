package launcher

import "pika/internal/processmonitor"

func (s *Service) ProcessMonitor() processmonitor.Snapshot { return s.processReadings.Read(s.ctx) }
