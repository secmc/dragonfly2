package entity

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/generator/vanilla"
	"github.com/go-gl/mathgl/mgl64"
	"math"
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

type EyeOfEnderBehaviour struct {
	conf       EyeOfEnderBehaviourConfig
	projectile *ProjectileBehaviour
	target     mgl64.Vec3
	lifetime   time.Duration
	resolved   bool
	hasTarget  bool
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
		b.resolveTarget(tx, e.Position())
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
	vel := eyeOfEnderVelocityToward(pos, previousVel, b.target)
	nextPos := pos.Add(vel)
	rot := eyeOfEnderRotationFromVelocity(vel, e.data.Rot)

	e.data.Pos, e.data.Vel, e.data.Rot = nextPos, vel, rot
	delta := b.target.Sub(nextPos)
	delta[1] = 0
	if delta.Len() < 12 {
		b.close = true
	}
	if e.Age() >= b.lifetime {
		b.close = true
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

func (b *EyeOfEnderBehaviour) resolveTarget(tx *world.Tx, pos mgl64.Vec3) {
	b.resolved = true

	locator, ok := tx.World().Generator().(eyeOfEnderStructureLocator)
	if !ok || tx.World().Dimension() != world.Overworld {
		return
	}
	info, ok := locator.LocateNearestPlannedStructureStart(
		"strongholds",
		cube.PosFromVec3(pos),
		4096,
	)
	if !ok {
		return
	}

	sizeX := max(info.Size[0], 1)
	sizeZ := max(info.Size[2], 1)
	b.target = mgl64.Vec3{
		float64(info.Origin.X()) + float64(sizeX)/2,
		pos[1],
		float64(info.Origin.Z()) + float64(sizeZ)/2,
	}
	b.hasTarget = true
}

func eyeOfEnderVelocityToward(pos, vel, target mgl64.Vec3) mgl64.Vec3 {
	delta := target.Sub(pos)
	delta[1] = 0
	dist := delta.Len()
	if dist < 0.001 {
		return vel.Mul(0.8)
	}

	dir := delta.Mul(1 / dist)
	speed := clampFloat(0.55+dist/96.0, 0.7, 1.6)
	desiredY := 0.08
	if dist < 64 {
		desiredY = 0.05
	}
	if dist < 24 {
		desiredY = 0.02
	}
	desired := mgl64.Vec3{dir[0] * speed, desiredY, dir[2] * speed}
	if vel.Len() == 0 {
		return desired
	}
	return vel.Mul(0.72).Add(desired.Mul(0.28))
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

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
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
