package client

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type PathOfBuilding struct {
	XMLName xml.Name `xml:"PathOfBuilding"`
	Build   Build    `xml:"Build"`
	Skills  Skills   `xml:"Skills"`
}

type Build struct {
	PlayerStats PlayerStats
}

type PlayerStats struct {
	AverageDamage                   float64
	AverageBurstDamage              float64
	Speed                           float64
	PreEffectiveCritChance          float64
	CritChance                      float64
	CritMultiplier                  float64
	HitChance                       float64
	TotalDPS                        float64
	TotalDot                        float64
	WithBleedDPS                    float64
	WithIgniteDPS                   float64
	PoisonDPS                       float64
	PoisonDamage                    float64
	WithPoisonDPS                   float64
	TotalDotDPS                     float64
	CullingDPS                      float64
	ReservationDPS                  float64
	CombinedDPS                     float64
	AreaOfEffectRadiusMetres        float64
	ManaCost                        float64
	ManaPercentCost                 float64
	ManaPerSecondCost               float64
	ManaPercentPerSecondCost        float64
	LifeCost                        float64
	LifePercentCost                 float64
	LifePerSecondCost               float64
	LifePercentPerSecondCost        float64
	ESCost                          float64
	ESPerSecondCost                 float64
	ESPercentPerSecondCost          float64
	RageCost                        float64
	SoulCost                        float64
	Str                             float64
	ReqStr                          float64
	Dex                             float64
	ReqDex                          float64
	Int                             float64
	ReqInt                          float64
	Devotion                        float64
	TotalEHP                        float64
	PhysicalMaximumHitTaken         float64
	LightningMaximumHitTaken        float64
	FireMaximumHitTaken             float64
	ColdMaximumHitTaken             float64
	ChaosMaximumHitTaken            float64
	MainHandAccuracy                float64
	Life                            float64
	SpecLifeInc                     float64
	LifeUnreserved                  float64
	LifeRecoverable                 float64
	LifeUnreservedPercent           float64
	LifeRegenRecovery               float64
	LifeLeechGainRate               float64
	Mana                            float64
	SpecManaInc                     float64
	ManaUnreserved                  float64
	ManaUnreservedPercent           float64
	ManaRegenRecovery               float64
	ManaLeechGainRate               float64
	EnergyShield                    float64
	EnergyShieldRecoveryCap         float64
	SpecEnergyShieldInc             float64
	EnergyShieldRegenRecovery       float64
	EnergyShieldLeechGainRate       float64
	Ward                            float64
	RageRegenRecovery               float64
	TotalBuildDegen                 float64
	TotalNetRegen                   float64
	NetLifeRegen                    float64
	NetManaRegen                    float64
	NetEnergyShieldRegen            float64
	Evasion                         float64
	SpecEvasionInc                  float64
	MeleeEvadeChance                float64
	ProjectileEvadeChance           float64
	Armour                          float64
	SpecArmourInc                   float64
	PhysicalDamageReduction         float64
	EffectiveBlockChance            float64
	EffectiveSpellBlockChance       float64
	AttackDodgeChance               float64
	SpellDodgeChance                float64
	EffectiveSpellSuppressionChance float64
	FireResist                      float64
	FireResistOverCap               float64
	ColdResist                      float64
	ColdResistOverCap               float64
	LightningResist                 float64
	LightningResistOverCap          float64
	ChaosResist                     float64
	ChaosResistOverCap              float64
	EffectiveMovementSpeedMod       float64
	FullDPS                         float64
	FullDotDPS                      float64
	PowerCharges                    float64
	PowerChargesMax                 float64
	FrenzyCharges                   float64
	FrenzyChargesMax                float64
	EnduranceCharges                float64
	EnduranceChargesMax             float64
}

type playerStatXML struct {
	Stat  string  `xml:"stat,attr"`
	Value float64 `xml:"value,attr"`
}

func (ps *PlayerStats) SetStat(stat string, value float64) {
	switch stat {
	case "AverageDamage":
		ps.AverageDamage = value
	case "AverageBurstDamage":
		ps.AverageBurstDamage = value
	case "Speed":
		ps.Speed = value
	case "PreEffectiveCritChance":
		ps.PreEffectiveCritChance = value
	case "CritChance":
		ps.CritChance = value
	case "CritMultiplier":
		ps.CritMultiplier = value
	case "HitChance":
		ps.HitChance = value
	case "TotalDPS":
		ps.TotalDPS = value
	case "TotalDot":
		ps.TotalDot = value
	case "WithBleedDPS":
		ps.WithBleedDPS = value
	case "WithIgniteDPS":
		ps.WithIgniteDPS = value
	case "PoisonDPS":
		ps.PoisonDPS = value
	case "PoisonDamage":
		ps.PoisonDamage = value
	case "WithPoisonDPS":
		ps.WithPoisonDPS = value
	case "TotalDotDPS":
		ps.TotalDotDPS = value
	case "CullingDPS":
		ps.CullingDPS = value
	case "ReservationDPS":
		ps.ReservationDPS = value
	case "CombinedDPS":
		ps.CombinedDPS = value
	case "AreaOfEffectRadiusMetres":
		ps.AreaOfEffectRadiusMetres = value
	case "ManaCost":
		ps.ManaCost = value
	case "ManaPercentCost":
		ps.ManaPercentCost = value
	case "ManaPerSecondCost":
		ps.ManaPerSecondCost = value
	case "ManaPercentPerSecondCost":
		ps.ManaPercentPerSecondCost = value
	case "LifeCost":
		ps.LifeCost = value
	case "LifePercentCost":
		ps.LifePercentCost = value
	case "LifePerSecondCost":
		ps.LifePerSecondCost = value
	case "LifePercentPerSecondCost":
		ps.LifePercentPerSecondCost = value
	case "ESCost":
		ps.ESCost = value
	case "ESPerSecondCost":
		ps.ESPerSecondCost = value
	case "ESPercentPerSecondCost":
		ps.ESPercentPerSecondCost = value
	case "RageCost":
		ps.RageCost = value
	case "SoulCost":
		ps.SoulCost = value
	case "Str":
		ps.Str = value
	case "ReqStr":
		ps.ReqStr = value
	case "Dex":
		ps.Dex = value
	case "ReqDex":
		ps.ReqDex = value
	case "Int":
		ps.Int = value
	case "ReqInt":
		ps.ReqInt = value
	case "Devotion":
		ps.Devotion = value
	case "TotalEHP":
		ps.TotalEHP = value
	case "PhysicalMaximumHitTaken":
		ps.PhysicalMaximumHitTaken = value
	case "LightningMaximumHitTaken":
		ps.LightningMaximumHitTaken = value
	case "FireMaximumHitTaken":
		ps.FireMaximumHitTaken = value
	case "ColdMaximumHitTaken":
		ps.ColdMaximumHitTaken = value
	case "ChaosMaximumHitTaken":
		ps.ChaosMaximumHitTaken = value
	case "MainHandAccuracy":
		ps.MainHandAccuracy = value
	case "Life":
		ps.Life = value
	case "Spec:LifeInc":
		ps.SpecLifeInc = value
	case "LifeUnreserved":
		ps.LifeUnreserved = value
	case "LifeRecoverable":
		ps.LifeRecoverable = value
	case "LifeUnreservedPercent":
		ps.LifeUnreservedPercent = value
	case "LifeRegenRecovery":
		ps.LifeRegenRecovery = value
	case "LifeLeechGainRate":
		ps.LifeLeechGainRate = value
	case "Mana":
		ps.Mana = value
	case "Spec:ManaInc":
		ps.SpecManaInc = value
	case "ManaUnreserved":
		ps.ManaUnreserved = value
	case "ManaUnreservedPercent":
		ps.ManaUnreservedPercent = value
	case "ManaRegenRecovery":
		ps.ManaRegenRecovery = value
	case "ManaLeechGainRate":
		ps.ManaLeechGainRate = value
	case "EnergyShield":
		ps.EnergyShield = value
	case "EnergyShieldRecoveryCap":
		ps.EnergyShieldRecoveryCap = value
	case "Spec:EnergyShieldInc":
		ps.SpecEnergyShieldInc = value
	case "EnergyShieldRegenRecovery":
		ps.EnergyShieldRegenRecovery = value
	case "EnergyShieldLeechGainRate":
		ps.EnergyShieldLeechGainRate = value
	case "Ward":
		ps.Ward = value
	case "RageRegenRecovery":
		ps.RageRegenRecovery = value
	case "TotalBuildDegen":
		ps.TotalBuildDegen = value
	case "TotalNetRegen":
		ps.TotalNetRegen = value
	case "NetLifeRegen":
		ps.NetLifeRegen = value
	case "NetManaRegen":
		ps.NetManaRegen = value
	case "NetEnergyShieldRegen":
		ps.NetEnergyShieldRegen = value
	case "Evasion":
		ps.Evasion = value
	case "Spec:EvasionInc":
		ps.SpecEvasionInc = value
	case "MeleeEvadeChance":
		ps.MeleeEvadeChance = value
	case "ProjectileEvadeChance":
		ps.ProjectileEvadeChance = value
	case "Armour":
		ps.Armour = value
	case "Spec:ArmourInc":
		ps.SpecArmourInc = value
	case "PhysicalDamageReduction":
		ps.PhysicalDamageReduction = value
	case "EffectiveBlockChance":
		ps.EffectiveBlockChance = value
	case "EffectiveSpellBlockChance":
		ps.EffectiveSpellBlockChance = value
	case "AttackDodgeChance":
		ps.AttackDodgeChance = value
	case "SpellDodgeChance":
		ps.SpellDodgeChance = value
	case "EffectiveSpellSuppressionChance":
		ps.EffectiveSpellSuppressionChance = value
	case "FireResist":
		ps.FireResist = value
	case "FireResistOverCap":
		ps.FireResistOverCap = value
	case "ColdResist":
		ps.ColdResist = value
	case "ColdResistOverCap":
		ps.ColdResistOverCap = value
	case "LightningResist":
		ps.LightningResist = value
	case "LightningResistOverCap":
		ps.LightningResistOverCap = value
	case "ChaosResist":
		ps.ChaosResist = value
	case "ChaosResistOverCap":
		ps.ChaosResistOverCap = value
	case "EffectiveMovementSpeedMod":
		ps.EffectiveMovementSpeedMod = value
	case "FullDPS":
		ps.FullDPS = value
	case "FullDotDPS":
		ps.FullDotDPS = value
	case "PowerCharges":
		ps.PowerCharges = value
	case "PowerChargesMax":
		ps.PowerChargesMax = value
	case "FrenzyCharges":
		ps.FrenzyCharges = value
	case "FrenzyChargesMax":
		ps.FrenzyChargesMax = value
	case "EnduranceCharges":
		ps.EnduranceCharges = value
	case "EnduranceChargesMax":
		ps.EnduranceChargesMax = value
	}
}

func (b *Build) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "PlayerStat" {
				var stat playerStatXML
				if err := d.DecodeElement(&stat, &se); err != nil {
					return err
				}
				b.PlayerStats.SetStat(stat.Stat, stat.Value)
			}
		}
		if end, ok := t.(xml.EndElement); ok && end.Name.Local == start.Name.Local {
			break
		}
	}
	return nil
}

// --- Skills Section Structs ---
type Skills struct {
	ActiveSkillSet      int        `xml:"activeSkillSet,attr"`
	SortGemsByDPS       string     `xml:"sortGemsByDPS,attr"`
	SortGemsByDPSField  string     `xml:"sortGemsByDPSField,attr"`
	ShowSupportGemTypes string     `xml:"showSupportGemTypes,attr"`
	ShowAltQualityGems  string     `xml:"showAltQualityGems,attr"`
	DefaultGemLevel     string     `xml:"defaultGemLevel,attr"`
	DefaultGemQuality   string     `xml:"defaultGemQuality,attr"`
	SkillSets           []SkillSet `xml:"SkillSet"`
}

type SkillSet struct {
	ID     int     `xml:"id,attr"`
	Skills []Skill `xml:"Skill"`
}

type Skill struct {
	Label                string `xml:"label,attr"`
	Slot                 string `xml:"slot,attr"`
	MainActiveSkillCalcs string `xml:"mainActiveSkillCalcs,attr"`
	MainActiveSkill      string `xml:"mainActiveSkill,attr"`
	IncludeInFullDPS     string `xml:"includeInFullDPS,attr"`
	Enabled              string `xml:"enabled,attr"`
	Gems                 []Gem  `xml:"Gem"`
}

type Gem struct {
	GemID         string `xml:"gemId,attr"`
	VariantID     string `xml:"variantId,attr"`
	EnableGlobal1 string `xml:"enableGlobal1,attr"`
	NameSpec      string `xml:"nameSpec,attr"`
	QualityID     string `xml:"qualityId,attr"`
	Enabled       string `xml:"enabled,attr"`
	EnableGlobal2 string `xml:"enableGlobal2,attr"`
	Quality       string `xml:"quality,attr"`
	SkillID       string `xml:"skillId,attr"`
	Count         string `xml:"count,attr"`
	Level         string `xml:"level,attr"`
	SkillPart     *int   `xml:"skillPart,attr"`
}

func DecodePoBExport(input string) (*PathOfBuilding, error) {
	input = strings.ReplaceAll(input, "-", "+")
	input = strings.ReplaceAll(input, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}
	b := bytes.NewReader(decoded)
	z, err := zlib.NewReader(b)
	if err != nil {
		return nil, fmt.Errorf("zlib decompress error: %w", err)
	}
	defer z.Close()
	var pob PathOfBuilding
	if err := xml.NewDecoder(z).Decode(&pob); err != nil {
		return nil, fmt.Errorf("xml decode error: %w", err)
	}
	return &pob, nil
}
