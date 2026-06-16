package server

import "context"

func (s *Server) handleBackup(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling backup event")
	return map[string]string{"status": "backup completed (dummy)"}, nil
}

func (s *Server) handleSnapshot(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling snapshot event")
	return map[string]string{"status": "snapshot completed (dummy)"}, nil
}

func (s *Server) handleDefrag(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling defrag event")
	return map[string]string{"status": "defrag completed (dummy)"}, nil
}

func (s *Server) handleScrub(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling scrub event")
	return map[string]string{"status": "scrub completed (dummy)"}, nil
}

func (s *Server) handleBalance(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling balance event")
	return map[string]string{"status": "balance completed (dummy)"}, nil
}
