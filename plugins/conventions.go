package plugins

import "path"

func (c ProjectConfig) DefinitionsDirectory() string {
	if c.DefinitionsPath != "" {
		return path.Join(c.ProjectRoot, c.DefinitionsPath)
	}
	return path.Join(c.RuntimeDir, "graphql")
}

func (c ProjectConfig) DefinitionsEnumRuntime() string {
	return path.Join(c.DefinitionsDirectory(), "enums.js")
}

func (c ProjectConfig) DefinitionsEnumTypes() string {
	return path.Join(c.DefinitionsDirectory(), "enums.d.ts")
}

func (c ProjectConfig) ArtifactDirectory() string {
	return path.Join(
		c.ProjectRoot,
		c.RuntimeDir,
		"artifacts",
	)
}

func (c ProjectConfig) ArtifactPath(name string) string {
	return path.Join(
		c.ArtifactDirectory(),
		name+".js",
	)
}
