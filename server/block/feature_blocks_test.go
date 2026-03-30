package block

import (
	"sync"
	"testing"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

var finaliseFeatureBlocksOnce sync.Once

//go:linkname worldFinaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func worldFinaliseBlockRegistry()

func withFeatureBlockTx(t *testing.T, fn func(tx *world.Tx)) {
	t.Helper()

	finaliseFeatureBlocksOnce.Do(worldFinaliseBlockRegistry)
	w := world.Config{
		Generator:       world.NopGenerator{},
		RandomTickSpeed: -1,
	}.New()
	t.Cleanup(func() {
		_ = w.Close()
	})

	<-w.Exec(func(tx *world.Tx) {
		fn(tx)
	})
}

func TestSmallDripleafBoneMealGrowsBigDripleaf(t *testing.T) {
	withFeatureBlockTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 1, 0}
		tx.SetBlock(pos.Side(cube.FaceDown), MossBlock{}, nil)
		tx.SetBlock(pos, SmallDripleaf{Facing: cube.North}, nil)
		tx.SetBlock(pos.Side(cube.FaceUp), SmallDripleaf{Upper: true, Facing: cube.North}, nil)

		if !(SmallDripleaf{Facing: cube.North}).BoneMeal(pos, tx) {
			t.Fatal("expected small dripleaf bonemeal to grow")
		}
		if got, ok := tx.Block(pos).(BigDripleaf); !ok || got.Head {
			t.Fatalf("expected lower block to become a big dripleaf stem, got %#v", tx.Block(pos))
		}

		headCount := 0
		for y := 1; y <= 4; y++ {
			got, ok := tx.Block(pos.Add(cube.Pos{0, y, 0})).(BigDripleaf)
			if !ok {
				break
			}
			if got.Head {
				headCount++
			}
		}
		if headCount != 1 {
			t.Fatalf("expected exactly one big dripleaf head after bonemeal, got %d", headCount)
		}
	})
}

func TestMangrovePropaguleBoneMealAdvancesHangingStage(t *testing.T) {
	withFeatureBlockTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 1, 0}
		tx.SetBlock(pos.Side(cube.FaceUp), Leaves{Type: MangroveLeaves()}, nil)
		tx.SetBlock(pos, MangrovePropagule{Hanging: true}, nil)

		if !(MangrovePropagule{Hanging: true}).BoneMeal(pos, tx) {
			t.Fatal("expected hanging mangrove propagule bonemeal to advance age")
		}
		got, ok := tx.Block(pos).(MangrovePropagule)
		if !ok || got.Stage != 1 {
			t.Fatalf("expected hanging propagule stage 1 after bonemeal, got %#v", tx.Block(pos))
		}
	})
}

func TestPaleMossCarpetNeighbourUpdateBreaksWithoutSupport(t *testing.T) {
	withFeatureBlockTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 1, 0}
		tx.SetBlock(pos, PaleMossCarpet{Upper: true}, nil)

		PaleMossCarpet{Upper: true}.NeighbourUpdateTick(pos, pos.Side(cube.FaceDown), tx)
		if _, ok := tx.Block(pos).(Air); !ok {
			t.Fatalf("expected unsupported pale moss carpet to break, got %#v", tx.Block(pos))
		}
	})
}

func TestPaleHangingMossNeighbourUpdateStaysAttachedToLeavesAndChains(t *testing.T) {
	withFeatureBlockTx(t, func(tx *world.Tx) {
		top := cube.Pos{0, 1, 0}
		bottom := cube.Pos{0, 0, 0}

		tx.SetBlock(top.Side(cube.FaceUp), Leaves{Type: PaleOakLeaves()}, nil)
		tx.SetBlock(top, PaleHangingMoss{Tip: true}, nil)
		tx.SetBlock(bottom, PaleHangingMoss{Tip: true}, nil)

		PaleHangingMoss{Tip: true}.NeighbourUpdateTick(top, top.Side(cube.FaceUp), tx)
		PaleHangingMoss{Tip: true}.NeighbourUpdateTick(bottom, top, tx)

		gotTop, ok := tx.Block(top).(PaleHangingMoss)
		if !ok {
			t.Fatalf("expected upper pale hanging moss to remain, got %#v", tx.Block(top))
		}
		if gotTop.Tip {
			t.Fatalf("expected upper pale hanging moss to become a non-tip chain segment, got %#v", gotTop)
		}
		gotBottom, ok := tx.Block(bottom).(PaleHangingMoss)
		if !ok {
			t.Fatalf("expected lower pale hanging moss to remain, got %#v", tx.Block(bottom))
		}
		if !gotBottom.Tip {
			t.Fatalf("expected lower pale hanging moss to remain the tip, got %#v", gotBottom)
		}
	})
}
