package block

import "github.com/df-mc/dragonfly/server/world"

// BrownMushroomBlock is the cap block used by huge brown mushrooms.
type BrownMushroomBlock struct {
	solid

	HugeMushroomBits int
}

// BreakInfo ...
func (b BrownMushroomBlock) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, hoeEffective, oneOf(b))
}

// EncodeItem ...
func (BrownMushroomBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:brown_mushroom_block", 0
}

// EncodeBlock ...
func (b BrownMushroomBlock) EncodeBlock() (string, map[string]any) {
	return "minecraft:brown_mushroom_block", map[string]any{"huge_mushroom_bits": int32(b.HugeMushroomBits)}
}

// RedMushroomBlock is the cap block used by huge red mushrooms.
type RedMushroomBlock struct {
	solid

	HugeMushroomBits int
}

// BreakInfo ...
func (r RedMushroomBlock) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, hoeEffective, oneOf(r))
}

// EncodeItem ...
func (RedMushroomBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:red_mushroom_block", 0
}

// EncodeBlock ...
func (r RedMushroomBlock) EncodeBlock() (string, map[string]any) {
	return "minecraft:red_mushroom_block", map[string]any{"huge_mushroom_bits": int32(r.HugeMushroomBits)}
}

// MushroomStem is the stem block used by huge mushrooms.
type MushroomStem struct {
	solid

	HugeMushroomBits int
}

// BreakInfo ...
func (m MushroomStem) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, hoeEffective, oneOf(m))
}

// EncodeItem ...
func (MushroomStem) EncodeItem() (name string, meta int16) {
	return "minecraft:mushroom_stem", 0
}

// EncodeBlock ...
func (m MushroomStem) EncodeBlock() (string, map[string]any) {
	return "minecraft:mushroom_stem", map[string]any{"huge_mushroom_bits": int32(m.HugeMushroomBits)}
}

func allBrownMushroomBlock() (b []world.Block) {
	for bits := 0; bits <= 15; bits++ {
		b = append(b, BrownMushroomBlock{HugeMushroomBits: bits})
	}
	return
}

func allRedMushroomBlock() (b []world.Block) {
	for bits := 0; bits <= 15; bits++ {
		b = append(b, RedMushroomBlock{HugeMushroomBits: bits})
	}
	return
}

func allMushroomStem() (b []world.Block) {
	for bits := 0; bits <= 15; bits++ {
		b = append(b, MushroomStem{HugeMushroomBits: bits})
	}
	return
}
