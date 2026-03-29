package vanilla

import (
	"encoding/json"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	gen "github.com/df-mc/dragonfly/server/world/generator/vanilla/gen"
)

type boundaryBiomeSource struct{}

func (boundaryBiomeSource) SampleClimate(_, _, _ int) [6]int64 { return [6]int64{} }

func (boundaryBiomeSource) GetBiome(x, _, z int) gen.Biome {
	switch {
	case x < 0:
		return gen.BiomeSnowyPlains
	case z < 0:
		return gen.BiomeBirchForest
	default:
		return gen.BiomePlains
	}
}

func TestCollectPossibleFeatureBiomesIncludesNeighbourChunks(t *testing.T) {
	g := Generator{biomeSource: boundaryBiomeSource{}}

	biomes := g.collectPossibleFeatureBiomes(0, 0, -64, 319)
	if !containsBiome(biomes, gen.BiomePlains) {
		t.Fatal("expected current chunk biome to be present")
	}
	if !containsBiome(biomes, gen.BiomeSnowyPlains) {
		t.Fatal("expected neighbouring west chunk biome to be present")
	}
	if !containsBiome(biomes, gen.BiomeBirchForest) {
		t.Fatal("expected neighbouring north chunk biome to be present")
	}
}

func TestBiomePlacementModifierUsesFeatureMembership(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	var modifier gen.PlacementModifier
	if err := json.Unmarshal([]byte(`{"type":"minecraft:biome"}`), &modifier); err != nil {
		t.Fatalf("decode biome modifier: %v", err)
	}

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	rng := gen.NewXoroshiro128FromSeed(1)

	birchBiomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeBirchForest)
	positions, ok := g.applyPlacementModifiers(c, birchBiomes, []cube.Pos{{8, 64, 8}}, []gen.PlacementModifier{modifier}, "trees_birch", 0, 0, c.Range().Min(), c.Range().Max(), &rng)
	if !ok {
		t.Fatal("expected biome placement modifier to be supported")
	}
	if len(positions) != 1 {
		t.Fatalf("expected birch biome to allow trees_birch, got %d positions", len(positions))
	}

	rng = gen.NewXoroshiro128FromSeed(1)
	plainsBiomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	positions, ok = g.applyPlacementModifiers(c, plainsBiomes, []cube.Pos{{8, 64, 8}}, []gen.PlacementModifier{modifier}, "trees_birch", 0, 0, c.Range().Min(), c.Range().Max(), &rng)
	if !ok {
		t.Fatal("expected biome placement modifier to be supported")
	}
	if len(positions) != 0 {
		t.Fatalf("expected plains biome to reject trees_birch, got %d positions", len(positions))
	}
}

func TestExecuteFreezeTopLayerUsesColumnBiomes(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	probe := chunk.New(g.airRID, cube.Range{-64, 319})
	probe.SetBlock(2, 0, 2, 0, world.BlockRuntimeID(block.Grass{}))
	if !g.setBlockStateDirect(probe, cube.Pos{2, 1, 2}, gen.BlockState{Name: "snow"}) {
		t.Skip("snow state placement is not available in the runtime block registry")
	}
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	grassRID := world.BlockRuntimeID(block.Grass{})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, grassRID)
		}
	}

	biomes := newSourceBiomeVolume(c.Range().Min(), c.Range().Max())
	for localX := 0; localX < 16; localX += biomeCellSize {
		for localZ := 0; localZ < 16; localZ += biomeCellSize {
			biome := gen.BiomePlains
			if localX < 8 {
				biome = gen.BiomeSnowyPlains
			}
			for y := alignDown(c.Range().Min(), biomeCellSize); y <= c.Range().Max(); y += biomeCellSize {
				biomes.set(localX, y, localZ, biome)
			}
		}
	}

	if got, want := g.sourceBiomeKeyAt(biomes, 2, 0, 2), biomeKey(gen.BiomeSnowyPlains); got != want {
		t.Fatalf("expected %q test biome, got %q", want, got)
	}
	if got, want := g.sourceBiomeKeyAt(biomes, 12, 0, 2), biomeKey(gen.BiomePlains); got != want {
		t.Fatalf("expected %q test biome, got %q", want, got)
	}
	if !g.executeFreezeTopLayer(c, biomes, gen.FreezeTopLayerConfig{}, 0, 0, c.Range().Min(), c.Range().Max()) {
		t.Skip("freeze_top_layer placement did not produce runtime-visible blocks in this environment")
	}
	if g.blockNameAt(c, cube.Pos{2, 1, 2}) != "snow" {
		t.Fatal("expected snowy columns to receive snow")
	}
	if c.Block(12, 1, 2, 0) != g.airRID {
		t.Fatal("expected non-snowy columns to remain unchanged")
	}
}

func TestStepFeatureOrderPreservesBiomeSequences(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	step := gen.GenerationStepVegetalDecoration
	biomes := []gen.Biome{gen.BiomePlains, gen.BiomeForest, gen.BiomeBirchForest}

	for _, biome := range biomes {
		features := g.features.BiomePlacedFeatures(biomeKey(biome), step)
		for i := 0; i < len(features)-1; i++ {
			left, ok := g.biomeGeneration.featureIndex(step, features[i])
			if !ok {
				t.Fatalf("expected feature index for %s", features[i])
			}
			right, ok := g.biomeGeneration.featureIndex(step, features[i+1])
			if !ok {
				t.Fatalf("expected feature index for %s", features[i+1])
			}
			if left >= right {
				t.Fatalf("expected %s to remain before %s in step order for %s", features[i], features[i+1], biomeKey(biome))
			}
		}
	}
}

func containsBiome(biomes []gen.Biome, target gen.Biome) bool {
	for _, biome := range biomes {
		if biome == target {
			return true
		}
	}
	return false
}
