//
// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package pcidb

import (
	"testing"
)

func TestMergeOptions(t *testing.T) {
	// Verify the default values are set if no overrides are passed
	opts := mergeOptions()
	if opts.Chroot == nil {
		t.Fatalf("Expected opts.Chroot to be non-nil.")
	}
	if opts.CacheOnly == nil {
		t.Fatalf("Expected opts.CacheOnly to be non-nil.")
	}

	// Verify if we pass an override, that value is used not the default
	opts = mergeOptions(WithChroot("/override"))
	if opts.Chroot == nil {
		t.Fatalf("Expected opts.Chroot to be non-nil.")
	} else if *opts.Chroot != "/override" {
		t.Fatalf("Expected opts.Chroot to be /override.")
	}
}
