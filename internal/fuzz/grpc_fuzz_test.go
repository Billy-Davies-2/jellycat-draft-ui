package fuzz

import (
	"context"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	grpcserver "github.com/Billy-Davies-2/jellycat-draft-ui/internal/grpc"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
	pb "github.com/Billy-Davies-2/jellycat-draft-ui/proto"
)

// FuzzGRPCDraftPlayer fuzzes the gRPC DraftPlayer endpoint
func FuzzGRPCDraftPlayer(f *testing.F) {
	// Seed corpus
	f.Add("1", "1")
	f.Add("invalid", "999")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, playerId, teamId string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		req := &pb.DraftPlayerRequest{
			PlayerId: playerId,
			TeamId:   teamId,
		}

		// Should not panic
		_, _ = server.DraftPlayer(context.Background(), req)
	})
}

// FuzzGRPCAddTeam fuzzes the gRPC AddTeam endpoint
func FuzzGRPCAddTeam(f *testing.F) {
	// Seed corpus
	f.Add("Team Name", "Owner", "🦊", "bg-red-100")
	f.Add("", "", "", "")
	f.Add("A", "B", "C", "D")

	f.Fuzz(func(t *testing.T, name, owner, mascot, color string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		req := &pb.AddTeamRequest{
			Name:   name,
			Owner:  owner,
			Mascot: mascot,
			Color:  color,
		}

		_, _ = server.AddTeam(context.Background(), req)
	})
}

// FuzzGRPCSendChatMessage fuzzes the gRPC SendChatMessage endpoint
func FuzzGRPCSendChatMessage(f *testing.F) {
	// Seed corpus
	f.Add("Hello", "user")
	f.Add("", "system")
	f.Add(string(make([]byte, 10000)), "user")

	f.Fuzz(func(t *testing.T, text, msgType string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		req := &pb.SendChatRequest{
			Text: text,
			Type: msgType,
		}

		_, _ = server.SendChatMessage(context.Background(), req)
	})
}

// FuzzGRPCSetPlayerPoints fuzzes the gRPC SetPlayerPoints endpoint
func FuzzGRPCSetPlayerPoints(f *testing.F) {
	// Seed corpus
	f.Add("1", int32(100))
	f.Add("invalid", int32(-999))
	f.Add("", int32(0))

	f.Fuzz(func(t *testing.T, id string, points int32) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		req := &pb.SetPlayerPointsRequest{
			Id:     id,
			Points: points,
		}

		_, _ = server.SetPlayerPoints(context.Background(), req)
	})
}

// FuzzGRPCAddReaction fuzzes the gRPC AddReaction endpoint
func FuzzGRPCAddReaction(f *testing.F) {
	// Seed corpus
	f.Add("msg_1", "🎉", "user1")
	f.Add("", "", "")

	f.Fuzz(func(t *testing.T, messageId, emote, user string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		// Add a test message first
		dal.AddChatMessage("Test", "user")

		req := &pb.AddReactionRequest{
			MessageId: messageId,
			Emote:     emote,
			User:      user,
		}

		_, _ = server.AddReaction(context.Background(), req)
	})
}

// FuzzGRPCAddPlayer fuzzes the gRPC AddPlayer endpoint
func FuzzGRPCAddPlayer(f *testing.F) {
	// Seed corpus
	f.Add("Test Player", "CC", "Woodland", int32(100), "S")
	f.Add("", "", "", int32(0), "")

	f.Fuzz(func(t *testing.T, name, position, team string, points int32, tier string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		req := &pb.Player{
			Name:     name,
			Position: position,
			Team:     team,
			Points:   points,
			Tier:     tier,
			Image:    "/test.png",
		}

		_, _ = server.AddPlayer(context.Background(), req)
	})
}

// FuzzGRPCGetPlayerProfile fuzzes the gRPC GetPlayerProfile endpoint
func FuzzGRPCGetPlayerProfile(f *testing.F) {
	// Seed corpus
	f.Add("1")
	f.Add("invalid")
	f.Add("")
	f.Add("999999")

	f.Fuzz(func(t *testing.T, id string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		req := &pb.GetPlayerProfileRequest{
			Id: id,
		}

		_, _ = server.GetPlayerProfile(context.Background(), req)
	})
}

// FuzzGRPCReorderTeams fuzzes the gRPC ReorderTeams endpoint
func FuzzGRPCReorderTeams(f *testing.F) {
	// Seed corpus with various team order combinations
	f.Add([]byte{1, 2, 3}) // Will be converted to team IDs
	f.Add([]byte{})
	f.Add([]byte{99, 100, 101})

	f.Fuzz(func(t *testing.T, orderBytes []byte) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		server := grpcserver.NewServer(dal, ps)

		// Convert bytes to string IDs
		order := make([]string, len(orderBytes))
		for i, b := range orderBytes {
			order[i] = string(rune(b))
		}

		req := &pb.ReorderTeamsRequest{
			Order: order,
		}

		_, _ = server.ReorderTeams(context.Background(), req)
	})
}
