package ondemand_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOndemand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ondemand Suite")
}
