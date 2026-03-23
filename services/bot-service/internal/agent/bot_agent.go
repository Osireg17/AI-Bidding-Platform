package agent

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/bidclient"
	"github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/domain"
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

// modelFallbackChain is tried in order when a model returns a rate-limit error.
var modelFallbackChain = []string{
	"gemini-2.5-flash",
	"gemini-2.0-flash",
}

// isRateLimit reports whether err is a Gemini 429 / resource-exhausted response.
func isRateLimit(err error) bool {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == 429
	}
	return false
}

// AuctionContext holds the information passed to a bot agent for evaluation.
type AuctionContext struct {
	AuctionID    int64
	Title        string
	Description  string
	StartPrice   float64
	HighestBid   float64
	EndTime      time.Time
	TriggerEvent string
}

// BotAgent wraps an ADK llmagent for a single bot personality.
type BotAgent struct {
	bot          *domain.Bot
	agent        adkagent.Agent
	bidClient    *bidclient.BidServiceClient
	repo         domain.BotBidRepository
	logger       *zap.Logger
	geminiAPIKey string
	placeBidTool adktool.Tool
	instruction  string
}

type placeBidArgs struct {
	AuctionID int64   `json:"auction_id"`
	Amount    float64 `json:"amount"`
}

type placeBidResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

const bidRule = `
CRITICAL RULE: Any bid you place MUST be strictly greater than the Highest Bid shown in the auction context.
If Highest Bid is 0, your bid must exceed the Start Price.
Never place a bid equal to or lower than the Highest Bid — it will always be rejected.`

var personalityInstructions = map[domain.BotPersonality]string{
	domain.PersonalityAggressive: `You are Aggressive Alice. You love winning auctions. When you see a new auction,
always bid approximately 25% above the current highest bid immediately. Be bold.` + bidRule,

	domain.PersonalitySniper: `You are Sniper Steve. You wait for the perfect moment. Only bid when an auction
is ending soon or when you have been outbid. Bid 1-5% above the current highest bid.` + bidRule,

	domain.PersonalityValue: `You are Value Victor. You are disciplined. Estimate the fair market value of the
item from its title and description. Only use the place_bid tool if the current highest bid
is less than 70% of your estimated value. Bid at 80% of your estimate, but only if that
amount is strictly greater than the current highest bid.` + bidRule,

	domain.PersonalityChaos: `You are Chaos Charlie. You are unpredictable. Randomly decide whether to bid
(roughly 50% of the time). If you bid, choose a random amount between 10% and
50% above the current highest bid.` + bidRule,
}

func (ba *BotAgent) ID() int64    { return ba.bot.ID }
func (ba *BotAgent) Name() string { return ba.bot.Name }

func NewBotAgent(ctx context.Context, bot *domain.Bot, geminiAPIKey string, bidClient *bidclient.BidServiceClient, repo domain.BotBidRepository, logger *zap.Logger) (*BotAgent, error) {
	ba := &BotAgent{
		bot:       bot,
		bidClient: bidClient,
		repo:      repo,
		logger:    logger,
	}

	placeBidTool, err := functiontool.New(functiontool.Config{
		Name:        "place_bid",
		Description: "Place a bid on an auction. Call this when you decide to bid.",
	}, func(toolCtx adktool.Context, args placeBidArgs) (placeBidResult, error) {
		err := bidClient.PlaceBid(toolCtx, args.AuctionID, bot.ID, args.Amount)
		if err != nil {
			logger.Warn("bid rejected",
				zap.String("bot", bot.Name),
				zap.Int64("auction_id", args.AuctionID),
				zap.Float64("amount", args.Amount),
				zap.Error(err),
			)
			return placeBidResult{Success: false, Message: err.Error()}, fmt.Errorf("place bid failed: %w", err)
		}

		botBid, err := domain.NewBotBid(bot.ID, args.AuctionID, args.Amount)
		if err != nil {
			return placeBidResult{Success: false, Message: err.Error()}, nil
		}

		if err := repo.Create(toolCtx, botBid); err != nil {
			logger.Error("failed to persist bot bid",
				zap.String("bot", bot.Name),
				zap.Int64("auction_id", args.AuctionID),
				zap.Error(err),
			)
			return placeBidResult{Success: false, Message: "bid placed but failed to persist record"}, fmt.Errorf("failed to persist bot bid: %w", err)
		}

		logger.Info("bid placed",
			zap.String("bot", bot.Name),
			zap.Int64("auction_id", args.AuctionID),
			zap.Float64("amount", args.Amount),
		)

		return placeBidResult{Success: true, Message: "bid placed successfully"}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create place_bid tool: %w", err)
	}

	instruction, ok := personalityInstructions[bot.Personality]
	if !ok {
		return nil, fmt.Errorf("unknown bot personality: %s", bot.Personality)
	}

	llm, err := gemini.NewModel(ctx, modelFallbackChain[0], &genai.ClientConfig{
		APIKey: geminiAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini model: %w", err)
	}

	a, err := llmagent.New(llmagent.Config{
		Name:        bot.Name,
		Description: string(bot.Personality) + " bidding bot",
		Model:       llm,
		Instruction: instruction,
		Tools:       []adktool.Tool{placeBidTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llmagent: %w", err)
	}

	ba.agent = a
	ba.geminiAPIKey = geminiAPIKey
	ba.placeBidTool = placeBidTool
	ba.instruction = instruction
	return ba, nil
}

func (ba *BotAgent) Evaluate(ctx context.Context, ac AuctionContext) error {
	delay := time.Duration(5+rand.IntN(11)) * time.Second
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	minBid := ac.HighestBid
	if minBid == 0 {
		minBid = ac.StartPrice
	}
	msg := genai.NewContentFromText(fmt.Sprintf(
		"Auction ID: %d\nTitle: %s\nDescription: %s\nStart Price: %.2f\nHighest Bid: %.2f\nMinimum Valid Bid: %.2f\nEnds At: %s\nEvent: %s\n\nDecide whether to bid. Your bid MUST be greater than %.2f or it will be rejected.",
		ac.AuctionID, ac.Title, ac.Description, ac.StartPrice, ac.HighestBid,
		minBid, ac.EndTime.Format(time.RFC3339), ac.TriggerEvent, minBid,
	), genai.RoleUser)

	return ba.runWithFallback(ctx, msg)
}

// runWithFallback attempts each model in modelFallbackChain in order.
// On a 429 it logs, waits 30 s, then tries the next model.
func (ba *BotAgent) runWithFallback(ctx context.Context, msg *genai.Content) error {
	const rateLimitBackoff = 30 * time.Second

	for i, modelName := range modelFallbackChain {
		ag, err := ba.buildAgent(ctx, modelName)
		if err != nil {
			return fmt.Errorf("failed to build agent with model %s: %w", modelName, err)
		}

		runErr := ba.runAgent(ctx, ag, msg)
		if runErr == nil {
			return nil
		}

		if isRateLimit(runErr) && i < len(modelFallbackChain)-1 {
			ba.logger.Warn("rate limited, falling back to next model",
				zap.String("bot", ba.bot.Name),
				zap.String("model", modelName),
				zap.String("next_model", modelFallbackChain[i+1]),
				zap.Duration("backoff", rateLimitBackoff),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(rateLimitBackoff):
			}
			continue
		}

		return fmt.Errorf("agent run error (model=%s): %w", modelName, runErr)
	}

	return fmt.Errorf("all models exhausted for bot %s", ba.bot.Name)
}

func (ba *BotAgent) buildAgent(ctx context.Context, modelName string) (adkagent.Agent, error) {
	llm, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		APIKey: ba.geminiAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini model %s: %w", modelName, err)
	}

	return llmagent.New(llmagent.Config{
		Name:        ba.bot.Name,
		Description: string(ba.bot.Personality) + " bidding bot",
		Model:       llm,
		Instruction: ba.instruction,
		Tools:       []adktool.Tool{ba.placeBidTool},
	})
}

func (ba *BotAgent) runAgent(ctx context.Context, ag adkagent.Agent, msg *genai.Content) error {
	sessionSvc := session.InMemoryService()
	userID := fmt.Sprintf("bot-%d", ba.bot.ID)
	sessionID := fmt.Sprintf("auction-%d-%d", ba.bot.ID, time.Now().UnixNano())

	if _, err := sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   ba.bot.Name,
		UserID:    userID,
		SessionID: sessionID,
	}); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:        ba.bot.Name,
		Agent:          ag,
		SessionService: sessionSvc,
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	for event, err := range r.Run(ctx, userID, sessionID, msg, adkagent.RunConfig{}) {
		if err != nil {
			return err
		}
		_ = event
	}
	return nil
}
