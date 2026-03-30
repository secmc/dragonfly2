package entity

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/generator/vanilla"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"math"
	"math/rand/v2"
	"time"
)

// NewEyeOfEnder creates a throwable eye of ender signal entity.
func NewEyeOfEnder(opts world.EntitySpawnOpts, owner world.Entity) *world.EntityHandle {
	conf := eyeOfEnderConf
	conf.Owner = owner.H()
	return opts.New(EyeOfEnderType, conf)
}

var eyeOfEnderConf = EyeOfEnderBehaviourConfig{}

type EyeOfEnderBehaviourConfig struct {
	Owner *world.EntityHandle
}

func (conf EyeOfEnderBehaviourConfig) Apply(data *world.EntityData) {
	data.Data = conf.New()
}

func (conf EyeOfEnderBehaviourConfig) New() *EyeOfEnderBehaviour {
	projectileConf := eyeOfEnderProjectileConf
	projectileConf.Owner = conf.Owner
	return &EyeOfEnderBehaviour{
		conf:       conf,
		projectile: projectileConf.New(),
		lifetime:   4 * time.Second,
	}
}

var eyeOfEnderProjectileConf = ProjectileBehaviourConfig{
	Gravity: 0.02,
	Drag:    0.01,
	Damage:  -1,
}

const eyeOfEnderStrongholdSearchChunks = 4096

type EyeOfEnderBehaviour struct {
	conf       EyeOfEnderBehaviourConfig
	projectile *ProjectileBehaviour
	target     mgl64.Vec3
	lifetime   time.Duration
	life       int
	resolved   bool
	hasTarget  bool
	survive    bool
	close      bool
}

func (b *EyeOfEnderBehaviour) Owner() *world.EntityHandle {
	return b.conf.Owner
}

func (b *EyeOfEnderBehaviour) Tick(e *Ent, tx *world.Tx) *Movement {
	if b.close {
		_ = e.Close()
		return nil
	}
	if !b.resolved {
		b.resolveTarget(tx, e)
	}
	if b.hasTarget {
		return b.tickGuided(e, tx)
	}

	m := b.projectile.Tick(e, tx)
	if m == nil {
		return nil
	}
	if e.Age() >= b.lifetime {
		b.close = true
	}
	return m
}

func (b *EyeOfEnderBehaviour) tickGuided(e *Ent, tx *world.Tx) *Movement {
	pos := e.data.Pos
	previousVel := e.data.Vel
	nextPos := pos.Add(previousVel)
	vel := eyeOfEnderUpdateDeltaMovement(previousVel, nextPos, b.target)
	rot := eyeOfEnderRotationFromVelocity(vel, e.data.Rot)

	e.data.Pos, e.data.Vel, e.data.Rot = nextPos, vel, rot
	b.life++
	if b.life > 80 {
		b.finishGuided(e, tx)
		return nil
	}
	return &Movement{
		v:        tx.Viewers(pos),
		e:        e,
		pos:      nextPos,
		vel:      vel,
		dpos:     nextPos.Sub(pos),
		dvel:     vel.Sub(previousVel),
		rot:      rot,
		onGround: false,
	}
}

type eyeOfEnderStructureLocator interface {
	LocateNearestPlannedStructureStart(setName string, origin cube.Pos, maxChunkDistance int) (vanilla.PlannedStructureInfo, bool)
}

func (b *EyeOfEnderBehaviour) resolveTarget(tx *world.Tx, e *Ent) {
	b.resolved = true

	locator, ok := tx.World().Generator().(eyeOfEnderStructureLocator)
	if !ok || tx.World().Dimension() != world.Overworld {
		return
	}
	info, ok := locator.LocateNearestPlannedStructureStart(
		"strongholds",
		cube.PosFromVec3(e.Position()),
		eyeOfEnderStrongholdSearchChunks,
	)
	if !ok {
		return
	}

	structureTarget := mgl64.Vec3{
		float64(info.Origin.X()),
		float64(info.Origin.Y()),
		float64(info.Origin.Z()),
	}
	b.target = eyeOfEnderSignalTarget(e.Position(), structureTarget)
	if e.data.Vel.LenSqr() < 1e-12 {
		e.data.Vel = eyeOfEnderUpdateDeltaMovement(zeroVec3, e.Position(), b.target)
	}
	b.hasTarget = true
	b.life = 0
	b.survive = rand.IntN(5) > 0
}

func eyeOfEnderSignalTarget(pos, target mgl64.Vec3) mgl64.Vec3 {
	delta := target.Sub(pos)
	delta[1] = 0
	dist := delta.Len()
	if dist > 12 {
		return pos.Add(mgl64.Vec3{
			delta[0] / dist * 12,
			8,
			delta[2] / dist * 12,
		})
	}
	return target
}

func eyeOfEnderUpdateDeltaMovement(oldMovement, pos, target mgl64.Vec3) mgl64.Vec3 {
	horizontalDelta := mgl64.Vec3{target[0] - pos[0], 0, target[2] - pos[2]}
	horizontalLength := math.Hypot(horizontalDelta[0], horizontalDelta[2])
	oldHorizontalSpeed := math.Hypot(oldMovement[0], oldMovement[2])
	wantedSpeed := oldHorizontalSpeed + (horizontalLength-oldHorizontalSpeed)*0.0025
	movementY := oldMovement[1]
	if horizontalLength < 1 {
		wantedSpeed *= 0.8
		movementY *= 0.8
	}
	wantedMovementY := -1.0
	if pos[1]-oldMovement[1] < target[1] {
		wantedMovementY = 1.0
	}
	if horizontalLength < 0.000001 {
		return mgl64.Vec3{0, movementY + (wantedMovementY-movementY)*0.015, 0}
	}
	return horizontalDelta.Mul(wantedSpeed / horizontalLength).Add(mgl64.Vec3{
		0,
		movementY + (wantedMovementY-movementY)*0.015,
		0,
	})
}

func eyeOfEnderRotationFromVelocity(vel mgl64.Vec3, fallback cube.Rotation) cube.Rotation {
	horizontal := math.Hypot(vel[0], vel[2])
	if horizontal < 0.001 && math.Abs(vel[1]) < 0.001 {
		return fallback
	}
	yaw := mgl64.RadToDeg(math.Atan2(-vel[0], vel[2]))
	pitch := mgl64.RadToDeg(-math.Atan2(vel[1], horizontal))
	return cube.Rotation{yaw, pitch}
}

func (b *EyeOfEnderBehaviour) finishGuided(e *Ent, tx *world.Tx) {
	tx.PlaySound(e.Position(), sound.ItemBreak{})
	if b.survive {
		tx.AddEntity(NewItem(world.EntitySpawnOpts{Position: e.Position()}, item.NewStack(item.EyeOfEnder{}, 1)))
	} else {
		tx.AddParticle(e.Position(), particle.EndermanTeleport{})
	}
	b.close = true
	_ = e.Close()
}

// EyeOfEnderType is a world.EntityType implementation for thrown eyes of
// ender.
var EyeOfEnderType eyeOfEnderType

type eyeOfEnderType struct{}

func (t eyeOfEnderType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return &Ent{tx: tx, handle: handle, data: data}
}

func (eyeOfEnderType) EncodeEntity() string { return "minecraft:eye_of_ender_signal" }
func (eyeOfEnderType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.125, 0, -0.125, 0.125, 0.25, 0.125)
}
func (eyeOfEnderType) DecodeNBT(_ map[string]any, data *world.EntityData) {
	data.Data = eyeOfEnderConf.New()
}
func (eyeOfEnderType) EncodeNBT(*world.EntityData) map[string]any { return nil }
