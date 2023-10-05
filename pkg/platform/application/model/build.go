package model

type BuildCommand struct {
	Executable string
	Args       []string
	DependsOn  []string
}

type BuildConfig struct {
	Sources     BuildCommand
	DockerImage BuildCommand
}
