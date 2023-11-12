package buildconfig

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"

	"github.com/tss-calculator/tools/pkg/platform/application/model/build"
)

type Command struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

type Image struct {
	Name       string `json:"name"`
	Context    string `json:"context"`
	DockerFile string `json:"dockerFile"`
	TagBy      string `json:"tagBy,omitempty"`
	SkipPush   bool   `json:"skipPush"`
}

type Build struct {
	Sources Command `json:"sources"`
	Images  []Image `json:"images"`
}

type Config struct {
	Build Build `json:"build"`
}

func NewLoader() *Loader {
	return &Loader{cache: make(map[string]build.Config)}
}

type Loader struct {
	cache map[string]build.Config
}

func (l *Loader) Load(path string) (build.Config, error) {
	config, ok := l.cache[path]
	if ok {
		return config, nil
	}
	config, err := load(path)
	if err != nil {
		return build.Config{}, err
	}
	l.cache[path] = config
	return config, nil
}

func load(path string) (build.Config, error) {
	configBody, err := os.ReadFile(path)
	if err != nil {
		return build.Config{}, errors.Wrapf(err, "failed to read config file: %v", path)
	}
	var infraConfig Config
	err = json.Unmarshal(configBody, &infraConfig)
	if err != nil {
		return build.Config{}, errors.Wrap(err, "failed to unmarshal config")
	}
	return mapInfraConfigToAppConfig(infraConfig), nil
}

func mapInfraConfigToAppConfig(config Config) build.Config {
	images := make([]build.Image, 0, len(config.Build.Images))
	for _, image := range config.Build.Images {
		images = append(images, build.Image{
			Name:       image.Name,
			Context:    image.Context,
			DockerFile: image.DockerFile,
			TagBy:      toOptString(image.TagBy),
			SkipPush:   image.SkipPush,
		})
	}
	return build.Config{
		Sources: build.Command{
			Executable: config.Build.Sources.Executable,
			Args:       config.Build.Sources.Args,
		},
		Images: images,
	}
}

func toOptString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
