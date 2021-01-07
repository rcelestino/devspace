package save

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct {
	*flags.GlobalFlags

	SecretName string
}

func newVarsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &varsCmd{
		GlobalFlags: globalFlags,
	}

	varsCmd := &cobra.Command{
		Use:   "vars",
		Short: "Saves variable values to kubernetes",
		Long: `
#######################################################
################ devspace save vars ###################
#######################################################
Saves devspace config variable values into a kubernetes 
secret. Variable values can be shared or restored via
devspace restore vars.

Examples:
devspace save vars
devspace save vars --namespace test 
devspace save vars --vars-secret my-secret
#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		}}

	varsCmd.Flags().StringVar(&cmd.SecretName, "vars-secret", "devspace-vars", "The secret to use to save the variables into")

	return varsCmd
}

// RunSetVar executes the set var command logic
func (cmd *varsCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader(nil, logger)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, logger)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, logger)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	_, err = configLoader.Load()
	if err != nil {
		return err
	}

	// Make sure the vars are also saved to file
	err = configLoader.SaveGenerated()
	if err != nil {
		return fmt.Errorf("error saving generated.yaml: %v", err)
	}

	// save the vars into the kubernetes secret
	err = loader.SaveVarsInSecret(client, generatedConfig.Vars, cmd.SecretName)
	if err != nil {
		return err
	}

	logger.Donef("Successfully written vars to secret %s/%s", client.Namespace(), cmd.SecretName)
	return nil
}
