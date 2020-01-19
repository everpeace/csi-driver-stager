package image_test

import (
	"testing"
	"time"

	clock "k8s.io/utils/clock/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testDriverName = "test.image.stager.csi.k8s.io"
	fakeNow        = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock      = clock.NewFakeClock(fakeNow)
)

func TestImageStager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Image Stager Test Suite")
}
