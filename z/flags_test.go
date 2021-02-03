package z

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlag(t *testing.T) {
	opt := `bool_key=true; int-key=5; float-key=0.05; string_key=value; ;`
	sf := NewSuperFlag(opt)
	t.Logf("Got SuperFlag: %s\n", sf)

	def := `bool_key=false; int-key=0; float-key=1.0; string-key=; other-key=5`

	// bool-key and int-key should not be overwritten. Only other-key should be set.
	sf.MergeAndCheckDefault(def)

	c := func() {
		// Has a typo.
		NewSuperFlag("boolo-key=true").MergeAndCheckDefault(def)
	}
	require.Panics(t, c)
	require.Equal(t, true, sf.GetBool("bool-key"))
	require.Equal(t, uint64(5), sf.GetUint64("int-key"))
	require.Equal(t, "value", sf.GetString("string-key"))
	require.Equal(t, uint64(5), sf.GetUint64("other-key"))
}
