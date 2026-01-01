package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"squash-ladder/server"
	ladderpb "squash-ladder/server/gen/ladder"
)

func main() {
	// Create a temporary file for the database
	tmpfile, err := os.CreateTemp("", "ladder_verify_*.jsonl")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up
	tmpfile.Close()

	// Pick ports for testing
	grpcPort := "9091"
	httpPort := "8081"

	// Start Server
	go func() {
		cfg := server.Config{
			DataPath: tmpfile.Name(),
			HTTPPort: httpPort,
			GRPCPort: grpcPort,
		}
		if err := server.Run(cfg); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// Give it a moment to start
	time.Sleep(1 * time.Second)

	// Connect to server
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := ladderpb.NewLadderServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Add Players
	fmt.Println("Adding players...")
	alice, err := c.AddPlayer(ctx, &ladderpb.AddPlayerRequest{Name: "Alice"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Added Alice: %s\n", alice.Player.Id)

	bob, err := c.AddPlayer(ctx, &ladderpb.AddPlayerRequest{Name: "Bob"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Added Bob: %s\n", bob.Player.Id)

	charlie, err := c.AddPlayer(ctx, &ladderpb.AddPlayerRequest{Name: "Charlie"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Added Charlie: %s\n", charlie.Player.Id)

	// 2. List Players
	listResp, err := c.ListPlayers(ctx, &ladderpb.ListPlayersRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nCurrent Ladder:")
	for _, p := range listResp.Players {
		fmt.Printf("%d. %s (%s)\n", p.Rank, p.Name, p.Id)
	}

	// 3. Record Match: Charlie (Rank 3) beats Alice (Rank 1) -> Charlie becomes Rank 1, Alice -> 2, Bob -> 3
	// Wait, logic says: if lower rank (Rank 3) beats higher rank (Rank 1).
	// Winner (Charlie) takes loser's position (Rank 1).
	// Everyone from Loser (Rank 1) to Winner-1 (Rank 2) moves down.
	// So Alice (old 1) -> 2. Bob (old 2) -> 3.
	// New Order: 1. Charlie, 2. Alice, 3. Bob
	fmt.Println("\nMatch: Charlie beats Alice (11-9, 11-8, 9-11, 11-5)")
	_, err = c.AddMatchResult(ctx, &ladderpb.AddMatchResultRequest{
		Player1Id: alice.Player.Id,
		Player2Id: charlie.Player.Id,
		WinnerId:  charlie.Player.Id,
		SetScores: []*ladderpb.SetScore{
			{Player1Points: 9, Player2Points: 11},
			{Player1Points: 8, Player2Points: 11},
			{Player1Points: 11, Player2Points: 9},
			{Player1Points: 5, Player2Points: 11},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 4. Verify Ladder
	listResp, err = c.ListPlayers(ctx, &ladderpb.ListPlayersRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nLadder After Match:")
	for _, p := range listResp.Players {
		fmt.Printf("%d. %s (%s)\n", p.Rank, p.Name, p.Id)
	}

	// Check Order
	if listResp.Players[0].Name != "Charlie" || listResp.Players[1].Name != "Alice" || listResp.Players[2].Name != "Bob" {
		log.Fatal("Ladder order is incorrect!")
	}

	fmt.Println("\nVerification Successful!")
}
