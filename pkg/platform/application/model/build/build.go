package build

type Command struct {
	Executable string
	Args       []string
}

type Image struct {
	Name       string
	Context    string
	DockerFile string
	TagBy      *string
}

type Config struct {
	Sources Command
	Images  []Image
}
