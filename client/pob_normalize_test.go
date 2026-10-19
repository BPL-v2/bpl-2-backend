package client

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePoBXMLIgnoringGemLevels_OnlyGemLevelsDiffer(t *testing.T) {
	oldXML, err := os.ReadFile("testdata/pob_gem_old.xml")
	require.NoError(t, err)
	newXML, err := os.ReadFile("testdata/pob_gem_new.xml")
	require.NoError(t, err)
	assert.NotEqual(t, string(oldXML), string(newXML))
	assert.Equal(t, NormalizePoBXMLIgnoringGemLevels(string(oldXML)), NormalizePoBXMLIgnoringGemLevels(string(newXML)))
}

func TestNormalizePoBXMLIgnoringGemLevels_DetectsRealChanges(t *testing.T) {
	oldXML, err := os.ReadFile("testdata/pob_gem_old.xml")
	require.NoError(t, err)
	base := NormalizePoBXMLIgnoringGemLevels(string(oldXML))

	swappedGem := strings.Replace(string(oldXML), `nameSpec="Haste"`, `nameSpec="Grace"`, 1)
	assert.NotEqual(t, base, NormalizePoBXMLIgnoringGemLevels(swappedGem))

	changedItem := strings.Replace(string(oldXML), "+41 to maximum Mana", "+42 to maximum Mana", 1)
	assert.NotEqual(t, base, NormalizePoBXMLIgnoringGemLevels(changedItem))
}

func TestItemEquals_FlaggedModsFromSeparateParses(t *testing.T) {
	js := `{"name":"a","typeLine":"b","baseType":"c","explicitMods":[{"description":"x","flags":{"fractured":true}}]}`
	var a, b, c Item
	require.NoError(t, json.Unmarshal([]byte(js), &a))
	require.NoError(t, json.Unmarshal([]byte(js), &b))
	require.NoError(t, json.Unmarshal([]byte(strings.Replace(js, "fractured", "crafted", 1)), &c))
	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestNormalizePoBXMLIgnoringGemLevels_QualityOnly(t *testing.T) {
	gem := func(quality string) string {
		return `<Gem level="20" quality="` + quality + `" nameSpec="Haste" skillId="Haste"/>`
	}
	assert.Equal(t, NormalizePoBXMLIgnoringGemLevels(gem("0")), NormalizePoBXMLIgnoringGemLevels(gem("20")))
}

func TestNormalizePoBXMLIgnoringGemLevels_IgnoresElementOrder(t *testing.T) {
	a := `<PathOfBuilding>
	<Build level="98"><PlayerStat stat="TotalDPS" value="1"/></Build>
	<Tree><Spec nodes="3,1,2" masteryEffects="{1,2},{3,4}"><URL>abc</URL></Spec></Tree>
	<Skills><Skill label="a"><Gem nameSpec="Haste" level="14" quality="0"/><Gem nameSpec="Grace" level="1" quality="0"/></Skill><Skill label="b"/></Skills>
</PathOfBuilding>`
	b := `<PathOfBuilding>
	<Skills><Skill label="b"/><Skill label="a"><Gem nameSpec="Grace" level="20" quality="20"/><Gem nameSpec="Haste" level="15" quality="10"/></Skill></Skills>
	<Tree><Spec masteryEffects="{3,4},{1,2}" nodes="1,2,3"><URL>abc</URL></Spec></Tree>
	<Build level="98"><PlayerStat stat="TotalDPS" value="2"/></Build>
</PathOfBuilding>`
	assert.Equal(t, NormalizePoBXMLIgnoringGemLevels(a), NormalizePoBXMLIgnoringGemLevels(b))

	changedNode := strings.Replace(b, `nodes="1,2,3"`, `nodes="1,2,4"`, 1)
	assert.NotEqual(t, NormalizePoBXMLIgnoringGemLevels(a), NormalizePoBXMLIgnoringGemLevels(changedNode))
	changedGem := strings.Replace(b, `nameSpec="Grace"`, `nameSpec="Haste"`, 1)
	assert.NotEqual(t, NormalizePoBXMLIgnoringGemLevels(a), NormalizePoBXMLIgnoringGemLevels(changedGem))
}
