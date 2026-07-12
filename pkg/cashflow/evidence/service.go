package evidence

import (
	"context"
	"errors"
)

type SnapshotReader interface {
	LoadCashflowEvidence(ctx context.Context, window TimeWindow) (CommandCenterInput, error)
}

type Service struct {
	reader SnapshotReader
}

func NewService(reader SnapshotReader) *Service {
	return &Service{reader: reader}
}

func (s *Service) BuildCommandCenter(ctx context.Context, window TimeWindow) (CommandCenter, error) {
	if s == nil || s.reader == nil {
		return CommandCenter{}, errors.New("cashflow evidence snapshot reader is required")
	}
	input, err := s.reader.LoadCashflowEvidence(ctx, window)
	if err != nil {
		return CommandCenter{}, err
	}
	if isEmptyWindow(input.Window) && !isEmptyWindow(window) {
		input.Window = window
	}
	return BuildCommandCenter(input), nil
}

func isEmptyWindow(window TimeWindow) bool {
	return window.Start.IsZero() && window.End.IsZero() && window.Label == ""
}
