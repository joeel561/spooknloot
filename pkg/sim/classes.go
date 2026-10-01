package sim

// Class is the character class a player picks before the game.
type Class uint8

const (
	ClassWarrior Class = iota
	ClassMage
	ClassHealer
	classCount
)

// AllClasses lists the classes in menu order.
var AllClasses = []Class{ClassWarrior, ClassMage, ClassHealer}

type ClassStats struct {
	Name        string
	Description string
	MaxHealth   float32
	Damage      float32 // per hit
	Range       float32 // attack reach in pixels
	ReviveTime  int     // frames of holding E to revive someone
	// AuraHeal heals nearby teammates by this fraction of their max
	// health every AuraInterval frames (healer only).
	AuraHeal float32
}

const (
	AuraInterval = 3 * 60
	AuraRadius   = 64
)

var classStats = [classCount]ClassStats{
	ClassWarrior: {
		Name:        "Warrior",
		Description: "Tough melee fighter with 30 health",
		MaxHealth:   30,
		Damage:      2.5,
		Range:       40,
		ReviveTime:  3 * 60,
	},
	ClassMage: {
		Name:        "Mage",
		Description: "Fragile, but hits hard from a distance",
		MaxHealth:   16,
		Damage:      3.5,
		Range:       90,
		ReviveTime:  3 * 60,
	},
	ClassHealer: {
		Name:        "Healer",
		Description: "Heals nearby allies, revives twice as fast",
		MaxHealth:   20,
		Damage:      1.5,
		Range:       40,
		ReviveTime:  90,
		AuraHeal:    0.05,
	},
}

// Stats returns the class values; unknown classes (e.g. from a bad
// packet) fall back to the warrior.
func (c Class) Stats() ClassStats {
	if c >= classCount {
		c = ClassWarrior
	}
	return classStats[c]
}

func (c Class) Valid() bool { return c < classCount }
