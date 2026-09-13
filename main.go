package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gtps/VallenSource/VallenConfig"
	"gtps/VallenSource/VallenDatabase"
	"gtps/VallenSource/VallenHttps"
	"gtps/VallenSource/VallenItems"
	"gtps/VallenSource/VallenServer"
	"gtps/VallenSource/VallenStore"
	ui "gtps/VallenSource/VallenUI"
)

func main() {
	// Initialize Terminal UI first (initial status is already STARTING)
	termUI := ui.NewTerminalUI()

	// Setup log router to redirect all logs to TUI panels
	logRouter := ui.NewLogRouter(termUI.GetHTTPSWriter(), termUI.GetGTPSWriter())
	log.SetOutput(logRouter)
	log.SetFlags(0) // Remove default timestamp since LogRouter adds it

	var (
		srv *server.Server
		db  *database.JSONDatabase
	)

	// Setup signal channel for external kill / SIGINT / SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start all servers in background
	go func() {
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		if enetInit() != 0 {
			log.Fatal("Failed to initialize ENet")
		}

		log.Println("ENet initialized successfully")

		log.Println("Loading items.dat...")
		itemDB, err := items.Load("items.dat")
		if err != nil {
			log.Fatalf("Failed to load items.dat: %v", err)
		}
		log.Printf("[Items] Loaded: %d items (hash: %d)", len(itemDB.Items), itemDB.Hash)

		store.LoadStore("VallenSetting/resources/store.txt")

		var dbErr error
		db, dbErr = database.NewJSONDatabase("./databasevallen")
		if dbErr != nil {
			log.Fatalf("Failed to load database: %v", dbErr)
		}

		log.Printf("Database loaded: %d players, %d worlds", db.PlayerCount(), db.WorldCount())

		httpsServer := https.NewHTTPSServer(443)
		go func() {
			log.Println("[HTTPS] Starting HTTPS server on port 443...")
			if err := httpsServer.Start(); err != nil {
				log.Printf("[HTTPS] Server error: %v", err)
				log.Println("[HTTPS] [WARN] HTTPS server failed. Client won't be able to get server_data!")
			}
		}()

		srv = server.New(cfg, db)
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}

		log.Printf("ENet Server started on %s:%d", cfg.Host, cfg.Port)
		log.Println("[Server] Ready! Waiting for connections...")

		// Set status to ONLINE after all servers started
		termUI.SetServerStatus("ONLINE")
	}()

	// Background listener for OS signals to stop TUI cleanly
	go func() {
		<-sigChan
		termUI.Stop()
	}()

	// Start TUI - blocks until app.Stop() is called (via Ctrl+C or sigChan)
	if err := termUI.Start(); err != nil {
		// Fallback to console mode if TUI cannot initialize (e.g. headless, non-tty)
		log.SetOutput(os.Stdout)
		log.Printf("[UI] TUI mode not available (%v). Running in console mode.", err)
		<-sigChan
	}

	// Graceful shutdown after TUI exits
	termUI.SetServerStatus("OFFLINE")
	log.SetOutput(os.Stdout)
	log.Println("\nShutting down server...")

	if srv != nil {
		srv.Stop()
	}
	if db != nil {
		db.Save()
		log.Println("Database saved successfully.")
	}
	enetDeinit()
	log.Println("Server terminated cleanly. Goodbye!")
}
