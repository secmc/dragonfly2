package vanilla

import (
	"math"
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	gen "github.com/df-mc/dragonfly/server/world/generator/vanilla/gen"
)

func TestStructureTemplateDecode(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	template, err := g.structureTemplates.Template("village/plains/town_centers/plains_meeting_point_1")
	if err != nil {
		t.Fatalf("decode structure template: %v", err)
	}
	if len(template.Palette) == 0 {
		t.Fatal("expected template palette entries")
	}
	if len(template.Blocks) == 0 {
		t.Fatal("expected template blocks")
	}
}

func TestPlanVillageStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	planner, ok := g.findStructurePlanner("villages")
	if !ok {
		t.Fatal("load village structure planner")
	}
	surfaceSampler := newStructureHeightSampler(g, -64, 319)

	for gridX := -8; gridX <= 8; gridX++ {
		for gridZ := -8; gridZ <= 8; gridZ++ {
			startChunk := randomSpreadPotentialChunk(g.seed, planner.placement, gridX, gridZ)
			start, exists := g.planStructureStart(planner, startChunk, -64, 319, surfaceSampler)
			if !exists {
				continue
			}
			if start.templateName == "" {
				t.Fatal("planned start is missing template name")
			}
			if start.size[0] <= 0 || start.size[1] <= 0 || start.size[2] <= 0 {
				t.Fatalf("expected positive planned start dimensions, got %+v", start.size)
			}
			if len(start.pieces) <= 1 {
				t.Fatalf("expected village start to expand beyond the root template, got %d piece(s)", len(start.pieces))
			}
			return
		}
	}
	t.Fatal("did not find a planned village structure start")
}

func TestPlanVillageStructureStartProjectsToWorldSurface(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "villages", 24)

	centerX := (start.rootOrigin[0]*2 + start.rootSize[0] - 1) / 2
	centerZ := (start.rootOrigin[2]*2 + start.rootSize[2] - 1) / 2
	wantY := g.worldSurfaceLevelAt(centerX, centerZ, -64, 319)
	gotY := start.rootOrigin[1] + 1
	if gotY != wantY {
		t.Fatalf("expected village root to project to world surface y=%d at (%d,%d), got %d", wantY, centerX, centerZ, gotY)
	}
}

func TestStructureHeightSamplerMatchesGenerator(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	surfaceSampler := newStructureHeightSampler(g, -64, 319)
	positions := [][2]int{
		{0, 0},
		{8, 8},
		{127, -193},
		{-255, 319},
		{2048, -1536},
	}
	for _, pos := range positions {
		blockX, blockZ := pos[0], pos[1]
		if got, want := surfaceSampler.preliminarySurfaceLevelAt(blockX, blockZ), g.preliminarySurfaceLevelAt(blockX, blockZ, -64, 319); got != want {
			t.Fatalf("expected preliminary sampler height %d at (%d,%d), got %d", want, blockX, blockZ, got)
		}
		if got, want := surfaceSampler.worldSurfaceLevelAt(blockX, blockZ), g.worldSurfaceLevelAt(blockX, blockZ, -64, 319); got != want {
			t.Fatalf("expected world sampler height %d at (%d,%d), got %d", want, blockX, blockZ, got)
		}
	}
}

func TestStructureTerrainSamplerMatchesVanillaFormula(t *testing.T) {
	sampler := &structureTerrainSampler{
		pieces: []structureTerrainPiece{{
			box:               structureBox{minX: 0, minY: 10, minZ: 0, maxX: 4, maxY: 14, maxZ: 4},
			terrainAdaptation: "encapsulate",
			groundLevelDelta:  1,
		}},
		junctions: []plannedStructureJunction{{
			sourceX:       8,
			sourceGroundY: 12,
			sourceZ:       8,
		}},
	}

	got := sampler.sample(2, 8, 2)

	pieceExpected := clampFloat64(1.0-math.Sqrt(1.0)/6.0, 0.0, 1.0) * 0.8
	dx, dy, dz := -6, -4, -6
	offsetY := float64(dy) + 0.5
	distanceSq := float64(dx*dx+dz*dz) + offsetY*offsetY
	junctionExpected := (-offsetY / math.Sqrt(distanceSq/2.0) / 2.0) * structureWeightTable[(dz+structureWeightIndexOffset)*structureWeightEdgeLength*structureWeightEdgeLength+(dx+structureWeightIndexOffset)*structureWeightEdgeLength+(dy+structureWeightIndexOffset)] * 0.4
	want := pieceExpected + junctionExpected

	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("expected terrain sampler weight %.12f, got %.12f", want, got)
	}
}

func TestVillageTerrainAdaptationCollectsRigidPiecesAndJunctions(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, chunkX, chunkZ := findPlannedStartForPlanner(t, g, "villages", 24)
	if start.terrainAdaptation != "beard_thin" {
		t.Fatalf("expected village terrain adaptation beard_thin, got %q", start.terrainAdaptation)
	}

	piecesWithJunctions := 0
	for _, piece := range start.pieces {
		if len(piece.junctions) != 0 {
			piecesWithJunctions++
		}
	}
	if piecesWithJunctions == 0 {
		t.Fatal("expected planned village pieces to record jigsaw junctions")
	}

	sampler := newStructureTerrainSampler(g, chunkX, chunkZ, -64, 319)
	if sampler == nil {
		t.Fatal("expected village chunk to collect a terrain sampler")
	}
	if len(sampler.pieces) == 0 {
		t.Fatal("expected village terrain sampler to include rigid pieces")
	}
	if len(sampler.junctions) == 0 {
		t.Fatal("expected village terrain sampler to include jigsaw junctions")
	}

	centerX := (start.rootOrigin[0]*2 + start.rootSize[0] - 1) / 2
	centerZ := (start.rootOrigin[2]*2 + start.rootSize[2] - 1) / 2
	if weight := sampler.sample(centerX, start.rootOrigin[1], centerZ); weight == 0 {
		t.Fatal("expected village terrain sampler to contribute non-zero density near the start")
	}
}

func TestVillageTerrainAdaptationChangesFinalDensity(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "villages", 24)

	centerX := (start.rootOrigin[0]*2 + start.rootSize[0] - 1) / 2
	centerZ := (start.rootOrigin[2]*2 + start.rootSize[2] - 1) / 2
	blockY := start.rootOrigin[1]
	chunkX := floorDiv(centerX, 16)
	chunkZ := floorDiv(centerZ, 16)

	terrainSampler := newStructureTerrainSampler(g, chunkX, chunkZ, -64, 319)
	if terrainSampler == nil {
		t.Fatal("expected village chunk to collect a terrain sampler")
	}

	root := g.rootIndex("final_density")
	flat := g.graph.NewFlatCacheGrid(chunkX, chunkZ, g.noises)
	column := g.graph.NewColumnContext(centerX, centerZ, g.noises, flat)
	ctx := gen.FunctionContext{BlockX: centerX, BlockY: blockY, BlockZ: centerZ}
	baseDensity := gen.EvalDensityScalar(g.graph, root, ctx, g.noises, flat, column, g.finalDensityScalar)
	adaptedEval := terrainSampler.scalarEvaluator(g, g.finalDensityScalar)
	adaptedDensity := adaptedEval(ctx, g.noises, flat, column)
	wantDelta := terrainSampler.sample(centerX, blockY, centerZ)
	gotDelta := adaptedDensity - baseDensity
	if math.Abs(gotDelta) <= 0 {
		t.Fatal("expected terrain adaptation to change final density near the village start")
	}
	if math.Abs(gotDelta-wantDelta) > 1e-12 {
		t.Fatalf("expected terrain adaptation to change density by %.12f, got %.12f", wantDelta, gotDelta)
	}
}

func TestPlacePlannedStructureWritesBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	planner, ok := g.findStructurePlanner("villages")
	if !ok {
		t.Fatal("load village structure planner")
	}
	surfaceSampler := newStructureHeightSampler(g, -64, 319)

	var (
		start foundStart
		found bool
	)
	for gridX := -8; gridX <= 8 && !found; gridX++ {
		for gridZ := -8; gridZ <= 8; gridZ++ {
			chunkPos := randomSpreadPotentialChunk(g.seed, planner.placement, gridX, gridZ)
			planned, exists := g.planStructureStart(planner, chunkPos, -64, 319, surfaceSampler)
			if !exists {
				continue
			}
			start = foundStart{planned: planned, chunkX: int(chunkPos[0]), chunkZ: int(chunkPos[1])}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("did not find a placeable village structure start")
	}

	c := chunk.New(0, cube.Range{-64, 319})
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	g.placePlannedStructure(c, biomes, start.chunkX, start.chunkZ, c.Range().Min(), c.Range().Max(), start.planned)

	nonAir := 0
	for y := start.planned.origin[1]; y <= start.planned.origin[1]+start.planned.size[1]-1; y++ {
		if y < c.Range().Min() || y > c.Range().Max() {
			continue
		}
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				if c.Block(uint8(x), int16(y), uint8(z), 0) != g.airRID {
					nonAir++
				}
			}
		}
	}
	if nonAir == 0 {
		t.Fatal("expected planned structure placement to write non-air blocks")
	}
}

func TestChooseStructureForPlannerRejectsInvalidVillageBiome(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	planner, ok := g.findStructurePlanner("villages")
	if !ok {
		t.Fatal("load village structure planner")
	}

	if _, ok := g.chooseStructureForPlanner(planner, gen.BiomeOcean, world.ChunkPos{0, 0}); ok {
		t.Fatal("expected villages to be rejected in ocean biomes")
	}
}

func TestChooseStructureForPlannerUsesExactVillageBiomeTags(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	planner, ok := g.findStructurePlanner("villages")
	if !ok {
		t.Fatal("load village structure planner")
	}

	candidate, ok := g.chooseStructureForPlanner(planner, gen.BiomeMeadow, world.ChunkPos{0, 0})
	if !ok {
		t.Fatal("expected village planner to allow meadow villages")
	}
	if candidate.structureName != "village_plains" {
		t.Fatalf("expected meadow village candidate to be village_plains, got %q", candidate.structureName)
	}
	if _, ok := g.chooseStructureForPlanner(planner, gen.BiomeSunflowerPlains, world.ChunkPos{0, 0}); ok {
		t.Fatal("expected sunflower plains villages to be rejected")
	}
	if _, ok := g.chooseStructureForPlanner(planner, gen.BiomeSnowySlopes, world.ChunkPos{0, 0}); ok {
		t.Fatal("expected snowy slopes villages to be rejected")
	}
}

func TestChooseStructureForPlannerUsesBiomeTagsForSingleCandidateJigsaws(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	outposts, ok := g.findStructurePlanner("pillager_outposts")
	if !ok {
		t.Fatal("load pillager outpost structure planner")
	}
	if candidate, ok := g.chooseStructureForPlanner(outposts, gen.BiomePlains, world.ChunkPos{0, 0}); !ok || candidate.structureName != "pillager_outpost" {
		t.Fatalf("expected plains pillager outpost candidate, got %+v ok=%v", candidate, ok)
	}
	if _, ok := g.chooseStructureForPlanner(outposts, gen.BiomeSwamp, world.ChunkPos{0, 0}); ok {
		t.Fatal("expected swamp pillager outposts to be rejected")
	}

	ancientCities, ok := g.findStructurePlanner("ancient_cities")
	if !ok {
		t.Fatal("load ancient city structure planner")
	}
	if candidate, ok := g.chooseStructureForPlanner(ancientCities, gen.BiomeDeepDark, world.ChunkPos{0, 0}); !ok || candidate.structureName != "ancient_city" {
		t.Fatalf("expected deep dark ancient city candidate, got %+v ok=%v", candidate, ok)
	}
	if _, ok := g.chooseStructureForPlanner(ancientCities, gen.BiomeDripstoneCaves, world.ChunkPos{0, 0}); ok {
		t.Fatal("expected non-deep-dark ancient cities to be rejected")
	}
}

func TestStructureCandidateAllowedUsesExactDirectBiomeTags(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	if !g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/end_city"}, gen.BiomeEndHighlands) {
		t.Fatal("expected end cities in end highlands")
	}
	if g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/end_city"}, gen.BiomeEndBarrens) {
		t.Fatal("expected end barrens to be rejected for end cities")
	}
	if !g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/ruined_portal_jungle"}, gen.BiomeSparseJungle) {
		t.Fatal("expected sparse jungle to match ruined_portal_jungle")
	}
	if g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/woodland_mansion"}, gen.BiomeForest) {
		t.Fatal("expected woodland mansion to reject plain forest")
	}
	if !g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/trial_chambers"}, gen.BiomePlains) {
		t.Fatal("expected trial chambers in plains")
	}
	if g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/trial_chambers"}, gen.BiomeDeepDark) {
		t.Fatal("expected deep dark to be rejected for trial chambers")
	}
	if !g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/bastion_remnant"}, gen.BiomeNetherWastes) {
		t.Fatal("expected bastion remnants in nether wastes")
	}
	if g.structureCandidateAllowed(structurePlannerCandidate{biomeTag: "has_structure/bastion_remnant"}, gen.BiomeBasaltDeltas) {
		t.Fatal("expected basalt deltas to be rejected for bastion remnants")
	}
}

func TestPillagerOutpostPlacementUsesVillageExclusionZone(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	outposts, ok := g.findStructurePlanner("pillager_outposts")
	if !ok {
		t.Fatal("load pillager outpost structure planner")
	}
	if outposts.placement.ExclusionZone.OtherSet != "villages" || outposts.placement.ExclusionZone.ChunkCount != 10 {
		t.Fatalf("expected pillager outposts exclusion zone to target villages within 10 chunks, got %+v", outposts.placement.ExclusionZone)
	}

	surfaceSampler := newStructureHeightSampler(g, -64, 319)
	for gridX := -32; gridX <= 32; gridX++ {
		for gridZ := -32; gridZ <= 32; gridZ++ {
			startChunk := randomSpreadPotentialChunk(g.seed, outposts.placement, gridX, gridZ)
			if !structurePlacementAllows(g.seed, outposts.placement, int(startChunk[0]), int(startChunk[1])) {
				continue
			}
			if !g.structurePlacementExcludedByOtherSet(outposts, startChunk, -64, 319, surfaceSampler) {
				continue
			}
			if _, exists := g.planStructureStart(outposts, startChunk, -64, 319, surfaceSampler); exists {
				t.Fatalf("expected outpost start at %v to be rejected by village exclusion zone", startChunk)
			}
			return
		}
	}
	t.Fatal("did not find an outpost candidate blocked by a village exclusion zone")
}

func TestResolveVillageTreePoolIncludesFeaturePlacement(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	pool, err := g.structureResolver.Pool("village/plains/trees")
	if err != nil {
		t.Fatalf("load village tree pool: %v", err)
	}
	if len(pool.entries) == 0 {
		t.Fatal("expected village tree pool entries")
	}
	for _, entry := range pool.entries {
		for _, feature := range entry.features {
			if feature.featureName == "oak" {
				return
			}
		}
	}
	t.Fatal("expected village tree pool to resolve oak feature placements")
}

func TestResolveTrialChamberPoolAliases(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	def, err := g.worldgen.Structure("trial_chambers")
	if err != nil {
		t.Fatalf("load trial chambers structure: %v", err)
	}
	jigsaw, err := def.Jigsaw()
	if err != nil {
		t.Fatalf("decode trial chambers jigsaw: %v", err)
	}

	aliases := resolveStructurePoolAliases(jigsaw.PoolAliases, cube.Pos{32, -24, 64}, g.seed)
	if len(aliases) == 0 {
		t.Fatal("expected resolved pool aliases")
	}

	assertAllowedAlias := func(alias string, allowed ...string) {
		value := aliases.lookup(alias)
		for _, target := range allowed {
			if value == target {
				return
			}
		}
		t.Fatalf("alias %q resolved to %q, expected one of %v", alias, value, allowed)
	}

	assertAllowedAlias(
		"trial_chambers/spawner/contents/ranged",
		"trial_chambers/spawner/ranged/skeleton",
		"trial_chambers/spawner/ranged/stray",
		"trial_chambers/spawner/ranged/poison_skeleton",
	)
	assertAllowedAlias(
		"trial_chambers/spawner/contents/slow_ranged",
		"trial_chambers/spawner/slow_ranged/skeleton",
		"trial_chambers/spawner/slow_ranged/stray",
		"trial_chambers/spawner/slow_ranged/poison_skeleton",
	)
	assertAllowedAlias(
		"trial_chambers/spawner/contents/melee",
		"trial_chambers/spawner/melee/zombie",
		"trial_chambers/spawner/melee/husk",
		"trial_chambers/spawner/melee/spider",
	)
	assertAllowedAlias(
		"trial_chambers/spawner/contents/small_melee",
		"trial_chambers/spawner/small_melee/slime",
		"trial_chambers/spawner/small_melee/cave_spider",
		"trial_chambers/spawner/small_melee/silverfish",
		"trial_chambers/spawner/small_melee/baby_zombie",
	)
}

func TestPlacePlannedStructureExecutesFeaturePoolElements(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	grassRID := world.BlockRuntimeID(block.Grass{})
	plainsRID := biomeRuntimeID(gen.BiomePlains)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, grassRID)
			for y := 0; y <= 15; y++ {
				c.SetBiome(uint8(x), int16(y), uint8(z), plainsRID)
			}
		}
	}

	start := plannedStructureStart{
		structureName: "test_structure",
		pieces: []plannedStructurePiece{{
			element: resolvedPoolElement{
				features: []structureFeaturePlacement{{featureName: "oak_checked"}},
			},
			origin: cube.Pos{8, 1, 8},
		}},
	}

	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomePlains)
	g.placePlannedStructure(c, biomes, 0, 0, c.Range().Min(), c.Range().Max(), start)
	if countTreeBlocks(c) == 0 {
		t.Fatal("expected structure feature pool element to place tree blocks")
	}
}

func TestPlanIglooStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)

	start, _, _ := findPlannedStartForPlanner(t, g, "igloos", 96)
	if start.templateName != "igloo/top" {
		t.Fatalf("expected igloo top template, got %q", start.templateName)
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned igloo pieces")
	}
}

func TestGeneratedIglooChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "igloos", 96)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated igloo chunk intersecting the planned igloo to contain structure blocks")
	}
}

func TestPlanBuriedTreasureStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "buried_treasures", 256)
	if start.structureName != "buried_treasure" {
		t.Fatalf("expected buried treasure structure, got %q", start.structureName)
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned buried treasure pieces")
	}
}

func TestGeneratedBuriedTreasureChunkContainsStructureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "buried_treasures", 256)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated buried treasure chunk intersecting the planned treasure to contain structure blocks")
	}
}

func TestBuriedTreasurePlacementWritesChest(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	surfaceSampler := newStructureHeightSampler(g, -64, 319)
	_, pieces, box, _, _, ok := g.buildBuriedTreasureStructure(0, 0, surfaceSampler, gen.RandomSpreadPlacement{Frequency: 1})
	if !ok || len(pieces) == 0 {
		t.Fatal("expected buried treasure plan")
	}

	start := plannedStructureStart{
		structureName: "buried_treasure",
		pieces:        pieces,
		origin:        cube.Pos{box.minX, box.minY, box.minZ},
		size:          [3]int{box.maxX - box.minX + 1, box.maxY - box.minY + 1, box.maxZ - box.minZ + 1},
	}
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), gen.BiomeBeach)
	g.placePlannedStructure(c, biomes, 0, 0, c.Range().Min(), c.Range().Max(), start)

	chestRID, ok := g.lookupTemplateBlock(structureLookupName("chest"), structureLookupProperties("chest", map[string]string{"facing": "north"}))
	if !ok {
		t.Fatal("expected chest block lookup")
	}
	foundChest := false
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				if c.Block(uint8(x), int16(y), uint8(z), 0) == chestRID {
					foundChest = true
					break
				}
			}
		}
	}
	if !foundChest {
		t.Fatal("expected buried treasure placement to write a chest")
	}
}

func TestGeneratedVillageChunksContainStructureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "villages", 24)
	palette := make(map[string]struct{}, 64)
	for _, piece := range start.pieces {
		for _, blockInfo := range piece.manualBlocks {
			switch blockInfo.state.Name {
			case "air", "minecraft:air", "structure_void", "minecraft:structure_void", "structure_block", "minecraft:structure_block":
				continue
			}
			palette[normalizeStructureTestStateName(blockInfo.state.Name)] = struct{}{}
		}
		for _, placement := range piece.element.placements {
			template, err := g.structureTemplates.Template(placement.templateName)
			if err != nil {
				continue
			}
			for _, state := range template.Palette {
				switch state.Name {
				case "minecraft:air", "minecraft:jigsaw", "minecraft:structure_void":
					continue
				}
				palette[state.Name] = struct{}{}
			}
		}
	}

	found := 0
	c := chunk.New(g.airRID, cube.Range{-64, 319})
	g.GenerateChunk(world.ChunkPos{int32(startChunkX), int32(startChunkZ)}, c)
	for y := max(start.origin.Y(), c.Range().Min()); y <= min(start.origin.Y()+start.size[1]-1, c.Range().Max()); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b := c.Block(uint8(x), int16(y), uint8(z), 0)
				blockValue, ok := world.BlockByRuntimeID(b)
				if !ok {
					continue
				}
				name, _ := blockValue.EncodeBlock()
				if _, ok := palette[name]; ok {
					found++
					break
				}
			}
			if found > 0 {
				break
			}
		}
		if found > 0 {
			break
		}
	}
	if found == 0 {
		t.Fatal("expected generated village start chunk to contain structure palette blocks")
	}
}

func TestGenerateColumnPersistsStructureMetadata(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "villages", 24)

	col := &chunk.Column{Chunk: chunk.New(g.airRID, cube.Range{-64, 319})}
	g.GenerateColumn(world.ChunkPos{int32(startChunkX), int32(startChunkZ)}, col)

	if len(col.StructureStarts) == 0 {
		t.Fatal("expected generated start chunk to contain structure starts")
	}
	if len(col.StructureRefs) == 0 {
		t.Fatal("expected generated start chunk to contain structure references")
	}
	foundVillageStart := false
	for _, entry := range col.StructureStarts {
		if entry.Structure == start.structureName {
			foundVillageStart = true
			break
		}
	}
	if !foundVillageStart {
		t.Fatalf("expected structure start metadata for %q", start.structureName)
	}

	refChunkX, refChunkZ := startChunkX, startChunkZ
	foundRefChunk := false
	for chunkX := floorDiv(start.origin.X(), 16); chunkX <= floorDiv(start.origin.X()+start.size[0]-1, 16) && !foundRefChunk; chunkX++ {
		for chunkZ := floorDiv(start.origin.Z(), 16); chunkZ <= floorDiv(start.origin.Z()+start.size[2]-1, 16); chunkZ++ {
			if chunkX == startChunkX && chunkZ == startChunkZ {
				continue
			}
			refChunkX, refChunkZ = chunkX, chunkZ
			foundRefChunk = true
			break
		}
	}
	if !foundRefChunk {
		t.Skip("planned structure only intersects its start chunk")
	}
	refCol := &chunk.Column{Chunk: chunk.New(g.airRID, cube.Range{-64, 319})}
	g.GenerateColumn(world.ChunkPos{int32(refChunkX), int32(refChunkZ)}, refCol)
	if len(refCol.StructureRefs) == 0 {
		t.Fatal("expected intersecting chunk to contain structure references")
	}
}

func TestStructurePlannersAreDimensionScoped(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	overworld := New(0)
	nether := NewForDimension(0, world.Nether)
	end := NewForDimension(0, world.End)

	if _, ok := overworld.findStructurePlanner("nether_fossils"); ok {
		t.Fatal("expected overworld to exclude Nether fossil planner")
	}
	if _, ok := overworld.findStructurePlanner("end_cities"); ok {
		t.Fatal("expected overworld to exclude End city planner")
	}
	if _, ok := nether.findStructurePlanner("nether_fossils"); !ok {
		t.Fatal("expected Nether to include Nether fossil planner")
	}
	if _, ok := nether.findStructurePlanner("end_cities"); ok {
		t.Fatal("expected Nether to exclude End city planner")
	}
	if _, ok := end.findStructurePlanner("end_cities"); !ok {
		t.Fatal("expected End to include End city planner")
	}
	if _, ok := end.findStructurePlanner("villages"); ok {
		t.Fatal("expected End to exclude village planner")
	}
}

func TestStructurePlannersIncludeImplementedRegistrySets(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	overworld := New(0)
	for _, setName := range []string{
		"ancient_cities",
		"buried_treasures",
		"igloos",
		"ocean_ruins",
		"pillager_outposts",
		"ruined_portals",
		"shipwrecks",
		"swamp_huts",
		"trail_ruins",
		"trial_chambers",
		"villages",
	} {
		if _, ok := overworld.findStructurePlanner(setName); !ok {
			t.Fatalf("expected overworld planner for implemented registry set %q", setName)
		}
	}
	for _, setName := range []string{
		"desert_pyramids",
		"jungle_temples",
		"mineshafts",
		"ocean_monuments",
		"strongholds",
		"woodland_mansions",
	} {
		if _, ok := overworld.findStructurePlanner(setName); ok {
			t.Fatalf("did not expect planner for unsupported overworld registry set %q", setName)
		}
	}

	nether := NewForDimension(0, world.Nether)
	for _, setName := range []string{"nether_complexes", "nether_fossils", "ruined_portals"} {
		if _, ok := nether.findStructurePlanner(setName); !ok {
			t.Fatalf("expected Nether planner for implemented registry set %q", setName)
		}
	}

	end := NewForDimension(0, world.End)
	if _, ok := end.findStructurePlanner("end_cities"); !ok {
		t.Fatal("expected End planner for end cities")
	}
}

func TestPlanNetherFossilStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	start, _, _ := findPlannedStartForPlanner(t, g, "nether_fossils", 64)
	if start.templateName == "" {
		t.Fatal("expected planned Nether fossil template")
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned Nether fossil pieces")
	}
}

func TestGeneratedNetherFossilChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "nether_fossils", 64)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, world.Nether.Range())
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, world.Nether.Range()) == 0 {
		t.Fatal("expected generated Nether fossil chunk intersecting the planned fossil to contain template palette blocks")
	}
}

func TestPlanShipwreckStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "shipwrecks", 128)
	if start.templateName == "" {
		t.Fatal("expected planned shipwreck template")
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned shipwreck pieces")
	}
}

func TestGeneratedShipwreckChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "shipwrecks", 128)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated shipwreck chunk intersecting the planned shipwreck to contain template palette blocks")
	}
}

func TestPlanPillagerOutpostStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "pillager_outposts", 24)
	if start.structureName != "pillager_outpost" {
		t.Fatalf("expected pillager outpost structure, got %q", start.structureName)
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned pillager outpost pieces")
	}
}

func TestGeneratedPillagerOutpostChunkContainsStructureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "pillager_outposts", 24)
	pos := firstPlacedStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319}, gen.BiomePlains)
	if countPlacedStructureBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}, gen.BiomePlains) == 0 {
		t.Fatal("expected generated pillager outpost chunk intersecting the planned outpost to contain structure blocks")
	}
}

func TestPlanOceanRuinStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "ocean_ruins", 128)
	if start.templateName == "" {
		t.Fatal("expected planned ocean ruin template")
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned ocean ruin pieces")
	}
}

func TestGeneratedOceanRuinChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "ocean_ruins", 128)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated ocean ruin chunk intersecting the planned ruin to contain template palette blocks")
	}
}

func TestPlanRuinedPortalStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "ruined_portals", 128)
	if start.templateName == "" {
		t.Fatal("expected planned ruined portal template")
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned ruined portal pieces")
	}
}

func TestGeneratedRuinedPortalChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "ruined_portals", 128)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated ruined portal chunk intersecting the planned portal to contain template palette blocks")
	}
}

func TestPlanSwampHutStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "swamp_huts", 256)
	if start.templateName == "" {
		t.Fatal("expected planned swamp hut template")
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned swamp hut pieces")
	}
}

func TestGeneratedSwampHutChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "swamp_huts", 256)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated swamp hut chunk intersecting the planned hut to contain template palette blocks")
	}
}

func TestSwampHutManualBlockStatesResolve(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "swamp_huts", 256)
	seen := make(map[string]struct{})
	for _, piece := range start.pieces {
		for _, blockInfo := range piece.manualBlocks {
			switch blockInfo.state.Name {
			case "air", "minecraft:air":
				continue
			}
			key := blockInfo.state.Name
			if len(blockInfo.state.Properties) != 0 {
				key += "|" + normalizeStructureTestStateProperties(blockInfo.state.Properties)
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if _, ok := g.lookupTemplateBlock(structureLookupName(blockInfo.state.Name), structureLookupProperties(blockInfo.state.Name, blockInfo.state.Properties)); !ok {
				t.Fatalf("expected swamp hut manual block state %q %+v to resolve", blockInfo.state.Name, blockInfo.state.Properties)
			}
		}
	}
}

func TestPlanNetherComplexStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	start, _, _ := findPlannedStartForPlanner(t, g, "nether_complexes", 96)
	if start.structureName != "bastion_remnant" {
		t.Fatalf("expected bastion remnant structure, got %q", start.structureName)
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned bastion remnant pieces")
	}
}

func TestGeneratedBastionRemnantChunkContainsStructureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "nether_complexes", 96)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, world.Nether.Range())
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, world.Nether.Range()) == 0 {
		t.Fatal("expected generated bastion remnant chunk intersecting the planned bastion to contain structure blocks")
	}
}

func TestPlanNetherRuinedPortalStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	start, _, _ := findPlannedStartForPlanner(t, g, "ruined_portals", 256)
	if start.structureName != "ruined_portal_nether" {
		t.Fatalf("expected Nether ruined portal structure, got %q", start.structureName)
	}
	if len(start.pieces) == 0 {
		t.Fatal("expected planned Nether ruined portal pieces")
	}
}

func TestGeneratedNetherRuinedPortalChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.Nether)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "ruined_portals", 256)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, world.Nether.Range())
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, world.Nether.Range()) == 0 {
		t.Fatal("expected generated Nether ruined portal chunk intersecting the planned portal to contain structure blocks")
	}
}

func TestPlanTrailRuinsStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "trail_ruins", 32)
	if start.structureName != "trail_ruins" {
		t.Fatalf("expected trail ruins structure, got %q", start.structureName)
	}
	if start.terrainAdaptation != "bury" {
		t.Fatalf("expected trail ruins terrain adaptation bury, got %q", start.terrainAdaptation)
	}
	if len(start.pieces) <= 1 {
		t.Fatalf("expected planned trail ruins to expand beyond the root template, got %d piece(s)", len(start.pieces))
	}
}

func TestGeneratedTrailRuinsChunkContainsStructureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "trail_ruins", 32)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated trail ruins chunk intersecting the planned ruins to contain structure blocks")
	}
}

func TestPlanTrialChambersStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, _, _ := findPlannedStartForPlanner(t, g, "trial_chambers", 32)
	if start.structureName != "trial_chambers" {
		t.Fatalf("expected trial chambers structure, got %q", start.structureName)
	}
	if start.terrainAdaptation != "encapsulate" {
		t.Fatalf("expected trial chambers terrain adaptation encapsulate, got %q", start.terrainAdaptation)
	}
	if len(start.pieces) <= 1 {
		t.Fatalf("expected planned trial chambers to expand beyond the root template, got %d piece(s)", len(start.pieces))
	}
}

func TestGeneratedTrialChambersChunkContainsStructureBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := New(0)
	start, startChunkX, startChunkZ := findPlannedStartForPlanner(t, g, "trial_chambers", 32)
	pos := firstStructureChunkContainingBlocks(t, g, start, startChunkX, startChunkZ, cube.Range{-64, 319})
	if countTemplatePaletteBlocksInChunk(t, g, start, pos, cube.Range{-64, 319}) == 0 {
		t.Fatal("expected generated trial chambers chunk intersecting the planned chambers to contain structure blocks")
	}
}

func TestPlanEndCityStructureStart(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.End)
	start, _, _ := findEndCityStartForTests(t, g)
	if start.templateName != "end_city/base_floor" {
		t.Fatalf("expected end city base floor template, got %q", start.templateName)
	}
	if len(start.pieces) <= 1 {
		t.Fatalf("expected planned End city to expand beyond the root template, got %d piece(s)", len(start.pieces))
	}
}

func TestGeneratedEndCityChunkContainsTemplateBlocks(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.End)
	start, startChunkX, startChunkZ := findEndCityStartForTests(t, g)
	if countTemplatePaletteBlocksInChunk(t, g, start, world.ChunkPos{int32(startChunkX), int32(startChunkZ)}, world.End.Range()) == 0 {
		t.Fatal("expected generated End city chunk to contain template palette blocks")
	}
}

func TestBuildEndCityStructureOnKnownOuterIslandChunk(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.End)
	candidates := []world.ChunkPos{
		{80, 80},
		{96, 96},
		{112, 80},
		{-80, 80},
		{-96, 96},
	}
	for _, pos := range candidates {
		rng := g.structureRNG("end_cities", pos)
		surfaceSampler := newStructureHeightSampler(g, world.End.Range()[0], world.End.Range()[1])
		templateName, pieces, _, _, _, ok := g.buildEndCityStructure(pos, int(pos[0])*16, int(pos[1])*16, surfaceSampler, &rng)
		if ok && templateName != "" && len(pieces) > 1 {
			return
		}
	}
	t.Fatal("expected End city builder to succeed on at least one known outer-island chunk")
}

func TestEndCityTemplatesDecode(t *testing.T) {
	finaliseBlocksOnce.Do(worldFinaliseBlockRegistry)

	g := NewForDimension(0, world.End)
	for _, name := range []string{
		"end_city/base_floor",
		"end_city/second_floor_1",
		"end_city/third_floor_1",
		"end_city/third_roof",
		"end_city/tower_base",
		"end_city/tower_piece",
		"end_city/tower_top",
	} {
		template, err := g.structureTemplates.Template(name)
		if err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if len(template.Blocks) == 0 {
			t.Fatalf("expected template blocks for %s", name)
		}
	}
}

type foundStart struct {
	planned plannedStructureStart
	chunkX  int
	chunkZ  int
}

func findPlannedStartForPlanner(t *testing.T, g Generator, plannerName string, maxGrid int) (plannedStructureStart, int, int) {
	return findPlannedStartForPlannerInGridRange(t, g, plannerName, -maxGrid, maxGrid, -maxGrid, maxGrid)
}

func findPlannedStartForPlannerInGridRange(t *testing.T, g Generator, plannerName string, minGridX, maxGridX, minGridZ, maxGridZ int) (plannedStructureStart, int, int) {
	t.Helper()

	start, chunkX, chunkZ, ok := tryFindPlannedStartForPlannerInGridRange(g, plannerName, minGridX, maxGridX, minGridZ, maxGridZ)
	if ok {
		return start, chunkX, chunkZ
	}
	t.Fatalf("did not find a planned start for %s in grid range x=[%d,%d] z=[%d,%d]", plannerName, minGridX, maxGridX, minGridZ, maxGridZ)
	return plannedStructureStart{}, 0, 0
}

func tryFindPlannedStartForPlannerInGridRange(g Generator, plannerName string, minGridX, maxGridX, minGridZ, maxGridZ int) (plannedStructureStart, int, int, bool) {
	planner, ok := g.findStructurePlanner(plannerName)
	if !ok {
		return plannedStructureStart{}, 0, 0, false
	}
	surfaceSampler := newStructureHeightSampler(g, -64, 319)
	for gridX := minGridX; gridX <= maxGridX; gridX++ {
		for gridZ := minGridZ; gridZ <= maxGridZ; gridZ++ {
			startChunk := randomSpreadPotentialChunk(g.seed, planner.placement, gridX, gridZ)
			start, exists := g.planStructureStart(planner, startChunk, -64, 319, surfaceSampler)
			if exists {
				return start, int(startChunk[0]), int(startChunk[1]), true
			}
		}
	}
	return plannedStructureStart{}, 0, 0, false
}

func findEndCityStartForTests(t *testing.T, g Generator) (plannedStructureStart, int, int) {
	t.Helper()

	ranges := [][4]int{
		{16, 64, 16, 64},
		{-64, -16, 16, 64},
		{16, 64, -64, -16},
		{-64, -16, -64, -16},
	}
	for _, r := range ranges {
		start, chunkX, chunkZ, ok := tryFindPlannedStartForPlannerInGridRange(g, "end_cities", r[0], r[1], r[2], r[3])
		if ok {
			return start, chunkX, chunkZ
		}
	}
	t.Fatal("did not find an End city planned start in the tested outer-island grid ranges")
	return plannedStructureStart{}, 0, 0
}

func firstStructureChunkContainingBlocks(t *testing.T, g Generator, start plannedStructureStart, defaultChunkX, defaultChunkZ int, r cube.Range) world.ChunkPos {
	t.Helper()

	minChunkX := floorDiv(start.origin.X(), 16)
	maxChunkX := floorDiv(start.origin.X()+start.size[0]-1, 16)
	minChunkZ := floorDiv(start.origin.Z(), 16)
	maxChunkZ := floorDiv(start.origin.Z()+start.size[2]-1, 16)

	candidates := make([]world.ChunkPos, 0, (maxChunkX-minChunkX+1)*(maxChunkZ-minChunkZ+1))
	if defaultChunkX >= minChunkX && defaultChunkX <= maxChunkX && defaultChunkZ >= minChunkZ && defaultChunkZ <= maxChunkZ {
		candidates = append(candidates, world.ChunkPos{int32(defaultChunkX), int32(defaultChunkZ)})
	}
	for chunkX := minChunkX; chunkX <= maxChunkX; chunkX++ {
		for chunkZ := minChunkZ; chunkZ <= maxChunkZ; chunkZ++ {
			if chunkX == defaultChunkX && chunkZ == defaultChunkZ {
				continue
			}
			candidates = append(candidates, world.ChunkPos{int32(chunkX), int32(chunkZ)})
		}
	}

	for _, pos := range candidates {
		if countTemplatePaletteBlocksInChunk(t, g, start, pos, r) > 0 {
			return pos
		}
	}
	return world.ChunkPos{int32(defaultChunkX), int32(defaultChunkZ)}
}

func firstPlacedStructureChunkContainingBlocks(t *testing.T, g Generator, start plannedStructureStart, defaultChunkX, defaultChunkZ int, r cube.Range, biome gen.Biome) world.ChunkPos {
	t.Helper()

	minChunkX := floorDiv(start.origin.X(), 16)
	maxChunkX := floorDiv(start.origin.X()+start.size[0]-1, 16)
	minChunkZ := floorDiv(start.origin.Z(), 16)
	maxChunkZ := floorDiv(start.origin.Z()+start.size[2]-1, 16)

	candidates := make([]world.ChunkPos, 0, (maxChunkX-minChunkX+1)*(maxChunkZ-minChunkZ+1))
	if defaultChunkX >= minChunkX && defaultChunkX <= maxChunkX && defaultChunkZ >= minChunkZ && defaultChunkZ <= maxChunkZ {
		candidates = append(candidates, world.ChunkPos{int32(defaultChunkX), int32(defaultChunkZ)})
	}
	for chunkX := minChunkX; chunkX <= maxChunkX; chunkX++ {
		for chunkZ := minChunkZ; chunkZ <= maxChunkZ; chunkZ++ {
			if chunkX == defaultChunkX && chunkZ == defaultChunkZ {
				continue
			}
			candidates = append(candidates, world.ChunkPos{int32(chunkX), int32(chunkZ)})
		}
	}

	for _, pos := range candidates {
		if countPlacedStructureBlocksInChunk(t, g, start, pos, r, biome) > 0 {
			return pos
		}
	}
	return world.ChunkPos{int32(defaultChunkX), int32(defaultChunkZ)}
}

func countTemplatePaletteBlocksInChunk(t *testing.T, g Generator, start plannedStructureStart, pos world.ChunkPos, r cube.Range) int {
	t.Helper()

	palette := make(map[string]struct{}, 64)
	for _, piece := range start.pieces {
		for _, blockInfo := range piece.manualBlocks {
			switch blockInfo.state.Name {
			case "air", "minecraft:air", "structure_void", "minecraft:structure_void", "structure_block", "minecraft:structure_block":
				continue
			}
			palette[normalizeStructureTestStateName(blockInfo.state.Name)] = struct{}{}
		}
		for _, placement := range piece.element.placements {
			template, err := g.structureTemplates.Template(placement.templateName)
			if err != nil {
				continue
			}
			for _, state := range template.Palette {
				switch state.Name {
				case "minecraft:air", "minecraft:jigsaw", "minecraft:structure_void", "minecraft:structure_block":
					continue
				}
				palette[state.Name] = struct{}{}
			}
		}
	}

	c := chunk.New(g.airRID, r)
	g.GenerateChunk(pos, c)
	found := 0
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				b, ok := world.BlockByRuntimeID(c.Block(uint8(x), int16(y), uint8(z), 0))
				if !ok {
					continue
				}
				name, _ := b.EncodeBlock()
				if _, ok := palette[name]; ok {
					found++
				}
			}
		}
	}
	return found
}

func countPlacedStructureBlocksInChunk(t *testing.T, g Generator, start plannedStructureStart, pos world.ChunkPos, r cube.Range, biome gen.Biome) int {
	t.Helper()

	c := chunk.New(g.airRID, r)
	biomes := filledTestBiomeVolume(c.Range().Min(), c.Range().Max(), biome)
	g.placePlannedStructure(c, biomes, int(pos[0]), int(pos[1]), c.Range().Min(), c.Range().Max(), start)

	found := 0
	for y := c.Range().Min(); y <= c.Range().Max(); y++ {
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				if c.Block(uint8(x), int16(y), uint8(z), 0) != g.airRID {
					found++
				}
			}
		}
	}
	return found
}

func normalizeStructureTestStateProperties(properties map[string]string) string {
	if len(properties) == 0 {
		return ""
	}
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make([]byte, 0, len(keys)*8)
	for _, key := range keys {
		out = append(out, key...)
		out = append(out, '=')
		out = append(out, properties[key]...)
		out = append(out, ';')
	}
	return string(out)
}

func normalizeStructureTestStateName(name string) string {
	if name == "" {
		return name
	}
	if name[0] == '#' {
		return name
	}
	if name[:min(len(name), len("minecraft:"))] == "minecraft:" {
		return name
	}
	return "minecraft:" + name
}
