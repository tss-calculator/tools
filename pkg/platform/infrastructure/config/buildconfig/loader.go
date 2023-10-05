package buildconfig

import (
	"encoding/json"
	"io"
	"os"

	"github.com/tss-calculator/tools/pkg/platform/application/model"
)

type Command struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args,omitempty"`
	DependsOn  []string `json:"dependsOn,omitempty"`
}

type BuildConfig struct {
	Sources     Command `json:"sources"`
	DockerImage Command `json:"docker-image"`
}

type Config struct {
	Build BuildConfig `json:"build"`
}

type ConfigLoader interface {
	Load(filePath string) (model.BuildConfig, error)
}

func NewConfigLoader() ConfigLoader {
	return &configLoader{configs: make(map[string]model.BuildConfig)}
}

type configLoader struct {
	configs map[string]model.BuildConfig
}

func (loader *configLoader) Load(filePath string) (model.BuildConfig, error) {
	config, ok := loader.configs[filePath]
	if ok {
		return config, nil
	}
	config, err := loader.load(filePath)
	if err != nil {
		return model.BuildConfig{}, err
	}
	loader.configs[filePath] = config
	return config, nil
}

func (loader *configLoader) load(filePath string) (model.BuildConfig, error) {
	configFile, err := os.Open(filePath)
	if err != nil {
		return model.BuildConfig{}, err
	}
	defer configFile.Close()
	configBody, err := io.ReadAll(configFile)
	if err != nil {
		return model.BuildConfig{}, err
	}

	var config Config
	err = json.Unmarshal(configBody, &config)
	if err != nil {
		return model.BuildConfig{}, err
	}
	return model.BuildConfig{
		Sources: model.BuildCommand{
			Executable: config.Build.Sources.Executable,
			Args:       config.Build.Sources.Args,
			DependsOn:  config.Build.Sources.DependsOn,
		},
		DockerImage: model.BuildCommand{
			Executable: config.Build.DockerImage.Executable,
			Args:       config.Build.DockerImage.Args,
			DependsOn:  config.Build.DockerImage.DependsOn,
		},
	}, err
}
