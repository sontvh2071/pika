package launcher

import "pika/internal/gold"

func (s *Service) GoldToday(refresh bool) gold.Snapshot { return s.goldPrices.Read(s.ctx, refresh) }
