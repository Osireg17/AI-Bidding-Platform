package agent

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// GeneratedAuction holds the LLM-generated auction item fields.
type GeneratedAuction struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartPrice  float64 `json:"start_price"`
	DurationSec int     `json:"duration_sec"`
}

// AuctionAgent uses a Gemini LLM to generate new auction items.
type AuctionAgent struct {
	geminiAPIKey string
	logger       *zap.Logger
}

const auctionGeneratorInstruction = `You are an auction item generator for an online bidding platform.
Your job is to create interesting, realistic auction items that bidding bots would want to compete for.

When asked to generate an auction item, you MUST call the create_auction_item tool with:
- title: A short, specific item name (e.g. "Vintage Rolex Submariner", "2019 MacBook Pro 16-inch")
- description: 1-2 sentences describing the item and why it's valuable
- start_price: A realistic starting price in GBP for the item (consider its actual market value — range from £1 to £10,000)
- duration_sec: How long the auction should run in seconds (between 60 and 300 seconds)

Vary the items — mix electronics, collectibles, sports items, fashion, art, books, and more.
Make the items feel real and specific, not generic.`

// NewAuctionAgent creates a new auction generator agent backed by Gemini.
func NewAuctionAgent(geminiAPIKey string, logger *zap.Logger) *AuctionAgent {
	return &AuctionAgent{
		geminiAPIKey: geminiAPIKey,
		logger:       logger,
	}
}

// Generate asks the Gemini LLM to create a new auction item.
// The agent is re-created per call so the captured closure is always fresh.
func (a *AuctionAgent) Generate(ctx context.Context) (*GeneratedAuction, error) {
	var captured *GeneratedAuction

	createItemTool, err := functiontool.New(functiontool.Config{
		Name:        "create_auction_item",
		Description: "Create a new auction item with the given fields",
	}, func(_ adktool.Context, args GeneratedAuction) (GeneratedAuction, error) {
		captured = &args
		return args, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tool: %w", err)
	}

	llm, err := gemini.NewModel(ctx, "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: a.geminiAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini model: %w", err)
	}

	ag, err := llmagent.New(llmagent.Config{
		Name:        "AuctionGenerator",
		Description: "Generates auction items for the bidding platform",
		Model:       llm,
		Instruction: auctionGeneratorInstruction,
		Tools:       []adktool.Tool{createItemTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llmagent: %w", err)
	}

	sessionSvc := session.InMemoryService()
	userID := "auction-generator"
	sessionID := fmt.Sprintf("gen-%d", time.Now().UnixNano())

	if _, err := sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   "AuctionGenerator",
		UserID:    userID,
		SessionID: sessionID,
	}); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:        "AuctionGenerator",
		Agent:          ag,
		SessionService: sessionSvc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	msg := genai.NewContentFromText("Generate a new auction item now.", genai.RoleUser)

	for event, err := range r.Run(ctx, userID, sessionID, msg, adkagent.RunConfig{}) {
		if err != nil {
			return nil, fmt.Errorf("agent run error: %w", err)
		}
		_ = event
	}

	if captured == nil {
		return nil, fmt.Errorf("agent did not call create_auction_item tool")
	}

	a.logger.Info("auction item generated",
		zap.String("title", captured.Title),
		zap.Float64("start_price", captured.StartPrice),
		zap.Int("duration_sec", captured.DurationSec),
	)

	return captured, nil
}
