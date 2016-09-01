# Change Log

## [0.1.6](https://github.com/grammarly/rocker-compose/tree/0.1.6) (2016-09-01)

[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.5...v0.1.6)

**Merged pull requests:**

- Added configurable timeouts for connection to docker socket [\#54](https://github.com/grammarly/rocker-compose/pull/54) ([rodio](https://github.com/rodio))
- Updated dependencies to support docker 1.12 [\#53](https://github.com/grammarly/rocker-compose/pull/53) ([rodio](https://github.com/rodio))

## [0.1.5](https://github.com/grammarly/rocker-compose/tree/0.1.5) (2016-02-12)

[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.4...0.1.5)

**Implemented enhancements:**

- New S3 naming schema support [\#42](https://github.com/grammarly/rocker-compose/pull/42)

## [0.1.4](https://github.com/grammarly/rocker-compose/tree/HEAD) (2016-02-08)

[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.3...0.1.4)

**Implemented enhancements:**

- Use Docker credentials state to access private repositories [\#20](https://github.com/grammarly/rocker-compose/issues/20)
- Build was moved to GO15VENDOREXPERIMENT [\#41](https://github.com/grammarly/rocker-compose/pull/41) ([romank87](https://github.com/romank87))
- Experimental integration with [AWS ECR](https://aws.amazon.com/ecr/)
- Experimental support of S3 as a backend for storing images instead of Docker Registry


## [0.1.3](https://github.com/grammarly/rocker-compose/tree/0.1.3) (2015-11-26)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.2...0.1.3)

**Fixed bugs:**

- rocker-compose remove volumes if dependant container re-creates, Docker 1.9 [\#35](https://github.com/grammarly/rocker-compose/issues/35)

## [0.1.2](https://github.com/grammarly/rocker-compose/tree/0.1.2) (2015-11-23)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.3-rc1...0.1.2)

**Implemented enhancements:**

- make better non-colored logger [\#34](https://github.com/grammarly/rocker-compose/issues/34)
- Ping docker server before running [\#33](https://github.com/grammarly/rocker-compose/issues/33)
- Dump stack when receive SIGUSR1 [\#32](https://github.com/grammarly/rocker-compose/issues/32)
- Semver matching for image names [\#31](https://github.com/grammarly/rocker-compose/issues/31)
- Use `image` helper to run with artifacts from rocker [\#23](https://github.com/grammarly/rocker-compose/issues/23)
- Feature RQ: include function for the go templates [\#19](https://github.com/grammarly/rocker-compose/issues/19)

**Fixed bugs:**

- Invalid volume on Windows [\#25](https://github.com/grammarly/rocker-compose/issues/25)
- Rocker-compose cannot pull image if name contains '/' character [\#16](https://github.com/grammarly/rocker-compose/issues/16)
- External dependencies without namespace are not resolved [\#13](https://github.com/grammarly/rocker-compose/issues/13)
- If wait\_for points to `state:running` containers, rocker-compose hangs [\#9](https://github.com/grammarly/rocker-compose/issues/9)

**Merged pull requests:**

- 0.1.2 [\#30](https://github.com/grammarly/rocker-compose/pull/30) ([ybogdanov](https://github.com/ybogdanov))
- Fix logging into stdin, when print option are enabled [\#26](https://github.com/grammarly/rocker-compose/pull/26) ([ctrlok](https://github.com/ctrlok))

## [0.1.3-rc1](https://github.com/grammarly/rocker-compose/tree/0.1.3-rc1) (2015-09-30)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.1...0.1.3-rc1)

## [0.1.1](https://github.com/grammarly/rocker-compose/tree/0.1.1) (2015-09-17)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.1.0...0.1.1)

**Implemented enhancements:**

- Installation instructions are absent in README.md  [\#3](https://github.com/grammarly/rocker-compose/issues/3)

**Fixed bugs:**

- Incorrect file extension for binary release file [\#8](https://github.com/grammarly/rocker-compose/issues/8)

**Closed issues:**

- Mistake in README [\#4](https://github.com/grammarly/rocker-compose/issues/4)

**Merged pull requests:**

- Adding support for creating local binary without rocker [\#7](https://github.com/grammarly/rocker-compose/pull/7) ([ybogdanov](https://github.com/ybogdanov))
- Update README.md [\#6](https://github.com/grammarly/rocker-compose/pull/6) ([vseloved](https://github.com/vseloved))

## [0.1.0](https://github.com/grammarly/rocker-compose/tree/0.1.0) (2015-09-01)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.22...0.1.0)

**Merged pull requests:**

- Fixes to README [\#2](https://github.com/grammarly/rocker-compose/pull/2) ([vseloved](https://github.com/vseloved))

## [0.0.22](https://github.com/grammarly/rocker-compose/tree/0.0.22) (2015-08-18)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.21...0.0.22)

## [0.0.21](https://github.com/grammarly/rocker-compose/tree/0.0.21) (2015-08-18)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.20...0.0.21)

## [0.0.20](https://github.com/grammarly/rocker-compose/tree/0.0.20) (2015-08-11)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.19...0.0.20)

## [0.0.19](https://github.com/grammarly/rocker-compose/tree/0.0.19) (2015-08-11)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.18...0.0.19)

## [0.0.18](https://github.com/grammarly/rocker-compose/tree/0.0.18) (2015-07-30)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.17...0.0.18)

## [0.0.17](https://github.com/grammarly/rocker-compose/tree/0.0.17) (2015-07-28)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.16...0.0.17)

## [0.0.16](https://github.com/grammarly/rocker-compose/tree/0.0.16) (2015-07-28)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.15...0.0.16)

## [0.0.15](https://github.com/grammarly/rocker-compose/tree/0.0.15) (2015-07-27)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.14...0.0.15)

## [0.0.14](https://github.com/grammarly/rocker-compose/tree/0.0.14) (2015-07-27)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.13...0.0.14)

## [0.0.13](https://github.com/grammarly/rocker-compose/tree/0.0.13) (2015-07-23)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.12...0.0.13)

## [0.0.12](https://github.com/grammarly/rocker-compose/tree/0.0.12) (2015-07-22)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.11...0.0.12)

## [0.0.11](https://github.com/grammarly/rocker-compose/tree/0.0.11) (2015-07-16)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.10...0.0.11)

## [0.0.10](https://github.com/grammarly/rocker-compose/tree/0.0.10) (2015-07-14)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.9...0.0.10)

## [0.0.9](https://github.com/grammarly/rocker-compose/tree/0.0.9) (2015-07-12)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.8...0.0.9)

## [0.0.8](https://github.com/grammarly/rocker-compose/tree/0.0.8) (2015-07-10)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.7...0.0.8)

## [0.0.7](https://github.com/grammarly/rocker-compose/tree/0.0.7) (2015-07-08)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.6...0.0.7)

## [0.0.6](https://github.com/grammarly/rocker-compose/tree/0.0.6) (2015-07-07)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.5...0.0.6)

## [0.0.5](https://github.com/grammarly/rocker-compose/tree/0.0.5) (2015-07-07)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.4...0.0.5)

**Fixed bugs:**

- Diff test is handing on resolving external dependencies [\#1](https://github.com/grammarly/rocker-compose/issues/1)

## [0.0.4](https://github.com/grammarly/rocker-compose/tree/0.0.4) (2015-07-04)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.3...0.0.4)

## [0.0.3](https://github.com/grammarly/rocker-compose/tree/0.0.3) (2015-07-04)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.2...0.0.3)

## [0.0.2](https://github.com/grammarly/rocker-compose/tree/0.0.2) (2015-07-03)
[Full Changelog](https://github.com/grammarly/rocker-compose/compare/0.0.1...0.0.2)

## [0.0.1](https://github.com/grammarly/rocker-compose/tree/0.0.1) (2015-07-02)


\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*

\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*
