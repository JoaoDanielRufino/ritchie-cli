package init

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/ZupIT/ritchie-cli/functional"
)

func TestRitSingleInit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rit Suite Init")
}

var _ = Describe("RitSingleInit", func() {
	BeforeEach(func() {
		functional.RitClearConfigs()
	})

	scenariosCore := functional.LoadScenarios("init_feature.json")

	DescribeTable("When running command without init",
		func(scenario functional.Scenario) {
			out, err := scenario.RunSteps()
			Expect(err).To(Succeed())
			Expect(out).To(ContainSubstring(scenario.Result))
		},

		Entry("Show context", scenariosCore[0]),
		Entry("Set context", scenariosCore[1]),
		Entry("Delete context", scenariosCore[2]),
		Entry("Add new repo", scenariosCore[3]),
		Entry("List repo", scenariosCore[4]),
		Entry("Delete repo", scenariosCore[5]),
		Entry("Set Credential", scenariosCore[6]),
		Entry("Update repo", scenariosCore[7]),
		Entry("Do init", scenariosCore[8]),
	)

})
