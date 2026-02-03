package svg

import "github.com/gogpu/gg/recording"

func init() {
	recording.Register("svg", func() recording.Backend {
		return NewBackend()
	})
}
