package manifest

import (
	"fmt"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/experimentalflags"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

// A Build represents the build environment for other pipelines. As a
// general rule, tools required to build pipelines are used from the build
// environment, rather than from the pipeline itself. Without a specified
// build environment, the build host's root filesystem would be used, which
// is not predictable nor reproducible. For the purposes of building the
// build pipeline, we do use the build host's filesystem, this means we should
// make minimal assumptions about what's available there.

type Build interface {
	Name() string
	Checkpoint()
	Manifest() *Manifest

	addDependent(dep Pipeline)
}

type BuildrootFromPackages struct {
	Base

	runner       runner.Runner
	dependents   []Pipeline
	repos        []rpmmd.RepoConfig
	packageSpecs []rpmmd.PackageSpec

	containerBuildable bool
}

type BuildOptions struct {
	// ContainerBuildable tweaks the buildroot to be container friendly,
	// i.e. to not rely on an installed osbuild-selinux
	ContainerBuildable bool
}

// NewBuild creates a new build pipeline from the repositories in repos
// and the specified packages.
func NewBuild(m *Manifest, runner runner.Runner, repos []rpmmd.RepoConfig, opts *BuildOptions) Build {
	if opts == nil {
		opts = &BuildOptions{}
	}

	name := "build"
	pipeline := &BuildrootFromPackages{
		Base:               NewBase(name, nil),
		runner:             runner,
		dependents:         make([]Pipeline, 0),
		repos:              filterRepos(repos, name),
		containerBuildable: opts.ContainerBuildable,
	}

	// This allows to bootstrap the buildroot with a custom container
	// for e.g. cross-arch-build experiments,
	maybeAddExperimentalContainerBootstrap(m, runner, opts, pipeline)

	m.addPipeline(pipeline)
	return pipeline
}

func (p *BuildrootFromPackages) addDependent(dep Pipeline) {
	p.dependents = append(p.dependents, dep)
	man := p.Manifest()
	if man == nil {
		panic("cannot add build dependent without a manifest")
	}
	man.addPipeline(dep)
}

func (p *BuildrootFromPackages) getPackageSetChain(distro Distro) []rpmmd.PackageSet {
	// TODO: make the /usr/bin/cp dependency conditional
	// TODO: make the /usr/bin/xz dependency conditional
	packages := []string{
		"selinux-policy-targeted", // needed to build the build pipeline
		"coreutils",               // /usr/bin/cp - used all over
		"xz",                      // usage unclear
	}

	packages = append(packages, p.runner.GetBuildPackages()...)

	for _, pipeline := range p.dependents {
		packages = append(packages, pipeline.getBuildPackages(distro)...)
	}

	return []rpmmd.PackageSet{
		{
			Include:         packages,
			Repositories:    p.repos,
			InstallWeakDeps: true,
		},
	}
}

func (p *BuildrootFromPackages) getPackageSpecs() []rpmmd.PackageSpec {
	return p.packageSpecs
}

func (p *BuildrootFromPackages) serializeStart(inputs Inputs) {
	if len(p.packageSpecs) > 0 {
		panic("double call to serializeStart()")
	}
	p.packageSpecs = inputs.Depsolved.Packages
	p.repos = append(p.repos, inputs.Depsolved.Repos...)
}

func (p *BuildrootFromPackages) serializeEnd() {
	if len(p.packageSpecs) == 0 {
		panic("serializeEnd() call when serialization not in progress")
	}
	p.packageSpecs = nil
}

func (p *BuildrootFromPackages) serialize() osbuild.Pipeline {
	if len(p.packageSpecs) == 0 {
		panic("serialization not started")
	}
	pipeline := p.Base.serialize()
	pipeline.Runner = p.runner.String()

	pipeline.AddStage(osbuild.NewRPMStage(osbuild.NewRPMStageOptions(p.repos), osbuild.NewRpmStageSourceFilesInputs(p.packageSpecs)))
	pipeline.AddStage(osbuild.NewSELinuxStage(&osbuild.SELinuxStageOptions{
		FileContexts: "etc/selinux/targeted/contexts/files/file_contexts",
		Labels:       p.getSELinuxLabels(),
	},
	))

	return pipeline
}

// Returns a map of paths to labels for the SELinux stage based on specific
// packages found in the pipeline.
func (p *BuildrootFromPackages) getSELinuxLabels() map[string]string {
	labels := make(map[string]string)
	for _, pkg := range p.getPackageSpecs() {
		switch pkg.Name {
		case "coreutils":
			labels["/usr/bin/cp"] = "system_u:object_r:install_exec_t:s0"
			if p.containerBuildable {
				labels["/usr/bin/mount"] = "system_u:object_r:install_exec_t:s0"
				labels["/usr/bin/umount"] = "system_u:object_r:install_exec_t:s0"
			}
		case "tar":
			labels["/usr/bin/tar"] = "system_u:object_r:install_exec_t:s0"
		}
	}
	return labels
}

// maybeAddExperimentalContainerBootstrap will return a container buildroot
// if the "IMAGE_BUILDER_EXPERIMENTAL=bootstrap=<container-ref>" is
// defined. This allows us to do cross-arch build experimentation.
//
// A "bootstrap" container has only these requirements:
// - python3 for the runners
// - rpm so that the real buildroot rpms can get installed
// - setfiles so that the selinux stage for the real buildroot can run
// (and does not even need a working dnf or repo setup).
func maybeAddExperimentalContainerBootstrap(m *Manifest, runner runner.Runner, opts *BuildOptions, build *BuildrootFromPackages) {
	bootstrapBuildrootRef := experimentalflags.String("bootstrap")
	if bootstrapBuildrootRef == "" {
		return
	}

	cntSrcs := []container.SourceSpec{
		{
			Source:    bootstrapBuildrootRef,
			Name:      bootstrapBuildrootRef,
			TLSVerify: common.ToPtr(false),
		},
	}
	name := "bootstrap-buildroot"
	bootstrapPipeline := &BuildrootFromContainer{
		Base:           NewBase(name, nil),
		runner:         runner,
		dependents:     make([]Pipeline, 0),
		containers:     cntSrcs,
		disableSelinux: true,
	}
	m.addPipeline(bootstrapPipeline)
	build.build = bootstrapPipeline
}

type BuildrootFromContainer struct {
	Base

	runner     runner.Runner
	dependents []Pipeline

	containers     []container.SourceSpec
	containerSpecs []container.Spec

	containerBuildable bool
	disableSelinux     bool
}

// NewBuildFromContainer creates a new build pipeline from the given
// containers specs
func NewBuildFromContainer(m *Manifest, runner runner.Runner, containerSources []container.SourceSpec, opts *BuildOptions) Build {
	if opts == nil {
		opts = &BuildOptions{}
	}

	name := "build"
	pipeline := &BuildrootFromContainer{
		Base:       NewBase(name, nil),
		runner:     runner,
		dependents: make([]Pipeline, 0),
		containers: containerSources,

		containerBuildable: opts.ContainerBuildable,
	}
	m.addPipeline(pipeline)
	return pipeline
}

func (p *BuildrootFromContainer) addDependent(dep Pipeline) {
	p.dependents = append(p.dependents, dep)
	man := p.Manifest()
	if man == nil {
		panic("cannot add build dependent without a manifest")
	}
	man.addPipeline(dep)
}

func (p *BuildrootFromContainer) getContainerSources() []container.SourceSpec {
	return p.containers
}

func (p *BuildrootFromContainer) getContainerSpecs() []container.Spec {
	return p.containerSpecs
}

func (p *BuildrootFromContainer) serializeStart(inputs Inputs) {
	if len(p.containerSpecs) > 0 {
		panic("double call to serializeStart()")
	}
	p.containerSpecs = inputs.Containers
}

func (p *BuildrootFromContainer) serializeEnd() {
	if len(p.containerSpecs) == 0 {
		panic("serializeEnd() call when serialization not in progress")
	}
	p.containerSpecs = nil
}

func (p *BuildrootFromContainer) getSELinuxLabels() map[string]string {
	if p.disableSelinux {
		return nil
	}

	labels := map[string]string{
		"/usr/bin/ostree": "system_u:object_r:install_exec_t:s0",
	}
	if p.containerBuildable {
		labels["/usr/bin/mount"] = "system_u:object_r:install_exec_t:s0"
		labels["/usr/bin/umount"] = "system_u:object_r:install_exec_t:s0"
	}
	return labels
}

func (p *BuildrootFromContainer) serialize() osbuild.Pipeline {
	if len(p.containerSpecs) == 0 {
		panic("serialization not started")
	}
	if len(p.containerSpecs) != 1 {
		panic(fmt.Sprintf("BuildrootFromContainer expectes exactly one container input, got: %v", p.containerSpecs))
	}

	pipeline := p.Base.serialize()
	pipeline.Runner = p.runner.String()

	image := osbuild.NewContainersInputForSingleSource(p.containerSpecs[0])
	// Make skopeo copy to remove the signatures of signed containers by default to workaround
	// build failures until https://github.com/containers/image/issues/2599 is implemented
	stage, err := osbuild.NewContainerDeployStage(image, &osbuild.ContainerDeployOptions{RemoveSignatures: true})
	if err != nil {
		panic(err)
	}
	pipeline.AddStage(stage)
	if !p.disableSelinux {
		pipeline.AddStage(osbuild.NewSELinuxStage(
			&osbuild.SELinuxStageOptions{
				FileContexts: "etc/selinux/targeted/contexts/files/file_contexts",
				ExcludePaths: []string{"/sysroot"},
				Labels:       p.getSELinuxLabels(),
			},
		))
	}

	return pipeline
}
