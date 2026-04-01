package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// BuddingAmethyst is the geode growth block that hosts amethyst buds.
type BuddingAmethyst struct {
	solid
}

// BreakInfo ...
func (b BuddingAmethyst) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(b))
}

// EncodeItem ...
func (BuddingAmethyst) EncodeItem() (string, int16) {
	return "minecraft:budding_amethyst", 0
}

// EncodeBlock ...
func (BuddingAmethyst) EncodeBlock() (string, map[string]any) {
	return "minecraft:budding_amethyst", nil
}

// SmallAmethystBud is a small bud attached to a budding amethyst face.
type SmallAmethystBud struct {
	empty
	transparent
	Face cube.Face
}

// BreakInfo ...
func (b SmallAmethystBud) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(b))
}

// EncodeItem ...
func (SmallAmethystBud) EncodeItem() (string, int16) {
	return "minecraft:small_amethyst_bud", 0
}

// EncodeBlock ...
func (b SmallAmethystBud) EncodeBlock() (string, map[string]any) {
	return "minecraft:small_amethyst_bud", map[string]any{"minecraft:block_face": b.Face.String()}
}

// MediumAmethystBud is a medium bud attached to a budding amethyst face.
type MediumAmethystBud struct {
	empty
	transparent
	Face cube.Face
}

// BreakInfo ...
func (b MediumAmethystBud) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(b))
}

// EncodeItem ...
func (MediumAmethystBud) EncodeItem() (string, int16) {
	return "minecraft:medium_amethyst_bud", 0
}

// EncodeBlock ...
func (b MediumAmethystBud) EncodeBlock() (string, map[string]any) {
	return "minecraft:medium_amethyst_bud", map[string]any{"minecraft:block_face": b.Face.String()}
}

// LargeAmethystBud is a large bud attached to a budding amethyst face.
type LargeAmethystBud struct {
	empty
	transparent
	Face cube.Face
}

// BreakInfo ...
func (b LargeAmethystBud) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(b))
}

// EncodeItem ...
func (LargeAmethystBud) EncodeItem() (string, int16) {
	return "minecraft:large_amethyst_bud", 0
}

// EncodeBlock ...
func (b LargeAmethystBud) EncodeBlock() (string, map[string]any) {
	return "minecraft:large_amethyst_bud", map[string]any{"minecraft:block_face": b.Face.String()}
}

// AmethystCluster is the full geode crystal cluster attached to a budding amethyst face.
type AmethystCluster struct {
	empty
	transparent
	Face cube.Face
}

// BreakInfo ...
func (b AmethystCluster) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(b))
}

// EncodeItem ...
func (AmethystCluster) EncodeItem() (string, int16) {
	return "minecraft:amethyst_cluster", 0
}

// EncodeBlock ...
func (b AmethystCluster) EncodeBlock() (string, map[string]any) {
	return "minecraft:amethyst_cluster", map[string]any{"minecraft:block_face": b.Face.String()}
}

func allSmallAmethystBud() (out []world.Block) {
	for _, face := range cube.Faces() {
		out = append(out, SmallAmethystBud{Face: face})
	}
	return
}

func allMediumAmethystBud() (out []world.Block) {
	for _, face := range cube.Faces() {
		out = append(out, MediumAmethystBud{Face: face})
	}
	return
}

func allLargeAmethystBud() (out []world.Block) {
	for _, face := range cube.Faces() {
		out = append(out, LargeAmethystBud{Face: face})
	}
	return
}

func allAmethystCluster() (out []world.Block) {
	for _, face := range cube.Faces() {
		out = append(out, AmethystCluster{Face: face})
	}
	return
}
