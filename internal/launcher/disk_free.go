package launcher

import "pika/internal/diskfree"

func (s *Service) DiskFree() diskfree.Snapshot { return s.diskReadings.Read(s.ctx) }
