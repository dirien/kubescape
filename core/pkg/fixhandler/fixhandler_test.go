package fixhandler

import (
	"os"
	"path/filepath"
	"testing"

	logger "github.com/kubescape/go-logger"
	metav1 "github.com/kubescape/kubescape/v2/core/meta/datastructures/v1"
	reporthandlingv2 "github.com/kubescape/opa-utils/reporthandling/v2"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"gopkg.in/op/go-logging.v1"
)

type indentationTestCase struct {
	inputFile      string
	yamlExpression string
	expectedFile   string
}

func NewFixHandlerMock() (*FixHandler, error) {
	backendLoggerLeveled := logging.AddModuleLevel(logging.NewLogBackend(logger.L().GetWriter(), "", 0))
	backendLoggerLeveled.SetLevel(logging.ERROR, "")
	yqlib.GetLogger().SetBackend(backendLoggerLeveled)

	return &FixHandler{
		fixInfo:       &metav1.FixInfo{},
		reportObj:     &reporthandlingv2.PostureReport{},
		localBasePath: "",
	}, nil
}

func getTestdataPath() string {
	currentDir, _ := os.Getwd()
	return filepath.Join(currentDir, "testdata")
}

func getTestCases() []indentationTestCase {
	indentationTestCases := []indentationTestCase{
		// Insertion Scenarios
		{
			"inserts/tc-01-00-input.yaml",
			"select(di==0).spec.containers[0].securityContext.allowPrivilegeEscalation |= false",
			"inserts/tc-01-01-expected.yaml",
		},
		{
			"inserts/tc-02-00-input.yaml",
			"select(di==0).spec.containers[0].securityContext.capabilities.drop += [\"NET_RAW\"]",
			"inserts/tc-02-01-expected.yaml",
		},
		{
			"inserts/tc-03-00-input.yaml",
			"select(di==0).spec.containers[0].securityContext.capabilities.drop += [\"SYS_ADM\"]",
			"inserts/tc-03-01-expected.yaml",
		},
		{
			"inserts/tc-04-00-input.yaml",

			`select(di==0).spec.template.spec.securityContext.allowPrivilegeEscalation |= false |
			 select(di==0).spec.template.spec.containers[0].securityContext.capabilities.drop += ["NET_RAW"] |
			 select(di==0).spec.template.spec.containers[0].securityContext.seccompProfile.type |= "RuntimeDefault" |
			 select(di==0).spec.template.spec.containers[0].securityContext.allowPrivilegeEscalation |= false |
			 select(di==0).spec.template.spec.containers[0].securityContext.readOnlyRootFilesystem |= true`,

			"inserts/tc-04-01-expected.yaml",
		},
		{
			"inserts/tc-05-00-input.yaml",
			"select(di==0).spec.containers[0].securityContext.allowPrivilegeEscalation |= false",
			"inserts/tc-05-01-expected.yaml",
		},
		{
			"inserts/tc-06-00-input.yaml",
			"select(di==0).spec.containers[0].securityContext.capabilities.drop += [\"SYS_ADM\"]",
			"inserts/tc-06-01-expected.yaml",
		},
		{
			"inserts/tc-07-00-input.yaml",

			`select(di==0).spec.containers[0].securityContext.allowPrivilegeEscalation |= false |
			 select(di==1).spec.containers[0].securityContext.allowPrivilegeEscalation |= false`,

			"inserts/tc-07-01-expected.yaml",
		},

		// Removal Scenarios
		{
			"removals/tc-01-00-input.yaml",
			"del(select(di==0).spec.containers[0].securityContext)",
			"removals/tc-01-01-expected.yaml",
		},
		{
			"removals/tc-02-00-input.yaml",
			"del(select(di==0).spec.containers[1])",
			"removals/tc-02-01-expected.yaml",
		},
		{
			"removals/tc-03-00-input.yaml",
			"del(select(di==0).spec.containers[0].securityContext.capabilities.drop[1])",
			"removals/tc-03-01-expected.yaml",
		},
		{
			"removes/tc-04-00-input.yaml",
			`del(select(di==0).spec.containers[0].securityContext) | 
			 del(select(di==1).spec.containers[1])`,
			"removes/tc-04-01-expected.yaml",
		},

		// Replace Scenarios
		{
			"replaces/tc-01-00-input.yaml",
			"select(di==0).spec.containers[0].securityContext.runAsRoot |= false",
			"replaces/tc-01-01-expected.yaml",
		},
		{
			"replaces/tc-02-00-input.yaml",
			`select(di==0).spec.containers[0].securityContext.capabilities.drop[0] |= "SYS_ADM" |
			 select(di==0).spec.containers[0].securityContext.capabilities.add[0] |= "NET_RAW"`,
			"replaces/tc-02-01-expected.yaml",
		},

		// Hybrid Scenarios
		{
			"hybrids/tc-01-00-input.yaml",
			`del(select(di==0).spec.containers[0].securityContext) |
			 select(di==0).spec.securityContext.runAsRoot |= false`,
			"hybrids/tc-01-01-expected.yaml",
		},
		{
			"hybrids/tc-02-00-input-indented-list.yaml",
			`del(select(di==0).spec.containers[0].securityContext) |
			 select(di==0).spec.securityContext.runAsRoot |= false`,
			"hybrids/tc-02-01-expected.yaml",
		},
	}

	return indentationTestCases
}

func TestApplyFixKeepsIndentation(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(tc.inputFile, func(t *testing.T) {
			getTestDataPath := func(filename string) string {
				currentDir, _ := os.Getwd()
				currentFile := "testdata/" + filename
				return filepath.Join(currentDir, currentFile)
			}

			input, _ := os.ReadFile(getTestDataPath(tc.inputFile))
			want, _ := os.ReadFile(getTestDataPath(tc.expectedFile))
			expression := tc.yamlExpression

			h, _ := NewFixHandlerMock()

			got, _ := h.ApplyFixToContent(string(input), expression)

			if got != string(want) {
				t.Errorf(
					"Contents of the fixed file don't match the expectation.\n"+
						"FilePath: %s\n\n"+
						"Got:\n<%s>\n\n"+
						"Want:\n<%s>",
					tc.inputFile,
					got,
					want,
				)
			}
		},
		)

	}
}

func Test_fixPathToValidYamlExpression(t *testing.T) {
	type args struct {
		fixPath             string
		value               string
		documentIndexInYaml int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "fix path with boolean value",
			args: args{
				fixPath:             "spec.template.spec.containers[0].securityContext.privileged",
				value:               "true",
				documentIndexInYaml: 2,
			},
			want: "select(di==2).spec.template.spec.containers[0].securityContext.privileged |= true",
		},
		{
			name: "fix path with string value",
			args: args{
				fixPath:             "metadata.namespace",
				value:               "YOUR_NAMESPACE",
				documentIndexInYaml: 0,
			},
			want: "select(di==0).metadata.namespace |= \"YOUR_NAMESPACE\"",
		},
		{
			name: "fix path with number",
			args: args{
				fixPath:             "xxx.yyy",
				value:               "123",
				documentIndexInYaml: 0,
			},
			want: "select(di==0).xxx.yyy |= 123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixPathToValidYamlExpression(tt.args.fixPath, tt.args.value, tt.args.documentIndexInYaml); got != tt.want {
				t.Errorf("fixPathToValidYamlExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
