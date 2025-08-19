package component

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEnum(t *testing.T) {
	t.Parallel()

	t.Run("not", func(t *testing.T) {
		f := validateEnum([]string{"ltr", "rtl"})
		require.Error(t, f("what"))
	})

	t.Run("empty", func(t *testing.T) {
		f := validateEnum([]string{"ltr", "rtl"})
		require.NoError(t, f(""))
	})
}

func TestValidateColort(t *testing.T) {
	t.Parallel()

	t.Run("less-three-digit", func(t *testing.T) {
		f := validateColor()
		require.Error(t, f("#FF"))
	})

	t.Run("more-six-digit", func(t *testing.T) {
		f := validateColor()
		require.Error(t, f("#FFFFFFF"))
	})

	t.Run("valid-six", func(t *testing.T) {
		f := validateColor()
		require.NoError(t, f("#FFFFFF"))
	})

	t.Run("empty", func(t *testing.T) {
		f := validateColor()
		require.NoError(t, f(""))
	})

	t.Run("no-#-prefix", func(t *testing.T) {
		f := validateColor()
		require.Error(t, f("FFFFF"))
	})
}
