# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased
### Changed
- Updated GH worker for windows

## v3.8.4 - 2025-08-14

### ⛓️ Dependencies
- Updated golang patch version to v1.24.6

## v3.8.3 - 2025-08-07

### ⛓️ Dependencies
- Updated golang patch version to v1.24.5

## v3.9.0 - 2025-07-31

### 🛡️ Security notices
- fix: internal tools module path

### ⛓️ Dependencies
- Updated golang patch version to v1.24.5

## v3.8.2 - 2025-06-26

### ⛓️ Dependencies
- Updated golang version to v1.24.4

## v3.8.1 - 2025-02-20

### ⛓️ Dependencies
- Updated golang patch version to v1.23.6

## v3.8.0 - 2025-02-07

### 🚀 Enhancements
- Add FIPS compliant packages
- Updated go to v1.23.5
- Updated module github.com/golangci/golangci-lint to v1.63.4

## v3.7.2 - 2024-12-19

### 🐞 Bug fixes
- Updated go to v1.22.9

## 2.5.0
### Added

- Moved default config.sample to [V4](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-newer-configuration-format/), added a dependency for infra-agent version 1.20.0

Please notice that old [V3](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-standard-configuration-format/) configuration format is deprecated, but still supported.

## 2.4.7 (2021-06-10)
### Changed
- ARM support

## 2.4.6 (2021-04-26)
### Changed
- Upgraded github.com/newrelic/infra-integrations-sdk to v3.6.7
- Switched to go modules
- Upgraded pipeline to go 1.16
- Replaced gometalinter with golangci-lint

## 2.4.5 (2021-01-13)
### Changed
- This fixes scenario where warnings received from nrjmx tool discarded data subsequent data. Now all warnings get logged and all data get processed.
- Bundles previous non jlinked nrjmx tool version, due to issue when querying.

## 2.4.4 (2020-01-22)
## Changed
- Username and password now default to empty to support no auth

## 2.4.3 (2020-01-22)
## Fixed
- Continue collection when a query returns an empty result

## 2.4.2 (2019-12-09)
## Added
- Supported new `metric_type` values:
    - `pdelta`: like `delta` but only reports positive values (returning `0` if the accounted value is negative)
    - `prate`: like `rate` but only reports positive values (returning `0` if the accounted value is negative) 

## 2.4.1 (2019-12-02)
### Changed
- Updated the nrjmx path in the definition file

## 2.4.0 (2019-11-18)
### Changed
- Renamed the integration executable from nr-jmx to nri-jmx in order to be consistent with the package naming. **Important Note:** if you have any security module rules (eg. SELinux), alerts or automation that depends on the name of this binary, these will have to be updated.

## 2.3.5 - 2019-11-19
### Added
- Error when a metric is set for the second time

## 2.3.4 - 2019-11-19
### Added
- Error when a metric is set for the second time

## 2.3.3 - 2019-11-18
### Added
- Add nrjmx version dependency to 1.5.2, so jmxterm can be bundled within
  package.

## 2.3.2 - 2019-11-14
### Added
- Remove windows definition from linux package

## 2.3.1 - 2019-11-14
### Added
- Nrjmx tool within the Windows installer
 
## 2.3.0 - 2019-11-13
### Changed
- Updated `nrjmx` version to the more stable latest one.

## 2.2.5 - 2019-10-16
### Fixed
- Windows installer GUIDs

## 2.2.4 - 2019-09-06
### Fixed
- Panic when JMX attributes have commas in them

## 2.2.3 - 2019-09-06
### Fixed
- Broken build

## 2.2.2 - 2019-08-28
### Added
- Custom uri path

## 2.2.1 - 2019-08-28
### Added
- Local entity support

## 2.2.0 - 2019-08-22
### Added
- Windows build support

## 2.1.0 - 2019-06-19
### Added
- Remote JBoss Standalone support
- Updated the SDK

## 2.0.0 - 2019-04-29
### Changed
- Added identity attributes for better uniqueness
- Updated the SDK

## 1.0.4 - 2019-03-19
### Changed
- Include jvm-metrics.yml.sample in package

## 1.0.3 - 2019-03-18
### Added
- Added remote option to use remote URL connections. Format: service:jmx:remoting-jmx://host:port

## 1.0.2 - 2019-02-13
### Added
- Added SSL option to Jmx.Open

## 1.0.1 - 2019-02-04
### Fixed
- Updated protocol version

## 1.0.0 - 2018-11-16
### Changed
- Updated to version 1.0.0

## 0.1.8 - 2018-11-09
### Fixed
- Fix error with incorrect metric type interface

## 0.1.7 - 2018-11-01
### Added
- Sample file for Hikaridb

## 0.1.6 - 2018-09-26
### Changed
- Updated sample configuration file with JMX-specific fields

## 0.1.5 - 2018-09-20
### Fixed
- Fixed bug with parsing JMX queries

## 0.1.4 - 2018-09-19
### Added
- Added tomcat-metrics.yml.sample back as an additional sample

## 0.1.3 - 2018-09-18
### Changed
- Removed extra yml files and renamed existing ones with .sample extension

## 0.1.2 - 2018-09-14
### Added
- Logic to enforce a soft limit on the number of metrics that can be collect. If the number of metrics per Entity exceeds this limit the Entity will not be reported to NR.

## 0.1.1 - 2018-09-13
### Changed
- Renamed nr-jmx-config.yml.template to jmx-config.yml.sample
- Renamed nr-jmx-definition.yml to jmx-definition.yml

## 0.1.0 - 2018-07-24
### Added
- Initial version: Includes Metrics and Inventory data
