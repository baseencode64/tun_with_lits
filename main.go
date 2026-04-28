package main

	"time"
>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)

	"github.com/goxray/tun/pkg/client"
)

<<<<<<< HEAD
var cmdArgsErr = `ERROR: no config_link provided
usage: %s <config_url>
  - config_url - xray connection link, like "vless://example..."
var cmdArgsErr = `ERROR: no config provided
usage: %s <config_url_or_link>
  - config_url - xray connection link, like "vless://example..."
  - or raw list URL: --from-raw <https://example.com/links.txt>
=======
	"time"
>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)

	"github.com/goxray/tun/pkg/client"
)

<<<<<<< HEAD
var cmdArgsErr = `ERROR: no config_link provided
usage: %s <config_url>
  - config_url - xray connection link, like "vless://example..."
=======
var cmdArgsErr = `ERROR: no config provided
usage: %s <config_url_or_link>
  - config_url - xray connection link, like "vless://example..."
  - or raw list URL: --from-raw <https://example.com/links.txt>
>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)
  - or set GOXRAY_CONFIG_URL env var
`

func main() {
<<<<<<< HEAD
	// Get connection link from first cmd argument or env var.
	var clientLink string
	if len(os.Args[1:]) > 0 {
		clientLink = os.Args[1]
	} else {
		clientLink = os.Getenv("GOXRAY_CONFIG_URL")
	}
	if clientLink == "" {
=======
	// Get connection link from cmd arguments
	var clientLink string
	var fromRaw bool
	var rawURL string

	args := os.Args[1:]

	// Check for --from-raw flag
	if len(args) > 0 && args[0] == "--from-raw" {
		fromRaw = true
		if len(args) < 2 {
			fmt.Printf(cmdArgsErr, os.Args[0])
			os.Exit(1)
		}
		rawURL = args[1]
	} else if len(args) > 0 {
		clientLink = args[0]
	} else {
		clientLink = os.Getenv("GOXRAY_CONFIG_URL")
	}

	if clientLink == "" && !fromRaw {
>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)
		fmt.Printf(cmdArgsErr, os.Args[0])
		os.Exit(0)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	vpn, err := client.NewClientWithOpts(client.Config{
		TLSAllowInsecure: false,
		Logger:           logger,
	})
	if err != nil {
		log.Fatal(err)
	}

<<<<<<< HEAD
=======
	// If using raw URL, fetch and select best server
	if fromRaw {
		slog.Info("Fetching server list from raw URL", "url", rawURL)

		loggerAdapter := client.NewSlogAdapter(logger)
		selector := client.NewServerSelector(loggerAdapter, 5*time.Second, 10)
		best, err := selector.SelectBestFromURL(rawURL)
		if err != nil {
			log.Fatalf("Failed to select server: %v", err)
		}

		clientLink = best.Link
		slog.Info("Selected optimal server", "host", best.Host, "port", best.Port, "latency", best.Latency)
	}

>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)
	slog.Info("Connecting to VPN server")
	err = vpn.Connect(clientLink)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Connected to VPN server")
	<-sigterm
	slog.Info("Received term signal, disconnecting...")
	if err = vpn.Disconnect(context.Background()); err != nil {
		slog.Warn("Disconnecting VPN failed", "error", err)
		os.Exit(0)
	}

	slog.Info("VPN disconnected successfully")
	os.Exit(0)
}
