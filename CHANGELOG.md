# v0.6.0 - 2026-02-13

- Docker-based dev/test runs on Cortex v1.20.1 (blocks storage); config and compose updated accordingly.
- Terraform Plugin SDK and cortex-tools upgraded; direct Prometheus dependency removed (replace directives in go.mod).
- Go 1.26; Makefile and GoReleaser aligned (release flags, OS/arch matrix, version/commit ldflags); releases built only for Terraform-supported OS/arch, including darwin/arm64.
- Alertmanager config marked sensitive; YAML rule handling normalised for consistent state/diff.

# v0.0.2 - 2020-02-03

- Improve logging. #6

# v0.0.2 - 2020-02-03

- Add examples to the documentation

# v0.0.1 - 2020-02-03

- Initial release
