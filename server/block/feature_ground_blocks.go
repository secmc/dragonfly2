package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// MossBlock is a naturally generated block found in lush caves.
type MossBlock struct {
	solid
}

// SoilFor ...
func (MossBlock) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, Flower, DoubleFlower, NetherSprouts, PinkPetals, DeadBush, Azalea, MangrovePropagule, Bamboo:
		return true
	}
	return false
}

// BreakInfo ...
func (m MossBlock) BreakInfo() BreakInfo {
	return newBreakInfo(0.1, alwaysHarvestable, hoeEffective, oneOf(m))
}

// EncodeItem ...
func (MossBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:moss_block", 0
}

// EncodeBlock ...
func (MossBlock) EncodeBlock() (string, map[string]any) {
	return "minecraft:moss_block", nil
}

// RootedDirt is a dirt variant with hanging roots underneath.
type RootedDirt struct {
	solid
}

// SoilFor ...
func (r RootedDirt) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, Flower, DoubleFlower, NetherSprouts, PinkPetals, DeadBush, SugarCane, Azalea, Bamboo:
		return true
	}
	return false
}

// BreakInfo ...
func (r RootedDirt) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, shovelEffective, oneOf(r))
}

// EncodeItem ...
func (RootedDirt) EncodeItem() (name string, meta int16) {
	return "minecraft:rooted_dirt", 0
}

// EncodeBlock ...
func (RootedDirt) EncodeBlock() (string, map[string]any) {
	return "minecraft:dirt_with_roots", nil
}

// PaleMossBlock is a pale oak forest ground-cover block.
type PaleMossBlock struct {
	solid
}

// SoilFor ...
func (PaleMossBlock) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, Flower, DoubleFlower, NetherSprouts, PinkPetals, DeadBush, Azalea, MangrovePropagule, Bamboo:
		return true
	}
	return false
}

// BreakInfo ...
func (p PaleMossBlock) BreakInfo() BreakInfo {
	return newBreakInfo(0.1, alwaysHarvestable, hoeEffective, oneOf(p))
}

// EncodeItem ...
func (PaleMossBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:pale_moss_block", 0
}

// EncodeBlock ...
func (PaleMossBlock) EncodeBlock() (string, map[string]any) {
	return "minecraft:pale_moss_block", nil
}

// HangingRoots are ceiling-attached decorative roots.
type HangingRoots struct {
	empty
	transparent
	replaceable
	sourceWaterDisplacer
}

// HasLiquidDrops ...
func (HangingRoots) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (h HangingRoots) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !supportsFeatureHangingBlock(pos, tx) {
		breakBlock(h, pos, tx)
	}
}

// UseOnBlock ...
func (h HangingRoots) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, h)
	if !used || !supportsFeatureHangingBlock(pos, tx) {
		return false
	}

	place(tx, pos, h, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (h HangingRoots) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(h))
}

// EncodeItem ...
func (HangingRoots) EncodeItem() (name string, meta int16) {
	return "minecraft:hanging_roots", 0
}

// EncodeBlock ...
func (HangingRoots) EncodeBlock() (string, map[string]any) {
	return "minecraft:hanging_roots", nil
}

// MangroveRoots are decorative full-block roots generated beneath mangroves.
type MangroveRoots struct {
	solid
	sourceWaterDisplacer
}

// SoilFor ...
func (MangroveRoots) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, Flower, DoubleFlower, NetherSprouts, PinkPetals:
		return true
	}
	return false
}

// BreakInfo ...
func (m MangroveRoots) BreakInfo() BreakInfo {
	return newBreakInfo(0.7, alwaysHarvestable, axeEffective, oneOf(m))
}

// EncodeItem ...
func (MangroveRoots) EncodeItem() (name string, meta int16) {
	return "minecraft:mangrove_roots", 0
}

// EncodeBlock ...
func (MangroveRoots) EncodeBlock() (string, map[string]any) {
	return "minecraft:mangrove_roots", nil
}

// LeafLitter is a thin decorative carpet of leaves.
type LeafLitter struct {
	carpet
	transparent
	replaceable
	sourceWaterDisplacer

	// Growth is the Bedrock growth value. Values 0-3 correspond to segment counts 1-4 used by worldgen.
	Growth int
	// Facing is the horizontal direction the leaf litter faces.
	Facing cube.Direction
}

// SideClosed ...
func (LeafLitter) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// NeighbourUpdateTick ...
func (l LeafLitter) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(Air); ok {
		breakBlock(l, pos, tx)
	}
}

// HasLiquidDrops ...
func (LeafLitter) HasLiquidDrops() bool {
	return true
}

// UseOnBlock ...
func (l LeafLitter) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, l)
	if !used {
		return false
	}
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(Air); ok {
		return false
	}

	l.Facing = user.Rotation().Direction().Opposite()
	place(tx, pos, l, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (l LeafLitter) BreakInfo() BreakInfo {
	return newBreakInfo(0.1, alwaysHarvestable, nothingEffective, oneOf(l))
}

// EncodeItem ...
func (LeafLitter) EncodeItem() (name string, meta int16) {
	return "minecraft:leaf_litter", 0
}

// EncodeBlock ...
func (l LeafLitter) EncodeBlock() (string, map[string]any) {
	return "minecraft:leaf_litter", map[string]any{
		"growth":                       int32(l.Growth),
		"minecraft:cardinal_direction": l.Facing.String(),
	}
}

// allLeafLitter returns all possible leaf litter states.
func allLeafLitter() (b []world.Block) {
	for growth := 0; growth < 8; growth++ {
		for _, facing := range cube.Directions() {
			b = append(b, LeafLitter{Growth: growth, Facing: facing})
		}
	}
	return
}

// PaleMossCarpetSide is one of the side shapes used by pale moss carpet.
type PaleMossCarpetSide struct {
	paleMossCarpetSide
}

// PaleMossCarpetNone returns a side with no extra edge growth.
func PaleMossCarpetNone() PaleMossCarpetSide {
	return PaleMossCarpetSide{0}
}

// PaleMossCarpetShort returns a short side edge.
func PaleMossCarpetShort() PaleMossCarpetSide {
	return PaleMossCarpetSide{1}
}

// PaleMossCarpetTall returns a tall side edge.
func PaleMossCarpetTall() PaleMossCarpetSide {
	return PaleMossCarpetSide{2}
}

// PaleMossCarpetSides returns all pale moss carpet side variants.
func PaleMossCarpetSides() []PaleMossCarpetSide {
	return []PaleMossCarpetSide{PaleMossCarpetNone(), PaleMossCarpetShort(), PaleMossCarpetTall()}
}

type paleMossCarpetSide uint8

// Uint8 ...
func (s paleMossCarpetSide) Uint8() uint8 {
	return uint8(s)
}

// String ...
func (s paleMossCarpetSide) String() string {
	switch s {
	case 0:
		return "none"
	case 1:
		return "short"
	case 2:
		return "tall"
	}
	panic("unknown pale moss carpet side")
}

// PaleMossCarpet is a carpet-style block used in pale oak forests.
type PaleMossCarpet struct {
	carpet
	transparent
	replaceable
	sourceWaterDisplacer

	Upper bool
	North PaleMossCarpetSide
	East  PaleMossCarpetSide
	South PaleMossCarpetSide
	West  PaleMossCarpetSide
}

// SideClosed ...
func (PaleMossCarpet) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// HasLiquidDrops ...
func (PaleMossCarpet) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (p PaleMossCarpet) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if p.Upper {
		if below, ok := tx.Block(pos.Side(cube.FaceDown)).(PaleMossCarpet); !ok || below.Upper {
			breakBlockNoDrops(p, pos, tx)
		}
		return
	}
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(Air); ok {
		breakBlock(p, pos, tx)
	}
}

// UseOnBlock ...
func (p PaleMossCarpet) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, p)
	if !used {
		return false
	}
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(Air); ok {
		return false
	}

	place(tx, pos, p, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (p PaleMossCarpet) BreakInfo() BreakInfo {
	return newBreakInfo(0.1, alwaysHarvestable, nothingEffective, oneOf(p))
}

// EncodeItem ...
func (PaleMossCarpet) EncodeItem() (name string, meta int16) {
	return "minecraft:pale_moss_carpet", 0
}

// EncodeBlock ...
func (p PaleMossCarpet) EncodeBlock() (string, map[string]any) {
	return "minecraft:pale_moss_carpet", map[string]any{
		"pale_moss_carpet_side_east":  p.East.String(),
		"pale_moss_carpet_side_north": p.North.String(),
		"pale_moss_carpet_side_south": p.South.String(),
		"pale_moss_carpet_side_west":  p.West.String(),
		"upper_block_bit":             p.Upper,
	}
}

// allPaleMossCarpet returns all pale moss carpet states.
func allPaleMossCarpet() (b []world.Block) {
	sides := PaleMossCarpetSides()
	for _, north := range sides {
		for _, east := range sides {
			for _, south := range sides {
				for _, west := range sides {
					b = append(b,
						PaleMossCarpet{North: north, East: east, South: south, West: west},
						PaleMossCarpet{Upper: true, North: north, East: east, South: south, West: west},
					)
				}
			}
		}
	}
	return
}

// PaleHangingMoss is the hanging decorative pale moss plant.
type PaleHangingMoss struct {
	empty
	transparent
	replaceable
	sourceWaterDisplacer

	Tip bool
}

// HasLiquidDrops ...
func (PaleHangingMoss) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (p PaleHangingMoss) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !supportsFeaturePaleHangingMoss(pos, tx) {
		breakBlock(p, pos, tx)
		return
	}

	tip := true
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(PaleHangingMoss); ok {
		tip = false
	}
	if p.Tip != tip {
		p.Tip = tip
		tx.SetBlock(pos, p, nil)
	}
}

// UseOnBlock ...
func (p PaleHangingMoss) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, p)
	if !used || !supportsFeaturePaleHangingMoss(pos, tx) {
		return false
	}

	if abovePos := pos.Side(cube.FaceUp); !abovePos.OutOfBounds(tx.Range()) {
		if above, ok := tx.Block(abovePos).(PaleHangingMoss); ok && above.Tip {
			above.Tip = false
			tx.SetBlock(abovePos, above, nil)
		}
	}
	p.Tip = true
	place(tx, pos, p, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (p PaleHangingMoss) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(p))
}

// EncodeItem ...
func (PaleHangingMoss) EncodeItem() (name string, meta int16) {
	return "minecraft:pale_hanging_moss", 0
}

// EncodeBlock ...
func (p PaleHangingMoss) EncodeBlock() (string, map[string]any) {
	return "minecraft:pale_hanging_moss", map[string]any{"tip": p.Tip}
}

// allPaleHangingMoss returns all pale hanging moss states.
func allPaleHangingMoss() []world.Block {
	return []world.Block{
		PaleHangingMoss{},
		PaleHangingMoss{Tip: true},
	}
}

func supportsFeatureHangingBlock(pos cube.Pos, tx *world.Tx) bool {
	above := pos.Side(cube.FaceUp)
	return tx.Block(above).Model().FaceSolid(above, cube.FaceDown, tx)
}

func supportsFeaturePaleHangingMoss(pos cube.Pos, tx *world.Tx) bool {
	above := pos.Side(cube.FaceUp)
	switch tx.Block(above).(type) {
	case PaleHangingMoss, Leaves:
		return true
	default:
		return tx.Block(above).Model().FaceSolid(above, cube.FaceDown, tx)
	}
}
