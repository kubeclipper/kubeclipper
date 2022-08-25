package framework

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/onsi/ginkgo"
	"k8s.io/client-go/util/homedir"

	cliconfig "github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// AfterEachActionFunc is a function that can be called after each test
type AfterEachActionFunc func(f *Framework, failed bool)

type Framework struct {
	BaseName string

	// beforeEachStarted indicates that BeforeEach has started
	beforeEachStarted bool

	// client is kubeclipper client
	Client *kc.Client

	// Timeouts contains the custom timeouts used during the test execution.
	Timeouts *TimeoutContext

	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle

	// afterEaches is a map of name to function to be called after each test.  These are not
	// cleared.  The call order is randomized so that no dependencies can grow between
	// the various afterEaches
	afterEaches map[string]AfterEachActionFunc
}

// NewFrameworkWithCustomTimeouts makes a framework with with custom timeouts.
func NewFrameworkWithCustomTimeouts(baseName string, timeouts *TimeoutContext) *Framework {
	f := NewDefaultFramework(baseName)
	f.Timeouts = timeouts
	return f
}

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	return NewFramework(baseName, nil)
}

// NewFramework creates a test framework.
func NewFramework(baseName string, kcClient *kc.Client) *Framework {
	f := &Framework{
		BaseName: baseName,
		Client:   kcClient,
		Timeouts: NewTimeoutContextWithDefaults(),
	}

	// f.AddAfterEach("dumpLogs", func(f *Framework, failed bool) {
	//	if !failed {
	//		return
	//	}
	//	if !TestContext.DumpLogsOnFailure {
	//		return
	//	}
	//	// TODO: dump logs
	// })

	ginkgo.BeforeEach(f.BeforeEach)
	ginkgo.AfterEach(f.AfterEach)

	return f
}

// AddAfterEach is a way to add a function to be called after every test.  The execution order is intentionally random
// to avoid growing dependencies.  If you register the same name twice, it is a coding error and will panic.
func (f *Framework) AddAfterEach(name string, fn AfterEachActionFunc) {
	if _, ok := f.afterEaches[name]; ok {
		panic(fmt.Sprintf("%q is already registered", name))
	}

	if f.afterEaches == nil {
		f.afterEaches = map[string]AfterEachActionFunc{}
	}
	f.afterEaches[name] = fn
}

// BeforeEach gets a client
func (f *Framework) BeforeEach() {
	f.beforeEachStarted = true
	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	f.cleanupHandle = AddCleanupAction(f.AfterEach)
	if f.Client == nil {
		// TODO: when config not exist, login first
		ginkgo.By("Creating a KubeClipper client")
		cfg, err := cliconfig.TryLoadFromFile(filepath.Join(homedir.HomeDir(), ".kc", "config"))
		ExpectNoError(err)
		f.Client, err = kc.FromConfig(*cfg)
		ExpectNoError(err)
	}
	// Check f.client is work
	_, err := f.Client.Version(context.TODO())
	ExpectNoError(err)
	// Other init
}

// AfterEach deletes the resource, after reading its events.
func (f *Framework) AfterEach() {
	// If BeforeEach never started AfterEach should be skipped.
	// Currently some tests under e2e/storage have this condition.
	if !f.beforeEachStarted {
		return
	}
	RemoveCleanupAction(f.cleanupHandle)

	// This should not happen. Given ClientSet is a public field a test must have updated it!
	// Error out early before any API calls during cleanup.
	if f.Client == nil {
		Failf("The framework ClientSet must not be nil at this point")
	}

	// run all aftereach functions in random order to ensure no dependencies grow
	for _, afterEachFn := range f.afterEaches {
		afterEachFn(f, ginkgo.CurrentGinkgoTestDescription().Failed)
	}

	// TODO: other check
}
