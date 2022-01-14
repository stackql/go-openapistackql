package argparse

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var (
	BuildMajorVersion   string = ""
	BuildMinorVersion   string = ""
	BuildPatchVersion   string = ""
	BuildCommitSHA      string = ""
	BuildShortCommitSHA string = ""
	BuildDate           string = ""
	BuildPlatform       string = ""
)

var SemVersion string = fmt.Sprintf("%s.%s.%s", BuildMajorVersion, BuildMinorVersion, BuildPatchVersion)

type runtimeContext struct {
	CPUProfile  string
	LogLevelStr string
}

var (
	runtimeCtx      runtimeContext
	replicateCtrMgr bool = false
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "openapistackql",
	Version: SemVersion,
	Short:   "model for SQL semantics from openapi docs",
	Long: `
	model for SQL semantics from openapi docs
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// in the root command is executed with no arguments, print the help message
		usagemsg := cmd.Long + "\n\n" + cmd.UsageString()
		fmt.Println(usagemsg)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetVersionTemplate("InfraQL v{{.Version}} " + BuildPlatform + " (" + BuildShortCommitSHA + ")\nBuildDate: " + BuildDate + "\nhttps://infraql.io\n")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&runtimeCtx.CPUProfile, "cpuprofile", "", "cpuprofile file, none if empty")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.LogLevelStr, "loglevel", "warn", "loglevel")

	rootCmd.AddCommand(execCmd)

}

func setLogLevel() {
	logLevel, err := log.ParseLevel(runtimeCtx.LogLevelStr)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logLevel)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	setLogLevel()

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
