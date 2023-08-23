package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/salesforce"
	"golang.org/x/oauth2"
)

// To run this test, make sure you have the sfdx CLI installed, and are logged in to a Salesforce instance.

// Then add your username as a command line argument, e.g.
// go run test/salesforce.go -login myusername@mydomain

// You can optionally add an `instance` argument to specify a Salesforce instance,
// or leave empty to use the Ampersand's dev instance.

const TimeoutSeconds = 30

func main() {
	os.Exit(mainFn())
}

func mainFn() int {
	instance := flag.String("instance", "ampersand-dev-ed.develop", "Salesforce instance")
	flag.Parse()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	cfg := &oauth2.Config{
		ClientID:     "<client id>",
		ClientSecret: "<client secret>",
		Endpoint: oauth2.Endpoint{
			AuthURL:   fmt.Sprintf("https://%s.my.salesforce.com/services/oauth2/authorize", *instance),
			TokenURL:  fmt.Sprintf("https://%s.my.salesforce.com/services/oauth2/token", *instance),
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	tok := &oauth2.Token{
		AccessToken:  "<access token>",
		RefreshToken: "<refresh token>",
		TokenType:    "bearer",
		Expiry:       time.Now().Add(-1 * time.Hour), // just pretend it's expired already, whatever, it'll fetch a new one.
	}

	ctx := context.Background()

	// Create a new Salesforce connector, with a token provider that uses the sfdx CLI to fetch an access token.
	sfc, err := connectors.Salesforce.New(ctx,
		salesforce.WithOAuthConfig(cfg),
		salesforce.WithOAuthToken(tok),
		salesforce.WithWorkspace(*instance))
	if err != nil {
		slog.Error("Error creating Salesforce connector", "error", err)

		return 1
	}

	defer func() {
		_ = sfc.Close()
	}()

	if err := testConnector(ctx, sfc); err != nil {
		slog.Error("Error testing", "connector", sfc, "error", err)

		return 1
	}

	return 0
}

func testConnector(ctx context.Context, conn connectors.Connector) error {
	// Create a context with a timeout
	ctx, done := context.WithTimeout(ctx, TimeoutSeconds*time.Second)
	defer done()

	// Read some data from Salesforce
	res, err := conn.Read(ctx, connectors.ReadParams{
		ObjectName:     "Account",
		Fields:         []string{"Id", "Name", "BillingCity", "IsDeleted"},
		IncludeDeleted: true,
	})
	if err != nil {
		return fmt.Errorf("error reading from Salesforce: %w", err)
	}

	// Print the results
	jsonStr, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	_, _ = os.Stdout.Write(jsonStr)
	_, _ = os.Stdout.WriteString("\n")

	return nil
}
