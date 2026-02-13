package spinner

import (
	"context"

	"github.com/yarlson/pin"
)

type Spinner interface {
	Start()
	Stop()
	UpdateMessage(message string)
}

type spinner struct {
	p *pin.Pin
}

func NewSpinner() *spinner {
	return &spinner{}
}

func (s *spinner) Start(message string) {
	s.p = pin.New(message)
	s.p.Start(context.Background())
}

func (s *spinner) Stop(message string) {
	if s.p != nil && s.p.IsRunning() {
		s.p.Stop(message)
		s.p = &pin.Pin{}
	}
}

func (s *spinner) UpdateMessage(message string) {
	if s.p.IsRunning() {
		s.p.UpdateMessage(message)
	}
}
