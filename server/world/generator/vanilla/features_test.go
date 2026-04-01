package vanilla

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	gen "github.com/df-mc/dragonfly/server/world/generator/vanilla/gen"
)

func TestGenerateChunkDecoratesVegetation(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	positions := []world.ChunkPos{
		{0, 0},
		{16, 16},
		{32, 0},
		{64, 32},
		{-48, 16},
	}

	totalDecor := 0
	for _, pos := range positions {
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		g.GenerateChunk(pos, c)
		totalDecor += countDecorativeBlocks(c)
	}

	if totalDecor == 0 {
		t.Fatal("expected generated sample chunks to contain vegetation or simple placed features")
	}
}

func TestGenerateChunkPlacesUndergroundFeatures(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	positions := []world.ChunkPos{
		{0, 0},
		{16, 16},
		{32, 0},
		{64, 32},
		{-48, 16},
	}

	totalOres := 0
	for _, pos := range positions {
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		g.GenerateChunk(pos, c)
		totalOres += countOreBlocks(c)
	}

	if totalOres == 0 {
		t.Fatal("expected generated sample chunks to contain underground ore features")
	}
}

func TestExecuteSpringFeaturePlacesSource(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	y := 32
	center := cube.Pos{8, y, 8}
	stoneRID := world.BlockRuntimeID(block.Stone{})

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for dy := y - 2; dy <= y+2; dy++ {
				c.SetBlock(uint8(x), int16(dy), uint8(z), 0, stoneRID)
			}
		}
	}
	c.SetBlock(uint8(center[0]), int16(center[1]), uint8(center[2]), 0, g.airRID)
	c.SetBlock(uint8(center[0]+1), int16(center[1]), uint8(center[2]), 0, g.airRID)

	cfg, err := g.features.Configured("spring_water")
	if err != nil {
		t.Fatalf("failed to load spring_water: %v", err)
	}
	spring, err := cfg.SpringFeature()
	if err != nil {
		t.Fatalf("failed to decode spring_water: %v", err)
	}
	if !g.executeSpringFeature(c, center, spring, 0, 0, c.Range().Min(), c.Range().Max(), nil) {
		t.Fatal("expected spring feature to place a fluid source")
	}

	b, ok := world.BlockByRuntimeID(c.Block(uint8(center[0]), int16(center[1]), uint8(center[2]), 0))
	if !ok {
		t.Fatal("expected spring center block to exist")
	}
	if _, ok := b.(block.Water); !ok {
		t.Fatalf("expected spring to place water, got %T", b)
	}
}

func TestFeatureBlockFromStateNormalizesTreeStates(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	logBlock, ok := g.featureBlockFromState(gen.BlockState{
		Name: "oak_log",
		Properties: map[string]string{
			"axis": "y",
		},
	}, nil)
	if !ok {
		t.Fatal("expected oak_log state to resolve")
	}
	if _, ok := logBlock.(block.Log); !ok {
		t.Fatalf("expected oak_log to resolve to block.Log, got %T", logBlock)
	}

	leafBlock, ok := g.featureBlockFromState(gen.BlockState{
		Name: "oak_leaves",
		Properties: map[string]string{
			"distance":    "7",
			"persistent":  "false",
			"waterlogged": "false",
		},
	}, nil)
	if !ok {
		t.Fatal("expected oak_leaves state to resolve")
	}
	if _, ok := leafBlock.(block.Leaves); !ok {
		t.Fatalf("expected oak_leaves to resolve to block.Leaves, got %T", leafBlock)
	}
}

func TestFeatureBlockFromStateResolvesImplementedFeatureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	tests := []struct {
		name       string
		state      gen.BlockState
		expectType any
	}{
		{
			name:       "bamboo",
			state:      gen.BlockState{Name: "minecraft:bamboo"},
			expectType: block.Bamboo{},
		},
		{
			name:       "rooted_dirt",
			state:      gen.BlockState{Name: "minecraft:rooted_dirt"},
			expectType: block.RootedDirt{},
		},
		{
			name: "leaf_litter",
			state: gen.BlockState{
				Name: "minecraft:leaf_litter",
				Properties: map[string]string{
					"facing":         "north",
					"segment_amount": "4",
				},
			},
			expectType: block.LeafLitter{},
		},
		{
			name:       "pale_moss_block",
			state:      gen.BlockState{Name: "minecraft:pale_moss_block"},
			expectType: block.PaleMossBlock{},
		},
		{
			name: "big_dripleaf",
			state: gen.BlockState{
				Name: "minecraft:big_dripleaf",
				Properties: map[string]string{
					"facing":      "east",
					"tilt":        "none",
					"waterlogged": "false",
				},
			},
			expectType: block.BigDripleaf{},
		},
		{
			name: "small_dripleaf",
			state: gen.BlockState{
				Name: "minecraft:small_dripleaf",
				Properties: map[string]string{
					"facing":      "east",
					"half":        "lower",
					"waterlogged": "false",
				},
			},
			expectType: block.SmallDripleaf{},
		},
		{
			name: "mangrove_propagule",
			state: gen.BlockState{
				Name: "minecraft:mangrove_propagule",
				Properties: map[string]string{
					"age":         "4",
					"hanging":     "true",
					"stage":       "0",
					"waterlogged": "false",
				},
			},
			expectType: block.MangrovePropagule{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, ok := g.featureBlockFromState(tt.state, nil)
			if !ok {
				t.Fatalf("expected %s fallback state to resolve", tt.name)
			}
			if reflect.TypeOf(b) != reflect.TypeOf(tt.expectType) {
				t.Fatalf("expected %s to resolve to %T, got %T", tt.name, tt.expectType, b)
			}
		})
	}
}

func TestFeatureBlockNameNormalizesImplementedBedrockStateNames(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	c.SetBlock(1, 1, 1, 0, world.BlockRuntimeID(block.RootedDirt{}))
	c.SetBlock(2, 1, 2, 0, world.BlockRuntimeID(block.SmallDripleaf{Facing: cube.East}))
	c.SetBlock(3, 1, 3, 0, world.BlockRuntimeID(block.BigDripleaf{Facing: cube.South}))
	c.SetBlock(4, 1, 4, 0, world.BlockRuntimeID(block.BigDripleaf{Facing: cube.South, Head: true}))

	if got := g.blockNameAt(c, cube.Pos{1, 1, 1}); got != "rooted_dirt" {
		t.Fatalf("expected rooted dirt block name, got %q", got)
	}
	if got := g.blockNameAt(c, cube.Pos{2, 1, 2}); got != "small_dripleaf" {
		t.Fatalf("expected small dripleaf block name, got %q", got)
	}
	if got := g.blockNameAt(c, cube.Pos{3, 1, 3}); got != "big_dripleaf_stem" {
		t.Fatalf("expected big dripleaf stem name, got %q", got)
	}
	if got := g.blockNameAt(c, cube.Pos{4, 1, 4}); got != "big_dripleaf" {
		t.Fatalf("expected big dripleaf name, got %q", got)
	}
}

func TestFeatureBlockTagMatchesJavaSupportTags(t *testing.T) {
	if !featureBlockTagMatches("podzol", "substrate_overworld") {
		t.Fatal("expected substrate_overworld to include podzol")
	}
	if !featureBlockTagMatches("muddy_mangrove_roots", "substrate_overworld") {
		t.Fatal("expected substrate_overworld to include muddy mangrove roots")
	}
	if !featureBlockTagMatches("coarse_dirt", "supports_big_dripleaf") {
		t.Fatal("expected supports_big_dripleaf to include coarse dirt")
	}
	if !featureBlockTagMatches("red_sand", "azalea_grows_on") {
		t.Fatal("expected azalea_grows_on to include red sand")
	}
	if !featureBlockTagMatches("terracotta", "azalea_grows_on") || !featureBlockTagMatches("white_terracotta", "azalea_grows_on") {
		t.Fatal("expected azalea_grows_on to include terracotta variants")
	}
	if !featureBlockTagMatches("moss_carpet", "mangrove_roots_can_grow_through") {
		t.Fatal("expected mangrove_roots_can_grow_through to include moss carpet")
	}
	if !featureBlockTagMatches("mangrove_log", "mangrove_logs_can_grow_through") {
		t.Fatal("expected mangrove_logs_can_grow_through to include mangrove logs")
	}
	if featureBlockTagMatches("air", "mangrove_logs_can_grow_through") {
		t.Fatal("expected mangrove_logs_can_grow_through to reject air")
	}
	if !featureBlockTagMatches("cave_vines", "moss_replaceable") {
		t.Fatal("expected moss_replaceable to include cave vines")
	}
	if !featureBlockTagMatches("grass", "forest_rock_can_place_on") || !featureBlockTagMatches("stone", "forest_rock_can_place_on") {
		t.Fatal("expected forest_rock_can_place_on to include overworld substrate and stone")
	}
	if !featureBlockTagMatches("gravel", "azalea_root_replaceable") {
		t.Fatal("expected azalea_root_replaceable to include gravel")
	}
	if !featureBlockTagMatches("grass", "beneath_tree_podzol_replaceable") {
		t.Fatal("expected beneath_tree_podzol_replaceable to include grass blocks")
	}
	if !featureBlockTagMatches("mycelium", "huge_brown_mushroom_can_place_on") || !featureBlockTagMatches("crimson_nylium", "huge_red_mushroom_can_place_on") {
		t.Fatal("expected huge mushroom placement tags to include vanilla mushroom substrates")
	}
	if !featureBlockTagMatches("brown_mushroom_block", "replaceable_by_mushrooms") {
		t.Fatal("expected replaceable_by_mushrooms to include mushroom blocks")
	}
	if !featureBlockTagMatches("ice", "ice_spike_replaceable") || !featureBlockTagMatches("snow_block", "ice_spike_replaceable") {
		t.Fatal("expected ice_spike_replaceable to include ice and snow blocks")
	}
	if !featureBlockTagMatches("dandelion", "replaceable_by_trees") {
		t.Fatal("expected replaceable_by_trees to include small flowers")
	}
	if featureBlockTagMatches("red_mushroom", "replaceable_by_trees") {
		t.Fatal("expected replaceable_by_trees to reject mushrooms")
	}
	if featureBlockTagMatches("snow", "replaceable_by_trees") {
		t.Fatal("expected replaceable_by_trees to reject snow layers")
	}
}

func TestMangrovePropaguleSurvivesOnMudAndClay(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	for _, tc := range []struct {
		name string
		base world.Block
		pos  cube.Pos
	}{
		{name: "mud", base: block.Mud{}, pos: cube.Pos{8, 1, 8}},
		{name: "clay", base: block.Clay{}, pos: cube.Pos{9, 1, 9}},
	} {
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		c.SetBlock(uint8(tc.pos[0]), 0, uint8(tc.pos[2]), 0, world.BlockRuntimeID(tc.base))

		ok := g.canBlockStateSurvive(c, tc.pos, gen.BlockState{
			Name: "minecraft:mangrove_propagule",
			Properties: map[string]string{
				"age":         "0",
				"hanging":     "false",
				"stage":       "0",
				"waterlogged": "false",
			},
		}, nil, c.Range().Min(), c.Range().Max())
		if !ok {
			t.Fatalf("expected mangrove propagule to survive on %s", tc.name)
		}
	}
}

func TestAzaleaSurvivesOnMud(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	c.SetBlock(8, 0, 8, 0, world.BlockRuntimeID(block.Mud{}))

	if !g.canBlockStateSurvive(c, cube.Pos{8, 1, 8}, gen.BlockState{Name: "minecraft:azalea"}, nil, c.Range().Min(), c.Range().Max()) {
		t.Fatal("expected azalea to survive on mud")
	}
}

func TestDripleafSurvivalMatchesJavaSupportTags(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	c.SetBlock(4, 0, 4, 0, world.BlockRuntimeID(block.MossBlock{}))
	c.SetBlock(5, 0, 5, 0, world.BlockRuntimeID(block.Stone{}))
	c.SetBlock(6, 0, 6, 0, world.BlockRuntimeID(block.Farmland{}))
	c.SetBlock(7, 0, 7, 0, world.BlockRuntimeID(block.Stone{}))

	if !g.canBlockStateSurvive(c, cube.Pos{4, 1, 4}, gen.BlockState{
		Name: "minecraft:small_dripleaf",
		Properties: map[string]string{
			"facing": "north",
			"half":   "lower",
		},
	}, nil, c.Range().Min(), c.Range().Max()) {
		t.Fatal("expected small dripleaf to survive on moss block")
	}
	if g.canBlockStateSurvive(c, cube.Pos{5, 1, 5}, gen.BlockState{
		Name: "minecraft:small_dripleaf",
		Properties: map[string]string{
			"facing": "north",
			"half":   "lower",
		},
	}, nil, c.Range().Min(), c.Range().Max()) {
		t.Fatal("expected small dripleaf to reject unsupported stone")
	}
	if !g.canBlockStateSurvive(c, cube.Pos{6, 1, 6}, gen.BlockState{
		Name: "minecraft:big_dripleaf",
		Properties: map[string]string{
			"facing": "north",
			"tilt":   "none",
		},
	}, nil, c.Range().Min(), c.Range().Max()) {
		t.Fatal("expected big dripleaf to survive on farmland")
	}
	if g.canBlockStateSurvive(c, cube.Pos{7, 1, 7}, gen.BlockState{
		Name: "minecraft:big_dripleaf",
		Properties: map[string]string{
			"facing": "north",
			"tilt":   "none",
		},
	}, nil, c.Range().Min(), c.Range().Max()) {
		t.Fatal("expected big dripleaf to reject unsupported stone")
	}
}

func TestExecuteKelpPreservesWaterLayer(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	pos := cube.Pos{8, 4, 8}
	c.SetBlock(uint8(pos[0]), int16(pos[1]-1), uint8(pos[2]), 0, world.BlockRuntimeID(block.Stone{}))
	for y := pos[1]; y <= pos[1]+8; y++ {
		c.SetBlock(uint8(pos[0]), int16(y), uint8(pos[2]), 0, g.waterRID)
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	if !g.executeKelp(c, pos, c.Range().Min(), c.Range().Max(), &rng) {
		t.Fatal("expected kelp feature to place at least one kelp block")
	}

	placed, ok := world.BlockByRuntimeID(c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 0))
	if !ok {
		t.Fatal("expected placed kelp block to resolve")
	}
	if _, ok := placed.(block.Kelp); !ok {
		t.Fatalf("expected kelp at placement position, got %T", placed)
	}
	if c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 1) != g.waterRID {
		t.Fatal("expected kelp to preserve source water in chunk layer 1")
	}
}

func TestExecuteSeaPicklePreservesWaterLayer(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	pos := cube.Pos{8, 4, 8}
	c.SetBlock(uint8(pos[0]), int16(pos[1]-1), uint8(pos[2]), 0, world.BlockRuntimeID(block.Stone{}))
	c.SetBlock(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 0, g.waterRID)

	rng := gen.NewXoroshiro128FromSeed(1)
	if !g.executeSeaPickle(c, pos, gen.SeaPickleConfig{Count: 1}, 0, 0, c.Range().Min(), c.Range().Max(), &rng) {
		t.Fatal("expected sea pickle feature to place a plant")
	}

	if c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 0) == g.waterRID {
		t.Fatal("expected sea pickle block to replace foreground water")
	}
	if c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 1) != g.waterRID {
		t.Fatal("expected sea pickle block to preserve source water in chunk layer 1")
	}
}

func TestExecuteConfiguredOakTreePlacesBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "oak"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected oak configured feature to place a tree")
	}
	if countTreeBlocks(c) == 0 {
		t.Fatal("expected oak configured feature to create logs or leaves")
	}
}

func TestExecuteConfiguredOakTreeReplacesGrassBelowTrunkWithDirt(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "oak"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected oak configured feature to place a tree")
	}

	b, ok := world.BlockByRuntimeID(c.Block(8, 0, 8, 0))
	if !ok {
		t.Fatal("expected block below oak trunk to resolve")
	}
	if _, ok := b.(block.Dirt); !ok {
		t.Fatalf("expected oak below-trunk provider to replace grass with dirt, got %T", b)
	}
}

func TestTreeLeafUpdateKeepsConnectedLeavesStable(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	logPos := cube.Pos{8, 1, 8}
	leafPos := cube.Pos{9, 1, 8}
	if !g.setFeatureBlock(c, logPos, block.Log{Wood: block.OakWood(), Axis: cube.Y}) {
		t.Fatal("expected test log placement to succeed")
	}
	if !g.setFeatureBlock(c, leafPos, block.Leaves{Type: block.OakLeaves(), ShouldUpdate: true}) {
		t.Fatal("expected test leaf placement to succeed")
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	rt := newTreeRuntime(g, c, logPos, gen.TreeConfig{}, c.Range().Min(), c.Range().Max(), &rng)
	rt.trunks.add(logPos)
	rt.foliage.add(leafPos)
	rt.updateLeaves()

	shouldUpdate, ok := leafShouldUpdateAt(c, leafPos)
	if !ok {
		t.Fatal("expected connected leaf block to remain present")
	}
	if shouldUpdate {
		t.Fatal("expected leaf within six blocks of the trunk to be marked stable")
	}
}

func TestTreeLeafUpdateMarksDisconnectedLeavesForDecay(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	logPos := cube.Pos{1, 1, 1}
	leafPos := cube.Pos{8, 1, 1}
	if !g.setFeatureBlock(c, logPos, block.Log{Wood: block.OakWood(), Axis: cube.Y}) {
		t.Fatal("expected test log placement to succeed")
	}
	if !g.setFeatureBlock(c, leafPos, block.Leaves{Type: block.OakLeaves(), ShouldUpdate: false}) {
		t.Fatal("expected test leaf placement to succeed")
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	rt := newTreeRuntime(g, c, logPos, gen.TreeConfig{}, c.Range().Min(), c.Range().Max(), &rng)
	rt.trunks.add(logPos)
	rt.foliage.add(leafPos)
	rt.updateLeaves()

	shouldUpdate, ok := leafShouldUpdateAt(c, leafPos)
	if !ok {
		t.Fatal("expected disconnected leaf block to remain present")
	}
	if !shouldUpdate {
		t.Fatal("expected leaf beyond the Java connection radius to be marked for decay")
	}
}

func TestExecuteConfiguredFancyOakPlacesHorizontalBranchLogs(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	for seed := int64(1); seed <= 256; seed++ {
		g := New(0)
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
			}
		}

		rng := gen.NewXoroshiro128FromSeed(seed)
		biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeForest)
		if g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "fancy_oak"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) &&
			countHorizontalLogsForWood(c, block.OakWood()) > 0 {
			return
		}
	}
	t.Fatal("expected fancy_oak to place horizontal branch logs for at least one deterministic seed")
}

func TestExecuteConfiguredCherryPlacesHorizontalBranchLogs(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	for seed := int64(1); seed <= 8; seed++ {
		g := New(0)
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
			}
		}

		rng := gen.NewXoroshiro128FromSeed(seed)
		biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeCherryGrove)
		if g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "cherry"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) &&
			countHorizontalLogsForWood(c, block.CherryWood()) > 0 {
			return
		}
	}
	t.Fatal("expected cherry trunk placer to place horizontal branch logs for at least one deterministic seed")
}

func TestExecuteConfiguredAzaleaTreeRejectsBlockingVines(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}
	c.SetBlock(8, 2, 8, 0, world.BlockRuntimeID((block.Vines{}).WithAttachment(cube.North, true)))

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeLushCaves)
	if g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "azalea_tree"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected azalea_tree to reject trunk space blocked by vines when ignore_vines is false")
	}
}

func TestExecuteConfiguredMangrovePlacesRootBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	rng := gen.NewXoroshiro128FromSeed(1)
	origin := cube.Pos{8, 2, 8}
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Stone{}))
		}
	}
	cfg := gen.TreeConfig{
		RootPlacer: gen.TypedJSONValue{
			Type: "mangrove_root_placer",
			Data: json.RawMessage(`{
				"root_provider":{"type":"minecraft:simple_state_provider","state":{"Name":"minecraft:mangrove_roots","Properties":{"waterlogged":"false"}}},
				"mangrove_root_placement":{
					"can_grow_through":"",
					"max_root_length":32,
					"max_root_width":8,
					"muddy_roots_in":["minecraft:mud"],
					"muddy_roots_provider":{"type":"minecraft:simple_state_provider","state":{"Name":"minecraft:muddy_mangrove_roots","Properties":{"axis":"y"}}},
					"random_skew_chance":1.0
				}
			}`),
		},
	}

	rt := newTreeRuntime(g, c, origin, cfg, c.Range().Min(), c.Range().Max(), &rng)
	if !rt.placeRoots(origin, origin) {
		t.Fatal("expected mangrove root placement to succeed with a generous root budget")
	}
	if countBlocksNamed(c, "minecraft:mangrove_roots")+countBlocksNamed(c, "minecraft:muddy_mangrove_roots") == 0 {
		t.Fatal("expected mangrove root placement to place root blocks")
	}
}

func TestMangroveRootPlacementRejectsExceededMaxRootLength(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	rng := gen.NewXoroshiro128FromSeed(1)
	origin := cube.Pos{8, 1, 8}
	cfg := gen.TreeConfig{
		RootPlacer: gen.TypedJSONValue{
			Type: "mangrove_root_placer",
			Data: json.RawMessage(`{
				"root_provider":{"type":"minecraft:simple_state_provider","state":{"Name":"minecraft:mangrove_roots","Properties":{"waterlogged":"false"}}},
				"mangrove_root_placement":{
					"can_grow_through":"",
					"max_root_length":0,
					"max_root_width":8,
					"muddy_roots_in":["minecraft:mud"],
					"muddy_roots_provider":{"type":"minecraft:simple_state_provider","state":{"Name":"minecraft:muddy_mangrove_roots","Properties":{"axis":"y"}}},
					"random_skew_chance":0.2
				}
			}`),
		},
	}

	rt := newTreeRuntime(g, c, origin, cfg, c.Range().Min(), c.Range().Max(), &rng)
	if rt.placeRoots(origin, origin.Add(cube.Pos{0, 1, 0})) {
		t.Fatal("expected mangrove root placement to fail when the root path exceeds max_root_length")
	}
}

func TestExecuteConfiguredMegaPinePlacesAlterGroundPodzol(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeOldGrowthPineTaiga)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "mega_pine"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected mega_pine configured feature to place a tree")
	}
	if countBlocksNamed(c, "minecraft:podzol") == 0 {
		t.Fatal("expected mega_pine alter_ground decorator to place podzol")
	}
}

func TestExecuteConfiguredMegaPineAlterGroundRespectsJavaReplaceable(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			rid := world.BlockRuntimeID(block.Stone{})
			if x >= 6 && x <= 10 && z >= 6 && z <= 10 {
				rid = world.BlockRuntimeID(block.Grass{})
			}
			c.SetBlock(uint8(x), 0, uint8(z), 0, rid)
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeOldGrowthPineTaiga)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "mega_pine"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected mega_pine configured feature to place a tree")
	}

	outsidePatchPodzol := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			if x >= 6 && x <= 10 && z >= 6 && z <= 10 {
				continue
			}
			b, ok := world.BlockByRuntimeID(c.Block(uint8(x), 0, uint8(z), 0))
			if !ok {
				continue
			}
			name, _ := b.EncodeBlock()
			if name == "minecraft:podzol" {
				outsidePatchPodzol++
			}
		}
	}
	if outsidePatchPodzol != 0 {
		t.Fatalf("expected mega_pine alter_ground to avoid non-replaceable stone, got %d outside-patch podzol blocks", outsidePatchPodzol)
	}
}

func TestExecuteConfiguredForestRockPlacesMossyCobblestone(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeOldGrowthSpruceTaiga)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 12, 8}, gen.ConfiguredFeatureRef{Name: "forest_rock"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected forest_rock configured feature to place a blob")
	}
	if countBlocksNamed(c, "minecraft:mossy_cobblestone") == 0 {
		t.Fatal("expected forest_rock configured feature to place mossy cobblestone")
	}
}

func TestExecuteConfiguredIceSpikePlacesPackedIce(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 80, uint8(z), 0, world.BlockRuntimeID(block.Snow{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(2)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeIceSpikes)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 96, 8}, gen.ConfiguredFeatureRef{Name: "ice_spike"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected ice_spike configured feature to place packed ice")
	}
	if countBlocksNamed(c, "minecraft:packed_ice") == 0 {
		t.Fatal("expected ice_spike configured feature to place packed ice")
	}
}

func TestExecuteConfiguredHugeBrownMushroomPlacesCapAndStem(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	c.SetBlock(8, 0, 8, 0, world.BlockRuntimeID(block.Grass{}))

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeMushroomFields)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "huge_brown_mushroom"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected huge_brown_mushroom configured feature to place")
	}
	if countBlocksNamed(c, "minecraft:brown_mushroom_block") == 0 || countBlocksNamed(c, "minecraft:mushroom_stem") == 0 {
		t.Fatal("expected huge_brown_mushroom to place both cap and stem blocks")
	}
}

func TestExecuteConfiguredMushroomIslandVegetationCanPlaceHugeMushroom(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	for seed := int64(1); seed <= 16; seed++ {
		g := New(0)
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		c.SetBlock(8, 0, 8, 0, world.BlockRuntimeID(block.Grass{}))

		rng := gen.NewXoroshiro128FromSeed(seed)
		biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeMushroomFields)
		if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "mushroom_island_vegetation"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
			continue
		}
		if countBlocksNamed(c, "minecraft:brown_mushroom_block")+countBlocksNamed(c, "minecraft:red_mushroom_block") > 0 {
			return
		}
	}
	t.Fatal("expected mushroom_island_vegetation to place a huge mushroom for at least one deterministic seed")
}

func TestExecuteConfiguredDesertWellPlacesSandstoneAndWater(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 60; y <= 62; y++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, world.BlockRuntimeID(block.Sand{}))
			}
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeDesert)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 80, 8}, gen.ConfiguredFeatureRef{Name: "desert_well"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected desert_well configured feature to place")
	}
	if countBlocksNamed(c, "minecraft:sandstone") == 0 {
		t.Fatal("expected desert well to place sandstone")
	}
	centerRID0 := c.Block(8, 62, 8, 0)
	centerRID1 := c.Block(8, 62, 8, 1)
	center0, _ := world.BlockByRuntimeID(centerRID0)
	center1, _ := world.BlockByRuntimeID(centerRID1)
	name0, _ := center0.EncodeBlock()
	name1, _ := center1.EncodeBlock()
	if name0 != "minecraft:water" && name1 != "minecraft:water" && name0 != "minecraft:flowing_water" && name1 != "minecraft:flowing_water" {
		t.Fatal("expected desert well center to contain water")
	}
}

func TestExecuteConfiguredVoidStartPlatformPlacesStonePlatform(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.End)
	c := chunk.New(g.airRID, world.End.Range())
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeTheVoid)
	rng := gen.NewXoroshiro128FromSeed(1)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{0, 0, 0}, gen.ConfiguredFeatureRef{Name: "void_start_platform"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected void_start_platform configured feature to place")
	}
	if countBlocksNamed(c, "minecraft:stone") == 0 {
		t.Fatal("expected void_start_platform to place stone")
	}
	if countBlocksNamed(c, "minecraft:cobblestone") == 0 {
		t.Fatal("expected void_start_platform to place cobblestone center")
	}
}

func TestExecuteConfiguredIcebergPlacesIce(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := seaLevel - 8; y <= seaLevel; y++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, world.BlockRuntimeID(block.Water{Still: true, Depth: 8}))
			}
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeFrozenOcean)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 0, 8}, gen.ConfiguredFeatureRef{Name: "iceberg_blue"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected iceberg configured feature to place")
	}
	if countBlocksNamed(c, "minecraft:blue_ice")+countBlocksNamed(c, "minecraft:packed_ice")+countBlocksNamed(c, "minecraft:snow") == 0 {
		t.Fatal("expected iceberg configured feature to place iceberg blocks")
	}
}

func TestExecuteConfiguredMonsterRoomPlacesSpawnerAndRoomShell(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 20; y <= 50; y++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, world.BlockRuntimeID(block.Stone{}))
			}
		}
	}
	// Ensure 1-5 valid openings regardless of xr/zr being 2 or 3.
	for _, x := range []int{11, 12} {
		c.SetBlock(uint8(x), 40, 8, 0, g.airRID)
		c.SetBlock(uint8(x), 41, 8, 0, g.airRID)
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 40, 8}, gen.ConfiguredFeatureRef{Name: "monster_room"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected monster_room configured feature to place")
	}
	if countBlocksNamed(c, "minecraft:mob_spawner") == 0 {
		t.Fatal("expected monster_room to place a mob spawner")
	}
	if countBlocksNamed(c, "minecraft:cobblestone")+countBlocksNamed(c, "minecraft:mossy_cobblestone") == 0 {
		t.Fatal("expected monster_room to place cobblestone shell blocks")
	}
}

func TestExecuteConfiguredLargeDripstonePlacesDripstoneBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 20; y <= 60; y++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, g.airRID)
			}
			c.SetBlock(uint8(x), 20, uint8(z), 0, world.BlockRuntimeID(block.Stone{}))
			c.SetBlock(uint8(x), 60, uint8(z), 0, world.BlockRuntimeID(block.Stone{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeDripstoneCaves)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 40, 8}, gen.ConfiguredFeatureRef{Name: "large_dripstone"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected large_dripstone configured feature to place")
	}
	if countBlocksNamed(c, "minecraft:dripstone_block") == 0 {
		t.Fatal("expected large_dripstone to place dripstone blocks")
	}
}

func TestExecuteConfiguredAmethystGeodePlacesGeodeBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y <= 48; y++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, world.BlockRuntimeID(block.Stone{}))
			}
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 24, 8}, gen.ConfiguredFeatureRef{Name: "amethyst_geode"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected amethyst_geode configured feature to place")
	}
	total := countBlocksNamed(c, "minecraft:amethyst_block") +
		countBlocksNamed(c, "minecraft:budding_amethyst") +
		countBlocksNamed(c, "minecraft:calcite") +
		countBlocksNamed(c, "minecraft:smooth_basalt")
	if total == 0 {
		t.Fatal("expected amethyst_geode to place layered geode blocks")
	}
}

func TestExecuteConfiguredMegaJunglePlacesVines(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	for seed := int64(1); seed <= 32; seed++ {
		g := New(0)
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
			}
		}

		rng := gen.NewXoroshiro128FromSeed(seed)
		biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeJungle)
		if g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "mega_jungle_tree"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) &&
			countBlocksNamed(c, "minecraft:vine") > 0 {
			return
		}
	}
	t.Fatal("expected mega_jungle_tree decorators to place vines for at least one deterministic seed")
}

func TestExecuteConfiguredBirchBeesPlacesBeeNest(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	for seed := int64(1); seed <= 1024; seed++ {
		g := New(0)
		c := chunk.New(g.airRID, cube.Range{-64, 319})
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
			}
		}

		rng := gen.NewXoroshiro128FromSeed(seed)
		biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
		if g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "birch_bees_005"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) &&
			countBlocksNamed(c, "minecraft:bee_nest") > 0 {
			return
		}
	}
	t.Fatal("expected birch_bees_005 decorator to place a bee nest for at least one deterministic seed")
}

func TestRunPlacedTreeFeatureAcrossRegionReplaysNeighbourTreesIntoCenterChunk(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	const featureName = "trees_birch"
	for seed := int64(1); seed <= 1024; seed++ {
		g := New(seed)
		featureIndex, ok := g.biomeGeneration.featureIndex(gen.GenerationStepVegetalDecoration, featureName)
		if !ok {
			t.Fatalf("expected %s feature index", featureName)
		}

		region := &treeDecorationRegion{
			centerChunkX: 0,
			centerChunkZ: 0,
			minY:         -64,
			maxY:         319,
			slots:        make(map[[2]int]treeDecorationRegionSlot, 9),
		}
		for chunkX := -1; chunkX <= 1; chunkX++ {
			for chunkZ := -1; chunkZ <= 1; chunkZ++ {
				c := chunk.New(g.airRID, cube.Range{region.minY, region.maxY})
				for x := 0; x < 16; x++ {
					for z := 0; z < 16; z++ {
						c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
					}
				}

				biome := gen.BiomePlains
				if chunkX == -1 && chunkZ == 0 {
					biome = gen.BiomeBirchForest
				}
				region.set(chunkX, chunkZ, c, filledTestBiomeVolume(region.minY, region.maxY, biome))
			}
		}

		g.runPlacedTreeFeatureAcrossRegion(region, gen.GenerationStepVegetalDecoration, featureName, featureIndex)
		center, ok := region.slot(0, 0)
		if !ok {
			t.Fatal("expected center chunk in tree replay region")
		}
		if countTreeBlocks(center.chunk) > 0 {
			return
		}
	}
	t.Fatal("expected at least one deterministic seed to replay a neighbouring birch tree into the center chunk")
}

func TestExecuteNetherQuartzOrePlacesBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	c := chunk.New(g.airRID, world.Nether.Range())
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, world.BlockRuntimeID(block.Netherrack{}))
			}
		}
	}

	cfg, err := g.features.Configured("ore_quartz")
	if err != nil {
		t.Fatalf("failed to load ore_quartz: %v", err)
	}
	ore, err := cfg.Ore()
	if err != nil {
		t.Fatalf("failed to decode ore_quartz: %v", err)
	}
	rng := gen.NewXoroshiro128FromSeed(1)
	if !g.executeOre(c, cube.Pos{8, 32, 8}, ore, 0, 0, c.Range().Min(), c.Range().Max(), &rng, false) {
		t.Fatal("expected nether quartz ore feature to place at least one block")
	}

	totalOres := 0
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				name, _ := b.EncodeBlock()
				if strings.HasSuffix(strings.TrimPrefix(name, "minecraft:"), "_ore") {
					totalOres++
				}
			}
		}
	}
	if totalOres == 0 {
		t.Fatal("expected executeOre to leave quartz ore blocks in the chunk")
	}
}

func TestRunPlacedFeatureNetherQuartzPlacesBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	c := chunk.New(g.airRID, world.Nether.Range())
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, world.BlockRuntimeID(block.Netherrack{}))
				c.SetBiome(uint8(x), int16(y), uint8(z), biomeRuntimeID(gen.BiomeNetherWastes))
			}
		}
	}

	placed, err := g.features.Placed("ore_quartz_nether")
	if err != nil {
		t.Fatalf("failed to load ore_quartz_nether: %v", err)
	}
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeNetherWastes)
	featureIndex, ok := g.biomeGeneration.featureIndex(gen.GenerationStepUndergroundDecoration, "ore_quartz_nether")
	if !ok {
		t.Fatal("expected ore_quartz_nether feature index")
	}
	rng := g.featureRNG(g.decorationSeed(0, 0), featureIndex, gen.GenerationStepUndergroundDecoration)
	positions, ok := g.applyPlacementModifiers(c, biomes, []cube.Pos{{0, c.Range().Min(), 0}}, placed.Placement, "ore_quartz_nether", 0, 0, c.Range().Min(), c.Range().Max(), &rng)
	if !ok {
		t.Fatal("expected placement modifiers for ore_quartz_nether to be supported")
	}
	if len(positions) == 0 {
		t.Fatal("expected ore_quartz_nether placement modifiers to produce candidate positions")
	}
	for _, pos := range positions {
		g.executeConfiguredFeature(c, biomes, pos, placed.Feature, "ore_quartz_nether", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0)
	}

	totalOres := 0
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				name, _ := b.EncodeBlock()
				if strings.HasSuffix(strings.TrimPrefix(name, "minecraft:"), "_ore") {
					totalOres++
				}
			}
		}
	}
	if totalOres == 0 {
		t.Fatal("expected placed feature ore_quartz_nether to leave ore blocks in the chunk")
	}
}

func TestCountOnEveryLayerPlacementFindsSuccessiveLayers(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	c := chunk.New(g.airRID, world.Nether.Range())
	netherrackRID := world.BlockRuntimeID(block.Netherrack{})
	for y := 0; y <= 29; y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, netherrackRID)
			}
		}
	}
	for y := 36; y <= 50; y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, netherrackRID)
			}
		}
	}

	var modifier gen.PlacementModifier
	if err := json.Unmarshal([]byte(`{"count":1,"type":"minecraft:count_on_every_layer"}`), &modifier); err != nil {
		t.Fatalf("decode count_on_every_layer modifier: %v", err)
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeNetherWastes)
	positions, ok := g.applyPlacementModifiers(c, biomes, []cube.Pos{{0, c.Range().Min(), 0}}, []gen.PlacementModifier{modifier}, biomeKey(gen.BiomeNetherWastes), 0, 0, c.Range().Min(), c.Range().Max(), &rng)
	if !ok {
		t.Fatal("expected count_on_every_layer modifier to be supported")
	}
	if len(positions) != 2 {
		t.Fatalf("expected one placement per exposed layer, got %d", len(positions))
	}
	if positions[0][1] != 51 || positions[1][1] != 30 {
		t.Fatalf("expected layer placements at y=51 and y=30, got %v", positions)
	}
}

func TestExecuteConfiguredOakTreeHasRoundedTop(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "oak"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected oak configured feature to place a tree")
	}

	top, ok := highestTreeLog(c)
	if !ok {
		t.Fatal("expected placed oak tree to contain a log")
	}
	cardinalLeaves := 0
	for _, off := range []cube.Pos{{1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}} {
		if isLeafBlockAt(c, top.Add(off)) {
			cardinalLeaves++
		}
	}
	if cardinalLeaves < 3 {
		t.Fatalf("expected oak canopy around top log to place cardinal leaves, got %d", cardinalLeaves)
	}
	if !isLeafBlockAt(c, top.Side(cube.FaceUp)) {
		t.Fatalf("expected oak canopy to place a leaf above the top log at %v", top.Side(cube.FaceUp))
	}
}

func TestExecuteConfiguredOakTreeRejectsBlockedCanopy(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	buildSoil := func(c *chunk.Chunk) {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
			}
		}
	}

	preview := chunk.New(g.airRID, cube.Range{-64, 319})
	buildSoil(preview)
	biomes := filledTestBiomeVolume(preview.Range().Min(), preview.Range().Max(), gen.BiomePlains)
	previewRNG := gen.NewXoroshiro128FromSeed(1)
	if !g.executeConfiguredFeature(preview, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "oak"}, "", 0, 0, preview.Range().Min(), preview.Range().Max(), &previewRNG, 0) {
		t.Fatal("expected preview oak tree to place")
	}
	top, ok := highestTreeLog(preview)
	if !ok {
		t.Fatal("expected preview oak tree to contain a top log")
	}

	blocked := chunk.New(g.airRID, cube.Range{-64, 319})
	buildSoil(blocked)
	obstacle := top.Side(cube.FaceUp)
	blocked.SetBlock(uint8(obstacle[0]), int16(obstacle[1]), uint8(obstacle[2]), 0, world.BlockRuntimeID(block.Stone{}))

	rng := gen.NewXoroshiro128FromSeed(1)
	if g.executeConfiguredFeature(blocked, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "oak"}, "", 0, 0, blocked.Range().Min(), blocked.Range().Max(), &rng, 0) {
		t.Fatalf("expected blocked oak tree generation to fail when canopy space at %v is occupied", obstacle)
	}
	if countTreeBlocks(blocked) != 0 {
		t.Fatal("expected blocked oak tree generation to leave the chunk unchanged")
	}
}

func TestExecutePlacedOakCheckedPlacesBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	if !g.executePlacedFeatureRef(c, biomes, cube.Pos{8, 1, 8}, gen.PlacedFeatureRef{Name: "oak_checked"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected oak_checked placed feature to place a tree")
	}
	if countTreeBlocks(c) == 0 {
		t.Fatal("expected oak_checked placed feature to create logs or leaves")
	}
}

func TestHeightmapPlacementYCountsWaterAboveOceanFloor(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	c.SetBlock(8, 0, 8, 0, world.BlockRuntimeID(block.Grass{}))
	c.SetBlock(8, 1, 8, 0, g.waterRID)
	c.SetBlock(8, 2, 8, 0, g.waterRID)

	worldSurface := g.heightmapPlacementY(c, 8, 8, "WORLD_SURFACE", c.Range().Min(), c.Range().Max())
	if worldSurface != 3 {
		t.Fatalf("expected world surface height 3, got %d", worldSurface)
	}
	oceanFloor := g.heightmapPlacementY(c, 8, 8, "OCEAN_FLOOR", c.Range().Min(), c.Range().Max())
	if oceanFloor != 1 {
		t.Fatalf("expected ocean floor height 1, got %d", oceanFloor)
	}
	if depth := g.surfaceWaterDepthAt(c, 8, 8, c.Range().Min()); depth != 2 {
		t.Fatalf("expected surface water depth 2, got %d", depth)
	}
}

func TestExecuteLakeLavaRejectsWaterBoundary(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	stoneRID := world.BlockRuntimeID(block.Stone{})
	for y := 0; y <= 24; y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, stoneRID)
			}
		}
	}

	pos := cube.Pos{8, 16, 8}
	for x := 4; x <= 12; x++ {
		for z := 4; z <= 12; z++ {
			for y := pos[1] - 1; y <= pos[1]+2; y++ {
				c.SetBlock(uint8(x), int16(y), uint8(z), 0, g.waterRID)
			}
		}
	}

	feature, err := g.features.Configured("lake_lava")
	if err != nil {
		t.Fatalf("load lake_lava: %v", err)
	}
	cfg, err := feature.Lake()
	if err != nil {
		t.Fatalf("decode lake_lava: %v", err)
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	if g.executeLake(c, pos, cfg, 0, 0, c.Range().Min(), c.Range().Max(), &rng) {
		t.Fatal("expected lava lake generation to abort when the cavity intersects water")
	}
	if c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 0) == g.lavaRID {
		t.Fatal("expected no lava to be placed after lake validation failed")
	}
}

func TestTreesBirchPlacementSkipsWaterColumns(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	grassRID := world.BlockRuntimeID(block.Grass{})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, grassRID)
			c.SetBlock(uint8(x), 1, uint8(z), 0, g.waterRID)
			c.SetBlock(uint8(x), 2, uint8(z), 0, g.waterRID)
		}
	}

	placed, err := g.features.Placed("trees_birch")
	if err != nil {
		t.Fatalf("failed to load trees_birch: %v", err)
	}
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeBirchForest)
	featureIndex, ok := g.biomeGeneration.featureIndex(gen.GenerationStepVegetalDecoration, "trees_birch")
	if !ok {
		t.Fatal("expected trees_birch feature index")
	}
	rng := g.featureRNG(g.decorationSeed(0, 0), featureIndex, gen.GenerationStepVegetalDecoration)
	positions, ok := g.applyPlacementModifiers(c, biomes, []cube.Pos{{0, c.Range().Min(), 0}}, placed.Placement, "trees_birch", 0, 0, c.Range().Min(), c.Range().Max(), &rng)
	if !ok {
		t.Fatal("expected trees_birch placement modifiers to be supported")
	}
	if len(positions) != 0 {
		t.Fatalf("expected water depth filter to reject birch tree placements, got %d position(s)", len(positions))
	}
}

func TestGenerateChunkAtSpawnHintPlacesTrees(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	hint, ok := g.FindSpawnChunk(128)
	if !ok {
		t.Fatal("expected to find a spawn hint chunk")
	}

	c := chunk.New(g.airRID, cube.Range{-64, 319})
	g.GenerateChunk(hint, c)
	if countTreeBlocks(c) == 0 {
		t.Fatalf("expected spawn hint chunk %v to contain trees", hint)
	}
}

func TestExecuteConfiguredBambooPlacesBamboo(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.Grass{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeBambooJungle)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "bamboo_some_podzol"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected bamboo configured feature to place stalks")
	}

	found := false
	largeLeaves := false
	for y := 1; y <= c.Range().Max(); y++ {
		b, ok := world.BlockByRuntimeID(c.Block(8, int16(y), 8, 0))
		if !ok {
			continue
		}
		bamboo, ok := b.(block.Bamboo)
		if ok {
			found = true
			if bamboo.LeafSize == block.BambooLargeLeaves() {
				largeLeaves = true
			}
		}
	}
	if !found {
		t.Fatal("expected bamboo blocks to be present")
	}
	if !largeLeaves {
		t.Fatal("expected bamboo feature to place leafy bamboo top states")
	}
}

func TestExecuteConfiguredBambooPodzolizesSubstrateOverworld(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, world.BlockRuntimeID(block.RootedDirt{}))
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	if !g.executeBamboo(c, cube.Pos{8, 1, 8}, gen.BambooConfig{Probability: 1}, 0, 0, c.Range().Min(), c.Range().Max(), &rng) {
		t.Fatal("expected bamboo feature to place stalks")
	}

	found := false
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			b, ok := world.BlockByRuntimeID(c.Block(uint8(x), 0, uint8(z), 0))
			if !ok {
				continue
			}
			if _, ok := b.(block.Podzol); ok {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("expected bamboo podzol pass to replace rooted dirt like Java's substrate_overworld tag")
	}
}

func TestPlaceFeatureStateSmallDripleafIsAtomic(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	c.SetBlock(8, 0, 8, 0, world.BlockRuntimeID(block.MossBlock{}))
	c.SetBlock(8, 2, 8, 0, world.BlockRuntimeID(block.Stone{}))

	rng := gen.NewXoroshiro128FromSeed(1)
	ok := g.placeFeatureState(c, cube.Pos{8, 1, 8}, gen.BlockState{
		Name: "minecraft:small_dripleaf",
		Properties: map[string]string{
			"facing": "north",
			"half":   "lower",
		},
	}, &rng, c.Range().Min(), c.Range().Max())
	if ok {
		t.Fatal("expected small dripleaf placement to fail when the upper half is blocked")
	}
	if c.Block(8, 1, 8, 0) != g.airRID {
		t.Fatalf("expected failed small dripleaf placement to leave the lower block untouched, got runtime ID %d", c.Block(8, 1, 8, 0))
	}
}

func TestExecuteConfiguredMossPatchPlacesGround(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	stoneRID := world.BlockRuntimeID(block.Stone{})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, stoneRID)
		}
	}

	rng := gen.NewXoroshiro128FromSeed(1)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeLushCaves)
	if !g.executeConfiguredFeature(c, biomes, cube.Pos{8, 1, 8}, gen.ConfiguredFeatureRef{Name: "moss_patch"}, "", 0, 0, c.Range().Min(), c.Range().Max(), &rng, 0) {
		t.Fatal("expected moss_patch configured feature to place ground blocks")
	}

	found := false
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			b, ok := world.BlockByRuntimeID(c.Block(uint8(x), 0, uint8(z), 0))
			if !ok {
				continue
			}
			if _, ok := b.(block.MossBlock); ok {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected vegetation patch to replace ground with moss")
	}
}

func countDecorativeBlocks(c *chunk.Chunk) int {
	total := 0
	for y := c.Range().Min() + 1; y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				switch b.(type) {
				case block.ShortGrass, block.DoubleTallGrass, block.Flower, block.Pumpkin:
					total++
				}
			}
		}
	}
	return total
}

func countOreBlocks(c *chunk.Chunk) int {
	total := 0
	for y := c.Range().Min() + 1; y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				name, _ := b.EncodeBlock()
				name = strings.TrimPrefix(name, "minecraft:")
				if strings.HasSuffix(name, "_ore") || strings.HasPrefix(name, "infested_") || name == "ancient_debris" {
					total++
				}
			}
		}
	}
	return total
}

func countTreeBlocks(c *chunk.Chunk) int {
	total := 0
	for y := c.Range().Min() + 1; y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				switch b.(type) {
				case block.Log, block.Leaves:
					total++
					continue
				}
				name, _ := b.EncodeBlock()
				name = strings.TrimPrefix(name, "minecraft:")
				if strings.HasSuffix(name, "_log") || strings.HasSuffix(name, "_leaves") {
					total++
				}
			}
		}
	}
	return total
}

func highestTreeLog(c *chunk.Chunk) (cube.Pos, bool) {
	for y := c.Range().Max(); y >= c.Range().Min()+1; y-- {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				switch b.(type) {
				case block.Log:
					return cube.Pos{x, y, z}, true
				}
				name, _ := b.EncodeBlock()
				if strings.HasSuffix(strings.TrimPrefix(name, "minecraft:"), "_log") {
					return cube.Pos{x, y, z}, true
				}
			}
		}
	}
	return cube.Pos{}, false
}

func isLeafBlockAt(c *chunk.Chunk, pos cube.Pos) bool {
	if pos[1] <= c.Range().Min() || pos[1] > c.Range().Max() {
		return false
	}
	b, ok := world.BlockByRuntimeID(c.Block(uint8(pos[0]&15), int16(pos[1]), uint8(pos[2]&15), 0))
	if !ok {
		return false
	}
	switch b.(type) {
	case block.Leaves:
		return true
	}
	name, _ := b.EncodeBlock()
	return strings.HasSuffix(strings.TrimPrefix(name, "minecraft:"), "_leaves")
}

func countHorizontalLogsForWood(c *chunk.Chunk, wood block.WoodType) int {
	total := 0
	for y := c.Range().Min() + 1; y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				log, ok := b.(block.Log)
				if !ok || log.Wood != wood || log.Axis == cube.Y {
					continue
				}
				total++
			}
		}
	}
	return total
}

func countMuddyMangroveRoots(c *chunk.Chunk) int {
	total := 0
	for y := c.Range().Min() + 1; y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				if _, ok := b.(block.MuddyMangroveRoots); ok {
					total++
				}
			}
		}
	}
	return total
}

func countBlocksNamed(c *chunk.Chunk, name string) int {
	total := 0
	for y := c.Range().Min() + 1; y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				blockName, _ := b.EncodeBlock()
				if blockName == name {
					total++
				}
			}
		}
	}
	return total
}

func leafShouldUpdateAt(c *chunk.Chunk, pos cube.Pos) (bool, bool) {
	b, ok := world.BlockByRuntimeID(c.Block(uint8(pos[0]&15), int16(pos[1]), uint8(pos[2]&15), 0))
	if !ok {
		return false, false
	}
	leaves, ok := b.(block.Leaves)
	if !ok {
		return false, false
	}
	return leaves.ShouldUpdate, true
}

func filledTestBiomeVolume(minY, maxY int, biome gen.Biome) sourceBiomeVolume {
	volume := newSourceBiomeVolume(minY, maxY)
	for x := 0; x < 16; x += biomeCellSize {
		for z := 0; z < 16; z += biomeCellSize {
			for y := alignDown(minY, biomeCellSize); y <= maxY; y += biomeCellSize {
				volume.set(x, y, z, biome)
			}
		}
	}
	return volume
}

var finaliseBlocksOnce sync.Once

//go:linkname worldFinaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func worldFinaliseBlockRegistry()
