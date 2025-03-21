package dnfjson

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/osbuild/images/pkg/rpmmd"

	"github.com/stretchr/testify/assert"
)

func truncate(path string, size int64) {
	fp, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	if err := fp.Truncate(size); err != nil {
		panic(err)
	}
}

// create a test cache based on the config, where the config keys are file
// paths and the values are file sizes
func createTestCache(root string, config testCache) uint64 {
	var totalSize uint64
	for path, fi := range config {
		fullPath := filepath.Join(root, path)
		parPath := filepath.Dir(fullPath)
		if err := os.MkdirAll(parPath, 0770); err != nil {
			panic(err)
		}
		truncate(fullPath, int64(fi.size))
		mtime := time.Unix(fi.mtime, 0)
		if err := os.Chtimes(fullPath, mtime, mtime); err != nil {
			panic(err)
		}

		// if the path has multiple parts, touch the top level directory of the
		// element
		pathParts := strings.Split(path, "/")
		if len(pathParts) > 1 {
			top := pathParts[0]
			if err := os.Chtimes(filepath.Join(root, top), mtime, mtime); err != nil {
				panic(err)
			}
		}
		if len(path) >= 64 {
			// paths with shorter names will be ignored by the cache manager
			totalSize += fi.size
		}
	}

	// add directory sizes to total
	sizer := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			// don't count root
			return nil
		}
		if info.IsDir() {
			totalSize += uint64(info.Size())
		}
		return nil
	}
	if err := filepath.Walk(root, sizer); err != nil {
		panic(err)
	}

	return totalSize
}

type fileInfo struct {
	size  uint64
	mtime int64
}

type testCache map[string]fileInfo

var testCfgs = map[string]testCache{
	"rhel84-aarch64": { // real repo metadata file names and sizes
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc.solv":                                                                                                                      fileInfo{2095095, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-filenames.solvx":                                                                                                           fileInfo{14473401, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-33d346d177279673/repodata/gen/groups.xml":                                                                                  fileInfo{1419587, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-33d346d177279673/repodata/3eabd1122210e4def18ae4b96a18aa5bcc186abf2ec14e2e8f1c1bb1ab4d11da-modules.yaml.gz":                fileInfo{156314, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-33d346d177279673/repodata/90fd2e7463220a07457e76ae905e1bad754c29e22202bb3202c971a5ece28396-comps-AppStream.aarch64.xml.gz": fileInfo{199426, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-33d346d177279673/repodata/77a66c76b5f6ba51aaee6c0cf76d701601e8b622d1701d1781dabec434f27413-filelists.xml.gz":               fileInfo{14370201, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-33d346d177279673/repodata/1941c723c94218eed43eac3174aa94cefbe921e15547c39251a95895024207ca-primary.xml.gz":                 fileInfo{11439375, 100},
		"9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc-33d346d177279673/repodata/repomd.xml":                                                                                      fileInfo{13285, 100},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62.solv":                                                                                                                      fileInfo{1147863, 300},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62-filenames.solvx":                                                                                                           fileInfo{11133964, 300},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62-98177081b9162766/repodata/gen/groups.xml":                                                                                  fileInfo{1298102, 300},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62-98177081b9162766/repodata/d74783221709ab27d543c1cfc4c02562fde6edfaaaac33ac73a68ecf53188695-comps-BaseOS.aarch64.xml.gz":    fileInfo{174076, 300},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62-98177081b9162766/repodata/5ded48b4c9e238288130c6670d99f5febdb7273e4a31ac213836a15a2076514d-filelists.xml.gz":               fileInfo{11081612, 300},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62-98177081b9162766/repodata/8120caf8ebbb8c8b37f6f0dd027d866020ebe7acf9c9ce49ae9903b761986f0c-primary.xml.gz":                 fileInfo{1836471, 300},
		"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62-98177081b9162766/repodata/repomd.xml":                                                                                      fileInfo{12817, 300},
	},
	"fake-real": { // fake but resembling real data
		"3333333333333333333333333333333333333333333333333333333333333333.solv":           fileInfo{100, 0},
		"3333333333333333333333333333333333333333333333333333333333333333-filenames.solv": fileInfo{200, 0},
		"3333333333333333333333333333333333333333333333333333333333333333.whatever":       fileInfo{110, 0},
		"3333333333333333333333333333333333333333333333333333333333333333/repodata/a":     fileInfo{1000, 0},
		"3333333333333333333333333333333333333333333333333333333333333333/repodata/b":     fileInfo{3829, 0},
		"3333333333333333333333333333333333333333333333333333333333333333/repodata/c":     fileInfo{831989, 0},
		"2222222222222222222222222222222222222222222222222222222222222222.solv":           fileInfo{120, 2},
		"2222222222222222222222222222222222222222222222222222222222222222-filenames.solv": fileInfo{232, 2},
		"2222222222222222222222222222222222222222222222222222222222222222.whatever":       fileInfo{110, 2},
		"2222222222222222222222222222222222222222222222222222222222222222/repodata/a":     fileInfo{1000, 2},
		"2222222222222222222222222222222222222222222222222222222222222222/repodata/b":     fileInfo{3829, 2},
		"2222222222222222222222222222222222222222222222222222222222222222/repodata/c":     fileInfo{831989, 2},
		"1111111111111111111111111111111111111111111111111111111111111111.solv":           fileInfo{105, 4},
		"1111111111111111111111111111111111111111111111111111111111111111-filenames.solv": fileInfo{200, 4},
		"1111111111111111111111111111111111111111111111111111111111111111.whatever":       fileInfo{110, 4},
		"1111111111111111111111111111111111111111111111111111111111111111/repodata/a":     fileInfo{2390, 4},
		"1111111111111111111111111111111111111111111111111111111111111111/repodata/b":     fileInfo{1234890, 4},
		"1111111111111111111111111111111111111111111111111111111111111111/repodata/c":     fileInfo{483, 4},
	},
	"completely-fake": { // just a mess of files (including files without a repo ID)
		"somefile": fileInfo{192, 10291920},
		"yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy-repofiley":  fileInfo{29384, 11},
		"yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy-repofiley2": fileInfo{293, 31},
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-repofile":   fileInfo{29384, 30},
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-repofileb":  fileInfo{293, 45},
	},
}

type testCase struct {
	cache              testCache
	maxSize            uint64
	minSizeAfterShrink uint64
	repoIDsAfterShrink []string
}

func getRepoIDs(ct testCache) []string {
	idMap := make(map[string]bool)
	ids := make([]string, 0)
	for path := range ct {
		if len(path) >= 64 {
			id := path[:64]
			if !idMap[id] {
				idMap[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func TestCacheRead(t *testing.T) {
	assert := assert.New(t)
	for name, cfg := range testCfgs {
		t.Run(name, func(t *testing.T) {
			testCacheRoot := t.TempDir()
			// Cache is now per-distro, use the name of the config as a distro name
			s := createTestCache(filepath.Join(testCacheRoot, name), cfg)

			// Cache covers all distros, pass in top directory
			cache := newRPMCache(testCacheRoot, 1048576) // 1 MiB, but doesn't matter for this test

			nrepos := len(getRepoIDs(cfg))
			assert.Equal(s, cache.size)
			assert.Equal(nrepos, len(cache.repoElements))
			assert.Equal(nrepos, len(cache.repoRecency))
		})
	}
}

func TestMultiDirCacheRead(t *testing.T) {
	assert := assert.New(t)
	t.Run("MultiDir", func(t *testing.T) {
		testCacheRoot := t.TempDir()
		// Cache is now per-distro, use the name of the config as a distro name
		size1 := createTestCache(filepath.Join(testCacheRoot, "rhel84-aarch64"), testCfgs["rhel84-aarch64"])
		size2 := createTestCache(filepath.Join(testCacheRoot, "fake-real"), testCfgs["fake-real"])

		// Cache covers all distros, pass in top directory
		cache := newRPMCache(testCacheRoot, 1048576) // 1 MiB, but doesn't matter for this test

		nrepos := len(getRepoIDs(testCfgs["rhel84-aarch64"])) + len(getRepoIDs(testCfgs["fake-real"]))
		assert.Equal(size1+size2, cache.size)
		assert.Equal(nrepos, len(cache.repoElements))
		assert.Equal(nrepos, len(cache.repoRecency))

		// Check sorting by mtime
		var last int64 = -1
		for _, f := range cache.repoRecency {
			var cur int64
			if val, ok := testCfgs["rhel84-aarch64"][f]; ok {
				cur = val.mtime
			}
			if val, ok := testCfgs["fake-real"][f]; ok {
				cur = val.mtime
			}
			assert.GreaterOrEqual(cur, last)
			last = cur
		}
	})
}

func sizeSum(cfg testCache, repoIDFilter ...string) uint64 {
	var sum uint64
	for path, info := range cfg {
		if len(path) < 64 {
			continue
		}
		rid := path[:64]
		if len(repoIDFilter) == 0 || (len(repoIDFilter) > 0 && strSliceContains(repoIDFilter, rid)) {
			sum += info.size
		}
	}
	return sum
}

func TestCacheCleanup(t *testing.T) {
	rhelRecentRepoSize := sizeSum(testCfgs["rhel84-aarch64"], "df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62")
	rhelTotalRepoSize := sizeSum(testCfgs["rhel84-aarch64"])

	fakeRealSize2 := sizeSum(testCfgs["fake-real"], "2222222222222222222222222222222222222222222222222222222222222222")
	fakeRealSize3 := sizeSum(testCfgs["fake-real"], "1111111111111111111111111111111111111111111111111111111111111111")

	testCases := map[string]testCase{
		// max size 1 byte -> clean will delete everything
		"fake-real-full-delete": {
			cache:              testCfgs["fake-real"],
			maxSize:            1,
			minSizeAfterShrink: 0,
		},
		"rhel-full-delete": {
			cache:              testCfgs["rhel84-aarch64"],
			maxSize:            1,
			minSizeAfterShrink: 0,
		},
		"completely-fake-full-delete": {
			cache:              testCfgs["completely-fake"],
			maxSize:            1,
			minSizeAfterShrink: 0,
		},
		"completely-fake-full-delete-2": {
			cache:              testCfgs["completely-fake"],
			maxSize:            100,
			minSizeAfterShrink: 0,
		},
		// max size a bit larger than most recent repo -> clean will delete older repos
		"completely-fake-half-delete": {
			cache:              testCfgs["completely-fake"],
			maxSize:            29384 + 293 + 1,                                                              // one byte larger than the files of one repo
			minSizeAfterShrink: 29384 + 293,                                                                  // size of files from one repo
			repoIDsAfterShrink: []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, // most recent repo timestamp (45)
		},
		"rhel-half-delete": {
			cache:              testCfgs["rhel84-aarch64"],
			maxSize:            rhelRecentRepoSize + 102400,                                                  // most recent repo file sizes + 100k buffer (for directories)
			minSizeAfterShrink: rhelRecentRepoSize,                                                           // after shrink it should be at least as big as the most recent repo
			repoIDsAfterShrink: []string{"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62"}, // most recent repo timestamp (45)
		},
		"fake-real-delete-1": {
			cache:              testCfgs["fake-real"],
			maxSize:            fakeRealSize3 + fakeRealSize2 + 102400,
			minSizeAfterShrink: fakeRealSize3 + fakeRealSize2,
			repoIDsAfterShrink: []string{"1111111111111111111111111111111111111111111111111111111111111111", "2222222222222222222222222222222222222222222222222222222222222222"},
		},
		"fake-real-delete-2": {
			cache:              testCfgs["fake-real"],
			maxSize:            fakeRealSize3 + 102400,
			minSizeAfterShrink: fakeRealSize3,
			repoIDsAfterShrink: []string{"1111111111111111111111111111111111111111111111111111111111111111"},
		},
		// max size is huge -> clean wont delete anything
		"rhel-no-delete": {
			cache:              testCfgs["rhel84-aarch64"],
			maxSize:            45097156608, // 42 GiB
			minSizeAfterShrink: rhelTotalRepoSize,
			repoIDsAfterShrink: []string{"df2665154150abf76f4d86156228a75c39f3f31a79d4a861d76b1edd89814b62", "9adf133053f0691a0ec12e73cbf1875a90c9268b4f09162fc3387fd76ecb3bcc"},
		},
	}

	for name, cfg := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			testCacheRoot := t.TempDir()
			// Cache is now per-distro, use the name of the config as a distro name
			createTestCache(filepath.Join(testCacheRoot, name), cfg.cache)

			// Cache covers all distros, pass in top directory
			cache := newRPMCache(testCacheRoot, cfg.maxSize)

			err := cache.shrink()
			assert.NoError(err)

			// it's hard to predict the exact size after shrink because of directory sizes
			// so let's just check that the new size is between min and max
			assert.LessOrEqual(cfg.minSizeAfterShrink, cache.size)
			assert.Greater(cfg.maxSize, cache.size)
			assert.Equal(len(cfg.repoIDsAfterShrink), len(cache.repoElements))
			for _, id := range cfg.repoIDsAfterShrink {
				assert.Contains(cache.repoElements, id)
			}
		})
	}
}

// Mock package list to use in testing
var PackageList = rpmmd.PackageList{
	rpmmd.Package{
		Name:        "package0",
		Summary:     "package summary",
		Description: "package description",
		URL:         "https://package-url/",
		Epoch:       0,
		Version:     "1.0.0",
		Release:     "3",
		Arch:        "x86_64",
		License:     "MIT",
	},
}

func TestDNFCacheStoreGet(t *testing.T) {
	cache := NewDNFCache(1 * time.Second)
	assert.Equal(t, cache.timeout, 1*time.Second)
	assert.NotNil(t, cache.RWMutex)

	cache.Store("notreallyahash", PackageList)
	assert.Equal(t, 1, len(cache.results))
	pkgs, ok := cache.Get("notreallyahash")
	assert.True(t, ok)
	assert.Equal(t, "package0", pkgs[0].Name)
}

func TestDNFCacheTimeout(t *testing.T) {
	cache := NewDNFCache(1 * time.Second)
	cache.Store("notreallyahash", PackageList)
	_, ok := cache.Get("notreallyahash")
	assert.True(t, ok)
	time.Sleep(2 * time.Second)
	_, ok = cache.Get("notreallyahash")
	assert.False(t, ok)
}

func TestDNFCacheCleanup(t *testing.T) {
	cache := NewDNFCache(1 * time.Second)
	cache.Store("notreallyahash", PackageList)
	time.Sleep(2 * time.Second)
	assert.Equal(t, 1, len(cache.results))
	cache.CleanCache()
	assert.Equal(t, 0, len(cache.results))
	_, ok := cache.Get("notreallyahash")
	assert.False(t, ok)
}

func TestCleanupOldCacheDirs(t *testing.T) {
	// Run the cleanup without the cache present and with dummy distro names
	CleanupOldCacheDirs("/var/tmp/test-no-cache-rpmmd/", []string{"fedora-40", "fedora-41"})

	testCacheRoot := t.TempDir()
	// Make all the test caches under root, using their keys as a distro name.
	var distros []string
	for name, cfg := range testCfgs {
		// Cache is now per-distro, use the name of the config as a distro name
		createTestCache(filepath.Join(testCacheRoot, name), cfg)
		distros = append(distros, name)
	}
	sort.Strings(distros)

	// Add the content of the 'fake-real' cache to the top directory
	// this will be used to simulate an old cache without distro subdirs
	createTestCache(testCacheRoot, testCfgs["fake-real"])

	CleanupOldCacheDirs(testCacheRoot, distros)

	// The fake-real files under the root directory should all be gone.
	for path := range testCfgs["fake-real"] {
		_, err := os.Stat(filepath.Join(testCacheRoot, path))
		assert.NotNil(t, err)
	}

	// The distro cache files should all still be present
	for name, cfg := range testCfgs {
		for path := range cfg {
			_, err := os.Stat(filepath.Join(testCacheRoot, name, path))
			assert.Nil(t, err)
		}
	}

	// Remove the fake-real distro from the list
	// This simulates retiring an older distribution and cleaning up its cache
	distros = []string{}
	for name := range testCfgs {
		if name == "fake-real" {
			continue
		}
		distros = append(distros, name)
	}
	// Cleanup should now remove the fake-real subdirectory and files
	CleanupOldCacheDirs(testCacheRoot, distros)

	// The remaining distro's cache files should all still be present
	for _, name := range distros {
		for path := range testCfgs[name] {
			_, err := os.Stat(filepath.Join(testCacheRoot, name, path))
			assert.Nil(t, err)
		}
	}

	// But the fake-real ones should be gone
	for path := range testCfgs["fake-real"] {
		_, err := os.Stat(filepath.Join(testCacheRoot, "fake-real", path))
		assert.NotNil(t, err)
	}
}
