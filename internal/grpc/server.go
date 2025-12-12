package grpc

import (
	"context"
	"fmt"
	"math"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
	pb "github.com/Billy-Davies-2/jellycat-draft-ui/proto"
)

// Server implements the gRPC DraftService
type Server struct {
	pb.UnimplementedDraftServiceServer
	dal    dal.DraftDAL
	pubsub *pubsub.PubSub
}

// NewServer creates a new gRPC server
func NewServer(dal dal.DraftDAL, ps *pubsub.PubSub) *Server {
	return &Server{
		dal:    dal,
		pubsub: ps,
	}
}

// GetState returns the current draft state
func (s *Server) GetState(ctx context.Context, req *pb.Empty) (*pb.DraftState, error) {
	logger.Debug("gRPC: Getting draft state")
	state, err := s.dal.GetState()
	if err != nil {
		logger.Error("gRPC: Failed to get draft state", "error", err)
		return nil, err
	}

	return modelsToPbDraftState(state), nil
}

// DraftPlayer drafts a player to a team
func (s *Server) DraftPlayer(ctx context.Context, req *pb.DraftPlayerRequest) (*pb.DraftPlayerResponse, error) {
	logger.Info("gRPC: Drafting player", "player_id", req.PlayerId, "team_id", req.TeamId)
	err := s.dal.DraftPlayer(req.PlayerId, req.TeamId)
	if err != nil {
		logger.Error("gRPC: Failed to draft player", "error", err, "player_id", req.PlayerId, "team_id", req.TeamId)
		return &pb.DraftPlayerResponse{Success: false}, err
	}

	s.pubsub.Publish(pubsub.Event{
		Type: "draft:pick",
		Payload: map[string]interface{}{
			"playerId": req.PlayerId,
			"teamId":   req.TeamId,
		},
	})

	return &pb.DraftPlayerResponse{Success: true}, nil
}

// ResetDraft resets the draft
func (s *Server) ResetDraft(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	logger.Info("gRPC: Resetting draft")
	err := s.dal.Reset()
	if err != nil {
		logger.Error("gRPC: Failed to reset draft", "error", err)
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{Type: "draft:reset"})
	return &pb.Empty{}, nil
}

// AddTeam adds a new team
func (s *Server) AddTeam(ctx context.Context, req *pb.AddTeamRequest) (*pb.Team, error) {
	team, err := s.dal.AddTeam(req.Name, req.Owner, req.Mascot, req.Color)
	if err != nil {
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{
		Type: "teams:add",
		Payload: map[string]interface{}{
			"id": team.ID,
		},
	})

	return modelsToPbTeam(team), nil
}

// ListTeams lists all teams
func (s *Server) ListTeams(ctx context.Context, req *pb.Empty) (*pb.TeamsResponse, error) {
	state, err := s.dal.GetState()
	if err != nil {
		return nil, err
	}

	teams := make([]*pb.Team, len(state.Teams))
	for i, team := range state.Teams {
		teams[i] = modelsToPbTeam(&team)
	}

	return &pb.TeamsResponse{Teams: teams}, nil
}

// ReorderTeams reorders the teams
func (s *Server) ReorderTeams(ctx context.Context, req *pb.ReorderTeamsRequest) (*pb.TeamsResponse, error) {
	teams, err := s.dal.ReorderTeams(req.Order)
	if err != nil {
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{Type: "teams:reorder"})

	pbTeams := make([]*pb.Team, len(teams))
	for i, team := range teams {
		pbTeams[i] = modelsToPbTeam(&team)
	}

	return &pb.TeamsResponse{Teams: pbTeams}, nil
}

// AddPlayer adds a new player
func (s *Server) AddPlayer(ctx context.Context, req *pb.Player) (*pb.Player, error) {
	player := pbToModelsPlayer(req)
	result, err := s.dal.AddPlayer(player)
	if err != nil {
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{
		Type: "players:add",
		Payload: map[string]interface{}{
			"id": result.ID,
		},
	})

	return modelsToPbPlayer(result), nil
}

// SetPlayerPoints updates player points
func (s *Server) SetPlayerPoints(ctx context.Context, req *pb.SetPlayerPointsRequest) (*pb.Player, error) {
	player, err := s.dal.SetPlayerPoints(req.Id, int(req.Points))
	if err != nil {
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{
		Type: "players:updatePoints",
		Payload: map[string]interface{}{
			"id":     player.ID,
			"points": player.Points,
		},
	})

	return modelsToPbPlayer(player), nil
}

// GetPlayerProfile returns extended player information
func (s *Server) GetPlayerProfile(ctx context.Context, req *pb.GetPlayerProfileRequest) (*pb.PlayerProfile, error) {
	state, err := s.dal.GetState()
	if err != nil {
		return nil, err
	}

	var player *models.Player
	for _, p := range state.Players {
		if p.ID == req.Id {
			player = &p
			break
		}
	}

	if player == nil {
		return nil, nil
	}

	// Generate mock metrics
	seed := player.Points
	for _, c := range player.ID {
		seed += int(c)
	}

	norm := func(x int) int32 {
		return int32(math.Max(0, math.Min(100, float64(x))))
	}

	profile := &pb.PlayerProfile{
		Id:        player.ID,
		Name:      player.Name,
		Position:  player.Position,
		Team:      player.Team,
		Points:    int32(player.Points),
		Tier:      string(player.Tier),
		Drafted:   player.Drafted,
		DraftedBy: player.DraftedBy,
		Image:     player.Image,
		Metrics: &pb.PlayerMetrics{
			Consistency: norm((seed * 13) % 101),
			Popularity:  norm((seed * 29) % 101),
			Efficiency:  norm((seed * 47) % 101),
			TrendDelta:  float64(((seed%15)-7)/7.0) * 100 / 100,
		},
	}

	return profile, nil
}

// ListChat returns all chat messages
func (s *Server) ListChat(ctx context.Context, req *pb.Empty) (*pb.ChatResponse, error) {
	state, err := s.dal.GetState()
	if err != nil {
		return nil, err
	}

	messages := make([]*pb.ChatMessage, len(state.Chat))
	for i, msg := range state.Chat {
		messages[i] = modelsToPbChatMessage(&msg)
	}

	return &pb.ChatResponse{Messages: messages}, nil
}

// SendChatMessage sends a new chat message
func (s *Server) SendChatMessage(ctx context.Context, req *pb.SendChatRequest) (*pb.ChatMessage, error) {
	msgType := req.Type
	if msgType == "" {
		msgType = "user"
	}

	msg, err := s.dal.AddChatMessage(req.Text, msgType)
	if err != nil {
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{
		Type: "chat:add",
		Payload: map[string]interface{}{
			"id": msg.ID,
		},
	})

	return modelsToPbChatMessage(msg), nil
}

// AddReaction adds a reaction to a chat message
func (s *Server) AddReaction(ctx context.Context, req *pb.AddReactionRequest) (*pb.ChatMessage, error) {
	msg, err := s.dal.AddReaction(req.MessageId, req.Emote, req.User)
	if err != nil {
		return nil, err
	}

	s.pubsub.Publish(pubsub.Event{
		Type: "chat:react",
		Payload: map[string]interface{}{
			"id":    msg.ID,
			"emote": req.Emote,
		},
	})

	return modelsToPbChatMessage(msg), nil
}

// StreamEvents streams events to clients
func (s *Server) StreamEvents(req *pb.Empty, stream pb.DraftService_StreamEventsServer) error {
	logger.Debug("gRPC: New client connected to event stream")
	eventChan := s.pubsub.Subscribe()
	defer s.pubsub.Unsubscribe(eventChan)

	for {
		select {
		case event := <-eventChan:
			payload := make(map[string]string)
			for k, v := range event.Payload {
				payload[k] = fmt.Sprint(v)
			}
			pbEvent := &pb.Event{
				Type:    event.Type,
				Payload: payload,
			}
			if err := stream.Send(pbEvent); err != nil {
				logger.Error("gRPC: Failed to send event to stream", "error", err)
				return err
			}
		case <-stream.Context().Done():
			logger.Debug("gRPC: Client disconnected from event stream")
			return nil
		}
	}
}

// Helper conversion functions
func modelsToPbPlayer(p *models.Player) *pb.Player {
	return &pb.Player{
		Id:        p.ID,
		Name:      p.Name,
		Position:  p.Position,
		Team:      p.Team,
		Points:    int32(p.Points),
		Tier:      string(p.Tier),
		Drafted:   p.Drafted,
		DraftedBy: p.DraftedBy,
		Image:     p.Image,
	}
}

func pbToModelsPlayer(p *pb.Player) *models.Player {
	return &models.Player{
		ID:        p.Id,
		Name:      p.Name,
		Position:  p.Position,
		Team:      p.Team,
		Points:    int(p.Points),
		Tier:      models.Tier(p.Tier),
		Drafted:   p.Drafted,
		DraftedBy: p.DraftedBy,
		Image:     p.Image,
	}
}

func modelsToPbTeam(t *models.Team) *pb.Team {
	players := make([]*pb.Player, len(t.Players))
	for i, p := range t.Players {
		players[i] = modelsToPbPlayer(&p)
	}

	return &pb.Team{
		Id:      t.ID,
		Name:    t.Name,
		Owner:   t.Owner,
		Mascot:  t.Mascot,
		Color:   t.Color,
		Players: players,
	}
}

func modelsToPbChatMessage(m *models.ChatMessage) *pb.ChatMessage {
	emotes := make(map[string]int32)
	for k, v := range m.Emotes {
		emotes[k] = int32(v)
	}

	return &pb.ChatMessage{
		Id:     m.ID,
		Ts:     m.TS,
		Type:   m.Type,
		Text:   m.Text,
		Emotes: emotes,
	}
}

func modelsToPbDraftState(s *models.DraftState) *pb.DraftState {
	players := make([]*pb.Player, len(s.Players))
	for i, p := range s.Players {
		players[i] = modelsToPbPlayer(&p)
	}

	teams := make([]*pb.Team, len(s.Teams))
	for i, t := range s.Teams {
		teams[i] = modelsToPbTeam(&t)
	}

	chat := make([]*pb.ChatMessage, len(s.Chat))
	for i, m := range s.Chat {
		chat[i] = modelsToPbChatMessage(&m)
	}

	return &pb.DraftState{
		Players: players,
		Teams:   teams,
		Chat:    chat,
	}
}
