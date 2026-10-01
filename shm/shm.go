package shm

import "sync/atomic"

type ShmData struct {
	Bmp851_temperature atomic.Int32  // temperature*100
	Bmp851_pressure    atomic.Uint32 // druck*10
}
