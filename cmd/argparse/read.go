package argparse

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/spf13/cobra"

	"github.com/stackql/go-openapistackql/openapistackql"
)

func printErrorAndExitOneIfNil(subject interface{}, msg string) {
	if subject == nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintln(msg))
		os.Exit(1)
	}
}

func printErrorAndExitOneIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintln(err.Error()))
		os.Exit(1)
	}
}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "read",
	Short: "Simple textual openapistackql doc read and display",
	Long:  `Simple textual openapistackql doc read and display`,
	Run: func(cmd *cobra.Command, args []string) {

		if runtimeCtx.CPUProfile != "" {
			f, err := os.Create(runtimeCtx.CPUProfile)
			if err != nil {
				printErrorAndExitOneIfError(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}

		if len(args) == 0 || args[0] == "" {
			cmd.Help()
			os.Exit(0)
		}

		RunCommand(runtimeCtx, args[0])
	},
}

func RunCommand(rtCtx runtimeContext, arg string) {
	b, err := os.ReadFile(arg)
	printErrorAndExitOneIfError(err)
	l := openapistackql.NewLoader()
	svc, err := l.LoadFromBytes(b)
	printErrorAndExitOneIfError(err)
	printErrorAndExitOneIfNil(svc, "doc parse gave me doughnuts!!!\n\n")
	fmt.Fprintf(os.Stdout, "\nsuccessfully parsed svc = '%s'\n", svc.GetName())

}
