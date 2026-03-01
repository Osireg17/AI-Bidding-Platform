package domain

type BotPersonality string

const (
	PersonalityAggressive BotPersonality = "aggressive"
	PersonalitySniper     BotPersonality = "sniper"
	PersonalityValue      BotPersonality = "value"
	PersonalityChaos      BotPersonality = "chaos"
)

type Bot struct {
	ID          int64          `bun:",pk,autoincrement"`
	Name        string         `bun:",notnull"`
	Personality BotPersonality `bun:",notnull"`
}

var (
	Alice   = &Bot{ID: 1, Name: "Aggressive Alice", Personality: PersonalityAggressive}
	Steve   = &Bot{ID: 2, Name: "Sniper Steve", Personality: PersonalitySniper}
	Victor  = &Bot{ID: 3, Name: "Value Victor", Personality: PersonalityValue}
	Charlie = &Bot{ID: 4, Name: "Chaos Charlie", Personality: PersonalityChaos}

	AllBots = []*Bot{Alice, Steve, Victor, Charlie}
)
