package scene

import (
	"math"

	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

// LoginCamera is the login3 pull-back (0x4b8d60 / 0x4bd680; Phoenix
// tools/reference/ep5_login.json and scripts/login_camera.gd). The
// previous eye Z feeds the next update; steps run at a fixed 60 Hz.
type LoginCamera struct {
	angle    float64
	reverse  bool
	Progress float64
	Eye      render.V3
	acc      float64
}

var (
	LoginTarget     = render.V3{441.2099914550781, 305.8699951171875, 495.3699951171875}
	loginInitialEye = render.V3{441.2099914550781, 365.8699951171875, 420.0}
	loginSettledZ   = 255.0
)

// loginProgressRate is per second (ep5_login.json); with the Z feedback it
// reproduces the
// native trajectory exactly (logincam_test.go).
const loginProgressRate = 0.0010000000474974513

func NewLoginCamera() *LoginCamera { return &LoginCamera{Eye: loginInitialEye} }

func (c *LoginCamera) step(dt float64) {
	prev := c.angle
	drive := 0.6 * math.Pi / 180 * dt
	if c.reverse {
		c.angle -= drive
		c.angle += math.Abs(c.angle) * dt * 0.7
		if prev-c.angle <= 0 {
			c.reverse = false
		}
	} else {
		c.angle += drive
		c.angle -= math.Abs(c.angle) * dt * 0.7
		if prev-c.angle >= 0 {
			c.reverse = true
		}
	}
	z := float64(c.Eye[2])
	radius := z + (loginSettledZ-z)*c.Progress - float64(LoginTarget[2])
	c.Eye = render.V3{
		LoginTarget[0] - float32(math.Sin(c.angle)*radius),
		LoginTarget[1] + 60,
		LoginTarget[2] + float32(math.Cos(c.angle)*radius),
	}
	c.Progress = math.Min(1, c.Progress+dt*loginProgressRate)
}

// Update advances in fixed 1/60 s steps.
func (c *LoginCamera) Update(dt float64) {
	c.acc += math.Min(dt, 0.1)
	for c.acc >= 1.0/60 {
		c.step(1.0 / 60)
		c.acc -= 1.0 / 60
	}
}

// Skip jumps to the settled pose (Escape in the original).
func (c *LoginCamera) Skip() {
	for i := 0; i < 3000 && c.Progress < 1; i++ {
		c.step(1.0 / 60)
	}
}
