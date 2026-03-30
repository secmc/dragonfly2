package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"math/rand/v2"
)

// BambooLeafSize represents the leaf state of a bamboo stalk.
type BambooLeafSize struct {
	bambooLeafSize
}

// BambooNoLeaves returns bamboo with no leaves.
func BambooNoLeaves() BambooLeafSize {
	return BambooLeafSize{0}
}

// BambooSmallLeaves returns bamboo with small leaves.
func BambooSmallLeaves() BambooLeafSize {
	return BambooLeafSize{1}
}

// BambooLargeLeaves returns bamboo with large leaves.
func BambooLargeLeaves() BambooLeafSize {
	return BambooLeafSize{2}
}

// BambooLeafSizes returns all bamboo leaf sizes.
func BambooLeafSizes() []BambooLeafSize {
	return []BambooLeafSize{BambooNoLeaves(), BambooSmallLeaves(), BambooLargeLeaves()}
}

type bambooLeafSize uint8

// Uint8 ...
func (b bambooLeafSize) Uint8() uint8 {
	return uint8(b)
}

// String ...
func (b bambooLeafSize) String() string {
	switch b {
	case 0:
		return "no_leaves"
	case 1:
		return "small_leaves"
	case 2:
		return "large_leaves"
	}
	panic("unknown bamboo leaf size")
}

// BambooStalkThickness represents the thickness of a bamboo stalk.
type BambooStalkThickness struct {
	bambooStalkThickness
}

// ThinBamboo returns a thin bamboo stalk.
func ThinBamboo() BambooStalkThickness {
	return BambooStalkThickness{0}
}

// ThickBamboo returns a thick bamboo stalk.
func ThickBamboo() BambooStalkThickness {
	return BambooStalkThickness{1}
}

// BambooStalkThicknesses returns all bamboo thickness states.
func BambooStalkThicknesses() []BambooStalkThickness {
	return []BambooStalkThickness{ThinBamboo(), ThickBamboo()}
}

type bambooStalkThickness uint8

// Uint8 ...
func (b bambooStalkThickness) Uint8() uint8 {
	return uint8(b)
}

// String ...
func (b bambooStalkThickness) String() string {
	if b == 0 {
		return "thin"
	}
	return "thick"
}

// Bamboo is the bamboo stalk block.
type Bamboo struct {
	empty
	transparent
	replaceable
	sourceWaterDisplacer

	AgeBit    bool
	LeafSize  BambooLeafSize
	Thickness BambooStalkThickness
}

// BoneMeal ...
func (b Bamboo) BoneMeal(pos cube.Pos, tx *world.Tx) bool {
	return growFeatureBamboo(pos, tx, 2)
}

// HasLiquidDrops ...
func (Bamboo) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (b Bamboo) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !supportsFeatureBambooBlock(tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(b, pos, tx)
		return
	}
	if above, ok := tx.Block(pos.Side(cube.FaceUp)).(Bamboo); ok && above.AgeBit && !b.AgeBit {
		b.AgeBit = true
		b.Thickness = ThickBamboo()
		tx.SetBlock(pos, b, nil)
	}
}

// RandomTick ...
func (b Bamboo) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	if !supportsFeatureBambooBlock(tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(b, pos, tx)
		return
	}
	growthPos := pos.Side(cube.FaceUp)
	if growthPos.OutOfBounds(tx.Range()) || tx.Light(growthPos) < 9 || r.IntN(3) != 0 {
		return
	}
	_, air := tx.Block(growthPos).(Air)
	if !air {
		return
	}
	_, liquid := tx.Liquid(growthPos)
	if liquid {
		return
	}
	_ = growFeatureBamboo(pos, tx, 1)
}

// UseOnBlock ...
func (b Bamboo) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	if _, ok := tx.Block(pos).(Bamboo); ok {
		pos = pos.Side(cube.FaceUp)
	} else {
		var used bool
		pos, _, used = firstReplaceable(tx, pos, face, b)
		if !used {
			return false
		}
	}
	if pos.OutOfBounds(tx.Range()) || !supportsFeatureBambooBlock(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}
	if _, ok := tx.Block(pos).(Air); !ok {
		return false
	}

	if below, ok := tx.Block(pos.Side(cube.FaceDown)).(Bamboo); ok {
		b.AgeBit = below.AgeBit
		if b.AgeBit {
			b.Thickness = ThickBamboo()
		}
	}
	place(tx, pos, b, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (b Bamboo) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(b))
}

// EncodeItem ...
func (Bamboo) EncodeItem() (name string, meta int16) {
	return "minecraft:bamboo", 0
}

// EncodeBlock ...
func (b Bamboo) EncodeBlock() (string, map[string]any) {
	return "minecraft:bamboo", map[string]any{
		"age_bit":                b.AgeBit,
		"bamboo_leaf_size":       b.LeafSize.String(),
		"bamboo_stalk_thickness": b.Thickness.String(),
	}
}

// allBamboo returns all bamboo states.
func allBamboo() (b []world.Block) {
	for _, leafSize := range BambooLeafSizes() {
		for _, thickness := range BambooStalkThicknesses() {
			b = append(b,
				Bamboo{LeafSize: leafSize, Thickness: thickness},
				Bamboo{AgeBit: true, LeafSize: leafSize, Thickness: thickness},
			)
		}
	}
	return
}

// Azalea is an azalea shrub block. If Flowering is true, the flowering azalea variant is encoded.
type Azalea struct {
	empty
	transparent
	replaceable
	sourceWaterDisplacer

	Flowering bool
}

// NeighbourUpdateTick ...
func (a Azalea) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !supportsFeatureAzaleaBlock(tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(a, pos, tx)
	}
}

// HasLiquidDrops ...
func (Azalea) HasLiquidDrops() bool {
	return true
}

// UseOnBlock ...
func (a Azalea) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, a)
	if !used || !supportsFeatureAzaleaBlock(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}

	place(tx, pos, a, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (a Azalea) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(a))
}

// EncodeItem ...
func (a Azalea) EncodeItem() (name string, meta int16) {
	if a.Flowering {
		return "minecraft:flowering_azalea", 0
	}
	return "minecraft:azalea", 0
}

// EncodeBlock ...
func (a Azalea) EncodeBlock() (string, map[string]any) {
	if a.Flowering {
		return "minecraft:flowering_azalea", nil
	}
	return "minecraft:azalea", nil
}

// allAzalea returns all azalea block states.
func allAzalea() []world.Block {
	return []world.Block{
		Azalea{},
		Azalea{Flowering: true},
	}
}

// MangrovePropagule is the mangrove sapling/propagule block.
type MangrovePropagule struct {
	empty
	transparent
	replaceable
	sourceWaterDisplacer

	Hanging bool
	Stage   int
}

// BoneMeal ...
func (m MangrovePropagule) BoneMeal(pos cube.Pos, tx *world.Tx) bool {
	if m.Hanging && m.Stage < 4 {
		m.Stage++
		tx.SetBlock(pos, m, nil)
		return true
	}
	return false
}

// HasLiquidDrops ...
func (MangrovePropagule) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (m MangrovePropagule) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if m.Hanging {
		if !supportsFeatureHangingMangrovePropagule(tx.Block(pos.Side(cube.FaceUp))) {
			breakBlock(m, pos, tx)
		}
		return
	}
	if !supportsFeatureMangrovePropaguleBlock(tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(m, pos, tx)
	}
}

// RandomTick ...
func (m MangrovePropagule) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	if !m.Hanging || m.Stage >= 4 {
		return
	}
	m.Stage++
	tx.SetBlock(pos, m, nil)
}

// UseOnBlock ...
func (m MangrovePropagule) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, m)
	if !used || !supportsFeatureMangrovePropaguleBlock(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}

	m.Stage = 4
	m.Hanging = false
	place(tx, pos, m, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (m MangrovePropagule) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(m))
}

// EncodeItem ...
func (MangrovePropagule) EncodeItem() (name string, meta int16) {
	return "minecraft:mangrove_propagule", 0
}

// EncodeBlock ...
func (m MangrovePropagule) EncodeBlock() (string, map[string]any) {
	return "minecraft:mangrove_propagule", map[string]any{
		"hanging":         m.Hanging,
		"propagule_stage": int32(m.Stage),
	}
}

// allMangrovePropagule returns all mangrove propagule states.
func allMangrovePropagule() (b []world.Block) {
	for stage := 0; stage <= 4; stage++ {
		b = append(b,
			MangrovePropagule{Stage: stage},
			MangrovePropagule{Hanging: true, Stage: stage},
		)
	}
	return
}

// SmallDripleaf is the small dripleaf double plant block.
type SmallDripleaf struct {
	empty
	transparent
	replaceable
	sourceWaterDisplacer

	Upper  bool
	Facing cube.Direction
}

// BoneMeal ...
func (s SmallDripleaf) BoneMeal(pos cube.Pos, tx *world.Tx) bool {
	if s.Upper {
		return false
	}
	upperPos := pos.Side(cube.FaceUp)
	height := 2
	for height < 5 {
		nextPos := pos.Add(cube.Pos{0, height, 0})
		if nextPos.OutOfBounds(tx.Range()) {
			break
		}
		if _, ok := tx.Block(nextPos).(Air); !ok {
			break
		}
		height++
	}
	tx.SetBlock(upperPos, nil, nil)
	for i := 0; i < height-1; i++ {
		tx.SetBlock(pos.Add(cube.Pos{0, i, 0}), BigDripleaf{Facing: s.Facing}, nil)
	}
	tx.SetBlock(pos.Add(cube.Pos{0, height - 1, 0}), BigDripleaf{Head: true, Facing: s.Facing}, nil)
	return true
}

// HasLiquidDrops ...
func (SmallDripleaf) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (s SmallDripleaf) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if s.Upper {
		if lower, ok := tx.Block(pos.Side(cube.FaceDown)).(SmallDripleaf); !ok || lower.Upper {
			breakBlockNoDrops(s, pos, tx)
		}
		return
	}
	if upper, ok := tx.Block(pos.Side(cube.FaceUp)).(SmallDripleaf); !ok || !upper.Upper {
		breakBlockNoDrops(s, pos, tx)
		return
	}
	if !supportsFeatureSmallDripleafBlock(tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(s, pos, tx)
	}
}

// UseOnBlock ...
func (s SmallDripleaf) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, s)
	if !used || !replaceableWith(tx, pos.Side(cube.FaceUp), SmallDripleaf{Upper: true}) || !supportsFeatureSmallDripleafBlock(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}

	s.Facing = user.Rotation().Direction().Opposite()
	place(tx, pos, s, user, ctx)
	place(tx, pos.Side(cube.FaceUp), SmallDripleaf{Upper: true, Facing: s.Facing}, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (s SmallDripleaf) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(s))
}

// EncodeItem ...
func (SmallDripleaf) EncodeItem() (name string, meta int16) {
	return "minecraft:small_dripleaf", 0
}

// EncodeBlock ...
func (s SmallDripleaf) EncodeBlock() (string, map[string]any) {
	return "minecraft:small_dripleaf_block", map[string]any{
		"minecraft:cardinal_direction": s.Facing.String(),
		"upper_block_bit":              s.Upper,
	}
}

// allSmallDripleaf returns all small dripleaf states.
func allSmallDripleaf() (b []world.Block) {
	for _, facing := range cube.Directions() {
		b = append(b,
			SmallDripleaf{Facing: facing},
			SmallDripleaf{Upper: true, Facing: facing},
		)
	}
	return
}

// DripleafTilt is the tilt state used by big dripleaf.
type DripleafTilt struct {
	dripleafTilt
}

// DripleafTiltNone returns the resting big dripleaf tilt.
func DripleafTiltNone() DripleafTilt {
	return DripleafTilt{0}
}

// DripleafTiltUnstable returns the unstable big dripleaf tilt.
func DripleafTiltUnstable() DripleafTilt {
	return DripleafTilt{1}
}

// DripleafTiltPartial returns the partially tilted big dripleaf state.
func DripleafTiltPartial() DripleafTilt {
	return DripleafTilt{2}
}

// DripleafTiltFull returns the fully tilted big dripleaf state.
func DripleafTiltFull() DripleafTilt {
	return DripleafTilt{3}
}

// DripleafTilts returns all big dripleaf tilt states.
func DripleafTilts() []DripleafTilt {
	return []DripleafTilt{DripleafTiltNone(), DripleafTiltUnstable(), DripleafTiltPartial(), DripleafTiltFull()}
}

type dripleafTilt uint8

// Uint8 ...
func (d dripleafTilt) Uint8() uint8 {
	return uint8(d)
}

// String ...
func (d dripleafTilt) String() string {
	switch d {
	case 0:
		return "none"
	case 1:
		return "unstable"
	case 2:
		return "partial_tilt"
	case 3:
		return "full_tilt"
	}
	panic("unknown dripleaf tilt")
}

// BigDripleaf represents both the stem and head Bedrock states for big dripleaf.
type BigDripleaf struct {
	empty
	transparent
	sourceWaterDisplacer

	Head   bool
	Tilt   DripleafTilt
	Facing cube.Direction
}

// BoneMeal ...
func (b BigDripleaf) BoneMeal(pos cube.Pos, tx *world.Tx) bool {
	topPos, top, ok := topBigDripleafPos(pos, tx)
	if !ok {
		return false
	}
	newHeadPos := topPos.Side(cube.FaceUp)
	if newHeadPos.OutOfBounds(tx.Range()) || !replaceableWith(tx, newHeadPos, BigDripleaf{Head: true, Facing: top.Facing}) {
		return false
	}

	tx.SetBlock(topPos, BigDripleaf{Facing: top.Facing}, nil)
	tx.SetBlock(newHeadPos, BigDripleaf{Head: true, Facing: top.Facing}, nil)
	return true
}

// HasLiquidDrops ...
func (BigDripleaf) HasLiquidDrops() bool {
	return true
}

// NeighbourUpdateTick ...
func (b BigDripleaf) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	below := tx.Block(pos.Side(cube.FaceDown))
	if b.Head {
		if _, ok := tx.Block(pos.Side(cube.FaceUp)).(BigDripleaf); ok {
			b.Head = false
			tx.SetBlock(pos, b, nil)
			return
		}
		if _, ok := below.(BigDripleaf); ok || supportsFeatureBigDripleafGround(below) {
			return
		}
		breakBlock(b, pos, tx)
		return
	}
	above, ok := tx.Block(pos.Side(cube.FaceUp)).(BigDripleaf)
	if !ok {
		breakBlockNoDrops(b, pos, tx)
		return
	}
	if !above.Head && !supportsFeatureBigDripleafGround(below) && !isFeatureBigDripleaf(below) {
		breakBlock(b, pos, tx)
		return
	}
	if !isFeatureBigDripleaf(below) && !supportsFeatureBigDripleafGround(below) {
		breakBlock(b, pos, tx)
	}
}

// UseOnBlock ...
func (b BigDripleaf) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, b)
	if !used || !supportsFeatureBigDripleafGround(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}

	b.Head = true
	b.Tilt = DripleafTiltNone()
	b.Facing = user.Rotation().Direction().Opposite()
	place(tx, pos, b, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (b BigDripleaf) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(b))
}

// EncodeItem ...
func (BigDripleaf) EncodeItem() (name string, meta int16) {
	return "minecraft:big_dripleaf", 0
}

// EncodeBlock ...
func (b BigDripleaf) EncodeBlock() (string, map[string]any) {
	return "minecraft:big_dripleaf", map[string]any{
		"big_dripleaf_head":            b.Head,
		"big_dripleaf_tilt":            b.Tilt.String(),
		"minecraft:cardinal_direction": b.Facing.String(),
	}
}

// allBigDripleaf returns all big dripleaf Bedrock states.
func allBigDripleaf() (b []world.Block) {
	for _, facing := range cube.Directions() {
		for _, tilt := range DripleafTilts() {
			b = append(b,
				BigDripleaf{Facing: facing, Tilt: tilt},
				BigDripleaf{Head: true, Facing: facing, Tilt: tilt},
			)
		}
	}
	return
}

func supportsFeatureSubstrateOverworld(b world.Block) bool {
	switch b.(type) {
	case Dirt, Grass, Podzol, Mud, MuddyMangroveRoots, RootedDirt, MossBlock, PaleMossBlock:
		return true
	default:
		return false
	}
}

func supportsFeatureVegetationBlock(b world.Block) bool {
	if supportsFeatureSubstrateOverworld(b) {
		return true
	}
	_, ok := b.(Farmland)
	return ok
}

func supportsFeatureAzaleaBlock(b world.Block) bool {
	if supportsFeatureVegetationBlock(b) {
		return true
	}
	_, ok := b.(Clay)
	return ok
}

func supportsFeatureMangrovePropaguleBlock(b world.Block) bool {
	return supportsFeatureAzaleaBlock(b)
}

func supportsFeatureHangingMangrovePropagule(b world.Block) bool {
	leaves, ok := b.(Leaves)
	return ok && leaves.Type == MangroveLeaves()
}

func supportsFeatureBambooBlock(b world.Block) bool {
	if supportsFeatureSubstrateOverworld(b) {
		return true
	}
	switch b.(type) {
	case Bamboo, Sand, Gravel:
		return true
	default:
		return false
	}
}

func supportsFeatureSmallDripleafBlock(b world.Block) bool {
	switch b.(type) {
	case Clay, MossBlock:
		return true
	default:
		return false
	}
}

func supportsFeatureBigDripleafGround(b world.Block) bool {
	if supportsFeatureSmallDripleafBlock(b) {
		return true
	}
	switch b.(type) {
	case Dirt, Grass, Podzol, RootedDirt, Mud, MuddyMangroveRoots, Farmland:
		return true
	default:
		return false
	}
}

func isFeatureBigDripleaf(b world.Block) bool {
	_, ok := b.(BigDripleaf)
	return ok
}

func growFeatureBamboo(pos cube.Pos, tx *world.Tx, growth int) bool {
	topPos, top, ok := topFeatureBamboo(pos, tx)
	if !ok {
		return false
	}
	heightBelow := featureBambooHeightBelow(pos, tx)
	heightAbove := featureBambooHeightAbove(pos, tx)
	totalHeight := heightBelow + heightAbove + 1
	grew := false

	for i := 0; i < growth; i++ {
		growthPos := topPos.Side(cube.FaceUp)
		if growthPos.OutOfBounds(tx.Range()) || totalHeight >= 16 {
			return grew
		}
		if _, ok := tx.Block(growthPos).(Air); !ok {
			return grew
		}
		if _, ok := tx.Liquid(growthPos); ok {
			return grew
		}

		below, belowIsBamboo := tx.Block(topPos.Side(cube.FaceDown)).(Bamboo)
		twoBelowPos := topPos.Side(cube.FaceDown).Side(cube.FaceDown)
		twoBelow, twoBelowIsBamboo := tx.Block(twoBelowPos).(Bamboo)
		leaves := BambooNoLeaves()
		if totalHeight >= 1 {
			if !belowIsBamboo || below.LeafSize == BambooNoLeaves() {
				leaves = BambooSmallLeaves()
			} else {
				leaves = BambooLargeLeaves()
				if twoBelowIsBamboo {
					below.LeafSize = BambooSmallLeaves()
					tx.SetBlock(topPos.Side(cube.FaceDown), below, nil)
					twoBelow.LeafSize = BambooNoLeaves()
					tx.SetBlock(twoBelowPos, twoBelow, nil)
				}
			}
		}

		age := top.AgeBit || twoBelowIsBamboo
		thickness := ThinBamboo()
		if age {
			thickness = ThickBamboo()
		}
		topPos = growthPos
		top = Bamboo{AgeBit: age, LeafSize: leaves, Thickness: thickness}
		tx.SetBlock(topPos, top, nil)
		totalHeight++
		grew = true
	}
	return grew
}

func featureBambooHeightAbove(pos cube.Pos, tx *world.Tx) int {
	height := 0
	for height < 16 {
		if _, ok := tx.Block(pos.Add(cube.Pos{0, height + 1, 0})).(Bamboo); !ok {
			break
		}
		height++
	}
	return height
}

func featureBambooHeightBelow(pos cube.Pos, tx *world.Tx) int {
	height := 0
	for height < 16 {
		if _, ok := tx.Block(pos.Add(cube.Pos{0, -(height + 1), 0})).(Bamboo); !ok {
			break
		}
		height++
	}
	return height
}

func topFeatureBamboo(pos cube.Pos, tx *world.Tx) (cube.Pos, Bamboo, bool) {
	currentPos := pos
	current, ok := tx.Block(currentPos).(Bamboo)
	if !ok {
		return cube.Pos{}, Bamboo{}, false
	}
	for {
		nextPos := currentPos.Side(cube.FaceUp)
		next, ok := tx.Block(nextPos).(Bamboo)
		if !ok {
			return currentPos, current, true
		}
		currentPos, current = nextPos, next
	}
}

func topBigDripleafPos(pos cube.Pos, tx *world.Tx) (cube.Pos, BigDripleaf, bool) {
	currentPos := pos
	current, ok := tx.Block(currentPos).(BigDripleaf)
	if !ok {
		return cube.Pos{}, BigDripleaf{}, false
	}
	for {
		nextPos := currentPos.Side(cube.FaceUp)
		next, ok := tx.Block(nextPos).(BigDripleaf)
		if !ok {
			return currentPos, current, true
		}
		currentPos, current = nextPos, next
	}
}
