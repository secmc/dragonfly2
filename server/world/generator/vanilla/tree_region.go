package vanilla

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	gen "github.com/df-mc/dragonfly/server/world/generator/vanilla/gen"
)

type treeDecorationRegion struct {
	centerChunkX int
	centerChunkZ int
	minY         int
	maxY         int
	slots        map[[2]int]treeDecorationRegionSlot
}

type treeDecorationRegionSlot struct {
	chunk  *chunk.Chunk
	biomes sourceBiomeVolume
}

func newTreeDecorationRegion(g Generator, c *chunk.Chunk, biomes sourceBiomeVolume, chunkX, chunkZ, minY, maxY int) *treeDecorationRegion {
	region := &treeDecorationRegion{
		centerChunkX: chunkX,
		centerChunkZ: chunkZ,
		minY:         minY,
		maxY:         maxY,
		slots:        make(map[[2]int]treeDecorationRegionSlot, 9),
	}
	region.set(chunkX, chunkZ, c, biomes)
	for sourceChunkX := chunkX - 1; sourceChunkX <= chunkX+1; sourceChunkX++ {
		for sourceChunkZ := chunkZ - 1; sourceChunkZ <= chunkZ+1; sourceChunkZ++ {
			if sourceChunkX == chunkX && sourceChunkZ == chunkZ {
				continue
			}
			neighbor := chunk.New(g.airRID, cube.Range{minY, maxY})
			neighborBiomes, _, _, _, _ := g.prepareChunkForDecoration(world.ChunkPos{int32(sourceChunkX), int32(sourceChunkZ)}, neighbor)
			region.set(sourceChunkX, sourceChunkZ, neighbor, neighborBiomes)
		}
	}
	return region
}

func (r *treeDecorationRegion) set(chunkX, chunkZ int, c *chunk.Chunk, biomes sourceBiomeVolume) {
	r.slots[[2]int{chunkX, chunkZ}] = treeDecorationRegionSlot{chunk: c, biomes: biomes}
}

func (r treeDecorationRegion) slot(chunkX, chunkZ int) (treeDecorationRegionSlot, bool) {
	slot, ok := r.slots[[2]int{chunkX, chunkZ}]
	return slot, ok
}

func (r treeDecorationRegion) contains(pos cube.Pos) bool {
	chunkX := floorDiv(pos[0], 16)
	chunkZ := floorDiv(pos[2], 16)
	return abs(chunkX-r.centerChunkX) <= 1 &&
		abs(chunkZ-r.centerChunkZ) <= 1 &&
		pos[1] >= r.minY &&
		pos[1] <= r.maxY
}

func (r treeDecorationRegion) chunkAtPos(pos cube.Pos) (*chunk.Chunk, bool) {
	slot, ok := r.slot(floorDiv(pos[0], 16), floorDiv(pos[2], 16))
	if !ok {
		return nil, false
	}
	return slot.chunk, true
}

func (g Generator) chunkForActiveTreePos(c *chunk.Chunk, pos cube.Pos) *chunk.Chunk {
	if g.activeTreeRegion == nil {
		return c
	}
	if regionChunk, ok := g.activeTreeRegion.chunkAtPos(pos); ok {
		return regionChunk
	}
	return c
}

func (g Generator) runPlacedTreeFeatureAcrossRegion(region *treeDecorationRegion, step gen.GenerationStep, featureName string, featureIndex int) {
	placed, err := g.features.Placed(featureName)
	if err != nil {
		return
	}

	regionGenerator := g
	regionGenerator.activeTreeRegion = region
	for sourceChunkZ := region.centerChunkZ - 1; sourceChunkZ <= region.centerChunkZ+1; sourceChunkZ++ {
		for sourceChunkX := region.centerChunkX - 1; sourceChunkX <= region.centerChunkX+1; sourceChunkX++ {
			slot, ok := region.slot(sourceChunkX, sourceChunkZ)
			if !ok {
				continue
			}

			origin := cube.Pos{sourceChunkX * 16, region.minY, sourceChunkZ * 16}
			rng := regionGenerator.featureRNG(regionGenerator.decorationSeed(sourceChunkX, sourceChunkZ), featureIndex, step)
			positions, ok := regionGenerator.applyPlacementModifiers(
				slot.chunk,
				slot.biomes,
				[]cube.Pos{origin},
				placed.Placement,
				featureName,
				sourceChunkX,
				sourceChunkZ,
				region.minY,
				region.maxY,
				&rng,
			)
			if !ok {
				continue
			}
			for _, pos := range positions {
				regionGenerator.executeConfiguredFeature(
					slot.chunk,
					slot.biomes,
					pos,
					placed.Feature,
					featureName,
					sourceChunkX,
					sourceChunkZ,
					region.minY,
					region.maxY,
					&rng,
					0,
				)
			}
		}
	}
}

func (g Generator) placedFeatureMayPlaceTrees(featureName string) bool {
	return g.placedFeatureRefMayPlaceTrees(gen.PlacedFeatureRef{Name: featureName}, map[string]struct{}{}, map[string]struct{}{})
}

func (g Generator) placedFeatureRefMayPlaceTrees(ref gen.PlacedFeatureRef, seenPlaced, seenConfigured map[string]struct{}) bool {
	if ref.Name != "" {
		if _, ok := seenPlaced[ref.Name]; ok {
			return false
		}
		seenPlaced[ref.Name] = struct{}{}
	}

	placed, err := g.features.ResolvePlaced(ref)
	if err != nil {
		return false
	}
	return g.configuredFeatureRefMayPlaceTrees(placed.Feature, seenPlaced, seenConfigured)
}

func (g Generator) configuredFeatureRefMayPlaceTrees(ref gen.ConfiguredFeatureRef, seenPlaced, seenConfigured map[string]struct{}) bool {
	if ref.Name != "" {
		if _, ok := seenConfigured[ref.Name]; ok {
			return false
		}
		seenConfigured[ref.Name] = struct{}{}
	}

	feature, err := g.features.ResolveConfigured(ref)
	if err != nil {
		return false
	}

	switch feature.Type {
	case "tree":
		return true
	case "random_selector":
		cfg, err := feature.RandomSelector()
		if err != nil {
			return false
		}
		for _, entry := range cfg.Features {
			if g.placedFeatureRefMayPlaceTrees(entry.Feature, seenPlaced, seenConfigured) {
				return true
			}
		}
		return g.placedFeatureRefMayPlaceTrees(cfg.Default, seenPlaced, seenConfigured)
	case "simple_random_selector":
		cfg, err := feature.SimpleRandomSelector()
		if err != nil {
			return false
		}
		for _, entry := range cfg.Features {
			if g.placedFeatureRefMayPlaceTrees(entry, seenPlaced, seenConfigured) {
				return true
			}
		}
		return false
	case "random_boolean_selector":
		cfg, err := feature.RandomBooleanSelector()
		if err != nil {
			return false
		}
		return g.placedFeatureRefMayPlaceTrees(cfg.FeatureTrue, seenPlaced, seenConfigured) ||
			g.placedFeatureRefMayPlaceTrees(cfg.FeatureFalse, seenPlaced, seenConfigured)
	case "random_patch", "flower":
		var (
			cfg gen.RandomPatchConfig
			err error
		)
		if feature.Type == "flower" {
			cfg, err = feature.Flower()
		} else {
			cfg, err = feature.RandomPatch()
		}
		if err != nil {
			return false
		}
		return g.placedFeatureRefMayPlaceTrees(cfg.Feature, seenPlaced, seenConfigured)
	case "vegetation_patch", "waterlogged_vegetation_patch":
		var (
			cfg gen.VegetationPatchConfig
			err error
		)
		if feature.Type == "waterlogged_vegetation_patch" {
			cfg, err = feature.WaterloggedVegetationPatch()
		} else {
			cfg, err = feature.VegetationPatch()
		}
		if err != nil {
			return false
		}
		return g.placedFeatureRefMayPlaceTrees(cfg.VegetationFeature, seenPlaced, seenConfigured)
	case "root_system":
		cfg, err := feature.RootSystem()
		if err != nil {
			return false
		}
		return g.placedFeatureRefMayPlaceTrees(cfg.Feature, seenPlaced, seenConfigured)
	default:
		return false
	}
}
