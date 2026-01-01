package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ladderpb "squash-ladder/server/gen/ladder"
)

func main() {
	// Connect to server
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := ladderpb.NewLadderServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Add Players
	fmt.Println("Adding players...")
	p1, err := c.AddPlayer(ctx, &ladderpb.AddPlayerRequest{Name: "P1"})
	if err != nil {
		log.Fatal(err)
	}
	p2, err := c.AddPlayer(ctx, &ladderpb.AddPlayerRequest{Name: "P2"})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Add Matches
	fmt.Println("Adding matches...")
	for i := 0; i < 5; i++ {
		_, err = c.AddMatchResult(ctx, &ladderpb.AddMatchResultRequest{
			Player1Id: p1.Player.Id,
			Player2Id: p2.Player.Id,
			WinnerId:  p1.Player.Id,
			SetScores: []*ladderpb.SetScore{
				{Player1Points: 11, Player2Points: 5},
				{Player1Points: 11, Player2Points: 5},
				{Player1Points: 11, Player2Points: 5},
			},
		})
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond) // Ensure slightly different timestamps
	}

	// 3. List Recent 3 Matches
	fmt.Println("Listing recent 3 matches...")
	resp, err := c.ListRecentMatches(ctx, &ladderpb.ListRecentMatchesRequest{Limit: 3})
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Results) != 3 {
		log.Fatalf("Expected 3 matches, got %d", len(resp.Results))
	}

	for i, m := range resp.Results {
		fmt.Printf("Match %d: Winner %s, Timestamp: %d, TxID: %s\n", i, m.WinnerId, m.TimestampMs, m.TransactionId)
		if m.TimestampMs == 0 {
			log.Fatalf("Timestamp should not be 0")
		}
		if m.TransactionId == "" {
			log.Fatalf("Transaction ID should not be empty")
		}
	}

	// Check order (should be descending by time implicitly because we read backwards)
	if resp.Results[0].TimestampMs < resp.Results[1].TimestampMs {
		log.Fatalf("Matches should be in reverse chronological order (newest first)")
	}

	fmt.Println("Verification Successful!")
}
