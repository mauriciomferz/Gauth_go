package claims

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaims(t *testing.T) {
	claims := NewClaims()

	t.Run("String Claims", func(t *testing.T) {
		err := claims.Set("name", "John")
		assert.NoError(t, err)

		val, err := claims.GetString("name")
		assert.NoError(t, err)
		assert.Equal(t, "John", val)

		_, err = claims.GetString("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Int Claims", func(t *testing.T) {
		err := claims.Set("age", int64(25))
		assert.NoError(t, err)

		val, err := claims.GetInt("age")
		assert.NoError(t, err)
		assert.Equal(t, int64(25), val)

		_, err = claims.GetInt("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Float Claims", func(t *testing.T) {
		err := claims.Set("score", 98.5)
		assert.NoError(t, err)

		val, err := claims.GetFloat("score")
		assert.NoError(t, err)
		assert.Equal(t, 98.5, val)

		_, err = claims.GetFloat("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Bool Claims", func(t *testing.T) {
		err := claims.Set("active", true)
		assert.NoError(t, err)

		val, err := claims.GetBool("active")
		assert.NoError(t, err)
		assert.True(t, val)

		_, err = claims.GetBool("nonexistent")
		assert.Error(t, err)
	})

	t.Run("String Slice Claims", func(t *testing.T) {
		roles := []string{"admin", "user"}
		err := claims.Set("roles", roles)
		assert.NoError(t, err)

		val, err := claims.GetStringSlice("roles")
		assert.NoError(t, err)
		assert.Equal(t, roles, val)

		_, err = claims.GetStringSlice("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Int Slice Claims", func(t *testing.T) {
		nums := []int64{1, 2, 3}
		err := claims.Set("numbers", nums)
		assert.NoError(t, err)

		val, err := claims.GetIntSlice("numbers")
		assert.NoError(t, err)
		assert.Equal(t, nums, val)

		_, err = claims.GetIntSlice("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Float Slice Claims", func(t *testing.T) {
		scores := []float64{98.5, 99.0, 97.5}
		err := claims.Set("scores", scores)
		assert.NoError(t, err)

		val, err := claims.GetFloatSlice("scores")
		assert.NoError(t, err)
		assert.Equal(t, scores, val)

		_, err = claims.GetFloatSlice("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Has Claims", func(t *testing.T) {
		assert.True(t, claims.Has("name"))
		assert.False(t, claims.Has("nonexistent"))
	})

	t.Run("Delete Claims", func(t *testing.T) {
		claims.Delete("name")
		assert.False(t, claims.Has("name"))
	})

	t.Run("Clear Claims", func(t *testing.T) {
		claims.Clear()
		assert.Equal(t, 0, claims.Len())
	})

	t.Run("Keys", func(t *testing.T) {
		claims.Clear()
		err := claims.Set("key1", "value1")
		assert.NoError(t, err)
		err = claims.Set("key2", "value2")
		assert.NoError(t, err)

		keys := claims.Keys()
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "key1")
		assert.Contains(t, keys, "key2")
	})

	t.Run("Invalid Types", func(t *testing.T) {
		err := claims.Set("invalid", struct{}{})
		assert.Error(t, err)
	})

	t.Run("Type Mismatch", func(t *testing.T) {
		err := claims.Set("mixed", "string")
		assert.NoError(t, err)

		_, err = claims.GetInt("mixed")
		assert.Error(t, err)
		_, err = claims.GetFloat("mixed")
		assert.Error(t, err)
		_, err = claims.GetBool("mixed")
		assert.Error(t, err)
		_, err = claims.GetStringSlice("mixed")
		assert.Error(t, err)
	})
}

func BenchmarkClaimOperations(b *testing.B) {
	claims := NewClaims()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = claims.Set("key", "value")
		}
	})

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = claims.GetString("key")
		}
	})

	b.Run("Has", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = claims.Has("key")
		}
	})
}
