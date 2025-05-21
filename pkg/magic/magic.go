package magic

type MagicString string

const (
	ARM64MagicOffset             = 56
	ARM64Magic       MagicString = "ARM\x64"
)

func (m MagicString) Bytes() []byte { return []byte(m) }
