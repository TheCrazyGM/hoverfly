package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/srbde/hoverfly/rpc"
	"github.com/srbde/hoverfly/state"
)

func main() {
	port := flag.Int("port", 8090, "port to listen on")
	dbPath := flag.String("db", "", "path to BadgerDB directory (default is in-memory ephemeral)")
	reset := flag.Bool("reset", false, "reset/wipe the database directory on startup")
	debug := flag.Bool("debug", false, "enable verbose debug logging (logs every RPC call)")
	strict := flag.Bool("strict", false, "run server in strict mode (verifies state and disables fallbacks)")
	flag.Parse()

	fmt.Println("=====================================================")
	fmt.Println("     🛸 HOVERFLY - HIVE MOCK BLOCKCHAIN SERVER 🛸")
	fmt.Println("=====================================================")

	// 1. Initialize State Manager
	s, err := state.NewState(*dbPath, *reset)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize state manager: %v", err)
	}
	defer s.Close()

	if *dbPath == "" {
		log.Info("State Manager: Ephemeral Mode (In-Memory)")
	} else {
		log.Infof("State Manager: Persistent Mode (Directory: %s)", *dbPath)
		if *reset {
			log.Info("State Manager: Wiped and reset database directory")
		}
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Info("Logging: Debug Mode Enabled (Verbose RPC logs)")
	}

	// 2. Print Seeded Accounts & Rotating Key Pairs
	fmt.Println("🔑 Seeded Accounts & Rotating Key Pairs:")
	if alice, err := s.GetAccount("alice"); err == nil && alice != nil {
		fmt.Printf("  👤 @%s\n", alice.Name)
		fmt.Printf("     🔑 Active  - Pub: %s\n", alice.ActiveKey)
		fmt.Printf("                - WIF: %s\n", alice.ActivePrivateKey)
		fmt.Printf("     🔑 Posting - Pub: %s\n", alice.PostingKey)
		fmt.Printf("                - WIF: %s\n", alice.PostingPrivateKey)
	}
	if bob, err := s.GetAccount("bob"); err == nil && bob != nil {
		fmt.Printf("  👤 @%s\n", bob.Name)
		fmt.Printf("     🔑 Active  - Pub: %s\n", bob.ActiveKey)
		fmt.Printf("                - WIF: %s\n", bob.ActivePrivateKey)
		fmt.Printf("     🔑 Posting - Pub: %s\n", bob.PostingKey)
		fmt.Printf("                - WIF: %s\n", bob.PostingPrivateKey)
	}
	fmt.Println("=====================================================")

	// 3. Start Background Block Ticker (simulate block generation every 3 seconds)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			props, err := s.GetDynamicProperties()
			if err == nil && props != nil {
				props.HeadBlockNumber++
				props.LastIrreversibleBlockNum = props.HeadBlockNumber - 10
				props.Time = time.Now().UTC().Format("2006-01-02T15:04:05")
				props.HeadBlockID = fmt.Sprintf("05f5e100f72d57fd5a542459a94f3a8153c68c%02d", props.HeadBlockNumber%100)
				s.SaveDynamicProperties(props)
			}
		}
	}()
	log.Info("Block Generator: Active (producing mock blocks every 3s)")

	// 4. Set up JSON-RPC Handler
	handler := rpc.NewRPCHandler(s, *debug, *strict)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			dbMode := "Ephemeral (In-Memory)"
			if *dbPath != "" {
				dbMode = fmt.Sprintf("Persistent (%s)", *dbPath)
			}
			fmt.Fprintf(w, "🛸 Hoverfly Mock Hive Server is running!\nMode: %s\nPort: %d\nTime: %s\n", dbMode, *port, time.Now().UTC().Format(time.RFC3339))
			return
		}
		handler.ServeHTTP(w, r)
	})

	log.Infof("Server: Listening on http://localhost:%d", *port)
	fmt.Println("=====================================================")

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatalf("Fatal: HTTP server error: %v", err)
	}
}
