package launcher

import (
	"pika/internal/sensors"
)

// Sensor monitoring is a built-in, read-only detail view, never a shell action.
func (s *Service) Sensors() sensors.Snapshot { return s.sensorReadings.Read(s.ctx) }
