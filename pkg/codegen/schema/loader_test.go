package schema

import (
	"os"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/stretchr/testify/require"
)

func initLoader(b *testing.B, options PluginLoaderCacheOptions) ReferenceLoader {
	cwd, err := os.Getwd()
	contract.AssertNoError(err)
	sink := cmdutil.Diag()
	ctx, err := plugin.NewContext(sink, sink, nil, nil, cwd, nil, true, nil)
	contract.AssertNoError(err)
	loader := NewPluginLoaderWithOptions(ctx.Host, options)
	b.Cleanup(func() {
		err := loader.Close()
		require.NoError(b, err)
	})

	return loader
}

func BenchmarkLoadPackageReference(b *testing.B) {
	cacheWarmingLoader := initLoader(b, PluginLoaderCacheOptions{})
	// ensure the file cache exists for later tests:
	_, err := cacheWarmingLoader.LoadPackageReference("azure-native", nil)
	contract.AssertNoError(err)

	b.Run("full-load", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			loader := initLoader(b, PluginLoaderCacheOptions{})

			_, err := loader.LoadPackageReference("azure-native", nil)
			contract.AssertNoError(err)
		}
	})

	b.Run("full-cache", func(b *testing.B) {
		loader := initLoader(b, PluginLoaderCacheOptions{})

		b.StopTimer()
		_, err := loader.LoadPackageReference("azure-native", nil)
		contract.AssertNoError(err)
		b.StartTimer()

		for n := 0; n < b.N; n++ {
			_, err := loader.LoadPackageReference("azure-native", nil)
			contract.AssertNoError(err)
		}
	})

	b.Run("mmap-cache", func(b *testing.B) {
		// Disables in-memory cache (single instancing), retains mmap of files:
		loader := initLoader(b, PluginLoaderCacheOptions{
			disableEntryCache: true,
		})

		b.StopTimer()
		_, err := loader.LoadPackageReference("azure-native", nil)
		contract.AssertNoError(err)
		b.StartTimer()

		for n := 0; n < b.N; n++ {
			_, err := loader.LoadPackageReference("azure-native", nil)
			contract.AssertNoError(err)
		}
	})

	b.Run("file-cache", func(b *testing.B) {
		// Disables in-memory cache and mmaping of files:
		loader := initLoader(b, PluginLoaderCacheOptions{
			disableEntryCache: true,
			disableMmap:       true,
		})

		b.StopTimer()
		_, err := loader.LoadPackageReference("azure-native", nil)
		contract.AssertNoError(err)
		b.StartTimer()

		for n := 0; n < b.N; n++ {
			_, err := loader.LoadPackageReference("azure-native", nil)
			contract.AssertNoError(err)
		}
	})

	b.Run("no-cache", func(b *testing.B) {
		// Disables in-memory cache, mmaping, and using schema files:
		loader := initLoader(b, PluginLoaderCacheOptions{
			disableEntryCache: true,
			disableMmap:       true,
			disableFileCache:  true,
		})

		b.StopTimer()
		_, err := loader.LoadPackageReference("azure-native", nil)
		contract.AssertNoError(err)
		b.StartTimer()

		for n := 0; n < b.N; n++ {
			_, err := loader.LoadPackageReference("azure-native", nil)
			contract.AssertNoError(err)
		}
	})
}
