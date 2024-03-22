package rhel8

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
)

func mkWslImgType() *rhel.ImageType {
	it := rhel.NewImageType(
		"wsl",
		"disk.tar.gz",
		"application/x-tar",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: ubiCommonPackageSet,
		},
		rhel.TarImage,
		[]string{"build"},
		[]string{"os", "archive"},
		[]string{"archive"},
	)

	it.DefaultImageConfig = &distro.ImageConfig{
		Locale:    common.ToPtr("en_US.UTF-8"),
		NoSElinux: common.ToPtr(true),
		WSLConfig: &osbuild.WSLConfStageOptions{
			Boot: osbuild.WSLConfBootOptions{
				Systemd: true,
			},
		},
	}

	return it
}

func ubiCommonPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	ps := rpmmd.PackageSet{
		Include: []string{
			"alternatives",
			"audit-libs",
			"basesystem",
			"bash",
			"brotli",
			"ca-certificates",
			"coreutils-single",
			"crypto-policies-scripts",
			"curl",
			"libcurl",
			"dnf",
			"filesystem",
			"findutils",
			"gdb-gdbserver",
			// Differs from official UBI, as we don't include CRB repos
			// "gdbm",
			"glibc-minimal-langpack",
			"gmp",
			"gnupg2",
			"gobject-introspection",
			"hostname",
			"langpacks-en",
			"pam",
			"passwd",
			"python3",
			"python3-inotify",
			"python3-systemd",
			"redhat-release",
			"rootfiles",
			"rpm",
			"sed",
			"setup",
			"shadow-utils",
			"subscription-manager",
			"systemd",
			"tar",
			"tpm2-tss",
			"tzdata",
			"util-linux",
			"vim-minimal",
			"yum",
		},
		Exclude: []string{
			"aic94xx-firmware",
			"alsa-firmware",
			"alsa-lib",
			"alsa-tools-firmware",
			"biosdevname",
			"cpio",
			"diffutils",
			"dnf-plugin-spacewalk",
			"dracut",
			"elfutils-debuginfod-client",
			"fedora-release",
			"fedora-repos",
			"fontpackages-filesystem",
			"gawk-all-langpacks",
			"gettext",
			"glibc-gconv-extra",
			"glibc-langpack-en",
			"gnupg2-smime",
			"grub2-common",
			"hardlink",
			"iprutils",
			"ivtv-firmware",
			"kbd",
			"kmod",
			"kpartx",
			"libcroco",
			"libcrypt-compat",
			"libevent",
			"libkcapi",
			"libkcapi-hmaccalc",
			"libsecret",
			"libselinux-utils",
			"libxkbcommon",
			"libertas-sd8787-firmware",
			"memstrack",
			"nss",
			"openssl",
			"openssl-pkcs11",
			"os-prober",
			"pigz",
			"pinentry",
			"plymouth",
			"policycoreutils",
			"python3-unbound",
			"redhat-release-eula",
			"rng-tools",
			"rpm-plugin-selinux",
			"rpm-plugin-systemd-inhibit",
			"selinux-policy",
			"selinux",
			"selinux-policy-targeted",
			"shared-mime-info",
			"systemd-udev",
			"trousers",
			"udisks2",
			"unbound-libs",
			"xkeyboard-config",
			"xz",
		},
	}

	return ps
}
